# Retriever RAG — Documentação da Implementação

> Pipeline de busca híbrida (vetorial + full-text) sobre a base `frr_docs`
> (PostgreSQL + pgvector), alimentado pelo briefing gerado pela skill de
> análise de intenção (`intent-analyser`).
> Saída: JSON estruturado `[{"query": string, "protocol": string, "daemon": string, "chunk_type": string, "response": string}]`.

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
[1] intent-analyser ──► briefing JSON estruturado (classificação, daemons,
   │                     filtros, plano de diagnóstico, queries p/ busca)
   ▼
[2] skill rag-retriever (ESTA IMPLEMENTAÇÃO) ──► busca híbrida no pgvector
   │                                              (frr_docs) usando as queries
   │                                              extraídas do briefing
   ▼
[3] resposta final gerada pelo LLM com o contexto recuperado
```

A skill `rag-retriever` é o elo **[2]**: ela lê o briefing JSON, transforma em
**queries de busca reais** (parse determinístico do `ragQueries`, com fallback
por LLM), executa a recuperação de documentação relevante e grava o resultado
num shape determinístico, para que o passo [3] não precise re-reinterpretar a
pergunta em linguagem natural. Por fim, devolve o **caminho do arquivo JSON**
gerado.

---

## 2. Objetivo e Contrato de Saída

A skill recebe o caminho do briefing JSON produzido pelo `intent-analyser` e
gera:

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
  usados na busca para essa query. Podem estar vazios quando o briefing não
  especifica o filtro.
- **`response`** — o **contexto recuperado** (`parent_content` + origem:
  `source_url` e `section_path`), **não** uma resposta gerada pelo LLM.
  Gerar a resposta é responsabilidade do passo [3].

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

A lógica foi **extraída para a skill** `internal/skills/rag-retriever/`
(espelhando o `internal/skills/intent-analyser/`), que orquestra o fluxo e
carrega **seus próprios prompts** (`prompt.md` via `//go:embed`). O mecanismo
puro de busca permanece em `internal/llm/rag/retriever/`:

```
cmd/cli/main.go
   │  intentanalyser.Do(...)                      → briefing JSON (.agent/tmp/network_agent/)
   │  runRetrieval(...) → ragretriever.Do(...)    → JSON (.agent/tmp/retrieval/)
   ▼
internal/skills/rag-retriever/        (skill — orquestração, prompts, fallback LLM)
   ├─ retriever.go                    → Do, SaveResults, BuildQueries,
   │                                    ParseQueriesFromBriefing, LLMQueryGenerator
   ├─ prompt.go / prompt.md           → prompt do fallback LLM (//go:embed)
   ▼
internal/llm/rag/retriever/           (mecanismo de busca — sem IO de arquivo)
   ├─ retriever.go                    → Retrieve, VectorLiteral, FuseRRF, buildResponse
   ├─ embed.go                        → embeddings em lote + probe de dimensão (4096)
   ├─ db.go                           → SQL híbrido (vetorial + FTS) + lookup de chunks
   └─ queries.go                      → constantes SQL
internal/llm/rag/
   ├─ types.go / constants.go         → tipos (Config, Result, QuerySuggestion) e constantes
   └─ env/env.go                      → leitura de DATABASE_URL / EMBEDDING_MODEL (.env)
pkg/db                                → pgxpool lazy singleton
```

### 4.1 `internal/skills/rag-retriever/retriever.go` — orquestração (a skill)

- **`Do(ctx, briefingPath, cfg)`** é o ponto de entrada: lê o briefing JSON,
  chama `BuildQueries`, executa a busca via `retriever.Retrieve(...)`, grava o
  resultado com `SaveResults` e **retorna o caminho** do JSON gerado.
- **`SaveResults(briefingPath, results)`** grava o JSON em
  `.agent/tmp/retrieval/<mesmo-nome>.json` (mesma convenção de nome do
  briefing), retornando o caminho do arquivo criado.
- **`BuildQueries(ctx, content, gen)`** tenta o parse determinístico do campo
  `ragQueries`; se ausente, cai no fallback LLM (`gen`); se não há generator,
  retorna `rag.ErrNoQueries`.
- **`LLMQueryGenerator`** implementa o fallback usando a API de Responses do
  openai-go (mesmo padrão do `intent-analyser`), com `systemPrompt` +
  `prompt.md` embutidos na própria skill.
