# FRR Config Creator

Produces FRRouting configuration files for one routing technology per run.

## Folder Map

```
creator/
├── AGENT.md           (you are here)
├── CONTEXT.md         (start here for task routing)
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

## Routing

| Task | Go To |
|------|-------|
| Any new config request | `CONTEXT.md`, pick a tech pack, then `stages/01-intent/CONTEXT.md` |
| Resume after answering questions | `stages/01-intent/CONTEXT.md` |
| Resume after approving a plan | `stages/04-render/CONTEXT.md` |
| Validation failed, fix the artifact | `stages/04-render/CONTEXT.md` |

## What to Load

Every token of irrelevant context is a token of diluted attention. Load the
minimum. Each stage CONTEXT.md declares its own Inputs table, and that table
is authoritative.

| Stage | Load These | Do NOT Load |
|-------|-----------|-------------|
| 01-intent | `shared/execution-rules.md`, tech pack `TECH.md`, `references/required-facts.md` | syntax, examples, checklist: this stage writes no config |
| 02-evidence | nothing, script stage | anything, no model runs here |
| 03-plan | `shared/execution-rules.md`, tech pack `TECH.md` and `doctrine.md` | syntax, examples: the plan names blocks, not commands |
| 04-render | `shared/execution-rules.md`, tech pack `syntax.md` and `examples/` | doctrine, questionnaire: those were settled in stage 03 |
| 05-validate | nothing, script stage | anything, no model runs here |
| 06-review | `shared/execution-rules.md`, tech pack `checklist.md` | syntax, examples: the checklist is self-contained |

## Placeholders

`{{TECH}}` in a stage Inputs table resolves to the tech pack slug chosen at
the start of the run, for example `tech-packs/bgp/`.

## Stage Handoffs

Each stage writes its artifacts to its own folder in the agent workspace. The
next stage reads them from there. If a human edits an artifact, the next stage
picks up the edit.

Definition and execution live in different trees. This folder is the
definition: versioned, read only. Artifacts go to
`.agent/workspace/creator/<stage>/`, so a contract naming
`../01-intent/brief.json` resolves to
`.agent/workspace/creator/01-intent/brief.json`, and an Outputs row naming
`plan.md` is written to `.agent/workspace/creator/03-plan/plan.md`.

One workspace per agent, one active pipeline at a time. A new request reuses
the same folders, so check `status` first if the previous pipeline still
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
| `Previous stage` | 4 | `.agent/workspace/creator/` | the turn |
| `Retry only` | 4 | `.agent/workspace/creator/`, only on a retry pass | the turn |
| `User` | 4 | the current message | the turn |

Layer 3 is stable across runs and belongs in the system prompt, where it reads
as constraint. Layer 4 changes every run and belongs in the turn, where it
reads as material to work on.
