# Tech Packs

Per-technology support material for the teacher. One file per technology,
`tech-packs/<slug>.md`, which the teacher reads with `ReadFile` when a question
touches that technology.

The shape below is local to the teacher. The builder keeps its own, a folder of
files per technology, in `../../builder/tech-packs/CONTEXT.md`.

## Contents of a Pack

A pack is a single Markdown file named after its slug: a practical guide to the
technology, from concepts to FRR configuration and troubleshooting. Its sections
are free, since nothing addresses a pack by section: the teacher reads the file
whole. `bgp.md` is the reference.

`ReadFile` returns only the first `READ_MAX_CHARS_TO_READ` bytes of a file
(`internal/tools/file/prompts.go`), with no offset, so a pack must fit under
that limit: whatever lies past it never reaches the teacher.
`TestAgentResourcesFitTheReadCap` (`tests/internal/llm`) fails on a pack that
does not fit.

## Available Packs

| Slug | File | Status |
|------|------|--------|
| `bgp` | `bgp.md` | complete |

## Adding a Pack

Create `tech-packs/<slug>.md` and add a row to the table above. Then name the
file in the Behavior section of `../AGENT.md`: no code reads this folder, so a
pack the `AGENT.md` does not name is a pack the teacher never opens.
