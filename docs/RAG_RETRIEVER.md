# Retriever RAG — Documentação da Implementação

> Pipeline de busca híbrida (vetorial + full-text) sobre a base `frr_docs`
> (PostgreSQL + pgvector), alimentado pelo **Classificador** em Modo RAG
> (`internal/skills/classifier`) e por um gerador de queries LLM.
> Saída híbrida: resultados em memória **+** espelho JSON em
> `.agent/tmp/retrieval/`.
> Forma do resultado: `[{"query": string, "protocol": string, "daemon": string, "chunk_type": string, "response": string}]`.

---

## 1. Visão Geral

O Network Agent responde perguntas de redes (configuração, troubleshooting,
conceitos) usando o framework **FRRouting (FRR)** como domínio. A pergunta do
usuário, antes de virar resposta, passa por um pipeline de **RAG
(Retrieval-Augmented Generation)**:

```
pergunta do usuário
   │
   ▼
[1] Classificador (internal/skills/classifier)
   │    • chamada genérica (Modo RAG desligado) — só exibida na tela
   │    • chamada Modo RAG → map[string]string → rag.FiltrosFrom → rag.Filtros
   ▼
[1b] skip rule (deveBuscar) — a busca só roda quando a classificação RAG mudou
   ▼
[2] skill rag-retriever (ESTA IMPLEMENTAÇÃO) ──► queries reescritas pelo LLM
   │                                              → busca híbrida no pgvector
   │                                                (frr_docs) com os filtros
   ▼
[3] resposta final gerada pelo LLM com o <recovered_context> no prompt
```

O **Classificador** escolhe, para cada **Lista** passada na chamada, um valor
canônico — ou `""` quando nada se encaixa. As Listas do chamador RAG são
`protocol`/`daemon`/`chunk_type`/`agent` (fonte única: `rag.Listas()`), e o mapa
de saída vira o **conjunto único de filtros** da busca via `rag.FiltrosFrom` —
que considera só as três primeiras: `agent` é dimensão de **roteamento** (qual
agente responde o input: `builder`/`modifier`/`teacher`), não filtro da
`frr_docs`.
O **briefing rico** (`internal/skills/briefing`) é a skill preservada do antigo
`intent-analyser` e está **fora do fluxo do CLI** (ver §12).

A skill `rag-retriever` é o elo **[2]**: ela reescreve o input em **queries de
busca reais** (gerador LLM, com degradação avisada para o input cru), executa a
recuperação de documentação relevante com os filtros do Classificador e devolve
os resultados **em memória** e **gravados num espelho JSON** em forma
determinística, para que o passo [3] não precise re-reinterpretar a pergunta em
linguagem natural.

---

## 2. Objetivo e Contrato de Saída

O ponto de entrada é:

```go
ragretriever.Do(ctx, input string, filtros rag.Filtros, cfg rag.Config) ([]rag.Result, string, error)
```

Recebe o **input textual** e o **conjunto único de filtros** do Classificador e
gera um `rag.Result` por query:

```json
[
  {
    "query": "OSPF MTU mismatch neighbor adjacency",
    "protocol": "ospf",
    "daemon": "ospfd",
    "chunk_type": "concept",
    "response": "<contexto recuperado da documentação do FRR>"
  }
]
```

- **`query`** — a busca reescrita (em terminologia FRR), pronta para o motor
  de recuperação.
- **`protocol`**, **`daemon`** e **`chunk_type`** — os filtros de metadados
  **da busca inteira** (um conjunto por input, não por query), vindos do
  Classificador. Podem estar vazios ("sem filtro").
- **`response`** — o **contexto recuperado** (`parent_content` + origem:
  `source_url` e `section_path`), **não** uma resposta gerada pelo LLM.
  Gerar a resposta é responsabilidade do passo [3]. Query sem resultado devolve
  `response` vazio, preservando os metadados.

