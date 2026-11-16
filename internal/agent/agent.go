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
	// workspace é a pasta do usuário: a única raiz de escrita das file tools,
	// e a raiz contra a qual um caminho sem o prefixo de material resolve.
	// Quem a fornece é o front-end (ticket 05, Q3) — a UI dá a pasta escolhida,
	// o cmd/cli dá o CWD, porque no terminal o diretório onde o usuário está
	// parado **é** a escolha dele. Vazia significa "nenhuma escolhida", e as
	// tools recusam com texto próprio; não é estado de erro do Agent.
	workspace    string
	systemPrompt string
}

// Init monta um Agent. providerURL e apiKey são a conexão — a URL do provedor
// é por onde ContextWindow sonda a janela de contexto, que o cliente do SDK não
// guarda em lugar nenhum legível.
func Init(client openai.Client, providerURL, apiKey, modelName string, modelContextSize int64, tools []responses.ToolUnionParam, systemPrompt, workspace string) *Agent {
	return &Agent{
		Client:           client,
		ModelName:        modelName,
		providerURL:      providerURL,
		apiKey:           apiKey,
		modelContextSize: modelContextSize,
		availableTools:   tools,
		workspace:        workspace,
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
// Devolve também o nome de cada skill que o modelo ordenou ativar no turno,
// na ordem das chamadas: é o fato que o turno loga ao lado dos blocos de
// diagrama, e sem ele "desenhou sem a skill" só se descobre perguntando ao
// modelo (ticket 32 do mapa da UI).
//
// Devolve também as falhas de tool do turno, na ordem em que aconteceram
// (ticket 18). Falha de tool **é** degradação: antes só a string de erro que o
// modelo lê existia, e o app ficava sabendo da recusa no máximo pela boca dele.
// A lista vem vazia num turno limpo, e vem junto com o erro quando o provedor
// derruba o turno depois de uma tool ter falhado.
//
// Um turno pode gastar várias idas ao provedor por causa das ferramentas. Como
// não há estado no servidor, cada volta reenvia o que já foi trocado: o input
// acumulado mais as chamadas e os resultados desta resposta.
func (a *Agent) Chat(ctx context.Context, transcript []history.Message, userMessage string) (*responses.Response, []tools.Failure, []string, error) {
	input := a.buildInput(transcript, userMessage)
	var failures []tools.Failure
	var activated []string

	for {
		resp, err := a.Client.Responses.New(ctx, responses.ResponseNewParams{
			Model: a.ModelName,
			Input: responses.ResponseNewParamsInputUnion{OfInputItemList: input},
			Tools: a.availableTools,
		})
		if err != nil {
			return nil, failures, activated, err
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
			result, failed := a.callTool(ctx, call)
			results = append(results, result)
			failures = append(failures, failed...)

			// A ativação só conta depois do dispatch, e só quando ele
			// entregou o corpo: a tool que este Agent não oferece é
			// recusada sem falha nenhuma (a.offers), e uma skill fora do
			// registry volta como falha. Registrar a intenção em vez da
			// entrega diria "ativou" num turno em que o modelo nunca leu
			// a regra — o oposto do que a linha existe para medir.
			if skill := skills.ActivatedSkill(call); skill != "" && a.offers(call.Name) && len(failed) == 0 {
				activated = append(activated, skill)
			}
		}

		if len(calls) == 0 {
			return resp, failures, activated, nil
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
//
// É o gargalo único das quatro famílias de tool, e por isso o ponto onde as
// falhas se acumulam (ticket 18, §2). A recusa de tool não oferecida **não**
// entra: é a regra operando como desenhada, não degradação, e já é um log.Warn.
func (a *Agent) callTool(ctx context.Context, call responses.ResponseFunctionToolCall) (responses.ResponseInputItemUnionParam, []tools.Failure) {
	var result responses.ResponseInputItemUnionParam
	var failures []tools.Failure

	switch {

	case !a.offers(call.Name):
		// A tool this Agent was not given never runs, whatever the model
		// named: it is how the Modifier stays without WriteFile and how a
		// dispatched Subagente stays without sub_agent (no nesting).
		log.Warn("tool não oferecida recusada", "name", call.Name)
		result = responses.ResponseInputItemParamOfFunctionCallOutput(
			fmt.Sprintf("Error: tool %q was not offered to you. Use only the tools in your list.", call.Name))

	case call.Name == skills.ToolUseSkill, call.Name == skills.ToolReadSkillFile:
		var err error
		result, err = a.SkillCall(ctx, call)
		if err != nil {
			failures = append(failures, tools.Failure{Tool: call.Name, Err: err})
		}

	case call.Name == tools.SUB_AGENT_TOOL_NAME:
		result, failures = a.SubAgentCall(ctx, call)

	default:
		var failed *tools.Failure
		result, failed = a.ToolCall(ctx, call)
		if failed != nil {
			failures = append(failures, *failed)
		}
	}

	if result.OfFunctionCallOutput != nil {
		result.OfFunctionCallOutput.CallID = openai.String(call.CallID)
	}

	return result, failures
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
// de arquivos) e devolve o resultado para o modelo. O Workspace deste Agent vai
// junto: é a raiz de escrita, e a outra raiz — o material do agente — o
// CallFunction resolve sozinho, porque é global aos dois front-ends.
func (a *Agent) ToolCall(ctx context.Context, call responses.ResponseFunctionToolCall) (responses.ResponseInputItemUnionParam, *tools.Failure) {
	return tools.CallFunction(ctx, call, a.workspace)
}

// SkillCall ativa ou lê arquivos de uma skill (use_skill / read_skill_file),
// devolvendo o corpo da skill (ou do arquivo bundlado) para o modelo.
func (a *Agent) SkillCall(ctx context.Context, call responses.ResponseFunctionToolCall) (responses.ResponseInputItemUnionParam, error) {
	return skills.HandleSkillCall(call)
}

// SubAgentCall cria um sub-agente dedicado a partir da task fornecida,
// executa o loop de conversação dele e devolve a resposta final para o modelo.
// O sub-agente não herda o transcript do pai: ele nasce sem estado e recebe a
// task como sua única mensagem.
//
// As falhas de tool do aninhado se **compõem** nas do Chat de fora: o Chat do
// sub-agente roda dentro de uma tool deste, então uma leitura recusada lá
// dentro é degradação deste turno igual (ticket 18, §2).
func (a *Agent) SubAgentCall(ctx context.Context, call responses.ResponseFunctionToolCall) (responses.ResponseInputItemUnionParam, []tools.Failure) {
	var args struct {
		Task string `json:"task"`
	}
	if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
		return responses.ResponseInputItemParamOfFunctionCallOutput(fmt.Sprintf("erro: argumentos inválidos: %v", err)),
			[]tools.Failure{{Tool: call.Name, Err: fmt.Errorf("argumentos ilegíveis na chamada do sub-agente: %w", err)}}
	}

	sub := &Agent{
		Client:         a.Client,
		ModelName:      a.ModelName,
		providerURL:    a.providerURL,
		apiKey:         a.apiKey,
		availableTools: a.availableTools,
		workspace:      a.workspace,
		systemPrompt:   a.systemPrompt,
	}

	// A ativação do aninhado não sobe: o SubAgentCall devolve o resultado da
	// tool e as falhas, e o fato de skill é do turno de fora, que é quem loga.
	resp, failures, _, err := sub.Chat(ctx, nil, args.Task)
	if err != nil {
		return responses.ResponseInputItemParamOfFunctionCallOutput(fmt.Sprintf("erro no sub-agente: %v", err)),
			append(failures, tools.Failure{Tool: call.Name, Err: fmt.Errorf("o turno do sub-agente falhou: %w", err)})
	}

	return responses.ResponseInputItemParamOfFunctionCallOutput(resp.OutputText()), failures
}
