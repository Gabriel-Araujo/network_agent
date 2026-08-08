# RAG Query Rewriting

Your job: turn the intent briefing below into search queries for an FRR
documentation knowledge base stored in a PostgreSQL+pgvector table called
`frr_docs`.

The briefing (JSON) may lack the `ragQueries` field. When that happens,
build the queries yourself from the fields `classification` (intentType,
protocols, daemons), `problemAndGoalSummary` and `executionPipeline`.

## Rules

1. Rewrite informal language into FRR/RFC terminology (e.g. "neighbor
   won't come up" → "BGP session Active state" or "OSPF neighbor stuck
   Exstart"; "my router won't talk to the other one" → "OSPF neighbor
   adjacency down").
2. Prefer the exact FRR command name when the user signals it (e.g. if they
   mention "redistribute", keep the term).
3. One query per distinct aspect, 2 to 5 queries total — don't try to cover
   the whole plan in a single string.
4. For each query, suggest metadata filters:
   - `protocol`: bgp | ospf | ospf6 | isis | rip | ripng | pim | ldp | vrrp | bfd | static
   - `daemon`: bgpd | ospfd | ospf6d | isisd | ripd | ripngd | pimd | pim6d | ldpd | vrrpd | bfdd | staticd | zebra
   - `chunk_type`: `command_reference` when the question is about a specific
     command's syntax/behavior, `concept` when it's about general protocol
     behavior.
   Leave protocol/daemon empty if the briefing doesn't make it clear (do not
   guess loudly — empty means "no filter").
5. Output ONLY a raw JSON array, no prose, no code fences. Shape:

```json
[
  {"query": "...", "protocol": "...", "daemon": "...", "chunk_type": "..."}
]
```