O retorno é **híbrido**: os resultados completos voltam em memória e o mesmo
conteúdo é gravado no espelho `.agent/tmp/retrieval/<slug>-<hash8>.json`
(§7). Se só a gravação do espelho falhar, o valor em memória volta completo,
o path vem vazio e o erro é a sentinela
`ragretriever.ErrEspelhoNaoGravado` (checável com `errors.Is`) — o espelho
nunca derruba o caminho. Falha na busca é erro normal.

Esse contrato é importante por um motivo de **separação de responsabilidades**
(sobre o qual voltamos mais adiante): o retriever só deve trazer evidência;
quem raciocina sobre ela é a etapa de geração.

---

## 3. Por que busca híbrida (vetorial + full-text)?

A tabela `frr_docs` indexa documentação RST do FRR em *chunks* que possuem:

- **`embedding VECTOR(4096)`** com índice **diskann** (busca por similaridade
  semântica/cosseno);
- **`content_tsv TSVECTOR`** com índice **GIN** (busca por correspondência
  léxica/full-text);
- colunas de filtro `daemon`, `protocol`, `chunk_type` com índice btree.

Cada família de busca cobre um caso de falha da outra:

| Característica | Vetorial (`<=>`) | Full-text (`@@`) |
|---|---|---|
| Entende sinonímia/semântica | ✅ | ❌ |
| Casa termos exatos (ex.: nome de comando `redistribute`) | ❌ | ✅ |
| Rápido por índice aproximado (diskann) | ✅ | ✅ (GIN) |
| Sanitiza input (evita erro de sintaxe de tsquery) | — | ✅ (`plainto_tsquery`) |

Uma query como *"OSPF neighbor stuck Exstart"* pode não conter a palavra
exata que aparece no doc, mas é semanticamente próxima (a vetorial acerta);
já *"redistribute"* precisa casar exatamente (a full-text acerta). **Buscar
com as duas fontes e fundir os rankings aproveita os dois índices** e torna o
resultado robusto a variações de vocabulário — o ponto central do RAG.

---

## 4. Arquitetura do Código

A orquestração vive na skill `internal/skills/rag-retriever/`, que carrega
**seus próprios prompts** (`prompt.md` via `//go:embed`). O mecanismo puro de
busca permanece em `internal/agent/rag`:

```
cmd/cli/main.go
   │  classifier.Do(..., modoRAG=false)      → só exibição na tela
   │  classifier.Do(..., modoRAG=true)       → rag.FiltrosFrom → rag.Filtros
   │  deveBuscar(...) → runRetrieval(...) → ragretriever.Do(...)  → memória + espelho
   ▼
internal/skills/classifier/          (Classificador — validação no cliente + retry ×2)
internal/skills/briefing/            (briefing rico — FORA do fluxo do CLI, preservado)
internal/skills/rag-retriever/       (skill — orquestração, geração de queries)
   ├─ retriever.go                   → Do, SaveResults, BuildQueries,
   │                                    LLMQueryGenerator, ParseGeneratedQueries
   ├─ prompt.go / prompt.md          → prompt do gerador de queries (//go:embed)
   ▼
internal/agent/rag/
   ├─ types.go                       → tipos (Config, Filtros, QuerySuggestion,
   │                                    Result, QueryGenerator)
   ├─ listas.go                      → Listas() (fonte única das Listas RAG)
   │                                    e FiltrosFrom
   ├─ constants.go                   → ChunkType*, RetrievalDir, DefaultEmbeddingModel, RrfK
   ├─ retriever/                     (mecanismo de busca — sem IO de arquivo)
   │   ├─ retriever.go               → Retrieve, VectorLiteral, FuseRRF, buildResponse
   │   ├─ embed.go                   → embeddings em lote + ProbeEmbeddingDim (2560)
   │   ├─ db.go                      → SQL híbrido (vetorial + FTS) + lookup de chunks
   │   └─ queries.go                 → constantes SQL
   └─ env/env.go                     → LoadEnv (DATABASE_URL / EMBEDDING_MODEL), Probe
pkg/db                                → pgxpool lazy singleton
pkg/util                              → Slugify, Hash8 (determinístico), SafePath
```

