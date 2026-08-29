# Report: Dynamic System Prompt Construction via ICM

Date: 2026-08-23
Branch: `develop`
Scope: design session plus authoring of `resources/agents/creator/`
Code changed: none. This was a design and authoring pass only.

---

## 1. Starting Point

The request was a mechanism for building system prompts dynamically: classify
the first message of a session into a few words, use those words to select
folders under `resources/`, and assemble a system prompt from them. The stated
inspiration was arXiv 2603.16021.

State of the repo at the start:

| Path | State |
|------|-------|
| `resources/agents/AGENT.md` | 6 lines, identity paragraph only |
| `resources/agents/CONTEXT.md` | empty |
| `resources/agents/traces/{creator,editor,intentAnalyser}/` | `AGENT.md` and `CONTEXT.md`, all empty |
| `resources/agents/prompts/system-prompt-v1b.md` | 221 lines, the working monolith |
| `resources/agents/skills/frr-file-builder/SKILL.md` | 4 line stub |
| `resources/agents/skills/frr-intent-analyzer/SKILL.md` | complete |
| `resources/references/examples/BGP.md` | 877 lines of BGP scenarios |
| `internal/agent.go` | a second, older system prompt hardcoded in Go |

---

## 2. What the Paper Actually Says

The abstract page gave only a summary. The full text at
`arxiv.org/html/2603.16021v2` and the reference implementation at
`github.com/RinDig/Interpretable-Context-Methodology` gave the mechanics.

Paper: "Interpretable Context Methodology: Folder Structure as Agentic
Architecture", Van Clief and McDermott, March 2026. The protocol is the Model
Workspace Protocol (MWP), MIT licensed.

Core mechanism, five context layers:

| Layer | File | Question |
|-------|------|----------|
| 0 | `CLAUDE.md` | Where am I? |
| 1 | `CONTEXT.md` | Where do I go? |
| 2 | `stages/0N-name/CONTEXT.md` | What do I do? |
| 3 | reference material | What rules apply? |
| 4 | `output/` of prior stages | What am I working with? |

Every stage CONTEXT.md carries the same three-section contract: Inputs,
Process, Outputs. Numbering encodes execution order. Folder boundaries enforce
separation of concerns. The load-bearing rule is "no agent reads everything".

### Correction made during the session

The first design proposal treated the paper as endorsing lateral selection: a
classifier picks a bag of tag folders. It does not. The paper is about
sequential staging, and the classifier reintroduces exactly the nondeterminism
the methodology removes. This was flagged before any files were written, and
the design was redirected when the user asked for something closer to the
paper.

---

## 3. Design Decisions Taken

Three decisions were put to the user and answered.

| # | Question | Answer | Consequence |
|---|----------|--------|-------------|
| 1 | Human review gate, or run straight through? | Gate | One mandatory pause after the plan stage |
| 2 | Missing facts become questions, or assumptions? | Questions | Identity facts (ASN, prefix, peer address, keychain) are never guessed |
| 3 | Technology as a Layer 3 pack, or a full workspace per technology? | Pack | One pipeline, N packs, bound by `{{TECH}}` |
| 4 | Where do stage artifacts go? | `.agent/workspace/<agent>/<stage>/` | Definition and execution split into separate trees, and the `output/` segment disappears |

Decision 4 came late and changed more than it looks. Stock MWP puts `output/`
inside `stages/0N-name/`, which works when one human drives one workspace and
the tree is both definition and scratch space. Here `resources/agents/creator/`
is versioned read-only definition, so artifacts have to live elsewhere.
Contracts still read as if paths were local, and the walker rebases them.

Two consequences worth recording:

**The `output/` segment is gone entirely.** A contract now names
`../01-intent/brief.json`, not `../01-intent/output/brief.json`. On the
execution side there is nothing in a stage folder except artifacts, so the
segment carried no information.

