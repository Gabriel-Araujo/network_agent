package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Gabriel-Araujo/network_agent/internal/skills"
	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
	"github.com/openai/openai-go/v3/responses"
)

const DEFAULT_ROOT_DIRECTORY = "."

var Tools = FileTools

func GetSkillAndTools() []responses.ToolUnionParam {
	reg := skills.LoadSkills()
	skillsTools := reg.BuildTools()

	return append(skillsTools, Tools...)
}

func CallFunction(
	ctx context.Context,
	call responses.ResponseFunctionToolCall,
) responses.ResponseInputItemUnionParam {
	log.Printf("function '%s' called with arguments: [%s]\n", call.Name, call.Arguments)

	args := make(map[string]string)

	err := json.Unmarshal([]byte(call.Arguments), &args)
	if err != nil {
		log.Default().Println(err)
		return responses.ResponseInputItemParamOfFunctionCallOutput(call.CallID, "Error: Failed to get args from function call object.")
	}

	args["workingDirectory"] = DEFAULT_ROOT_DIRECTORY

	switch call.Name {
	case filetools.READ_TOOL_NAME:
		return responses.ResponseInputItemParamOfFunctionCallOutput(call.CallID, filetools.ReadFile(args["workingDirectory"], args["filePath"]))
	case filetools.EDIT_TOOL_NAME:
		return responses.ResponseInputItemParamOfFunctionCallOutput(call.CallID, filetools.EditFile(
			args["workingDirectory"],
			args["filePath"],
			args["oldText"],
			args["newText"]))
	case filetools.WRITE_TOOL_NAME:
		return responses.ResponseInputItemParamOfFunctionCallOutput(call.CallID, filetools.WriteFile(
			args["workingDirectory"],
			args["filePath"],
			args["content"]))
	case filetools.SEARCH_TOOL_NAME:
		return responses.ResponseInputItemParamOfFunctionCallOutput(call.CallID, filetools.SearchPattern(
			args["workingDirectory"],
			args["path"],
			args["pattern"]))
	case skills.ToolUseSkill:
		reg := skills.LoadSkills()
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
	case skills.ToolReadSkillFile:
		reg := skills.LoadSkills()

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
		return responses.ResponseInputItemParamOfFunctionCallOutput(call.CallID, "Error: Called invalid function")
	}
}
