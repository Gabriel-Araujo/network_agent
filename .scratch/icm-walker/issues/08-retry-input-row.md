# The retry input row for 04-render

Type: grilling
Status: open

## Question

Surfaced by [How an LLM stage produces its declared artifacts](01-stage-artifacts.md).

`04-render` on a retry is told to "fix only the items in `report.json`. No
wholesale rewrite", and its Inputs table already carries
`| Retry only | ../05-validate/report.json | findings[] | What failed, on a
second pass |`. It carries no row for the artifact being fixed. The model is
asked to correct `frr.conf` without being shown `frr.conf`.

Ticket 01 settled that the walker injects the stage's own previous artifacts
into the user turn on a retry. Open is whether that injection is **declared**
or **implicit**.

Declared means a row per artifact in the Inputs table,
`| Retry only | frr.conf | Full file | The artifact being corrected |`, with a
path that for the first time points at the stage's *own* folder rather than an
earlier one. The Layer 4 rebasing rule in plan section 1.2b resolves
`Previous stage`, `Retry only` and `User` against the agent workspace, so the
mechanism exists, but every path in it today is `../<other-stage>/`.

Implicit means the walker adds them silently whenever it re-enters a stage, and
no contract mentions them.

Declared keeps the contract the single source of truth for what a stage sees,
which is the premise the whole Inputs parser rests on. Implicit keeps the table
free of a row that is true only on a second pass and confusing on the first.

Whichever wins applies to any future retry loop, not just 04, so decide the
rule rather than the one case.
