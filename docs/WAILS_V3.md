# Wails v3 no app

Fatos sobre o Wails v3 que pesam na migração de `cmd/ui` (Go + WebView2 no Windows, frontend React 19
+ Vite 8). Coletado em 2026-11-19, cada fato com a fonte. Só fatos. O que o app **decidiu e
implementou** fica em [`FRONTEND.md`](FRONTEND.md). A única escolha registrada aqui é a versão
fixada, e ela é do usuário.

## Como usar e manter

- **Leia antes de pesquisar.** Pesquise só o que não estiver aqui, ou o que estiver numa versão do
  Wails diferente da fixada.
- **Fato novo ou corrigido entra aqui**, na seção do tema, com a versão, a data e a fonte. Um fato
  medido no app (`wails3 dev`, binário, WebView2) entra marcado como **Medido no app** e sai de
  [Não confirmado](#não-confirmado).
- **Ao subir a versão do Wails**, confira as seções contra a versão nova e atualize as versões de
  referência.

## Resumo

1. **Versão:** fixar a **3.1.0, decisão do usuário (2026-11-19)**. A 3.0.0 (2026-10-26) e a 3.1.0
   (2026-11-01) têm as mesmas opções públicas de app e de janela. A 3.1.0 traz correções (um diálogo
   pedido antes do `Run` não trava mais o processo) e o Wake, experimental. As duas pedem Go 1.26 ou
   mais novo, e o `go.mod` do app declara 1.27.1. A CLI é a `wails3`, com `wails3 dev` e
   `wails3 build` sobre um Taskfile.
2. **Migração:** `wails.Run` vira `application.New` + `app.Window.NewWithOptions` + `app.Run`.
   `Bind` vira `Services`, e `startup(ctx)` vira `ServiceStartup(ctx, opts)`. `runtime.EventsEmit`
   vira `app.Event.Emit`, com a mesma assinatura sem o `ctx`. O seletor de pasta vira
   `app.Dialog.OpenFile().CanChooseDirectories(true)`, e no Windows cancelar devolve **erro**. No
   front não há `window.go` nem `window.runtime`: tudo vem de import (bindings geradas em
   `frontend/bindings/` e o pacote `@wailsio/runtime`). O dado do evento chega em `e.data`, e o erro
   de binding chega como `RuntimeError`.
3. **Janela:** todas as opções decididas existem no v3, algumas em outro lugar. O frameless mantém
   canto e sombra por padrão. `Hidden` + `Show()` no `WebViewNavigationCompleted` é o par do
   `StartHidden` + `OnDomReady`. `WebviewUserDataPath` sobe para as opções do app. O arrasto continua
   `--wails-draggable`. No Windows, o runtime não maximiza no duplo clique. O Snap Layouts passa a
   existir em modo opcional (`WebView2CompositionHosting` + `--wails-non-client-region`). O estado
   de maximizada tem evento.
4. **Frontend:** o dev usa a porta 9245, a janela carrega `http://wails.localhost:9245` e o asset
   server repassa ao Vite. O template e o `@wailsio/runtime` já são do Vite 8, como o app. O
   `optimizeDeps.exclude` continua. O build de dev roda `npm run build:dev`, que o app não tem, e um
   `go mod tidy`. O CSP continua valendo. CDP: `Windows.AdditionalBrowserArgs` com
   `--remote-debugging-port` abre a porta na própria janela. O `localhost:34115` do v2 não existe
   mais. O download do diagrama fica com a interface do WebView2: o v3 não intercepta o
   `DownloadStarting`.
5. **Risco:** em 2026-11-19 não há issue aberta sobre frameless, arrasto, janela escondida ou
   bindings no Windows. Tocam o app: #6219 (primeira pintura 1–2 s atrasada no Windows), #6592
   (fatal no arranque se um monitor falha, 3.1.0), #6368 (`OnShutdown` não roda ao fechar a última
   janela), #6136 (o MCP derruba os listeners de evento) e #6337 (a documentação diverge do código).

## Fatos levantados

