# Agent material is embedded in the binary and projected into the home

The agent's material — the **Agent** definitions at `resources/agents/` and the
reference examples at `resources/references/` — travels inside the binary
through `//go:embed` and is written to `~/.network_agent/resources/...` on the
first material read of each process. That mirror is the **Agent root**: the
single root every reader of agent material resolves against at run time
(`LoadSkills`, `ForSubagent`, and the file tool the Teacher's tech packs go
through). The embedded path and the mirrored path are identical, so the mirror
is a plain walk that writes back the same relative it read.

The mirror is a **projection, not a copy-if-missing**: a file is overwritten
whenever the embedded content differs from what is on disk. Nothing the user
owns lives there — the **Workspace** is the only write root — so there is
nothing to preserve and no conflict to negotiate. Agent material is customised
in the repo, not in the home.

Two consequences of where the material comes from: `internal/paths.RepoFile`,
which resolves paths from the *compile-time* location of a source file through
`runtime.Caller`, leaves the agent-material path entirely; and no turn depends
any more on the source tree being present on the machine that runs the app.

## Considered Options

**Reading the source tree, the status quo.** `paths.RepoFile` resolved both Go
readers and `workingDirectory="."` resolved the tech packs through the CWD. Both
were measured broken before this ADR: under `wails dev` the CWD is `cmd/ui`, so
the Teacher's `resources/agents/teacher/tech-packs/bgp.md` resolved to
`cmd/ui/resources/...` and the read was refused — in a live drive, with the RAG
search down, which is exactly when the tech pack is the fallback. In a built
binary the source tree is not there at all. It was rejected because it makes
agent material depend on a tree that only exists on a dev machine.

**Copy-if-missing, per file.** Rejected because it freezes the material on the
day of installation: a corrected `AGENT.md` never reaches a machine that already
has one, and the fix is a user deleting a folder nobody told them about. Adding
an "update material" button buys that back with UI surface plus a third state
(seeded-but-stale) to administer — state the overwrite does not need.

**Moving `resources/` into a Go package to embed it.** A `//go:embed` directive
cannot reach above its own directory, and there was no `.go` file at the module
root. The alternative to adding one was moving the two trees under a package,
which decouples the embedded path from the mirrored path — forcing the writer to
re-glue a prefix — and invalidates every `resources/agents/...` citation in the
docs, in `CONTEXT.md`, and in the `AGENT.md` files the model itself reads. The
module root became a package instead.

**Seeding eagerly in each front-end's `main`.** Rejected because it is one
caller per binary where the lazy path needs none: the seeding sits in a
`sync.OnceValues` inside the root resolver, so the CLI, the UI and every test
that reads material are covered without editing a `main`. An eager `main` would
still have needed the lazy path, since packages that call `LoadSkills` under
test have no `main` to seed from.

## Consequences

**The paths the model is taught stay literally true.** `resources/agents/teacher/AGENT.md`
teaches the tech-pack path as `resources/agents/teacher/tech-packs/`; because the
mirror preserves the embedded path, that sentence needs no edit — it now resolves
against the Agent root.

**An edit inside the mirror does not survive the next open.** This is the point,
not a limitation, but it will surprise someone who finds a copy of the repo's
material in their home and changes it.

**A seeding failure surfaces at the first material read, not at startup.** It
does not pass silently: the error is memoised with the root, so every later read
gets the same failure, and `LoadSkills` still dies loudly on it.

**Tests of other packages seed the real `~/.network_agent`.** Accepted cost. The
test of the projection itself overrides the home directory through the seam in
`internal/config`, so the mechanics are measured in a `t.TempDir()`.

**Two trees stayed out of the binary.** `resources/frrouting/` (9.7M) is the RAG
ingester's dev-time corpus, handed to it through `-dir`, and no turn reads it;
carrying it would put 9.7M in a user's home to serve nothing. And
`resources/agents/skills/frr-file-builder/scripts/validator` — 3.07M of dead
Mach-O on a Windows dev machine, a build artifact referenced by no document and
no line of Go — was deleted from the repo instead of excluded from the embed, so
the directive needs no exclusion rule at all.
