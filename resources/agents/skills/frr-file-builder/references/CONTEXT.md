# Tech Packs

Per-technology reference material of the `frr-file-builder` skill. One folder
per routing technology, the same files in each, so a reader can name a file
without knowing which technology the request touches. Two readers load it: the
skill body, through `read_skill_file`, and the stage contracts of the builder
pipeline (`../../../builder/CONTEXT.md`), through `{{TECH}}`.

The shape below is local to this skill. The teacher keeps its own, in
`../../../teacher/tech-packs/CONTEXT.md`.

## Contents of a Pack

| File | Loaded By | Contains |
|------|-----------|----------|
| `TECH.md` | skill, 01-intent, 03-plan | Scope, daemon set, vocabulary, required facts, example index |
| `syntax.md` | skill, 04-render | Artifact shape and syntax rules |
| `doctrine.md` | skill, 03-plan | Session policy, non negotiable security and performance rules |
| `checklist.md` | 06-review | Verifiable pass/fail questions over the artifact |
| `examples/` | skill, 03-plan, 04-render | Canonical shapes of `frr.conf`, short, values are placeholders. The skill reads the one closest to the request; 03 sees the titles and names one, 04 loads that one |
| `reference/` | none | Material that is not a `frr.conf` shape: other daemons, other formats, technologies the pack declares out of scope. Human reading only |

## Available Packs

| Slug | Status |
|------|--------|
| `bgp` | complete |

## Adding a Pack

Create `<slug>/` with all four files plus `examples/`, and list every example
in the "Examples" section of its `TECH.md`: the model picks an example by the
description there, so an unlisted one is only ever picked blind. Add a row to
the table above, to the pack table in `../SKILL.md`, and to the selection table
in `../../../builder/CONTEXT.md`. No stage contract changes: they address the pack
through `{{TECH}}`.

`reference/` is optional. Create it only when the pack has material worth
keeping that no reader should load. Every file in `examples/` has to be a
pasteable `frr.conf` shape; anything else belongs in `reference/`.
