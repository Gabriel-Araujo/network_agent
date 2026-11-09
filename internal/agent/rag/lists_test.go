package rag

import "testing"

// TestFiltersFromUsesListNames pins the map -> Filters seam: every List name
// that is a frr_docs column must land in the right field, untranslated. If
// Lists() gains a new dimension that is a column, this test demands the new
// field. The "subagent" list is a routing dimension (which Subagente answers),
// not a search filter — out of the seam by decision.
func TestFiltersFromUsesListNames(t *testing.T) {
	campo := map[string]func(Filters) string{
		"protocol":   func(f Filters) string { return f.Protocol },
		"daemon":     func(f Filters) string { return f.Daemon },
		"chunk_type": func(f Filters) string { return f.ChunkType },
	}
	naoFiltro := map[string]bool{SubagentListName: true}
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

	// "subagent" must not leak into the filters: FiltersFrom ignores it.
	if f := FiltersFrom(map[string]string{SubagentListName: "builder"}); f != (Filters{}) {
		t.Errorf(`FiltersFrom({%q:"builder"}) = %+v, want Filters{}`, SubagentListName, f)
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

// TestSubagentListIsTheRoutingList pins the re-ask's list: SubagentList() is
// the routing list of Lists(), the complement of FilterLists() — the turn's
// targeted re-ask classifies with it and nothing else, so a Lists() reorder
// must not hand it a filter list.
func TestSubagentListIsTheRoutingList(t *testing.T) {
	l := SubagentList()
	if l.Name != SubagentListName {
		t.Fatalf("SubagentList() = %q, want %q", l.Name, SubagentListName)
	}
	if len(l.Values) == 0 {
		t.Fatal("SubagentList() came back with no values: the re-ask would be rejected by the classifier")
	}
	for _, f := range FilterLists() {
		if f.Name == l.Name {
			t.Errorf("SubagentList() is also in FilterLists(): %q", l.Name)
		}
	}
}

// TestFilterListsAreTheFilterColumns pins the boundary between the two calls:
// FilterLists() is exactly the Lists with a Filters field — the only ones that
// ride the RAG call and the skip rule comparison. The routing list stays out
// on purpose: switching Subagente does not re-trigger the retrieval.
func TestFilterListsAreTheFilterColumns(t *testing.T) {
	esperados := []string{"protocol", "daemon", "chunk_type"}
	filtraveis := FilterLists()
	if len(filtraveis) != len(esperados) {
		t.Fatalf("FilterLists() = %d listas, want %d", len(filtraveis), len(esperados))
	}
	for i, l := range filtraveis {
		if l.Name != esperados[i] {
			t.Errorf("FilterLists()[%d] = %q, want %q", i, l.Name, esperados[i])
		}
		if l.Name == SubagentListName {
			t.Errorf("FilterLists() leva a Lista de roteamento %q", SubagentListName)
		}
	}
}
