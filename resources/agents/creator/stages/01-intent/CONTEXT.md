---
name: 01-intent
executor: llm
gate_when: questions.md
description: Turns the request into a structured briefing and lists what must be asked.
---

# Intent

Turns an informal request into a briefing every later stage can consume
without reinterpreting natural language again.

## Inputs

| Source | File/Location | Section/Scope | Why |
|--------|--------------|---------------|-----|
| Shared | `../../shared/execution-rules.md` | Full file | Boundaries and register |
| Tech pack | `../../tech-packs/{{TECH}}/TECH.md` | Full file | Scope, daemon set, vocabulary |
| Reference | `references/required-facts.md` | Full file | What may be assumed, what must be asked |
| User | Current turn | Full text | The request itself |

## Process

1. State the objective in one sentence: what the artifact must do.
2. List every device involved, including the peer the user did not name. For
   each: label, role, platform (FRR by default), relevant daemons. With no
   name given, invent a short stable label (`R1`, `peer-upstream`) and use it
   for the rest of the run.
3. List every session or link: from whom to whom, type, interface when stated.
4. List known facts: only what the user said or what a file that was read
   contains.
5. Cross check against `references/required-facts.md`. Split what is missing
   into must-ask and may-assume.
6. Save to the stage folder.

Design nothing. Write no config. Suggest no syntax. This stage understands.

## Checkpoints

| After Step | Agent Presents | Human Decides |
|------------|----------------|---------------|
| 5 | `questions.md`, if it has any entries | The missing identity facts |

Empty `questions.md` means no pause: the run proceeds to stage 02.

## Audit

| Check | Pass Condition |
|-------|---------------|
| Peer completeness | Every session names two devices, even if one is `peer (unidentified)` |
| No invention | No ASN, prefix, address, or keychain name appears that the user did not supply |
| Question reasons | Every entry in `questions.md` states which later stage it unblocks |
| Label stability | Every device label used in sessions also appears in the device list |

## Outputs

| Artifact | Location | Format |
|----------|----------|--------|
| Briefing | `brief.json` | `objective`, `devices`, `sessions`, `known_facts`, `assumptions` |
| Questions | `questions.md` | One question per missing fact, each with a `why:` line. Empty file if none |
