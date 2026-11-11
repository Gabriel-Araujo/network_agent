# Issue tracker: markdown local

Issues, specs e mapas deste repo vivem como arquivos markdown em `.agent/`.

> Este repo **não** usa GitHub Issues nem GitLab Issues. `gh` e `glab` não estão instalados, e o
> tracker é local por decisão explícita. Não crie issues no remoto.

## Convenções

- Uma feature por diretório: `.agent/<feature-slug>/`
- A spec é `.agent/<feature-slug>/spec.md`
- Issues de implementação são um arquivo por ticket em
  `.agent/<feature-slug>/issues/<NN>-<slug>.md`, numerados a partir de `01`, nunca um arquivo
  único combinando todos os tickets
- O estado fica numa linha `Status:` perto do topo de cada arquivo de issue
- Comentários e histórico de conversa são anexados ao fim do arquivo sob um cabeçalho
  `## Comments`
- Artefatos produzidos ao resolver um ticket (relatórios, protótipos) vão em
  `.agent/<feature-slug>/assets/` e são **linkados** a partir da issue, nunca colados nela

`.agent/` **está** no `.gitignore`: nada sob ele é versionado (`git ls-files .agent` devolve
zero), e os tickets vivem na máquina de quem os escreve.

`.agent/` é do tracker, e só dele. Artefato que uma execução grava — espelho, log, histórico —
mora em `~/.network_agent/`, não aqui, e dado de usuário nunca entra. O **Run tree** do ICM
(layer 4, ver `CONTEXT.md`) não tem escritor em lugar nenhum do repo hoje; quando tiver, a raiz
dele não é esta.

## Quando uma skill diz "publish to the issue tracker"

Crie um arquivo novo sob `.agent/<feature-slug>/`, criando o diretório se preciso.

## Quando uma skill diz "fetch the relevant ticket"

Leia o arquivo no caminho referenciado. Normalmente o usuário passa o caminho ou o número.

## Wayfinding operations

Usado pelo `/wayfinder`. O **mapa** é um arquivo com um **arquivo filho** por ticket.

- **Mapa**: `.agent/<effort>/map.md`. Cabeçalhos em pt-BR, nesta ordem: `## Destino`,
  `## Notas`, `## Decisões até agora`, `## Ainda não especificado`, `## Fora de escopo`.
- **Ticket filho**: `.agent/<effort>/issues/NN-<slug>.md`, numerado a partir de `01`, com a
  pergunta no corpo sob `## Questão`. Linhas de controle perto do topo:
  - `Type:` — `research` | `prototype` | `grilling` | `task`
  - `Status:` — `open` (aberto e não reivindicado) | `claimed` (reivindicado, em andamento) |
    `resolved` (respondido)
  - `Blocked by:` — números separados por vírgula, ou `—` se não houver bloqueio
- **Bloqueio**: a linha `Blocked by: NN, NN`. Um ticket está desbloqueado quando **todos** os
  arquivos que ele lista estão `resolved`.
- **Fronteira**: os arquivos em `issues/` que estão `open`, desbloqueados e não reivindicados.
  Em ordem numérica, o primeiro vence.
- **Reivindicar**: gravar `Status: claimed` e salvar **antes** de qualquer trabalho, para que
  sessões concorrentes pulem o ticket.
- **Resolver**: anexar a resposta sob `## Resposta`, gravar `Status: resolved`, e então anexar um
  ponteiro de contexto (gist + link) em `## Decisões até agora` do `map.md`. Um ticket resolvido
  nunca é listado como aberto, e uma decisão vive em exatamente um lugar: o ticket.
- **Referir por nome**: em tudo que o humano lê, cite o ticket pelo **título**, nunca por um id
  solto. O número e o caminho viajam dentro do link, nunca no lugar do nome.
- **Resolver um ticket por sessão**, com exceção dos tickets `research`, que podem ser
  despachados em paralelo.

Convenção de nomenclatura de arquivo: `NN-<slug-em-kebab-case>.md`, slug em português quando o
título é em português.

## Vocabulário de triagem

Ainda não configurado — não há `docs/agents/triage-labels.md`. Se a skill `triage` for usada
neste repo, os rótulos canônicos (`needs-triage`, `needs-info`, `ready-for-agent`,
`ready-for-human`, `wontfix`) precisam ser mapeados para as linhas `Status:` acima antes da
primeira execução.
