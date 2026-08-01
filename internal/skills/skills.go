package skills

import (
	"fmt"
	"log"
	"strings"

	"github.com/openai/openai-go/v3"
)

const (
	ToolUseSkill      = "use_skill"
	ToolReadSkillFile = "read_skill_file"
)

// BuildTools gera as tools enviadas ao modelo a cada request.
// Estágio de Discovery: só nome + descrição de cada skill aparecem aqui,
// nunca o corpo inteiro do SKILL.md.
func (r *Registry) BuildTools() []openai.ChatCompletionToolUnionParam {
	var catalog strings.Builder
	catalog.WriteString("Ativa uma skill (procedimento especializado) pelo nome exato. Skills disponíveis:\n")

	names := make([]string, 0, len(r.skills))
	for _, s := range r.skills {
		names = append(names, s.Name)
		fmt.Fprintf(&catalog, "- %s: %s\n", s.Name, s.Description)
	}

	useSkill := openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
		Name:        ToolUseSkill,
		Description: openai.String(catalog.String()),
		Parameters: openai.FunctionParameters{
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
	})

	readFile := openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
		Name: ToolReadSkillFile,
		Description: openai.String(
			"Lê um arquivo bundled (em scripts/, references/ ou assets/) de uma skill " +
				"já ativada com use_skill. Use somente depois de ativar a skill.",
		),
		Parameters: openai.FunctionParameters{
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
	})

	return []openai.ChatCompletionToolUnionParam{useSkill, readFile}
}

func LoadSkills() *Registry {

	reg, err := LoadDir("./resources/agent/skills")
	if err != nil {
		log.Fatalf("carregando skills: %v", err)
	}

	return reg
}
