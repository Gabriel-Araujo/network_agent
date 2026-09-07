package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Gabriel-Araujo/network_agent/internal/skills"
	"github.com/Gabriel-Araujo/network_agent/internal/tools"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

type Agent struct {
	Client             openai.Client
	ModelName          string
	modelContextSize   int64
	availableTools     []responses.ToolUnionParam
	systemPrompt       string
	previousResponseID string
}

func Init(client openai.Client, modelName string, modelContextSize int64, tools []responses.ToolUnionParam, systemPrompt string) *Agent {
	return &Agent{
		client,
		modelName,
		modelContextSize,
		tools,
		systemPrompt,
		"",
	}
}

func (a *Agent) Chat(ctx context.Context, userMessage string) (string, error) {
	input := responses.ResponseNewParamsInputUnion{}

	if a.previousResponseID == "" {
		input = responses.ResponseNewParamsInputUnion{
			OfInputItemList: responses.ResponseInputParam{
				responses.ResponseInputItemParamOfMessage(a.systemPrompt, responses.EasyInputMessageRoleSystem),
				responses.ResponseInputItemParamOfMessage(userMessage, responses.EasyInputMessageRoleUser),
			},
		}

	} else {
		input = responses.ResponseNewParamsInputUnion{
			OfString: openai.String(userMessage),
		}
	}

	params := responses.ResponseNewParams{
		Model: a.ModelName,
		Input: input,
		Tools: a.availableTools,
	}

	if a.previousResponseID != "" {
		params.PreviousResponseID = openai.String(a.previousResponseID)
	}

	for {

		resp, err := a.Client.Responses.New(ctx, params)
		if err != nil {
			return "", err
		}

		a.previousResponseID = resp.ID

		var outputs []responses.ResponseInputItemUnionParam

		finished := true

		for _, item := range resp.Output {

			switch item.Type {

			case "function_call":

				finished = false

				call := item.AsFunctionCall()
				var result responses.ResponseInputItemUnionParam

				switch call.Name {

				case skills.ToolUseSkill, skills.ToolReadSkillFile:
					result = a.SkillCall(ctx, call)

				case tools.SUB_AGENT_TOOL_NAME:
					result = a.SubAgentCall(ctx, call)

				default:
					result = a.ToolCall(ctx, call)
				}

				outputs = append(outputs, result)

			case "message":

				// será retornado quando não houver mais tool
			}
		}

		if finished {
			return resp.OutputText(), nil
		}

		params = responses.ResponseNewParams{
			Model:              a.ModelName,
			PreviousResponseID: openai.String(resp.ID),
			Input:              responses.ResponseNewParamsInputUnion{OfInputItemList: outputs},
			Tools:              a.availableTools,
		}
	}
}

// ToolCall executa uma tool de operação direta (ex.: leitura/escrita/edição
// de arquivos) e devolve o resultado para o modelo.
func (a *Agent) ToolCall(ctx context.Context, call responses.ResponseFunctionToolCall) responses.ResponseInputItemUnionParam {
	return tools.CallFunction(ctx, call)
}

// SkillCall ativa ou lê arquivos de uma skill (use_skill / read_skill_file),
// devolvendo o corpo da skill (ou do arquivo bundlado) para o modelo.
func (a *Agent) SkillCall(ctx context.Context, call responses.ResponseFunctionToolCall) responses.ResponseInputItemUnionParam {
	return skills.HandleSkillCall(call)
}

// SubAgentCall cria um sub-agente dedicado a partir da task fornecida,
// executa o loop de conversação dele e devolve a resposta final para o modelo.
func (a *Agent) SubAgentCall(ctx context.Context, call responses.ResponseFunctionToolCall) responses.ResponseInputItemUnionParam {
	var args struct {
		Task string `json:"task"`
	}
	if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
		return responses.ResponseInputItemParamOfFunctionCallOutput(fmt.Sprintf("erro: argumentos inválidos: %v", err))
	}

	sub := &Agent{
		Client:         a.Client,
		ModelName:      a.ModelName,
		availableTools: a.availableTools,
		systemPrompt:   a.systemPrompt,
	}

	out, err := sub.Chat(ctx, args.Task)
	if err != nil {
		return responses.ResponseInputItemParamOfFunctionCallOutput(fmt.Sprintf("erro no sub-agente: %v", err))
	}

	return responses.ResponseInputItemParamOfFunctionCallOutput(out)
}
