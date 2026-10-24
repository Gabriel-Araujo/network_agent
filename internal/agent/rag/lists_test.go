package rag

import "testing"

// TestFiltersFromUsesListNames fixa a costura mapa → Filters: cada nome
// de Lista que é coluna da frr_docs tem de cair no campo certo, sem tradução.
// Se Lists() ganhar uma dimensão nova que seja coluna, este teste exige o
// campo novo. A lista "agent" é dimensão de roteamento (qual agente responde),
// não filtro da busca — fora da costura por decisão.
func TestFiltersFromUsesListNames(t *testing.T) {
	campo := map[string]func(Filters) string{
		"protocol":   func(f Filters) string { return f.Protocol },
		"daemon":     func(f Filters) string { return f.Daemon },
		"chunk_type": func(f Filters) string { return f.ChunkType },
	}
	naoFiltro := map[string]bool{"agent": true}
	for _, l := range Lists() {
		if naoFiltro[l.Name] {
			continue
		}
		pega, ok := campo[l.Name]
		if !ok {
			t.Fatalf("Lista %q sem campo correspondente em Filtros (coluna nova da frr_docs?)", l.Name)
		}
		f := FiltersFrom(map[string]string{l.Name: "x"})
		if got := pega(f); got != "x" {
			t.Errorf("FiltrosFrom(%q) preencheu %q, want %q", l.Name, got, "x")
		}
	}

	// "agent" não vaza para os filtros: FiltrosFrom o ignora.
	if f := FiltersFrom(map[string]string{"agent": "builder"}); f != (Filters{}) {
		t.Errorf(`FiltrosFrom({"agent":"builder"}) = %+v, want Filtros{}`, f)
	}
}

// TestFiltersFromEmptyMeansNoFilter: "" e mapa ausente viram "sem filtro".
func TestFiltersFromEmptyMeansNoFilter(t *testing.T) {
	f := FiltersFrom(map[string]string{"protocol": "", "daemon": "", "chunk_type": ""})
	if f != (Filters{}) {
		t.Errorf("FiltrosFrom(vazio) = %+v, want Filtros{}", f)
	}
	if f := FiltersFrom(nil); f != (Filters{}) {
		t.Errorf("FiltrosFrom(nil) = %+v, want Filtros{}", f)
	}
}
