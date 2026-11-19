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
  superfícies sem equivalente (composer, trilho de Perguntas, diagrama, gaveta de Logs).
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
  `build.target: ['es2022']`, `cssMinify: 'lightningcss'` e um `manualChunks` que junta o CSS do S2
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
- **CSS global:** `src/global.css`, importado primeiro no `main.tsx`, é o único CSS global do app.
  Ele declara a ordem das camadas, `@layer legacy, _, reduced-motion;`, e traz os `@font-face`
  (**Fonte**) e a regra de movimento reduzido (**Movimento**). No build, o `index-*.css` (legado e
  global) carrega antes do `s2-styles-*.css`, e a ordem final é `legacy`, `_`, `reduced-motion`.
- **CSS legado:** `App.css` e `Settings.css` ficam inteiros dentro de `@layer legacy { … }`, abaixo da
  camada `_` do S2 pela ordem do `global.css`. Assim, o CSS global antigo (`*`, `body`, `button`) não
  vence os controles do S2. Fica assim até o visual novo apagar os dois arquivos.
  - Conflito conhecido: popover do S2 (Picker, Menu) aberto dentro de um overlay legado com
    `z-index` (o modal de Configurações, `.settings-scrim`, `z-index: 100`) fica invisível atrás
    dele. Controle S2 com popover só entra em superfície legada se essa superfície já tiver saído.
  - As superfícies legadas seguem com a pilha de sistema do `App.css` (Segoe UI no Windows). Só os
    controles S2 e as superfícies novas recebem a Source Sans 3.
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
  - Cancelar o seletor de pasta não é erro: o `PickWorkspace` trata o `"cancelled by user"` do v3
    como o `""` do v2.
