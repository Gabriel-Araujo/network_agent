# Implementation Plan: ICM Creator Workspace

Status: Phase 0 done, contracts settled for Decision 4, Phases 1 to 6 not started
Workspace under implementation: `resources/agents/creator/`
Methodology reference: arXiv 2603.16021, `_core/CONVENTIONS.md` of
https://github.com/RinDig/Interpretable-Context-Methodology

The workspace definition is complete: 42 markdown files as of commit `e0f9b59`,
including 24 BGP examples. Nothing executes it yet. This plan covers the four
remaining items, plus one prerequisite the codebase forces.

---

## Phase 0: Unblock (prerequisite for Phase 1) -- DONE

`ValidateConfig` lived in `package main` at `cmd/utility/validator/validate.go`,
so no other package could import it. Stage 05 needs it.

| Step | Action | State |
|------|--------|-------|
| 0.1 | Move `validate.go` to `internal/frr/validate/validate.go`, package `validate` | done |
| 0.2 | Leave `cmd/utility/validator/main.go` as a thin CLI calling the new package | done |
| 0.3 | Move `WrapSnippet` and `contextWrappers` with it | done |
| 0.4 | Update whatever imports `WrapSnippet` today | no-op |

Rename-only: the sole content change is the `package` clause. Function bodies
are byte identical, so the CLI behaves exactly as before.

Step 0.4 turned out to be a no-op. The plan assumed the RAG chunker path
imported `WrapSnippet`; it does not. `cmd/utility/validator/main.go` was the
only caller of either symbol.

---

## Phase 1: The Walker

The critical path. Everything else gets easier once this exists.

New package: `internal/workspace/`

### 1.1 Types

> **Naming superseded** by [ADR 0002](../../docs/adr/0002-agent-and-workspace-are-different-trees.md),
> via [The two meanings of workspace](../../.scratch/icm-walker/issues/02-two-workspaces.md).
> The `Workspace` type below is now `Agent`, and the `Space` type in 1.3 is now
> `Workspace`. The package is `internal/agents`, not `internal/workspace`, and
> `Gate`/`GateWhen` are `Checkpoint`/`CheckpointWhen`. The shapes are unchanged.

```go
// Ref is one row of a stage Inputs table.
type Ref struct {
    Path  string // resolved, absolute
    Scope string // "Full file", or the section names to extract
    Retry bool   // true when the Source column says "Retry only"
}

type Stage struct {
    Name     string // "01-intent"
    Executor string // "llm" | "script"
    Dir      string // absolute, inside resources/
    Body     string // CONTEXT.md after the frontmatter
    Refs     []Ref  // Layer 3
    Working  []Ref  // Layer 4, rebased onto the agent workspace
    Gate     bool   // derived: the file has a "## Checkpoints" section
    GateWhen string // optional frontmatter: pause only if this file is non-empty
}

type Workspace struct {
    Root   string
    Tech   string // the bound {{TECH}} slug
    Shared string // contents of shared/execution-rules.md
    Stages []Stage
}
```

### 1.2 Functions

| Function | Does | Est. lines |
|----------|------|-----------|
| `Load(root, tech string) (*Workspace, error)` | Walks `stages/*/`, parses each CONTEXT.md, sorts by folder name | 30 |
| `parseFrontmatter(path)` | Reuses the split logic in `internal/skills/discover.go`. Extract it into a shared helper rather than copying it | 10 |
| `parseInputs(body, tech, wsDir string)` | Reads the `## Inputs` table, splits Layer 3 from Layer 4 on the Source column, substitutes `{{TECH}}`, rebases Layer 4 onto the agent workspace | 35 |
| `section(content, scope string)` | Pattern 4, over the closed vocabulary in 1.2b. Unknown token is a hard error at `Load` | 40 |
| `(s Stage) SystemPrompt(shared string)` | `shared` plus each Layer 3 ref, section-scoped, plus the contract body | 20 |
| `(s Stage) UserTurn(wsDir string)` | Each Layer 4 ref, wrapped in a named tag so the model can tell the artifacts apart | 20 |

