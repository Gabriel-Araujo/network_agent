package skills

import (
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v3/responses"
)

// HandleSkillCall processa uma chamada de função relacionada a skills
// (use_skill / read_skill_file) e devolve o resultado para o modelo
// como uma saída de função (function call output).
func HandleSkillCall(call responses.ResponseFunctionToolCall) responses.ResponseInputItemUnionParam {
	reg := LoadSkills()

	switch call.Name {
	case ToolUseSkill:
		var args struct {
			SkillName string `json:"skill_name"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return responses.ResponseInputItemParamOfFunctionCallOutput(call.CallID, fmt.Sprintf("erro: argumentos inválidos: %v", err))
		}

		sk, ok := reg.Get(args.SkillName)
		if !ok {
			return responses.ResponseInputItemParamOfFunctionCallOutput(call.CallID, fmt.Sprintf("erro: skill %q não encontrada", args.SkillName))
		}
		return responses.ResponseInputItemParamOfFunctionCallOutput(call.CallID, sk.Body)

	case ToolReadSkillFile:
		var args struct {
			SkillName    string `json:"skill_name"`
			RelativePath string `json:"relative_path"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return responses.ResponseInputItemParamOfFunctionCallOutput(call.CallID, fmt.Sprintf("erro: argumentos inválidos: %v", err))
		}

		content, err := reg.ReadFile(args.SkillName, args.RelativePath)
		if err != nil {
			return responses.ResponseInputItemParamOfFunctionCallOutput(call.CallID, fmt.Sprintf("erro: %v", err))
		}
		return responses.ResponseInputItemParamOfFunctionCallOutput(call.CallID, content)

	default:
		return responses.ResponseInputItemParamOfFunctionCallOutput(call.CallID, "Error: Called invalid skill function")
	}
}