- **Bundle (Vite 8):** com o Shell e as Abas (Titlebar, Sidebar, TabBar, StatusBar, botões da
  janela) e só o pt-BR, o JS tem 1.535 KB (gzip 505 KB) e o chunk `s2-styles` 59 KB (gzip 10 KB).
  Só com `Provider` + `ToastContainer` eram 1.327 KB (gzip 446 KB) e 44 KB (gzip 7,3 KB); sem o
  filtro de locales, 1.427 KB (gzip 469 KB). As três fontes somam 98 KB. Medição anterior em
  [`REACT_SPECTRUM_S2.md`](REACT_SPECTRUM_S2.md#9-peso).

## Tema

O `Provider` (`main.tsx`) não recebe `colorScheme`, então hoje o app está sempre no Sistema. O
**Seletor** e o **Diagrama** ainda faltam.

- **Três escolhas, padrão Sistema.** Claro e Escuro passam `colorScheme` ao `Provider`. Sistema,
  que vale também quando nada foi salvo, omite o prop, e então o app acompanha o Windows mesmo com a
  janela aberta. Não há paleta própria: o escuro é o do S2.
- **Seletor** (_Decidido, a implementar_): em Configurações → Aparência, um `SelectBoxGroup` com cartões de prévia. A troca
  vale ao clicar, sem botão Salvar.
- **Persistência:** a escolha vai para o `localStorage` do WebView2, na chave `theme` (`'light'` ou
  `'dark'`; ausente é o Sistema). Para que o `wails3 dev`, o `go run` e o binário construído
  enxerguem a mesma escolha, o `main.go` fixa `application.Options.Windows.WebviewUserDataPath` em
  `~/.network_agent/webview2` (ao lado do `config.json`). Sem ele, o WebView2 usa
  `%AppData%\<nome do exe>`, uma pasta por forma de rodar.
- **Sem piscar ao abrir:** a janela nasce com `Hidden: true`, e o `main.go` chama `window.Show()` no
  `events.Windows.WebViewNavigationCompleted`. Antes disso, um script inline no `<head>` do
  `index.html` lê a chave `theme` e pinta o fundo do `<html>` com `background-color: Canvas` sob
  `color-scheme` `light` ou `dark`, ou `light dark` no Sistema. A janela não tem mais
  `BackgroundColour`. Enquanto o visual legado existir, o `body` do `App.css` pinta o vidro escuro
  por cima nos dois temas.
- **Diagrama** (_Decidido, a implementar_): o SVG na tela usa `var(--pele-*)`, então troca de tema sozinho, sem renderizar de
  novo. O SVG baixado leva o tema em uso, com as cores **resolvidas** (rgb/hex) no `<style>`
  injetado: um `light-dark(…)` dos tokens do S2 não pode ir para o arquivo, porque fora do app ele
  sempre resolveria para o claro ou nem renderizaria.
- **Bloco de código:** sem realce de sintaxe, então as cores `--code-*` do protótipo não entram.

## Shell e janela

A janela é o app: o shell não desenha a moldura do protótipo (padding, gradiente, raio, sombra), e
não há brilhos de fundo. Canto arredondado e sombra ficam com o Windows 11, que os dá à janela
frameless. As opções de janela ficam no `cmd/ui/main.go`.

- **Grade** (`App.tsx`, macro `style`): a titlebar (48 px) em cima, na largura toda, as três colunas
  no meio (sidebar ou faixa, conteúdo e gaveta de Logs) e a barra de status embaixo, na largura
  toda. A coluna do conteúdo é o `Tabs` do RAC: a barra de abas em cima e um painel por aba embaixo.
  O conteúdo das abas ainda é o legado (`ChatArea` + `InputDock`), pintado pelo vidro escuro do
  `App.css` nos dois temas; a titlebar, a sidebar, a barra de abas e a barra de status já seguem o
  tema.
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
  pasta: "Nenhuma pasta"). **Hoje** o menu tem um item só, "Abrir outra pasta…", que chama o mesmo
  `PickWorkspace` das Configurações e já grava; cancelar o diálogo não muda nada, e uma falha cai no
  card de erro da sidebar. _Decidido, a implementar_ (corte _Workspaces recentes_): o menu lista até 5 **Workspaces recentes** (`menuitemradio`: nome e, embaixo, o caminho), por uso
  mais recente, com o atual marcado, e depois um separador e "Abrir outra pasta…" (o
  `PickWorkspace`). A lista fica no `config.json` (`recentWorkspaces`, ao lado do `workspace`), e o Go
  a atualiza a cada escolha de Workspace, pelo chip ou pelas Configurações. Um recente cuja pasta
  não existe mais aparece desabilitado, com "Pasta não encontrada", e sai sozinho quando cinco outros
  entram na frente; não há remoção à mão. Se a pasta some entre abrir o menu e clicar, o Go recusa,
  o Workspace não muda e o menu reabre com o item desabilitado, sem toast. Enquanto um turno roda,
  trocar de Workspace fica bloqueado no chip e no campo das Configurações, com o motivo no `title`,
  como o renomear e o apagar.
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
  0,3 s ao abrir e fechar; a gaveta fica montada e, fechada, é `inert`. O conteúdo dela ainda é o
  legado.
- **Tamanho:** a janela abre em 1024×768, com mínimo de 900×600 (`MinWidth`/`MinHeight`). As regras
  do `App.css` para larguras até 768 px saíram. O
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

## Componentes

Cada elemento é um **controle S2** ou uma **superfície própria** (RAC ou HTML, estilizada pelo macro
`style` com os tokens do S2). O único CSS global é o `src/global.css`, com os `@font-face`, a ordem
das camadas e a regra de movimento reduzido.

_Decidido, a implementar_ nas linhas da tabela abaixo cuja coluna **Hoje** ainda aponta o legado; as
linhas com **pronto** já estão no código. `App.css` e `Settings.css` são apagados
sem uma folha de superfícies no lugar. O Markdown estiliza cada elemento pelo
`components` do `react-markdown`, com uma classe do macro por elemento.

