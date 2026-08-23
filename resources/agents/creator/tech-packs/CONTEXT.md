# Tech Packs

Per-technology reference material. One folder per routing technology, same
four files in each, so a stage contract can name a file without knowing which
technology the run picked.

## Contents of a Pack

| File | Loaded By | Contains |
|------|-----------|----------|
| `TECH.md` | 01-intent, 03-plan | Scope, daemon set, run vocabulary |
| `doctrine.md` | 03-plan | Non negotiable security and performance rules |
| `syntax.md` | 04-render | Artifact shape and version gated feature list |
| `checklist.md` | 06-review | Verifiable pass/fail questions over the artifact |
| `examples/` | 04-render | Canonical shapes, short, values are placeholders |

## Available Packs

| Slug | Status |
|------|--------|
| `bgp` | complete |

## Adding a Pack

Create `tech-packs/<slug>/` with all four files plus `examples/`. Add a row to
the table above and to the selection table in the workspace `CONTEXT.md`. No
stage contract changes: they address the pack through `{{TECH}}`.
