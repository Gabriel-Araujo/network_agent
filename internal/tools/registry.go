package tools

import (
	"encoding/json"
	"log"

	file_tools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
	"github.com/openai/openai-go/v3"
)

var DEFAULT_ROOT_DIRECTORY = "."
var Tools = []openai.ChatCompletionToolUnionParam{File_read_tool, File_edit_tool}

func Call_function(function_call openai.ChatCompletionChunkChoiceDeltaToolCall, verbose bool) openai.ChatCompletionMessageParamUnion {
	if verbose {
		log.Default().Printf("function '%s' called with arguments: [%s]", function_call.Function.Name, function_call.Function.Arguments)
	}

	args := make(map[string]string)

	err := json.Unmarshal([]byte(function_call.Function.Arguments), &args)
	if err != nil {
		log.Default().Println(err)
		return openai.ToolMessage("Error: Failed to get args from function call object.", function_call.ID)
	}

	args["working_directory"] = DEFAULT_ROOT_DIRECTORY

	switch function_call.Function.Name {
	case file_tools.READ_TOOL_NAME:
		return openai.ToolMessage(
			file_tools.ReadFile(args["working_directory"], args["file_path"]),
			function_call.ID)
	case file_tools.EDIT_TOOL_NAME:
		return openai.ToolMessage(file_tools.EditFile(
			args["working_directory"],
			args["file_path"],
			args["old_text"],
			args["new_text"]),
			function_call.ID)
	default:
		return openai.ToolMessage("Error: Called invalid function", function_call.ID)
	}
}
