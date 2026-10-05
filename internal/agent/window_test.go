package agent

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// ---------- fixtures dos provedores ----------
//
// Os corpos abaixo são recortes dos payloads reais dos endpoints de sondagem.
// O do LM Studio foi capturado do servidor rodando (models[] com `key`, e a
// instância carregada com 65024 enquanto o arquivo anuncia 262144 — a diferença
// entre os dois campos é justamente o que este teste fixa).

const lmStudioReal = `{"models":[
	{"type":"llm","key":"google/gemma-4-12b-qat","display_name":"Gemma 4 12B QAT",
	 "loaded_instances":[{"id":"google/gemma-4-12b-qat","config":{"context_length":65024,"eval_batch_size":2048}}],
	 "max_context_length":262144},
	{"type":"llm","key":"google/gemma-4-e2b","loaded_instances":[],"max_context_length":131072}]}`

// lmStudioNaoCarregado é o mesmo modelo sem instância em memória: o provedor
// não tem janela a informar (o max_context_length vale até para quem não está
// carregado, e por isso não serve).
const lmStudioNaoCarregado = `{"models":[
	{"type":"llm","key":"google/gemma-4-12b-qat","loaded_instances":[],"max_context_length":262144}]}`

// ollamaPS é o GET /api/ps do Ollama: a janela efetiva do runner, 4096 —
// enquanto o /api/show do mesmo modelo anuncia o teto do GGUF, 131072.
const ollamaPS = `{"models":[
	{"name":"gemma4:latest","model":"gemma4:latest","size":6592285696,"context_length":4096}]}`

const vllmModels = `{"object":"list","data":[
	{"id":"google/gemma-4-12b-qat","object":"model","created":1,"owned_by":"vllm","max_model_len":32768}]}`

// openAIPlainModels é uma listagem OpenAI-compatible sem `max_model_len`, o que
// o DeepSeek devolve: 200 sem o campo.
const openAIPlainModels = `{"object":"list","data":[
	{"id":"google/gemma-4-12b-qat","object":"model","created":1,"owned_by":"deepseek"}]}`

// ---------- provedor de mentira da sondagem ----------

// windowProvider serve só os endpoints de sondagem: caminho presente responde
// 200 com o corpo, ausente responde 404 — como um provedor que não expõe a
// janela. Registra os caminhos pedidos e o Authorization de cada requisição.
type windowProvider struct {
	bodies map[string]string
	delay  time.Duration

	mu    sync.Mutex
	paths []string
	auth  []string
}

func (p *windowProvider) handler(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	p.paths = append(p.paths, r.URL.Path)
	p.auth = append(p.auth, r.Header.Get("Authorization"))
	body, ok := p.bodies[r.URL.Path]
	delay := p.delay
	p.mu.Unlock()

	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-r.Context().Done():
			// o cliente desistiu (prazo da sondagem): não segura o servidor
			return
		}
	}

	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, body)
}

// sonda roda uma sondagem contra um provedor de mentira e devolve o que o
// ContextWindow respondeu. O providerURL termina em /v1, como o da configuração
// real da UI.
func sonda(t *testing.T, p *windowProvider, model string, declared int64, apiKey string) (int64, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(p.handler))
	defer srv.Close()

	a := &Agent{
		ModelName:        model,
		providerURL:      srv.URL + "/v1",
		apiKey:           apiKey,
		modelContextSize: declared,
	}
	return a.ContextWindow(context.Background())
}

// ---------- testes ----------

// TestContextWindowLeAJanelaDaCargaDoLMStudio prova o caso que dá sentido ao
// ticket: o número é o context_length da instância carregada, não o
// max_context_length do arquivo (o declarado, 128000, é outro ainda).
func TestContextWindowLeAJanelaDaCargaDoLMStudio(t *testing.T) {
	p := &windowProvider{bodies: map[string]string{"/api/v1/models": lmStudioReal}}

	total, source := sonda(t, p, "google/gemma-4-12b-qat", 128000, "")
	if total != 65024 {
		t.Errorf("janela %d, esperava 65024 (o context_length da instância carregada; 262144 é o teto do arquivo)", total)
	}
	if source != WindowSourceProvider {
		t.Errorf("origem %q, esperava %q", source, WindowSourceProvider)
	}
}

