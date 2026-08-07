package skills

import (
	"fmt"
	"log"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/paths"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
)

const (
	ToolUseSkill      = "use_skill"
	ToolReadSkillFile = "read_skill_file"
)

// BuildTools gera as tools enviadas ao modelo a cada request.
// Estágio de Discovery: só nome + descrição de cada skill aparecem aqui,
// nunca o corpo inteiro do SKILL.md.
func (r *Registry) BuildTools() []responses.ToolUnionParam {
	var catalog strings.Builder
	catalog.WriteString("Ativa uma skill (procedimento especializado) pelo nome exato. Skills disponíveis:\n")

	names := make([]string, 0, len(r.skills))
	for _, s := range r.skills {
		names = append(names, s.Name)
		fmt.Fprintf(&catalog, "- %s: %s\n", s.Name, s.Description)
	}

	useSkill := responses.ToolUnionParam{
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

	readFile := responses.ToolUnionParam{
		OfFunction: &responses.FunctionToolParam{
			Name: ToolReadSkillFile,
			Description: param.NewOpt(
				"Lê um arquivo bundled (em scripts/, references/ ou assets/) de uma skill " +
					"já ativada com use_skill. Use somente depois de ativar a skill.",
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

	return []responses.ToolUnionParam{useSkill, readFile}
}

func LoadSkills() *Registry {

	root, err := paths.RepoFile("resources/agent/skills")
	if err != nil {
		log.Fatalf("carregando skills: %v", err)
	}

	reg, err := LoadDir(root)
	if err != nil {
		log.Fatalf("carregando skills: %v", err)
	}

	return reg
}
