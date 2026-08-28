# Tech Packs

Per-technology reference material for this workspace. One folder per routing
technology, the same files in each, so a stage contract can name a file
without knowing which technology the run picked.

The shape below is local to the creator. Another agent with a `tech-packs/`
folder defines its own in its own `CONTEXT.md`.

## Contents of a Pack

| File | Loaded By | Contains |
|------|-----------|----------|
| `TECH.md` | 01-intent, 03-plan | Scope, daemon set, run vocabulary |
| `doctrine.md` | 03-plan | Non negotiable security and performance rules |
| `syntax.md` | 04-render | Artifact shape and version gated feature list |
| `checklist.md` | 06-review | Verifiable pass/fail questions over the artifact |
| `examples/` | 03-plan, 04-render | Canonical shapes of `frr.conf`, short, values are placeholders. 03 sees the titles and names one, 04 loads that one |
| `reference/` | none | Material that is not a `frr.conf` shape: other daemons, other formats, technologies the pack declares out of scope. Human reading only |

## Available Packs

| Slug | Status |
|------|--------|
| `bgp` | complete |

## Adding a Pack

Create `tech-packs/<slug>/` with all four files plus `examples/`. Add a row to
the table above and to the selection table in the workspace `CONTEXT.md`. No
stage contract changes: they address the pack through `{{TECH}}`.

`reference/` is optional. Create it only when the pack has material worth
keeping that no stage should load. Every file in `examples/` has to be a
pasteable `frr.conf` shape; anything else belongs in `reference/`.