**That broke the layer classification rule.** The walker was going to split
Layer 3 from Layer 4 by testing whether a path contained `/output/`. With the
segment gone, the marker is gone. The replacement is the Source column, which
every Inputs table already carries and which the ICM template put there for
exactly this: `Shared`, `Tech pack`, and `Reference` are Layer 3, while
`Previous stage`, `Retry only`, and `User` are Layer 4. The mapping is now
written down in an `## Input Sources` table in `AGENT.md` instead of living
implicitly in a path convention, which is the better outcome.

**One workspace per agent means one active pipeline.** There is no run
identity in the path, so a second request overwrites the first one's
artifacts. This is fine for a local CLI with a human gate, and it matches the
protocol's own assumption, but it needs a guard. Left as an open decision in
the plan, with a recommendation to refuse a new pipeline while the workspace
is dirty rather than silently discarding an approved plan.

---

## 4. What Was Built

`resources/agents/creator/`, 14 markdown files.

```
creator/
├── AGENT.md                                    L0  navigation
├── CONTEXT.md                                  L1  routing
├── setup/questionnaire.md                          workspace onboarding
├── shared/execution-rules.md                   L3  cross-stage constraints
├── tech-packs/
│   ├── CONTEXT.md                                  routing within the collection
│   └── bgp/
│       ├── TECH.md
│       ├── doctrine.md
│       ├── syntax.md
│       ├── checklist.md
│       └── examples/ebgp-peer.md
└── stages/
    ├── 01-intent/CONTEXT.md
    ├── 01-intent/references/required-facts.md
    ├── 02-evidence/CONTEXT.md
    ├── 03-plan/CONTEXT.md
    ├── 04-render/CONTEXT.md
    ├── 05-validate/CONTEXT.md
    └── 06-review/CONTEXT.md
```

### The pipeline

| # | Stage | Executor | Gate | Produces |
|---|-------|----------|------|----------|
| 01 | intent | llm | conditional | `brief.json`, `questions.md` |
| 02 | evidence | script | none | `evidence.json` |
| 03 | plan | llm | **human, always** | `plan.md` |
| 04 | render | llm | none | `frr.conf`, `daemons` |
| 05 | validate | script | none | `report.json` |
| 06 | review | llm | none | `review.md` |

Stage 05 failing returns the run to 04 with the report attached, capped at two
round trips. On the third the run stops: a persistent failure means the plan is
wrong, not the render, and the fix belongs in stage 03.

### Where the v1b monolith went

The workspace is mostly a slicing of `system-prompt-v1b.md`, not new writing.

| v1b section | Destination |
|-------------|-------------|
| Identity, §3.1 execution boundary, §3.2 version of record, §6 response style, §7 safety | `shared/execution-rules.md` |
| §1.3 security doctrine, §1.4 performance doctrine | `tech-packs/bgp/doctrine.md` |
| §2.1 BGP knowledge, §2.3 domain scope | `tech-packs/bgp/TECH.md` |
| §5 tool usage rules, artifact shape | `tech-packs/bgp/syntax.md` |
| §4.1 P3 PLAN | `stages/03-plan/CONTEXT.md` |
| §4.1 P4 BUILD | `stages/04-render/CONTEXT.md` |
| §4.1 P5 REVIEW | `stages/05-validate` plus `stages/06-review` |
| §4.1 P6 DELIVER | `stages/06-review/CONTEXT.md` Process step 3 |
| §1.1 classification A/B/C/D | **deleted** |
| §1.2 TODO / PIPELINE CHECK machinery | **deleted** |
| §4.2 diagnostics, §4.3 audit, §4.4 mixed | **out of scope**, this workspace only creates |

The two deletions are the substantive change. Roughly 45 lines of v1b exist
only so a monolithic prompt can police its own step ordering. The walker
enforces order structurally: the model never sees the next stage, so it cannot
skip one. That machinery is not needed and was dropped.

A related simplification: the model no longer chooses when to activate a skill.
The walker decides, because the order is in the folder names. One
nondeterministic model decision removed.

