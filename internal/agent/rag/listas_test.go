package rag

import "testing"

// TestFiltrosFromUsaOsNomesDasListas fixa a costura mapa → Filtros: cada nome
// de Lista que é coluna da frr_docs tem de cair no campo certo, sem tradução.
// Se Listas() ganhar uma dimensão nova que seja coluna, este teste exige o
// campo novo. A lista "agent" é dimensão de roteamento (qual agente responde),
// não filtro da busca — fora da costura por decisão.
func TestFiltrosFromUsaOsNomesDasListas(t *testing.T) {
	campo := map[string]func(Filtros) string{
		"protocol":   func(f Filtros) string { return f.Protocol },
		"daemon":     func(f Filtros) string { return f.Daemon },
		"chunk_type": func(f Filtros) string { return f.ChunkType },
	}
	naoFiltro := map[string]bool{"agent": true}
	for _, l := range Listas() {
		if naoFiltro[l.Nome] {
			continue
		}
		pega, ok := campo[l.Nome]
		if !ok {
			t.Fatalf("Lista %q sem campo correspondente em Filtros (coluna nova da frr_docs?)", l.Nome)
		}
		f := FiltrosFrom(map[string]string{l.Nome: "x"})
		if got := pega(f); got != "x" {
			t.Errorf("FiltrosFrom(%q) preencheu %q, want %q", l.Nome, got, "x")
		}
	}

	// "agent" não vaza para os filtros: FiltrosFrom o ignora.
	if f := FiltrosFrom(map[string]string{"agent": "builder"}); f != (Filtros{}) {
		t.Errorf(`FiltrosFrom({"agent":"builder"}) = %+v, want Filtros{}`, f)
	}
}

// TestFiltrosFromVazioEhSemFiltro: "" e mapa ausente viram "sem filtro".
func TestFiltrosFromVazioEhSemFiltro(t *testing.T) {
	f := FiltrosFrom(map[string]string{"protocol": "", "daemon": "", "chunk_type": ""})
	if f != (Filtros{}) {
		t.Errorf("FiltrosFrom(vazio) = %+v, want Filtros{}", f)
	}
	if f := FiltrosFrom(nil); f != (Filtros{}) {
		t.Errorf("FiltrosFrom(nil) = %+v, want Filtros{}", f)
	}
}
