package tools

import (
	"context"
	"encoding/json"

	"github.com/Gabriel-Araujo/network_agent/internal/logger"
	"github.com/Gabriel-Araujo/network_agent/internal/skills"
	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
	"github.com/openai/openai-go/v3/responses"
)

// log é o handle do pacote (uma declaração por pacote, não por arquivo).
var log = logger.Named("TOOLS")

const DEFAULT_ROOT_DIRECTORY = "."

var Tools = FileTools

func GetSkillAndTools() []responses.ToolUnionParam {
	reg := skills.LoadSkills()
	skillsTools := reg.BuildTools()

	all := append(skillsTools, Tools...)
	return append(all, SubAgentTool)
}

func CallFunction(
	ctx context.Context,
	call responses.ResponseFunctionToolCall,
) responses.ResponseInputItemUnionParam {
	log.Debugf("function '%s' called with arguments: [%s]\n", call.Name, call.Arguments)

	args := make(map[string]string)

	err := json.Unmarshal([]byte(call.Arguments), &args)
	if err != nil {
		log.Error("argumentos inválidos na chamada de função", "name", call.Name, "err", err)
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

	default:
		return responses.ResponseInputItemParamOfFunctionCallOutput(call.CallID, "Error: Called invalid function")
	}
}
