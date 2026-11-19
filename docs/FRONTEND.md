# Frontend (`cmd/ui/frontend`)

O frontend do app desktop: React 19 + TypeScript + Vite, empacotado pelo Wails v3.1.0 e renderizado no
WebView2. Conversa com o Go por `src/wails.ts`, que chama os métodos do `App` pelo nome com o
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
  `lightningcss` 1.33.0. O Vite é o 8.3.8, que empacota com Rolldown 1.2.19, com
  `@vitejs/plugin-react` 5.2.0 e `vitest` 5.1.0.
- **`vite.config.ts`:** a config oficial do S2. `macros.vite()` é o primeiro plugin, com
  `build.target: ['es2022']`, `cssMinify: 'lightningcss'` e um `manualChunks` que junta o CSS do S2
  e dos macros no chunk `s2-styles`. Há ainda `optimizeDeps.exclude: ['@react-spectrum/s2/style']`:
  sem ele, o scan do `vite dev` não resolve o import do macro e recarrega a página na primeira
  carga (no Vite 8 também). Os nomes `rollupOptions` e `manualChunks` estão deprecados no Vite 8, mas
  funcionam; ficam como na doc do S2.
- **`tsconfig.json`:** `moduleResolution: "Bundler"`, porque com `"Node"` os sub-paths
  (`@react-spectrum/s2/Button`) dão `TS2307`. O `tsconfig.node.json`, que só cobre o
  `vite.config.ts`, segue em `"Node"`. Ele já não passava no `tsc -p` antes do S2 e não entra no
  `npm run build`.
- **`allowScripts`:** `"@swc/core": false`. O `postinstall` dele só instala o fallback wasm, e o
  binário nativo chega por `optionalDependencies`.
- **CSS legado:** `App.css` e `Settings.css` ficam inteiros dentro de `@layer legacy { … }`.
  `App.css` abre com `@layer legacy, _;`, que põe o legado abaixo da camada `_` do S2. Assim, o CSS
  global antigo (`*`, `body`, `button`) não vence os controles do S2. Fica assim até o visual novo
  apagar os dois arquivos.
  - Conflito conhecido: popover do S2 (Picker, Menu) aberto dentro de um overlay legado com
    `z-index` (o modal de Configurações, `.settings-scrim`, `z-index: 100`) fica invisível atrás
    dele. Controle S2 com popover só entra em superfície legada se essa superfície já tiver saído.
  - `style.css` não é importado por ninguém, então a Nunito de `assets/fonts` não chega ao bundle. A
    fonte do app hoje é a pilha de sistema do `App.css` (Segoe UI no Windows).
- **Rede:** o `index.html` traz `<meta http-equiv="Content-Security-Policy" content="font-src 'self' data:">`.
  O `Provider` do S2 sempre tenta a Adobe Clean em `use.typekit.net`, e essa política faz o WebView2
  barrar essas requisições antes de saírem. Os controles S2 caem na Segoe UI.
- **Wails v3.1.0** (fixada por decisão do usuário, 2026-11-19; fatos em [`WAILS_V3.md`](WAILS_V3.md)):
  - `cmd/ui/main.go` monta `application.New` com o `App` como único serviço e uma janela frameless
    1024×768 de fundo `#0a0f1c`. O `App.ServiceStartup` guarda o `ctx` dos turnos, e o `emit` dos
    eventos `agent:token`/`agent:done` vai pelo `Event.Emit` do app.
  - `src/wails.ts` é a única ponte. `goApp()` devolve um proxy que chama `main.App.<Método>` por
    `Call.ByName`, ou `undefined` fora do app (sem `window._wails.environment`). Ele também
    reexporta `Events`, `Window`, `Application` e `Clipboard`. Importá-lo liga o arrasto da
    titlebar (`--wails-draggable`). O evento chega com o dado em `e.data`.
  - Não há bindings geradas nem `wails3` no build: os tipos são os de `types.ts`. O erro de uma
    chamada é um `RuntimeError` cuja `message` é o erro do Go, e `errorText(e)` tira dele o texto.
  - Cancelar o seletor de pasta não é erro: o `PickWorkspace` trata o `"cancelled by user"` do v3
    como o `""` do v2.