### 1.2b Scope vocabulary

Resolved 2026-08-27. The Section/Scope column is a closed vocabulary,
validated at `Load`, documented in `AGENT.md`. Prose in that column was the
root cause of Decision 4, and two other rows had the same defect.

| Token | Target | Walker does |
|-------|--------|-------------|
| `Full file` | file | reads it |
| `Full text` | not a file | the live user turn |
| `All files` | directory | reads every file |
| `Titles only` | directory | readdir plus the first `# ` line of each, as a list |
| `Named in <path>` | directory | reads the artifact, resolves the one name inside this directory |
| `"A" and "B"` | markdown | the named headings |
| `` `field` `` | JSON | the named field |
| `Read by the script` | anything | nothing, the row is documentation for a script stage |

Double quotes mean a heading, backticks mean a JSON field. `Named in` rejects
anything resolving outside its own directory, and a name with no matching file
stops the run listing the valid names. A missing artifact counts as no
selection and the run continues without that input.

Layer classification rule: the Source column decides, per the table in
`AGENT.md`. `Shared`, `Tech pack`, and `Reference` are Layer 3 and resolve
against `resources/agents/<agent>/`. `Previous stage`, `Retry only`, and `User`
are Layer 4 and resolve against the agent workspace. An unrecognized Source
value is a hard error at `Load`, so a typo in a contract fails loudly instead
of silently dropping an input.

### 1.3 Agent workspace

> **Naming superseded**: `Space` below is the `Workspace` of ADR 0002. See 1.1.

Artifacts live at `.agent/workspace/<agent>/<stage>/`. One workspace per agent,
one active pipeline at a time, no per-run identity in the path.

```go
type Space struct {
    Dir   string // .agent/workspace/<agent>
    Agent string // "creator"
    Tech  string
}
func Open(agent, tech string) (*Space, error)
func (s *Space) StageDir(stage string) string // <Dir>/<stage>
func (s *Space) Clear() error                 // wipe before a new pipeline
```

Guard every write with `util.SafePath`.

`Slugify` and `Hash8` are no longer needed for pathing. Leave them in
`internal/skills/intent-analyser`, where the report naming still uses them.

**Consequence to decide:** a new request reuses the same folders, so it
overwrites the previous pipeline's artifacts. Options: refuse to start when the
workspace is non-empty and make the user clear it, wipe silently, or archive to
`.agent/workspace/<agent>/.archive/<timestamp>/`. Recommendation: refuse and
point at `status`, since the whole design assumes a human reviews each stage
and losing an approved plan without warning is the worst of the three.

### 1.4 Executors

```go
func (w *Workspace) Run(ctx, agent, space, userInput) error
```

Loop over the stages in order. Per stage:

| Executor | Action |
|----------|--------|
| `llm` | Build the system prompt and the user turn, call the model, write the declared Outputs |
| `script` | Dispatch on stage name to existing Go code, no model involved |

Script dispatch table:

| Stage | Calls |
|-------|-------|
| `02-evidence` | `ragretriever.Do(ctx, briefPath, cfg)` |
| `05-validate` | `validate.ValidateConfig(ctx, configText)` plus the new checks from Phase 4 |

An unknown script stage is a hard error at `Load`, not at run time.

### 1.5 Gates

`Gate` is derived from the presence of a `## Checkpoints` section, per the repo
convention. The pause condition:

- `GateWhen` empty: always pause. This is 03-plan.
- `GateWhen` set: pause only if that file exists and is non-empty. This is
  01-intent, which already carries `gate_when: questions.md` in its
  frontmatter.

On pause: print the stage output, hand control back to the REPL, and persist
pipeline state so the next user message resumes at the following stage.

### 1.6 Router

One LLM call at run start. The enum comes from `ls tech-packs/`, the
descriptions from the first line of each `TECH.md`. Follows the pattern already
in `skills.BuildTools()`. Returns one slug, or `UNROUTABLE`.

