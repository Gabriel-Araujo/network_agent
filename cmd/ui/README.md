# UI desktop (Wails v3)

O app desktop do network_agent, em Wails v3.1.0. O frontend é descrito em
[`docs/FRONTEND.md`](../../docs/FRONTEND.md), e os fatos do Wails ficam em
[`docs/WAILS_V3.md`](../../docs/WAILS_V3.md).

## Dev

Os comandos rodam em `cmd/ui`, com a CLI `wails3` 3.1.0
(`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0`):

```powershell
wails3 dev     # Vite na 9245, Go recompila ao salvar
wails3 build   # .exe em cmd/ui/bin/
```

> **Transição:** isso só funciona depois que o `build/config.yml`, o Taskfile e as bindings forem
> gerados. Até lá, o `wails dev` (CLI do v2) não roda o app, e o dev são dois terminais (PowerShell):

```powershell
# 1. em cmd/ui/frontend
npm run dev -- --port 9245 --strictPort

# 2. na raiz do repo
$env:FRONTEND_DEVSERVER_URL = 'http://localhost:9245'; go run ./cmd/ui
```

O front recarrega sozinho; mudou Go, rode o passo 2 de novo. Com
`$env:NETWORK_AGENT_CDP_PORT = '9333'` no passo 2, a janela abre a porta de CDP.

## Build (transição, até o `wails3 build`)

```sh
cd cmd/ui/frontend && npm run build
go build -tags production -ldflags="-H windowsgui" ./cmd/ui
```
