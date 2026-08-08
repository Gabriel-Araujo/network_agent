package llm

import (
	"errors"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag/querygen"
)

const sampleBriefing = `{
  "title": "Intent Analysis - FRR",
  "timestamp": "2026-07-14T15:20:00-03:00",
  "originalQuery": "meu ospf não converge entre o core (R1) e o edge (R2)",
  "classification": {
    "intentType": "troubleshooting",
    "protocols": ["ospf (primary)", "bgp (context/possible interaction)"],
    "daemons": ["ospfd", "bgpd"]
  },
  "ragQueries": {
    "10": {"query": "OSPF MTU mismatch neighbor adjacency", "protocol": "ospf", "daemon": "ospfd", "suggestedChunkType": "concept"},
    "1":  {"query": "OSPF neighbor stuck Exstart Exchange state", "protocol": "ospf", "daemon": "ospfd", "suggestedChunkType": "concept"},
    "2":  {"query": "OSPF interface authentication area configuration", "protocol": "ospf", "daemon": "ospfd", "suggestedChunkType": "command_reference"},
    "3":  {"query": "BGP OSPF redistribution interaction", "protocol": "BGP", "daemon": "BGPD", "suggestedChunkType": "concept"}
  }
}
`

func TestParseQueriesFromBriefing(t *testing.T) {
	got, err := querygen.ParseQueriesFromBriefing([]byte(sampleBriefing))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("len = %d, want 4: %+v", len(got), got)
	}

	want := []rag.QuerySuggestion{
		{Query: "OSPF neighbor stuck Exstart Exchange state", Protocol: "ospf", Daemon: "ospfd", ChunkType: "concept"},
		{Query: "OSPF interface authentication area configuration", Protocol: "ospf", Daemon: "ospfd", ChunkType: "command_reference"},
		{Query: "BGP OSPF redistribution interaction", Protocol: "bgp", Daemon: "bgpd", ChunkType: "concept"},
		{Query: "OSPF MTU mismatch neighbor adjacency", Protocol: "ospf", Daemon: "ospfd", ChunkType: "concept"},
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestParseQueriesFromBriefingNumericOrder(t *testing.T) {
	// Ordem deve seguir as chaves numéricas (1,2,...), não a ordem do JSON.
	qs, err := querygen.ParseQueriesFromBriefing([]byte(sampleBriefing))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qs[0].Query != "OSPF neighbor stuck Exstart Exchange state" {
		t.Errorf("qs[0] = %q, want first by numeric key (1)", qs[0].Query)
	}
	if qs[len(qs)-1].Query != "OSPF MTU mismatch neighbor adjacency" {
		t.Errorf("last = %q, want key 10 after 3", qs[len(qs)-1].Query)
	}
}

func TestParseQueriesFromBriefingInvalidChunkType(t *testing.T) {
	// chunk_type inválido deve virar "" (sem filtro), não quebrar o parse.
	briefing := `{
  "ragQueries": {
    "1": {"query": "OSPF MTU mismatch", "protocol": "ospf", "daemon": "ospfd", "suggestedChunkType": "bogus"},
    "2": {"query": "OSPF cost", "protocol": "ospf", "daemon": "ospfd", "suggestedChunkType": "concept"}
  }
}`
	got, err := querygen.ParseQueriesFromBriefing([]byte(briefing))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0].ChunkType != "" {
		t.Errorf("chunk_type[0] = %q, want %q", got[0].ChunkType, "")
	}
	if got[1].ChunkType != "concept" {
		t.Errorf("chunk_type[1] = %q, want %q", got[1].ChunkType, "concept")
	}
}

func TestParseQueriesFromBriefingMissingSection(t *testing.T) {
	briefing := `{
  "classification": {"intentType": "troubleshooting", "protocols": ["ospf"], "daemons": ["ospfd"]},
  "problemAndGoalSummary": "Nothing here about queries."
}`
	_, err := querygen.ParseQueriesFromBriefing([]byte(briefing))
	if !errors.Is(err, rag.ErrNoQueries) {
		t.Fatalf("err = %v, want ErrNoQueries", err)
	}
}

func TestParseQueriesFromBriefingNotJSON(t *testing.T) {
	// Conteúdo que não é JSON -> ErrNoQueries (cai no fallback LLM).
	_, err := querygen.ParseQueriesFromBriefing([]byte("## 4. Problem/goal summary\nNot JSON."))
	if !errors.Is(err, rag.ErrNoQueries) {
		t.Fatalf("err = %v, want ErrNoQueries", err)
	}
}
