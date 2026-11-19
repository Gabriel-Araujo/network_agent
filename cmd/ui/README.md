# UI desktop (Wails v3)

O app desktop do network_agent, em Wails v3.1.0. O frontend é descrito em
[`docs/FRONTEND.md`](../../docs/FRONTEND.md), e os fatos do Wails ficam em
[`docs/WAILS_V3.md`](../../docs/WAILS_V3.md).

## Dev

```sh
cd cmd/ui/frontend && npm run dev -- --port 9245 --strictPort
# em outro terminal, na raiz do repo:
FRONTEND_DEVSERVER_URL=http://localhost:9245 go run ./cmd/ui
```

Com `NETWORK_AGENT_CDP_PORT=9333`, a janela abre a porta de CDP.

## Build

```sh
cd cmd/ui/frontend && npm run build
go build -tags production -ldflags="-H windowsgui" ./cmd/ui
```