With a single pack installed this is a no-op. Ship it anyway: Phase 3 makes it
real with no code change.

### 1.7 Wiring

`cmd/cli/main.go` today calls `intentanalyser.Do`, then `runRetrieval`, then
`agent.Chat`, on every message. Replace with: if no run is active, route and
start one, otherwise resume the active run.

Keep the old path behind a flag until the Phase 6 evaluation is done
(Decision 3, resolved).

Constraint on how: no existing function is edited. `runRetrieval` and
`loadRetrievalJSON` stay exactly as they are. The pipeline arrives as new code,
and only the call site inside the REPL loop changes, into a branch on the flag.
If the flag is later dropped, those two functions become unreferenced and stay
in place as dead code rather than being deleted.

Acceptance:

- `Load` against the real workspace returns 6 stages in order, with the right
  executors and gates
- Table test: a stage contract fixture produces the expected system prompt byte
  for byte
- A full BGP run reaches `.agent/workspace/creator/06-review/review.md`
- A run that stops at the 03 gate resumes correctly on the next message

Estimated: 200 to 250 lines of Go plus tests. Larger than the 150 quoted
earlier, because section scoping (Pattern 4) and run state persistence were not
in that estimate.

---

## Phase 2: Triggers

Both are declared in `AGENT.md`, neither is implemented.

### 2.1 `status`

Scans the agent workspace and renders the ASCII pipeline diagram the
convention specifies:

```
Pipeline Status: creator (bgp)

  [01-intent] --> [02-evidence] --> [03-plan] --> [04-render] --> [05-validate] --> [06-review]
   COMPLETE         COMPLETE         WAITING        PENDING          PENDING          PENDING
  (brief.json)   (evidence.json)   (plan.md)       (empty)          (empty)          (empty)
```

Per stage: files present means COMPLETE, list them. Empty means PENDING. Paused
at a gate means WAITING.

```go
func (s *Space) Status() []StageStatus
```

Estimated: 40 lines.

### 2.2 `setup`

Reads `setup/questionnaire.md`, asks the eight questions conversationally in a
single pass, collects the answers, and writes them somewhere the stages read.

Answers live in `.agent/config/` (Decision 1, resolved, option B). The
`resources/` tree stays immutable versioned definition; nothing writes into it.

Write `.agent/config/workspace.json` and load it as an extra Layer 3 ref, by
special-casing it in `SystemPrompt` next to `shared/execution-rules.md` rather
than adding a row to every Inputs table. One line, and the contracts stay
clean.

Estimated: 60 lines.

---

## Phase 3: Tech Packs

Pure authoring. No code changes, because stages address packs through
`{{TECH}}`.

Each pack needs the five files described in `tech-packs/CONTEXT.md`.

### 3.1 `tech-packs/ospf/`

| File | Content |
|------|---------|
| `TECH.md` | Scope, daemons (`zebra`, `ospfd`, `ospf6d`, `bfdd`), vocabulary (area, LSA, DR/BDR, stub/NSSA), verification commands |
| `doctrine.md` | Authentication per area or per interface, passive-interface by default on non-transit links, reference-bandwidth set explicitly, area sizing, BFD |
| `syntax.md` | Block order, `router ospf` vs `router ospf6`, where `network` and `area` statements belong, version gated list |
| `checklist.md` | The verifiable form of the doctrine |
| `examples/` | One canonical shape: area 0 plus a stub area |

Do not translate the BGP doctrine. The two are genuinely different:
passive-interface-by-default has no BGP analogue, and BGP inbound/outbound
prefix filtering has no OSPF analogue.

### 3.2 `tech-packs/evpn-vxlan/`

Do this one last, and expect it to break the one-pipeline assumption. EVPN
needs the underlay decided before the overlay, which is a stage the other packs
do not have. When that happens, use the escape hatch: allow
`tech-packs/<slug>/stages/` to override a stage of the same name. Do not build
the override until EVPN actually demands it.

