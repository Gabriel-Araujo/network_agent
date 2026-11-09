# network_agent

Agente de LLM para engenharia de rede com FRRouting. Backend OpenAI-compatible em
`internal/agent`, UI desktop em `cmd/ui` (Wails v2 + React/Vite) e CLI em `cmd/cli`.

## Agent skills

### Issue tracker

Markdown local, um diretório por esforço em `.agent/<effort>/` (mapa + `issues/NN-<slug>.md`).
Não há issues no GitHub nem no GitLab. See `docs/agents/issue-tracker.md`.

### Domain docs

Single-context: glossário em `CONTEXT.md` na raiz, ADRs em `docs/adr/`.
See `docs/agents/domain.md`.
