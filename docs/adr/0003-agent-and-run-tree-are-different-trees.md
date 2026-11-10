# Agent and Run tree are different trees

Supersedes [ADR 0002](./0002-agent-and-workspace-are-different-trees.md). The
split 0002 made stands: the ICM definition of a pipeline is versioned and
read-only, the artifacts a run produces are neither, and this repo keeps them in
two trees. Only the second name changes. **Agent** is still the definition at
`resources/agents/<name>/`; what 0002 called **Workspace** is now the **Run
tree**, at `.agent/runs/<agent>/<stage>/`. `Workspace` is freed, and names the
folder the user picks in the desktop app: the single root the file tools may
write to.

## Considered Options

Inventing a word for the user's folder (`Project`, `Bench`, `Lab`) and leaving
`Workspace` where 0002 put it was the alternative. It was rejected because the
owner already calls that folder the workspace, so an invented term would fight
the house usage forever — and because 0002's own argument favours the trade:
"the two names need to say which tree is which" is better served by `Agent tree`
× `Run tree` than by the pair 0002 chose, where only one of the two names says
anything about its tree.

The ceded term cost nothing to cede. 0002 planned `agents.Workspace` in a
package `internal/agents`; neither the package nor the type was ever written,
and no `.agent/workspace/` directory exists. The rename is documents only.

## Consequences

**0002's narrowing of ICM's vocabulary dissolves.** 0002 recorded as a cost that
it narrowed `workspace`, which in the methodology means the whole thing,
definition and outputs together. `Run tree` makes no claim on that word, so a
reader holding the methodology and this repo side by side no longer has to
reconcile two meanings of it — they have to learn one new name instead.

**`Workspace` now means the user's folder, and it is the only write root.** Its
counterpart is the **Agent root**, the read-only root where agent material lands
on a machine. Which of the two resolves a path is decided by the path's prefix,
not by a search order (see
[Desenho da pasta de escrita](../../.agent/ui/issues/05-desenho-do-workspace-de-escrita.md)).

**One line of the rename was not mechanical.** `resources/agents/teacher/AGENT.md`
told the Teacher to never browse "the Workspace" looking for something to
explain. Under the new meaning that sentence forbids exactly what the Teacher
exists to do — read the user's files — so it was rewritten to say what it meant
(don't go hunting for a file nobody named) instead of having its path swapped.
