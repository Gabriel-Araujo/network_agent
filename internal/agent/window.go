package agent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Origem da janela de contexto. O valor acompanha o número até o disco
// (history.TokenUsage.ContextSource) e até o medidor da UI: número declarado
// não se passa por medido.
const (
	// WindowSourceProvider: a janela foi lida do endpoint do provedor.
	WindowSourceProvider = "provider"
	// WindowSourceConfig: o provedor não a expõe (ou não respondeu); o número é
	// o context_size declarado na configuração.
	WindowSourceConfig = "config"
)

// probeTimeout é o teto de uma sondagem inteira, e existe para que provedor que
// não responde não trave o chat: estourou o prazo, cai no declarado e segue.
// É var, e não const, para o teste provar o teto em milissegundos.
var probeTimeout = 2 * time.Second

// maxProbeBody limita o corpo lido de um endpoint de sondagem: a listagem de
// modelos de um provedor com muitos modelos é grande, e nada aqui precisa dela
// inteira.
const maxProbeBody = 1 << 20 // 1 MiB

// ContextWindow devolve a janela de contexto real do modelo ativo e a origem do
// número (WindowSourceProvider ou WindowSourceConfig).
//
// A janela é relida a cada pergunta, e não uma vez por conexão: LM Studio e
// Ollama a definem na carga do modelo, então um valor guardado envelhece em
// silêncio quando o modelo é recarregado (ticket 11). Provedor que não expõe a
// janela (ex.: DeepSeek), cujo modelo não está carregado, ou que não responde,
// devolve o context_size declarado — com a marca de origem, porque declarado
// não é medido.
func (a *Agent) ContextWindow(ctx context.Context) (int64, string) {
	if n := a.probeWindow(ctx); n > 0 {
		return n, WindowSourceProvider
	}
	return a.modelContextSize, WindowSourceConfig
}

// windowEndpoint é um endereço de sondagem e o leitor do corpo dele: cada
// provedor expõe a janela num endpoint e num campo diferentes.
type windowEndpoint struct {
	url  string
	pick func(body []byte) int64
}

// windowEndpoints monta os endereços de sondagem a partir da URL do provedor
// ativo. LM Studio e Ollama penduram a API de gestão fora do /v1, na origem; o
// vLLM responde no próprio /v1, no mesmo caminho da listagem
// OpenAI-compatible. Quem não expõe a janela responde 404/200 sem o campo — é
// isso que separa "expõe" de "não expõe", sem o app precisar saber de antemão
// qual dos provedores está do outro lado.
func (a *Agent) windowEndpoints() []windowEndpoint {
	u, err := url.Parse(a.providerURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil
	}
	origin := u.Scheme + "://" + u.Host

	return []windowEndpoint{
		{origin + "/api/v1/models", a.pickLMStudio},
		{origin + "/api/ps", a.pickOllama},
		{strings.TrimRight(a.providerURL, "/") + "/models", a.pickVLLM},
	}
}

// probeWindow pergunta a janela a todos os endpoints conhecidos de uma vez (só
// o do provedor ativo responde; os outros devolvem 404) e devolve o primeiro
// número encontrado, ou 0 quando nenhum expõe a janela. A ordem de preferência
// é a de windowEndpoints.
func (a *Agent) probeWindow(ctx context.Context) int64 {
	endpoints := a.windowEndpoints()
	if len(endpoints) == 0 {
		return 0
	}

	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	found := make([]int64, len(endpoints))
	var wg sync.WaitGroup
	for i := range endpoints {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			found[i] = endpoints[i].fetch(ctx, a.apiKey)
		}(i)
	}
	wg.Wait()

	for _, n := range found {
		if n > 0 {
			return n
		}
	}
	return 0
}

// fetch busca o endpoint e entrega o corpo ao leitor. Qualquer falha — rede,
// status, JSON, prazo — vale "não expõe": a sondagem devolve 0 e o chamador cai
// no declarado, porque não há erro que o medidor faça algo com.
func (e windowEndpoint) fetch(ctx context.Context, apiKey string) int64 {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.url, nil)
	if err != nil {
		return 0
	}
	// A chave vai à mesma origem que já recebe os turnos (um vLLM com
	// --api-key recusa a listagem sem ela). Endpoint local a ignora.
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxProbeBody))
	if err != nil {
		return 0
	}
	return e.pick(body)
}

// lmStudioModel é um item da listagem do LM Studio. O servidor devolve a lista
// sob `models` e identifica o modelo por `key`; a doc já mostrou `data`/`id`,
// então os dois nomes são aceitos — uma linha de estrutura por um provedor que
// muda de forma entre versões.
type lmStudioModel struct {
	Key             string `json:"key"`
	ID              string `json:"id"`
	LoadedInstances []struct {
		Config struct {
			ContextLength int64 `json:"context_length"`
		} `json:"config"`
	} `json:"loaded_instances"`
}

// pickLMStudio lê GET /api/v1/models. A janela é o `context_length` da instância
// carregada — a janela da carga — e nunca o `max_context_length`, que é o teto
// do arquivo do modelo mesmo para quem não está carregado (ticket 11). Modelo
// sem instância carregada não tem janela a informar: 0, e o chamador cai no
// declarado.
func (a *Agent) pickLMStudio(body []byte) int64 {
	var out struct {
		Models []lmStudioModel `json:"models"`
		Data   []lmStudioModel `json:"data"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return 0
	}

	for _, m := range append(out.Models, out.Data...) {
		if m.Key != a.ModelName && m.ID != a.ModelName {
			continue
		}
		if len(m.LoadedInstances) == 0 {
			continue
		}
		return m.LoadedInstances[0].Config.ContextLength
	}
	return 0
}

// pickOllama lê GET /api/ps do Ollama: a janela EFETIVA do runner
// (`context_length`), e não o `model_info.*.context_length` do /api/show, que é
// o teto do GGUF — o exemplo oficial mostra 4096 no ps e 131072 no show para o
// mesmo modelo (ticket 11). Casa pelo nome do modelo como ele foi carregado.
func (a *Agent) pickOllama(body []byte) int64 {
	var out struct {
		Models []struct {
			Name          string `json:"name"`
			Model         string `json:"model"`
			ContextLength int64  `json:"context_length"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return 0
	}

	for _, m := range out.Models {
		if m.Name == a.ModelName || m.Model == a.ModelName {
			return m.ContextLength
		}
	}
	return 0
}

// pickVLLM lê GET /v1/models (a listagem OpenAI-compatible) e casa o
// `max_model_len` do modelo ativo: no vLLM ele é a janela do engine em execução.
// Provedor OpenAI-compatible que não publica o campo (DeepSeek) responde 200 sem
// ele, e a sondagem devolve 0 — declarado.
func (a *Agent) pickVLLM(body []byte) int64 {
	var out struct {
		Data []struct {
			ID          string `json:"id"`
			MaxModelLen int64  `json:"max_model_len"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return 0
	}

	for _, m := range out.Data {
		if m.ID == a.ModelName {
			return m.MaxModelLen
		}
	}
	return 0
}
