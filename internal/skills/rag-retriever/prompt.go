package ragretriever

import _ "embed"

//go:embed prompt.md
var skillPrompt string

// systemPrompt define a identidade do agente de reescrita de queries,
// enviado como instrução de sistema a cada chamada de fallback.
const systemPrompt = `
# Identity

You are a senior network engineer specialized in **FRRouting (FRR)**. Your
job is to rewrite a structured intent briefing into the exact search
queries to run against an FRR documentation knowledge base, following the
rules in the skill prompt.
`
