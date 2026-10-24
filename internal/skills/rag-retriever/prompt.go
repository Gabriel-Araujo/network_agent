package ragretriever

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag"
)

//go:embed prompt.md
var skillPrompt string

// systemPrompt defines the query-rewriting agent identity and is sent as the
// system instruction on every call.
const systemPrompt = `
# Identity

You are a senior network engineer specialized in **FRRouting (FRR)**. Your
job is to rewrite a user's networking question into the exact search
queries to run against an FRR documentation knowledge base, following the
rules in the skill prompt.
`

// buildPrompt builds generator instructions from the identity, prompt.md
// rules, selected filters, and the vocabulary from rag.Lists().
func buildPrompt(filters rag.Filters) string {
	var b strings.Builder
	b.WriteString(systemPrompt + "\n" + skillPrompt)

	b.WriteString("\n# Search filters (apply to the whole search)\n")
	fmt.Fprintf(&b, "- protocol: %s\n", anyFilter(filters.Protocol))
	fmt.Fprintf(&b, "- daemon: %s\n", anyFilter(filters.Daemon))
	fmt.Fprintf(&b, "- chunk_type: %s\n", anyFilter(filters.ChunkType))

	b.WriteString("\n# Valid values (closed vocabulary)\n")
	for _, list := range rag.Lists() {
		names := make([]string, 0, len(list.Values))
		for _, value := range list.Values {
			names = append(names, value.Name)
		}
		fmt.Fprintf(&b, "- %s: %s\n", list.Name, strings.Join(names, ", "))
	}
	return b.String()
}

// anyFilter formats an empty filter for the prompt.
func anyFilter(value string) string {
	if strings.TrimSpace(value) == "" {
		return "any (no filter)"
	}
	return value
}
