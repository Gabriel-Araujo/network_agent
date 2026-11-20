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
   vence o S2. Sub-paths por componente pedem `moduleResolution: "bundler"`. **Medido no app:** roda
   no Vite 7 e no Vite 8 (Rolldown) com a mesma config; no 8, `rollupOptions` e `manualChunks` funcionam
   mas estão deprecados. Nos dois, o `vite dev` precisa de `optimizeDeps.exclude` no
   `@react-spectrum/s2/style`. Popover do S2 fica atrás de overlay legado com `z-index` (o overlay legado já foi apagado).
3. **Inventário:** os controles padrão existem. Faltam: `Tab` com fechar, Toast com título +
   subtítulo/ícone (o conteúdo é uma string), Toolbar, Drawer/Sheet, seções no ListView. O RAC
   (`react-aria-components`, sem estilo) tem Toolbar, GridList com seções e Sheet.
4. **Tema:** `colorScheme` é `'light' | 'dark'`; "sistema" é omitir o prop. O acento `#3B63FB` é o
   padrão do S2. **Não há API para trocar o acento**: as cores compilam para hex em build.
5. **Customização:** o prop `styles` aceita só layout. Visual próprio = RAC + macro `style`.
6. **Fonte:** a Adobe Clean só vem da Typekit (`use.typekit.net`) e é restrita a produtos Adobe. A
   família troca por `--s2-font-family-sans` (não documentada). O `Provider` emite um `preload` para a
   Typekit sempre, mesmo com a família trocada. **Medido no app:** sem a Typekit, saem 49 requisições e
   a fonte cai na Segoe UI; um CSP `font-src 'self' data:` barra todas no próprio navegador.
7. **Ícones:** um módulo por ícone em `@react-spectrum/s2/icons/<Nome>`. Faltam sidebar/painel,
   restaurar janela e sol/lua.
8. **Movimento:** Toast, Tabs, Disclosure e SideNavPanel respeitam `prefers-reduced-motion`; Modal,
   Popover, Menu e Tooltip não têm a regra no CSS compilado.
9. **Peso:** sub-path por componente e CSS atômico. **Medido no app:** `Provider` + `Button` + `Picker`
   somam +106 KB gz de JS e +9,6 KB gz de CSS; os 34 locales vêm junto (restringir a `pt-BR` tira 27
   KB gz).

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

**Método dos fatos _Medido no app_ (2026-11-19).** S2 1.8.2 instalado em `cmd/ui/frontend` (branch
`frontend-v2`), Vite 7.3.6, `wails dev` do Wails 2.15.0, WebView2 157.0.4341.60. Um `Button` e um
`Picker` dentro de um `Provider` foram renderizados no modal de Configurações. O Wails zera
`WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS` e as chaves de registro equivalentes
(`preventEnvAndRegistryOverrides` no `go-webview2`), então a janela não abre porta de depuração.
Por isso, rede, fontes e estilos computados foram lidos por CDP num Edge 157.0.4341.60 headless
(mesmo motor e mesma versão) apontado para o servidor do `wails dev` (`localhost:34115`). A janela
real do WebView2 foi conferida por captura de tela.

**Método dos fatos _Medido no app: Vite 8_ (2026-11-19).** Mesmo S2 1.8.2, Vite 8.3.8 com Rolldown
1.2.19, `@vitejs/plugin-react` 5.2.0, `vitest` 5.1.0. Um `Button` dentro de um `Provider`, com um
`style()`, foi montado no `main.tsx` e tirado depois. O `vite dev` foi carregado num Edge headless
(`--dump-dom`) com o cache `node_modules/.vite` apagado antes, e o log do Vite lido ao fim.

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
- **Medido no app: Vite 7.** `unplugin-parcel-macros` 0.2.0 com a config oficial acima builda no
  Vite 7.3.6 sem aviso, e o `s2-styles` sai como chunk de CSS próprio. O `npm install` do S2 1.8.2
  trouxe 15 pacotes; o do plugin e do `lightningcss` 1.33.0, outros 15.
