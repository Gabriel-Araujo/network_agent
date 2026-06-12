package tools

import (
	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

var FileReadTool = openai.ChatCompletionToolUnionParam{
	OfFunction: &openai.ChatCompletionFunctionToolParam{
		Function: shared.FunctionDefinitionParam{
			Name:        filetools.READ_TOOL_NAME,
			Description: openai.String(filetools.READ_TOOL_DESCRIPTION),
			Parameters:  filetools.READ_TOOL_PARAMETERS,
		},
	},
}

var FileEditTool = openai.ChatCompletionToolUnionParam{
	OfFunction: &openai.ChatCompletionFunctionToolParam{
		Function: shared.FunctionDefinitionParam{
			Name:        filetools.EDIT_TOOL_NAME,
			Description: openai.String(filetools.EDIT_TOOL_DESCRIPTION),
			Parameters:  filetools.EDIT_TOOL_PARAMETERS,
		},
	},
}

var FileWriteTool = openai.ChatCompletionToolUnionParam{
	OfFunction: &openai.ChatCompletionFunctionToolParam{
		Function: shared.FunctionDefinitionParam{
			Name:        filetools.WRITE_TOOL_NAME,
			Description: openai.String(filetools.WRITE_TOOL_DESCRIPTION),
			Parameters:  filetools.WRITE_TOOL_PARAMETERS,
		},
	},
}
