package skills

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3/responses"
)

// HandleSkillCall processa uma chamada de função relacionada a skills
// (use_skill / read_skill_file / read_skill_directory) e devolve o resultado para o modelo
// como uma saída de função (function call output).
//
// Devolve também a falha, quando houve: o erro é o canal que leva a falha de
// tool até o Chat (ticket 18), e sem ele a única notícia de que a skill não
// carregou é a frase seca que o modelo recebe.
func HandleSkillCall(call responses.ResponseFunctionToolCall) (responses.ResponseInputItemUnionParam, error) {
	reg := LoadSkills()

	switch call.Name {
	case ToolUseSkill:
		var args struct {
			SkillName string `json:"skill_name"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return responses.ResponseInputItemParamOfFunctionCallOutput(fmt.Sprintf("erro: argumentos inválidos: %v", err)),
				fmt.Errorf("argumentos ilegíveis na chamada de skill: %w", err)
		}

		sk, ok := reg.Get(args.SkillName)
		if !ok {
			return responses.ResponseInputItemParamOfFunctionCallOutput(fmt.Sprintf("erro: skill %q não encontrada", args.SkillName)),
				fmt.Errorf("skill %q não encontrada no registry", args.SkillName)
		}
		return responses.ResponseInputItemParamOfFunctionCallOutput(sk.Body), nil

	case ToolReadSkillFile:
		var args struct {
			SkillName    string `json:"skill_name"`
			RelativePath string `json:"relative_path"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return responses.ResponseInputItemParamOfFunctionCallOutput(fmt.Sprintf("erro: argumentos inválidos: %v", err)),
				fmt.Errorf("argumentos ilegíveis na chamada de skill: %w", err)
		}

		content, err := reg.ReadFile(args.SkillName, args.RelativePath)
		if err != nil {
			return responses.ResponseInputItemParamOfFunctionCallOutput(fmt.Sprintf("erro: %v", err)),
				fmt.Errorf("lendo o arquivo %q da skill %q: %w", args.RelativePath, args.SkillName, err)
		}
		return responses.ResponseInputItemParamOfFunctionCallOutput(content), nil

	case ToolReadSkillDirectory:
		var args struct {
			SkillName    string `json:"skill_name"`
			RelativePath string `json:"relative_path"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return responses.ResponseInputItemParamOfFunctionCallOutput(fmt.Sprintf("erro: argumentos inválidos: %v", err)),
				fmt.Errorf("argumentos ilegíveis na chamada de skill: %w", err)
		}

		items, err := reg.ListDir(args.SkillName, args.RelativePath)
		if err != nil {
			return responses.ResponseInputItemParamOfFunctionCallOutput(fmt.Sprintf("erro: %v", err)),
				fmt.Errorf("listando a pasta %q da skill %q: %w", args.RelativePath, args.SkillName, err)
		}
		return responses.ResponseInputItemParamOfFunctionCallOutput(strings.Join(items, "\n")), nil

	default:
		return responses.ResponseInputItemParamOfFunctionCallOutput("Error: Called invalid skill function"),
			errors.New("não é uma tool de skill")
	}
}

// ActivatedSkill devolve o nome da skill que uma chamada use_skill ordena
// ativar, e "" quando a chamada não é de ativação ou os argumentos não são
// legíveis. É o fato que o turno registra: sem ele, "o agente desenhou sem a
// skill" só se descobre perguntando ao modelo, que é como o dono descobriu
// (.agent/ui/issues/32-o-generico-desenha-sem-a-skill.md).
//
// O parse é o mesmo do HandleSkillCall, no mesmo encoding/json: dois parsers
// de strictness diferente sobre os mesmos bytes diriam coisas diferentes sobre
// a mesma chamada, e o nome registrado tem de ser o nome que carregou o corpo.
func ActivatedSkill(call responses.ResponseFunctionToolCall) string {
	if call.Name != ToolUseSkill {
		return ""
	}
	var args struct {
		SkillName string `json:"skill_name"`
	}
	if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
		return ""
	}
	return args.SkillName
}
