# React Spectrum S2 no stack do app

Fatos sobre o React Spectrum 2 (`@react-spectrum/s2`) que pesam no frontend `cmd/ui/frontend`
(Vite, React 19, TypeScript, WebView2 do Wails v2). Coletado em 2026-11-18, cada fato com a fonte.
Só fatos, nenhuma decisão: o que o app **decidiu e implementou** fica em
[`FRONTEND.md`](FRONTEND.md).

## Como usar e manter

- **Leia antes de pesquisar.** Pesquise só o que não estiver aqui, ou o que estiver numa versão do S2
  diferente da instalada.
- **Fato novo ou corrigido entra aqui**, na seção do tema, com a versão do pacote, a data e a fonte.
  Um fato medido no app (build, `wails dev`, WebView2) entra marcado como **Medido no app** e sai de
  [Não confirmado](#não-confirmado).
- **Ao atualizar o S2**, confira as seções contra a versão nova e atualize as versões de referência.

## Resumo

1. **Biblioteca:** o site oficial apresenta o S2 como atual (o v3 foi para `/v3/`). Estável desde a
   1.0.0 (2025-12-16), exige React ^19. `SideNav`, `SideNavPanel` e `@react-spectrum/ai` são alpha.
2. **Build:** `unplugin-parcel-macros` como **primeiro** plugin do Vite, `lightningcss` como devDep,
   `cssMinify: 'lightningcss'`, e todo o CSS do S2 + macros num chunk só. CSS global fora de `@layer`
   vence o S2. Sub-paths por componente pedem `moduleResolution: "bundler"`.
3. **Inventário:** os controles padrão existem. Faltam: `Tab` com fechar, Toast com título +
   subtítulo/ícone (o conteúdo é uma string), Toolbar, Drawer/Sheet, seções no ListView. O RAC
   (`react-aria-components`, sem estilo) tem Toolbar, GridList com seções e Sheet.
4. **Tema:** `colorScheme` é `'light' | 'dark'`; "sistema" é omitir o prop. O acento `#3B63FB` é o
   padrão do S2. **Não há API para trocar o acento**: as cores compilam para hex em build.
5. **Customização:** o prop `styles` aceita só layout. Visual próprio = RAC + macro `style`.
6. **Fonte:** a Adobe Clean só vem da Typekit (`use.typekit.net`) e é restrita a produtos Adobe. A
   família troca por `--s2-font-family-sans` (não documentada). O `Provider` emite um `preload` para a
   Typekit sempre, mesmo com a família trocada.
7. **Ícones:** um módulo por ícone em `@react-spectrum/s2/icons/<Nome>`. Faltam sidebar/painel,
   restaurar janela e sol/lua.
8. **Movimento:** Toast, Tabs, Disclosure e SideNavPanel respeitam `prefers-reduced-motion`; Modal,
   Popover, Menu e Tooltip não têm a regra no CSS compilado.
9. **Peso:** sub-path por componente e CSS atômico; o min+gz no app ainda não foi medido.

## Fatos levantados

**Versões de referência (registro npm, nesta data):** `@react-spectrum/s2` **1.8.2** (latest,
2026-10-15) · `@adobe/react-spectrum` (v3) 3.47.7 (2026-10-09) · `react-aria-components` 1.22.1 ·
`@react-spectrum/ai` 0.5.2 · `unplugin-parcel-macros` 0.2.0 (2026-02-26).
Fontes: <https://registry.npmjs.org/@react-spectrum/s2>, <https://registry.npmjs.org/@adobe/react-spectrum>,
<https://registry.npmjs.org/react-aria-components>, <https://registry.npmjs.org/@react-spectrum/ai>,
<https://registry.npmjs.org/unplugin-parcel-macros>.

**Método.** O site oficial serve cada página também como Markdown (`<página>.md`, índice em
`/llms.txt`), e é dessas páginas que vêm as citações. O código-fonte é o do branch `main` de
`adobe/react-spectrum`. O pacote publicado 1.8.2 foi lido arquivo a arquivo pelo jsDelivr, que
espelha o tarball do npm (`https://cdn.jsdelivr.net/npm/@react-spectrum/s2@1.8.2/...`). Nada foi
instalado.

---

### 1. Qual biblioteca

- **O site oficial apresenta o S2 como a biblioteca atual.** O índice da raiz se chama
  "React Spectrum (S2) Documentation", e o Getting started manda instalar `@react-spectrum/s2`.
  O v3 foi para `/v3/`, com o banner "Spectrum 2 is now available! Check out the S2 getting-started
  docs and the migration docs".
  Fontes: <https://react-spectrum.adobe.com/llms.txt>, <https://react-spectrum.adobe.com/getting-started.md>,
  <https://react-spectrum.adobe.com/v3/getting-started.html>.
- **Maturidade: estável (GA).** A 1.0.0 saiu em 2025-12-16, com o texto "This milestone marks the
  first stable version". Depois dela vieram releases mensais, de 1.1 a 1.8.2.
  Fontes: <https://react-spectrum.adobe.com/releases/v1-0-0.md>, <https://registry.npmjs.org/@react-spectrum/s2>.
- **Alguns componentes ainda são alpha:** `SideNav`, `SideNavPanel` e a página de AI Components
  (`export const version = 'alpha'` no MDX). Toast, SelectBoxGroup, ListView, TreeView e Tabs não
  carregam marca de versão, ou seja, são estáveis.
  Fontes: <https://github.com/adobe/react-spectrum/blob/main/packages/dev/s2-docs/pages/s2/SideNav.mdx>,
  <https://github.com/adobe/react-spectrum/blob/main/packages/dev/s2-docs/pages/s2/SideNavPanel.mdx>,
  <https://github.com/adobe/react-spectrum/blob/main/packages/dev/s2-docs/pages/s2/ai-components.mdx>,
  <https://github.com/adobe/react-spectrum/blob/main/packages/dev/s2-docs/src/VersionBadge.tsx>.
- **Peer de React: 19.** A 1.8.2 declara `react`/`react-dom` `^19.0.0-rc.1`. O React 18 saiu do
  peer na 1.0.0; até a 0.12 era `^18.0.0 || ^19.0.0-rc.1`. O v3 e o RAC aceitam de 16.8 a 19.
  O app usa React `^19.1.0`, então é compatível.
  Fonte: <https://registry.npmjs.org/@react-spectrum/s2>.
- **Convivência com o v3:** "When using S2 together with other versions of Spectrum, ensure that the
  S2 provider is the inner-most provider."
  Fonte: <https://react-spectrum.adobe.com/getting-started.md>.
- Licença do pacote: Apache-2.0. Fonte: <https://registry.npmjs.org/@react-spectrum/s2>.

### 2. Build (style macro no Vite)

- **Plugin:** `npm install unplugin-parcel-macros --dev`. Configuração oficial para Vite, copiada
  na íntegra:

  ```ts
  import {defineConfig} from 'vite';
  import react from '@vitejs/plugin-react';
  import macros from 'unplugin-parcel-macros';

  export default defineConfig({
    plugins: [
      macros.vite(), // Must be first!
      react()
    ],
    build: {
      target: ['es2022'],
      cssMinify: 'lightningcss',
      rollupOptions: {
        output: {
          manualChunks(id) {
            if (/macro-(.*)\.css$/.test(id) || /@react-spectrum\/s2\/.*\.css$/.test(id)) {
              return 's2-styles';
            }
          }
        }
      }
    }
  });
  ```

  Fonte: <https://react-spectrum.adobe.com/getting-started.md> (aba Vite).
- **Exemplo oficial no repo:** `examples/s2-vite-project` usa exatamente essa configuração, mais
  `optimizeLocales`. As devDeps incluem `lightningcss ^1.27.0`, `unplugin-parcel-macros` e
  `vite ^8.0.0`. O tsconfig do exemplo usa `"moduleResolution": "bundler"`.
  Fontes: <https://github.com/adobe/react-spectrum/blob/main/examples/s2-vite-project/vite.config.ts>,
  <https://github.com/adobe/react-spectrum/blob/main/examples/s2-vite-project/package.json>,
  <https://github.com/adobe/react-spectrum/blob/main/examples/s2-vite-project/tsconfig.json>.
- **`lightningcss` é dependência opcional do Vite:** `build.cssMinify: 'lightningcss'` exige
  `npm add -D lightningcss`.
  Fonte: <https://v7.vite.dev/guide/features> (seção Lightning CSS).
- **Exigências de CSS** (doc de Styling): "Ensure styles are extracted into a CSS bundle; do not
  inject at runtime with `<style>` tags", "Use a CSS minifier like `lightningcss` to deduplicate
  common rules" e "Bundle all CSS for S2 components and style macros into a single CSS bundle rather
  than code splitting". Sem essas otimizações "the generated CSS may contain duplicate rules".
  Fonte: <https://react-spectrum.adobe.com/styling.md>.
- **Locales (opcional):** `@react-aria/optimize-locales-plugin` com `enforce: 'pre'` e uma lista de
  `locales`. Por padrão o pacote traz strings de 30+ idiomas, `pt-BR` entre eles.
  Fontes: <https://react-spectrum.adobe.com/getting-started.md>, <https://react-spectrum.adobe.com/Provider.md>.
- **Cadeia de dependências nativas:** `unplugin-parcel-macros` 0.2.0 depende de `@swc/core`,
  `napi-wasm`, `@parcel/source-map` e `unplugin ^1.9.0`. O `@swc/core` tem script `postinstall`
  (`node postinstall.js`). Isso pesa porque o `package.json` do app tem uma lista `allowScripts`,
  que hoje só libera o esbuild.
  Fontes: <https://registry.npmjs.org/unplugin-parcel-macros>, <https://registry.npmjs.org/@swc/core>.
- **TypeScript:** a importação do macro usa import attributes
  (`import {style} from '@react-spectrum/s2/style' with {type: 'macro'}`), sintaxe suportada desde
  o TS 5.3. O app está no TS 5.6.
  Fonte: <https://www.typescriptlang.org/docs/handbook/release-notes/typescript-5-3.html>.
- **Sub-path imports (`@react-spectrum/s2/Button`) dependem de `exports`.** As notas da 1.3.0
  dizem: "you must be using tools that support the package.json `exports` field… TypeScript
  (`"moduleResolution": "bundler"`)". Com `moduleResolution: "node"`/`node10`, o TS só lê `exports`
  se `resolvePackageJsonExports` estiver ligado.
  - **Fato do repo:** `cmd/ui/frontend/tsconfig.json` está em `"moduleResolution": "Node"`.
  - No pacote 1.8.2 não existe pasta física `Button/`. Já a entrada principal
    (`@react-spectrum/s2`, campo `types`), `style/` (tem `package.json` próprio) e
    `icons/*.d.ts` existem fisicamente.
  - Consequência: com `Node`, a entrada principal, o macro e os ícones resolvem os tipos, mas os
    sub-paths de componente não. Essa parte é inferência minha, não testada.

  Fontes: <https://react-spectrum.adobe.com/releases/v1-3-0.md>,
  <https://www.typescriptlang.org/tsconfig/#moduleResolution>,
  <https://data.jsdelivr.com/v1/packages/npm/@react-spectrum/s2@1.8.2?structure=flat>.
- **Convivência com CSS global legado:** o S2 usa CSS Cascade Layers, e "global CSS outside a
  `@layer` will override S2's CSS". A doc também diz que "CSS resets are strongly discouraged". Se
  não der para tirar o reset, ele vai numa camada abaixo da `_` do S2:

  ```css
  @layer reset, _;
  @import "reset.css" layer(reset);
  ```

  O CSS compilado da 1.8.2 confirma camadas `@layer _.a`, `_.b`, `_.b.l`… `_.h`.
  - **Fato do repo:** `App.css` tem seletores globais fora de `@layer` (`:root`, `*`, `body`,
    `button`, `*::-webkit-scrollbar*`), e `style.css` tem `html`, `body` e `@font-face`. Pela regra
    acima, esses seletores vencem o S2 durante a migração se não forem para uma camada.

  Fontes: <https://react-spectrum.adobe.com/styling.md> (seção CSS Resets),
  <https://cdn.jsdelivr.net/npm/@react-spectrum/s2@1.8.2/dist/private/Button.css>.
- **`page.css`:** é para apps de página inteira. Aplica fundo e color scheme no `<html>` antes do
  JS carregar e pode ser ajustado com `data-background` e `data-color-scheme` no `<html>`. A partir
  da 0.10 o `Provider` continua obrigatório, porque é ele que carrega as fontes.
  Fontes: <https://react-spectrum.adobe.com/getting-started.md>, <https://react-spectrum.adobe.com/releases/v0-10-0.md>.
- **Alvo de navegador:** o `package.json` do S2 declara o browserslist "last 2 Chrome versions, last
  2 Safari versions, last 2 Firefox versions, last 2 Edge versions". O WebView2 Evergreen "updates
  automatically… receives the same Microsoft Edge updates", mas "Updates… might not occur if an IT
  Admin disables updating, or if the client is offline".
  Fontes: <https://cdn.jsdelivr.net/npm/@react-spectrum/s2@1.8.2/package.json>,
  <https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/distribution>.

### 3. Inventário: elementos do protótipo e o que existe no S2

**Existem no S2 1.8.2**, todos documentados com sub-path próprio: Button (variantes `accent`,
`primary`, `secondary`, `negative`, `premium`, `genai`), LinkButton, ActionButton (`isQuiet`),
ActionButtonGroup, ToggleButton, ToggleButtonGroup, ActionMenu, Picker, Menu, MenuTrigger,
SubmenuTrigger e MenuSection (`selectionMode`, `trigger="contextMenu"`), Tabs, TabList, Tab e
TabPanel (`orientation`, `density`, `labelBehavior`, colapso automático em Picker), Dialog,
DialogTrigger, DialogContainer, AlertDialog, FullscreenDialog, CustomDialog e Popover,
ToastContainer e ToastQueue, TextField (`type="password"` etc.), TextArea, SearchField, RadioGroup
e Radio, SelectBoxGroup, SegmentedControl, Switch, Checkbox, Badge, StatusLight, NotificationBadge,
ProgressCircle, ProgressBar, Divider, Link, Tooltip, InlineAlert, IllustratedMessage, Skeleton,
Disclosure, Accordion, Card, ListView, TreeView, TableView, Breadcrumbs, SideNav (alpha) e
SideNavPanel (alpha).
Fontes: <https://react-spectrum.adobe.com/llms.txt> e as páginas `<Componente>.md` correspondentes
(ex.: <https://react-spectrum.adobe.com/Dialog.md>, <https://react-spectrum.adobe.com/Menu.md>).

**Não existem no S2:** `Flex`, `Grid`, `View` e `Well` ("These no longer exist. Please style
`<div>`… with the style macro"). Também não há Toolbar, ListBox, GridList, Sheet nem Drawer.
Fontes: <https://github.com/adobe/react-spectrum/blob/main/packages/dev/s2-docs/src/S2FAQ.tsx>,
<https://react-spectrum.adobe.com/migrating.md>, <https://react-spectrum.adobe.com/llms.txt>.

**Existem só no `react-aria-components` (sem estilo):** Toolbar, GridList (com `GridListSection`),
ListBox, Tree, NavigationTree, Group, Separator, Sheet, Modal, FileTrigger, Select, Autocomplete,
TokenField, Virtualizer e hooks como `useClipboard`. O `Toast` do RAC é **alpha** e sai com prefixo
`UNSTABLE_` (`UNSTABLE_ToastRegion`, `UNSTABLE_ToastQueue`). A doc de Styling indica o macro `style`
sobre RAC como o caminho para componentes próprios com visual Spectrum.
Fontes: <https://react-aria.adobe.com/llms.txt>, <https://react-aria.adobe.com/GridList.md>,
<https://react-aria.adobe.com/Toast.md>,
<https://github.com/adobe/react-spectrum/blob/main/packages/dev/s2-docs/pages/react-aria/Toast.mdx>,
<https://react-spectrum.adobe.com/styling.md>.

| Elemento do protótipo ([`FRONTEND_PROTOTIPO.html`](FRONTEND_PROTOTIPO.html)) | Equivalente S2 | Lacuna / nota |
|---|---|---|
| Botões da janela (min/max/fechar) | ActionButton `isQuiet` + ícones `Minimize`, `Maximize`, `Close` | Não há componente de chrome de janela nem ícone de "restaurar". |
| Recolher painel (titlebar) | ActionButton | Não há ícone de sidebar/painel na lista. O SideNavPanel (alpha) traz o próprio toggle, no rodapé do painel. |
| Menu principal / chip de Workspace (`menuitemradio`) | MenuTrigger + Menu + MenuItem/MenuSection, `selectionMode="single"` | Coberto. |
| Faixa vertical de ícones (`stripe`) | Sem Toolbar no S2. Opções: RAC `Toolbar` (sem estilo) ou o trilho recolhido do SideNavPanel (alpha) | Lacuna. |
| Lista de Conversas com grupos ("Ontem", "Últimos 7 dias") e ações renomear/excluir por linha | ListView: `ListViewItem` aceita ActionMenu/ActionButtonGroup como slot, `selectionStyle="highlight"`. SideNav (alpha): `SideNavSection`/`SideNavHeader` e slots de ActionMenu/ActionButtonGroup, mas com semântica de navegação por link (`href`, `selectedRoute`) | ListView não documenta seções. RAC `GridList` tem `GridListSection`. |
| "Nova conversa" | Button `variant="accent"` | Coberto. |
| Abas com ponto e "x" de fechar | Tabs/TabList/Tab/TabPanel | **`Tab` não tem prop de fechar.** O exemplo oficial remove abas com um ActionButton externo. |
| Composer (textarea + enviar) | TextArea + ActionButton/Button com ícone `Send`/`ArrowUpSend`. Alternativa: `PromptField` do `@react-spectrum/ai` (alpha, pacote separado 0.5.2) | O `@react-spectrum/ai` também traz Chat, Thread, UserMessage, ResponseStatus e MessageSuggestionList, todos alpha. |
| Sugestões da tela Início | Button/ActionButton (ou `MessageSuggestionList`, alpha) | Coberto. |
| "Limpar" (`link-action`) | Link ou ActionButton `isQuiet` | Coberto. |
| Barra do diagrama (zoom −/+, redefinir, ajustar, ver código, baixar) | ActionButtonGroup + ActionButton + `ZoomOut`, `ZoomIn`, `ZoomFitToScreen`, `Code`, `Download`, `Refresh` | Coberto. |
| Copiar código | ActionButton + `Copy`, Tooltip | Coberto. |
| Trilho de Perguntas | Sem equivalente | Superfície própria. |
| Toast (ícone + título + subtítulo + fechar) | ToastContainer + `ToastQueue.neutral/positive/negative/info(children: string, {actionLabel, onAction, timeout, onClose})`, `placement` `'top' \| 'top end' \| 'bottom' \| 'bottom end'` | **O conteúdo é uma única string** (`children: string` no código-fonte). Título + subtítulo e ícone próprio não estão na API. Timeout mínimo de 5 s, e toasts com ação não somem sozinhos. |
| Gaveta de Logs | Sem Drawer/Sheet no S2. RAC tem `Sheet` | Superfície própria. |
| Barra de status | Sem equivalente | `div` + macro. |
| Modal de Configurações | DialogTrigger/DialogContainer + Dialog (`size` S/M/L/XL, `isDismissible`) | Coberto. |
| Navegação interna (Workspace/Provedores/Aparência) | Tabs `orientation="vertical"` ou ListView | Escolha do de-para. |
| Campos de texto e senha | TextField (`type="password"`) | Coberto. |
| Tema com cartões de prévia (Claro/Escuro/Sistema) | RadioGroup/Radio, SelectBoxGroup (cartões, estável) ou SegmentedControl | Coberto, com formas diferentes. |
| Lista de provedores / linhas de modelo com ações | ListView + ActionButtonGroup | Coberto. |
| Botões primário / perigo | Button `accent` / `negative` | Coberto. |
| Badge, spinner, divisor | Badge, ProgressCircle, Divider | Coberto. |

Fontes da tabela: <https://react-spectrum.adobe.com/Tabs.md>, <https://react-spectrum.adobe.com/ListView.md>,
<https://react-spectrum.adobe.com/SideNav.md>, <https://react-spectrum.adobe.com/SideNavPanel.md>,
<https://react-spectrum.adobe.com/Toast.md>,
<https://github.com/adobe/react-spectrum/blob/main/packages/@react-spectrum/s2/src/Toast.tsx>,
<https://react-spectrum.adobe.com/Dialog.md>, <https://react-spectrum.adobe.com/TextField.md>,
<https://react-spectrum.adobe.com/RadioGroup.md>, <https://react-spectrum.adobe.com/ai-components.md>,
<https://react-spectrum.adobe.com/icons.md>.

### 4. Tema

- **`Provider`:** `colorScheme` aceita `'light' | 'dark'`, e o padrão "Defaults to operating system
  preferences". **Não existe o valor `'system'`.** "Sistema" é simplesmente omitir o prop.
  `background` aceita `'base' | 'layer-1' | 'layer-2'`; sem ele o fundo é transparente.
  Fontes: <https://react-spectrum.adobe.com/Provider.md>,
  <https://github.com/adobe/react-spectrum/blob/main/packages/@react-spectrum/s2/src/Provider.tsx>.
- **Como segue o SO:** um CSS de baixa especificidade define `--s2-color-scheme: light dark` e usa
  `@media (prefers-color-scheme: dark)`. As cores são emitidas como `light-dark(<claro>, <escuro>)`.
  A troca em runtime é só re-renderizar com outro `colorScheme`, ou, com `page.css`, trocar
  `data-color-scheme` no `<html>`. Não requer recarregar a página; isso é inferência do mecanismo
  CSS, não testada.
  Fontes: <https://github.com/adobe/react-spectrum/blob/main/packages/@react-spectrum/s2/src/page.macro.ts>,
  <https://cdn.jsdelivr.net/npm/@react-spectrum/s2@1.8.2/page.css>.
- **O acento do protótipo é o acento padrão do S2.** `#3B63FB` = `rgb(59, 99, 251)` =
  `accent-color-900` claro = `accent-background-color-default`. O `--accent-hover` (`#4069FD`) e o
  `--accent-down` (`#274DEA`) do protótipo também aparecem literais no `Button.css` compilado. No
  escuro, o `accent-color-900` vale `rgb(86, 129, 255)`.
  Fontes: <https://cdn.jsdelivr.net/npm/@adobe/spectrum-tokens/dist/json/variables.json>,
  <https://cdn.jsdelivr.net/npm/@react-spectrum/s2@1.8.2/dist/private/Button.css>.
- **Trocar o acento: não há API documentada.** Os tokens de cor são resolvidos **em build** para
  valores literais (`colorToken()` lê `@adobe/spectrum-tokens` e devolve `light`/`dark`), e o CSS
  compilado traz hex literais, sem variável de acento.
  - Isso é diferente do v3, onde "Colors, sizing, and spacing options can be customized through the
    use of CSS variables which are defined using the Provider component".
  - A FAQ do S2 diz que "Overrides to the Spectrum styles is highly discouraged, consider styling an
    equivalent React Aria Component instead".

  Fontes: <https://github.com/adobe/react-spectrum/blob/main/packages/@react-spectrum/s2/style/tokens.ts>,
  <https://react-spectrum.adobe.com/v3/theming.html>,
  <https://github.com/adobe/react-spectrum/blob/main/packages/dev/s2-docs/src/S2FAQ.tsx>.
- **Variáveis CSS expostas** (em `page.css` e no CSS compilado): `--s2-color-scheme`,
  `--s2-container-bg` (fundo da camada, em `light-dark()`), `--s2-scale` (1, ou 1.25 em touch),
  `--s2-font-size-base` (14, ou 17 em touch) e `--s2-font-family-sans`/`--s2-font-family-serif`
  (ver item 6). **Não há variáveis para a paleta.** Para pintar superfícies próprias com os mesmos
  valores, o caminho documentado é o macro `style` nos elementos próprios (ex.:
  `style({backgroundColor: 'layer-1', color: 'body'})`). Ele aceita todos os tokens: `gray-25…1000`,
  `accent-100…1600`, `base`, `layer-1`, `layer-2`, `elevated`, `pasteboard`, além de utilitários
  `lightDark()`, `colorMix()` e `color('accent-900/50')`. Também é possível definir variáveis CSS
  próprias pelo macro (`'--x': {type: 'backgroundColor', value: 'gray-400'}`).
  Fontes: <https://cdn.jsdelivr.net/npm/@react-spectrum/s2@1.8.2/page.css>,
  <https://react-spectrum.adobe.com/style-macro.md>, <https://react-spectrum.adobe.com/styling.md>.
- Os valores brutos estão em `@adobe/spectrum-tokens` (Apache-2.0, JSON em `dist/json/variables.json`).
  Fonte: <https://registry.npmjs.org/@adobe/spectrum-tokens>.

### 5. Customização

- **O prop `styles` só aceita layout.** "The `styles` prop accepts a limited set of CSS properties,
  including layout, spacing, sizing, and positioning. Other styles such as colors and internal
  padding cannot be customized within Spectrum components."
  - **Lista base:** `margin`, `marginStart`, `marginEnd`, `marginTop`, `marginBottom`, `marginX`,
    `marginY`, `width`, `minWidth`, `maxWidth`, `flexGrow`, `flexShrink`, `flexBasis`,
    `justifySelf`, `alignSelf`, `order`, `gridArea`, `gridRow*`, `gridColumn*`, `position`,
    `zIndex`, `top`, `bottom`, `inset`, `insetX`, `insetY`, `insetStart`, `insetEnd` e
    `visibility`.
  - **Variantes por componente** (código-fonte): `StylesPropWithHeight` acrescenta `size`,
    `height`, `minHeight` e `maxHeight` (Tabs e Provider, por exemplo); `StylesPropWithoutWidth`
    tira a largura; `StylesPropWithFont` aceita só `font`, `fontFamily`, `fontWeight`, `fontSize` e
    `lineHeight` (Disclosure, por exemplo).

  Fontes: <https://react-spectrum.adobe.com/styling.md>,
  <https://github.com/adobe/react-spectrum/blob/main/packages/@react-spectrum/s2/src/style-utils.ts>,
  <https://react-spectrum.adobe.com/Disclosure.md>.
- **`UNSAFE_className` / `UNSAFE_style`** existem em todos os componentes, com o aviso "Only use as a
  **last resort**". Passar o resultado do macro para `UNSAFE_className` é erro de TypeScript desde
  o PR #8109 (`UnsafeClassName = string & {properties?: never}`), e "The style macro is not meant to
  be used with UNSAFE_className". Como o S2 está em `@layer`, uma classe própria fora de camada passa
  por cima. Isso é inferência da regra de cascata documentada no item 2.
  Fontes: <https://react-spectrum.adobe.com/Provider.md>,
  <https://github.com/adobe/react-spectrum/pull/8109>,
  <https://github.com/adobe/react-spectrum/blob/main/packages/dev/s2-docs/src/S2FAQ.tsx>.
- **Caminho oficial fora desses limites:** montar o componente sobre `react-aria-components` e
  estilizar com o macro `style` (que aceita conditions de render props como `isHovered` e
  `isSelected`, e utilitários como `focusRing()`). Para o que o macro não cobre, existe o
  escape-hatch `css()` ("should be used sparingly"), que injeta CSS cru dentro de uma `@layer`.
  Fontes: <https://react-spectrum.adobe.com/styling.md>, <https://react-spectrum.adobe.com/style-macro.md>.

### 6. Fonte

- **Quem carrega:** o `Provider` renderiza sempre o `<Fonts />`, que importa `font-faces.css`.
  - Esse CSS declara `@font-face` para `adobe-clean-spectrum-vf` (pesos 100–900),
    `adobe-clean-spectrum-srf-vf`, `adobe-clean-arabic`, `adobe-clean-hebrew` e `source-code-pro`.
    **Todas as 48 URLs são `https://use.typekit.net/...`** (Adobe Fonts), com
    `font-display: auto`.
  - Para CJK, o `Provider` injeta `<script src="https://use.typekit.net/<kit>.js">`.
  - Para os demais idiomas (incluindo `pt`), chama `ReactDOM.preload(<woff2 do Adobe Clean em
    use.typekit.net>)`, porque `preloads[locale.language] || preloads.en` cai no `en`.

  Fontes: <https://github.com/adobe/react-spectrum/blob/main/packages/@react-spectrum/s2/src/Provider.tsx>,
  <https://github.com/adobe/react-spectrum/blob/main/packages/@react-spectrum/s2/src/Fonts.tsx>,
  <https://github.com/adobe/react-spectrum/blob/main/packages/@react-spectrum/s2/src/font-faces.css>,
  <https://react-spectrum.adobe.com/releases/v0-10-0.md>.
- **Offline:** não há cópia local da fonte no pacote, só URLs da Typekit. Sem rede, a requisição
  falha e vale a pilha de fallback do token `sans`:
  `var(--s2-font-family-sans, adobe-clean-spectrum-vf), adobe-clean-variable, adobe-clean, ui-sans-serif, system-ui, sans-serif`.
  No Windows, `system-ui` costuma ser Segoe UI; isso é suposição, não confirmada.
  **Mesmo com a família trocada, o `preload` para `use.typekit.net` continua sendo emitido**, porque
  é incondicional no `Fonts.tsx`.
  Fontes: <https://github.com/adobe/react-spectrum/blob/main/packages/@react-spectrum/s2/style/spectrum-theme.ts> (linhas 685–696),
  <https://github.com/adobe/react-spectrum/blob/main/packages/@react-spectrum/s2/src/Fonts.tsx>.
- **Licença para embutir:** a documentação da Adobe para desenvolvedores diz que "Adobe Clean, Adobe
  Clean Serif, and Adobe Clean Han are restricted fonts… for the exclusive use of Adobe products and
  software". Uma engenheira do Spectrum, numa discussão pública, completou: "I am an engineer and
  cannot speak to the legality around the use of the proprietary font" e "You can override this
  stack and provide your own definitions". **Embutir a Adobe Clean no bundle de um app não-Adobe
  não tem permissão publicada.** Não achei licença formal da fonte para terceiros.
  Fontes: <https://developer.adobe.com/app-builder/docs/resources/spectrum-intro/lesson1>,
  <https://github.com/adobe/spectrum-web-components/discussions/4364>.
- **Trocar a família:**
  - No macro, `fontFamily` só aceita os tokens `'sans' | 'serif' | 'code'`.
  - O token `sans` compila para `var(--s2-font-family-sans, adobe-clean-spectrum-vf), …`, e
    essa variável está presente no CSS compilado da 1.8.2 (ex.: 5 ocorrências em `Button.css`).
    Então definir `--s2-font-family-sans: 'Source Sans 3'` num ancestral troca a família dos
    componentes. **A variável não aparece na documentação**: está no código, não no contrato
    público.
  - O token `code` é fixo (`source-code-pro, "Source Code Pro", Monaco, monospace`, sem
    variável). O JetBrains Mono do protótipo não entra por token.

  Fontes: <https://react-spectrum.adobe.com/style-macro.md>,
  <https://github.com/adobe/react-spectrum/blob/main/packages/@react-spectrum/s2/style/spectrum-theme.ts>,
  <https://cdn.jsdelivr.net/npm/@react-spectrum/s2@1.8.2/dist/private/Button.css>.
- As métricas tipográficas do S2 assumem a Adobe Clean Spectrum VF ("slightly updated metrics
  allowing for better vertical centering"). O efeito visual de outra família nos componentes não
  foi verificado.
  Fonte: <https://react-spectrum.adobe.com/releases/v0-10-0.md>.

### 7. Ícones

- Ficam dentro do próprio `@react-spectrum/s2` (sem pacote à parte), com **um módulo por ícone**:
  `import Edit from '@react-spectrum/s2/icons/Edit'` (export `./icons/*`). São 412 workflow icons
  na lista da doc e 417 `.mjs` no pacote, de 1,2 a 3,4 KB cada, sem minificar. A doc os chama de
  "open source icons", e a licença do pacote é Apache-2.0.
  Fontes: <https://react-spectrum.adobe.com/icons.md>,
  <https://data.jsdelivr.com/v1/packages/npm/@react-spectrum/s2@1.8.2?structure=flat>.
- **Os ícones do protótipo estão todos cobertos:** casa → `Home`; conversa → `Chat` / `Comment`;
  enviar → `Send` / `ArrowUpSend`; copiar → `Copy`; baixar → `Download`; zoom → `ZoomIn`,
  `ZoomOut`, `ZoomFitToScreen`; fechar → `Close`; engrenagem → `Settings`; pasta → `Folder` /
  `FolderOpen`. Também existem: `Add`, `Edit`, `Delete`, `Code`, `Refresh`, `MenuHamburger`,
  `More`, `Minimize`, `Maximize`, `Star`, `Attach`, `History`, `Clock` e `ChevronDown`.
  **Não encontrei** ícone de sidebar/painel, de "restaurar janela" nem de sol/lua.
  Fonte: <https://react-spectrum.adobe.com/icons.md>.
- O estilo vem pelo macro `iconStyle({size, color})`, com tamanhos XS/S/M/L/XL = 14/16/18/20/22 px
  e cores semânticas. Ícones próprios precisam de SVG 20×20, `viewBox="0 0 20 20"` e fill
  `var(--iconPrimary, #222)`, convertidos com o CLI `@react-spectrum/s2-icon-builder` (fora do
  Parcel).
  Fonte: <https://react-spectrum.adobe.com/icons.md>.

### 8. Movimento

Contagem feita no CSS compilado da 1.8.2 (`dist/private/*.css`) e no código-fonte:

- **Toast:** anima com **View Transitions API** (`document.startViewTransition`), com 400 ms por
  grupo. **Respeita `prefers-reduced-motion`**: `useMediaQuery('(prefers-reduced-motion)')` liga
  a classe `reduceMotion`, e aí, nas palavras do código, "With reduce motion enabled, it fades by
  default".
  Fontes: <https://github.com/adobe/react-spectrum/blob/main/packages/@react-spectrum/s2/src/Toast.tsx>,
  <https://cdn.jsdelivr.net/npm/@react-spectrum/s2@1.8.2/src/Toast.module.css>.
- **Tabs** (indicador de seleção: `translate`, `width`, `height` em 0,2 s), **Disclosure** e
  **SideNavPanel** (`width` em 0,3 s) têm
  `@media (prefers-reduced-motion: reduce) { transition-property: none }`. O SideNavPanel
  documenta: "The panel's width animates first, and its contents animate only once it has settled".
  Fontes: <https://cdn.jsdelivr.net/npm/@react-spectrum/s2@1.8.2/dist/private/Tabs.css>,
  <https://cdn.jsdelivr.net/npm/@react-spectrum/s2@1.8.2/dist/private/SideNavPanel.css>,
  <https://react-spectrum.adobe.com/SideNavPanel.md>.
- **Overlays sem regra de movimento reduzido no CSS compilado:**
  - Modal (Dialog): `opacity, translate`, 0,25 s / 0,13 s, com delay de 0,16 s.
  - Popover: `opacity, translate`, 0,2 s.
  - Menu: `transform…`, 0,15 s.
  - Tooltip: 0,2 s.

  Não achei `prefers-reduced-motion` nesses arquivos; se isso é tratado por JS em outro lugar, não
  confirmei. `Dialog` expõe `isEntering` ("Focus is not moved into the dialog until the animation
  completes").
  Fontes: <https://cdn.jsdelivr.net/npm/@react-spectrum/s2@1.8.2/dist/private/Modal.css>,
  <https://cdn.jsdelivr.net/npm/@react-spectrum/s2@1.8.2/dist/private/Popover.css>,
  <https://react-spectrum.adobe.com/Dialog.md>.
- O ProgressCircle usa `@keyframes`. O macro aceita `transition` (`'default'`, `'colors'`,
  `'opacity'`, `'shadow'`, `'transform'`, `'all'`, `'none'`) e timing functions
  `'default' | 'linear' | 'in' | 'out' | 'in-out'` para superfícies próprias.
  Fonte: <https://react-spectrum.adobe.com/style-macro.md>.

### 9. Peso

- **Tree-shaking por componente:** desde a 1.3.0 há sub-path imports por componente
  (`@react-spectrum/s2/Menu` inclui `Menu`, `MenuItem` etc.), com "Smaller bundles without relying
  on tree-shaking". O pacote declara `"sideEffects": ["*.css"]`. Na mesma versão, a consolidação em
  `react-aria`/`react-stately` reduziu "the number of dependencies installed… by over 90%" e
  melhorou o tree shaking.
  Fontes: <https://react-spectrum.adobe.com/releases/v1-3-0.md>,
  <https://cdn.jsdelivr.net/npm/@react-spectrum/s2@1.8.2/package.json>.
- **CSS atômico:** "Each property/value pair is emitted once and reused everywhere". A doc recomenda
  um único chunk de CSS para S2 e macros, porque "loading all CSS up front results in smaller bundles
  instead of producing duplication between pages".
  Fontes: <https://react-spectrum.adobe.com/styling.md>, <https://react-spectrum.adobe.com/getting-started.md>.
- **Ordem de grandeza (arquivos brutos, sem minificar e sem as deps `react-aria`/`react-stately`):**
  - Os módulos dos ~29 componentes do inventário somam ~508 KB de `.mjs` e ~183 KB de `.css`
    (ex.: Menu 57,6 KB + 15,7 KB; Picker 44,7 + 12,0; Button 31,3 + 13,6; Toast 26,1 + 6,7 + 3,8;
    Tabs 29,4 + 6,7).
  - Todo o CSS de componentes do pacote soma ~463 KB bruto, com duplicação que o lightningcss
    deduplica.
  - O pacote inteiro tem 54 MB desempacotados e 14.178 arquivos; a maior parte é `.map`, `.cjs` e
    ilustrações, que não entram no bundle.
  - As strings de i18n ficam em `dist/private/intl/<locale>.mjs` (~7–8 KB por locale), daí o
    `optimize-locales-plugin`.
  - **O tamanho real minificado + gzip no app não foi medido.** Fica para o build de verdade.

  Fontes: <https://data.jsdelivr.com/v1/packages/npm/@react-spectrum/s2@1.8.2?structure=flat>,
  <https://registry.npmjs.org/@react-spectrum/s2/1.8.2>.

---

## Não confirmado

- Se `unplugin-parcel-macros` 0.2.0 (que usa `unplugin ^1.9.0`) roda sem ressalvas no **Vite 7**:
  a doc não fixa versão do Vite, e o exemplo oficial já está no Vite 8. Precisa de build real
  no app.
- O comportamento do macro em `vite dev`, com HMR e CSS injetado, dentro do WebView2 do Wails.
- O tamanho real do bundle (min+gz) com os componentes do inventário.
- Se os overlays (Modal, Popover, Menu, Tooltip) reduzem movimento por algum caminho JS: o CSS
  compilado não tem a media query.
- Uma licença formal da Adobe Clean para terceiros. Só achei a restrição "exclusive use of Adobe
  products".
- Se `--s2-font-family-sans` é contrato estável: está no código-fonte e no CSS compilado da 1.8.2,
  mas não na documentação.
- Se há plano de API de tema/acento no S2: não achei issue ou discussão oficial sobre isso.
- A fonte que `system-ui` resolve no WebView2/Windows (presumivelmente Segoe UI).
