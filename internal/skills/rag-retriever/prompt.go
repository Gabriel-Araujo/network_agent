package ragretriever

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag"
)

//go:embed prompt.md
var skillPrompt string

// systemPrompt define a identidade do agente de reescrita de queries,
// enviado como instrução de sistema a cada chamada.
const systemPrompt = `
# Identity

You are a senior network engineer specialized in **FRRouting (FRR)**. Your
job is to rewrite a user's networking question into the exact search
queries to run against an FRR documentation knowledge base, following the
rules in the skill prompt.
`

// montarPrompt monta as instruções do gerador: identidade + regras (prompt.md),
// os filtros escolhidos para a busca inteira e o vocabulário das Listas —
// derivado da fonte única rag.Listas(), nunca copiado à mão.
func montarPrompt(filtros rag.Filtros) string {
	var b strings.Builder
	b.WriteString(systemPrompt + "\n" + skillPrompt)

	b.WriteString("\n# Search filters (apply to the whole search)\n")
	fmt.Fprintf(&b, "- protocol: %s\n", filtroOuQualquer(filtros.Protocol))
	fmt.Fprintf(&b, "- daemon: %s\n", filtroOuQualquer(filtros.Daemon))
	fmt.Fprintf(&b, "- chunk_type: %s\n", filtroOuQualquer(filtros.ChunkType))

	b.WriteString("\n# Valid values (closed vocabulary)\n")
	for _, l := range rag.Listas() {
		nomes := make([]string, 0, len(l.Valores))
		for _, v := range l.Valores {
			nomes = append(nomes, v.Nome)
		}
		fmt.Fprintf(&b, "- %s: %s\n", l.Nome, strings.Join(nomes, ", "))
	}
	return b.String()
}

// filtroOuQualquer formata um filtro vazio ("sem filtro") para o prompt.
func filtroOuQualquer(valor string) string {
	if strings.TrimSpace(valor) == "" {
		return "any (no filter)"
	}
	return valor
}
