# Frontend (`cmd/ui/frontend`)

O frontend do app desktop: React 19 + TypeScript 7 + Vite, empacotado pelo Wails v3.1.0 e renderizado
no WebView2. Conversa com o Go pelas bindings geradas em `frontend/bindings/` e pelo
`@wailsio/runtime`.

Este documento descreve o frontend **como ele está**. Os fatos sobre a biblioteca de componentes
(React Spectrum S2) ficam em [`REACT_SPECTRUM_S2.md`](REACT_SPECTRUM_S2.md).

## Como manter

- **Mudou o comportamento ou a estrutura, atualize aqui no mesmo commit.** Escreva o que o app faz
  e onde isso mora (arquivo, componente), não a discussão que levou até lá.
- Uma seção tem um de três estados:
  - **_Pendente_:** nada decidido ainda.
  - **_Decidido, a implementar_:** o comportamento já está escrito, mas o código ainda não o faz.
    Quem decide troca o _Pendente_ por este marcador e escreve o comportamento.
  - **Sem marcador:** o código já faz o que o texto diz. Quem implementa tira o marcador e confere
    o texto contra o código.
- Fato sobre o S2 (API, limite, bug, versão) vai para [`REACT_SPECTRUM_S2.md`](REACT_SPECTRUM_S2.md),
  não aqui. Aqui fica só o que o app fez com ele.

## Regras de estilo

- **Referência visual:** o protótipo [`FRONTEND_PROTOTIPO.html`](FRONTEND_PROTOTIPO.html), que abre
  direto no navegador. Dele vêm a estrutura e as medidas do layout. Cor, tipo, raio, espaço e
  movimento vêm do S2 ([ADR 0005](adr/0005-own-surfaces-take-s2-tokens-not-the-prototype-css.md)).
  Ele carrega fontes do Google Fonts; o app, não.
- **Quem vence quando o Spectrum e o protótipo discordam:** o Spectrum, nos controles padrão (botão,
  picker, menu, dialog, tabs, toast, campos, radio); o protótipo, no layout do shell e das
  superfícies sem equivalente (composer, trilho de Perguntas, diagrama, gaveta de Logs). Exceção
  (usuário, 2026-11-20): no modal de Configurações vence o protótipo, com o menu vertical e o fundo
  borrado (ver **Configurações**).
- **Superfície própria** é montada em `react-aria-components` (ou HTML puro) e estilizada pelo macro
  `style` com os tokens do S2, não com cores soltas nem com as variáveis do protótipo.