- **Medido no app: Vite 8.** A mesma config builda no Vite 8.3.8 (Rolldown 1.2.19) sem aviso, e o
  `s2-styles` sai como chunk de CSS próprio, com as camadas `_.a`…`_.h` do S2 e o CSS do macro. Os
  peers aceitam o 8 sem subir nada: `@vitejs/plugin-react` 5.2.0 (`vite ^4 … ^8`), `vitest` 5.1.0
  (`^6.4.0 || ^7.0.0 || ^8.0.0`); `unplugin-parcel-macros` 0.2.0 não declara peer. O exemplo oficial
  `s2-vite-project` já usa `vite ^8.0.0` (ver acima).
  - **Deprecado no 8, ainda funciona:** `build.rollupOptions` é alias de `build.rolldownOptions`
    (`@deprecated Use rolldownOptions instead`), e `output.manualChunks` é
    `@deprecated Please use output.codeSplitting instead`; se os dois estiverem definidos, o
    `manualChunks` é ignorado. A config oficial do S2 (acima) ainda usa os nomes antigos.
    Fontes: `node_modules/vite/dist/node/index.d.ts` 8.3.8,
    `node_modules/rolldown/dist/shared/define-config-*.d.mts` 1.2.19.
  - **`vite dev` no 8:** o scan de dependências agora roda no Rolldown e falha igual, com outra
    mensagem: `"./style" is not exported under the conditions ["module", "browser", "development",
    "import"]`. Sem o `optimizeDeps.exclude`, o log termina em `optimized dependencies changed.
    reloading`; com ele, a página carrega sem recarregar e o macro compila em dev (o `className` sai
    como classes atômicas).
- **Medido no app: `vite dev`.** O scan de dependências do Vite 7 (esbuild) falha com
  `No known conditions for "./style" specifier in "@react-spectrum/s2"`. O motivo é que o `exports`
  do `./style` só tem a condição `node`, porque ele roda no build, via macro. O Vite então pula o
  pré-bundling e recarrega a página na primeira carga. `optimizeDeps.exclude:
  ['@react-spectrum/s2/style']` resolve. Com isso, o macro funciona no `wails dev`: editar um
  `style({...})` chega por HMR, sem recarregar a página nem perder estado.