| Elemento | Origem | Hoje | Arquivo |
|---|---|---|---|
| Titlebar (marca, região de arrasto) | própria | **pronto** | `components/Titlebar.tsx` |
| Recolher painel | S2 `ActionButton isQuiet` + ícone próprio | **pronto** | `Titlebar.tsx` |
| Menu principal (Logs, Configurações) | S2 `MenuTrigger` + `Menu` | **pronto** | `Titlebar.tsx` |
| Chip do Workspace | S2 `MenuTrigger` + `Menu` (gatilho: `ActionButton isQuiet`) | **pronto**, com um item só | `Titlebar.tsx` |
| Modelo e conexão | própria (texto "provedor · modelo") + S2 `ProgressCircle size="S"` (carregando) + S2 `ActionButton` "Reconectar" (falha) | **pronto** | `Titlebar.tsx` |
| Botões da janela | própria (o hover vermelho do fechar não sai do S2) | **pronto** | `WindowControls.tsx` |
| Faixa recolhida | `nav` própria + S2 `Button variant="accent"` só com ícone | **pronto** | `Sidebar.tsx` |
| Nova conversa | S2 `Button variant="accent"` | **pronto** | `Sidebar.tsx` |
| Lista de Conversas | própria em RAC `GridList` + `GridListSection` (grupos por data); ações em S2 `ActionButton isQuiet`; renomear inline | **pronto** | `Sidebar.tsx` |
| Card de erro da sidebar | S2 `InlineAlert variant="negative"` + `ActionButton` (Tentar de novo, Dispensar) | **pronto** | `Sidebar.tsx` |
| Abas | própria em RAC `Tabs`/`TabList`/`Tab`: o "x" dentro da aba, fora da ordem de Tab, e fechar também por teclado; rola na horizontal quando não cabe | **pronto** | `components/TabBar.tsx` |
| Início: título e subtítulo | própria | estado vazio do `ChatArea.tsx` | `components/Home.tsx` |
| Sugestões | própria, cartão em RAC `Button` | chips do `ChatArea.tsx` | `Home.tsx` |
| Composer (Início e Conversa) | própria: RAC `TextArea` + botão de enviar próprio | `InputDock.tsx` | `InputDock.tsx` |
| Markdown | própria | `Markdown.tsx` | `Markdown.tsx` |
| Bloco de código | própria: cabeçalho com a linguagem da fence (vazio se não houver) + S2 `ActionButton isQuiet` com `Copy` e `Tooltip`; sem realce de sintaxe | `<pre>` do `Markdown.tsx` | `components/CodeBlock.tsx` |
| Bloco de diagrama: canvas | própria | `Mermaid.tsx` | `Mermaid.tsx` |
| Bloco de diagrama: barra | S2 `ActionButtonGroup` + `ActionButton` (`ZoomOut`, `ZoomIn`, `Refresh`, `ZoomFitToScreen`, `Code`, `Download`) | `Mermaid.tsx` | `Mermaid.tsx` |
| Bolha do usuário | própria | `MessageBubble.tsx` | `MessageBubble.tsx` |
| Meta da mensagem | própria | rodapé do `MessageBubble.tsx` | `MessageBubble.tsx` |
| Tool call, "Mostrar resposta original" | S2 `Disclosure isQuiet`, densidade compacta | `<details>` do `MessageBubble.tsx` | `MessageBubble.tsx` |
| Pensando | S2 `ProgressCircle size="S" isIndeterminate` | `ThinkingIndicator` | `MessageBubble.tsx` |
| Erro do turno | S2 `InlineAlert variant="negative"` | `.bubble--error` | `MessageBubble.tsx` |
| Avisos | um S2 `InlineAlert variant="notice"` por mensagem, com a lista dentro; na bolha, nunca num toast | `WarningList` | `MessageBubble.tsx` |
| Status do reparo do diagrama | texto com `role="status"` | `MessageBubble.tsx` | `MessageBubble.tsx` |
| Trilho de Perguntas | própria | `QuestionNav.tsx` | `QuestionNav.tsx` |
| Toast | S2 `ToastContainer placement="bottom end"` + `ToastQueue`; uma frase só, sem subtítulo | o `ToastContainer` já está no `App.tsx`, sem nada que o dispare | `App.tsx` |
| Gaveta de Logs | `aside` própria + S2 `ActionButton` (fechar) + S2 `Link` (copiar o caminho) | `LogDrawer.tsx` | `LogDrawer.tsx` |
| Barra de status | própria | **pronto** | `components/StatusBar.tsx` |
| Configurações: modal | S2 `DialogContainer` + `Dialog size="XL"` | `SettingsModal.tsx` | `SettingsModal.tsx` |
| Configurações: navegação | S2 `Tabs orientation="vertical"` | `SettingsModal.tsx` | `SettingsModal.tsx` |
| Campos (Workspace, provedor) | S2 `TextField` (`type="password"` na chave) + `Button` | `SettingsModal.tsx` | `SettingsModal.tsx` |
| Lista de provedores | S2 `ListView selectionMode="single"` | `SettingsModal.tsx` | `SettingsModal.tsx` |
| Modelos | S2 `ListView` + `ActionButtonGroup` (padrão, editar, apagar) + `Badge` "padrão" | `ModelsSection.tsx` | `ModelsSection.tsx` |
| Aparência (tema) | S2 `SelectBoxGroup`, cartões com a prévia | — | `SettingsModal.tsx` |
| Salvar / Excluir | S2 `Button variant="accent"` / `variant="negative"` | `SettingsModal.tsx` | `SettingsModal.tsx` |
| Apagar Conversa | S2 `AlertDialog variant="destructive"` | **pronto** | `Sidebar.tsx` |
| Envio bloqueado | S2 `AlertDialog` | o `.alert` do `App.tsx` | `App.tsx` |

