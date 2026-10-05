package llm

import (
	"strings"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/skills/classifier"
)

// TestListas prova a fonte única das Listas do chamador RAG: nomes = colunas
// da frr_docs (exceto "agent", que não é filtro), conjunto fechado decidido em
// .agent/classifier/issues/03-como-o-modo-rag-alimenta-a-busca.md.
func TestListas(t *testing.T) {
	listas := rag.Listas()
	if len(listas) != 4 {
		t.Fatalf("len(listas) = %d, want 4", len(listas))
	}

	var protocol, daemon, chunkType, agent *classifier.Lista
	for i, l := range listas {
		switch l.Nome {
		case "protocol":
			protocol = &listas[i]
		case "daemon":
			daemon = &listas[i]
		case "chunk_type":
			chunkType = &listas[i]
		case "agent":
			agent = &listas[i]
		default:
			t.Errorf("lista inesperada %q", l.Nome)
		}
	}
	if protocol == nil || daemon == nil || chunkType == nil || agent == nil {
		t.Fatalf("faltam listas: protocol=%v daemon=%v chunk_type=%v agent=%v", protocol, daemon, chunkType, agent)
	}

	// Valores decididos: nada mais, nada menos.
	esperados := map[*classifier.Lista][]string{
		protocol:  {"bgp", "ospf", "ospf6", "isis", "rip", "ripng", "pim", "ldp", "vrrp", "bfd", "static"},
		daemon:    {"bgpd", "ospfd", "ospf6d", "isisd", "ripd", "ripngd", "pimd", "pim6d", "ldpd", "vrrpd", "bfdd", "staticd", "zebra"},
		chunkType: {rag.ChunkTypeCommandReference, rag.ChunkTypeConcept},
		agent:     {"builder", "modifier", "teacher"},
	}
	for l, want := range esperados {
		if len(l.Valores) != len(want) {
			t.Fatalf("lista %q: %d valores, want %d", l.Nome, len(l.Valores), len(want))
		}
		vistos := make(map[string]bool, len(want))
		for i, v := range l.Valores {
			if v.Nome != want[i] {
				t.Errorf("lista %q valor[%d] = %q, want %q", l.Nome, i, v.Nome, want[i])
			}
			if strings.TrimSpace(v.Nome) == "" {
				t.Errorf("lista %q tem valor sem nome", l.Nome)
			}
			if vistos[v.Nome] {
				t.Errorf("lista %q repete o valor %q", l.Nome, v.Nome)
			}
			vistos[v.Nome] = true
		}
	}
}