---

## 5. Two Rewrites

The workspace was written three times. Both rewrites were user-driven and both
were correct.

### Rewrite 1: language

First draft was in Portuguese. The user asked for English. This was the right
call and matches the repo: `system-prompt-v1b.md` and all three `SKILL.md`
files are already in English. Portuguese remains where it belongs, in the
domain glossary at repo-root `CONTEXT.md` and in Go comments, neither of which
was touched.

### Rewrite 2: conventions

The user asked for something closer to the reference repo. Fetching
`_core/CONVENTIONS.md` and the three templates showed the first version
violated the conventions in eight ways.

| Convention | Before | After |
|------------|--------|-------|
| Folder naming | `01_intent` | `01-intent`, lowercase-with-hyphens |
| Placeholder syntax | `{TECH}` | `{{TECH}}` |
| Inputs section | bullet list | table: Source, File/Location, Section/Scope, Why |
| Outputs section | bullet list | table: Artifact, Location, Format |
| Human gates | frontmatter `gate:` field | `## Checkpoints` table |
| Quality checks | prose | `## Audit` table |
| Layer 0 shape | doctrine inline | Folder Map, Triggers, Routing, What to Load, Stage Handoffs |
| Em dashes | around 40 | 0, an explicit repo guardrail |

Two deeper corrections came out of the conventions, both real design errors:

**Pattern 6, CONTEXT.md is routing and not content.** Identity, execution
boundary, version of record, and output register were piled into `AGENT.md`.
They moved to `shared/execution-rules.md` as a single canonical source that
every LLM stage loads. `AGENT.md` is now navigation only. This also fixed a
Pattern 5 violation, since the same rules had begun to be restated in
individual stage contracts.

**Pattern 8, questionnaires are system-level and not per-run.** The first
`questionnaire.md` asked for ASN, prefixes, and peer addresses. Those are
per-run facts, and the convention collects those conversationally in the entry
stage. The file split in two:

- `stages/01-intent/references/required-facts.md`, per run, what stage 01 must
  ask about versus what it may default
- `setup/questionnaire.md`, asked once, covering what is genuinely stable
  across runs: production FRR version, organization ASN, owned blocks, policy
  object naming convention, whether an RPKI validator exists, TCP-AO versus
  MD5 preference, output location, and whether the plan gate is mandatory

---

## 6. Measurements

Assembled system prompt per stage, in lines. Navigation (`AGENT.md` plus
`CONTEXT.md`, 109 lines) is excluded: it is read once at run start for routing,
and the walker resolves the rest, so it does not enter per-stage prompts.

| Stage | shared | tech pack | contract | total |
|-------|--------|-----------|----------|-------|
| 01-intent | 50 | 94 | 59 | **203** |
| 02-evidence | - | - | - | **0** |
| 03-plan | 50 | 91 | 65 | **206** |
| 04-render | 50 | 82 | 52 | **184** |
| 05-validate | - | - | - | **0** |
| 06-review | 50 | 38 | 42 | **130** |

Control: `system-prompt-v1b.md`, 221 lines, on every stage.

### Correction to an earlier number

An intermediate measurement reported 267 lines for 01-intent, worse than the
monolith. That measurement was wrong: it counted the 109 navigation lines in
every stage, which is not how the walker will assemble prompts. A second script
had a substitution bug, replacing `{{TECH}}` with `tech-packs/bgp` instead of
`bgp`, which silently zeroed the tech pack column. The table above is from the
corrected script.

### Honest reading of these numbers

The reduction in raw line count is modest, roughly 7 to 40 percent depending on
stage. The reduction is not the main effect. Two things matter more:

1. **Which lines.** Stage 01 never sees FRR syntax. Stage 04 never sees the
   questionnaire. Stage 06 sees the checklist and not the examples. This is the
   "no agent reads everything" rule doing the work that a tag classifier was
   originally proposed to do, except declared in a file instead of decided by a
   model.
