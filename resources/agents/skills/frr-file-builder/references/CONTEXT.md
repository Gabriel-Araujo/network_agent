# Tech Packs

Per-technology reference material of the `frr-file-builder` skill. One folder
per routing technology, the same files in each, so a reader can name a file
without knowing which technology the request touches. Two readers load it: the
skill body, through `read_skill_directory` and `read_skill_file`, and the stage
contracts of the builder pipeline (`../../../builder/CONTEXT.md`), through
`{{TECH}}`.

The shape below is local to this skill. The teacher keeps its own, in
`../../../teacher/tech-packs/CONTEXT.md`.

## Contents of a Pack

| File | Loaded By | Contains |
|------|-----------|----------|
| `TECH.md` | skill, 01-intent, 03-plan | Scope, daemon set, vocabulary, required facts, example index |
| `syntax.md` | skill, 04-render | Artifact shape and syntax rules |
| `doctrine.md` | skill, 03-plan | The protocol's policy section (Session Policy, Adjacency Policy, Profile Policy), non negotiable security and performance rules |
| `checklist.md` | 06-review | Verifiable pass/fail questions over the artifact |
| `examples/` | skill, 03-plan, 04-render | Canonical shapes of `frr.conf`, short, values are placeholders. The skill lists the pack and reads the shape closest to the request, from here or `reference/`; 03 sees the titles and names one, 04 loads that one |
| `reference/` | skill | Broader shapes the skill may pick as the closest one: other daemons, other formats, technologies the pack declares out of scope. Each file opens saying so when it is outside the pack's scope or not FRR syntax |

## Available Packs

| Slug | Status |
|------|--------|
| `bgp` | complete |
| `ospf` | complete |
| `ospf6` | complete |
| `isis` | complete |
| `bfd` | complete |
| `evpn` | complete |

## Adding a Pack

Create `<slug>/` with all four files plus `examples/`, and list every example
in the "Examples" section of its `TECH.md`: the model picks an example by the
description there, so an unlisted one is only ever picked blind. A pack file
names another by its path from the skill root (`references/<slug>/examples/<name>.md`):
the model passes that path to `read_skill_file` as written, and a path relative
to the pack folder does not exist there. Add a row to
the table above, to the pack table in `../SKILL.md`, and to the selection table
in `../../../builder/CONTEXT.md`. No stage contract changes: they address the pack
through `{{TECH}}`.

A pack is built from the FRR corpus (`resources/frrouting/docs/*.rst`), never from memory:
every example passes `vtysh --dryrun`, every line of the Shape and of the examples is a
`clicmd` of the corpus, and every rule of `doctrine.md` and `checklist.md` cites its section
as `<rst> › <section>`. `tests/internal/llm/packs_test.go` checks all three.

`reference/` is optional. Create it only when the pack has broader shapes
worth keeping. The skill reads them, so a file outside the pack's scope opens
saying so, and one that is not FRR syntax says that. Every file in `examples/`
has to be a pasteable `frr.conf` shape; anything else belongs in `reference/`.
