# Agent and Workspace are different trees

The ICM definition of a pipeline is versioned and read-only; the artifacts a run
produces are neither. This repo keeps them in two trees, and the two need names
that say which is which. We decided that **Agent** is the definition at
`resources/agents/<name>/`, and **Workspace** is where one run writes its
artifacts, at `.agent/workspace/<agent>/<stage>/`.

## Considered Options

The implementation plan called the definition `Workspace` and the artifact tree
`Space` (`.agent/plans/icm-creator-workspace.md`, sections 1.1 and 1.3). It was
rejected because the two names differ by one word that reads as an abbreviation
of the other, so neither says which tree it is, and because the filesystem
already votes the other way: the path is `.agent/workspace/`, and
`creator/AGENT.md` already says "the agent workspace" for the artifact tree.

`Definition` instead of `Agent` was the runner-up. `Agent` won because
`resources/agents/` and `AGENT.md` already name the tree that way, and because
the collision with the Go package `internal/llm/agent` is smaller than it looks:
that package is the LLM client, and four of its five call sites already alias it
on import.

## Consequences

**This narrows a word ICM uses more widely.** In the methodology, a workspace is
the whole thing, definition and outputs together, because ICM writes artifacts
into the definition tree at `stages/0N-name/output/`. Here it means only the
second half. Anyone reading the methodology alongside this repo will hit that,
and the mismatch is the point of the split: the definition is versioned and the
outputs are not, so they cannot share a tree.

The names land in Go as the package `internal/agents`, with `agents.Agent` and
`agents.Workspace`. `agents.Agent` stutters mildly, accepted so that the package
mirrors `resources/agents/` on disk.

Plan sections 1.1 and 1.3 are superseded on naming only. Their `Workspace` type
is this ADR's `Agent`, and their `Space` type is this ADR's `Workspace`.
