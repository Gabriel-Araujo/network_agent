package briefing

import (
	_ "embed"
	"fmt"
)

//go:embed prompt.md
var skillPrompt string

// systemPrompt define a identidade e o domínio da skill, enviado como
// instrução de sistema a cada geração de briefing.
const systemPrompt = `
# Identity

You are a senior network engineer specialized in **FRRouting (FRR)**. Your domain covers network hardware, routing protocols, switching, and the full configuration lifecycle of FRR-based environments.
`

// montarCorrecao cola o erro da tentativa rejeitada na próxima chamada.
func montarCorrecao(err error) string {
	return fmt.Sprintf(`Your previous answer was rejected: %v

Answer again with the full briefing as a single JSON object with exactly the four keys devices, connections, problemAndGoalSummary and executionPipeline — no preamble, no code fences, no file path.`, err)
}
