package rag

import "testing"

// TestFiltrosFromUsaOsNomesDasListas fixa a costura mapa → Filtros: cada nome
// de Lista (as colunas da frr_docs) tem de cair no campo certo, sem tradução.
// Se Listas() ganhar uma dimensão nova, este teste exige o campo novo.
func TestFiltrosFromUsaOsNomesDasListas(t *testing.T) {
	campo := map[string]func(Filtros) string{
		"protocol":   func(f Filtros) string { return f.Protocol },
		"daemon":     func(f Filtros) string { return f.Daemon },
		"chunk_type": func(f Filtros) string { return f.ChunkType },
	}
	for _, l := range Listas() {
		pega, ok := campo[l.Nome]
		if !ok {
			t.Fatalf("Lista %q sem campo correspondente em Filtros", l.Nome)
		}
		f := FiltrosFrom(map[string]string{l.Nome: "x"})
		if got := pega(f); got != "x" {
			t.Errorf("FiltrosFrom(%q) preencheu %q, want %q", l.Nome, got, "x")
		}
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
