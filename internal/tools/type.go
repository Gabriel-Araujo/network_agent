package tools

import (
	file_tools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

var File_read_tool = openai.ChatCompletionToolUnionParam{
	OfFunction: &openai.ChatCompletionFunctionToolParam{
		Function: shared.FunctionDefinitionParam{
			Name:        file_tools.READ_TOOL_NAME,
			Description: openai.String(file_tools.READ_TOOL_DESCRIPTION),
			Parameters:  file_tools.READ_TOOL_PARAMETERS,
		},
	},
}

var File_edit_tool = openai.ChatCompletionToolUnionParam{
	OfFunction: &openai.ChatCompletionFunctionToolParam{
		Function: shared.FunctionDefinitionParam{
			Name:        file_tools.EDIT_TOOL_NAME,
			Description: openai.String(file_tools.EDIT_TOOL_DESCRIPTION),
			Parameters:  file_tools.EDIT_TOOL_PARAMETERS,
		},
	},
}
