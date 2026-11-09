package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const lmStudioProbeTimeout = 3 * time.Second

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

	ctx, cancel := context.WithTimeout(context.Background(), lmStudioProbeTimeout)
	defer cancel()

	models, supported, err := fetchLMStudioModels(ctx, root, apiKey)
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, root+"/models/load", strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("criando carga do LM Studio: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
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
