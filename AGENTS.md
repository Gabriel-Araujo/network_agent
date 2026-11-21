# network_agent

Agente de LLM para engenharia de rede com FRRouting. Backend OpenAI-compatible em
`internal/agent`, UI desktop em `cmd/ui` (Wails v3 + React/Vite) e CLI em `cmd/cli`.

## Frontend

Antes de mudar `cmd/ui/frontend`, leia [`docs/FRONTEND.md`](docs/FRONTEND.md). Antes de pesquisar o
React Spectrum, leia [`docs/REACT_SPECTRUM_S2.md`](docs/REACT_SPECTRUM_S2.md). Toda mudança no
frontend e todo fato novo do S2 atualizam o documento correspondente no mesmo commit.

## Agent skills

### Issue tracker

Markdown local, um diretório por esforço em `.agent/<effort>/` (mapa + `issues/NN-<slug>.md`).
Não há issues no GitHub nem no GitLab. See `docs/agents/issue-tracker.md`.

### Domain docs

Single-context: glossário em `CONTEXT.md` na raiz, ADRs em `docs/adr/`.
See `docs/agents/domain.md`.
