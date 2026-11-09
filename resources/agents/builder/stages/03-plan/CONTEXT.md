---
name: 03-plan
executor: llm
description: Decides the shape of the configuration and submits it for human approval.
---

# Plan

Decides what the configuration will contain. No line of `frr.conf` comes out
of here: the plan names blocks, not commands.

## Inputs

| Source | File/Location | Section/Scope | Why |
|--------|--------------|---------------|-----|
| Shared | `../../shared/execution-rules.md` | Full file | Boundaries and register |
| Tech pack | `../../tech-packs/{{TECH}}/TECH.md` | "Daemons" and "Run vocabulary" | Which daemons are defensible |
| Tech pack | `../../tech-packs/{{TECH}}/doctrine.md` | Full file | The posture the plan must declare against |
| Tech pack | `../../tech-packs/{{TECH}}/examples/` | Titles only | The canonical shapes available to name |
| Previous stage | `../01-intent/brief.json` | Full file | What the user asked for |
| Previous stage | `../02-evidence/evidence.json` | Full file | What the version of record supports |

## Process

1. Files touched: `frr.conf` always, `daemons` or `vtysh.conf` only if daemon
   enablement changes. Say which and why.
2. Daemons: which stay enabled. Every daemon beyond the pack's minimum gets a
   one line justification.
3. Blocks: enumerate what stage 04 will emit, in the order it will appear.
   One item per block, no syntax.
4. Security posture: walk `doctrine.md` item by item. Declare each as met,
   not applicable with a reason, or a regression.
5. Performance posture: every timer, limit, or threshold gets a numeric value
   and its expected effect.
6. Rollback, differentiated by artifact. `frr.conf` only: restore the file
   and hot reload, no restart. `daemons` or `vtysh.conf`: restore and restart
   the daemon, which drops every session on it. Flag the second as high
   impact here, not only at delivery.
7. Version notes: each version gated feature the plan uses, with the evidence
   that confirms it.
8. Canonical shape: name the one example from the title list whose shape
   this plan follows, and say why in one line. Nothing fits is a valid
   answer: leave the name out and let the `why:` line say what the pack does
   not cover.
9. Save to the stage folder.

## Checkpoints

| After Step | Agent Presents | Human Decides |
|------------|----------------|---------------|
| 4 | `SECURITY WARNING` banner, if any item came out a regression | Whether to accept the regression |
| 9 | `plan.md` in full, plus `example.md` | Approve, or edit either file directly |

The run always stops at step 9. Stage 04 consumes whatever is on disk at
resume, so a human edit becomes the plan. A name in `example.md` that no file
in `examples/` matches stops the run at resume and lists the valid names.

## Audit

| Check | Pass Condition |
|-------|---------------|
| Doctrine coverage | Every item in `doctrine.md` has a verdict, none skipped |
| No syntax | No FRR command appears anywhere in `plan.md` |
| Numbers justified | Every timer or threshold states its effect and why that value |
| Rollback impact | The rollback section names session loss explicitly when daemons change |
| Example justified | `example.md` carries a `why:` line, with or without a name |

## Outputs

| Artifact | Location | Format |
|----------|----------|--------|
| Plan | `plan.md` | The seven sections above, in that order |
| Shape | `example.md` | The example filename, or nothing, plus one `why:` line |