- Os prompts vivem em `prompt.go` (`systemPrompt` + `//go:embed prompt.md`) —
  **dentro da pasta da skill**, seguindo o modelo do `intent-analyser`.

### 4.2 Parse determinístico do JSON (caminho preferido)

O briefing do `intent-analyser` é **JSON** e carrega as queries no campo
`ragQueries` (objeto chave → sugestão):

```json
"ragQueries": {
  "1": {"query": "OSPF neighbor stuck Exstart Exchange state", "protocol": "ospf", "daemon": "ospfd", "suggestedChunkType": "concept"}
}
```

`ParseQueriesFromBriefing` extrai esse objeto **sem chamar LLM**
(`json.Unmarshal` + ordenação numérica das chaves para preservar a ordem).
Por quê?

- **Determinismo**: a mesma entrada gera sempre as mesmas queries — essencial
  para testes e reprodução.
- **Custo/precisão**: as queries já foram esculpidas com vocabulário FRR na
  etapa [1]; re-gerá-las gastaria uma chamada de modelo e poderia desviar.
- O parser normaliza `protocol`/`daemon` para minúsculas e valida
  `suggestedChunkType` contra `command_reference|concept` (vira `""`
  = "sem filtro" se vier valor inválido), evitando que lixo do modelo vire
  filtro SQL quebrado.

### 4.3 Fallback por LLM

`BuildQueries(ctx, content, gen)` tenta **primeiro o parse determinístico** e
só cai no LLM se o briefing **não tiver `ragQueries`** (formato legado ou
campo ausente). O `LLMQueryGenerator` usa a mesma API de Responses do projeto
(mesmo padrão do `intent-analyser`), com um prompt dedicado
(`internal/skills/rag-retriever/prompt.md`) que devolve JSON puro
(`ParseGeneratedQueries`, que tolera *code fences* e valida campos).

O fallback nunca é o caminho normal — é um **seguro contra briefings
legados**. Isso honra o princípio "determinístico primeiro, LLM segundo":
chamadas de modelo são lentas, não-determinísticas e custosas.

### 4.4 `internal/llm/rag/retriever/embed.go` — embeddings e consistência de dimensão

`embedTexts` usa `client.Embeddings.New(...)` com o **mesmo modelo do
ingester** (`text-embedding-qwen3-embedding-8b`). Isso é um requisito físico:
se a query fosse embedded por um modelo diferente do que gerou os vetores do
banco, a similaridade de cosseno perderia o significado (espaços vetoriais
diferentes).