### Ícones

Os do S2 (`@react-spectrum/s2/icons/<Nome>`), em todo o app. Quatro são próprios, em `src/icons/`,
desenhados a partir do SVG do protótipo: **painel lateral** (`PainelLateral`, recolher painel) e os
três da janela, **minimizar**, **maximizar** e **restaurar** (`MinimizarJanela`, `MaximizarJanela`,
`RestaurarJanela`): o `Minimize`/`Maximize` do S2 são setas de encolher e expandir, não os glifos de
janela. O SVG fonte (20×20, fill `var(--iconPrimary, #222)`) fica em `src/icons/svg/`, e o `.tsx` sai
do `@react-spectrum/s2-icon-builder` 1.1.2, rodado fora do projeto, sem virar dependência:
`npx transform-icons -i <svg> -o <pasta>` (o cabeçalho do arquivo diz de onde veio). Ícone de ação
do tool call sem equivalente no S2 segue o mesmo caminho.

_Decidido, a implementar:_ o `@phosphor-icons/react` sai do `package.json`. A Titlebar, a Sidebar,
as Abas, a barra de status e os botões da janela já não o usam; as outras telas ainda usam.

### Fonte

Source Sans 3 (interface, normal e itálico) e JetBrains Mono (bloco de código, Logs), em woff2
variável embutido em `src/assets/fonts`, só o subconjunto latino (cobre o pt-BR), com as licenças
OFL ao lado (`OFL-SourceSans3.txt`, `OFL-JetBrainsMono.txt`). Os arquivos vêm dos pacotes
`@fontsource-variable/source-sans-3` 5.3.1 e `@fontsource-variable/jetbrains-mono` 5.3.0, copiados
à mão, sem dependência. Os `@font-face` ficam no `global.css`. Os controles S2 e as superfícies com
`fontFamily: 'sans'` recebem a família por `--s2-font-family-sans: 'Source Sans 3'` no `:root`. A variável não é documentada;
se sumir, a queda é a Segoe UI. A JetBrains Mono entra pelo escape-hatch `css()` do macro, porque o
`fontFamily` dele só aceita os tokens, e o token `code` é fixo. Nenhuma superfície usa a JetBrains
Mono ainda: o navegador só baixa o arquivo quando alguma usar.

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

**Início** (_Decidido, a implementar_, `components/Home.tsx`): o título "O que vamos fazer hoje?",
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
aba, nenhum provedor), quem avisa é o composer. A Início não tem "Limpar conversa". Enquanto este
corte não entra, a Início mostra o estado vazio legado do `ChatArea`.

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

**Toast** (_Decidido, a implementar_): o `ToastQueue` do S2, no `ToastContainer` do `App.tsx`. Hoje
o container está montado e nada o dispara. Ele só confirma uma ação que não deixa rastro visível ou
avisa que ela falhou:

| Ação | Variante | Frase |
|---|---|---|
| Copiar o bloco de código (`CodeBlock.tsx`) | `positive` | "Código copiado." |
| Copiar o caminho do log (`LogDrawer.tsx`) | `positive` | "Caminho copiado." |
| Uma dessas cópias falhar | `negative` | "Não foi possível copiar." |

O `Link` da gaveta de Logs fica sempre com o rótulo "Copiar o caminho do arquivo". A troca para
"Caminho copiado" por 1,6 s sai. Mais nada vira toast:

- Salvar nas Configurações: o próprio dialog mostra o resultado.
- Baixar o diagrama: o WebView2 mostra o download. Se o Wails v3 esconder essa interface, isso
  volta a ser decidido.
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
