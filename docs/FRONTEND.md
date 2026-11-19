# Frontend (`cmd/ui/frontend`)

O frontend do app desktop: React 19 + TypeScript + Vite, empacotado pelo Wails v2 e renderizado no
WebView2. Conversa com o Go pelas bindings geradas em `cmd/ui/frontend/wailsjs/`.

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
  direto no navegador. Os tokens, o layout e o movimento vêm dele. Ele carrega fontes do Google
  Fonts; o app, não.
- **Quem vence quando o Spectrum e o protótipo discordam:** o Spectrum, nos controles padrão (botão,
  picker, menu, dialog, tabs, toast, campos, radio); o protótipo, no shell e nas superfícies sem
  equivalente (composer, trilho de Perguntas, diagrama, gaveta de Logs).
- **Superfície própria** é montada em `react-aria-components` (ou HTML puro) e estilizada com os
  tokens do S2, não com cores soltas.
- **Nada depende de rede em runtime:** fontes, ícones e assets vão embutidos no bundle.
- **Nenhum CSS global fora de `@layer`:** ele vence o CSS do S2.

## Stack e build

_Pendente: S2 de pé no `wails dev`._ Versões instaladas, a config do Vite (plugin do macro,
`lightningcss`, chunk de CSS), o `tsconfig`, o `allowScripts`, a camada do CSS legado durante a
migração e o tamanho do bundle.

## Tema

_Pendente: Tema._ Claro, escuro e sistema: o padrão, onde a escolha persiste, a cor de fundo da
janela, o tema do diagrama e do SVG baixado.

## Shell e janela

_Pendente: Shell numa janela real._ Titlebar arrastável, WindowControls, sidebar e faixa recolhida,
gaveta de Logs, tamanho mínimo.

## Abas e Conversas

_Pendente: Aba e Conversa._ O que é uma aba e as regras de abrir, focar e fechar. Também o turno numa
aba fora de foco e a persistência entre execuções.

## Componentes

_Pendente: De-para dos elementos do protótipo._ Uma tabela com elemento → origem (S2, RAC ou
própria) → arquivo em `src/`. Também ícones, fonte e movimento.

## Início, barra de status e toast

_Pendente: Início, barra de status e toast._ De onde vêm as sugestões, o que a barra de status mostra
e quais ações disparam toast.

## Verificação

- `npm run build` e `npm test` verdes em `cmd/ui/frontend`.
- `gofmt -l .`, `go vet ./...` e `go build ./...` verdes na raiz.
- Ao vivo em `wails dev` (a partir de `cmd/ui`): Início, Conversa com diagrama e bloco de código,
  Logs e Configurações, conferidas contra o protótipo **nos dois temas**.
