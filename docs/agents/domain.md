# Domain Docs

How the engineering skills should consume this repo's domain documentation when exploring the codebase.

## Before exploring, read these

- **`CONTEXT.md`** at the repo root: the glossary. It started with the logging subsystem and now
  covers the ICM pipeline; other terms enter as they are settled.
- **`docs/adr/`**: read the ADRs that touch the area you're about to work in. There are two —
  `0001-trace-is-a-channel-not-a-level.md` and
  `0002-agent-and-workspace-are-different-trees.md`.

There is no `CONTEXT-MAP.md`: this is a **single-context** repo.

If any of these files don't exist, **proceed silently**. Don't flag their absence; don't suggest
creating them upfront. The `/domain-modeling` skill (reached via `/grill-with-docs` and
`/improve-codebase-architecture`) creates them lazily when terms or decisions actually get
resolved.

## File structure

Single-context repo:

```
/
├── CONTEXT.md          ← the glossary
├── docs/
│   ├── adr/            ← 0001, 0002
│   └── agents/         ← tracker, domain and triage conventions for the skills
├── cmd/                ← ui (Wails) and cli front-ends
├── internal/           ← agent, config, logger, skills, tools, types
└── pkg/                ← db, exception, util
```

## Use the glossary's vocabulary

When your output names a domain concept (in an issue title, a refactor proposal, a hypothesis, a
test name), use the term as defined in `CONTEXT.md`. Don't drift to synonyms the glossary
explicitly avoids — each entry carries an `_Avoid_` line for exactly this.

If the concept you need isn't in the glossary yet, that's a signal: either you're inventing
language the project doesn't use (reconsider) or there's a real gap (note it for
`/domain-modeling`).

## Flag ADR conflicts

If your output contradicts an existing ADR, surface it explicitly rather than silently overriding:

> _Contradicts ADR-0002 (agent and workspace are different trees), but worth reopening because…_
