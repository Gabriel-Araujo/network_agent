package rag

import (
	"errors"
	"regexp"
	"strings"
)

// ErrNoQueriesTable indica que o briefing não contém a seção
// "Rewritten queries for RAG", obrigando o fallback por LLM.
var ErrNoQueriesTable = errors.New("briefing sem seção 'Rewritten queries for RAG'")

// QuerySuggestion é uma query reescrita para busca, com filtros
// opcionais de metadados para a tabela frr_docs. ChunkType vazio
// significa "sem filtro".
type QuerySuggestion struct {
	Query     string
	Protocol  string
	Daemon    string
	ChunkType string
}

const (
	chunkTypeCommandReference = "command_reference"
	chunkTypeConcept          = "concept"

	queriesHeader = "Rewritten queries for RAG"
)

var (
	queriesHeaderRe = regexp.MustCompile(`(?m)^\s*#{1,6}\s*(?:\d+\.?\s*)?Rewritten queries for RAG\s*$`)
	tableRowRe      = regexp.MustCompile(`^\s*\|`)
)

// ParseQueriesFromBriefing extrai a tabela "Rewritten queries for RAG" do
// briefing em Markdown e devolve uma sugestão de query por linha, na ordem
// em que aparecem. Retorna ErrNoQueriesTable quando a seção não existe.
//
// Formato esperado, com header fixo:
//
//	| # | Rewritten query | protocol | daemon | suggested chunk_type |
//	|---|---|---|---|---|
//	| 1 | "OSPF MTU mismatch" | ospf | ospfd | concept |
func ParseQueriesFromBriefing(content []byte) ([]QuerySuggestion, error) {
	loc := queriesHeaderRe.FindIndex(content)
	if loc == nil {
		return nil, ErrNoQueriesTable
	}

	rest := string(content[loc[1]:])
	var out []QuerySuggestion
	// Cada linha da tabela vem logo após o header; paramos no primeiro
	// header de seção seguinte ou em linha não-tabela após termos
	// coletado pelo menos uma linha de dados.
	for _, line := range splitLines(rest) {
		trimmed := strings.TrimSpace(line)
		if tableRowRe.MatchString(line) {
			cols := splitTableRow(trimmed)
			// pula a linha de separação (|---|) e o header
			if isTableSeparator(cols) || isTableHeader(cols) {
				continue
			}
			s, ok := parseQueryRow(cols)
			if ok {
				out = append(out, s)
			}
			continue
		}
		// Saiu da tabela e já temos dados -> termina.
		if len(out) > 0 && trimmed != "" {
			break
		}
	}
	return out, nil
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(s, "\n")
}

// splitTableRow divide uma linha de tabela markdown em células, removendo
// espaços em branco ao redor e o texto "> quotes".
func splitTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	if line == "" {
		return nil
	}
	parts := strings.Split(line, "|")
	cols := make([]string, 0, len(parts))
	for _, p := range parts {
		cols = append(cols, strings.TrimSpace(p))
	}
	return cols
}

// isTableSeparator detecta a linha "|---|---|...".
func isTableSeparator(cols []string) bool {
	for _, c := range cols {
		if c == "" {
			continue
		}
		if !strings.HasPrefix(c, "-") && !strings.HasPrefix(c, ":") {
			return false
		}
	}
	return true
}

// isTableHeader detecta o cabeçalho fixo da tabela.
func isTableHeader(cols []string) bool {
	return len(cols) >= 2 && strings.TrimSpace(cols[1]) == "Rewritten query"
}

// parseQueryRow interpreta uma linha de dados: [#, query, protocol, daemon, chunk_type].
func parseQueryRow(cols []string) (QuerySuggestion, bool) {
	if len(cols) < 5 {
		return QuerySuggestion{}, false
	}
	query := unquote(strings.TrimSpace(cols[1]))
	if query == "" {
		return QuerySuggestion{}, false
	}
	ct := strings.ToLower(strings.TrimSpace(cols[4]))
	switch ct {
	case chunkTypeCommandReference, chunkTypeConcept:
		// ok
	default:
		ct = ""
	}
	return QuerySuggestion{
		Query:     query,
		Protocol:  strings.ToLower(strings.TrimSpace(cols[2])),
		Daemon:    strings.ToLower(strings.TrimSpace(cols[3])),
		ChunkType: ct,
	}, true
}

// unquote remove aspas duplas ou simples ao redor da query.
func unquote(s string) string {
	for _, q := range []byte{'"', '\''} {
		if len(s) >= 2 && s[0] == q && s[len(s)-1] == q {
			s = s[1 : len(s)-1]
			break
		}
	}
	return s
}
