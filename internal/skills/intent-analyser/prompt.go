package intentanalyser

import _ "embed"

//go:embed prompt.md
var skillPrompt string

// systemPrompt define a identidade e o domínio do agente, enviado como
// instrução de sistema a cada varredura de intenção.
const systemPrompt = `
# Identity

You are a senior network engineer specialized in **FRRouting (FRR)**. Your domain covers network hardware, routing protocols, switching, and the full configuration lifecycle of FRR-based environments.
`
