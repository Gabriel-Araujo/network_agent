# The two meanings of workspace

Type: grilling
Status: resolved

## Question

`workspace` names two opposite things. `resources/agents/creator/` is the
versioned, read-only definition. `.agent/workspace/creator/` is where a run
writes its artifacts. The plan calls the first `Workspace` and the second
`Space` (§1.1, §1.3), and neither name says which is which.

The root `CONTEXT.md` is 33 lines and covers logging only. Not one ICM term is
in it: stage, contract, tech pack, layer, gate, scope, walker, either
workspace.

Settle the canonical terms and write them into `CONTEXT.md` as the glossary
entries, with the `_Avoid_` lines the file's format uses. Go type names follow
from that, and every later ticket inherits them.

## Answer

**Agent** is the definition, **Workspace** is where a run writes. The two
trees swap the plan's naming, and the pause is a **Checkpoint**, not a gate.

### The pair (Q1)

| Term | Is | Layer |
|------|----|----|
| `Agent` | `resources/agents/<name>/`, versioned, read only | 1 to 3 |
| `Workspace` | `.agent/workspace/<agent>/<stage>/`, one run's artifacts | 4 |

The plan's `Workspace`/`Space` was rejected on the ticket's own grounds: the
names differ by one word that reads as an abbreviation of the other. `Space`
also collides with address space, which is live BGP vocabulary in the tech pack.

The filesystem decided the direction. The path is already `.agent/workspace/`,
`AGENT.md:61` already said "the agent workspace" for artifacts, and the map's
Destination pins `.agent/workspace/creator/06-review/review.md`. Keeping
`workspace` on the artifact tree cost nothing; inverting it would have moved a
path the destination names.

`Definition` was the runner up. `Agent` won because `resources/agents/` and
`AGENT.md` already name the tree that way, and the collision with the Go package
`internal/llm/agent` is smaller than it looks: four of its five call sites
already alias it on import (`agentapi`, `llm`, `llm2`).

Recorded as [ADR 0002](../../../docs/adr/0002-agent-and-workspace-are-different-trees.md).

### What ICM actually says

Fetched `_core/CONVENTIONS.md` of the methodology. Three findings:

1. **`workspace` there means the union of both trees**, because ICM writes
   artifacts inside the definition tree at `stages/0N-name/output/`. This repo
   split them, so it narrows the word. That mismatch is the ADR's Consequences
   section: anyone reading the methodology alongside this repo will hit it.
2. **`Checkpoint` is ICM canon** for the human pause. It never says gate.
3. **Five layers, 0 to 4**: 0 `CLAUDE.md`, 1 top-level `CONTEXT.md`, 2 stage
   `CONTEXT.md`, 3 reference material, 4 working artifacts. This repo has no
   `CLAUDE.md`, so it has no Layer 0, and Layer 1 is `creator/CONTEXT.md`.
   That is why the Inputs table's Source column only ever says 3 or 4.

### Checkpoint, not gate (Q2)

Decided on collision, not canon: `resources/` uses "version gated" ten times to
mean an FRR feature that needs a given release. One word, two unrelated
meanings, in files the model reads. `Checkpoint` was already the `##` heading
the walker derives the pause from, so the shipped word won.

Consequence: `Gate`/`GateWhen` become `Checkpoint`/`CheckpointWhen`, and the
frontmatter key `gate_when: questions.md` became `checkpoint_when`. Free today
because nothing parses frontmatter yet.

Note for the gates ticket: `03-plan` declares **two** checkpoint rows, after
step 4 and after step 9, but only the terminal one is a walker pause. The step 4
row happens inside the model's own turn. A Checkpoint is not one-per-stage.

### The glossary (Q3, Q4, Q5, Q6)

Thirteen terms in the root `CONTEXT.md` under a new `### ICM` heading, ahead of
`### Logging`: Agent, Workspace, Pipeline, Run, Stage, Contract, Layer, Tech
pack, Checkpoint, Scope, Artifact, Executor, Walker.

A second context under `resources/` was rejected: inside that tree `CONTEXT.md`
already means an ICM routing file, and repo-maintenance prose does not belong in
a versioned definition. The overload is itself a glossary line, on `Contract`.

The whole file converted to English, and ADR 0001 with it, renamed to
`0001-trace-is-a-channel-not-a-level.md`. Two things deliberately left alone:
the Portuguese error strings in `internal/paths/paths.go`, which are Go and not
a terminology matter, and the term name **Lugar**, which is load bearing in five
`internal/logger` comments. Translating a settled term is a rename, not a
translation, and it is not this ticket's business.

**Deviation from the agreed term list.** `Pipeline` was to fold into `Run` as an
`_Avoid_`. It did not: inside the creator tree `pipeline` is used structurally
("One pipeline, one tech pack per run, six stages in numeric order"), and only
`AGENT.md:72` used it to mean one traversal. Both concepts exist, so both got an
entry. Pipeline is the structure and belongs to the Agent; Run is one traversal
of it.

`Walker` and `Contract` were confirmed as they stand.

### Go names (Q8)

Package `internal/agents`, mirroring `resources/agents/` on disk. Types
`agents.Agent`, `agents.Workspace`, `agents.Stage`, `agents.Walker`, with
`agents.Load` and `agents.Open`. `agents.Agent` stutters mildly; accepted for
the mirror. Not `internal/workspace`, which would have stuttered as
`workspace.Workspace` once the pair was settled.

Plan sections 1.1 and 1.3 carry a supersession note pointing here, on naming
only. The type shapes are untouched.

### Edits applied

- `CONTEXT.md`: rewritten in English, `### ICM` with thirteen terms added.
- `docs/adr/0001-...`: renamed and translated.
- `docs/adr/0002-agent-and-workspace-are-different-trees.md`: new.
- Nine prose lines across `AGENT.md`, `creator/CONTEXT.md`,
  `setup/questionnaire.md`, `shared/execution-rules.md`,
  `references/required-facts.md` and `tech-packs/CONTEXT.md`, plus the
  frontmatter key and two `gate` lines in `04-render`.
- `.agent/plans/icm-creator-workspace.md`: supersession notes in 1.1 and 1.3.

Three of the ten `workspace` lines flagged while charting needed no change: they
meant the artifact tree all along, which is what made this ticket necessary.
Every surviving `workspace` in `resources/` now means the Workspace, and every
surviving `gate` means version gated.

### For the prompt assembly ticket

The walker never loads Layer 1. Plan section 1.2 builds a system prompt from
`shared/execution-rules.md`, the Layer 3 refs and the Contract body, so
`AGENT.md` and `creator/CONTEXT.md` are navigation for a human or a router and
never reach a model through the walker. The empty
`resources/agents/CONTEXT.md` is an unwritten Layer 1 file, which is why it
blocks nothing.