// TestContextWindowLeOllamaEoVLLM prova os outros dois endpoints verificados no
// ticket 11, com a chave do provedor indo junto na listagem OpenAI-compatible
// (um vLLM com --api-key recusa a sondagem sem ela).
func TestContextWindowLeOllamaEoVLLM(t *testing.T) {
	t.Run("ollama: a janela efetiva do /api/ps", func(t *testing.T) {
		p := &windowProvider{bodies: map[string]string{"/api/ps": ollamaPS}}

		total, source := sonda(t, p, "gemma4:latest", 128000, "")
		if total != 4096 {
			t.Errorf("janela %d, esperava 4096 (o teto do GGUF, 131072, é do /api/show)", total)
		}
		if source != WindowSourceProvider {
			t.Errorf("origem %q, esperava %q", source, WindowSourceProvider)
		}
	})

	t.Run("vllm: o max_model_len do /v1/models", func(t *testing.T) {
		p := &windowProvider{bodies: map[string]string{"/v1/models": vllmModels}}

		total, source := sonda(t, p, "google/gemma-4-12b-qat", 128000, "chave-do-provedor")
		if total != 32768 {
			t.Errorf("janela %d, esperava 32768", total)
		}
		if source != WindowSourceProvider {
			t.Errorf("origem %q, esperava %q", source, WindowSourceProvider)
		}

		p.mu.Lock()
		defer p.mu.Unlock()
		var vistos int
		for i, path := range p.paths {
			if path != "/v1/models" {
				continue
			}
			vistos++
			if p.auth[i] != "Bearer chave-do-provedor" {
				t.Errorf("a sondagem em %s mandou Authorization %q, esperava a chave do provedor", path, p.auth[i])
			}
		}
		if vistos == 0 {
			t.Error("a sondagem nunca pediu /v1/models")
		}
	})

	t.Run("url do provedor com barra no fim", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc((&windowProvider{
			bodies: map[string]string{"/v1/models": vllmModels},
		}).handler))
		defer srv.Close()

		a := &Agent{ModelName: "google/gemma-4-12b-qat", providerURL: srv.URL + "/v1/", modelContextSize: 128000}
		if total, source := a.ContextWindow(context.Background()); total != 32768 || source != WindowSourceProvider {
			t.Errorf("devolveu %d/%q, esperava 32768/%q", total, source, WindowSourceProvider)
		}
	})
}

// TestContextWindowMarcaODeclaradoQuandoOProvedorNaoAInforma prova o fallback
// em cada forma de "não expõe": modelo fora de memória, outro modelo carregado,
// listagem sem o campo e provedor que só devolve 404. Em todas, o número é o
// context_size da configuração — com a origem marcada, porque declarado não se
// passa por medido.
func TestContextWindowMarcaODeclaradoQuandoOProvedorNaoAInforma(t *testing.T) {
	const declared = 128000

	casos := []struct {
		nome   string
		bodies map[string]string
	}{
		{"lm studio com o modelo fora de memória", map[string]string{"/api/v1/models": lmStudioNaoCarregado}},
		{"lm studio com outro modelo carregado", map[string]string{"/api/v1/models": lmStudioReal}},
		{"listagem OpenAI-compatible sem a janela (DeepSeek)", map[string]string{"/v1/models": openAIPlainModels}},
		{"provedor que só devolve 404", map[string]string{}},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			p := &windowProvider{bodies: c.bodies}

			model := "google/gemma-4-12b-qat"
			if c.bodies["/api/v1/models"] == lmStudioReal {
				model = "outro/modelo" // o payload real tem o gemma carregado, não este
			}

			total, source := sonda(t, p, model, declared, "")
			if total != declared {
				t.Errorf("janela %d, esperava o declarado %d", total, declared)
			}
			if source != WindowSourceConfig {
				t.Errorf("origem %q, esperava %q", source, WindowSourceConfig)
			}
		})
	}

	t.Run("provedor fora do ar", func(t *testing.T) {
		a := &Agent{ModelName: "google/gemma-4-12b-qat", providerURL: "http://127.0.0.1:1/v1", modelContextSize: declared}
		if total, source := a.ContextWindow(context.Background()); total != declared || source != WindowSourceConfig {
			t.Errorf("devolveu %d/%q, esperava %d/%q", total, source, declared, WindowSourceConfig)
		}
	})

	t.Run("endereço do provedor que não é URL", func(t *testing.T) {
		a := &Agent{ModelName: "google/gemma-4-12b-qat", providerURL: "://sem-esquema", modelContextSize: declared}
		if total, source := a.ContextWindow(context.Background()); total != declared || source != WindowSourceConfig {
			t.Errorf("devolveu %d/%q, esperava %d/%q", total, source, declared, WindowSourceConfig)
		}
	})
}

// TestContextWindowNaoTravaQuandoOProvedorNaoResponde prova o item do ticket:
// provedor mudo não derruba o app nem prende o chat — a sondagem estoura o
// próprio prazo e devolve o declarado. O prazo é encurtado para o teste não
// esperar os 2s de produção.
func TestContextWindowNaoTravaQuandoOProvedorNaoResponde(t *testing.T) {
	defer func(anterior time.Duration) { probeTimeout = anterior }(probeTimeout)
	probeTimeout = 200 * time.Millisecond

	p := &windowProvider{bodies: map[string]string{"/api/v1/models": lmStudioReal}, delay: 10 * time.Second}
	srv := httptest.NewServer(http.HandlerFunc(p.handler))
	defer srv.Close()

	a := &Agent{ModelName: "google/gemma-4-12b-qat", providerURL: srv.URL + "/v1", modelContextSize: 128000}

	start := time.Now()
	total, source := a.ContextWindow(context.Background())
	elapsed := time.Since(start)

	if total != 128000 || source != WindowSourceConfig {
		t.Errorf("devolveu %d/%q, esperava o declarado 128000/%q", total, source, WindowSourceConfig)
	}
	if elapsed > time.Second {
		t.Errorf("a sondagem demorou %v com prazo de %v: o teto não está segurando o provedor mudo", elapsed, probeTimeout)
	}
}
