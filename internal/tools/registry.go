package tools

import (
	"encoding/json"
	"log"

	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
	"github.com/openai/openai-go/v3"
)

var DEFAULT_ROOT_DIRECTORY = "."
var Tools = []openai.ChatCompletionToolUnionParam{FileReadTool, FileEditTool, FileWriteTool}

func CallFunction(functionCall openai.ChatCompletionChunkChoiceDeltaToolCall, verbose bool) openai.ChatCompletionMessageParamUnion {
	if verbose {
		log.Default().Printf("function '%s' called with arguments: [%s]", functionCall.Function.Name, functionCall.Function.Arguments)
	}

	args := make(map[string]string)

	err := json.Unmarshal([]byte(functionCall.Function.Arguments), &args)
	if err != nil {
		log.Default().Println(err)
		return openai.ToolMessage("Error: Failed to get args from function call object.", functionCall.ID)
	}

	args["workingDirectory"] = DEFAULT_ROOT_DIRECTORY

	switch functionCall.Function.Name {
	case filetools.READ_TOOL_NAME:
		return openai.ToolMessage(
			filetools.ReadFile(args["workingDirectory"], args["filePath"]),
			functionCall.ID)
	case filetools.EDIT_TOOL_NAME:
		return openai.ToolMessage(filetools.EditFile(
			args["workingDirectory"],
			args["filePath"],
			args["oldText"],
			args["newText"]),
			functionCall.ID)
	case filetools.WRITE_TOOL_NAME:
		return openai.ToolMessage(filetools.WriteFile(
			args["workingDirectory"],
			args["filePath"],
			args["content"]),
			functionCall.ID)
	default:
		return openai.ToolMessage("Error: Called invalid function", functionCall.ID)
	}
}
