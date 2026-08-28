---
name: 05-validate
executor: script
description: Mechanical validation of the rendered artifacts, syntax and references.
---

# Validate

Deterministic checks only. Says whether the file is valid, not whether it is
good. Judgment belongs to stage 06.

Implementation: `internal/frr/validate`.

## Inputs

| Source | File/Location | Section/Scope | Why |
|--------|--------------|---------------|-----|
| Previous stage | `../04-render/` | Read by the script | The artifacts under test |

## Process

1. Check syntax against the version of record.
2. Find dangling references: prefix-lists, route-maps, community-lists, and
   keychains that are cited and never defined.
3. Check IP, ASN, and VRF consistency across blocks.
4. Confirm no per-daemon file was emitted.

## Outputs

| Artifact | Location | Format |
|----------|----------|--------|
| Report | `report.json` | `pass` bool, `findings[]` with `severity`, `file`, `line`, `rule`, `message` |

## Loop

`pass: false` returns the run to stage 04 with the report added to its Inputs.
Cap of two round trips. On the third, the run stops and hands the report to
the user: a persistent failure means the plan is wrong, not the render, and
the place to fix that is stage 03.
