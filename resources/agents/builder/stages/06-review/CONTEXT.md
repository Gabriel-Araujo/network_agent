---
name: 06-review
executor: llm
description: Runs the doctrine checklist over the artifact and assembles operator delivery.
---

# Review

Stage 05 said the file is valid. This stage says whether it is good, then
hands the operator what to run.

## Inputs

| Source | File/Location | Section/Scope | Why |
|--------|--------------|---------------|-----|
| Shared | `../../shared/execution-rules.md` | Full file | Boundaries and register |
| Tech pack | `../../tech-packs/{{TECH}}/checklist.md` | Full file | The pass/fail questions |
| Previous stage | `../04-render/` | All files | What is being judged |
| Previous stage | `../05-validate/report.json` | `findings[]` | What the mechanical pass already caught |
| Previous stage | `../03-plan/plan.md` | "Blocks" and "Rollback" | Conformance and the rollback path to publish |
| Previous stage | `../03-plan/example.md` | Full file | The shape the render was told to follow, and why |

## Process

1. Answer every item in `checklist.md` against the artifact as it stands:
   met, not applicable with a reason, or violated.
2. Compare artifact to plan: did every planned block appear, and did anything
   unplanned appear.
3. Assemble the operator commands as text for a human to run. Validate with
   `frr-reload.py --test`, apply with `frr-reload.py --reload`, verify with
   the `show` commands the tech pack names, roll back by the path the plan
   defined, restating its impact.
4. Consolidate `ASSUMPTIONS:` and `RISKS:` across the whole run, if non-empty.
5. Save to the stage folder.

A violated item is not fixed here. Report it and name the stage that fixes
it: 03 for a shape decision, 04 for an emission problem.

## Outputs

| Artifact | Location | Format |
|----------|----------|--------|
| Review | `review.md` | Checklist verdicts, plan conformance, operator commands, assumptions, risks |
