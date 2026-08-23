---
name: 02-evidence
executor: script
description: Retrieves FRR documentation for the version of record covering the briefing topics.
---

# Evidence

Mechanical stage. Derives queries from the briefing, searches the index, and
writes the passages with provenance. No model reasons here.

Implementation: `internal/skills/rag-retriever`.

## Inputs

| Source | File/Location | Section/Scope | Why |
|--------|--------------|---------------|-----|
| Previous stage | `../01-intent/brief.json` | `objective`, `sessions`, daemons | The topics to search for |

## Process

1. Derive search queries from the objective, the sessions, and the daemons
   named in the briefing.
2. Run hybrid search against the vector index.
3. Write each passage with its source document, section, and FRR version.

## Outputs

| Artifact | Location | Format |
|----------|----------|--------|
| Evidence | `evidence.json` | `chunks[]`, each with `source`, `section`, `frr_version` |

## Degradation

Retrieval being unavailable or empty does not stop the run. Write
`{"chunks": [], "degraded": true, "reason": "..."}` and continue. Stages 03
and 04 handle missing evidence by restricting themselves to long stable
syntax and flagging the version gated features they had to avoid.
