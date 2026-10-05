package main

import "testing"

// TestDeveBuscar fixa a skip rule do fluxo E2E (05/08): busca quando a
// classificação RAG difere do turno anterior, pula quando é igual (igualdade
// exata do mapa, vazio incluído), e sempre busca sem anterior (primeiro turno
// ou comparação zerada) ou com classificação falha.
func TestDeveBuscar(t *testing.T) {
	vazia := map[string]string{"protocol": "", "daemon": "", "chunk_type": ""}
	cheia := map[string]string{"protocol": "bgp", "daemon": "bgpd", "chunk_type": "concept"}

	casos := []struct {
		nome            string
		atual, anterior map[string]string
		quer            bool
	}{
		{"primeiro turno sempre busca", cheia, nil, true},
		{"classificação igual pula", cheia, cheia, false},
		{"igualdade exata inclui o vazio", vazia, map[string]string{"protocol": "", "daemon": "", "chunk_type": ""}, false},
		{"classificação diferente busca", cheia, vazia, true},
		{"falha do classificador busca", nil, cheia, true},
		{"falha sem anterior busca", nil, nil, true},
		{"comparação zerada busca", cheia, nil, true},
	}
	for _, c := range casos {
		if got := deveBuscar(c.atual, c.anterior); got != c.quer {
			t.Errorf("%s: deveBuscar = %v, want %v", c.nome, got, c.quer)
		}
	}
}

// TestJSONDoContexto: o bloco <recovered_context> recebe o JSON de
// []rag.Result exatamente como o espelho grava.
func TestJSONDoContexto(t *testing.T) {
	if got := jsonDoContexto(nil); got != "[]" {
		t.Errorf("jsonDoContexto(nil) = %q, want %q", got, "[]")
	}
}
