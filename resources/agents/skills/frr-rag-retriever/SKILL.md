---
name: frr-rag-retriever
description: "Retrieves FRR documentation context for an intent briefing. Consumes the .md/.json file produced by the intent analyzer (frr-intent-analyzer / intent-analyser), generates search queries from its ragQueries field (or via LLM fallback), runs hybrid pgvector search on frr_docs, and returns structured JSON — a list of objects with query, protocol, daemon, chunk_type, and response fields, where response is retrieved context (parent_content plus source/section), not a generated answer. Use after the intent briefing exists and before final answer generation."
---

# FRR RAG Retriever

This skill turns a structured intent briefing (`.json`) into RAG search
results in JSON. It does **not** answer the user's question — it fetches
the documentation context that a later step will use to build the answer.

## Input

Path to the briefing JSON file saved by the intent analyzer, e.g.:

```
.agent/tmp/network_agent/meu-ospf-nao-converge-entre-60becac1.json
```

The briefing contains (at minimum): Classification (protocol/daemon),
Devices, Connections, Problem/goal summary, Execution pipeline, and a
`ragQueries` object keyed by query number. Each value contains `query`,
`protocol`, `daemon`, and `suggestedChunkType`.

## Steps

1. **Locate `ragQueries`** — parse the briefing JSON deterministically. Sort
   numeric keys (`1`, `2`, `10`) numerically; each value yields one search
   query plus optional filters.
2. **Fallback** — if the briefing lacks `ragQueries` (older format), have the
   LLM rewrite queries from sections 1/4/5 (Classification, summary,
   pipeline), respecting the `frr_docs` column vocabulary.
3. **Embed** — batch-embed all queries with the SAME model used at ingest
   (`text-embedding-qwen3-embedding-8b`, 4096 dims; configurable via
   `EMBEDDING_MODEL`).
4. **Hybrid search per query** — vector (`embedding <=> $1::vector`, via
   diskann) + full-text (`content_tsv @@ plainto_tsquery`, via GIN), both
   honoring `daemon`/`protocol`/`chunk_type` filters.
5. **Fuse** — Reciprocal Rank Fusion (k=60) in Go; dedupe by chunk_id;
   cap at limit.
6. **Output** — write `[{"query": string, "protocol": string, "daemon": string, "chunk_type": string, "response": string}]` to
   `.agent/tmp/retrieval/<briefing-name>.json` and load it as context for
   final answer generation.

## Output contract

```json
[
  {
    "query": "OSPF MTU mismatch neighbor adjacency",
    "protocol": "ospf",
    "daemon": "ospfd",
    "chunk_type": "concept",
    "response": "<parent_content>\n\nSource: <source_url> (<section_path>)"
  }
]
```

- `query` is the rewritten search query.
- `protocol`, `daemon`, and `chunk_type` are the metadata filters used for the
  query. An empty value means that filter was not specified.
- `response` is **retrieved context**; never generate the answer here.

## Environment

- `DATABASE_URL` — postgres DSN for `frr_docs` (required).
- `EMBEDDING_MODEL` — optional; defaults to `text-embedding-qwen3-embedding-8b`.
- Embeddings use the same OpenAI-compatible endpoint as the agent (`.env`).

## Verification

- No DB: unit tests pass, integration tests skip (`DATABASE_URL` gated).
- With DB: `go test ./internal/llm/rag/ -run Retrieve -v` should pass;
  sanity probe reports `chunk_count > 0`.
- E2E: CLI run → briefing `.json` → retrieval JSON in `.agent/tmp/retrieval/`.

## Design notes

- Deterministic parse first, LLM fallback second — never the reverse
  (LLM calls are slow and non-deterministic).
- The `documents` (question/answer) table is unused today; retrieval
  targets `frr_docs` only.
- `response` content must stay bounded to avoid flooding context: `limit`
  defaults to 5 chunks per query.
