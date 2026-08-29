# FRR Config Creator

One pipeline, one tech pack per run, six stages in numeric order.

## Tech Pack Selection

Read the request and pick exactly one slug. The slug binds `{{TECH}}` for the
whole run. A request for a different technology is a new run.

| Slug | Covers |
|------|--------|
| `bgp` | eBGP/iBGP, route reflectors, policy, communities, aggregation, multihoming, BFD over BGP |

If nothing fits, return `UNROUTABLE` with one sentence saying what was missing.
Do not guess.

## Task Routing

| Task Type | Go To | Description |
|-----------|-------|-------------|
| Understand the request | `stages/01-intent/CONTEXT.md` | Briefing plus the questions that must be answered |
| Ground it in the docs | `stages/02-evidence/CONTEXT.md` | Retrieval for the version of record, script |
| Decide the shape | `stages/03-plan/CONTEXT.md` | Blocks, daemons, security posture, rollback. Human approves |
| Emit the artifact | `stages/04-render/CONTEXT.md` | frr.conf and daemons from the approved plan |
| Check it mechanically | `stages/05-validate/CONTEXT.md` | Syntax and dangling references, script |
| Check it by judgment | `stages/06-review/CONTEXT.md` | Doctrine checklist plus operator commands |

## Shared Resources

| Resource | Location | Contains |
|----------|----------|----------|
| Execution rules | `shared/execution-rules.md` | Boundaries, version of record, output register. Every LLM stage loads it |
| Tech packs | `tech-packs/CONTEXT.md` | Routing into the per-technology reference collection |
| Onboarding | `setup/questionnaire.md` | Agent level defaults, asked once, never per run |
| Skills | `../skills/*/SKILL.md` | Executable procedures the script stages call |

## Context Rule

No stage reads everything. A stage loads only what the Inputs table of its own
CONTEXT.md declares. Reading on your own initiative is not allowed.
