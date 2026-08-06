package tools

import (
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
)

const SUB_AGENT_TOOL_NAME = "sub_agent"

// SubAgentTool define a tool exposta ao modelo para delegar uma subtarefa
// a um sub-agente dedicado.
var SubAgentTool = responses.ToolUnionParam{
	OfFunction: &responses.FunctionToolParam{
		Name: SUB_AGENT_TOOL_NAME,
		Description: param.NewOpt(
			"Spawna um sub-agente para executar uma subtarefa específica de forma " +
				"isolada e retorna a resposta final dele. Use para delegar tarefas " +
				"longas, repetitivas ou paralelizáveis.",
		),
		Parameters: responses.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"task": map[string]any{
					"type":        "string",
					"description": "Instrução clara e autossuficiente para o sub-agente executar.",
				},
			},
			"required": []string{"task"},
		},
	},
}
