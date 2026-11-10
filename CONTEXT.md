# Network Agent

An LLM agent for network engineering with FRRouting, with context retrieval
through RAG (PostgreSQL + pgvector) and executable skills.

This glossary started with the logging subsystem and now covers the ICM
pipeline, the conversation history, classification and skills as well. Other domain terms enter as
they are settled.

## Language

### ICM

The Interpretable Context Methodology (arXiv 2603.16021) splits an agent into
layered context. The terms below are this repo's reading of it, which differs
from the methodology in one place: ICM keeps a run's artifacts inside the
definition tree, and this repo writes them to a separate one. See
[ADR 0003](./docs/adr/0003-agent-and-run-tree-are-different-trees.md).

**Agent**:
The versioned, read-only definition of one pipeline, authored at
`resources/agents/<name>/`. Holds layers 1 to 3: routing, stage contracts and
reference material. Nothing writes into it at run time. A running app reads it
from the **Agent root**, `~/.network_agent/resources/agents/<name>/`, mirrored
from the binary the material is embedded in — the read-only root of the pair
whose writable half is the **Workspace**. The Go package `internal/llm/agent`
is the LLM client, not an Agent in this sense.
_Avoid_: run tree, definition tree, agent folder

**Run tree**:
Where one run writes its artifacts, at `.agent/runs/<agent>/<stage>/`. Layer 4.
One per Agent, one active Run at a time. Not the **Workspace**, which is the
user's folder.
_Avoid_: workspace, space, output folder

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
holding its definition, and a folder in the Run tree holding its artifacts.
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
The reference material for one technology in a Subagente's `tech-packs/`
folder, named by a slug. Its shape is the one that folder's `CONTEXT.md`
defines: the Builder keeps a folder of files per slug, the Teacher a single
`<slug>.md`. In a Run, the slug picked at the start binds `{{TECH}}` for the
whole Run.
_Avoid_: module, plugin, technology folder, guide

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
written to that Stage's folder in the Run tree.
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
The subsystem that emitted a log line, `MAIN`, `UI`, `RAG`, `TOOLS`, named by
hand through `logger.Named("RAG")`. It is the third field of the console
format. Two subsystems never share one Lugar: `MAIN` is the CLI's and `UI` is
the desktop app's.
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
(JSON, with source, values intact). What defines a sink is its **rules**, not
its destination: the console sink writes to stderr **and** to the UI's Logs
drawer, and that is why the drawer shows exactly what the CLI shows.
_Avoid_: handler, output, appender, writer

### Conversation history

**Conversa**:
One exchange history with the agent: its ordered transcript, plus the provider
and model that produced it. Persisted in the app's data directory, so it
outlives the window. Distinct from a **Run**, which traverses a Pipeline.
_Avoid_: session, chat, thread

**Transcript**:
The ordered messages of a Conversa: the unit the app owns and replays whole on
every turn. Nothing about a Conversa lives only on the provider.
_Avoid_: history, log, message list

### Classification

**Classificador**:
The component that, given an input and one or more **Listas**, chooses one value from each — or
none, when nothing fits. With the **Modo RAG** flag on, it serves the RAG's classifier. It
replaces the classification role of the `intent-analyser` skill.
_Avoid_: intent analyser, router, tagger

**Lista**:
A closed set of candidate values for one dimension, supplied to the **Classificador** by whoever
calls it; one value is chosen per list, or none.
_Avoid_: opções, vocabulário, categorias

**Subagente**:
The persona that answers one turn with its own configuration: one of Builder, Modifier,
Illustrator or Teacher, chosen by the `subagent` **Lista** of the **Classificador** and
configured at `resources/agents/<name>/AGENT.md`. Exactly one answers a turn; an empty value may
buy a **Re-pergunta do roteamento** and, failing that, falls back to the generic agent.
_Avoid_: agente, agent, persona, roteador, router

**Re-pergunta do roteamento**:
The one extra classification a turn buys when the **Classificador** names no **Subagente** but
did fill some search filter — the input was understood as in-domain and only the routing
dimension came back empty. It classifies the `subagent` **Lista** alone, and its answer replaces
the empty one in the turn's classification; empty twice falls back to the generic agent with one
warning naming both attempts.
_Avoid_: retry, votação, segunda chance, correção