2. **Two of six stages have no prompt at all.** Evidence and validate are
   script stages. Their cost is zero tokens and zero latency.

Neither effect is proven better than the monolith. Phase 6 of the
implementation plan exists to measure it.

### Convention compliance

Verified mechanically after the final rewrite:

- em dashes: 0
- CONTEXT.md files over the 80 line limit: 0, range is 27 to 65
- reference files over the 200 line limit: 0, range is 38 to 69
- underscores or spaces in any path: 0

---

## 7. Defects Found and Fixed During Authoring

**Canonical example contradicted its own doctrine.** The first
`examples/ebgp-peer.md` included `no bgp ebgp-requires-policy`, which disables
the FRR guard requiring a policy on eBGP sessions. The pack's `doctrine.md`
mandates filtering in both directions, and the example already had it, so the
line was both unnecessary and directly contrary to the doctrine the same pack
defines. Removed, and replaced with an explicit note that the default stays on.

**Destructive shell command silently no-opped.** A `rm -rf creator && mkdir ...`
chain failed on the `rm` because a file was open in the IDE, and `&&`
short-circuited the rest. The directory contents were already deleted, so the
tree was empty and nothing was recreated. Caught by verifying with `find`
instead of assuming success. No content was lost, since everything was
regenerated. Worth noting as a pattern: on Windows with files open in an IDE,
`rm -rf` on a directory can partially succeed.

---

## 8. Gaps in the Existing Code That the Contracts Expose

Reading the codebase to write the implementation plan surfaced two mismatches
between what the contracts declare and what exists.

**`ValidateConfig` is unreachable.** It sits in `package main` at
`cmd/utility/validator/validate.go`. Stage 05 needs it and cannot import it.
It has to move to `internal/`. Phase 0 of the plan.

**Stage 05 declares four checks and one exists.** `ValidateConfig` shells out
to `vtysh -C`, which covers syntax only. Dangling references, IP/ASN/VRF
consistency, and the no-per-daemon-file check are all unimplemented. The plan
recommends implementing two of them mechanically, and moving IP/ASN/VRF
consistency into the stage 06 checklist because it needs judgment rather than a
regex. That requires editing the 05 contract, which currently overstates what
the stage does.

Also noted: `vtysh` must be on PATH for stage 05 to work at all, and the
behavior when it is absent is an open decision in the plan.

---

## 9. What Is Reusable Versus New

Of the six stages, three already have working implementations.

| Stage | Existing code |
|-------|---------------|
| 01-intent | `internal/skills/intent-analyser`, complete |
| 02-evidence | `internal/skills/rag-retriever`, complete |
| 05-validate | `cmd/utility/validator`, partial, see section 8 |
| 03-plan | none |
| 04-render | `frr-file-builder/SKILL.md` is a 4 line stub |
| 06-review | none |

The walker itself also reuses more than it invents: frontmatter parsing from
`internal/skills/discover.go`, the enum-catalog-from-directory pattern from
`skills.BuildTools()`, and
`util.SafePath` for write guarding.

---

## 10. State and Next Step

Nothing executes. The workspace is a complete definition with no runtime.

The implementation plan is at `.agent/plans/icm-creator-workspace.md`, covering
seven phases. The critical path is Phase 1, the walker, estimated at 200 to 250
lines of Go. Phase 5, verifying the canonical BGP example against a real
`vtysh` binary, has no code dependency and should start in parallel.

Three decisions are still open and are listed at the end of the plan: where
`setup` answers are stored, what happens when `vtysh` is missing, and whether
the existing `cmd/cli/main.go` path stays behind a flag during cutover.

Untouched and possibly now redundant: `resources/agents/traces/creator/` still
holds two empty files that were the original sketch of this same Layer 0 and
Layer 1 pair. The older hardcoded prompt in `internal/agent.go` is also still
live and unrelated to this workspace.
