package llm

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/skills/classifier"
	"github.com/jackc/pgx/v5"
)

// TestListsMatchCorpus pins every filter List to the labels its frr_docs column
// carries. A filter works only by exact WHERE match, so a value with no rows
// hides the whole corpus and a label missing from the List is reachable only
// through an empty value — re-ingesting a new doc fails here until its label is
// listed (.agent/classifier/issues/10-listas-de-filtro-alinhadas-ao-corpus.md).
// Gated on DATABASE_URL, like the E2E.
func TestListsMatchCorpus(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL não definido; pulando a conferência das Listas contra o corpus")
	}
	conn, err := pgx.Connect(t.Context(), dsn)
	if err != nil {
		t.Fatalf("conectando ao banco: %v", err)
	}
	defer conn.Close(t.Context())

	for _, list := range rag.FilterLists() {
		// The List name is the column name by contract, and it comes from
		// the package table, never from input.
		rows, err := conn.Query(t.Context(), "SELECT DISTINCT "+list.Name+" FROM frr_docs WHERE "+list.Name+" IS NOT NULL")
		if err != nil {
			t.Fatalf("lendo os rótulos de %q: %v", list.Name, err)
		}
		labels, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			t.Fatalf("lendo os rótulos de %q: %v", list.Name, err)
		}
		values := make([]string, 0, len(list.Values))
		for _, v := range list.Values {
			values = append(values, v.Name)
		}
		for _, label := range labels {
			if !slices.Contains(values, label) {
				t.Errorf("lista %q: o banco tem %q e a Lista não — esses chunks só aparecem com o valor vazio", list.Name, label)
			}
		}
		for _, v := range values {
			if !slices.Contains(labels, v) {
				t.Errorf("lista %q: o valor %q não tem nenhum chunk — escolhê-lo esconde o banco inteiro", list.Name, v)
			}
		}
	}
}

// TestLists proves the single source of the generic call's Lists: names =
// frr_docs columns (except "subagent", which is not a filter), the closed set
// decided in .agent/classifier/issues/03-como-o-modo-rag-alimenta-a-busca.md
// with the filter values realigned to the corpus in
// .agent/classifier/issues/10-listas-de-filtro-alinhadas-ao-corpus.md.
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
		protocol: {"bgp", "evpn", "bmp", "bgp-linkstate", "ospf", "ospf-fundamentals", "ospf6", "isis", "openfabric", "ldp", "bfd", "babel", "eigrp", "nhrp",
			"basic", "filter", "ipv6", "kernel", "nexthop-groups", "affinitymap", "extlog", "frr-reload", "grpc", "mgmt"},
		daemon:    {"bgpd", "ospfd", "ospf6d", "isisd", "fabricd", "ldpd", "bfdd", "babeld", "eigrpd", "nhrpd", "mgmtd", "zebra"},
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
