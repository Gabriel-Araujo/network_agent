# How an LLM stage produces its declared artifacts

Type: grilling
Status: resolved

## Question

Every LLM stage declares an Outputs table with one or more artifacts, and the
model returns one response. Nothing decides how that response becomes those
files.

- `01-intent`: `brief.json` and `questions.md`
- `03-plan`: `plan.md` and `example.md`
- `04-render`: `frr.conf`, plus `daemons` and `vtysh.conf` only when the plan
  called for them
- `06-review`: `review.md`

The one existing implementation does not satisfy its own contract.
`intentanalyser.Do` writes a single `<slug>-<hash8>.json` through a bare
`os.WriteFile` (`internal/skills/intent-analyser/analyser.go:58,71`). No
`questions.md`, wrong name, wrong folder.

Candidate mechanisms, all present in the repo already: tool calls, since
`internal/tools/file` and `skills.BuildTools()` exist; structured output; a
fenced-block convention the walker parses; or one model call per declared
artifact.

The answer has to cover the conditional case, where `04-render` emits
`daemons` only if the plan said enablement changes, and the empty case, where
`questions.md` is written empty on purpose so the gate does not fire.

## Answer

An LLM stage writes its own artifacts through **tool calls**, against a
stage-scoped tool set the walker owns. The contract is enforced after the fact
by verification, not before the fact by a schema.

### Mechanism

Tool calls (Q1). Structured output was the alternative and was rejected: it
would have made the contract mechanically binding, but forcing `plan.md` and
`review.md` through JSON string fields costs prose fidelity on the two longest
markdown artifacts in the pipeline.

### The tool set

Two tools, `write` and `edit`, both confined to `<space>/<stage>/` (Q11).
No read, no search, no `use_skill`, no sub-agent (Q12).

The walker pre-assembles every input, so a stage that needs to read from disk
is a stage whose assembly dropped an input. Better that it fails visibly in
verification than that an opportunistic read covers it. On the ICM premise the
contract is the procedure, so a stage that can activate an arbitrary skill has
two instruction sources competing, and the byte-for-byte prompt test loses its
footing.

Tool names may stay `WriteFile` and `EditFile`, but the descriptions are new.
The current `READ_TOOL_DESCRIPTION` orders the model to call
`SearchFile(".", ...)` before any read (`internal/tools/file/prompts.go:17-19`),
which pulls it straight out of the stage folder.

### Why the existing tools cannot be reused as they are

Three findings, each verified in the code:

1. `util.SafePath` (`pkg/util/util.go:11-25`) returns a path **relative to**
   `workingDirectory`, and `WriteFile` and `EditFile` hand that relative path to
   `os.WriteFile` and `os.Stat`, which resolve against the **process cwd**. So
   `workingDirectory` only feeds the traversal check and never places the file.
   Passing `.agent/workspace/creator/04-render` writes `frr.conf` into the repo
   root. It works today only because the cwd is already the intended root.
2. `tools.CallFunction` overwrites whatever the model passed with
   `args["workingDirectory"] = "."` (`internal/tools/registry.go:48`).
3. `Agent.ToolCall` dispatches to `tools.CallFunction` unconditionally
   (`internal/llm/agent/agent.go:108-110`), so `Agent.Chat` cannot carry a
   different tool set.

Scoping by parameter is therefore off the table, and the standing preference
forbids editing those functions. The walker brings its own scoped tools and its
own loop over `responses.New` (Q7). `filetools` and `Agent.Chat` stay untouched.

**Scope consequence:** the walker does not reuse `Agent.Chat`. Its own loop is
roughly 60 lines on top of the plan's 200 to 250 estimate for Phase 1.

### The declared artifact list

Parsed from the `## Outputs` table at `Load`, symmetric with `parseInputs`
(Q2). The Format column carries prose today ("Only when the plan called for
it"), which is the exact defect behind Decision 4, so it gets the treatment
plan section 1.2b gave the Section/Scope column: a closed vocabulary,
`Always` and `Conditional`, validated at `Load`, a typo failing loudly.
Documented in `AGENT.md` next to the scope vocabulary. Frontmatter was
rejected: it would duplicate the table and drift from it.

The walker injects a generated directive block into the system prompt, built
from the parsed table, naming the exact files to write (Q9). The contracts keep
saying "Save to the stage folder" and stay declarative; the instruction lives in
one place instead of being restated in four contracts. Where that block sits in
the assembled prompt belongs to the assembled-prompt ticket, not here.

### Conditional and empty artifacts

A `Conditional` artifact that is not emitted is declared, not merely absent
(Q3): the model states in its final message which one it skipped and why. That
declaration is logged and shown at the gate; it never becomes an artifact,
since `plan.md` is prose and the walker cannot verify the condition
mechanically.

Every `Always` artifact is written even when empty (Q4). `01-intent` writes a
zero-byte `questions.md` when there is nothing to ask, and the gate fires on
non-empty. The walker does not auto-create it: absence means failure, and the
injected block from Q9 orders its creation explicitly. This also keeps
`status` (Phase 2.1) honest, where files-present reads as COMPLETE, so a stage
that ran and produced nothing stays distinguishable from one that never ran.

### Verification, after the loop

- Every `Always` artifact present, else hard error naming stage and artifact
  (Q5). No repair round trip, no continuing: a stage that gated on a broken
  artifact is worse than a stopped run.
- Any file in the stage folder that the Outputs table does not declare is a
  hard error listing the name, mirroring the `04-render` audit line "nothing
  unplanned emitted" (Q10).

### Retry

The scoped `write` overwrites rather than refusing (Q8). The existing
`filetools.WriteFile` refuses an existing path (`write.go:19`), which would
block the second pass of `04-render`, and clearing the stage folder instead
would contradict the contract's "fix only the items in `report.json`. No
wholesale rewrite". `edit` stays available for surgical fixes.

On a retry the walker injects the stage's own previous artifacts into the user
turn. This exposed a contract gap: the `04-render` Inputs table has no row for
its own prior output, so the model is told to fix `frr.conf` without being
shown it. Whether that becomes a `Retry only` row or stays walker-implicit is
[The retry input row for 04-render](08-retry-input-row.md).

### Consequence for the existing implementation

`01-intent` runs through the generic LLM path (Q6). `intentanalyser.Do` and its
`systemPrompt` and `skillPrompt` become unreferenced and stay in place as dead
code, per plan Decision 3. Wrapping it would have preserved the bug this ticket
opened with, a stage that cannot satisfy its own contract, and the contract
already carries the six process steps its `skillPrompt` encodes. `Slugify` and
`Hash8` stay live for the report naming.

This narrows charting Q5: the walker wraps existing skills for the two
**script** stages, `02-evidence` and `05-validate`. It does not wrap them for
LLM stages.
