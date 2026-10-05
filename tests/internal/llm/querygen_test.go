package llm

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag"
	ragretriever "github.com/Gabriel-Araujo/network_agent/internal/skills/rag-retriever"
)

// fakeGenerator permite testar o caminho primário sem rede e registra os
// filtros que recebeu.
type fakeGenerator struct {
	qs           []rag.QuerySuggestion
	err          error
	recebidos    rag.Filtros
	recebidoInpt string
}

func (f *fakeGenerator) GenerateQueries(ctx context.Context, input string, filtros rag.Filtros) ([]rag.QuerySuggestion, error) {
	f.recebidos = filtros
	f.recebidoInpt = input
	return f.qs, f.err
}

func TestBuildQueriesUsaGerador(t *testing.T) {
	fake := &fakeGenerator{qs: []rag.QuerySuggestion{
		{Query: "OSPF MTU mismatch"},
		{Query: "show ip ospf neighbor"},
	}}
	filtros := rag.Filtros{Protocol: "ospf", Daemon: "ospfd", ChunkType: "concept"}

	qs := ragretriever.BuildQueries(context.Background(), "meu ospf não converge", filtros, fake)
	if len(qs) != 2 || qs[0].Query != "OSPF MTU mismatch" {
		t.Fatalf("qs = %+v, want as queries do gerador", qs)
	}
	// O gerador recebe o input e o conjunto de filtros da busca.
	if fake.recebidoInpt != "meu ospf não converge" {
		t.Errorf("input repassado = %q", fake.recebidoInpt)
	}
	if fake.recebidos != filtros {
		t.Errorf("filtros repassados = %+v, want %+v", fake.recebidos, filtros)
	}
}

// Caso 3 do contrato: falhou a geração, a busca roda com uma única query do
// input cru — nunca a busca inteira se perde por causa da reescrita.
func TestBuildQueriesDegradaParaInputCru(t *testing.T) {
	fake := &fakeGenerator{err: errors.New("provedor fora do ar")}
	qs := ragretriever.BuildQueries(context.Background(), "meu ospf não converge", rag.Filtros{}, fake)
	if len(qs) != 1 || qs[0].Query != "meu ospf não converge" {
		t.Fatalf("qs = %+v, want uma query do input cru", qs)
	}
}

func TestBuildQueriesVazioDegrada(t *testing.T) {
	fake := &fakeGenerator{qs: nil}
	qs := ragretriever.BuildQueries(context.Background(), "entrada qualquer", rag.Filtros{}, fake)
	if len(qs) != 1 || qs[0].Query != "entrada qualquer" {
		t.Fatalf("qs = %+v, want fallback para o input cru", qs)
	}
}

func TestBuildQueriesSemGerador(t *testing.T) {
	qs := ragretriever.BuildQueries(context.Background(), "entrada qualquer", rag.Filtros{}, nil)
	if len(qs) != 1 || qs[0].Query != "entrada qualquer" {
		t.Fatalf("qs = %+v, want fallback para o input cru", qs)
	}
}

func TestParseGeneratedQueriesJSON(t *testing.T) {
	text := `["OSPF MTU mismatch", "show ip ospf neighbor"]`
	qs, err := ragretriever.ParseGeneratedQueries(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(qs) != 2 || qs[0].Query != "OSPF MTU mismatch" || qs[1].Query != "show ip ospf neighbor" {
		t.Fatalf("qs = %+v", qs)
	}
}

func TestParseGeneratedQueriesWithCodeFence(t *testing.T) {
	text := "Aqui está o JSON:\n```json\n[\"OSPF neighbor stuck\"]\n```\nFim."
	qs, err := ragretriever.ParseGeneratedQueries(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(qs) != 1 || qs[0].Query != "OSPF neighbor stuck" {
		t.Fatalf("qs = %+v", qs)
	}
}

func TestParseGeneratedQueriesSkipsEmptyQuery(t *testing.T) {
	text := `["", "  ", "OSPF cost"]`
	qs, err := ragretriever.ParseGeneratedQueries(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(qs) != 1 || !strings.Contains(qs[0].Query, "cost") {
		t.Fatalf("qs = %+v, want only the non-empty query", qs)
	}
}

func TestParseGeneratedQueriesInvalido(t *testing.T) {
	// Conteúdo sem array JSON de strings → erro (a degradação é do BuildQueries).
	if _, err := ragretriever.ParseGeneratedQueries("sem json aqui"); err == nil {
		t.Fatal("esperava erro para saída sem JSON")
	}
	// Array de objetos (formato antigo) também é rejeitado: o contrato são strings.
	if _, err := ragretriever.ParseGeneratedQueries(`[{"query":"x"}]`); err == nil {
		t.Fatal("esperava erro para array de objetos")
	}
}