### 4.1 `internal/skills/rag-retriever/retriever.go` — orquestração (a skill)

- **`Do(ctx, input, filtros, cfg)`** é o ponto de entrada: monta as queries com
  `BuildQueries`, executa a busca via `retriever.Retrieve(ctx, cfg, queries,
  filtros)`, grava o espelho com `SaveResults` e retorna **resultados em
  memória + path do espelho** (sentinela `ErrEspelhoNaoGravado` quando só a
  gravação falha).
- **`SaveResults(input, results)`** grava o JSON em
  `.agent/tmp/retrieval/<slug>-<hash8>.json` (slug + hash derivados do input,
  molde do espelho do briefing), retornando o caminho do arquivo criado.
- **`BuildQueries(ctx, input, filtros, gen)`** usa o gerador como caminho
  primário; sem gerador, ou com geração falha/vazia, **degrada com aviso
  explícito** para uma única query do input cru.
- **`LLMQueryGenerator`** implementa a geração usando a API de Responses do
  openai-go, com `systemPrompt` + `prompt.md` embutidos na própria skill
  (`montarPrompt(filtros)` inclui os filtros escolhidos para orientar a
  reescrita).
- **`ParseGeneratedQueries`** extrai o array JSON de strings da saída do
  gerador, tolerando texto ao redor e *code fences*; strings vazias são
  descartadas.

### 4.2 `internal/skills/classifier` — a fonte dos filtros

O Classificador (`classifier.Do(ctx, input, listas, modoRAG, agent)`) devolve
`map[string]string`: um valor canônico por Lista, `""` = vazio explícito (nada
se encaixa), nunca um valor inventado. O contrato completo está em
[Contrato do classificador](../.agent/classifier/issues/02-contrato-do-classificador.md).

- **Validação no cliente em 100% dos caminhos** — o provedor pode ignorar o
  schema de saída em silêncio (ver
  [Pesquisa: saída estruturada](../.agent/classifier/assets/pesquisa-saida-estruturada.md)),
  então toda resposta passa por pertencimento à Lista antes de sair; saída
  inválida é refeita no máximo 2× com o erro colado na tentativa seguinte e,
  esgotadas as tentativas, o erro sobe explícito.
- **`rag.Listas()`** é a fonte única das Listas do chamador RAG —
  `protocol`/`daemon`/`chunk_type`, com os nomes das colunas da `frr_docs`
  (saída sem tradução). Conjunto fechado por contrato: **não** é derivado do
  banco (o conteúdo da base varia com a ingesta). Os prompts que precisam do
  vocabulário derivam da tabela.
- **`rag.FiltrosFrom(resultado)`** converte o mapa no `rag.Filtros` da busca;
  `""` vira "sem filtro" (o `NULL` do SQL). O retriever **não revalida** os
  valores — a garantia de pertencimento é do Classificador.

### 4.3 Geração de queries — caminho primário com degradação avisada

As queries da busca são **geradas pelo LLM** a partir do input + filtros
(`rag.QueryGenerator`): o modelo reescreve linguagem informal em terminologia
FRR/RFC, uma query por aspecto (2 a 5), sempre consistente com os filtros —
ver `internal/skills/rag-retriever/prompt.md`.

Falhou a geração (ou não há gerador), a busca **degrada com aviso explícito**
(`log.Warn`) para uma única query do input cru. Nunca falha silenciosa; nunca a
busca inteira se perde por causa da reescrita.

> O caminho antigo — parse determinístico do campo `ragQueries` do briefing
> (`ParseQueriesFromBriefing`, `ErrNoQueries`) com "fallback LLM" — foi
> **removido por inteiro** junto do `intent-analyser` (ver §11). Hoje o gerador
> é o caminho primário e o input cru é a degradação.

