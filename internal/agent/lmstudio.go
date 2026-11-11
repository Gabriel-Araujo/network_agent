package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Os dois deadlines da conversa com o LM Studio. São `var`, e não `const`,
// pelo mesmo motivo do seam de composição da UI: o teste precisa encurtá-los
// para provar qual dos dois governa a carga sem dormir segundos de verdade.
var (
	// lmStudioProbeTimeout dimensiona a sondagem: um GET no servidor local.
	lmStudioProbeTimeout = 3 * time.Second

	// lmStudioLoadTimeout dimensiona a carga, que é outra ordem de grandeza:
	// alargar a janela é descarregar e carregar o modelo de novo — 19s medidos
	// para o MENOR dos modelos locais (gemma-4-e4b a 128k, disco quente) e
	// mais a frio. Os 3s da sondagem faziam toda carga falhar, e com ela todo
	// o Connect (ticket 13).
	//
	// Generoso de propósito: o servidor fora do ar falha na hora, por conexão
	// recusada, sem gastar deadline nenhum. Este número só limita o caso
	// patológico do servidor que aceita a conexão e nunca responde a carga.
	lmStudioLoadTimeout = 5 * time.Minute
)

// errLMStudioLoadDeadline é a causa que o ctx da carga carrega, para a
// mensagem distinguir "a carga não terminou" de "o servidor não respondeu" —
// as duas chegavam ao usuário como o mesmo "context deadline exceeded".
var errLMStudioLoadDeadline = errors.New("a carga do modelo não terminou no prazo")

// ensureLMStudioContext aumenta a janela da instância do LM Studio quando a
// configuração do modelo pede mais tokens do que a carga atual. A API
// OpenAI-compatible não tem um campo padrão para isso: a janela é uma opção da
// carga em /api/v1/models/load.
//
// Provedores que não são LM Studio são ignorados: o endpoint de sondagem não
// existe neles, e um Connect genérico não deve depender de uma API proprietária.
func ensureLMStudioContext(baseURL, apiKey, model string, desired int64) error {
	if desired <= 0 {
		return nil
	}

	root, err := lmStudioAPIBase(baseURL)
	if err != nil {
		return nil
	}

	probeCtx, cancelProbe := context.WithTimeout(context.Background(), lmStudioProbeTimeout)
	defer cancelProbe()

	models, supported, err := fetchLMStudioModels(probeCtx, root, apiKey)
	if err != nil || !supported {
		return err
	}

	current := int64(0)
	for _, candidate := range append(models.Models, models.Data...) {
		if candidate.Key != model && candidate.ID != model {
			continue
		}
		logLMStudioInstances(model, candidate)
		if len(candidate.LoadedInstances) > 0 {
			current = candidate.LoadedInstances[0].Config.ContextLength
		}
		break
	}
	if current >= desired {
		return nil
	}

	body, err := json.Marshal(struct {
		Model         string `json:"model"`
		ContextLength int64  `json:"context_length"`
	}{Model: model, ContextLength: desired})
	if err != nil {
		return fmt.Errorf("serializando carga do LM Studio: %w", err)
	}

	// A carga tem deadline próprio, e não o da sondagem: ver lmStudioLoadTimeout.
	loadCtx, cancelLoad := context.WithTimeoutCause(context.Background(), lmStudioLoadTimeout, errLMStudioLoadDeadline)
	defer cancelLoad()

	req, err := http.NewRequestWithContext(loadCtx, http.MethodPost, root+"/models/load", strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("criando carga do LM Studio: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	// A carga é o trecho lento do Connect: registrada nas duas pontas, porque
	// sem isso o app que demora dezenas de segundos para subir não tem como
	// ser distinguido do app travado.
	log.Info("carregando o modelo no LM Studio para alargar a janela",
		"model", model, "context_length_atual", current, "context_length_pedido", desired, "prazo", lmStudioLoadTimeout)
	started := time.Now()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// context.Cause distingue as duas causas que caíam no mesmo
		// "context deadline exceeded": a carga que não terminou e o servidor
		// que não respondeu.
		if cause := context.Cause(loadCtx); errors.Is(cause, errLMStudioLoadDeadline) {
			return fmt.Errorf("carregando modelo %q no LM Studio: %w (esperou %s)",
				model, cause, time.Since(started).Round(time.Second))
		}
		return fmt.Errorf("carregando modelo %q no LM Studio: %w", model, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		return nil
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return fmt.Errorf("LM Studio recusou a janela %d para %q: HTTP %d: %s", desired, model, resp.StatusCode, strings.TrimSpace(string(message)))
	}
	log.Info("modelo carregado no LM Studio",
		"model", model, "context_length", desired, "duracao", time.Since(started).Round(time.Millisecond))
	return nil
}

func fetchLMStudioModels(ctx context.Context, root, apiKey string) (struct {
	Models []lmStudioModel `json:"models"`
	Data   []lmStudioModel `json:"data"`
}, bool, error) {
	var result struct {
		Models []lmStudioModel `json:"models"`
		Data   []lmStudioModel `json:"data"`
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, root+"/models", nil)
	if err != nil {
		return result, false, nil
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, false, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		return result, false, nil
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return result, true, fmt.Errorf("sondagem do LM Studio falhou: HTTP %d", resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return result, false, nil
	}
	return result, true, nil
}

func lmStudioAPIBase(baseURL string) (string, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("URL inválida do provedor")
	}
	u.Path = strings.TrimSuffix(u.Path, "/v1")
	u.Path = strings.TrimRight(u.Path, "/") + "/api/v1"
	u.RawPath = ""
	return strings.TrimRight(u.String(), "/"), nil
}

// logLMStudioInstances registra o estado do servidor para o modelo
// configurado: uma linha por instância carregada, com o id da instância, a
// janela, o `parallel` e a quantização, e um aviso quando há mais de uma.
//
// É observabilidade, não conserto: duas instâncias do mesmo modelo atendem
// requisição endereçada ao mesmo `model` com janela e `parallel` diferentes, e
// medida ao vivo que não registra isso não é atribuível — foi o que esvaziou o
// poder de discriminação do ticket 17. Uma linha só com a `[0]` teria escondido
// a carga duplicada. O conserto (não empilhar instância, fixar `parallel`) é do
// esforço da conexão com o LLM
// (decisões 7 e 8: .agent/roteamento/issues/24-remedio-para-a-classificacao-vazia.md).
func logLMStudioInstances(model string, candidate lmStudioModel) {
	if len(candidate.LoadedInstances) == 0 {
		log.Info("LM Studio sem instância carregada do modelo", "model", model)
		return
	}
	for _, instance := range candidate.LoadedInstances {
		log.Info("instância carregada no LM Studio",
			"model", model,
			"instance", instance.ID,
			"context_length", instance.Config.ContextLength,
			"parallel", instance.Config.Parallel,
			"quantization", candidate.Quantization.Name,
		)
	}
	if len(candidate.LoadedInstances) > 1 {
		log.Warn("mais de uma instância carregada do mesmo modelo — a requisição cai em qualquer uma delas, com janela e parallel diferentes; medida ao vivo neste estado não é atribuível",
			"model", model, "instances", len(candidate.LoadedInstances))
	}
}