**Modo RAG**:
The **Classificador**'s flag-on mode, in which it serves as the RAG's classifier instead of a
generic caller. The flag is a call parameter: the caller decides the mode.
_Avoid_: flag do RAG, RAG flag

**Filtros de busca**:
The values that scope one input's document search: the chosen value of each **Lista** passed by
the RAG caller, applied to the whole search and never to a single query. An empty value means
that dimension is not filtered.
_Avoid_: filtros por query, tags de busca

**Skip rule**:
The turn rule that decides whether the search runs: only when this turn's **Modo RAG**
classification differs from the previous turn's. A skipped turn re-injects the last search's
recovered context; a failed classification clears the comparison, so the search runs again.
_Avoid_: cache de busca, regra de cache, memoização

### Briefing

**Briefing rico**:
The structured picture a request paints of the network at hand: its devices, its connections, the
problem and goal summary, and the execution plan. Distinct from the classification the
**Classificador** makes and from the search queries.
_Avoid_: intent analysis, análise de intenção, briefing de intenção

**Espelho**:
The JSON file a skill writes beside its in-memory result, named
`<slug>-<hash8>.json` under `.agent/tmp` — a mirror, not the product: on a write
failure the value still comes back and the failure is a sentinel error. Both the
**Briefing rico** and the retrieval write one.
_Avoid_: output file, arquivo de saída, cache, log, backup

### Skills

**Skill**:
A specialized procedure packaged as a folder with a `SKILL.md` — a name, a description and a body
— in the common catalog. Its body reaches the model by one of two paths, **ativada** or
**montada**; the same skill can be montada for one **Subagente** and ativada by another.
_Avoid_: plugin, módulo, tool, procedimento

**Skill ativada**:
The path in which the model calls `use_skill` with the exact name and the body comes back as a
tool result. Before the call the model knows only the name and description in the tool's catalog,
so reading the body is the model's choice.
_Avoid_: skill carregada, discovery, ativação de tool

**Skill montada**:
The path in which the loader concatenates the body into a **Subagente**'s system prompt, so the
body is there before the turn starts and no call happens. Chosen when the content must not depend
on the model electing to read it.
_Avoid_: skill injetada, inlinada, pré-carregada, embutida

### Diagramas

**Bloco de diagrama**:
One ` ```mermaid ` fence of a turn's reply. The **Subagente** Illustrator emits them as its answer
and the Teacher may include them in an explanation; nothing writes them to a file. Every block of
a reply goes through the turn's scanner, whoever wrote it — the check is of the text, not of the
author.
_Avoid_: diagrama, mermaid, gráfico, imagem

**Bloco não reprovado**:
A **Bloco de diagrama** that the turn's scanner did not accuse. It is **not** a valid block: the
scanner is three narrow rules over the measured failure modes, not a Mermaid parser, so a block it
stays silent on may still fail to render. The asymmetry is deliberate — the scanner is calibrated
to never accuse a block that renders, and pays for it by missing modes it does not cover (a `note`
inside a flowchart, a hyphenated state id). Whoever consumes the metadata must read silence as
"these three rules found nothing", never as a render guarantee.
_Avoid_: bloco válido, bloco verificado, diagrama que renderiza, valid

**Bloco não renderizado**:
A **Bloco de diagrama** the client failed to draw: `mermaid.render` threw in the UI, so no SVG
exists and the bubble shows the parser's message plus the source as text. Orthogonal to **Bloco
não reprovado** — this one is the render on the client, that one is the turn's scanner on the
server. A block can be both, either or neither, and the two never imply each other: a block the
scanner stayed silent on may fail to render, and a block it accused may draw fine.
_Avoid_: diagrama quebrado, diagrama inválido, erro de mermaid, falha de render

### Arquivos

**Workspace**:
The folder the user picks in the desktop app: the single root the file tools may
write to, and the root a path from the user resolves against. Supplied by each
front-end — the UI from the user's choice, the CLI from its working directory.
Distinct from the **Run tree**, ICM's layer 4, and from the **Agent root**,
which is read-only. See
[ADR 0003](./docs/adr/0003-agent-and-run-tree-are-different-trees.md).
_Avoid_: run tree, project folder, output folder
