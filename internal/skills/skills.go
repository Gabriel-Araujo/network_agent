package skills

import (
	"fmt"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/config"
	"github.com/Gabriel-Araujo/network_agent/internal/logger"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
)

// log é o handle do pacote (uma declaração por pacote, não por arquivo).
var log = logger.Named("SKILLS")

const (
	ToolUseSkill           = "use_skill"
	ToolReadSkillFile      = "read_skill_file"
	ToolReadSkillDirectory = "read_skill_directory"

	// ReadSkillFileNoActivation diz o que o HandleSkillCall realmente faz: lê o
	// arquivo bundled pelo nome da skill, sem conferir ativação em lugar nenhum.
	// A descrição antiga prometia "use somente depois de ativar a skill", regra
	// que nenhum código impõe. Mesma forma do EDIT_MULTIPLE_MATCHES_NOTICE: a
	// constante é a frase que o modelo lê.
	//
	// A frase não nomeia o use_skill de propósito: só quem tem skill nomeada no
	// prompt recebe essa tool (estreita), e citar tool que não se tem é o
	// defeito que o modifier/AGENT.md corrigiu no ticket 20. O sentido é o mesmo
	// para quem tem ativação e para quem não tem: a leitura não depende dela.
	ReadSkillFileNoActivation = "Não exige ativação prévia da skill: o arquivo é lido pelo nome dela."
)

// UseSkillTool é a tool de ativação com o catálogo estreitado aos nomes
// recebidos: é o que o Subagente despachado leva — só as skills que o
// AGENT.md dele nomeia e manda ativar. Nome fora do registry é erro explícito:
// a lista é código, e um prompt silencioso sem a skill é exatamente o defeito
// que a estreiteira existe para não reintroduzir.
func (r *Registry) UseSkillTool(names []string) (responses.ToolUnionParam, error) {
	for _, name := range names {
		if _, ok := r.skills[name]; !ok {
			return responses.ToolUnionParam{}, fmt.Errorf("skill %q not found in the skills registry", name)
		}
	}
	return r.useSkillToolFor(names), nil
}

func (r *Registry) useSkillToolFor(names []string) responses.ToolUnionParam {
	var catalog strings.Builder
	catalog.WriteString("Ativa uma skill (procedimento especializado) pelo nome exato. Skills disponíveis:\n")

	for _, name := range names {
		s := r.skills[name]
		fmt.Fprintf(&catalog, "- %s: %s\n", s.Name, s.Description)
	}

	return responses.ToolUnionParam{
		OfFunction: &responses.FunctionToolParam{
			Name:        ToolUseSkill,
			Description: param.NewOpt(catalog.String()),
			Parameters: responses.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"skill_name": map[string]any{
						"type":        "string",
						"enum":        names,
						"description": "Nome exato da skill a ativar.",
					},
				},
				"required": []string{"skill_name"},
			},
		},
	}
}

// ReadSkillFileTool é a tool de leitura de arquivo bundled, que todo Subagente
// despachado leva ao lado de ReadSkillDirectoryTool. Não tem receiver de
// propósito: ela não enumera skill nenhuma, logo não precisa do registry — é
// exatamente por isso que serve ao despachado, que não deve ler catálogo. O alcance continua
// inteiro (HandleSkillCall resolve pelo registry completo), e os AGENT.md do
// Illustrator e do Teacher nomeiam a skill "mermaid" e esta tool, então as
// references seguem alcançáveis sem catálogo.
func ReadSkillFileTool() responses.ToolUnionParam {
	return responses.ToolUnionParam{
		OfFunction: &responses.FunctionToolParam{
			Name: ToolReadSkillFile,
			Description: param.NewOpt(
				"Lê um arquivo bundled (em scripts/, references/ ou assets/) de uma skill, " +
					"pelo nome dela. " + ReadSkillFileNoActivation,
			),
			Parameters: responses.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"skill_name": map[string]any{"type": "string"},
					"relative_path": map[string]any{
						"type":        "string",
						"description": "Caminho relativo à raiz da skill, ex: references/avancado.md",
					},
				},
				"required": []string{"skill_name", "relative_path"},
			},
		},
	}
}

// ReadSkillDirectoryTool é a irmã de ReadSkillFileTool, no mesmo formato e
// pelo mesmo motivo sem receiver: lista o que há numa pasta da skill, para o
// modelo achar o arquivo que depois lê com read_skill_file.
func ReadSkillDirectoryTool() responses.ToolUnionParam {
	return responses.ToolUnionParam{
		OfFunction: &responses.FunctionToolParam{
			Name: ToolReadSkillDirectory,
			Description: param.NewOpt(
				"Lista, recursivamente, os arquivos e as pastas dentro de uma pasta de uma skill, " +
					"pelo nome dela: um caminho por linha, relativo à raiz da skill (o formato que " +
					ToolReadSkillFile + " recebe), pasta terminando em /. " +
					"Não exige ativação prévia da skill: a pasta é listada pelo nome dela.",
			),
			Parameters: responses.FunctionParameters{
				"type": "object",
				"properties": map[string]any{
					"skill_name": map[string]any{"type": "string"},
					"relative_path": map[string]any{
						"type":        "string",
						"description": "Pasta relativa à raiz da skill, ex: references/; vazio lista a raiz",
					},
				},
				"required": []string{"skill_name", "relative_path"},
			},
		},
	}
}

// LoadSkills carrega o registro de skills do espelho do material do agente
// (~/.network_agent/resources/agents/skills), projetado do binário pelo
// config.MaterialFile: não depende de a árvore de fonte existir na máquina
// que roda o app. Falha de carga continua morrendo alto.
func LoadSkills() *Registry {

	root, err := config.MaterialFile("resources/agents/skills")
	if err != nil {
		log.Panicf("carregando skills: %v", err)
	}

	reg, err := LoadDir(root)
	if err != nil {
		log.Panicf("carregando skills: %v", err)
	}

	return reg
}
