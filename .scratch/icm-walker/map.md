# Map: ICM Walker

Label: `wayfinder:map`
Tracker: local markdown. Tickets are `issues/NN-<slug>.md`.

## Destination

A walker that runs the creator pipeline end to end for BGP: a real request
produces `.agent/workspace/creator/06-review/review.md`, having stopped at the
03 gate and resumed. Two tests pass with it: `Load` returns the six stages in
order with the right executors and gates, and a table test that pins the
assembled system prompt of one stage byte for byte.

## Notes

Domain: the ICM walker for `resources/agents/creator/`, specified as Phase 1 of
`.agent/plans/icm-creator-workspace.md`. The stage contracts are authoritative
for Inputs and Outputs; the plan is authoritative for everything else.

**This map carries execution.** Wayfinder plans by default; this effort
overrides that. Decision tickets come first, the build lands when the way is
clear.

Every session should call the Skill tool for `grilling` and `domain-modeling`.

The root `CONTEXT.md` glossary is canonical for the vocabulary: Agent is the
definition tree, Workspace is where a run writes, the pause is a Checkpoint.
Use those words in tickets, prose and Go.

Standing preferences for this effort:

- New code only. Existing functions are not edited. A call site may be swapped,
  and code left unreferenced stays as dead code (plan Decision 3).
- The walker wraps the existing skills rather than changing their signatures.
  `ragretriever.Do` and `intentanalyser.Do` keep the shape they have; the
  walker moves their output into the stage folder (charting Q5).
- Markdown under `resources/agents/`: English, no em dashes, a `CONTEXT.md`
  stays under 80 lines, a reference file under 200.
- Answer the human in the language they wrote in.
- Go 1.26.3. Every build session calls the `modern-go-guidelines:use-modern-go`
  skill before editing Go, and follows what it returns even where nearby code
  uses an older pattern. Most likely to bite the walker: `errors_as_type`,
  `new_expression`, `json_omitzero`, `strings_cut`/`CutPrefix`,
  `strings_split_seq`, `slices_*`/`maps_*`, `range_over_int`, `cmp_or`,
  `any`, `errors_join`, `testing_t_context`.

## Decisions so far

<!-- one line per closed ticket -->

- [How an LLM stage produces its declared artifacts](issues/01-stage-artifacts.md):
  the stage writes its own files through tool calls, against a write-and-edit
  tool set the walker owns and confines to the stage folder; the Outputs table
  is parsed at `Load` and enforced by verification after the loop.
- [The two meanings of workspace](issues/02-two-workspaces.md): the definition
  tree is the **Agent**, the run's artifact tree is the **Workspace**, and the
  human pause is a **Checkpoint**; thirteen terms written into the root
  `CONTEXT.md`, the split recorded as ADR 0002, and the Go package is
  `internal/agents`.

## Not yet specified

- **The build itself.** How the walker is sliced into sessions still waits on
  the assembly shape, which decides what the units are. The artifact-writing
  half is settled.
- **What the REPL shows at a gate, and how resume reads the next message.**
  Waits on where run state lives.
- **The 05 to 04 retry loop.** The contract caps it at two round trips and
  stops on the third. Where the counter lives and who increments it waits on
  run state.
- **Degraded 05 at the walker level.** Plan Decision 2 says a missing `vtysh`
  degrades and the run continues. What the walker writes into `report.json`
  when the binary is absent is not yet sharp.
- **Test fixtures.** The shape of a stage contract fixture waits on the
  assembly prototype.

## Out of scope

- **The agent router.** `resources/agents/router/AGENT.md` routes between
  `creator` and `general` and runs above the walker. A walker that reaches the
  destination never needs to know `general` exists. Separate effort, and it
  collides in name with the tech pack router that is in scope (charting Q3).
- **Environment preparation in the 06 contract.** Plan Decision 6.
  `shared/execution-rules.md` now promises the agent explains how to prepare
  the environment, and no stage requires it. Belongs to Phase 3, where the rule
  can be applied across three packs at once (charting Q2).
- **Phase 4 validation checks**, beyond the `vtysh -C` that already exists.
  Dangling references and the per-daemon file check are not needed to reach the
  destination.
- **Phases 3, 5 and 6**: the `ospf` and `evpn-vxlan` packs, verifying the 18
  examples against a real `vtysh`, and the evaluation.