Acceptance per pack: a full run produces a config that `vtysh -C` accepts, and
stage 06 finds no violated checklist item.

Estimated: half a day per pack, authoring plus one verified run.

---

## Phase 4: Validation Gaps

The 05-validate contract declares four checks. One exists.

| Check | Status | Action |
|-------|--------|--------|
| Syntax against the version of record | Implemented, via `vtysh -C` | None |
| Dangling references | Missing | Implement. Regex the defined names (`ip prefix-list X`, `route-map X`, `bgp community-list X`, `key chain X`) against the referenced ones. Around 40 lines, mechanical, no judgment required |
| IP/ASN/VRF consistency | Missing | Do not implement here. It needs judgment, not a regex. Move it into `tech-packs/bgp/checklist.md` as a stage 06 item and delete it from the 05 contract |
| No per-daemon file emitted | Missing | Implement. One directory listing check, 5 lines |

### `vtysh` absent (Decision 2, resolved)

Degrade, never hard-fail. The agent has no permission to execute anything: it
produces files and explains how to use them. A missing `vtysh` is therefore an
ordinary degraded path, not a run-ending error.

| Concern | Behavior |
|---------|----------|
| `vtysh` not on PATH | `report.json` records `syntax: unchecked` with the reason. The run continues to stage 06 |
| Stage 06 delivery | Carries a `RISKS:` entry saying syntax was not machine checked, plus the exact command the operator runs to check it themselves |
| Checks 2 and 4 | Pure Go, no external binary, so they still run and still report |

This mirrors how 02-evidence degrades when retrieval fails.

The Execution Boundary section of `shared/execution-rules.md` was rewritten on
2026-08-27 to state this directly: the agent produces files and explains how to
use them and how to prepare the environment, never runs a command, and never
asks for permission to. The section also separates the agent from the pipeline,
so the stage 05 script running a local checker no longer reads as a
contradiction.

Files edited: `stages/05-validate/CONTEXT.md`, `tech-packs/bgp/checklist.md`,
`internal/frr/validate/`.

Estimated: 80 lines plus the contract edits.

---

## Phase 5: Verify the Canonical Example

`tech-packs/bgp/examples/ebgp-peer.md` was written from long-stable FRR syntax
and documentation-range addresses (RFC 5737, RFC 5398). It has never been run
through a binary.

| Step | Action |
|------|--------|
| 5.1 | Install FRR, or pull a container image. Only the `vtysh` binary is needed, no daemons |
| 5.2 | Extract the fenced block and run `vtysh -C -f` against it |
| 5.3 | Fix whatever fails. Watch `ttl-security hops`, `maximum-prefix ... restart`, and `password` in particular: all three have varied across versions |
| 5.4 | Record the verified version in a footer line of the example |
| 5.5 | Add a Go test that runs every `examples/*.md` fenced block through `ValidateConfig`, skipped when `vtysh` is absent |

Step 5.5 is the one that matters long term: it makes every future tech pack
example self-verifying.

Estimated: half a day including the test. The original 2 hour estimate assumed
one example; `tech-packs/bgp/examples/` holds 18 after the Decision 4 cleanup,
and step 5.3 has to be paid per file that fails. `reference/` is out of scope
for this phase: no stage loads it, and two of its files are not FRR syntax.

---

## Phase 6: Evaluation

No control arm. Resolved 2026-08-27: the monolith is not finished, so it is
not a reference to measure against, and comparing prompt sizes is not a
present concern.

Worth recording, because it is easy to misread: the monolith is not merely a
document. `internal/llm/agent/prompts/system-prompt.md` is embedded by
`//go:embed` at `internal/llm/llm.go:17` and is byte identical to
`resources/agents/prompts/system-prompt-v1b.md`. The unfinished file is the
one running today.

The three quality metrics were always absolute and stay:

| Metric | How |
|--------|-----|
| `vtysh -C` pass rate on the first render | Higher is better |
| Checklist violations found by stage 06 | Lower is better |
| Human edits at the 03 gate | Proxy for plan quality |

