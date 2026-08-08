package llm

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/llm/rag/retriever"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func TestVectorLiteral(t *testing.T) {
	cases := []struct {
		in   []float64
		want string
	}{
		{[]float64{1.5, -2.25}, "[1.50000000,-2.25000000]"},
		{[]float64{}, "[]"},
		{[]float64{0}, "[0.00000000]"},
	}
	for _, c := range cases {
		if got := retriever.VectorLiteral(c.in); got != c.want {
			t.Errorf("VectorLiteral(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFuseRRFDedupe(t *testing.T) {
	// Mesmo chunk id aparece em ambas as fontes -> deve aparecer 1x,
	// com score somado (RRF) e ordenado decrescente.
	vec := map[string]int{"a": 0, "b": 1, "c": 2}
	fts := map[string]int{"b": 0, "d": 1, "a": 9}

	got := retriever.FuseRRF(vec, fts, 60, 5)
	if len(got) != 4 {
		t.Fatalf("len = %d, want 4: %v", len(got), got)
	}
	// 'a' e 'b' presentes nas duas -> maior que 'c' e 'd'.
	if got[0].ID != "a" && got[0].ID != "b" {
		t.Errorf("top = %q, want a or b", got[0].ID)
	}
	if got[0].ID == got[1].ID {
		t.Errorf("duplicated id in result: %v", got)
	}
	// 'a' deve estar acima de 'c' e 'd'.
	scoreA := gotScore(got, "a")
	scoreC := gotScore(got, "c")
	if scoreA <= scoreC {
		t.Errorf("score(a)=%v should be > score(c)=%v", scoreA, scoreC)
	}
}

func TestFuseRRFEmpty(t *testing.T) {
	if got := retriever.FuseRRF(nil, nil, 60, 5); len(got) != 0 {
		t.Fatalf("expected empty result, got %v", got)
	}
}

func TestFuseRRFRespectsLimit(t *testing.T) {
	vec := map[string]int{"a": 0, "b": 1, "c": 2, "d": 3}
	got := retriever.FuseRRF(vec, nil, 60, 2)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (limit)", len(got))
	}
}

// --- helpers de teste ---

func gotScore(rs []rag.ScoredChunk, id string) float64 {
	for _, r := range rs {
		if r.ID == id {
			return r.Score
		}
	}
	return -1
}

// TestRetrieveIntegration roda contra o postgres real, usando o modelo de
// embedding configurado em EMBEDDING_MODEL. Gated: SKIP sem DATABASE_URL.
func TestRetrieveIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL não definido; pulando teste de integração")
	}

	client := testEmbedderClient()

	cfg := rag.Config{
		DSN:      dsn,
		Embedder: client,
		Limit:    3,
	}

	queries := []rag.QuerySuggestion{
		{Query: "BGP neighbor session Active state", Protocol: "bgp", Daemon: "bgpd"},
		{Query: "OSPF stuck Exstart MTU mismatch", Protocol: "ospf", Daemon: "ospfd"},
	}

	results, err := retriever.Retrieve(context.Background(), cfg, queries)
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	for _, r := range results {
		if r.Query == "" {
			t.Errorf("result com query vazia: %+v", r)
		}
	}
}

// testEmbedderClient monta o cliente do LoadEmbbedAgent: embarca o LM
// Studio (localhost:1234) com o modelo de embedding do ingester (4096),
// idêntico ao usado em produção pelo cmd/cli.
func testEmbedderClient() openai.Client {
	return openai.NewClient(
		option.WithBaseURL("http://localhost:1234/v1"),
		option.WithAPIKey("none"),
	)
}

// TestSaveResults verifica o shape do JSON gravado e a convenção de nome.
func TestSaveResults(t *testing.T) {
	tmp := t.TempDir()
	t.Chdir(tmp)

	briefing := filepath.Join(tmp, "meu-ospf-abc123.md")
	if err := os.WriteFile(briefing, []byte("# x"), 0o644); err != nil {
		t.Fatal(err)
	}

	results := []rag.Result{
		{Query: "OSPF MTU mismatch", Response: "contexto A\n\nSource: https://docs.example (OSPF)"},
		{Query: "BGP session down", Response: "contexto B"},
	}
	path, err := retriever.SaveResults(briefing, results)
	if err != nil {
		t.Fatalf("saveResults: %v", err)
	}
	if filepath.Base(path) != "meu-ospf-abc123.json" {
		t.Errorf("nome = %q, want meu-ospf-abc123.json", filepath.Base(path))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got []rag.Result
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}
	if len(got) != 2 || got[0].Query != "OSPF MTU mismatch" || got[1].Response != "contexto B" {
		t.Fatalf("got = %+v", got)
	}
}
