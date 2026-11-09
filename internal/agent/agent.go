package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/history"
	"github.com/Gabriel-Araujo/network_agent/internal/skills"
	"github.com/Gabriel-Araujo/network_agent/internal/tools"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

// Agent é o cliente de um modelo em um provedor. É **sem estado**: nada da
// conversa vive aqui, e todo turno recebe o transcript do chamador.
type Agent struct {
	Client      openai.Client
	ModelName   string
	providerURL string
	apiKey      string
	// modelContextSize é o context_size declarado na configuração para este
	// modelo. É o piso da janela de contexto: quem a sonda no provedor é
	// ContextWindow (window.go), que cai nele quando o provedor não a expõe.
	modelContextSize int64
	availableTools   []responses.ToolUnionParam
	systemPrompt     string
}

// Init monta um Agent. providerURL e apiKey são a conexão — a URL do provedor
// é por onde ContextWindow sonda a janela de contexto, que o cliente do SDK não
// guarda em lugar nenhum legível.
func Init(client openai.Client, providerURL, apiKey, modelName string, modelContextSize int64, tools []responses.ToolUnionParam, systemPrompt string) *Agent {
	return &Agent{
		Client:           client,
		ModelName:        modelName,
		providerURL:      providerURL,
		apiKey:           apiKey,
		modelContextSize: modelContextSize,
		availableTools:   tools,
		systemPrompt:     systemPrompt,
	}
}

// Chat envia um turno ao provedor e devolve a resposta crua dele.
//
// O Agent é sem estado: o transcript vem do chamador (na UI, o que está no
// disco) e o payload é montado aqui, como [system] + transcript + a mensagem
// nova. Retomar uma conversa é passar o transcript dela, e trocar de conversa
// não exige reset nenhum.
//
// Devolve a resposta do SDK sem projetar nada — cada front lê o que precisa
// (OutputText, Usage). O provedor pode não reportar uso:
// resp.Usage.JSON.InputTokens.Valid distingue "não reportou" de zero.
//
// Um turno pode gastar várias idas ao provedor por causa das ferramentas. Como
// não há estado no servidor, cada volta reenvia o que já foi trocado: o input
// acumulado mais as chamadas e os resultados desta resposta.
func (a *Agent) Chat(ctx context.Context, transcript []history.Message, userMessage string) (*responses.Response, error) {
	input := a.buildInput(transcript, userMessage)

	for {
		resp, err := a.Client.Responses.New(ctx, responses.ResponseNewParams{
			Model: a.ModelName,
			Input: responses.ResponseNewParamsInputUnion{OfInputItemList: input},
			Tools: a.availableTools,
		})
		if err != nil {
			return nil, err
		}

		var calls, results []responses.ResponseInputItemUnionParam

		for _, item := range resp.Output {
			if item.Type != "function_call" {
				// Itens de mensagem ficam de fora do reenvio: o texto final é
				// o que volta ao chamador, e o transcript do próximo turno vem
				// do Store, não daqui.
				continue
			}

			call := item.AsFunctionCall()
			calls = append(calls, responses.ResponseInputItemParamOfFunctionCall(call.Arguments, call.CallID, call.Name))
			results = append(results, a.callTool(ctx, call))
		}

		if len(calls) == 0 {
			return resp, nil
		}

		input = append(input, calls...)
		input = append(input, results...)
	}
}

// buildInput monta o input de um turno: o system prompt, o transcript e a
// mensagem nova, nessa ordem. O system prompt vai à frente em todo turno, do
// valor atual do Agent — não é persistido por conversa.
func (a *Agent) buildInput(transcript []history.Message, userMessage string) []responses.ResponseInputItemUnionParam {
	items := make([]responses.ResponseInputItemUnionParam, 0, len(transcript)+2)
	items = append(items, responses.ResponseInputItemParamOfMessage(a.systemPrompt, responses.EasyInputMessageRoleSystem))
	items = append(items, transcriptItems(transcript)...)
	items = append(items, responses.ResponseInputItemParamOfMessage(userMessage, responses.EasyInputMessageRoleUser))
	return items
}

// transcriptItems projeta o transcript gravado nos itens de input do provedor.
//
// O v1 reenvia só texto, uma mensagem por Message: nenhuma tool call real chega
// à UI hoje, então itens verbatim dentro do Message seriam código morto (ticket
// 04, Q7). Ficam de fora as mensagens de conteúdo vazio e as de status error —
// um turno que falhou não é contexto, é ruído que o modelo repetiria.
func transcriptItems(transcript []history.Message) []responses.ResponseInputItemUnionParam {
	items := make([]responses.ResponseInputItemUnionParam, 0, len(transcript))
	for _, m := range transcript {
		if m.Status == history.StatusError || strings.TrimSpace(m.Content) == "" {
			continue
		}

		role := responses.EasyInputMessageRoleUser
		if m.Role == history.RoleAgent {
			role = responses.EasyInputMessageRoleAssistant
		}

		items = append(items, responses.ResponseInputItemParamOfMessage(m.Content, role))
	}
	return items
}

// callTool despacha uma chamada de ferramenta ao executor certo e devolve o
// resultado já com o call id preenchido: sem estado no servidor, é o call id
// que casa resultado e chamada quando o turno é reenviado.
func (a *Agent) callTool(ctx context.Context, call responses.ResponseFunctionToolCall) responses.ResponseInputItemUnionParam {
	var result responses.ResponseInputItemUnionParam

	switch {

	case !a.offers(call.Name):
		// A tool this Agent was not given never runs, whatever the model
		// named: it is how the Modifier stays without WriteFile and how a
		// dispatched Subagente stays without sub_agent (no nesting).
		log.Warn("tool não oferecida recusada", "name", call.Name)
		result = responses.ResponseInputItemParamOfFunctionCallOutput(
			fmt.Sprintf("Error: tool %q was not offered to you. Use only the tools in your list.", call.Name))

	case call.Name == skills.ToolUseSkill, call.Name == skills.ToolReadSkillFile:
		result = a.SkillCall(ctx, call)

	case call.Name == tools.SUB_AGENT_TOOL_NAME:
		result = a.SubAgentCall(ctx, call)

	default:
		result = a.ToolCall(ctx, call)
	}

	if result.OfFunctionCallOutput != nil {
		result.OfFunctionCallOutput.CallID = openai.String(call.CallID)
	}

	return result
}

// offers reports whether this Agent's request carries the named tool. It is
// the whole of the "refuse what was not offered" rule: availableTools is what
// the provider was sent, so it is the only honest list.
func (a *Agent) offers(name string) bool {
	return slices.ContainsFunc(a.availableTools, func(t responses.ToolUnionParam) bool {
		return t.OfFunction != nil && t.OfFunction.Name == name
	})
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
// O sub-agente não herda o transcript do pai: ele nasce sem estado e recebe a
// task como sua única mensagem.
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
		providerURL:    a.providerURL,
		apiKey:         a.apiKey,
		availableTools: a.availableTools,
		systemPrompt:   a.systemPrompt,
	}

	resp, err := sub.Chat(ctx, nil, args.Task)
	if err != nil {
		return responses.ResponseInputItemParamOfFunctionCallOutput(fmt.Sprintf("erro no sub-agente: %v", err))
	}

	return responses.ResponseInputItemParamOfFunctionCallOutput(resp.OutputText())
}
