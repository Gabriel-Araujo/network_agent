package llm

import (
	"strings"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/skills/classifier"
)

// TestLists proves the single source of the generic call's Lists: names =
// frr_docs columns (except "subagent", which is not a filter), the closed set
// decided in .agent/classifier/issues/03-como-o-modo-rag-alimenta-a-busca.md.
func TestLists(t *testing.T) {
	listas := rag.Lists()
	if len(listas) != 4 {
		t.Fatalf("len(listas) = %d, want 4", len(listas))
	}

	var protocol, daemon, chunkType, subagent *classifier.List
	for i, l := range listas {
		switch l.Name {
		case "protocol":
			protocol = &listas[i]
		case "daemon":
			daemon = &listas[i]
		case "chunk_type":
			chunkType = &listas[i]
		case rag.SubagentListName:
			subagent = &listas[i]
		default:
			t.Errorf("lista inesperada %q", l.Name)
		}
	}
	if protocol == nil || daemon == nil || chunkType == nil || subagent == nil {
		t.Fatalf("faltam listas: protocol=%v daemon=%v chunk_type=%v subagent=%v", protocol, daemon, chunkType, subagent)
	}

	// Valores decididos: nada mais, nada menos.
	esperados := map[*classifier.List][]string{
		protocol:  {"bgp", "ospf", "ospf6", "isis", "rip", "ripng", "pim", "ldp", "vrrp", "bfd", "static"},
		daemon:    {"bgpd", "ospfd", "ospf6d", "isisd", "ripd", "ripngd", "pimd", "pim6d", "ldpd", "vrrpd", "bfdd", "staticd", "zebra"},
		chunkType: {rag.ChunkTypeCommandReference, rag.ChunkTypeConcept},
		subagent:  {"builder", "modifier", "illustrator", "teacher"},
	}
	for l, want := range esperados {
		if len(l.Values) != len(want) {
			t.Fatalf("lista %q: %d valores, want %d", l.Name, len(l.Values), len(want))
		}
		vistos := make(map[string]bool, len(want))
		for i, v := range l.Values {
			if v.Name != want[i] {
				t.Errorf("lista %q valor[%d] = %q, want %q", l.Name, i, v.Name, want[i])
			}
			if strings.TrimSpace(v.Name) == "" {
				t.Errorf("lista %q tem valor sem nome", l.Name)
			}
			if vistos[v.Name] {
				t.Errorf("lista %q repete o valor %q", l.Name, v.Name)
			}
			vistos[v.Name] = true
		}
	}
}