**Versões de referência (nesta data):** `github.com/wailsapp/wails/v3` **v3.1.0** (2026-11-01,
commit `4b454e4`) e v3.0.0 (2026-10-26, commit `7b11001`). Não há 3.1.x. Depois da 3.1.0 só saíram
`v3.2.0-nightly.20261101` a `…20261103`. No npm, `@wailsio/runtime` **3.1.0** (`latest`,
2026-11-01) e 3.0.0 (2026-10-26). O tag `nightly` está em `3.2.0-nightly.20261103`. O app está em
`github.com/wailsapp/wails/v2 v2.15.0` e `go 1.27.1` (`go.mod:3,9`). O toolchain local é o go1.27.1.
O frontend está no Vite **8.3.8** (`package.json` `^8.3.8`, instalado; ticket _Base do S2 no Vite 8_,
commit `a2d5f53`), que é o `latest` do npm (2026-11-11).
Fontes: <https://proxy.golang.org/github.com/wailsapp/wails/v3/@v/v3.1.0.info>,
<https://proxy.golang.org/github.com/wailsapp/wails/v3/@v/v3.0.0.info>, `go list -m -versions
github.com/wailsapp/wails/v3`, <https://registry.npmjs.org/@wailsio/runtime>,
<https://registry.npmjs.org/vite>.

**Método.** Os módulos v3.0.0 e v3.1.0 foram baixados com `go mod download` fora do repo. O
`go.mod` do app não mudou. Os caminhos de código abaixo são relativos a
`github.com/wailsapp/wails/v3@v3.1.0`, salvo indicação. O código do app é citado a partir da raiz
do repo. O site `wails.io` (o domínio atual do v3, conforme o `llms.txt` dele) e o `v3.wails.io`
respondem 403 a cliente sem navegador. Por isso as páginas da documentação foram lidas do fonte
Markdown do site no repo, no tag v3.1.0 (`docs/mpress/content/`), e são citadas pelo URL desse
fonte. As issues vieram da API de busca do GitHub em 2026-11-19. **Na pesquisa, nada foi executado** (o que foi medido depois está no §6): a `wails3`
não está instalada (só a `wails` v2 em `~/go/bin`), e nenhum binário v3 foi construído.

---

### 1. Versão

- **3.0.0 → 3.1.0 é uma semana.** A 3.1.0 é a primeira "weekly release": "Minor version numbers
  advance with time, not with features". O texto também pede para fixar a versão exata em vez de
  `@latest`.
  Fonte: <https://github.com/wailsapp/wails/releases/tag/v3.1.0>.
- **O que a 3.1.0 adiciona:** os backends sem CGO para macOS e Linux (experimental, #6521) e o
  Wake, que é o `wails.yaml` ligado por `WAILS_EXP_USE_WAKE=1` (experimental, #6531). O Task continua
  sendo o build padrão. A migração para o Wake é de mão única.
  Fonte: <https://github.com/wailsapp/wails/blob/v3.1.0/docs/mpress/content/blog/wails-v3-1-released.md#L19-L92>.
- **Correções da 3.1.0 que tocam o app:**
  - um diálogo pedido antes do `Run` devolve vazio e um aviso, em vez de travar o processo (#6390);
  - o trabalho mandado à thread principal antes do `Run` espera o `Run` (#6287);
  - cada listener Go recebe os eventos na ordem de emissão (#6275 e #6276);
  - o corpo da requisição é mantido quando o WebView2 omite o `Content-Length` (#6326);
  - a janela maximizada não é redimensionada por uma restrição nova (#6388);
  - os processos do dev param quando o terminal fecha (#6371), e o proxy do dev não perde mais o
    corpo entre tentativas (#6423 e #6432).

  Fontes: <https://github.com/wailsapp/wails/releases/tag/v3.1.0>; diff de
  `pkg/application/dialogs.go` entre v3.0.0 e v3.1.0 (`dialogsCanBeShown`, `dialogs.go:114`).
- **A API pública não muda entre as duas.** `pkg/application/webview_window_options.go` e
  `pkg/application/application_options.go` são idênticos na v3.0.0 e na v3.1.0. No runtime JS, só
  `src/system.ts` difere, na captura do `invoke`. No Windows, o diff de
  `webview_window_windows.go` mexe em menu, fullscreen e zoom.
  Fonte: `diff -r` dos dois módulos no cache.
- **Go:** os dois `go.mod` declaram `go 1.26.0` e `toolchain go1.26.10` (`go.mod:3-5`). As notas
  pedem "Go 1.26 or newer" e recomendam "Go 1.26.10 or 1.27.3 (or later), which include a
  `crypto/x509` security fix". O app declara 1.27.1, o que cumpre o mínimo mas fica abaixo da
  recomendação. O guia de migração ainda mostra `go 1.25.0`, o que não bate com o módulo.
  Fontes: `release_notes.md:18`;
  <https://github.com/wailsapp/wails/blob/v3.1.0/docs/mpress/content/migration/v2-to-v3.md#L257-L274>.
- **v2 e v3 convivem:** "Can I run v2 and v3 side by side? Yes, they use different import paths."
  Fonte: <https://github.com/wailsapp/wails/blob/v3.1.0/docs/mpress/content/migration/v2-to-v3.md#L684>.
- **CLI:** `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0`. O Task já vem embutido
  (`github.com/wailsapp/task/v3 v3.40.1-patched3`, `go.mod:25`). Os comandos:

  | v2 | v3 | O que faz |
  |---|---|---|
  | `wails dev` | `wails3 dev` | lê `./build/config.yml`, sobe o Vite na porta 9245 (`--port` ou `WAILS_VITE_PORT`) e recompila o Go |
  | `wails build` | `wails3 build` | roda a task `build` do Taskfile; o binário sai em `bin/<APP_NAME>.exe` (não há `build/bin/`) |
  | (gerado no `wails dev`) | `wails3 generate bindings` | gera `frontend/bindings/`; o `wails3 dev` regenera |

  O `wails3` não lê `wails.json`. O guia de migração mostra um "v3 (wails.json)", mas não há leitura
  desse arquivo na CLI (`internal/commands`). A configuração do v3 é `build/config.yml` + Taskfiles,
  gerados por `wails3 generate build-assets`.
  Fontes: <https://github.com/wailsapp/wails/blob/v3.1.0/docs/mpress/content/reference/cli.md#L24-L36>,
  `internal/commands/dev.go:13-14,28,57-80`,
  <https://github.com/wailsapp/wails/blob/v3.1.0/docs/mpress/content/migration/v2-to-v3.md#L467-L495>.

### 2. Migração: o que o app usa do v2 e o equivalente no v3

Inventário conferido em `cmd/ui/*.go` e `cmd/ui/frontend/src` (sem os testes) em 2026-11-19.

**Go**

| Uso hoje | v3 | Fonte (v3) |
|---|---|---|
| `wails.Run(&options.App{Title, Width, Height, …})` (`cmd/ui/main.go:44-69`) | `app := application.New(application.Options{Name, Services, Assets})`, `app.Window.NewWithOptions(application.WebviewWindowOptions{Title, Width, Height, …})`, `app.Run()` | [migração](https://github.com/wailsapp/wails/blob/v3.1.0/docs/mpress/content/migration/v2-to-v3.md#L22-L58), `internal/templates/_common/main.go.tmpl` |
| `AssetServer: &assetserver.Options{Assets: assets}` + `//go:embed all:frontend/dist` (`main.go:15-16,48-50`) | `Assets: application.AssetOptions{Handler: application.AssetFileServerFS(assets)}`, com o mesmo `go:embed`. O handler procura o `index.html` dentro do FS, então `frontend/dist/` serve como raiz. | `pkg/application/application_options.go:242-244`, `internal/assetserver/asset_fileserver.go:33` |
| `Bind: []interface{}{app}` (`main.go:66-68`) | `Services: []application.Service{application.NewService(app)}`. Todo método exportado vira binding, menos `ServiceName`, `ServiceStartup` e `ServiceShutdown`. O nome completo é `main.App.<Método>` (`PkgPath.Tipo.Método`). | `pkg/application/application_options.go:39`, `pkg/application/bindings.go:241-245` |
| `OnStartup: app.startup` grava `a.ctx` (`main.go:58`, `app.go:255-257`). O `a.ctx` vai para o turno, a janela de contexto e o reparo (`context.go:30`, `history.go:318,326`, `mermaid_repair.go:76,280`). | `func (a *App) ServiceStartup(ctx context.Context, opts application.ServiceOptions) error`. O `ctx` vale enquanto o app roda e é cancelado logo antes do shutdown. Um método ligado também pode receber `context.Context` como primeiro parâmetro. | `pkg/application/services.go:77-100`, `pkg/application/bindings.go:202,271` |
| `runtime.EventsEmit(a.ctx, name, data...)` atrás do campo `emit` (`app.go:33-37,107-109`). Eventos: `agent:token` e `agent:done` (`history.go:370,377`), `agent:status` (`app.go:260`, sem ouvinte no front). | `app.Event.Emit(name string, data ...any) bool`: a mesma forma, sem `ctx`. Com um dado só, `Data` é o próprio valor. JS mandado a uma janela antes do runtime dela carregar fica numa fila e roda depois. | `pkg/application/event_manager.go:31-45`, `pkg/application/webview_window.go:880-892` |
| Guarda de `ctx` nil, porque o runtime v2 com `ctx` nil chama `log.Fatalf` (`app.go:210-212`, `settings.go:66-70`) | Não há `ctx` no runtime v3. O `Emit` sem janela não tem caminho fatal no código. Na 3.1.0, um diálogo antes do `Run` devolve `""` e `nil`. | `pkg/application/events.go:165-191`, `pkg/application/dialogs.go:114-127,287-290` |
| Gaveta de Logs: pull por `GetLogLines`/`GetLogPath`, sem evento (`logs.go:29-32,90-97`) | Só muda a chamada no front (binding). | — |
| `runtime.OpenDirectoryDialog(a.ctx, OpenDialogOptions{Title, DefaultDirectory})` (`settings.go:72-79`) | `app.Dialog.OpenFile().CanChooseDirectories(true).CanChooseFiles(false).SetTitle(…).SetDirectory(…).PromptForSingleSelection()`. No Windows, `CanChooseDirectories(true)` abre o seletor de pasta. **Cancelar devolve `("", erro "cancelled by user")`**. O erro é `cfd.ErrorCancelled`, de um pacote `internal`, que o app não pode importar para comparar. O v2 devolvia `""` sem erro, e o `PickWorkspace` trata o erro como falha. | `pkg/application/dialog_manager.go:16`, `pkg/application/dialogs.go:234-306`, `pkg/application/dialogs_windows.go:211,262-284,378-416`, `internal/go-common-file-dialog/cfd/hresult.go:5-10`, `cfd/errors.go:6` |

**Frontend**

| Uso hoje | v3 | Fonte (v3) |
|---|---|---|
| `window.go.main.App.*` (`App.tsx:117,134,149,169,182,330,446,470`, `LogDrawer.tsx:59`, `ModelsSection.tsx:12`, `SettingsModal.tsx:16`), tipado à mão em `globals.d.ts` | Funções importadas das bindings geradas em `frontend/bindings/<caminho de import Go>/<serviço>.js` (`-ts` gera TypeScript; `-d` muda a pasta). Elas importam `@wailsio/runtime`; com `-b`, importam `/wails/runtime.js`. O script que o Go injeta define só `window._wails` e `window.wails`. **Não há `window.go` nem `window.runtime`.** | [bindings](https://github.com/wailsapp/wails/blob/v3.1.0/docs/mpress/content/features/bindings/methods.md#L164-L222), `internal/generator/render/module.go:22-24`, `internal/flags/bindings.go:19`, `internal/runtime/runtime.go:9-21` |
| Imports de `frontend/wailsjs/`: **nenhum**. Só o comentário em `globals.d.ts:4`. A pasta `wailsjs/` é do v2. | Sai. Os modelos Go viram classes TS (interfaces com `-i`) dentro de `frontend/bindings/`. | [models](https://github.com/wailsapp/wails/blob/v3.1.0/docs/mpress/content/features/bindings/models.md) |
| `window.runtime.EventsOn('agent:token', d => …)` / `EventsOff('agent:token','agent:done')` (`App.tsx:296-302`) | `import {Events} from '@wailsio/runtime'`; `const off = Events.On('agent:token', e => appendToken(e.data))`. `On` devolve a função de desligar, e `Off(...names)` existe. O callback recebe um `WailsEvent` com `.data`. | `internal/runtime/desktop/@wailsio/runtime/src/events.ts:61,150-177`; [migração](https://github.com/wailsapp/wails/blob/v3.1.0/docs/mpress/content/migration/v2-to-v3.md#L445-L465) |
| `window.runtime.ClipboardSetText(path)` (`LogDrawer.tsx:98-100`) | `Clipboard.SetText(text): Promise<void>` (no v2, `Promise<boolean>`) | `…/src/clipboard.ts:24` |
| `WindowIsMaximised`, `WindowMinimise`, `WindowToggleMaximise`, `Quit` (`WindowControls.tsx:28-50`) | `Window.IsMaximised()`, `Window.Minimise()`, `Window.ToggleMaximise()`, `Application.Quit()` | `…/src/window.ts:344,367,561`, `…/src/application.ts:35` |
| Erro de binding tratado com `String(e)` (`SettingsModal.tsx:53,135,155,167,177`, `ModelsSection.tsx:57,70,80`) | A promessa rejeita com `RuntimeError` (um `Error` com `name = "RuntimeError"`), então `String(e)` dá `"RuntimeError: <mensagem>"`. A chamada devolve um `CancellablePromise`. | `…/src/calls.ts:70-81`, `…/src/runtime.ts:36-46` |
| Arrasto, duplo clique, redimensionar: o runtime v2 vem injetado | Só funcionam com o `@wailsio/runtime` importado ao menos uma vez ("make sure to include a side-effect import"). | [frontend runtime](https://github.com/wailsapp/wails/blob/v3.1.0/docs/mpress/content/reference/frontend-runtime.md#L65-L78) |

**Sem equivalente:** nada do que o app usa fica sem par. Somem de propósito os globais
`window.go`/`window.runtime`, o callback variádico dos eventos (agora é um `data`) e o servidor de
navegador do `wails dev` ([§4](#4-frontend)).

### 3. Janela, contra as decisões de Tema e Shell

| Decisão (nome no v2.15) | v3 | Fonte (v3) |
|---|---|---|
| `Frameless: true` | `WebviewWindowOptions.Frameless` | `pkg/application/webview_window_options.go:102-103` |
| Canto e sombra do Windows 11 (`DisableFramelessWindowDecorations` falso por padrão) | `WebviewWindowOptions.Windows.DisableFramelessWindowDecorations`, também falso por padrão. Sem ele, o Wails estende o frame do DWM no `WM_ACTIVATE`, como no v2. A doc lista "Automatic drop shadow" e "Snap layouts support (Windows 11)". | `webview_window_options.go:374-377`, `pkg/application/webview_window_windows.go:465-467,1786-1793`, [frameless](https://github.com/wailsapp/wails/blob/v3.1.0/docs/mpress/content/features/windows/frameless.md#L531-L545) |
| `MinWidth` 900, `MinHeight` 600, janela de 1024×768 | `WebviewWindowOptions.MinWidth/MinHeight/Width/Height`, aplicados na criação | `webview_window_options.go:88-91,105-109`, `webview_window_windows.go:757-758` |
| `StartHidden: true` | `WebviewWindowOptions.Hidden: true`. A janela nasce sem `WS_VISIBLE` (issue #4611, o flash branco). | `webview_window_options.go:177-178`, `webview_window_windows.go:580-585` |
| `OnDomReady` → `runtime.WindowShow` | No Windows, o `OnDomReady` do v2 dispara no `NavigationCompleted` do WebView2 (`v2@v2.15.0/internal/frontend/desktop/windows/frontend.go:952-954`). O v3 emite `events.Windows.WebViewNavigationCompleted` no mesmo ponto, então fica `window.OnWindowEvent(events.Windows.WebViewNavigationCompleted, …)` + `window.Show()`. Há ainda o `events.Common.WindowRuntimeReady`, que sai quando o `@wailsio/runtime` é avaliado, antes do React renderizar. Um `Show()` antes do `NavigationCompleted` mostra a janela e segura o WebView por até 3 s. | `webview_window_windows.go:2900-2911,1426-1455`, `pkg/application/webview_window.go:880-882`, `…/src/index.ts:101-103`, `pkg/events/events.go:412` |
| Plano B do Tema (o Go lê o tema antes de abrir a janela) | O `w32.IsCurrentlyDarkMode()` público lê o `AppsUseLightTheme` do registro e funciona antes do `Run`. O `app.Env.IsDarkMode()` devolve `false` antes do `Run`, porque a plataforma só nasce no `Run`. | `pkg/w32/theme.go:270-282`, `pkg/application/environment_manager.go:36-41`, `pkg/application/application.go:676-678` |
| `windows.Options.WebviewUserDataPath` | `application.Options.Windows.WebviewUserDataPath`. Fica nas opções do **app**, não da janela. O padrão continua `%AppData%\<nome do exe>`. O `wails3 dev` e o `wails3 build` geram o mesmo `bin/<APP_NAME>.exe`, mas o `go run` gera outro nome, então o caminho fixo continua necessário. | `pkg/application/application_options.go:296-298`, `internal/webview2/pkg/edge/chromium.go:241-250`, [cli](https://github.com/wailsapp/wails/blob/v3.1.0/docs/mpress/content/reference/cli.md#L108-L111) |
| `BackgroundColour: &options.RGBA{…}` | `BackgroundColour: application.NewRGB(r, g, b)`, um valor `RGBA{Red, Green, Blue, Alpha}` e não um ponteiro. Ele pinta a janela e o WebView2. A doc diz "Default: White". O código não preenche padrão (o `NewWindow` só completa largura, altura, URL e nome), então o zero é `RGBA{0,0,0,0}`. Há `SetBackgroundColour` em Go e em JS. | `webview_window_options.go:125-126,256-276`, `webview_window_windows.go:455-463`, `webview_window.go:342-360`, [options](https://github.com/wailsapp/wails/blob/v3.1.0/docs/mpress/content/features/windows/options.md#L384-L415) |
| Arrasto por `--wails-draggable: drag`, controles com `no-drag` | O mesmo nome. O runtime lê o valor **computado** (herdado): só `drag` arrasta, e qualquer outro valor bloqueia. O arrasto começa no primeiro `mousemove` depois do `mousedown`. O Go faz `PostMessage(WM_NCLBUTTONDOWN, HTCAPTION)`. | `…/src/drag.ts:99-109,236-245`, `webview_window_windows.go:360-367`, [frameless](https://github.com/wailsapp/wails/blob/v3.1.0/docs/mpress/content/features/windows/frameless.md#L124-L144) |
| Duplo clique na área de arrasto maximiza (no v2, o JS só arrasta com `e.detail === 1`) | **No Windows, o runtime não maximiza.** O `dblclick` só vira `wails:drag:doubleclick` no macOS, e fora dele a preferência do Go é `"None"`. A guarda do `e.detail !== 1` só vale fora do Windows, então no Windows o `dblclick` chega ao DOM, e um `onDoubleClick` → `Window.ToggleMaximise()` o recebe. Com o modo de composição (abaixo), a área `caption` recebe `HTCAPTION` de verdade, e o duplo clique fica com o Windows. | `…/src/drag.ts:77-89,179-184`, `pkg/application/webview_window_titlebar_doubleclick_other.go:5-7`, `pkg/application/titlebar_doubleclick_action.go:11-19`, `pkg/application/webview_window_windows_nonclient.go:146-172` |
| Snap Layouts no hover do maximizar: perdido no v2.15 | **Existe no v3, opcional.** São dois modos. O `Windows.NonClientRegionSupport` lê o CSS `app-region: drag` e só arrasta, sem Snap Layouts. O `Windows.WebView2CompositionHosting` + `--wails-non-client-region: caption \| minimize \| maximize \| close` dá `HTMAXBUTTON` ao maximizar, e com isso o Snap Layouts aparece no hover. Ligar a composição liga o rastreio dessas regiões. A doc avisa que esse modo "may have rendering, input, focus, or WebView2 Runtime compatibility issues". As regiões só têm quatro tipos e nenhum "client": um retângulo `caption` cobre tudo o que estiver dentro dele, e a região que vem depois vence. O botão direito na `caption` abre o menu de sistema. Há também `Window.SnapAssist()`, que simula `Win+Z`. | [frameless](https://github.com/wailsapp/wails/blob/v3.1.0/docs/mpress/content/features/windows/frameless.md#L211-L363), `webview_window_options.go:387-400`, `webview_window_windows_nonclient.go:126-172`, `webview_window_windows.go:1759-1767,2902-2906,3220-3230`, `…/src/appregion.ts:25-33,54-76,216-231`, `…/src/window.ts:687` |
| Glifo do maximizar por polling de 400 ms (`WindowControls.tsx:27-46`) | Há evento. No Windows: `SIZE_MAXIMIZED` emite `WindowMaximise`, `SIZE_RESTORED` emite `WindowRestore` (inclusive ao arrastar a janela maximizada) e `UnMaximise()` emite `WindowUnMaximise`. Os três viram `Common.*`. Eventos de janela vão ao JS da própria janela como `common:WindowMaximise` etc., então `Events.On(Events.Types.Common.WindowMaximise, …)` e o mesmo para `WindowRestore` e `WindowUnMaximise`, com `Window.IsMaximised()` para ler o estado. | `webview_window_windows.go:1078-1081,1963,1994`, `pkg/events/defaults.go:5-24`, `webview_window.go:1028-1058,1648-1655`, `…/src/event_types.ts:260-293` |
| Tema da janela | `Windows.Theme` (padrão `SystemDefault`) colore a barra e o menu nativos, que o frameless não tem. O v3 não define a `PreferredColorScheme` do WebView2 (não há chamada no código), então `prefers-color-scheme` fica com o padrão do WebView2. | `webview_window_options.go:366-368`; busca por `PreferredColorScheme` em `pkg/application` e `internal/webview2/pkg/edge`: nada |

### 4. Frontend

- **Como o dev sobe.** O `wails3 dev` checa a porta (9245 por padrão, `--port` ou
  `WAILS_VITE_PORT`) e exporta `FRONTEND_DEVSERVER_URL=http://localhost:<porta>`. Depois roda o
  `dev_mode` do `build/config.yml`. Primeiro vem, bloqueando, `wails3 build DEV=true`. Depois, em
  background, `wails3 task common:dev:frontend`, que roda `npm run dev -- --port <porta> --strictPort`
  e espera até 60 s por um HTTP 200 nessa URL.
  Fontes: `internal/commands/dev.go:57-80`, `internal/commands/build_assets/config.yml:34-64`,
  `internal/commands/build_assets/Taskfile.tmpl.yml:206-216`.
- **O build de dev roda `npm run build:dev`.** O `build:frontend` gera as bindings e chama o script
  `build:dev` quando `DEV=true`. O template React define `"build:dev": "tsc && vite build --minify
  false --mode development"`. O `package.json` do app só tem `dev`, `build`, `preview` e `test`.
  Fontes: `Taskfile.tmpl.yml:72-100,191`, `internal/templates/react/frontend/package.json.tmpl:8-9`,
  `cmd/ui/frontend/package.json:6-11`.
- **O Taskfile roda `go mod tidy`** antes de gerar as bindings (`go:mod:tidy`). O template supõe um
  `go.mod` próprio ao lado do `main.go`. Neste repo, `cmd/ui` é um pacote do módulo da raiz, então o
  `tidy` mexeria no `go.mod` e no `go.sum` do repo inteiro. Isso é inferência do Taskfile: não foi
  rodado.
  Fonte: `Taskfile.tmpl.yml:4-11,180`.
- **De onde a janela carrega.** No Windows, a origem é `http://wails.localhost`. No dev, ela ganha a
  porta do Vite (`http://wails.localhost:9245/`). O asset server responde `/wails/runtime*` e repassa
  o resto ao `FRONTEND_DEVSERVER_URL`, inclusive o WebSocket do HMR. O Vite 8.3.8 aceita `*.localhost`
  por padrão ("`localhost` and domains under `.localhost` and all IP addresses are allowed by
  default").
  Fontes: `internal/assetserver/assetserver_windows.go:5-8`, `internal/assetserver/assetserver.go:167-193`,
  [asset server](https://github.com/wailsapp/wails/blob/v3.1.0/docs/mpress/content/contributing/asset-server.md#L43-L84),
  <https://github.com/vitejs/vite/blob/v8.3.8/docs/config/server-options.md> (`server.allowedHosts`).
- **O `vite.config` do template** fixa `server.host: "127.0.0.1"`, a porta de `WAILS_VITE_PORT` e
  `strictPort`, e põe o plugin `wails("./bindings")`. O `vite.config.ts` do app não tem bloco
  `server` (o padrão do Vite 8.3.8 é `host: 'localhost'`). A CLI do Vite 8 aceita o `--port` e o
  `--strictPort` que o Taskfile passa.
  Fontes: `internal/templates/react/frontend/vite.config.ts:6-12`,
  <https://github.com/vitejs/vite/blob/v8.3.8/docs/config/server-options.md> (`server.host`),
  <https://github.com/vitejs/vite/blob/v8.3.8/docs/guide/cli.md> (`--port`, `--strictPort`).
- **`optimizeDeps.exclude` do macro do S2:** é configuração do Vite, e o Wails não toca no
  `vite.config`. A CLI só passa `--port` e `--strictPort`. Então continua. **Medido no app** (Vite
  8.3.8, 2026-11-19, ticket _Base do S2 no Vite 8_): sem ele, o scan do Rolldown falha e a página
  recarrega na primeira carga.
  Fontes: `Taskfile.tmpl.yml:214-216`, [`REACT_SPECTRUM_S2.md`](REACT_SPECTRUM_S2.md) §2.
- **Plugin Vite do Wails (`@wailsio/runtime/plugins/vite`):** é opcional e só serve para eventos
  tipados. Com ele, o build falha se as bindings não tiverem sido geradas. Ele usa só
  `buildStart`, `resolveId` e `load`. O `@wailsio/runtime` 3.1.0 não declara `peerDependencies` nem
  `dependencies`. Ele é desenvolvido contra o `vite ^8.2.4`, e o template React usa o
  `vite ^8.2.4` com React 18. É o mesmo major do app (8.3.8).
  Fontes: `internal/runtime/desktop/@wailsio/runtime/src/plugins/vite.ts`,
  `internal/runtime/desktop/@wailsio/runtime/package.json`, <https://registry.npmjs.org/@wailsio/runtime>,
  `internal/templates/react/frontend/package.json.tmpl`.
- **Embutir o `dist`:** o mesmo `//go:embed all:frontend/dist`, com `AssetFileServerFS` (ver §2). O
  build de produção no Windows usa `-tags production -trimpath -buildvcs=false -ldflags="-w -s -H
  windowsgui"` e `CGO_ENABLED=0` por padrão.
  Fonte: `internal/commands/build_assets/windows/Taskfile.yml:20-31,60-63`.
- **CSP:** o `<meta>` com `font-src 'self' data:` continua valendo. As fontes do bundle vêm da
  mesma origem (`wails.localhost`, com ou sem porta), que é o `'self'`. O asset server do v3 não
  manda cabeçalho de CSP (busca por `Content-Security-Policy` em `internal/assetserver` e
  `pkg/application`: nada), então o `<meta>` segue como a única política.
- **CDP e depuração:**
  - O v3 ainda zera `WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS` e as chaves de registro, então a
    variável de ambiente continua sem efeito.
  - Agora há a opção `application.Options.Windows.AdditionalBrowserArgs`, cujo exemplo é
    `"--remote-debugging-port=9222"`. Ela entra no ambiente WebView2 compartilhado por todas as
    janelas. Isso abre a porta de CDP **na própria janela** do app.
  - O servidor de navegador `localhost:34115` do `wails dev` não existe no v3 (busca por `34115` no
    módulo: nada).
  - O DevTools fica ligado nos builds sem a tag `production`.
  - Há também o servidor MCP (`WAILS_MCP=1 wails3 dev`, em `127.0.0.1:9099`), com `js_eval`,
    `dom_query` e `screenshot_dom`. Ver o risco #6136 em §5.

  Fontes: `internal/webview2/webviewloader/env_create.go:165-176`,
  `pkg/application/application_options.go:303-309`, `webview_window_windows.go:2585-2608`,
  [options](https://github.com/wailsapp/wails/blob/v3.1.0/docs/mpress/content/features/windows/options.md#L1151-L1162),
  `pkg/application/webview_window_windows_devtools.go:1`,
  [MCP](https://github.com/wailsapp/wails/blob/v3.1.0/docs/mpress/content/guides/mcp-service.md#L26-L31).
- **Download do diagrama (`<a download>` com blob):** o v3.1.0 no Windows **não intercepta**.
  - A janela não assina o `DownloadStarting` do WebView2, e nada mexe no diálogo de download (busca
    por `DownloadStarting` e `DownloadDialog` em `pkg/application` e `internal/webview2/pkg/edge`:
    nada). Os tipos COM existem, mas ninguém os liga.
  - O `navigationStarting` só limpa as regiões de non-client e não cancela nada.
  - Então o download fica com a interface padrão do WebView2, como no v2.
  - Nenhuma issue aberta sobre download no WebView2 do v3 (busca de 2026-11-19).

  Isso é inferência do código, não medida.
  Fontes: `internal/webview2/pkg/webview2/ICoreWebView2DownloadStartingEventArgs.go`,
  `pkg/application/webview_window_windows.go:2878-2880`.

### 5. Risco: issues abertas (busca de 2026-11-19)

Busca em `wailsapp/wails`, issues abertas, com os termos frameless, drag, draggable, hidden,
double click, title bar, maximize, snap layouts, non-client, composition, bindings, white flash e
Vite/HMR. **Nenhuma issue aberta trata de frameless, arrasto, janela escondida ou bindings no
Windows.** As que tocam o app:

| Issue | Versão relatada | O que toca |
|---|---|---|
| [#6219](https://github.com/wailsapp/wails/issues/6219) (2026-10-04) | v3.0.0-beta.27, Windows | A primeira pintura atrasa 1–2 s depois de imports paralelos de módulos ES, com a janela já visível e em branco. Isso toca o "nascer escondida e aparecer pronta" do Tema. |
| [#6592](https://github.com/wailsapp/wails/issues/6592) (2026-11-10) | v3.1.0 e v3.0.0, Windows | Se um monitor falha no `GetMonitorInfo` ou no `GetDpiForMonitor`, a enumeração aborta. O resultado é um fatal no arranque, e cada chamada vaza um slot de `syscall.NewCallback`. |
| [#6368](https://github.com/wailsapp/wails/issues/6368) (2026-10-26) | v3.0.0-beta.34, Windows | O `OnShutdown` não roda quando a última janela fecha pelo X. Só roda com `app.Quit()`. O código da 3.1.0 ainda só faz `PostQuitMessage` nesse caminho (`pkg/application/application_windows.go:360-361`). |
| [#6375](https://github.com/wailsapp/wails/issues/6375) (2026-10-27) | v3.0.0, Windows | Uma corrida no `Quit()` dá fatal e deixa um processo zumbi, mas só com um systray de terceiros numa thread própria. O app não tem systray. |
| [#6355](https://github.com/wailsapp/wails/issues/6355) (2026-10-25) | v3, Windows/Linux | Com `CanChooseDirectories(true)`, a pasta vence e não dá para escolher arquivo. Para o seletor de Workspace isso é o comportamento desejado. |
| [#6136](https://github.com/wailsapp/wails/issues/6136) (2026-09-17) | v3.0.0-beta.16, macOS | O `call_bound_method` do MCP carrega uma segunda cópia do runtime e cala todos os `Events.On` da página. Usar o MCP para conferir o stream do turno exige evitar essa ferramenta. |
| [#6530](https://github.com/wailsapp/wails/issues/6530), [#6550](https://github.com/wailsapp/wails/issues/6550) (2026-11) | v3, Windows | Corpos grandes de POST ficam lentos. As bindings vão por POST, mas os payloads do app são pequenos. |
| [#6595](https://github.com/wailsapp/wails/issues/6595) (2026-11-11) | v3, Windows | A janela de composição do IME (CJK) perde a âncora do cursor depois de trocar de janela. O pt-BR não usa IME. |
| [#6337](https://github.com/wailsapp/wails/issues/6337) (2026-10-25) | docs v3.0.0 | A auditoria achou 31 divergências entre a documentação e o código. Exemplos vistos aqui: o "Default: White" do `BackgroundColour`, o `wails.json` v3 e o `go 1.25.0` do guia de migração. Na dúvida, o código vence. |

---

### 6. Medido no app (v3.1.0, WebView2 157.0.4341.60, Windows 11, 2026-11-19)

Ticket _App no Wails v3_. O dev rodou sem a `wails3`: Vite em `--port 9245 --strictPort` e o binário
com `FRONTEND_DEVSERVER_URL=http://localhost:9245` (o handler de dev do asset server lê essa
variável em build sem a tag `production`, `internal/assetserver/build_dev.go:161`).

- **O proxy do dev disca só IPv4.** Com o `vite.config.ts` sem `server.host`, a página não carrega,
  e o log mostra `[ExternalAssetHandler] Proxy error error=dial tcp4 127.0.0.1:9245: connectex: …
  recusou`. O Vite no Windows escuta `localhost` em `::1`. Com `server.host: '127.0.0.1'` (o mesmo
  do template), a página carrega.
- **`Call.ByName('main.App.<Método>')` funciona** para o pacote `main` de `cmd/ui`: listagem,
  criação, renomear, apagar, `AskAgent`, `GetConfig`, `PickWorkspace`. Não foi preciso gerar
  bindings.
- **O stream chega:** um turno de ~80 palavras deu 82 eventos `agent:token` e um `agent:done` em
  `Events.On`, com o dado em `e.data`.
- **Cancelar o seletor de pasta** devolve o erro `"cancelled by user"`. O `PickWorkspace` o trata
  como cancelar, e a promessa resolve com o Workspace anterior, sem erro. O título do diálogo é o
  do `SetTitle`.
- **`Window.Minimise/ToggleMaximise/IsMaximised/IsMinimised` e `Application.Quit`** funcionam pelo
  `@wailsio/runtime`. O `Quit` encerra o processo com código 0.
- **`Clipboard.SetText`** grava na área de transferência do Windows.
- **CDP:** `AdditionalBrowserArgs: ["--remote-debugging-port=9333"]` abre a porta na janela do app.
  A página aparece em `/json/list` como `http://wails.localhost:9245/`.
- **`window._wails.environment`** (`{"OS":"windows","Arch":"amd64","Debug":true}`) existe na página
  do app. O app o usa para saber se a ponte existe, porque o `@wailsio/runtime` cria o
  `window._wails` sozinho, até no browser puro.
- **Primeira carga com o `@wailsio/runtime` novo:** o Vite otimiza a dependência e recarrega a
  página uma vez. Uma chamada feita antes do reload pode falhar. Na carga seguinte não acontece.

## Não confirmado

Os itens medidos saíram daqui e estão no §6.

- Se o arrasto pela topbar funciona e se o duplo clique não começa um arrasto. Não foi conferido
  por script, porque pede o mouse real.
- Se escolher uma pasta no seletor grava o Workspace (o cancelar foi medido).
- Se `Hidden` + `Show()` no `WebViewNavigationCompleted` abre sem WebView2 em branco, e se o plano B
  do Tema continua necessário. Também não se sabe se o #6219 aparece no bundle do app.
- O efeito visual do `BackgroundColour` zero (`RGBA{0,0,0,0}`) com a janela escondida.
- Se o `onDoubleClick` na titlebar recebe o duplo clique no Windows sem que um arrasto comece entre
  os dois cliques.
- Se `WebView2CompositionHosting` funciona com os controles do S2 (popover, foco, cursor) no
  WebView2 instalado, e se o Snap Layouts aparece no hover.
- Se `--wails-draggable` (arrasto em JS) e `--wails-non-client-region: maximize` convivem na mesma
  titlebar.
- O HMR do Vite 8 atrás do proxy em `wails.localhost:9245`. O app não usa o plugin Vite do Wails.
- Se o download do diagrama mostra a interface do WebView2 no v3 (o código diz que sim).
- Se a Typekit continua barrada pelo CSP no v3 (o mesmo `<meta>`, não medido).
