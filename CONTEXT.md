# Network Agent

An LLM agent for network engineering with FRRouting, with context retrieval
through RAG (PostgreSQL + pgvector) and executable skills.

This glossary started with the logging subsystem and now covers the ICM
pipeline as well. Other domain terms enter as they are settled.

## Language

### ICM

The Interpretable Context Methodology (arXiv 2603.16021) splits an agent into
layered context. The terms below are this repo's reading of it, which differs
from the methodology in one place: ICM keeps a run's artifacts inside the
definition tree, and this repo writes them to a separate one. See
[ADR 0002](./docs/adr/0002-agent-and-workspace-are-different-trees.md).

**Agent**:
The versioned, read-only definition of one pipeline, at
`resources/agents/<name>/`. Holds layers 1 to 3: routing, stage contracts and
reference material. Nothing writes into it at run time. The Go package
`internal/llm/agent` is the LLM client, not an Agent in this sense.
_Avoid_: workspace, definition tree, agent folder

**Workspace**:
Where one run writes its artifacts, at `.agent/workspace/<agent>/<stage>/`.
Layer 4. One per Agent, one active Run at a time.
_Avoid_: space, output folder, run directory

**Pipeline**:
An Agent's stages in numeric order, the structure a Run traverses. A property
of the Agent, not of any one execution.
_Avoid_: flow, chain, workflow

**Run**:
One traversal of the Pipeline for one request, bound to a single tech pack
slug from start to finish. A request for another technology is another Run.
_Avoid_: pipeline, session, execution

**Stage**:
One numbered step of the Pipeline, named `NN-name`. Owns a folder in the Agent
holding its definition, and a folder in the Workspace holding its artifacts.
_Avoid_: step, phase, node

**Contract**:
The `CONTEXT.md` inside a Stage folder: frontmatter plus Inputs, Process,
Checkpoints, Audit and Outputs. Layer 2, and authoritative for what the Stage
reads and writes. The `CONTEXT.md` at the repo root is this glossary, not a
Contract.
_Avoid_: stage prompt, spec, instructions

**Layer**:
ICM's split of context by lifetime, numbered 0 to 4: 0 `CLAUDE.md`, which this
repo does not have; 1 the Agent's own `CONTEXT.md` routing; 2 the Stage
Contracts; 3 reference material, stable across Runs; 4 the artifacts of the
Run in flight. Only 3 and 4 appear in the Source column of an Inputs table.
_Avoid_: tier, level

**Tech pack**:
A per-technology reference collection at `tech-packs/<slug>/`, the same five
files in each. The slug picked at the start of a Run binds `{{TECH}}` for the
whole Run.
_Avoid_: module, plugin, technology folder

**Checkpoint**:
A row of a Contract's `## Checkpoints` table: a point where the Agent presents
something and a human decides. A Stage whose Contract carries that section
pauses the Run. Unrelated to a **version gated** feature, which is an FRR
feature that needs a given release.
_Avoid_: gate, approval gate, breakpoint

**Scope**:
The Section/Scope column of an Inputs row: how much of the target the Walker
extracts, drawn from a closed vocabulary such as `Full file`, `Titles only`,
`Named in <path>`, a heading in double quotes or a JSON field in backticks.
_Avoid_: selector, filter, range

**Artifact**:
A file a Stage produces, declared in its Contract's `## Outputs` table and
written to that Stage's folder in the Workspace.
_Avoid_: output, result, deliverable

**Executor**:
The Contract frontmatter field saying how a Stage runs: `llm` for a model
call, `script` for a dispatch to Go code with no model involved.
_Avoid_: handler, runner, mode

**Walker**:
The Go component in `internal/agents` that loads an Agent, assembles each
Stage's prompt from its Contract, runs the Stages in order and stops at
Checkpoints.
_Avoid_: orchestrator, engine, runner, driver

### Logging

**Lugar**:
The subsystem that emitted a log line, `MAIN`, `RAG`, `TOOLS`, named by hand
through `logger.Named("RAG")`. It is the third field of the console format.
_Avoid_: origin, source, caller, module

**Source**:
The file:line pair the call came from, derived automatically by the runtime.
Distinct from **Lugar**: it appears only in the JSON file, never in the
console.
_Avoid_: lugar, origin

**TRACE**:
A file-only destination channel, **not** a severity level. TRACE messages
ignore the configured `LOG_LEVEL`, always go to the file and never to the
console. See
[ADR 0001](./docs/adr/0001-trace-is-a-channel-not-a-level.md).
_Avoid_: trace level, verbose, lowest level

**Sink**:
A log destination with its own format and rules. There are exactly two: the
**console sink** (text, coloured, truncated values) and the **file sink**
(JSON, with source, values intact).
_Avoid_: handler, output, appender, writer
