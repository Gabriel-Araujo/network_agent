package llm

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag/querygen"
)

// fakeGenerator permite testar o fallback sem rede.
type fakeGenerator struct {
	qs  []rag.QuerySuggestion
	err error
}

func (f *fakeGenerator) GenerateQueries(ctx context.Context, briefing []byte) ([]rag.QuerySuggestion, error) {
	return f.qs, f.err
}

// panickingGenerator prova que o caminho determinístico NÃO chama o LLM.
type panickingGenerator struct{}

func (panickingGenerator) GenerateQueries(ctx context.Context, briefing []byte) ([]rag.QuerySuggestion, error) {
	panic("LLM should not be called when briefing has the query table")
}

func TestBuildQueriesUsesParseNotLLM(t *testing.T) {
	qs, err := querygen.BuildQueries(context.Background(), []byte(sampleBriefing), panickingGenerator{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(qs) != 3 {
		t.Fatalf("len = %d, want 3", len(qs))
	}
	if qs[0].Query != "OSPF neighbor stuck Exstart Exchange state" {
		t.Errorf("query[0] = %q", qs[0].Query)
	}
}

func TestBuildQueriesFallsBackToLLM(t *testing.T) {
	briefing := []byte("## 4. Problem/goal summary\nNo query table here.\n")
	fake := &fakeGenerator{qs: []rag.QuerySuggestion{
		{Query: "OSPF MTU mismatch", Protocol: "ospf", Daemon: "ospfd", ChunkType: "concept"},
	}}
	qs, err := querygen.BuildQueries(context.Background(), briefing, fake)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(qs) != 1 || qs[0].Query != "OSPF MTU mismatch" {
		t.Fatalf("qs = %+v, want the fake's suggestion", qs)
	}
}

func TestBuildQueriesNoTableNoGenerator(t *testing.T) {
	_, err := querygen.BuildQueries(context.Background(), []byte("## 4. Nothing"), nil)
	if !errors.Is(err, rag.ErrNoQueriesTable) {
		t.Fatalf("err = %v, want ErrNoQueriesTable", err)
	}
}

func TestParseGeneratedQueriesJSON(t *testing.T) {
	// Sem code fence.
	text := `[{"query":"OSPF MTU mismatch","protocol":"ospf","daemon":"ospfd","chunk_type":"concept"},{"query":"show ip ospf neighbor","protocol":"ospf","daemon":"ospfd","chunk_type":"command_reference"}]`
	qs, err := querygen.ParseGeneratedQueries(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(qs) != 2 {
		t.Fatalf("len = %d, want 2", len(qs))
	}
	if qs[0].ChunkType != "concept" {

	}
}

func TestParseGeneratedQueriesWithCodeFence(t *testing.T) {
	text := "Aqui está o JSON:\n```json\n[{\"query\":\"OSPF neighbor stuck\",\"protocol\":\"ospf\",\"daemon\":\"ospfd\",\"chunk_type\":\"concept\"}]\n```\nFim."
	qs, err := querygen.ParseGeneratedQueries(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(qs) != 1 || qs[0].Query != "OSPF neighbor stuck" {
		t.Fatalf("qs = %+v", qs)
	}
}

func TestParseGeneratedQueriesRejectsInvalidChunkType(t *testing.T) {
	text := `[{"query":"x","protocol":"ospf","daemon":"ospfd","chunk_type":"bogus"}]`
	qs, err := querygen.ParseGeneratedQueries(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qs[0].ChunkType != "" {
		t.Errorf("chunk_type = %q, want empty (invalid normalized away)", qs[0].ChunkType)
	}
}

func TestParseGeneratedQueriesSkipsEmptyQuery(t *testing.T) {
	text := `[{"query":"","protocol":"ospf","daemon":"ospfd","chunk_type":"concept"},{"query":"OSPF cost","protocol":"ospf","daemon":"ospfd","chunk_type":"concept"}]`
	qs, err := querygen.ParseGeneratedQueries(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(qs) != 1 || !strings.Contains(qs[0].Query, "cost") {
		t.Fatalf("qs = %+v, want only the non-empty query", qs)
	}
}