- **Medido no app: TypeScript.** Com `moduleResolution: "Node"`, os sub-paths dão `TS2307` ("There
  are types at '…/dist/types/exports/Button.d.ts', but this result could not be resolved under your
  current 'moduleResolution' setting"). Com `"Bundler"`, passam. Isso confirma a inferência acima.
- **Medido no app: `@swc/core`.** O `postinstall` só checa se o binário nativo carrega e, se não
  carregar, instala `@swc/wasm`. O binário nativo (`@swc/core-win32-x64-msvc`) chega por
  `optionalDependencies` e carrega sem o script, então negar o script no `allowScripts`
  (`npm install-scripts deny @swc/core`) não quebra o build. Fonte:
  `node_modules/@swc/core/postinstall.js` 1.16.22.
- **Medido no app: CSS legado em `@layer`.** Com `@layer legacy, _;` seguido de `@layer legacy {…}`,
  o `lightningcss` reescreve a saída para `@layer legacy{…}` + `@layer _;`, e a ordem (legado abaixo
  do S2) se mantém. Nenhum conflito visível dentro dos controles: o `button`/`*` legado não alterou
  padding, raio nem fonte do `Button`/`Picker`.
- **Medido no app: popover atrás de overlay legado.** O popover do `Picker` vai num portal no fim do
  `<body>`, com `position: absolute` e `z-index: auto`. Dentro de um overlay legado
  `position: fixed; z-index: 100` (o `.settings-scrim`), ele pinta **atrás** do overlay e fica
  invisível. Ao abrir, o S2 põe `inert` no `#root`, e por isso o clique atravessa o overlay e acerta
  as opções invisíveis. Com `z-index` alto no popover, ele aparece normalmente. Visto no Edge
  headless e na janela real do WebView2.

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
| Botões da janela (min/max/fechar) | ActionButton `isQuiet` + ícone `Close` | Não há componente de chrome de janela nem glifos de janela: `Minimize` e `Maximize` são setas de encolher e expandir (medido no app, 1.8.2, 2026-11-19), e não há "restaurar". |
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
  `data-color-scheme` no `<html>`. **Medido no app:** re-renderizar o `Provider` com outro
  `colorScheme` troca as cores na hora, sem recarregar. O `color-scheme` computado acompanha
  (`dark`/`light`), e o `background="layer-1"` vale `rgb(27, 27, 27)` no escuro e
  `rgb(248, 248, 248)` no claro.
  Fontes: <https://github.com/adobe/react-spectrum/blob/main/packages/@react-spectrum/s2/src/page.macro.ts>,
  <https://cdn.jsdelivr.net/npm/@react-spectrum/s2@1.8.2/page.css>.
- **Fundos empilhados no macro, medido no app (S2 1.8.2, 2026-11-20, `getComputedStyle`):**
  `backgroundColor: 'layer-1'` vale `rgb(248, 248, 248)` no claro e `rgb(27, 27, 27)` no escuro, e
  `'layer-2'` vale `rgb(255, 255, 255)` e `rgb(34, 34, 34)`: o `layer-2` é mais claro que o `layer-1`
  nos dois temas, a mesma relação do protótipo (`#F4F4F5`/`#FFFFFF` e `#1A1B1E`/`#202125`), então
  moldura em `layer-1` e cartão em `layer-2` dão o efeito de cartão sem outro par de tokens.
  `borderColor: 'gray-300'` vale `rgb(218, 218, 218)` no claro e `rgb(57, 57, 57)` no escuro. O raio
  `'xl'` é 16 px (`corner-radius-extra-large-default`), igual ao `--radius-xl` do protótipo; a
  escala é `sm` 4, `default` 8, `lg` 10 e `xl` 16 (`style/spectrum-theme.ts`).
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

- **Medido no macro (1.8.2, 2026-11-19):** chaves de condição que começam com `:` viram
  pseudo-classe e as que começam com `@` viram at-rule (`':hover'`, `'@media (max-width: 1079px)'`),
  além das condições de runtime (`isX`). Custom property é `'--nome': {type: '<propriedade>', value}`;
  com `type: 'fill'` os tokens de texto (`neutral`) não passam no tipo, com `type: 'color'` passam.
  `transition: '[width]'` + `transitionDuration: 300` animam a largura, como faz o `SideNavPanel`.
  O `outlineColor` aceita `focus-ring`, o `borderColor` não. Fonte: `style/style-macro.ts`
  (`compileCondition`) e `src/SideNavPanel.tsx` do pacote.
  - `alignItems` usa `start`/`end` (não `flex-start`/`flex-end`), e o espaçamento (`gap`,
    `padding*`) só aceita a escala de tokens (2, 4, 8, 12, 16, …); `7`, `10` e `14` não compilam no
    TS do macro. Medido no app ao montar a Início, 2026-11-19.
- **RAC `GridList` com um `input` na linha (react-aria-components 1.22.1, medido no app):** um `ref`
  + `useEffect` que chama `focus()` não foca o input da tela, porque a coleção também monta os itens
  numa cópia fora do DOM; `autoFocus` foca. Com `keyboardNavigationBehavior="tab"` as setas não são
  capturadas pela linha, e um `stopPropagation` no `onKeyDown` e no `onPointerDown` do input tira as
  teclas (setas, espaço, typeahead) e o clique da grade.
- **RAC `Tabs` com os painéis montados (react-aria-components 1.22.1, medido no app):** `TabPanel`
  aceita `shouldForceMount`; o painel fora de foco recebe `inert`, e o `className` recebe `isInert`.
  O slide do indicador do S2 não é uma classe por aba: o `Tab` põe um `SelectionIndicator` (um
  `SharedElementTransition` chamado `SelectionIndicator`) dentro de cada aba, animando
  `translate`/`width`/`height` em 0,2 s com easing `out`. O `Tab` segue sem prop de fechar; o "x"
  do app fica dentro da aba, mas fora do `usePress` dela (o `pointerdown`/`click` param a
  propagação) e fora da ordem de Tab.
  Fontes: `@react-spectrum/s2/src/Tabs.tsx` e
  `react-aria-components/dist/private/SelectionIndicator.js` (1.22.1).
  **Medido no app (2026-11-19):** o `SelectionIndicator` ficou na aba anterior quando uma aba foi
  criada e selecionada no mesmo commit; a aba nova só entra no DOM da coleção depois do render, e o
  `SharedElementTransition` não recebe o novo elemento a tempo. O `TabBar` usa um `span` único,
  posicionado pela aba selecionada (`data-tab`) e reposicionado por um `MutationObserver` no
  `TabList` quando a aba entra.
- **O macro aceita `gridArea` no tipo, mas não emite a regra** (a classe sai vazia; dois painéis
  num grid auto-posicionado ganham uma linha cada). O que funciona para empilhar na mesma caixa é
  `position: 'relative'` no pai e `position: 'absolute'` + `inset: 0` no filho; os dois compilam.
  Fonte: medido no app, 2026-11-19 (`npm run build`, CSS gerado).
- **Macro `style`, medido na Conversa (S2 1.8.2, 2026-11-19, `tsc` e `npm run build`):**
  - A escala de espaço (`padding`, `margin`, `gap`) só aceita 0, 2, 4, 8, 12, 16, 20, 24, 28, 32, 36,
    40, 44, 48, 56, 64, 80, 96 (e `-N`); 6, 10, 14, 18 e 22 são erro de tipo. `borderWidth` só aceita
    0, 1, 2, 4 (3 é erro).
  - `borderStyle: 'solid'` sem largura vale a do navegador (3 px) nos lados não declarados: com só
    `borderBottomWidth: 1`, os outros três lados saem grossos. Declare `borderWidth: 0` antes.
  - Uma condição própria (`isOpen`) ou `isFocusVisible` faz o `style(...)` devolver uma função, que se
    chama com as props (`panel({isOpen})`), e o `className` deixa de ser uma `string`. Para o foco de
    teclado numa superfície própria sem render props, a condição de pseudo-classe `':focus-visible'`
    mantém o `style(...)` estático (é a que a Conversa usa, e foi conferida ao vivo).
  - O macro não tem `font-style`, `tab-size` nem `accent-color`. O `font: 'code-*'` fixa o `font-family`
    no token `code`; para outra família, use `fontSize`/`lineHeight` do token e uma classe de `css()`.
  - Seletor aninhado (`& > :first-child`) não é chave do macro: vai por `css()`.
  - `ToastQueue.positive(msg)` sem `timeout` **não some sozinho**; a confirmação de cópia passa
    `{timeout: 5000}` (o mínimo).
  - `InlineAlert` dimensiona pelo conteúdo; para ocupar a coluna, `styles={style({width: 'full'})}`.
  - **Custom property de cor no macro** (S2 1.8.2, medido no Bloco de diagrama, 2026-11-19):
    `'--x': {type: 'backgroundColor' | 'color' | 'borderColor', value: 'gray-400'}` compila, e
    `{type: 'fontFamily', value: 'sans'}` também. O build do lightningcss troca o `light-dark(…)`
    por `var(--lightningcss-light, …) var(--lightningcss-dark, …)`, e a variável segue o tema
    sem renderizar de novo. Uma custom property guarda o valor cru; para o valor resolvido
    (arquivo exportado, por exemplo), um elemento de prova com `color: var(--x)` e
    `getComputedStyle(…).color` devolve o rgb do tema em uso. O `css()` aceita aninhamento
    (`& > svg {…}` sai como `.classe > svg`). O `ToggleButton` S2 dentro de `TooltipTrigger` e o
    `ActionButton` dentro do `ActionButtonGroup` com `Tooltip` funcionam; o `ActionButtonGroup`
    não tem seleção, então o botão de alternar é um `ToggleButton` ao lado.
  - **Cor de texto e tamanho de código no macro, medido na gaveta de Logs** (S2 1.8.2, 2026-11-19, `tsc`):
    `positive` e `notice` valem para `backgroundColor`, mas não para `color` (erro de tipo); no texto,
    use `positive-900` e `notice-900` (`negative` vale nos dois). A cor por variante sai de
    `style<{level: 'a' | 'b'}>({color: {level: {a: …, b: …}}})`, chamado como `fn({level})`. O
    `fontSize: 'code-sm'` resolve para 14 px e o `'code-xs'` para 12 px. O `Link` sem `href`
    renderiza um `span role="link"` que responde ao `onPress`, e o `isDisabled` não existe nele: sem
    ação, não o renderize.
  - **Modal de Configurações, medido** (S2 1.8.2, 2026-11-20, `tsc`, `npm run build` e ao vivo):
    - `Dialog size="XL"` tem 960 px, `maxWidth: 90vw` e `maxHeight: 90%`. O `DialogContainer` aceita
      `false` como filho (fechado), e o `isDismissible` põe o fechar (aria-label "Descartar" em pt-BR)
      e tira os botões; o clique fora é do `isDismissable` do `Modal` (repassado do `isDismissible`, lido
      no código-fonte), e o Esc é do `isKeyboardDismissDisabled`. O Esc fechou o modal ao vivo.
    - `Tabs orientation="vertical"` fica numa linha (lista e painel), a aba tem `minHeight` 48 e a
      lista `marginStart` 12 e `marginEnd` 20; o `aria-label` é obrigatório (lança erro). O colapso em
      Picker é só do horizontal. O `TabPanel` inativo é desmontado.
    - `SelectBoxGroup`: o cartão é de 170 px (e a lista centraliza). Só o que passa pelo
      `IllustrationContext` (ícone ou ilustração S2) cai na área `illustration`; um `div` próprio
      precisa de `grid-area: illustration` por `css()`. `selectedKeys` aceita um array e
      `aria-labelledby` serve de rótulo.
    - Um `Provider` aninhado com `colorScheme="light"` ou `"dark"` pinta o filho com os tokens daquele
      esquema, qualquer que seja o tema do app (serve para prévia de tema).
    - `ListView` é virtualizado: sem altura definida não renderiza as linhas. Uma linha com nome e
      descrição mede 52 px. Um `Badge` dentro do `Text slot="label"` quebra a linha e a leva a 76 px;
      num `span` `inline-block` com `marginY: -4` ela fica em 52. O `ListViewItem` aceita um
      `ActionButtonGroup` como filho (vira `isQuiet`, `size="S"`).
    - `NumberField.onChange` devolve `NaN` com o campo vazio.
    - `scrollbar-width` não herda: `html { scrollbar-width: thin }` não alcança os painéis que rolam,
      e o `*` alcança. No Windows o `thin` encolhe a barra, mas as setas das pontas continuam.
    - No macro, `insetStart` (lógica) e um `css('left: auto')` (física) não se anulam: duas metades
      absolutas, uma com `insetStart: 0` e outra com `left: auto; right: 0`, ficaram as duas à
      esquerda. Declare `insetStart` numa e `insetEnd` na outra.
  - **Macro no vitest 5** (`vitest` 5.1.1, `unplugin-parcel-macros` do `package.json`, medido em
    2026-11-19): o `fsModuleCache` (padrão `true`) reaproveita o módulo transformado entre rodadas, e
    o macro não roda de novo, então o `macro-<hash>.css` que ele registra em memória não existe. O
    teste falha com `Cannot find module '/@id/macro-….css'` a partir da 2ª rodada. Por isso,
    `test.fsModuleCache: false`.

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
  No WebView2/Windows 11, `system-ui` resolve para a Segoe UI (medido, ver abaixo).
  **Mesmo com a família trocada, o `preload` para `use.typekit.net` continua sendo emitido**, porque
  é incondicional no `Fonts.tsx`.
  Fontes: <https://github.com/adobe/react-spectrum/blob/main/packages/@react-spectrum/s2/style/spectrum-theme.ts> (linhas 685–696),
  <https://github.com/adobe/react-spectrum/blob/main/packages/@react-spectrum/s2/src/Fonts.tsx>.
- **Medido no app: rede e fallback.**
  - **Online:** o `Provider` insere
    `<link rel="preload" as="font" href="https://use.typekit.net/af/ca4cba/…">` e a Adobe Clean
    Spectrum VF carrega da Typekit. É um único woff2 variável para todos os pesos.
  - **Typekit inalcançável** (URLs bloqueadas por CDP): o navegador tenta 49 URLs, o preload mais os
    16 `@font-face` × 3 formatos (`l`/`d`/`a`). Todas falham, e o `document.fonts` marca
    `adobe-clean-spectrum-vf` como `error`. A fonte renderizada (`CSS.getPlatformFontsForNode`) é a
    **Segoe UI**, o que confirma o `system-ui` no WebView2/Windows 11.
  - **Bloqueio:** a requisição não tem como sair do `Provider` por API. Um
    `<meta http-equiv="Content-Security-Policy" content="font-src 'self' data:">` no `index.html`
    faz o próprio navegador barrar as 49 (`blockedReason: csp`), nada sai para a rede e o fallback é
    o mesmo.
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
- **Medido no app (S2 1.8.2, WebView2, 2026-11-19, por CDP):** `--s2-font-family-sans: 'Source Sans 3'`
  no `:root` basta, porque o CSS compilado lê a variável por `var()` em cada regra de fonte (ex.:
  `dist/private/ActionButton.css`) e ela herda do documento. Com a Source Sans 3 e a JetBrains Mono
  embutidas e o CSP `font-src 'self' data:`, o `document.fonts` mostra as duas como `loaded` e as
  famílias da Adobe Clean como `unloaded`. As 49 URLs da Typekit continuam aparecendo no Resource
  Timing, todas com `responseStatus` 0, 0 bytes e duração 0, ou seja, barradas antes de sair.
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
- **Medido no app (2026-11-19, `@react-spectrum/s2-icon-builder` 1.1.2):** o binário é
  `transform-icons` (`npx transform-icons -i <svg> -o <pasta>`, `-i` repetível). Ele roda o Parcel
  e gera um `.tsx` por SVG, com `createIcon` do `@react-spectrum/s2`; o fill
  `var(--iconPrimary, #222)` sai como `var(--iconPrimary, light-dark(rgb(41, 41, 41), rgb(219, 219, 219)))`.
  O componente aceita `styles={iconStyle(…)}` como os do S2. Fonte: `index.js` do pacote.
- **Ícone de fora do S2 (medido, 2026-11-19):** o `@jetbrains/icons` 5.25.0 (Apache-2.0) passa pelo
  mesmo `transform-icons` sem ajuste, desde que o `fill="currentColor"` seja trocado por
  `var(--iconPrimary, #222)`. Fonte: `npm pack @jetbrains/icons`.
- **`ActionButton` com dois ícones** (ícone + `Text` + `ChevronDown`, 1.8.2): só o primeiro aparece
  (medido no app, 2026-11-19). Um chevron de "abre menu" não cabe no slot de ícone.

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
- **Medido no app (Vite 7.3.6, `npm run build`, gzip do relatório do Vite):** `Provider` + `Button`
  + `Picker` + um `style()` acrescentaram ao bundle:
  - JS: 1.193,05 → 1.551,61 KB (+358,6 KB); gzip 413,46 → 519,68 KB (+106,2 KB).
  - CSS: chunk novo `s2-styles` de 49,01 KB (gzip 9,58 KB). O CSS legado passou de 34,41 para
    33,27 KB por causa da troca de minificador para o `lightningcss`.
  - Os 34 locales do S2 entram no bundle. Com `@react-aria/optimize-locales-plugin`
    (`locales: ['pt-BR']`, `enforce: 'pre'`), o JS cai para 1.450,36 KB (gzip 492,62 KB), ou seja,
    −101,3 KB e −27,1 KB gz.
  - Sem nenhum componente importado, o S2 instalado não pesa: o tree-shaking o tira inteiro.
- **Medido no app (Vite 8.3.8 / Rolldown 1.2.19, mesma medida):**
  - Sem S2 importado, o JS sai menor que no 7: 1.167,15 KB (gzip 396,89 KB), contra 1.193,05 (413,46).
    O CSS legado fica em 33,26 KB.
  - `Provider` + `Button` + um `style()` (sem o `Picker`, então não comparável à medida do 7): JS
    1.325,70 KB (gzip 439,98 KB, +43,1 KB gz), `s2-styles` 25,48 KB (gzip 4,56 KB).

  - `Provider` + `ToastContainer` (2026-11-19, `@react-aria/optimize-locales-plugin` 2.1.0 pelo
    `optimizeLocales.vite()`): JS 1.327,11 KB (gzip 446,38 KB) com o filtro, 1.426,59 KB (gzip
    469,34 KB) sem ele, ou seja, −99,5 KB e −23,0 KB gz. `s2-styles` 43,72 KB (gzip 7,27 KB).

  Fontes: <https://data.jsdelivr.com/v1/packages/npm/@react-spectrum/s2@1.8.2?structure=flat>,
  <https://registry.npmjs.org/@react-spectrum/s2/1.8.2>.

---

## Não confirmado

- O tamanho do bundle com o inventário inteiro (só `Provider` + `Button` + `Picker` foram medidos).
- Se os overlays (Modal, Popover, Menu, Tooltip) reduzem movimento por algum caminho JS: o CSS
  compilado não tem a media query.
- Uma licença formal da Adobe Clean para terceiros. Só achei a restrição "exclusive use of Adobe
  products".
- Se `--s2-font-family-sans` é contrato estável: está no código-fonte e no CSS compilado da 1.8.2,
  mas não na documentação.
- Se há plano de API de tema/acento no S2: não achei issue ou discussão oficial sobre isso.
