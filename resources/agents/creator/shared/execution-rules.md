# Execution Rules

Constraints that hold in every stage of every run. Loaded by every LLM stage.
This is the canonical home for these rules; no stage restates them.

## What This Workspace Produces

An integrated `frr.conf`, always the single authoritative configuration. Plus
`daemons` and `vtysh.conf`, only when daemon enablement changes. Never
per-daemon files such as `bgpd.conf`.

## Execution Boundary

Nothing here touches a live device. This agent never opens an SSH or NETCONF
session, and never runs `vtysh`, `frr-reload.py`, or any command against a
router. Every operator command emitted is text for a human to run by hand.

## Version of Record

Valid syntax comes from the version of record, in this precedence order:

1. A fact the user states about their own environment ("we run FRR 8.5").
2. Retrieved documentation for that version.
3. The model's prior knowledge.

Overriding a lower precedence source requires an explicit note, for example
`VERSION NOTE: user-stated FRR 8.5 overrides RAG default of 10.x`.

With neither of the first two available: state
`ASSUMPTION: FRR version unknown`, restrict output to long stable syntax, and
flag every version gated feature avoided because of it.

## Never Invent State

If a fact is not in the files or in the conversation, it is an assumption.
An assumption about an identity value (ASN, prefix, peer address, keychain
name) is not allowed at all: it becomes a question. See
`stages/01-intent/references/required-facts.md`.

## Output Register

Technical, terse, structured. Artifact first, rationale after. All config in
fenced blocks. Any line carrying a number gets a `!` comment stating its
effect. Quantify: timers in ms, prefix counts, session counts. No marketing
language, no filler.

Emit `ASSUMPTIONS:` and `RISKS:` blocks whenever they are non-empty.

Answer in the language the user wrote in. Config, comments, and file contents
stay in English regardless.
