---
name: 04-render
executor: llm
description: Emits frr.conf and daemons from the approved plan.
---

# Render

Turns the approved plan into artifacts. The plan already passed a human gate,
so this stage executes it rather than reconsidering it.

## Inputs

| Source | File/Location | Section/Scope | Why |
|--------|--------------|---------------|-----|
| Shared | `../../shared/execution-rules.md` | Full file | Boundaries and register |
| Tech pack | `../../tech-packs/{{TECH}}/syntax.md` | Full file | Block order and version gated list |
| Tech pack | `../../tech-packs/{{TECH}}/examples/` | The example matching the session type | Canonical shape |
| Previous stage | `../03-plan/plan.md` | Full file | What to emit |
| Previous stage | `../02-evidence/evidence.json` | Passages for the blocks being emitted | Syntax for the version of record |
| Retry only | `../05-validate/report.json` | `findings[]` | What failed, on a second pass |

## Process

1. Emit one section of `frr.conf` per block in the plan, in the plan's order.
2. Introduce nothing the plan does not list. Noticing a missing block, stop
   and name it. A render that invents invalidates the approval gate.
3. Give every line carrying a timer, limit, or threshold a `!` comment with
   its expected effect.
4. Reference secrets by keychain or password name, never by value.
5. Emit `daemons` only if the plan said enablement changes.
6. Save to the stage folder.

On a retry, fix only the items in `report.json`. No wholesale rewrite.

## Audit

| Check | Pass Condition |
|-------|---------------|
| Plan conformance | Every planned block emitted, nothing unplanned emitted |
| Declaration order | Every policy object is defined before the block referencing it |
| Activation | Every neighbor has `activate` in the address family it uses |
| Secrets | No password or key value appears in any output file |
| Annotation | Every line containing a number carries a `!` comment |

## Outputs

| Artifact | Location | Format |
|----------|----------|--------|
| Config | `frr.conf` | Integrated FRR configuration |
| Daemons | `daemons` | Only when the plan called for it |
| vtysh | `vtysh.conf` | Only when the plan called for it |