Input tokens per run is recorded as a number, not as a ratio against another
arm. A fixed set of roughly 20 BGP requests.

---

## Ordering

```
Phase 0 ──> Phase 1 ──> Phase 2 ──> Phase 3 ──> Phase 6
                   └──> Phase 4 ──────┘
Phase 5 (independent, start it early)
```

Phase 5 has no code dependency and de-risks the BGP pack, so start it in
parallel with Phase 0. Phase 3 comes late: authoring a second pack against an
unproven pipeline wastes the work if the pipeline shape changes.

---

## Decisions

Resolved 2026-08-27.

| # | Decision | Answer | Lands in |
|---|----------|--------|----------|
| 1 | Where `setup` answers are stored | `.agent/config/`, `resources/` stays immutable | Phase 2.2 |
| 2 | `vtysh` missing | Degrade. The agent never executes; it writes files and explains how to use them | Phase 4 |
| 3 | Old `cmd/cli/main.go` path | Behind a flag. New code only, existing functions untouched, dead code tolerated | Phase 1.7 |

| 4 | How 04-render selects one example | 03-plan names exactly one, the walker resolves it. See below | Phase 1 |

### Decision 4, in full

Resolved 2026-08-27 across five rounds. The sub-decisions:

| Question | Answer |
|----------|--------|
| Who selects | 03-plan, because it already stops at a human gate where the choice can be reviewed and edited |
| How many | Exactly one. The example teaches form; the plan, `syntax.md` and `evidence.json` already carry content |
| How 03 sees the inventory | The walker generates the title list. A hand kept `INDEX.md` would rot |
| How the choice travels | `example.md`, a name plus a `why:` line, mirroring `questions.md` in 01-intent |
| Nothing fits | Legitimate. No name, the `why:` line records it, 06 surfaces the risk |
| Name does not exist | Hard error at resume, listing the valid names. Model error is not the same as human choice |
| Name escapes the directory | Same hard error. The value comes from model output and reaches a file read |
| Where the check lives | A new Audit row in 03 enforces the `why:` line |
| The wider fix | The whole Scope column becomes a closed vocabulary, 1.2b |

Two rows had the same defect and are fixed with it: `Passages for the blocks
being emitted` in 04 became `Full file`, since filtering belongs in the
retriever's top-k where the data is produced; and `objective, sessions,
daemons` in 02 became `Read by the script`, which also removed a `daemons`
field that the `brief.json` schema never had.

Pack content moved as a consequence. `examples/` is now defined as pasteable
`frr.conf` shapes only, and went from 24 files to 18. A new `reference/`
holds 7: two that are not FRR at all (`metallb-bgp-kubernetes`, YAML, and
`anycast-exabgp-healthcheck`, ExaBGP), four that the pack declares out of
scope in `TECH.md` (`evpn-vxlan-ebgp-fabric`, `mpls-l3vpn-vpnv4`,
`bgp-labeled-unicast`, `vrf-route-leaking`), and the FlowSpec half of the old
`rtbh-and-flowspec`, whose RTBH half stayed as `examples/rtbh.md`. Those four
were inert while nothing selected an example; selection makes them reachable,
and a run bound to `bgp` could otherwise have rendered EVPN against a
checklist that does not cover it.

`reference/` is declared in `tech-packs/CONTEXT.md` as loaded by no stage.
When Phase 3 creates the `evpn-vxlan` pack, its seed file moves from here into
that pack's `examples/`.

### Still open

| # | Decision | Blocks |
|---|----------|--------|
| 5 | Dirty workspace guard, open since 1.3: a new request reuses the same folders and overwrites the previous pipeline. The recommendation there is to refuse and point at `status` | Phase 1 |
| 6 | Stage 06 emits operator commands but nothing requires environment preparation, which `shared/execution-rules.md` now promises. `examples/bgp-unnumbered-ecmp.md` already shows the pattern, recording sysctls as `!` comments inside the artifact | Phase 3 |
