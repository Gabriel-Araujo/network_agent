# FRR Config Builder

Produces FRRouting configuration files for one routing technology per run.
One pipeline, one tech pack per run, six stages in numeric order.

This file is the pipeline's map and its entry point. `AGENT.md` is the Builder's
system prompt for a chat turn: a turn writes the configuration directly and runs
none of these stages.

## Folder Map

```
builder/
├── AGENT.md           (system prompt of a chat turn, outside the pipeline)
├── CONTEXT.md         (you are here: start here for task routing)
├── setup/             (onboarding questionnaire, run once)
├── shared/            (cross-stage rules every LLM stage loads)
├── tech-packs/        (per-technology reference: doctrine, syntax, checklist)
└── stages/
    ├── 01-intent/     (request into structured briefing)
    ├── 02-evidence/   (retrieve docs for the version of record)
    ├── 03-plan/       (decide config shape, human approves)
    ├── 04-render/     (emit frr.conf)
    ├── 05-validate/   (mechanical syntax and reference check)
    └── 06-review/     (checklist and operator delivery)
```

## Triggers

| Keyword | Action |
|---------|--------|
| `setup` | Run the onboarding questionnaire in `setup/questionnaire.md` |
| `status` | Show pipeline completion for every stage of the current run |

## Tech Pack Selection

Read the request and pick exactly one slug. The slug binds `{{TECH}}` for the
whole run. A request for a different technology is a new run.

| Slug | Covers |
|------|--------|
| `bgp` | eBGP/iBGP, route reflectors, policy, communities, aggregation, multihoming, BFD over BGP |

If nothing fits, return `UNROUTABLE` with one sentence saying what was missing.
Do not guess.

## Task Routing

| Task Type | Go To | Description |
|-----------|-------|-------------|
| Understand the request | `stages/01-intent/CONTEXT.md` | Briefing plus the questions that must be answered |
| Ground it in the docs | `stages/02-evidence/CONTEXT.md` | Retrieval for the version of record, script |
| Decide the shape | `stages/03-plan/CONTEXT.md` | Blocks, daemons, security posture, rollback. Human approves |
| Emit the artifact | `stages/04-render/CONTEXT.md` | frr.conf and daemons from the approved plan |
| Check it mechanically | `stages/05-validate/CONTEXT.md` | Syntax and dangling references, script |
| Check it by judgment | `stages/06-review/CONTEXT.md` | Doctrine checklist plus operator commands |

## Routing

| Task | Go To |
|------|-------|
| Any new config request | Tech Pack Selection above, then `stages/01-intent/CONTEXT.md` |
| Resume after answering questions | `stages/01-intent/CONTEXT.md` |
| Resume after approving a plan | `stages/04-render/CONTEXT.md` |
| Validation failed, fix the artifact | `stages/04-render/CONTEXT.md` |

## Shared Resources

| Resource | Location | Contains |
|----------|----------|----------|
| Execution rules | `shared/execution-rules.md` | Boundaries, version of record, output register. Every LLM stage loads it |
| Tech packs | `tech-packs/CONTEXT.md` | Routing into the per-technology reference collection |
| Onboarding | `setup/questionnaire.md` | Agent level defaults, asked once, never per run |
| Skills | `../skills/*/SKILL.md` | Executable procedures the script stages call |

## What to Load

Every token of irrelevant context is a token of diluted attention. Load the
minimum. Each stage CONTEXT.md declares its own Inputs table, and that table
is authoritative.

| Stage | Load These | Do NOT Load |
|-------|-----------|-------------|
| 01-intent | `shared/execution-rules.md`, tech pack `TECH.md`, `references/required-facts.md` | syntax, examples, checklist: this stage writes no config |
| 02-evidence | nothing, script stage | anything, no model runs here |
| 03-plan | `shared/execution-rules.md`, tech pack `TECH.md` and `doctrine.md`, the `examples/` title list | syntax, example bodies: the plan names blocks and one shape, not commands |
| 04-render | `shared/execution-rules.md`, tech pack `syntax.md`, the one example named by stage 03 | doctrine, questionnaire, every other example: those were settled in stage 03 |
| 05-validate | nothing, script stage | anything, no model runs here |
| 06-review | `shared/execution-rules.md`, tech pack `checklist.md` | syntax, examples: the checklist is self-contained |

## Context Rule

No stage reads everything. A stage loads only what the Inputs table of its own
CONTEXT.md declares. Reading on your own initiative is not allowed.

## Placeholders

`{{TECH}}` in a stage Inputs table resolves to the tech pack slug chosen at
the start of the run, for example `tech-packs/bgp/`.

## Stage Handoffs

Each stage writes its artifacts to its own folder in the Run tree. The
next stage reads them from there. If a human edits an artifact, the next stage
picks up the edit.

Definition and execution live in different trees. This folder is the
definition: versioned, read only. Artifacts go to
`.agent/runs/builder/<stage>/`, so a contract naming
`../01-intent/brief.json` resolves to
`.agent/runs/builder/01-intent/brief.json`, and an Outputs row naming
`plan.md` is written to `.agent/runs/builder/03-plan/plan.md`.

One Run tree per agent, one active run at a time. A new request reuses
the same folders, so check `status` first if the previous run still
matters.

## Input Sources

The Source column of a stage Inputs table declares which layer the row belongs
to. The walker reads that column to decide what becomes system prompt and what
becomes the turn.

| Source | Layer | Resolves against | Goes into |
|--------|-------|------------------|-----------|
| `Shared` | 3 | this folder | system prompt |
| `Tech pack` | 3 | this folder, with `{{TECH}}` bound | system prompt |
| `Reference` | 3 | the stage folder in this tree | system prompt |
| `Previous stage` | 4 | `.agent/runs/builder/` | the turn |
| `Retry only` | 4 | `.agent/runs/builder/`, only on a retry pass | the turn |
| `User` | 4 | the current message | the turn |

Layer 3 is stable across runs and belongs in the system prompt, where it reads
as constraint. Layer 4 changes every run and belongs in the turn, where it
reads as material to work on.

## Input Scopes

The Section/Scope column is a closed vocabulary. The walker rejects an
unrecognized value when it loads the agent, so a typo or a scope written
as prose fails before any model call instead of silently dropping an input.

| Scope | Target | Means |
|-------|--------|-------|
| `Full file` | a file | the whole file |
| `Full text` | not a file | the live user turn |
| `All files` | a directory | every file in it |
| `Titles only` | a directory | the first heading of each file, as a list |
| `Named in <path>` | a directory | the one file named by an artifact of an earlier stage |
| `"A" and "B"` | markdown | the named headings, in double quotes |
| `field` | JSON | the named field, in backticks |
| `Read by the script` | anything | a script stage reads it, nothing is assembled |

Double quotes mean a markdown heading, backticks mean a JSON field. Nothing
else parses.

A `Named in` scope resolves inside the directory on its own row. A value that
escapes that directory, or names a file that is not there, stops the run and
lists the valid names. A missing artifact counts as no selection, and the run
continues without that input.
