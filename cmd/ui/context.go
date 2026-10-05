package main

// contextWindow é a janela de contexto do modelo ativo — o teto que o medidor
// da UI compara com o uso medido — e de onde ela veio: "provider" quando o
// endpoint do provedor a informa, "config" quando é o context_size declarado na
// configuração (internal/agent.ContextWindow).
type contextWindow struct {
	Total  int64  `json:"total"`
	Source string `json:"source"`
}

// GetContextWindow devolve a janela de contexto do modelo ativo para o medidor
// da UI: o rodapé do dock (conversa vazia inclusive) e o bloco de uso dos
// balões.
//
// A janela é sondada no provedor a cada chamada, e não guardada no App: é
// barata, e é assim que ela acompanha o modelo recarregado — em LM Studio e
// Ollama a janela é definida na carga (ticket 11). Provedor que não a expõe ou
// não responde devolve o context_size declarado, com a origem marcada.
//
// Sem provedor/modelo ativos não há janela de ninguém: o erro estruturado diz
// isso, e a UI segue sem o número (nunca com um número inventado).
func (a *App) GetContextWindow() (contextWindow, error) {
	ag := a.currentAgent()
	if ag == nil {
		return contextWindow{}, newBridgeError(codeNoAgent,
			"Nenhum provedor e modelo configurados",
			"a janela de contexto é do modelo ativo, e não há nenhum conectado",
			"Abra Configurações e cadastre um provedor com ao menos um modelo.")
	}

	total, source := ag.ContextWindow(a.ctx)
	return contextWindow{Total: total, Source: source}, nil
}