- **Nada depende de rede em runtime:** fontes, ícones e assets vão embutidos no bundle. Para
  fontes, o CSP do `index.html` garante isso (ver [Stack e build](#stack-e-build)).
- **Nenhum CSS global fora de `@layer`:** ele vence o CSS do S2.

## Stack e build

- **Versões:** `@react-spectrum/s2` 1.8.2, que traz `react-aria-components` 1.22.1 e
  `react-aria` 3.53.1. Nas devDeps: `unplugin-parcel-macros` 0.2.0, que puxa `@swc/core` 1.16.22, e
  `lightningcss` 1.33.0 e `@react-aria/optimize-locales-plugin` 2.1.0. O Vite é o 8.3.8, que empacota com Rolldown 1.2.19, com
  `@vitejs/plugin-react` 5.2.0 e `vitest` 5.1.0.
- **`vite.config.ts`:** a config oficial do S2. `macros.vite()` é o primeiro plugin, seguido do
  `optimizeLocales.vite({locales: ['pt-BR']})` com `enforce: 'pre'` (ver **Idioma**), com
  `test.server.deps.inline` do S2 (o Markdown, nos testes, importa componentes dele, que importam `.css`, e o Node não os carrega), `test.fsModuleCache: false` (o cache em disco do vitest pula o macro; ver `REACT_SPECTRUM_S2.md`), `build.target: ['es2022']`, `cssMinify: 'lightningcss'` e um `manualChunks` que junta o CSS do S2
  e dos macros no chunk `s2-styles`. Há ainda `optimizeDeps.exclude: ['@react-spectrum/s2/style']`:
  sem ele, o scan do `vite dev` não resolve o import do macro e recarrega a página na primeira
  carga (no Vite 8 também). Os nomes `rollupOptions` e `manualChunks` estão deprecados no Vite 8, mas
  funcionam; ficam como na doc do S2. Do template do Wails v3 vêm o plugin `wails('./bindings')` e o
  bloco `server` (`host: '127.0.0.1'`, porta de `WAILS_VITE_PORT` ou 9245, `strictPort`). O
  `package.json` tem o `build:dev` do template (`vite build --minify false --mode development`), que
  o `wails3 dev` chama, e o `build` passa `--mode production`.
- **`tsconfig.json`:** `moduleResolution: "Bundler"`, porque com `"Node"` os sub-paths
  (`@react-spectrum/s2/Button`) dão `TS2307`. O `tsconfig.node.json`, que só cobre o
  `vite.config.ts`, segue em `"Node"`. Ele já não passava no `tsc -p` antes do S2 e não entra no
  `npm run build`. O `include` cobre `src` e `bindings`. O TypeScript é o 7.0.2, que removeu o
  `esModuleInterop` (sempre ligado), então a opção saiu do `tsconfig.json`.
- **`allowScripts`:** `"@swc/core": false`. O `postinstall` dele só instala o fallback wasm, e o
  binário nativo chega por `optionalDependencies`.
- **CSS global:** `src/global.css`, importado primeiro no `main.tsx`, é o único CSS do app: não há
  `App.css` nem `Settings.css`, e toda superfície é do macro `style`. Ele declara a ordem das camadas,
  `@layer base, _, reduced-motion;`, e traz os `@font-face` (**Fonte**), a regra de movimento reduzido
  (**Movimento**) e a camada `base`: `box-sizing: border-box` em tudo, `margin: 0` e
  `overscroll-behavior: none` no `body` e a barra de rolagem (**Barra de rolagem**, em **Shell e janela**).
  Nada mais do navegador é zerado: `margin` de `h1`, `p`, `ul`, `pre`, `h3` e a borda de
  `borderStyle: 'solid'` sem `borderWidth: 0` (os outros lados saem com 3 px) são do estilo de cada
  superfície. No build, o `index-*.css` (global) carrega antes do `s2-styles-*.css`, e a ordem final é
  `base`, `_`, `reduced-motion`.
- **Rede:** o `index.html` traz `<meta http-equiv="Content-Security-Policy" content="font-src 'self' data:">`.
  O `Provider` do S2 sempre tenta a Adobe Clean em `use.typekit.net`, e essa política faz o WebView2
  barrar essas requisições antes de saírem (medido: 49 entradas de `use.typekit.net`, todas com
  status 0 e 0 bytes). As fontes que carregam são só as do bundle (**Fonte**).
- **Wails v3.1.0** (fixada por decisão do usuário, 2026-11-19; fatos em [`WAILS_V3.md`](WAILS_V3.md)):
  - `cmd/ui/main.go` monta `application.New` com o `App` como único serviço e uma janela frameless
    1024×768 que nasce escondida (ver **Tema**). O `App.ServiceStartup` guarda o `ctx` dos turnos, e o `emit` dos
    eventos `agent:token`/`agent:done` vai pelo `Event.Emit` do app.
  - **Bindings:** `frontend/bindings/` é gerada pelo `wails3 generate bindings` em TypeScript
    (`-ts`, a única edição no Taskfile gerado: o TS 7 não lê o `@enum` do JSDoc das bindings em JS)
    e é **versionada**, para que `npm run build` e `npm test` passem sem a CLI. O `wails3 dev` e o
    `wails3 build` a regeneram. Os métodos do `App` vêm de
    `bindings/github.com/Gabriel-Araujo/network_agent/cmd/ui` (`App.<Método>`; no `App.tsx`,
    `GoApp`, porque o componente já se chama `App`). `Events`, `Window`, `Application` e `Clipboard`
    vêm direto do `@wailsio/runtime`, que as bindings importam, e isso liga o arrasto da titlebar.
    O evento chega com o dado em `e.data`.
  - **Tipos:** os modelos Go das bindings são os tipos do front onde há equivalente. O `types.ts`
    os reexporta (`AgentError`, `ToolCall`, `TokenUsage`, `MermaidFailure`, `MermaidReplacement`,
    `MermaidRepairStart`, `ContextWindow`, `Conversation`, e `AppConfig`/`ProviderConfig`/`ModelConfig`
    a partir de `types.Config`/`Provider`/`Model`). Ficam à mão os da tela: `Message` (o
    `toUiMessages` a projeta da mensagem do Go, convertendo os enums `Role`/`Status` em literais),
    `MermaidRepair` (o estado `repairing`) e `Connection` (o Go devolve `state` como `string`).
  - **Fora do app** (browser puro, testes): `inWails()` (`types.ts`) olha o
    `window._wails.environment`, que só existe quando o Go injeta o script, e nenhuma chamada ao Go é
    feita sem ele. O erro de uma chamada é um `RuntimeError` cuja `message` é o erro do Go, e
    `errorText(e)` tira dele o texto.
  - **A ponte chega depois do primeiro render:** o Go injeta o `environment` depois que a página
    carregou, então o `App` tem um estado `bridge` (`pending`/`ready`/`absent`) que espera até 3 s
    pela ponte antes de carregar config/histórico/conexão e de assinar `agent:token`/`agent:done`.
    Sem ela, abre como browser puro: sem histórico, e o envio falha visível.
  - Cancelar o seletor de pasta não é erro: o `PickWorkspace` trata o `"cancelled by user"` do v3
    como o `""` do v2.
- **Bundle (Vite 8):** com o Shell, as Abas, a Início, a Conversa (Titlebar, Sidebar, TabBar, StatusBar,
  Home, composer, botões da janela, bolhas, Markdown, bloco de código, trilho), o Bloco de diagrama, a
  gaveta de Logs, as Configurações e só o pt-BR, o JS tem 1.642,94 KB (gzip 536,03 KB) e o chunk
  `s2-styles` 80,85 KB (gzip 15,02 KB); o `index-*.css` tem 1,23 KB (o menu próprio em RAC no lugar do `Tabs` do S2 tirou 24 KB do JS, gzip 6,4 KB, e 0,7 KB do CSS). O modal trouxe `Dialog`, `Tabs`
  vertical, `ListView`, `SelectBoxGroup`, `NumberField` e `Badge` (+119 KB, gzip +30,6 KB no JS, e
  +15 KB no CSS), e tirar o phosphor não compensou, porque ele já saía do bundle por tree-shaking.
  Antes das Configurações eram 1.547 KB (gzip 511,69 KB) e 67,17 KB (gzip 11,96 KB), antes da gaveta
  1.543 KB (gzip 510,6 KB) e 66,76 KB (gzip 11,86 KB), antes do Bloco de diagrama 1.538 KB (gzip 508 KB) e 64,2 KB (gzip 11,4 KB), e antes da Conversa 1.525 KB
  (gzip 504 KB) e 60,6 KB (gzip 10,6 KB). Só com `Provider` + `ToastContainer` eram 1.327 KB (gzip 446 KB) e 44 KB
  (gzip 7,3 KB); sem o filtro de locales, 1.427 KB (gzip 469 KB). As três fontes somam 98 KB.
  Medição anterior em [`REACT_SPECTRUM_S2.md`](REACT_SPECTRUM_S2.md#9-peso).

## Tema

O `Provider` (`main.tsx`) lê o tema de `src/theme.ts` (`useTheme`) e passa `colorScheme` só quando a
escolha é Claro ou Escuro. O **Seletor** é a aba Aparência das Configurações.

- **Três escolhas, padrão Sistema.** Claro e Escuro passam `colorScheme` ao `Provider`. Sistema,
  que vale também quando nada foi salvo, omite o prop, e então o app acompanha o Windows mesmo com a
  janela aberta. Não há paleta própria: o escuro é o do S2.
- **Seletor:** em Configurações → Aparência (ver **Configurações**), um `SelectBoxGroup` com três
  cartões de prévia (Claro, Escuro, Sistema). A troca vale ao clicar, sem botão Salvar: o cartão chama
  `setTheme` (`theme.ts`), que guarda a escolha, põe o `color-scheme` no `<html>` (o `light-dark()` do fundo
  do script inline acompanha) e avisa os `useTheme` por `useSyncExternalStore`, então o `Provider` e todo o app
  trocam na hora, sem recarregar.
- **Persistência:** a escolha vai para o `localStorage` do WebView2, na chave `theme` (`'light'` ou
  `'dark'`; ausente é o Sistema, e escolher Sistema apaga a chave). Conferido: o escuro escolhido pelo
  seletor volta igual depois de reiniciar o app. Para que o `wails3 dev`, o `go run` e o binário construído
  enxerguem a mesma escolha, o `main.go` fixa `application.Options.Windows.WebviewUserDataPath` em
  `~/.network_agent/webview2` (ao lado do `config.json`). Sem ele, o WebView2 usa
  `%AppData%\<nome do exe>`, uma pasta por forma de rodar.
- **Sem piscar ao abrir:** a janela nasce com `Hidden: true`, e o `main.go` chama `window.Show()` no
  `events.Windows.WebViewNavigationCompleted`. Antes disso, um script inline no `<head>` do
  `index.html` lê a chave `theme` e pinta o fundo do `<html>` com
  `background-color: light-dark(rgb(248, 248, 248), rgb(27, 27, 27))` sob `color-scheme` `light` ou
  `dark`, ou `light dark` no Sistema. Esses dois valores são o `layer-1` da moldura (ver **Shell e
  janela**), o mesmo que o `Provider` pinta depois, então a janela não troca de cor quando o React
  monta (conferido com o bundle bloqueado: `root` vazio e o `<html>` já em `rgb(248, 248, 248)` e
  `rgb(27, 27, 27)`). Se o `layer-1` mudar, o script muda junto. A janela não tem mais
  `BackgroundColour`. O app não pinta um `body` próprio.
- **Diagrama** (`Mermaid.tsx`): o pele lê `--pele-*`, e o macro `style` do canvas as define com os tokens
  do S2 (fundo `gray-25`, texto `neutral`, linha `gray-600`, superfícies `gray-100`/`gray-200`,
  borda `gray-400`, acento `accent`, fonte `sans`; a `--pele-font-mono` é a JetBrains Mono, por
  `css()`). Elas saem como `light-dark(…)`, então o SVG na tela troca de tema sozinho, sem
  renderizar de novo. O SVG baixado leva o tema em uso, com as cores **resolvidas** (rgb) no `<style>`
  injetado, mais `background: var(--pele-bg)` (o SVG não pinta o fundo, e o escuro sobre a página
  branca de quem o abre ficaria ilegível). Um `light-dark(…)` dos tokens não pode ir para o arquivo,
  porque fora do app ele sempre resolveria para o claro ou nem renderizaria. Para resolver, o
  `resolvedTheme` põe um `span` de prova no canvas, aplica cada `--pele-*` em `color` (as fontes em
  `font-family`) e lê o valor computado.
- **Bloco de código:** sem realce de sintaxe, então as cores `--code-*` do protótipo não entram.

## Shell e janela

A janela é o app: o shell não desenha a moldura do protótipo (padding, gradiente, raio, sombra), e
não há brilhos de fundo. Canto arredondado e sombra ficam com o Windows 11, que os dá à janela
frameless. As opções de janela ficam no `cmd/ui/main.go`.

- **Grade** (`App.tsx`, macro `style`): a titlebar (48 px) em cima, na largura toda, as três colunas
  no meio (sidebar ou faixa, conteúdo e gaveta de Logs) e a barra de status embaixo, na largura
  toda. A coluna do conteúdo é o `Tabs` do RAC: a barra de abas em cima e um painel por aba embaixo.
- **Moldura e cartão:** a titlebar, a sidebar (ou a faixa recolhida), a gaveta de Logs e a barra de
  status são a **moldura**, em `layer-1`, **pintada uma vez só**: o `Provider` (`main.tsx`) leva
  `background="layer-1"`, e a grade do shell e as frestas entre as colunas (a margem de 8 px do cartão)
  ficam sobre ele. Nenhum desses quatro componentes pinta o próprio fundo. Sem isso, a margem do cartão
  mostrava o `Canvas` do `<html>`, branco no claro (o "quadro branco" do print do ticket
  [Fundo da moldura e barra de rolagem](../.agent/frontend-v2/issues/27-fundo-da-moldura-e-barra-de-rolagem.md)).
  Um fundo que o próprio componente precisa (o campo de renomear da sidebar, o bloco de código, a
  Início) segue no componente. A coluna do conteúdo é um **cartão** solto dela, em
  `layer-2` (`content` no `App.tsx`): `margin: 8`, `border` de 1 px em `gray-300`, `borderRadius:
  'xl'` (16 px, o raio exato do protótipo) e `overflow: hidden`, então a barra de abas fica dentro
  dele e com a cor dele. O cartão pinta o fundo uma vez: a Conversa, a Início, o composer e a barra
  de abas não pintam o delas. Valores resolvidos no app (S2 1.8.2, 2026-11-20): claro, moldura
  `rgb(248, 248, 248)`, cartão `rgb(255, 255, 255)`, borda `rgb(218, 218, 218)`; escuro, moldura
  `rgb(27, 27, 27)`, cartão `rgb(34, 34, 34)`, borda `rgb(57, 57, 57)`. O cartão é mais claro que a
  moldura nos dois temas, como no protótipo (`#F4F4F5`/`#FFFFFF` e `#1A1B1E`/`#202125`). A gaveta de
  Logs, na moldura, não tem borda: o cartão e a margem de 8 px a separam. Fechada, a coluna tem
  largura 0 e o cartão fica a 8 px da borda da janela.
- **Barra de rolagem** (`global.css`, camada `base`): a do protótipo, em toda área que rola (sidebar,
  Conversa, Logs, Configurações, painel de Perguntas, barra de abas, tabela e bloco de código). É
  `::-webkit-scrollbar` de 14 px com o polegar recuado 5 px por uma borda transparente
  (`background-clip: padding-box`), então o que se vê é uma barra de 4 px com cantos totalmente
  arredondados; trilho e canto transparentes; sem as setas (`::-webkit-scrollbar-button` em
  `display: none`). **Não pode haver `scrollbar-width` nem `scrollbar-color`** em lugar nenhum
  (inclusive em `style()`): com qualquer um dos dois o Chromium/WebView2 ignora o `::-webkit-scrollbar`
  e, no Windows, desenha a barra nativa com as setas. O seletor global não usa o macro, então as cores
  são o valor medido do token (S2 1.8.2, 2026-11-20), em `light-dark()`: polegar `gray-400`, claro
  `rgb(198, 198, 198)` e escuro `rgb(68, 68, 68)`; hover `gray-500`, claro `rgb(143, 143, 143)` e escuro
  `rgb(109, 109, 109)`. O protótipo usa `#D6D6D9`/`#BFBFC4` e `#45464D`/`#5A5B63`; o token mais próximo
  do polegar claro seria o `gray-300`, mas ele é a cor da borda do cartão e some nela. O `color-scheme`
  do `<html>` e o do `Provider` escolhem o lado, também para o que sai em portal (Configurações,
  menus). A barra de abas, com 14 px de rolagem horizontal, perde 15 px da altura útil das abas só
  enquanto rola.
  A Conversa (ver **Conversa**), a Início, o composer (`InputDock`), a titlebar, a sidebar, a
  barra de abas, a barra de status, o Bloco de diagrama, a gaveta de Logs e as Configurações seguem o tema.
- **Titlebar** (`components/Titlebar.tsx`): à esquerda, recolher painel, menu principal (Logs,
  Configurações), marca, "Network Agent" e o chip do Workspace. À direita, o modelo e os botões da
  janela. A janela liga o
  `Windows.WebView2CompositionHosting` do v3, e a titlebar marca regiões `--wails-non-client-region`.
  O texto "Network Agent" e o espaçador do meio são `caption`: arrastam, e o duplo clique e o botão
  direito (menu de sistema) ficam com o Windows. Os vãos entre controles vizinhos não arrastam, e
  um controle nunca fica dentro de uma `caption`, porque lá ele não recebe clique. Só a titlebar
  arrasta; a barra de abas, não. Não há `--wails-draggable` nem `onDoubleClick`.
  **Plano B:** se a conferência ao vivo quebrar algo (popover do S2, foco, cursor, o hover dos
  botões), o corte desliga a opção, volta ao `--wails-draggable: drag` com `no-drag` nos controles
  e ao `onDoubleClick` → `Window.ToggleMaximise()`, e registra no `WAILS_V3.md` o que quebrou.
- **Chip do Workspace:** mostra o nome da pasta do Workspace em uso, com o caminho no `title` (sem
  pasta: "Nenhuma pasta"). O menu lista até 5 **Workspaces recentes** (`MenuSection` com
  `selectionMode="single"`, então `menuitemradio`: nome e, embaixo, o caminho), por uso mais recente,
  com o atual marcado, e depois, noutra seção, "Abrir outra pasta…" (o `PickWorkspace`, cujo diálogo
  nativo cancelado não muda nada). A lista fica no `config.json` (`recentWorkspaces`, ao lado do
  `workspace`) e é do Go: `Config.UseWorkspace` põe a pasta no topo, sem duplicata (caminho comparado
  sem diferenciar maiúsculas) e corta em 5, a cada escolha, pelo chip ou pelas Configurações, e o
  `Normalize` põe o Workspace em uso no topo mesmo num `config.json` de antes da lista. O front lê
  `ListRecentWorkspaces` (cada recente com `exists`, conferido no disco pelo Go) no boot, a cada troca
  e a cada abertura do menu, e troca com `PickRecentWorkspace(path)`, que grava e reconecta como o
  `PickWorkspace`. Um recente cuja pasta não existe mais aparece desabilitado, com "Pasta não
  encontrada" no lugar do caminho, e sai sozinho quando cinco outros entram na frente; não há remoção
  à mão. Se a pasta some entre abrir o menu e clicar, o Go recusa e não muda nada, e o menu reabre
  (controlado, `isOpen`) com o item desabilitado, sem toast nem card; qualquer outra falha cai no
  card de erro da sidebar. Enquanto um turno roda (`isWorking`), o chip fica desabilitado, com o
  motivo no `title`, e o botão "Trocar..." do campo das Configurações também, como o renomear e o
  apagar da sidebar.
- **Modelo e conexão:** na titlebar, antes dos botões da janela, aparece o texto "provedor ·
  modelo". Enquanto o modelo carrega, um `ProgressCircle size="S"` fica ao lado dele. Quando a
  conexão falha, aparece um `ActionButton` "Reconectar", com a causa no `title` de um `span` em volta
  (o `ActionButton` não aceita `title`). Sem modelo
  configurado, o texto é "Nenhum modelo configurado".
- **Botões da janela** (`WindowControls.tsx`): minimizar, maximizar/restaurar e fechar, com ícones
  próprios (ver **Ícones**). O glifo do maximizar acompanha o estado real da janela (snap,
  `Win+setas`, duplo clique) pelos eventos `common:WindowMaximise`, `WindowRestore` e
  `WindowUnMaximise`, relendo o `Window.IsMaximised()`. O hover é `:hover` do macro, e o do fechar é
  vermelho (`negative`). Os três são regiões `minimize`, `maximize` e `close`, e o Snap Layouts aparece no hover
  do maximizar. O Wails repassa o mouse dessas regiões à página, então o `onClick` e o `:hover`
  continuam sendo do React.
- **Sidebar ↔ faixa recolhida** (`Sidebar.tsx`, prop `expanded`): um estado só. Expandida, a
  sidebar tem 244 px; recolhida, vira a faixa de 56 px com o "Nova conversa". O botão de recolher
  painel, na titlebar, alterna as duas.
  O app sempre abre com a sidebar expandida, e a escolha não persiste. Não há atalho de teclado. A
  coluna anima a largura em 0,3 s, com a mesma curva da gaveta de Logs, e o conteúdo troca na hora.
  Não há modo sobreposto com scrim. Logs e Configurações saíram do rodapé da sidebar para o menu
  principal.
- **Lista de Conversas:** RAC `GridList` com uma `GridListSection` por grupo de data e
  `keyboardNavigationBehavior="tab"`: as setas andam entre as linhas, e o Tab entra nas ações
  (renomear, apagar), que aparecem no hover, no foco e na Conversa aberta. Clicar na linha abre a
  Conversa. Renomear troca o título por um `input` na linha (Enter grava, Esc ou clicar fora
  cancela; em falha o input fica aberto e o card de erro explica). O input para a propagação das
  teclas e do clique, para a grade não os tratar como navegação. Apagar abre um `AlertDialog`
  destrutivo, com o foco em Cancelar. Renomear e apagar ficam desabilitados durante um turno. O
  card de erro é um `InlineAlert` negativo com Tentar de novo e Dispensar.
- **Gaveta de Logs:** é a terceira coluna, sem scrim. Tem 360 px fixos, que caem para 300 px com a
  janela abaixo de 1080 px de largura (a `--logs-w` da coluna, que a gaveta também lê para não
  refluir). Não tem divisor para redimensionar: as linhas quebram. A coluna anima a largura em
  0,3 s ao abrir e fechar; a gaveta fica montada e, fechada, é `inert`. O conteúdo é a seção **Gaveta de Logs**.
- **Tamanho:** a janela abre em 1024×768, com mínimo de 900×600 (`MinWidth`/`MinHeight`). As regras
  do protótipo para larguras até 768 px não entraram. O
  breakpoint de 1080 px vale no shell. O que ele muda no modal de Configurações e nas sugestões é
  das seções dessas telas.

## Abas e Conversas

Uma **Aba** é a **Início** ou uma **Conversa** aberta (ver `CONTEXT.md`). A barra mora em
`components/TabBar.tsx`, dentro do `Tabs` do `App.tsx`, que é dono da seleção e dos painéis. A
barra tem 46 px e o indicador de 2 px desliza entre as abas (ver **Movimento**). Ele é um `span`
só, posicionado pela aba selecionada (`data-tab`) e não pelo `SelectionIndicator` do RAC: ele não
acompanhava uma aba criada e selecionada no mesmo commit, porque a aba só entra no DOM da coleção
depois do render. Um `MutationObserver` no `TabList` reposiciona o indicador quando ela entra.

- **Início:** é sempre a primeira aba e não tem "x". O primeiro envio por ela cria a Conversa e a
  abre numa aba nova, focada. A Início volta vazia. **Nova conversa**, na sidebar e na faixa
  recolhida, foca a Início e põe o cursor no composer dela, sem apagar o rascunho.
- **Abrir:** clicar numa Conversa da sidebar foca a aba dela se já houver uma. Se não houver, abre
  uma aba nova no fim da barra, já focada. Uma Conversa tem no máximo uma aba. O título da aba é o
  da Conversa e acompanha quando ela é renomeada. A Conversa da aba focada fica destacada na
  sidebar.
- **Sem limite de abas:** a barra rola na horizontal quando elas não cabem.
- **Fechar:** pelo "x" ou por `Ctrl+W`, que fecha a aba focada e não faz nada na Início. O "x" é um
  `span` dentro da aba, `aria-hidden` e fora da ordem de Tab; pelo teclado, quem fecha é o `Ctrl+W`.
  O foco vai para a vizinha da direita, ou para a da esquerda se não houver uma à direita. A Início
  é o piso. Fechar uma aba não apaga a Conversa. Apagar uma Conversa fecha a aba dela, com a mesma
  regra de foco. Alternar abas pelo teclado usa as setas do `TabList`.
- **Estado da aba:** enquanto a aba está aberta, ela guarda o rascunho do composer e a posição de
  rolagem. Fechar a aba descarta os dois. Todos os painéis ficam montados (`shouldForceMount`) e se
  sobrepõem na mesma caixa; o de fora de foco é `inert` e fica `visibility: hidden`, que, ao
  contrário de `display: none`, guarda a rolagem.
- **Um turno por vez no app inteiro** (o `busyMux` do Go). A aba da Conversa que está no turno
  mostra o `dot` só enquanto o turno roda. Nas outras abas e na Início, dá para digitar, mas o envio
  fica bloqueado, com a linha "Newton está respondendo em _‹título›_". O resultado do turno, a falha
  e o **Aviso** caem na bolha da aba dele, mesmo fora de foco. Nunca viram toast nem o card da
  sidebar. Fechar a aba com o turno rodando não cancela o turno: ele termina no Go e é gravado no
  disco, e reabrir a Conversa religa o stream. Renomear e apagar seguem bloqueados durante um turno.
- **Sem persistência:** o app sempre abre só com a Início. As Conversas recentes estão na sidebar.

## Conversa

O conteúdo de uma Aba de Conversa (`components/ChatArea.tsx`): a coluna que rola, com as bolhas, e o
trilho de Perguntas por cima. O Bloco de diagrama tem seção própria, logo abaixo.

- **Coluna:** 720 px no máximo, centralizada, com 44 px de respiro dos lados; é ela que rola, e o
  composer fica fora dela, embaixo. Rola até o fim a cada mensagem nova.
- **Usuário** (`MessageBubble.tsx`): bolha à direita, `accent-subtle`, até 80% da coluna. Embaixo, a
  hora e a linha de tokens do turno (entrada, saída, total e o contexto com a origem).
- **Agente:** sem bolha, na largura da coluna. No topo, o autor (o Subagente, ou "Agente"), só depois
  do stream. Depois, os tool calls (cada um um `Disclosure isQuiet` compacto: ícone da ação, rótulo,
  `CheckmarkCircle` e o resultado no painel), o Markdown (ou o "Trabalhando" com `ProgressCircle`
  enquanto não há texto), o status do reparo do diagrama, o Aviso e a meta (hora, duração, tokens). O
  texto novo não faz fade: as keyframes do protótipo não entram.
- **Erro do turno:** um `InlineAlert negative` no lugar da resposta: o título é o "o quê", e o
  conteúdo traz a causa (em JetBrains Mono, `pre-wrap`, porque o parse error do mermaid vem
  multilinha e o caret precisa alinhar) e o próximo passo.
- **Aviso:** um `InlineAlert notice` por mensagem ("Aviso" ou "Avisos"), com a lista dentro. Vale na
  resposta e no erro. Nunca vira toast.
- **Markdown** (`Markdown.tsx`): cada elemento tem uma classe do macro, pelo `components` do
  `react-markdown` (parágrafo, listas, citação, títulos, tabela com borda e rolagem, link, código
  inline, `hr`, caixa de seleção do GFM). O `<pre>` do react-markdown some (`pre: Fragment`): o
  bloco de código e o diagrama trazem o invólucro deles.
- **Bloco de código** (`CodeBlock.tsx`): a linguagem da fence no cabeçalho e o copiar (`ActionButton
  isQuiet` + `Copy` + `Tooltip` "Copiar"), sem realce. Copiar usa o `Clipboard` do Wails (e o
  `navigator.clipboard` fora do app) e dispara o toast "Código copiado." ou "Não foi possível
  copiar." (ver **Toast**).
- **Bloco de diagrama** (`Mermaid.tsx`): ver a seção **Bloco de diagrama**.
- **Trilho de Perguntas** (`QuestionNav.tsx`): a partir da segunda pergunta, uma barra de 18 px por
  pergunta na borda direita (`gray-400`). Mouse ou foco no trilho abre um painel de 260 px à
  esquerda dele, com o número e o texto de cada pergunta; clicar numa barra ou num item leva à
  mensagem, e o painel fechado tira os itens da ordem de Tab. O destaque é **cruzado**: o trilho
  guarda o índice da pergunta ativa (mouse ou foco, em qualquer das duas pontas) e acende as duas
  juntas, a barra (24 px e `accent`) e a linha (fundo `accent-subtle`, o token de seleção da
  Sidebar, e texto `neutral`). Sair do trilho (mouse ou foco) apaga tudo. Não há "fixar o painel" do
  protótipo.

## Bloco de diagrama

O fonte Mermaid de uma fence `mermaid`, renderizado em SVG inline pelo pele (síncrono, sem estado de
carga). Mora em `components/Mermaid.tsx`; o `Markdown.tsx` o monta.

- **Canvas:** a caixa do bloco (borda `gray-300`, raio `xl`, fundo `gray-25`, o mesmo desenho do
  bloco de código), com a barra em cima e o viewport embaixo. O viewport tem no máximo 520 px de
  altura, e o diagrama cabe na largura da coluna.
- **Barra:** à esquerda, um `ActionButtonGroup` (`isQuiet`, `size="S"`) com Menos zoom (`ZoomOut`), Mais
  zoom (`ZoomIn`), Redefinir zoom e posição (`Refresh`: 100% e centro) e Ajustar à tela
  (`ZoomFitToScreen`), e o valor do zoom em `output`. À direita, Ver código (um S2 `ToggleButton`
  com `Code`, que alterna o canvas pelo fonte) e, num `ActionButtonGroup`, Baixar SVG (`Download`).
  Cada botão tem `Tooltip`. Com o código à vista, os botões de zoom somem. O download continua
  com a interface do WebView2.
- **Zoom:** de 25% a 300%, por botão (passo de 1,2) ou pela roda do mouse (o listener é nativo e
  não passivo, para não rolar a Conversa), e arrastar move. Ajustar à tela escolhe o zoom que cabe o
  diagrama (no tamanho de layout) no viewport e o centraliza: um diagrama alto encolhe até caber; um
  baixo fica em 100%.
  - **Como o zoom é aplicado:** um `transform: translate() scale()` no `canvasInner`, **sem
    `will-change: transform`**. Com a dica, o WebView2 mostra o SVG ampliado como bitmap esticado
    (borrado, mais quanto maior o zoom); sem ela, ele refaz o raster na escala e o SVG fica vetor
    nítido de 25% a 300%, inclusive depois de arrastar e de Ajustar à tela. Não volte com o
    `will-change` por "fluidez": tirá-lo não custou nada medido (abaixo). Quem confere isso ao vivo não
    pode usar o print do CDP, que refaz o raster e esconde o borrado: ver
    [`WAILS_V3.md`](WAILS_V3.md), §4, CDP e depuração.
  - **Medido** (2026-11-20, `wails3 dev`, WebView2 158, DPR 1, diagrama de 1.181 elementos SVG, trace
    do CDP com entrada de roda e de arrasto a cada 16 ms, 3 rodadas por caso, com e sem a dica): a roda
    desenhou 47 a 53 quadros em 1,6 s nos dois (mediana do intervalo de 29 a 34 ms), e o arrasto 89 a
    107 quadros em 3,2 s; o tempo de `RasterTask` por rodada foi de 13 a 23 ms na roda nos dois e, no
    arrasto, de 43 ms com a dica e 17 ms sem. O trace não enxerga diferença de fluidez; o ritmo dos
    quadros é o do trace e do CDP, não o do app (conferência de olho pendente, ver o ticket).
- **Código:** o fonte numa `pre` em JetBrains Mono, sem cabeçalho. Durante o streaming o bloco
  ainda não renderiza, e o fonte sai no `CodeBlock` (linguagem `mermaid`).
- **Falha de render:** um `InlineAlert negative` ("O diagrama não renderizou", a causa do pele em
  JetBrains Mono, `pre-wrap`, e o próximo passo), com o fonte num `CodeBlock` embaixo. Os blocos
  vizinhos não são afetados.
- **Cores:** ver **Tema** → **Diagrama**.

## Gaveta de Logs

As mesmas linhas que o CLI mostra no terminal, porque a fonte é o sink de console: o Go escreve nos
dois destinos no mesmo `Handle`. Daí o que ela não mostra: nada de `source`, valor truncado em 512,
nada de TRACE. Quem quer isso lê o arquivo. Mora em `components/LogDrawer.tsx`; a coluna, com as
larguras e a animação, é do **Shell e janela**.

- **Cabeçalho:** "LOGS", o contador de linhas (uma pílula `accent-subtle`) e o fechar (S2
  `ActionButton isQuiet` com `Close` e `Tooltip`).
- **Linhas:** em JetBrains Mono 12 px (`code-xs`), sem scroll lateral: elas quebram
  (`pre-wrap`, `overflow-wrap: anywhere`). O timestamp sai em `neutral-subdued`, e a cor do resto
  da linha só aparece no nível, lido do token `[INFO ]` (colunas 24 a 31 da linha do console):
  `DEBUG` em `cyan-900`, `INFO` em `positive-900`, `WARN` em `notice-900` e `ERROR` em
  `negative`. Uma mensagem que termina em `
` rende uma linha em branco, como no terminal.
- **Transporte:** pull de `GetLogLines` a cada 250 ms, só com a gaveta aberta (o writer do Go não
  conhece o Wails e não emite evento). Acompanha o fim só se a rolagem já estava no fim.
- **Rodapé:** o `Link` S2 "Copiar o caminho do arquivo" (`GetLogPath`) copia pelo `copyText` do
  `CodeBlock` e dispara o toast "Caminho copiado." ou "Não foi possível copiar." (ver **Toast**).
  Sem caminho (execução sem arquivo de log, ou fora do app), o rodapé diz "Esta execução não abriu
  arquivo de log." e não há `Link`. O rótulo nunca troca.

## Configurações

O modal de Configurações (`components/SettingsModal.tsx`, com os Modelos em `ModelsSection.tsx`), aberto
pelo menu principal da titlebar ou pelo alerta de envio bloqueado. É um `DialogContainer` com um
`Dialog size="XL"` (960 px, no máximo 90% da janela) montado só com `open`; o título é "Configurações".
`isDismissible` vem da prop `dismissible`: com a configuração obrigatória pendente (nenhum provedor com
modelo) não há o fechar (`Descartar`, o texto do S2 em pt-BR), nem Esc, nem clique fora, e o modal abre
direto em Provedores, com um `InlineAlert informative` "Configuração necessária". Fora disso abre em
Workspace. O erro de qualquer chamada é um `InlineAlert negative` "Não foi possível concluir", acima
das abas e com a causa do Go; não vira toast.

- **Navegação:** o menu vertical do protótipo (`.cfg-nav`), e não o `Tabs` do S2 (exceção pedida pelo
  usuário: aqui o protótipo vence o Spectrum). É o `Tabs`/`TabList`/`Tab`/`TabPanel` do RAC com o
  visual do macro, então a semântica é a de abas (`tablist`, `aria-orientation`, setas, tabindex
  móvel, `aria-selected`, `tabpanel` ligado à aba), a mesma de antes. Três itens, cada um com ícone S2
  (`XS`) e rótulo: **Workspace** (`Folder`), **Provedores** (`ListBulleted`) e **Aparência**
  (`BrightnessContrast`). O menu tem 200 px, com 16 px de respiro e uma linha de 1 px (`gray-200`) que o
  separa do conteúdo, a 24 px dele. O item tem raio `lg` (10 px) e texto `neutral-subdued`; o hover dá
  `gray-100` e texto `neutral`; o ativo, `accent-subtle`, texto `accent` e peso 600; o foco de teclado
  é o anel `focus-ring` de 2 px por dentro. Com a janela abaixo de 1080 px o `orientation` vira
  `horizontal` (um `matchMedia`): o menu é uma linha, com a linha embaixo, e a lista de provedores
  empilha sobre o formulário (um `@media` do macro, lista com 40% da altura).
- **Rolagem e borrado:** o corpo do modal tem altura fixa, `min(496px, 100vh - 224px)` (o
  `.modal-body` do protótipo, 540 px, menos o respiro que o `Dialog` do S2 já dá), e o cabeçalho (título
  e fechar) fica fora dele. O menu, a lista de provedores, o formulário de edição e cada painel
  rolam sozinhos, cada um com o próprio `overflow-y`. O app por trás fica borrado (4 px, como o
  `.overlay` do protótipo) além do véu do S2: o `Tabs` leva a classe `blurBehind`, um `css()` do macro
  que vira `body > :has(.x) { backdrop-filter: blur(4px) }`, porque o overlay do `Modal` do S2 é filho
  direto do `body` e não aceita `className`. O borrado acompanha a opacidade do véu na entrada e na
  saída, e o `Dialog` do S2 segue com o foco preso, o Esc e o rótulo.
- **Workspace:** um `TextField` somente leitura com a pasta (placeholder "Nenhuma pasta escolhida — o
  agente não grava arquivos.") e o `Button` "Trocar..." (ou "Escolher..." sem pasta), que chama o
  `PickWorkspace`. Durante um turno (`workspaceLocked`) o botão fica desabilitado e a linha embaixo
  diz por quê ("Trocar de Workspace fica bloqueado enquanto o agente responde."): o S2 não põe
  `title` no botão.
- **Provedores:** à esquerda, "PROVEDORES", o `ActionButton` com `Add` ("Novo provedor") e um
  `ListView selectionMode="single" selectionStyle="highlight"` com nome e "N modelos" (mais "· em
  uso" no selecionado do app); criar mostra uma linha "Novo provedor". À direita o formulário
  (`TextField` Nome, URL base, Chave de API com `type="password"`), "Usar este" (se não for o em uso) e
  `Button variant="negative" fillStyle="outline"` "Excluir" (sem confirmação, como antes), e o
  `Button variant="accent"` "Salvar alterações" ou "Criar provedor" (com "Cancelar" ao criar). A
  coluna do formulário rola. Sem provedor, o formulário diz "Selecione um provedor à esquerda ou crie
  um novo."
- **Modelos** (`ModelsSection.tsx`, dentro do formulário de um provedor já criado; ao criar, a dica
  "Crie o provedor para poder cadastrar os modelos dele."): "Novo modelo" (`Button` secundário) e um
  `ListView isQuiet selectionMode="none"` com uma linha por modelo (nome, ou o id; embaixo "id … ·
  contexto …"), o `Badge variant="accent" fillStyle="subtle"` "padrão" no modelo padrão e um
  `ActionButtonGroup` com Tornar padrão (`Star`, só nos que não são), Editar (`Edit`) e Excluir
  (`Delete`), cada um com `Tooltip`. A lista é virtualizada, então tem altura fixa: 52 px por linha
  (`ROW`). O formulário do modelo (ID, Nome de exibição e o `NumberField` do contexto máximo, sem
  separador de milhar) fica embaixo, com Cancelar e "Criar modelo" ou "Salvar modelo".
- **Aparência:** "Tema" e um `SelectBoxGroup` (seleção única, sem esvaziar) com os cartões Claro,
  Escuro e Sistema e a nota "O tema "Sistema" acompanha a preferência do sistema operacional.". A
  prévia é uma janelinha de 96×62 px (barra lateral e duas linhas), desenhada com um `Provider`
  aninhado em cada metade: `colorScheme="light"` ou `"dark"` pinta com os tokens do S2, sem hex, e
  vale qualquer que seja o tema em uso; o Sistema parte uma metade de cada. O S2 só põe na área
  `illustration` do cartão o que passa pelo `IllustrationContext`, então a prévia leva `grid-area:
  illustration` por `css()`. O comportamento da escolha está em **Tema**.
- **Estado:** o modal guarda a configuração e o rascunho. Fechar e reabrir recarrega o `GetConfig`; o
  Workspace só muda pelo `PickWorkspace`.

## Componentes

Cada elemento é um **controle S2** ou uma **superfície própria** (RAC ou HTML, estilizada pelo macro
`style` com os tokens do S2). O único CSS global é o `src/global.css`, com os `@font-face`, a ordem
das camadas e a regra de movimento reduzido.

`App.css` e `Settings.css` não existem mais, e nenhuma folha de superfícies tomou o lugar deles. O
Markdown estiliza cada elemento pelo `components` do `react-markdown`, com uma classe do macro por
elemento.

| Elemento | Origem | Hoje | Arquivo |
|---|---|---|---|
| Titlebar (marca, região de arrasto) | própria | **pronto** | `components/Titlebar.tsx` |
| Recolher painel | S2 `ActionButton isQuiet` + ícone próprio | **pronto** | `Titlebar.tsx` |
| Menu principal (Logs, Configurações) | S2 `MenuTrigger` + `Menu` | **pronto** | `Titlebar.tsx` |
| Chip do Workspace | S2 `MenuTrigger` + `Menu` (gatilho: `ActionButton isQuiet`) | **pronto**, com os recentes | `Titlebar.tsx` |
| Modelo e conexão | própria (texto "provedor · modelo") + S2 `ProgressCircle size="S"` (carregando) + S2 `ActionButton` "Reconectar" (falha) | **pronto** | `Titlebar.tsx` |
| Botões da janela | própria (o hover vermelho do fechar não sai do S2) | **pronto** | `WindowControls.tsx` |
| Faixa recolhida | `nav` própria + S2 `Button variant="accent"` só com ícone | **pronto** | `Sidebar.tsx` |
| Nova conversa | S2 `Button variant="accent"` | **pronto** | `Sidebar.tsx` |
| Lista de Conversas | própria em RAC `GridList` + `GridListSection` (grupos por data); ações em S2 `ActionButton isQuiet`; renomear inline | **pronto** | `Sidebar.tsx` |
| Card de erro da sidebar | S2 `InlineAlert variant="negative"` + `ActionButton` (Tentar de novo, Dispensar) | **pronto** | `Sidebar.tsx` |
| Abas | própria em RAC `Tabs`/`TabList`/`Tab`: o "x" dentro da aba, fora da ordem de Tab, e fechar também por teclado; rola na horizontal quando não cabe | **pronto** | `components/TabBar.tsx` |
| Início: título e subtítulo | própria | **pronto** | `components/Home.tsx` |
| Sugestões | própria, cartão em RAC `Button` | **pronto** | `Home.tsx` |
| Composer (Início e Conversa) | própria: RAC `TextArea` + botão de enviar próprio | **pronto** | `InputDock.tsx` |
| Markdown | própria: uma classe do macro por elemento, pelo `components` do `react-markdown` | **pronto** | `Markdown.tsx` |
| Bloco de código | própria: cabeçalho com a linguagem da fence (vazio se não houver) + S2 `ActionButton isQuiet` com `Copy` e `Tooltip`; sem realce de sintaxe | **pronto** | `components/CodeBlock.tsx` |
| Bloco de diagrama: canvas | própria | **pronto** | `Mermaid.tsx` |
| Bloco de diagrama: barra | S2 `ActionButtonGroup` + `ActionButton` (`ZoomOut`, `ZoomIn`, `Refresh`, `ZoomFitToScreen`, `Download`) e `ToggleButton` (`Code`) | **pronto** | `Mermaid.tsx` |
| Bolha do usuário | própria | **pronto** | `MessageBubble.tsx` |
| Meta da mensagem | própria | **pronto** | `MessageBubble.tsx` |
| Tool call, "Mostrar resposta original" | S2 `Disclosure isQuiet`, densidade compacta | **pronto** | `MessageBubble.tsx` |
| Pensando | S2 `ProgressCircle size="S" isIndeterminate` | **pronto** | `MessageBubble.tsx` |
| Erro do turno | S2 `InlineAlert variant="negative"` | **pronto** | `MessageBubble.tsx` |
| Avisos | um S2 `InlineAlert variant="notice"` por mensagem, com a lista dentro; na bolha, nunca num toast | **pronto** | `MessageBubble.tsx` |
| Status do reparo do diagrama | texto com `role="status"` | **pronto** | `MessageBubble.tsx` |
| Trilho de Perguntas | própria | **pronto** | `QuestionNav.tsx` |
| Toast | S2 `ToastContainer placement="bottom end"` + `ToastQueue`; uma frase só, sem subtítulo | **pronto** (o `ToastContainer` está no `App.tsx`; o `CodeBlock` e a gaveta de Logs o disparam) | `App.tsx` |
| Gaveta de Logs | `aside` própria + S2 `ActionButton` (fechar) + S2 `Link` (copiar o caminho) | **pronto** | `LogDrawer.tsx` |
| Barra de status | própria | **pronto** | `components/StatusBar.tsx` |
| Configurações: modal | S2 `DialogContainer` + `Dialog size="XL"` | **pronto** | `SettingsModal.tsx` |
| Configurações: navegação | própria em RAC `Tabs orientation="vertical"` (horizontal abaixo de 1080 px), o menu `.cfg-nav` do protótipo; exceção: aqui o protótipo vence o S2 `Tabs` | **pronto** | `SettingsModal.tsx` |
| Configurações: borrado sob o modal | própria: `css()` do macro com `body > :has(&) { backdrop-filter: blur(4px) }` sobre o véu do S2 | **pronto** | `SettingsModal.tsx` |
| Campos (Workspace, provedor, modelo) | S2 `TextField` (`type="password"` na chave), `NumberField` (contexto) + `Button` | **pronto** | `SettingsModal.tsx`, `ModelsSection.tsx` |
| Lista de provedores | S2 `ListView selectionMode="single"` (`selectionStyle="highlight"`) | **pronto** | `SettingsModal.tsx` |
| Modelos | S2 `ListView` + `ActionButtonGroup` (padrão, editar, apagar) + `Badge` "padrão" | **pronto** | `ModelsSection.tsx` |
| Aparência (tema) | S2 `SelectBoxGroup`, cartões com a prévia (um `Provider` aninhado por metade) | **pronto** | `SettingsModal.tsx`, `theme.ts` |
| Salvar / Excluir | S2 `Button variant="accent"` / `variant="negative"` | **pronto** | `SettingsModal.tsx` |
| Apagar Conversa | S2 `AlertDialog variant="destructive"` | **pronto** | `Sidebar.tsx` |
| Envio bloqueado | S2 `AlertDialog` | **pronto** | `App.tsx` |

### Ícones

Os do S2 (`@react-spectrum/s2/icons/<Nome>`), em todo o app. Quatro são próprios, em `src/icons/`,
desenhados a partir do SVG do protótipo: **painel lateral** (`PainelLateral`, recolher painel) e os
três da janela, **minimizar**, **maximizar** e **restaurar** (`MinimizarJanela`, `MaximizarJanela`,
`RestaurarJanela`): o `Minimize`/`Maximize` do S2 são setas de encolher e expandir, não os glifos de
janela. (Três saem do SVG do protótipo e um da JetBrains, ver abaixo.) O SVG fonte (20×20, fill `var(--iconPrimary, #222)`) fica em `src/icons/svg/`, e o `.tsx` sai
do `@react-spectrum/s2-icon-builder` 1.1.2, rodado fora do projeto, sem virar dependência:
`npx transform-icons -i <svg> -o <pasta>` (o cabeçalho do arquivo diz de onde veio). Ícone de ação
do tool call sem equivalente no S2 segue o mesmo caminho.

**Origem dos quatro.** A fonte preferida é o [JetBrains icons](https://github.com/JetBrains/icons)
(`@jetbrains/icons` 5.25.0, Apache-2.0, variante `-20px`). Das 2.333 entradas do pacote, só uma
serve: `remove-20px.svg`, um traço horizontal, é o **`MinimizarJanela`** (o fill
`currentColor` virou `var(--iconPrimary, #222)`; o cabeçalho do `.tsx` diz o arquivo e a versão).
Ele tem 18 de largura contra os 12 do quadrado do maximizar, e por isso o traço fica um pouco mais
largo que os vizinhos. `PainelLateral`, `MaximizarJanela` e `RestaurarJanela` **continuam
desenhados à mão**: o pacote não tem painel lateral nem glifo de janela (os `collapse`/`expand` são
quadrados com menos/mais dentro, e o `new-window` é uma seta de link externo). A licença está em
`src/icons/svg/LICENSE-JetBrains-icons.txt`; o pacote não traz NOTICE.

O `@phosphor-icons/react` não é mais dependência: saiu do `package.json` e do `package-lock.json`, e
nenhum arquivo o importa.

### Fonte

Source Sans 3 (interface, normal e itálico) e JetBrains Mono (bloco de código, Logs), em woff2
variável embutido em `src/assets/fonts`, só o subconjunto latino (cobre o pt-BR), com as licenças
OFL ao lado (`OFL-SourceSans3.txt`, `OFL-JetBrainsMono.txt`). Os arquivos vêm dos pacotes
`@fontsource-variable/source-sans-3` 5.3.1 e `@fontsource-variable/jetbrains-mono` 5.3.0, copiados
à mão, sem dependência. Os `@font-face` ficam no `global.css`. Os controles S2 e as superfícies com
`fontFamily: 'sans'` recebem a família por `--s2-font-family-sans: 'Source Sans 3'` no `:root`. A variável não é documentada;
se sumir, a queda é a Segoe UI. A JetBrains Mono entra pelo escape-hatch `css()` do macro, porque o
`fontFamily` dele só aceita os tokens, e o token `code` é fixo. Usam a JetBrains Mono o bloco de
código, as linhas da gaveta de Logs, o código inline e a causa do erro do turno, pela classe `mono` que o `CodeBlock.tsx` exporta (o
tamanho e a altura de linha vão pelo macro, com `fontSize: 'code-sm'`, sem `font`, para não disputar o
`font-family`).

### Movimento

O da Adobe. Os controles S2 animam com as próprias transições. A superfície própria imita o
componente S2 mais próximo, com os tokens de transição do macro (`transition`, easings `in`, `out`,
`default`):

- o indicador da aba desliza com `translate`/`width` (0,2 s), como o do S2 `Tabs`;
- a troca de aba não anima o conteúdo, porque o `TabPanel` do S2 não anima;
- a gaveta de Logs anima a largura como o `SideNavPanel` (0,3 s);
- o clique em sugestão, item da lista e aba comprime como os botões S2.

As keyframes do protótipo (`od-*`, `toast-in`, `toast-out`) não entram. Com movimento reduzido, uma
regra `@media (prefers-reduced-motion: reduce)` zera `transition` e `animation` numa `@layer`
declarada depois da `_` do S2 (`reduced-motion`, no `global.css`), que vence o S2 sem `!important`.
Ela cobre Modal, Popover, Menu e Tooltip, que não têm a regra no CSS do
S2.

### Idioma

Só pt-BR: `<Provider locale="pt-BR">` e `@react-aria/optimize-locales-plugin` com
`locales: ['pt-BR']` (`enforce: 'pre'`) no `vite.config.ts`. Os textos do próprio S2 saem em
português qualquer que seja o idioma do Windows, e os outros 33 locales saem do bundle (−27 KB gz,
medido).

## Início, barra de status e toast

**Início** (`components/Home.tsx`): o título "O que vamos fazer hoje?",
o subtítulo "Descreva a tarefa. O Newton monta, ajusta, desenha ou explica configurações FRR.", o
composer e quatro sugestões. As sugestões são uma lista fixa no próprio `Home.tsx`, uma por
capacidade do Newton, nesta ordem:

1. "Montar um eBGP entre o AS 65001 e o AS 65002"
2. "Adicionar BFD ao BGP do frr.conf"
3. "Desenhar um iBGP com route reflector"
4. "Explicar como o OSPF elege o DR"

Clicar numa sugestão põe o texto dela no composer da Início, no lugar do rascunho, e foca o campo
com o cursor no fim. Ela não envia, porque um envio do Builder pode substituir o `frr.conf` do
Workspace. As sugestões nunca ficam desabilitadas. Quando o envio está bloqueado (turno em outra
aba, nenhum provedor), quem avisa é o composer. A Início não tem "Limpar conversa". O composer é o
`InputDock` compartilhado com a Conversa, e o `App` o preenche pelo `InputDockHandle` (`fill`). A
Início monta as sugestões em uma coluna abaixo de 1080 px, como o protótipo.

**Barra de status** (`components/StatusBar.tsx`): uma linha de 30 px no pé da janela, na largura
toda, que mostra a Aba focada, não a do turno em voo.

- **À esquerda:** a hora da última resposta do Newton na Conversa, no formato da meta da mensagem e
  com o tooltip "Último turno". Antes da primeira resposta, ela não aparece.
- **À direita:** os itens que estavam no rodapé do `InputDock.tsx`: tokens de entrada, de saída, o
  total e o contexto utilizado/janela com a origem (`contextOrigin`). Os totais somam o `usage` das
  mensagens da Conversa; a janela vem da sondagem do modelo ativo, quando existe, senão da última
  medida no disco. Sem janela sondada nem medida, o item de contexto não aparece.
- **Durante um turno:** o horário e os totais continuam os do turno anterior até ele terminar.
- **Na Início:** a mesma barra, com os totais zerados, o contexto em 0 sobre a janela do modelo
  ativo e sem horário.

**Toast:** o `ToastQueue` do S2, no `ToastContainer` do `App.tsx`. Ele só confirma uma ação que não deixa rastro visível ou
avisa que ela falhou:

| Ação | Variante | Frase |
|---|---|---|
| Copiar o bloco de código (`CodeBlock.tsx`) | `positive` | "Código copiado." |
| Copiar o caminho do log (`LogDrawer.tsx`) | `positive` | "Caminho copiado." |
| Uma dessas cópias falhar | `negative` | "Não foi possível copiar." |

O `Link` da gaveta de Logs fica sempre com o rótulo "Copiar o caminho do arquivo": o toast é a
confirmação, sem troca de rótulo. Mais nada vira toast:

- Salvar nas Configurações: o próprio dialog mostra o resultado.
- Baixar o diagrama: o WebView2 mostra o download (medido no v3: abre o painel de downloads do
  Edge e grava em `Downloads`).
- Erro com lugar próprio. O **Aviso**, a falha e o resultado do turno ficam na bolha (ver
  [Abas e Conversas](#abas-e-conversas)). O erro das Configurações fica inline no dialog. O da lista
  de Conversas fica no card da sidebar.

## Verificação

- `npm run build` e `npm test` verdes em `cmd/ui/frontend`.
- `gofmt -l .`, `go vet ./...` e `go build ./...` verdes na raiz.
- Ao vivo no dev: Início, Conversa com diagrama e bloco de código, Logs e Configurações, conferidas
  contra o protótipo **nos dois temas**.
- **Dev e build:** o projeto segue o modelo do Wails v3, com a CLI `wails3` 3.1.0, e os comandos
  rodam de dentro de `cmd/ui` (ver [`cmd/ui/README.md`](../cmd/ui/README.md)). O dev é o
  `wails3 dev`: ele sobe o Vite na 9245 (`WAILS_VITE_PORT`, `strictPort`), o front recarrega sozinho
  e, a cada `.go` salvo, ele regenera as bindings, recompila e reabre a janela. Fechar a janela
  encerra o `wails3 dev`. O build é o `wails3 build`, e o `.exe`, com o ícone, sai em
  `cmd/ui/bin/net_agent.exe`. Os dois seguem o `build/config.yml` e os Taskfiles gerados
  (`cmd/ui/Taskfile.yml` e `cmd/ui/build/`). O `build/go.mod` isola do `go build ./...` da raiz os
  `.go` que o gerador cria para iOS e Android. O `go mod tidy` do Taskfile não muda o `go.mod`.
- **Porta de depuração:** com `NETWORK_AGENT_CDP_PORT=9333` no ambiente do app, a janela abre o
  CDP em `localhost:9333` (`/json/list` dá o `webSocketDebuggerUrl` da página). O DevTools da
  janela também abre à mão, em build sem a tag `production`.
