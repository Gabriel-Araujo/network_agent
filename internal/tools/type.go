package tools

import (
	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
)

var FileTools []responses.ToolUnionParam = []responses.ToolUnionParam{
	responses.ToolUnionParam{
		OfFunction: &responses.FunctionToolParam{
			Name:        filetools.READ_TOOL_NAME,
			Description: param.NewOpt(filetools.READ_TOOL_DESCRIPTION),
			Parameters:  filetools.READ_TOOL_PARAMETERS,
		},
	},
	responses.ToolUnionParam{
		OfFunction: &responses.FunctionToolParam{
			Name:        filetools.WRITE_TOOL_NAME,
			Description: param.NewOpt(filetools.WRITE_TOOL_DESCRIPTION),
			Parameters:  filetools.WRITE_TOOL_PARAMETERS,
		},
	},
	responses.ToolUnionParam{
		OfFunction: &responses.FunctionToolParam{
			Name:        filetools.EDIT_TOOL_NAME,
			Description: param.NewOpt(filetools.EDIT_TOOL_DESCRIPTION),
			Parameters:  filetools.EDIT_TOOL_PARAMETERS,
		},
	},
	responses.ToolUnionParam{
		OfFunction: &responses.FunctionToolParam{
			Name:        filetools.SEARCH_TOOL_NAME,
			Description: param.NewOpt(filetools.SEARCH_TOOL_DESCRIPTION),
			Parameters:  filetools.SEARCH_TOOL_PARAMETERS,
		},
	},
}
