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
	ToolUseSkill      = "use_skill"
	ToolReadSkillFile = "read_skill_file"

	// ReadSkillFileNoActivation diz o que o HandleSkillCall realmente faz: lê o
	// arquivo bundled pelo nome da skill, sem conferir ativação em lugar nenhum.
	// A descrição antiga prometia "use somente depois de ativar a skill", regra
	// que nenhum código impõe, e a montagem da skill no prompt depende de a
	// leitura funcionar sem ativação. Mesma forma do
	// EDIT_MULTIPLE_MATCHES_NOTICE: a constante é a frase que o modelo lê.
	//
	// A frase não nomeia o use_skill de propósito: o Subagente despachado não
	// recebe essa tool (tools.GetFileAndSkillTools), e citar tool que não se
	// tem é o defeito que o modifier/AGENT.md corrigiu no ticket 20. O sentido
	// é o mesmo para quem tem o catálogo e para quem não tem.
	ReadSkillFileNoActivation = "Não exige ativação prévia da skill: o arquivo é lido pelo nome dela."
)

// BuildTools gera as tools de skill enviadas ao modelo a cada request: o
// catálogo de ativação mais a leitura de arquivo bundled. É o par que o agente
// genérico recebe; o Subagente despachado leva só o ReadSkillFileTool.
// Estágio de Discovery: só nome + descrição de cada skill aparecem aqui,
// nunca o corpo inteiro do SKILL.md.
func (r *Registry) BuildTools() []responses.ToolUnionParam {
	return []responses.ToolUnionParam{r.useSkillTool(), ReadSkillFileTool()}
}

// useSkillTool é a tool de ativação, e a única que carrega o catálogo: o nome e
// a descrição de cada skill do registry viajam na descrição dela e no enum do
// parâmetro. Quem não recebe esta tool não lê nome de skill nenhum.
func (r *Registry) useSkillTool() responses.ToolUnionParam {
	var catalog strings.Builder
	catalog.WriteString("Ativa uma skill (procedimento especializado) pelo nome exato. Skills disponíveis:\n")

	names := make([]string, 0, len(r.skills))
	for _, s := range r.skills {
		names = append(names, s.Name)
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

// ReadSkillFileTool é a tool de leitura de arquivo bundled, a única tool de
// skill do Subagente despachado. Não tem receiver de propósito: ela não
// enumera skill nenhuma, logo não precisa do registry — é exatamente por isso
// que serve ao despachado, que não deve ler catálogo. O alcance continua
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
