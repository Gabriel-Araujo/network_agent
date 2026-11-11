package agent

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// lmStudioLento é um LM Studio de mentira que leva `demora` para responder a
// carga: o modelo aparece na sondagem com uma janela menor que a pedida, então
// ensureLMStudioContext é obrigado a carregar.
func lmStudioLento(t *testing.T, demora time.Duration) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/models":
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"models":[{"key":"modelo","loaded_instances":[{"config":{"context_length":8192}}]}]}`)
		case "/api/v1/models/load":
			select {
			case <-time.After(demora):
				w.WriteHeader(http.StatusOK)
			case <-r.Context().Done():
			}
		}
	}))
	t.Cleanup(server.Close)
	return server.URL + "/v1"
}

// prazos encurta os dois deadlines pela duração do teste. É o seam que permite
// provar qual dos dois governa a carga sem dormir os segundos de verdade.
func prazos(t *testing.T, sondagem, carga time.Duration) {
	t.Helper()
	probeAntes, cargaAntes := lmStudioProbeTimeout, lmStudioLoadTimeout
	t.Cleanup(func() { lmStudioProbeTimeout, lmStudioLoadTimeout = probeAntes, cargaAntes })
	lmStudioProbeTimeout, lmStudioLoadTimeout = sondagem, carga
}

// A carga tem deadline próprio: uma carga mais lenta que o prazo da SONDAGEM
// precisa terminar bem. É a medida do defeito do ticket 13 — os dois HTTP
// dividiam um context.WithTimeout de 3s dimensionado para o GET, e por isso
// toda carga de modelo real (19s medidos no menor dos modelos locais) fazia o
// Connect inteiro falhar e o app subir sem agente nenhum.
func TestCargaDoLMStudioNaoHerdaOPrazoDaSondagem(t *testing.T) {
	prazos(t, 200*time.Millisecond, time.Minute)

	if err := ensureLMStudioContext(lmStudioLento(t, 500*time.Millisecond), "", "modelo", 65536); err != nil {
		t.Fatalf("a carga falhou com o prazo da sondagem no caminho: %v", err)
	}
}

// A outra direção: o prazo da carga, quando estourado de verdade, diz que foi
// a CARGA que não terminou. Sem a causa no ctx as duas situações — o servidor
// que não responde e a carga que não acaba — chegavam ao usuário como o mesmo
// "context deadline exceeded".
func TestCargaDoLMStudioEstourandoNomeiaACausa(t *testing.T) {
	prazos(t, 200*time.Millisecond, 100*time.Millisecond)

	err := ensureLMStudioContext(lmStudioLento(t, 2*time.Second), "", "modelo", 65536)
	if err == nil {
		t.Fatal("a carga estourou o prazo e ensureLMStudioContext não devolveu erro")
	}
	if !errors.Is(err, errLMStudioLoadDeadline) {
		t.Errorf("erro %q não carrega a causa da carga", err)
	}
}