**`ProbeEmbeddingDim`** embeds uma string de teste no primeiro uso e assert
`len == 4096`. Se o modelo trocar, o erro é claro ("dimensão X, esperava
4096") em vez de uma busca silenciosamente sem sentido.

### 4.5 `internal/llm/rag/retriever/db.go` — SQL híbrido

Duas queries parametrizadas com filtros opcionais como `*string` (nil = sem
filtro):

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
vêm **das colunas sugestas pelo briefing** — viram `WHERE`, não metadados
decorativos, aproveitando o índice btree `(daemon, protocol, chunk_type)`.

`fetchChunks` busca os metadados dos vencedores **na ordem do RRF**, e
`nullableStr` converte filtro vazio em `NULL`.

### 4.6 `pkg/db` — pool lazy singleton

`db.OpenDB` cria um `pgxpool` na primeira chamada e o reutiliza
(`sync.Once`). Como o retriever roda a cada turno da conversa, reabrir socket
a cada busca seria caro; o singleton evita essa recriação. Um `Ping` no
primeiro uso valida a conexão cedo (fail-fast).

### 4.7 `internal/llm/rag/env/env.go` — configuração via ambiente

`LoadEnv()` lê `DATABASE_URL`/`EMBEDDING_MODEL` de variáveis de ambiente com
fallback para o `.env` na raiz do repo (`paths.RepoFile`), preservando a
convenção já usada por `internal/llm/config.go` (leitura própria de `.env`,
sem dependência de godotenv). `Probe(...)` oferece uma verificação de sanidade
do schema (contagem de chunks, FTS e `ORDER BY ... <=> $1::vector`) sem
precisar de embeddings reais.

---

## 5. Onde os Embeddings São Gerados (decisão importante)

Existem **dois clientes** no projeto:

| Cliente | Uso | Endpoint |
|---|---|---|
| `agent.Client` (chat) | conversa / geração de queries via LLM | provedor remoto (OpenRouter) |
| `LLM.LoadEmbbedAgent()` | **embeddings** | LM Studio local `http://localhost:1234/v1` |

O `cmd/cli/main.go` (função `runRetrieval`) usa **`LoadEmbbedAgent()`** como
fonte do `Embedder` e do `EmbeddingModel`. Por quê? O provedor remoto usado
para o chat **não serve o modelo de embedding** (`text-embedding-qwen3-embedding-8b`
dá 404/"model does not exist"), enquanto o LM Studio local o serve com as
mesmas 4096 dimensões usadas no ingest. Misturar os dois quebraria a
similaridade vetorial — então embeddings têm seu **próprio agente**, isolado
do chat. (Isso foi validado na integração: `len(embedding) == 4096`.)

---

## 6. Fusão RRF (Reciprocal Rank Fusion)

Cada fonte devolve um ranking (posição 0 = melhor). A **RRF** funde dois
rankings sem depender de calibragem de scores (que são incompatíveis entre
cosseno e `ts_rank_cd`):

```
score[id] += 1 / (k + rank + 1)     para cada fonte    (k = 60)
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

- **Entrada**: `.agent/tmp/network_agent/<slug>-<hash8>.json` — gerado pelo
  `intent-analyser.Do(...)` (slug de até 5 palavras + hash8, nomes únicos).
- **Saída**: `.agent/tmp/retrieval/<mesmo-nome>.json` — gravado pelo
  `rag-retriever.SaveResults(...)`, seguindo a mesma convenção de nome
  (rastreabilidade 1:1 briefing → resultado).
- `.agent/` está no `.gitignore`, então artefatos não vazam para o versionado.

O `cmd/cli/main.go` injeta o JSON no prompt do passo [3] dentro de um bloco
`<recovered_context>...</recovered_context>`, dando ao LLM evidência
verificável e citável no contexto — sem embutir a origem bruta na resposta
final.

---

## 8. Integração com o CLI (fluxo E2E)

```
leu "exit"? ──► não ──► intentanalyser.Do(ctx, pergunta, agent)
                         │                          │
                         │       briefing JSON  ▼
                         │                  runRetrieval(ctx, agent, intentPath)
                         │                          │  LoadEmbbedAgent (LM Studio)
                         │                          │  ragretriever.Do(...) → JSON
                         │                          ▼
                         │            JSON em .agent/tmp/retrieval/
                         ▼
         prompt = pergunta + <recovered_context> JSON
                         ▼
                 agent.Chat(ctx, prompt)  → resposta final
```

A skill `rag-retriever` recebe `intentPath` (o path do briefing), extrai as
queries (`ragQueries` + fallback LLM), busca no `frr_docs` e devolve o path do
JSON gravado — que é lido pelo CLI e injetado no prompt final.

Se o retrieval falhar (ex.: `DATABASE_URL` vazio), o CLI **degrada
graciosamente**: loga o erro e segue sem contexto RAG — a conversa não é
interrompida pela ausência de banco.

---

## 9. Decisões de Design e Seus Porquês (resumo)

| Decisão | Por quê |
|---|---|
| **Skill Go determinística** (não tool do agente) | O agente deserializa args de tool em `map[string]string`; args aninhados (array de queries) quebrariam. Uma skill que consome o arquivo evita o gotcha e mantém o pipeline testável fora do loop de tools. |
| **Retriever como skill (`internal/skills/rag-retriever`)** | Espelha o `intent-analyser`: orquestração (Do/SaveResults), parse e prompts embutidos (`prompt.md`) vivem na skill; o mecanismo de busca fica em `internal/llm/rag/retriever`. |
| **Parse do `ragQueries` (JSON) antes de LLM** | Determinismo, custo e fidelidade ao vocabulário já esculpido na etapa [1]. |
| **Fallback LLM** | Cobre briefings legados sem `ragQueries`, mantendo compatibilidade. |
| **Busca híbrida com RRF** | Aproveita diskann + GIN; robusto a escalas de score distintas. |
| **Filtros viram SQL `WHERE`** | Usam o índice btree `(daemon, protocol, chunk_type)` da tabela. |
| **`LoadEmbbedAgent` (LM Studio) p/ embeddings** | Provedor de chat não serve o modelo de embedding; consistência de espaço vetorial com o ingest. |
| **`ProbeEmbeddingDim`** | Detecta troca de modelo cedo, com erro claro (dim 4096). |
| **`response` = contexto, não resposta** | Separa recuperação de geração; evidência rastreável (`source_url`/`section_path`). |
| **Pool lazy singleton** | Evita reabrir conexão a cada turno; fail-fast no 1º uso. |
| **Nome de saída espelha o briefing** | Rastreabilidade 1:1 e diretório já gitignored. |

---

## 10. Testes

**Unitários** (não dependem de rede/banco; em `tests/internal/llm/`, cobrindo
a skill e o mecanismo de busca):
- `query_test.go` — parse do `ragQueries` da skill (ordem numérica,
  normalização, `chunk_type` inválido, campo ausente, JSON inválido).
- `querygen_test.go` — `BuildQueries`: parse preferencial vs. fallback (com
  fake), JSON com/sem code fence, validação.
- `retriever_test.go` — `VectorLiteral`, `FuseRRF` (dedupe, ordenação, limite,
  vazio), `SaveResults` (shape JSON com protocol/daemon/chunk_type).
- `env_test.go` — leitura de `.env` e defaults.

**Integração** (gated em `DATABASE_URL`; pulados sem banco):
- `TestRetrieveIntegration` — embeddings LM Studio + busca híbrida real em `frr_docs`.
- `TestE2EDoIntegration` — pipeline completo da skill: briefing de exemplo →
  `ragretriever.Do(...)` → JSON em disco.
- `TestProbeSkippedWithoutDB` — sanitização do schema.

**Validação executada** (banco local `localhost:5432` + LM Studio `:1234`):
`go build ./...` ✅ · `go vet` ✅ · unit `go test ./tests/internal/llm -count=1` ✅ ·
suíte completa `go test ./...` ✅ — resultado **ALL_GREEN**.

> **Nota de cobertura**: o corpus atual só contém docs de **BGP** (516 chunks:
> 375 `command_reference` + 141 `concept`). Queries de OSPF no briefing de
> exemplo retornam contexto vazio porque **não há docs de OSPF ingeridos** —
> comportamento correto do retriever, não um bug. Para cobrir outros
> protocolos, basta rodar `chucker` + `ingester` com o `.rst` correspondente.

---

## 11. Arquivos Modificados/Criados

**Criados**
- `internal/skills/rag-retriever/` — a skill do retriever:
  - `retriever.go` — `Do`, `SaveResults`, `BuildQueries`,
    `ParseQueriesFromBriefing`, `LLMQueryGenerator`, `ParseGeneratedQueries`
  - `prompt.go` — `systemPrompt` + `//go:embed prompt.md`
  - `prompt.md` — prompt do fallback LLM
- `docs/RAG_RETRIEVER.md` — esta documentação
- `.env.example`

**Modificados**
- `internal/skills/intent-analyser/{analyser,prompt}.md` — saída em JSON com o
  campo `ragQueries` (substituiu a antiga tabela markdown).
- `cmd/cli/main.go` — wiring do `runRetrieval` agora chama
  `ragretriever.Do(...)` + injeção de contexto.
- `internal/llm/rag/types.go` — `Result` passa a incluir `protocol`, `daemon`
  e `chunk_type`.
- `internal/llm/rag/retriever/retriever.go` — `Retrieve` preenche os metadados
  no `Result`; **removidos** `Do` e `SaveResults` (movidos para a skill);
  removido o import de `querygen`.
- `tests/internal/llm/` — testes apontando para a skill e validando os novos
  campos da saída.
- `resources/agent/skills/frr-rag-retriever/SKILL.md` — documentado o contrato
  com `protocol`/`daemon`/`chunk_type` e o formato JSON do briefing.

**Removidos**
- `internal/llm/rag/querygen/` — pacote inteiro (`querygen.go`,
  `querygen_prompt.go`, `querygen_prompt.md`): o parse do `ragQueries` e o
  fallback LLM migraram para a skill, e o antigo parser de tabela markdown já
  era obsoleto (commit `64e53fd`).

---

## 12. Passos Futuros Sugeridos

1. **Ingerir docs de OSPF/IS-IS/static** para o retriever ter base real
   multi-protocolo (hoje só BGP).
2. **Gerar a resposta final a partir do JSON** (passo [3]) — o contrato
   `[{query, protocol, daemon, chunk_type, response}]` já está pronto para consumo.
3. **Paginamento/limites por token** do `response` para proteger o contexto
   em briefings grandes.
4. **Observabilidade**: log do RRF por query (quais chunks vieram de cada
   fonte) para sintonizar `FetchFactor`.
```