### 4.4 `internal/agent/rag/retriever` — embeddings e consistência de dimensão

`embedTexts` usa `client.Embeddings.New(...)` com o **mesmo modelo do
ingester** (`text-embedding-qwen3-embedding-4b`, `rag.DefaultEmbeddingModel`).
Isso é um requisito físico: se a query fosse embedded por um modelo diferente
do que gerou os vetores do banco, a similaridade de cosseno perderia o
significado (espaços vetoriais diferentes).

**`ProbeEmbeddingDim`** embeds uma string de teste no primeiro uso e assert
`len == 2560`. Se o modelo trocar, o erro é claro ("dimensão X, esperava
2560") em vez de uma busca silenciosamente sem sentido.

### 4.5 `internal/agent/rag/retriever` — SQL híbrido

Duas queries parametrizadas com filtros opcionais como `*string` (nil = sem
filtro, via `nullableStr`):

```sql
-- vetorial (usa diskann)
SELECT chunk_id, 1 - (embedding <=> $1::vector) AS score
FROM frr_docs
WHERE ($2::text IS NULL OR daemon = $2)
  AND ($3::text IS NULL OR protocol = $3)
  AND ($4::text IS NULL OR chunk_type = $4)
ORDER BY embedding <=> $1::vector
LIMIT $5;

-- full-text (usa GIN)
SELECT chunk_id, ts_rank_cd(content_tsv, plainto_tsquery('english', $1)) AS score
FROM frr_docs
WHERE content_tsv @@ plainto_tsquery('english', $1)
  AND ($2::text IS NULL OR daemon = $2)
  ...
ORDER BY score DESC
LIMIT $5;
```

Cada fonte busca `limit × FetchFactor` (mais candidatos do que o resultado
final) para alimentar a fusão. Os filtros `daemon`/`protocol`/`chunk_type`
vêm do **conjunto único do Classificador** — viram `WHERE`, não metadados
decorativos, aproveitando o índice btree `(daemon, protocol, chunk_type)`.

`fetchChunks` busca os metadados dos vencedores **na ordem do RRF** (lookup
`chunk_id = ANY($1::text[])`).

### 4.6 `pkg/db` — pool lazy singleton

`db.OpenDB` cria um `pgxpool` na primeira chamada e o reutiliza
(`sync.Once`). Como o retriever roda quando a classificação muda, reabrir
socket a cada busca seria caro; o singleton evita essa recriação. Um `Ping` no
primeiro uso valida a conexão cedo (fail-fast).

### 4.7 `internal/agent/rag/env` — configuração via ambiente

`LoadEnv()` lê `DATABASE_URL`/`EMBEDDING_MODEL` de variáveis de ambiente com
fallback para o `.env` na raiz do repo, preservando a convenção já usada por
`internal/llm/config.go` (leitura própria de `.env`, sem dependência de
godotenv). `Probe(...)` oferece uma verificação de sanidade do schema
(contagem de chunks, FTS e `ORDER BY ... <=> $1::vector`) sem precisar de
embeddings reais.

---

## 5. Onde os Embeddings São Gerados (decisão importante)

Existem **dois clientes** no projeto:

| Cliente | Uso | Endpoint |
|---|---|---|
| `agent.Client` (chat) | conversa / geração de queries via LLM | provedor configurado (ex.: OpenRouter) |
| `agentapi.LoadEmbbedAgent()` | **embeddings** | LM Studio local `http://localhost:1234/v1` |

O `cmd/cli/main.go` (função `runRetrieval`) usa **`LoadEmbbedAgent()`** como
fonte do `Embedder` e do `EmbeddingModel`. Por quê? O provedor remoto usado
para o chat **não serve o modelo de embedding**
(`text-embedding-qwen3-embedding-4b` dá 404/"model does not exist"), enquanto o
LM Studio local o serve com as mesmas 2560 dimensões usadas no ingest.
Misturar os dois quebraria a similaridade vetorial — então embeddings têm seu
**próprio agente**, isolado do chat. (Isso foi validado na integração:
`len(embedding) == 2560`.)

---

## 6. Fusão RRF (Reciprocal Rank Fusion)

Cada fonte devolve um ranking (posição 0 = melhor). A **RRF** funde dois
rankings sem depender de calibragem de scores (que são incompatíveis entre
cosseno e `ts_rank_cd`):

```
score[id] += 1 / (k + rank + 1)     para cada fonte    (k = rag.RrfK = 60)
```

Por quê **RRF** em vez de soma de scores normalizados?

- Os scores de cada fonte **não são comparáveis** (cosseno ≈ [-1,1]; rank de
  texto em outra escala). RRF usa apenas a **ordenação**, tornando a fusão
  robusta a escalas diferentes.
- k=60 é o valor empírico padrão da literatura.
- É **puro e testável** (unidade `FuseRRF` recebe maps e devolve lista
  ordenada/deduplicada), sem estado externo.

---

## 7. Filesystem de Trabalho

- **Espelho do retrieval**: `.agent/tmp/retrieval/<slug>-<hash8>.json` —
  gravado por `ragretriever.SaveResults(...)`, nome derivado do input via
  `pkg/util.Slugify`/`Hash8`. O **Hash8 é determinístico**: o mesmo input
  regrava o mesmo espelho (sobrescreve, sem histórico de execuções).
- **Espelho do briefing rico**: `.agent/tmp/<slug>-<hash8>.json` — gravado
  pela skill `briefing` (fora do fluxo do CLI, §12), mesma convenção de nome.
- `.agent/` **é versionado** (os tickets do tracker moram em
  `.agent/<effort>/`); os espelhos em `.agent/tmp/` são artefatos de execução
  do agente, não confundir com os tickets.

O `cmd/cli/main.go` injeta o JSON dos resultados no prompt do passo [3] dentro
de um bloco `<recovered_context>...</recovered_context>`, dando ao LLM
evidência verificável e citável no contexto — sem embutir a origem bruta na
resposta final.

---

## 8. Integração com o CLI (fluxo E2E)

```
leu "exit"? ──► não ──► classifier.Do(..., modoRAG=false)  → exibição na tela
                         │                                    (lista=valor por Lista)
                         ▼
                   classifier.Do(..., modoRAG=true) → map → rag.FiltrosFrom
                         │        │
                         │        └─ falha: filtros vazios + aviso, comparação zerada
                         ▼
                   deveBuscar(atual, anterior)? ── não ──► re-injeta o contexto
                         │ sim                               do último retrieval
                         ▼
                   runRetrieval(ctx, agent, input, filtros)
                         │   LoadEmbbedAgent (LM Studio) + LLMQueryGenerator
                         │   ragretriever.Do(...) → memória + espelho
                         ▼
                   prompt = pergunta + <recovered_context> JSON
                         ▼
                   agent.Chat(ctx, transcript, prompt)  → resposta final
```

**Skip rule** (`deveBuscar`): o retrieval só roda quando a classificação RAG
do turno difere da do anterior — igualdade exata do mapa (`maps.Equal`),
vazio incluído. `atual == nil` (falha do classificador) ou `anterior == nil`
(primeiro turno / comparação zerada) sempre busca. Pulado, o contexto vem do
**último retrieval real** re-injetado no prompt.

**Degradações** (o turno nunca morre por falha do RAG):

- Classificação RAG falha → filtros vazios (idêntico a "nada se encaixa") e a
  comparação zera — o próximo turno busca de novo.
- Retrieval falha (ex.: `DATABASE_URL` vazio) → aviso e o turno segue sem
  contexto RAG.
- Só o espelho falha → `ErrEspelhoNaoGravado`; o contexto em memória segue e
  o turno continua.

O prompt **enriquecido** (com `<recovered_context>`) é o que entra no
transcript, preservando no histórico o contexto que o LLM realmente viu.

---

## 9. Decisões de Design e Seus Porquês (resumo)

| Decisão | Por quê |
|---|---|
| **Skill Go determinística** (não tool do agente) | O agente deserializa args de tool em `map[string]string`; args aninhados (array de queries) quebrariam. Uma skill que consome o input evita o gotcha e mantém o pipeline testável fora do loop de tools. |
| **Filtros vêm do Classificador** (`rag.Listas`/`FiltrosFrom`) | Conjunto fechado por contrato, validação de pertencimento 100% no cliente, um conjunto por busca inteira. |
| **Queries geradas pelo LLM como caminho primário** | Reescrita em terminologia FRR melhora a recuperação; a degradação é avisada e vai para o input cru — a busca nunca se perde. |
| **Validação no cliente + retry ×2** | O provedor pode ignorar o schema em silêncio; nunca sai valor inventado, e o erro final é explícito. |
| **Saída híbrida memória + espelho** | Em memória para o turno; espelho para rastreabilidade. Falha do espelho é sentinela, nunca derruba o valor. |
| **`Hash8` determinístico** | O mesmo input regrava o mesmo espelho — reprodutível, sem acúmulo. |
| **Busca híbrida com RRF** | Aproveita diskann + GIN; robusto a escalas de score distintas. |
| **Filtros viram SQL `WHERE`** | Usam o índice btree `(daemon, protocol, chunk_type)` da tabela. |
| **`LoadEmbbedAgent` (LM Studio) p/ embeddings** | Provedor de chat não serve o modelo de embedding; consistência de espaço vetorial com o ingest. |
| **`ProbeEmbeddingDim`** | Detecta troca de modelo cedo, com erro claro (dim 4096). |
| **`response` = contexto, não resposta** | Separa recuperação de geração; evidência rastreável (`source_url`/`section_path`). |
| **Pool lazy singleton** | Evita reabrir conexão a cada busca; fail-fast no 1º uso. |
| **Skip rule** | Evita re-buscar quando a classificação não mudou; re-injeta o último contexto. |

---

## 10. Testes

**Unitários** (sem rede/banco):

- `internal/skills/classifier/classifier_test.go` — contrato do Classificador
  (validação, retry ×2, resolução de grafia, vazio explícito) com LLM falso.
- `internal/skills/briefing/briefing_test.go` — contrato do briefing rico.
- `internal/agent/rag/listas_test.go` — costura `FiltrosFrom` ↔ nomes das
  Listas (fixa a fonte única).
- `tests/internal/llm/listas_test.go` — `rag.Listas()`.
- `tests/internal/llm/querygen_test.go` — `BuildQueries`: gerador primário vs.
  degradação para input cru (com fake), `ParseGeneratedQueries` (JSON, code
  fence, vazio, inválido).
- `tests/internal/llm/retriever_test.go` — `VectorLiteral`, `FuseRRF`
  (dedupe, ordenação, limite, vazio), `SaveResults` (shape JSON com
  protocol/daemon/chunk_type) e falha de gravação (sentinela).
- `tests/internal/llm/env_test.go` — leitura de `.env` e defaults.
- `cmd/cli/main_test.go` — skip rule (`TestDeveBuscar`) e o bloco
  `<recovered_context>` (`TestJSONDoContexto`).
- `pkg/util/util_test.go` — `Slugify`, `Hash8`, `SafePath`.

**Integração** (gated em `DATABASE_URL`; pulados sem banco):

- `TestRetrieveIntegration` — embeddings LM Studio + busca híbrida real em `frr_docs`.
- `TestE2EDoIntegration` — pipeline completo: input + filtros →
  `ragretriever.Do(...)` → memória + espelho em disco.
- `TestProbeSkippedWithoutDB` — sanitização do schema.

**Gates de aceite** (escopos das Notas do mapa
[Classificador](../.agent/classifier/map.md); `go build`/`go test` sem escopo
quebram fora do alcance — pré-existente):

```bash
gofmt -l <arquivos tocados>
go vet ./internal/... ./pkg/... ./cmd/... ./tests/internal/llm/...
go build ./internal/... ./pkg/... ./cmd/... ./tests/internal/llm/...
go test -count=1 <pacotes citados>
```

> **Nota de cobertura**: o corpus atual só contém docs de **BGP** (516 chunks:
> 375 `command_reference` + 141 `concept`). Queries de OSPF retornam contexto
> vazio porque **não há docs de OSPF ingeridos** — comportamento correto do
> retriever, não um bug. Para cobrir outros protocolos, basta rodar `chucker`
> + `ingester` com o `.rst` correspondente.

---

## 11. Peças vivas e peças mortas

**Vivas** (o fluxo atual):

- `internal/skills/classifier/` — o Classificador (contrato em
  `.agent/classifier/issues/02-contrato-do-classificador.md`).
- `internal/skills/briefing/` — briefing rico, **fora do fluxo do CLI** (§12).
- `internal/skills/rag-retriever/` — esta skill: `Do`, `SaveResults`,
  `BuildQueries`, `LLMQueryGenerator`, `ParseGeneratedQueries`, prompts.
- `internal/agent/rag/` — `Listas()`, `FiltrosFrom`, `Filtros`,
  `QuerySuggestion`, `Config`, `Result`, mecanismo de busca, `env`.
- `cmd/cli/main.go` — 2 chamadas ao Classificador por turno, skip rule,
  `runRetrieval`, `<recovered_context>`.
- `pkg/util` — `Slugify`/`Hash8` (nomes dos espelhos).

**Mortas** (removidas; não voltar a documentar como ativas):

- `internal/skills/intent-analyser/` — substituído pelo Classificador
  (classificação) + `internal/skills/briefing` (partes ricas).
- Campo `ragQueries` do briefing e `ParseQueriesFromBriefing` / `ErrNoQueries`.
- `loadRetrievalJSON` (o CLI lê o contexto dos resultados em memória).
- O "fallback LLM" de queries — hoje o gerador é o caminho primário e a
  degradação é o input cru com aviso (§4.3).
- `tests/internal/llm/query_test.go` (cobria o parse do `ragQueries`).

---

## 12. Briefing rico (fora do fluxo do CLI)

`internal/skills/briefing` é a skill preservada do antigo `intent-analyser`:
só as partes ricas (devices, conexões, resumo do problema e pipeline de
execução). Entrada: o input textual. Saída:
`Do(ctx, input, agent) (Briefing, string, error)` — valor em memória **+**
espelho `.agent/tmp/<slug>-<hash8>.json` (sentinela
`briefing.ErrEspelhoNaoGravado`), validação/retry ×2 como o Classificador.
O contrato está em
[Desenho da skill do briefing rico](../.agent/classifier/issues/04-desenho-da-skill-do-briefing-rico.md).
O CLI **não** a chama — ela fica pronta para as próximas partes do plano.

---

## 13. Passos Futuros Sugeridos

1. **Ingerir docs de OSPF/IS-IS/static** para o retriever ter base real
   multi-protocolo (hoje só BGP).
2. **Paginamento/limites por token** do `response` para proteger o contexto
   em buscas grandes.
3. **Observabilidade**: log do RRF por query (quais chunks vieram de cada
   fonte) para sintonizar `FetchFactor`.
4. **Avaliação de qualidade do classificador** quando for preciso provar
   acurácia (conjunto rotulado, revisão humana) — ver
   [Classificador](../.agent/classifier/map.md), "Ainda não especificado".