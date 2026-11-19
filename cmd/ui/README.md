# UI desktop (Wails v3)

O app desktop do network_agent, em Wails v3.1.0. O frontend é descrito em
[`docs/FRONTEND.md`](../../docs/FRONTEND.md), e os fatos do Wails ficam em
[`docs/WAILS_V3.md`](../../docs/WAILS_V3.md).

## Dev e build

Os comandos rodam em `cmd/ui`, com a CLI `wails3` 3.1.0
(`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0`):

```powershell
wails3 dev     # Vite na 9245; front recarrega, Go recompila e a janela reabre ao salvar
wails3 build   # .exe em cmd/ui/bin/net_agent.exe
```

Fechar a janela encerra o `wails3 dev`. Com `$env:NETWORK_AGENT_CDP_PORT = '9333'` antes do
`wails3 dev`, a janela abre a porta de CDP.

A configuração do projeto é o `build/config.yml` e os Taskfiles (`Taskfile.yml` e `build/`), gerados
por `wails3 generate build-assets`. A única edição no gerado é o `-ts` do `generate:bindings` em
`build/Taskfile.yml`. As bindings ficam em `frontend/bindings/`, versionadas: regenere-as (o
`wails3 dev` e o `wails3 build` já fazem isso) quando mudar um método ou tipo exposto ao front.