- **Bundle (Vite 8):** sem nenhum componente S2 importado, o tree-shaking deixa o S2 de fora (JS
  1.167 KB, gzip 397 KB). Com `Provider` + `Button`, o JS vai a 1.326 KB (gzip 440 KB) e o chunk
  `s2-styles` soma 25 KB (gzip 4,6 KB). Os 34 locales do S2 vão juntos. Medição completa em
  [`REACT_SPECTRUM_S2.md`](REACT_SPECTRUM_S2.md#9-peso).

## Tema

_Decidido, a implementar._

- **Três escolhas, padrão Sistema.** Claro e Escuro passam `colorScheme` ao `Provider`. Sistema,
  que vale também quando nada foi salvo, omite o prop, e então o app acompanha o Windows mesmo com a
  janela aberta. Não há paleta própria: o escuro é o do S2.
- **Seletor:** em Configurações → Aparência, um `SelectBoxGroup` com cartões de prévia. A troca
  vale ao clicar, sem botão Salvar.
- **Persistência:** a escolha vai para o `localStorage` do WebView2. Para que o `wails dev`, o `go run` e o binário construído
  enxerguem a mesma escolha, o `main.go` fixa `windows.Options.WebviewUserDataPath` em
  `~/.network_agent/webview2`. Sem ele, o go-webview2 usa `%AppData%\<nome do exe>`, uma pasta por
  forma de rodar.
- **Sem piscar ao abrir:** a janela nasce com `StartHidden: true`, e o `OnDomReady` do Go chama
  `runtime.WindowShow`. Antes disso, um script inline no `<head>` do `index.html` lê a escolha
  salva e pinta o fundo do documento: claro ou escuro direto, ou `color-scheme: light dark` no
  Sistema. O `BackgroundColour` fixo `#0a0f1c` deixa de importar. **A conferir no `wails dev`:** a
  janela escondida precisa aparecer já renderizada, sem WebView2 em branco. Se aparecer em branco,
  o plano B é o Go descobrir o tema antes de abrir a janela: a escolha salva passa para o
  `config.json`, e no Sistema ele lê o `AppsUseLightTheme` do registro.
- **Diagrama:** o SVG na tela usa `var(--pele-*)`, então troca de tema sozinho, sem renderizar de
  novo. O SVG baixado leva o tema em uso, com as cores **resolvidas** (rgb/hex) no `<style>`
  injetado: um `light-dark(…)` dos tokens do S2 não pode ir para o arquivo, porque fora do app ele
  sempre resolveria para o claro ou nem renderizaria.
- **Bloco de código:** sem realce de sintaxe, então as cores `--code-*` do protótipo não entram.

## Shell e janela

_Decidido, a implementar._ A janela é o app: o shell não desenha a moldura do protótipo (padding,
gradiente, raio, sombra), e os brilhos do `app__bg` saem. Canto arredondado e sombra ficam com o
Windows 11, que os dá à janela frameless. Os nomes das opções de janela abaixo são do Wails 2.15. No
Wails v3 eles são conferidos pela pesquisa _Wails v3 no app_ (ver [Stack e build](#stack-e-build)).

- **Grade:** a titlebar (48 px) em cima e a barra de status embaixo, as duas na largura toda. No
  meio ficam três colunas: sidebar ou faixa, conteúdo (barra de abas + Aba) e gaveta de Logs.
- **Titlebar** (`components/Titlebar.tsx`): à esquerda, recolher painel, menu principal, marca,
  "Network Agent" e o chip do Workspace. À direita, o modelo e os botões da janela. A barra toda
  arrasta (`--wails-draggable: drag`), e cada controle leva `no-drag`. Só a titlebar arrasta; a
  barra de abas, não. O duplo clique na área de arrasto maximiza ou restaura a janela: no Windows o
  runtime do v3 não faz isso sozinho, então um `onDoubleClick` chama `Window.ToggleMaximise()`.
- **Chip do Workspace:** mostra o nome da pasta do Workspace em uso, com o caminho no `title`. O
  menu lista até 5 **Workspaces recentes** (`menuitemradio`: nome e, embaixo, o caminho), por uso
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
  conexão falha, aparece um `ActionButton` "Reconectar", com a causa no `title`. Sem modelo
  configurado, o texto é "Nenhum modelo configurado".
- **Botões da janela** (`WindowControls.tsx`): minimizar, maximizar/restaurar e fechar. O glifo do
  maximizar acompanha o estado real da janela (snap, `Win+setas`, duplo clique). O hover do fechar é
  vermelho. Se o Snap Layouts aparece no hover do maximizar, num modo opcional do v3, é o ticket
  _Snap Layouts no maximizar_ que decide.
- **Sidebar ↔ faixa recolhida:** um estado só. Expandida, a sidebar tem 244 px; recolhida, vira a
  faixa de 56 px com o "Nova conversa". O botão de recolher painel, na titlebar, alterna as duas.
  O app sempre abre com a sidebar expandida, e a escolha não persiste. Não há atalho de teclado. A
  coluna anima a largura em 0,3 s, com a mesma curva da gaveta de Logs, e o conteúdo troca na hora.
  Não há modo sobreposto com scrim.
- **Gaveta de Logs:** é a terceira coluna, sem scrim. Tem 360 px fixos, que caem para 300 px com a
  janela abaixo de 1080 px de largura. Não tem divisor para redimensionar: as linhas quebram. Ela
  anima a largura ao abrir e fechar.
- **Tamanho:** a janela abre em 1024×768, com mínimo de 900×600 (`MinWidth`/`MinHeight`). O
  breakpoint de 1080 px vale no shell. O que ele muda no modal de Configurações e nas sugestões é
  das seções dessas telas.

## Abas e Conversas

_Decidido, a implementar._ Uma **Aba** é a **Início** ou uma **Conversa** aberta (ver `CONTEXT.md`).
A barra mora em `components/TabBar.tsx`.

- **Início:** é sempre a primeira aba e não tem "x". O primeiro envio por ela cria a Conversa e a
  abre numa aba nova, focada. A Início volta vazia. **Nova conversa**, na sidebar e na faixa
  recolhida, foca a Início e põe o cursor no composer dela, sem apagar o rascunho.
- **Abrir:** clicar numa Conversa da sidebar foca a aba dela se já houver uma. Se não houver, abre
  uma aba nova no fim da barra, já focada. Uma Conversa tem no máximo uma aba. O título da aba é o
  da Conversa e acompanha quando ela é renomeada. A Conversa da aba focada fica destacada na
  sidebar.
- **Sem limite de abas:** a barra rola na horizontal quando elas não cabem.
- **Fechar:** pelo "x" ou por `Ctrl+W`, que fecha a aba focada e não faz nada na Início. O foco vai
  para a vizinha da direita, ou para a da esquerda se não houver uma à direita. A Início é o piso.
  Fechar uma aba não apaga a Conversa. Apagar uma Conversa fecha a aba dela, com a mesma regra de
  foco. Alternar abas pelo teclado usa as setas do `TabList`.
- **Estado da aba:** enquanto a aba está aberta, ela guarda o rascunho do composer e a posição de
  rolagem. Fechar a aba descarta os dois.
- **Um turno por vez no app inteiro** (o `busyMux` do Go). A aba da Conversa que está no turno
  mostra o `dot` só enquanto o turno roda. Nas outras abas e na Início, dá para digitar, mas o envio
  fica bloqueado, com a linha "Newton está respondendo em _‹título›_". O resultado do turno, a falha
  e o **Aviso** caem na bolha da aba dele, mesmo fora de foco. Nunca viram toast nem o card da
  sidebar. Fechar a aba com o turno rodando não cancela o turno: ele termina no Go e é gravado no
  disco, e reabrir a Conversa religa o stream. Renomear e apagar seguem bloqueados durante um turno.
- **Sem persistência:** o app sempre abre só com a Início. As Conversas recentes estão na sidebar.

## Componentes

_Decidido, a implementar._ Cada elemento é um **controle S2** ou uma **superfície própria** (RAC ou
HTML, estilizada pelo macro `style` com os tokens do S2). `App.css` e `Settings.css` são apagados
sem uma folha de superfícies no lugar. O único CSS global é um arquivo pequeno com os `@font-face`,
a ordem das camadas e a regra de movimento reduzido. O Markdown estiliza cada elemento pelo
`components` do `react-markdown`, com uma classe do macro por elemento.

| Elemento | Origem | Hoje | Arquivo |
|---|---|---|---|
| Titlebar (marca, região de arrasto) | própria | topo da `Sidebar.tsx`, `App.tsx` | `components/Titlebar.tsx` |
| Recolher painel | S2 `ActionButton isQuiet` + ícone próprio | `Sidebar.tsx` | `Titlebar.tsx` |
| Menu principal (Logs, Configurações) | S2 `MenuTrigger` + `Menu` | itens da `Sidebar.tsx` | `Titlebar.tsx` |
| Chip do Workspace | S2 `MenuTrigger` + `Menu` | — | `Titlebar.tsx` |
| Modelo e conexão | própria (texto "provedor · modelo") + S2 `ProgressCircle size="S"` (carregando) + S2 `ActionButton` "Reconectar" (falha) | `model-pill` do `App.tsx` | `Titlebar.tsx` |
| Botões da janela | própria (o hover vermelho do fechar não sai do S2) | `WindowControls.tsx` | `WindowControls.tsx` |
| Faixa recolhida | `nav` própria + S2 `Button variant="accent"` só com ícone | modo `collapsed` da `Sidebar.tsx` | `Sidebar.tsx` |
| Nova conversa | S2 `Button variant="accent"` | `Sidebar.tsx` | `Sidebar.tsx` |
| Lista de Conversas | própria em RAC `GridList` + `GridListSection` (grupos por data); ações em S2 `ActionButton isQuiet`; renomear inline | `Sidebar.tsx` | `Sidebar.tsx` |
| Card de erro da sidebar | S2 `InlineAlert variant="negative"` + `ActionButton` (Tentar de novo, Dispensar) | `Sidebar.tsx` | `Sidebar.tsx` |
| Abas | própria em RAC `Tabs`/`TabList`/`Tab`: o "x" dentro da aba, fora da ordem de Tab, e fechar também por teclado; rola na horizontal quando não cabe | — | `components/TabBar.tsx` |
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
| Toast | S2 `ToastContainer placement="bottom end"` + `ToastQueue`; uma frase só, sem subtítulo | — | `App.tsx` |
| Gaveta de Logs | `aside` própria + S2 `ActionButton` (fechar) + S2 `Link` (copiar o caminho) | `LogDrawer.tsx` | `LogDrawer.tsx` |
| Barra de status | própria | rodapé do `InputDock.tsx` | `components/StatusBar.tsx` |
| Configurações: modal | S2 `DialogContainer` + `Dialog size="XL"` | `SettingsModal.tsx` | `SettingsModal.tsx` |
| Configurações: navegação | S2 `Tabs orientation="vertical"` | `SettingsModal.tsx` | `SettingsModal.tsx` |
| Campos (Workspace, provedor) | S2 `TextField` (`type="password"` na chave) + `Button` | `SettingsModal.tsx` | `SettingsModal.tsx` |
| Lista de provedores | S2 `ListView selectionMode="single"` | `SettingsModal.tsx` | `SettingsModal.tsx` |
| Modelos | S2 `ListView` + `ActionButtonGroup` (padrão, editar, apagar) + `Badge` "padrão" | `ModelsSection.tsx` | `ModelsSection.tsx` |
| Aparência (tema) | S2 `SelectBoxGroup`, cartões com a prévia | — | `SettingsModal.tsx` |
| Salvar / Excluir | S2 `Button variant="accent"` / `variant="negative"` | `SettingsModal.tsx` | `SettingsModal.tsx` |
| Envio bloqueado, apagar Conversa | S2 `AlertDialog` | os `.alert` do `App.tsx` | `App.tsx` |

### Ícones

Os do S2 (`@react-spectrum/s2/icons/<Nome>`), em todo o app. Dois são próprios, desenhados a partir
do SVG do protótipo e convertidos pelo `@react-spectrum/s2-icon-builder`: **painel lateral** (recolher)
e **restaurar janela**. Ícone de ação do tool call sem equivalente no S2 segue o mesmo caminho. O
`@phosphor-icons/react` sai do `package.json`.

### Fonte

Source Sans 3 (interface) e JetBrains Mono (bloco de código, Logs), em woff2 variável embutido em
`src/assets/fonts`. Os controles S2 e as superfícies com `fontFamily: 'sans'` recebem a família por
`--s2-font-family-sans: 'Source Sans 3'` num ancestral do `Provider`. A variável não é documentada;
se sumir, a queda é a Segoe UI. A JetBrains Mono entra pelo escape-hatch `css()` do macro, porque o
`fontFamily` dele só aceita os tokens, e o token `code` é fixo. A Nunito de `assets/fonts` e o
`style.css` são apagados.

### Movimento

O da Adobe. Os controles S2 animam com as próprias transições. A superfície própria imita o
componente S2 mais próximo, com os tokens de transição do macro (`transition`, easings `in`, `out`,
`default`):

- o indicador da aba desliza como o do S2 `Tabs` (`translate`/`width`, 0,2 s);
- a troca de aba não anima o conteúdo, porque o `TabPanel` do S2 não anima;
- a gaveta de Logs anima a largura como o `SideNavPanel` (0,3 s);
- o clique em sugestão, item da lista e aba comprime como os botões S2.

As keyframes do protótipo (`od-*`, `toast-in`, `toast-out`) não entram. Com movimento reduzido, uma
regra `@media (prefers-reduced-motion: reduce)` zera `transition` e `animation` numa `@layer`
declarada depois da `_` do S2. Ela cobre Modal, Popover, Menu e Tooltip, que não têm a regra no CSS do
S2.

### Idioma

Só pt-BR: `<Provider locale="pt-BR">` e `@react-aria/optimize-locales-plugin` com
`locales: ['pt-BR']` (`enforce: 'pre'`) no `vite.config.ts`. Os textos do próprio S2 saem em
português qualquer que seja o idioma do Windows, e os outros 33 locales saem do bundle (−27 KB gz,
medido).

## Início, barra de status e toast

_Decidido, a implementar._

**Início** (`components/Home.tsx`): o título "O que vamos fazer hoje?", o subtítulo "Descreva a
tarefa. O Newton monta, ajusta, desenha ou explica configurações FRR.", o composer e quatro
sugestões. As sugestões são uma lista fixa no próprio `Home.tsx`, uma por capacidade do Newton,
nesta ordem:

1. "Montar um eBGP entre o AS 65001 e o AS 65002"
2. "Adicionar BFD ao BGP do frr.conf"
3. "Desenhar um iBGP com route reflector"
4. "Explicar como o OSPF elege o DR"

Clicar numa sugestão põe o texto dela no composer da Início, no lugar do rascunho, e foca o campo
com o cursor no fim. Ela não envia, porque um envio do Builder pode substituir o `frr.conf` do
Workspace. As sugestões nunca ficam desabilitadas. Quando o envio está bloqueado (turno em outra
aba, nenhum provedor), quem avisa é o composer. A Início não tem "Limpar conversa".

**Barra de status** (`components/StatusBar.tsx`): uma linha de 30 px no pé da janela, na largura
toda, que mostra a Aba focada, não a do turno em voo.

- **À esquerda:** a hora da última resposta do Newton na Conversa, no formato da meta da mensagem e
  com o tooltip "Último turno". Antes da primeira resposta, ela não aparece.
- **À direita:** os itens que hoje estão no rodapé do `InputDock.tsx`: tokens de entrada, de saída,
  o total e o contexto utilizado/janela com a origem (`contextOrigin`). Sem janela sondada nem
  medida, o item de contexto não aparece.
- **Durante um turno:** o horário e os totais continuam os do turno anterior até ele terminar.
- **Na Início:** a mesma barra, com os totais zerados, o contexto em 0 sobre a janela do modelo
  ativo e sem horário.

**Toast**: o `ToastQueue` do S2, no `ToastContainer` do `App.tsx`. Ele só confirma uma ação que não
deixa rastro visível ou avisa que ela falhou:

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
- **Dev e build** (_Decidido, a implementar_): o projeto segue o modelo do Wails v3, com a CLI
  `wails3` 3.1.0, e os comandos rodam de dentro de `cmd/ui`. O dev é o `wails3 dev`: ele sobe o Vite
  na 9245, o front recarrega sozinho e o Go recompila ao salvar. O build é o `wails3 build`, e o
  `.exe` sai em `cmd/ui/bin/`. Os dois seguem o `build/config.yml` e o `Taskfile.yml` gerados. As
  bindings ficam em `frontend/bindings/`, versionadas. Até isso ser implementado, valem os dois
  processos do [`cmd/ui/README.md`](../cmd/ui/README.md).
- **Porta de depuração:** com `NETWORK_AGENT_CDP_PORT=9333` no ambiente do app, a janela abre o
  CDP em `localhost:9333` (`/json/list` dá o `webSocketDebuggerUrl` da página). O DevTools da
  janela também abre à mão, em build sem a tag `production`.
