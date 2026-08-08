---
name: frr-intent-analyzer
description: Analyzes user intent and builds a structured briefing. Use this skill whenever the user asks any configuration, troubleshooting, or conceptual question about BGP, OSPF, IS-IS, RIP, PIM, LDP, VRRP, BFD, static routes, or any FRR daemon — even if the question is short, informal, or doesn't explicitly mention "FRR" (e.g. "my neighbor won't come up", "how do I advertise a default route", "why isn't this route showing up"). This skill must produce a temporary .md file with devices, connections, a standardized summary, a resolution plan, and queries already rewritten for search. compatibility, requires a file-write tool (bash/file-write). If the agent doesn't have one available, produce the same output as text and let the orchestration layer persist the file.
---

# Intent Analysis — FRR RAG

This skill runs **EVERY TIME** the users asks for any network configuration question. Its goal isn't to answer the question — it's to turn a
networking question, often informal and incomplete, into a structured
briefing that any following step (retrieval, answer generation) can consume
without having to reinterpret natural language again.

Every time this skill runs, it produces **a single `.md` file** following
the pattern in the "Template" section below, and returns that file's path.

## When to run

Whenever the user's message asks for help with configuration, behavior, or
troubleshooting of anything that touches FRR — even indirectly. The user
doesn't need to say "FRR", "BGP", etc. explicitly: "my neighbor won't come
up" or "the route isn't showing up" already qualify.

Don't run this skill for questions that don't depend on the knowledge base
(e.g. "what is BGP" in general terms can pass straight through — but
troubleshooting, configuration, or any question about FRR-specific behavior
should go through this first).

## Steps

### 1. Classify the intent

Pick a `type` (more than one can apply, but pick the dominant one):

- `troubleshooting` — something isn't working as expected
- `configuration` — the user wants to know how to configure something from scratch
- `conceptual` — a question about behavior/differences between options, with no specific broken config
- `operational` — day-to-day commands (save config, debug, restart a daemon, etc.)

Also identify the likely protocol(s) and daemon(s). Use this protocol →
daemon mapping as reference:

| Protocol | Daemon(s) |
|---|---|
| bgp | bgpd |
| ospf (v2) | ospfd |
| ospf (v3/IPv6) | ospf6d |
| isis | isisd |
| rip / ripng | ripd / ripngd |
| pim | pimd / pim6d |
| ldp | ldpd |
| vrrp | vrrpd |
| bfd | bfdd |
| static | staticd |
| (general routing / RIB / kernel) | zebra |

If the question doesn't make the protocol clear, infer it from context
(symptoms, terminology used) and flag it as an assumption in section 7 of
the template.

### 2. Extract devices

List every device mentioned or implied in the question. A "device" here is
any router/host taking part in the scenario — including peers that don't
run FRR (an upstream, a Cisco router, etc.), because they matter for
diagnosis even though the knowledge base only covers the FRR side.

For each device, capture:
- **label**: how the user refers to it (e.g. "core", "R1", "the datacenter
  router"). If there's no name, invent a short, stable label (R1, R2...) and
  use it consistently through the rest of the document.
- **role**: core, edge, route-reflector, upstream, eBGP peer, CE, PE, etc.
  — whatever can be inferred.
- **platform**: FRR (default if not stated otherwise and context is clearly
  FRR), or another vendor (Cisco IOS, Juniper...), or "unknown".
- **relevant daemon(s)/protocol(s)** on that specific device.

If only one device is mentioned but the question clearly involves a peer
(e.g. "my BGP neighbor won't come up"), **include the peer as a second
device**, even without a name — label it "peer (unidentified)".

### 3. Extract connections

List every link/session between the devices from step 2:
- **from / to**: which two devices
- **interface or link type**: interface name if mentioned (e.g. `Gi0/1`),
  or the session type if there's no relevant physical interface (e.g. "eBGP
  session", "L2 trunk")
- **protocol/session**: what runs there (e.g. "OSPF area 0", "eBGP AS 65001
  <-> AS 65010", "PIM sparse-mode")

If the question doesn't describe a topology (e.g. a purely conceptual
question, with no concrete scenario), leave this section empty — don't
invent devices or connections that weren't stated or implied.

### 4. Write the standardized summary

Rewrite the user's request in 2–4 sentences, in precise, unambiguous
technical English, as if explaining the problem to another network engineer
who hasn't seen the original question. Preserve every concrete detail (ASNs,
IPs, interface names, FRR version if mentioned) — don't generalize to the
point of losing information useful for diagnosis.

### 5. Build the resolution plan

This is **not the final answer** — it's the sequence of hypotheses/checks
that guides which searches to run against the knowledge base, ordered from
most basic/likely to most specific. Adapt the shape to the intent type:

- `troubleshooting` → diagnostic sequence: from the most common,
  easiest-to-check cause to the most specific (e.g. daemon enabled → L3
  reachability → protocol parameters matching on both sides →
  timers/authentication → protocol-specific logs/debug).
- `configuration` → prerequisite sequence: what needs to exist/be decided
  before the next step (e.g. ASN/router-id defined → basic session →
  advertisement/filter policy → redundancy).
- `conceptual` → list of sub-topics the final answer will need to cover to
  be complete.
- `operational` → the steps of the procedure itself.

3 to 6 items is typical. Each item should be concrete enough to become (or
suggest) a search query — avoid generic items like "check the
configuration".

### 6. Rewrite the query for RAG

For each item from step 5's plan that needs information from the docs,
produce a rewritten query:

- Translate informal language into FRR/RFC terminology (e.g. "neighbor
  won't come up" → "neighbor adjacency"; "my router won't talk to the
  other one" → depending on the protocol, "BGP session Active state" or
  "OSPF neighbor stuck Exstart").
- Prefer the exact command name when the user already signals which one it
  is (e.g. if they mention "redistribute", keep the term).
- One query per distinct aspect — don't try to cover the whole plan in a
  single string. Usually 2 to 5 queries.
- For each query, suggest metadata filters for the `frr_docs` table
  (`protocol`, `daemon`, and preferred `chunk_type` — `command_reference`
  when the question is about a specific command's syntax/behavior,
  `concept` when it's about general protocol behavior).

### 7. Record assumptions and gaps

List any inference you made without explicit confirmation from the user
(assumed protocol, assumed FRR version, assumed device role). This lets a
human or later step correct it without reprocessing the original question.

### 8. Save the file - Mandatory

Use the `WriteFile` tool to save the result to:

```
./.agent/tmp/<slug>-<hash8>.md
```

- `slug`: up to 5 words from the original query, lowercase, hyphen-separated
  (same scheme as `chunk_id` — no accents, only `[a-z0-9-]`).
- `hash8`: first 8 chars of the SHA1 of the original query + timestamp, so
  it never collides between calls.

Create the directory if it doesn't exist. After saving, return the file's
path — the retrieval step consumes this file, not loose text in the
conversation.

## Template

```markdown
# Intent Analysis — FRR RAG

**Timestamp:** <ISO 8601>
**Original query:** "<user's literal text, unedited, in whatever language they used>"

## 1. Classification
- **Intent type:** <troubleshooting | configuration | conceptual | operational>
- **Protocol(s):** <e.g. ospf>
- **Daemon(s):** <e.g. ospfd>

## 2. Devices
| Label | Role | Platform | Daemon(s)/Protocol(s) |
|---|---|---|---|
| <R1> | <core> | <FRR> | <ospfd> |

## 3. Connections
| From | To | Interface/Link | Protocol/Session |
|---|---|---|---|
| <R1> | <R2> | <Gi0/1> | <OSPF area 0> |

## 4. Problem/goal summary
<2–4 sentences>

## 5. Resolution plan
1. <hypothesis/check 1>
2. <hypothesis/check 2>
3. ...

## 6. Rewritten queries for RAG
| # | Rewritten query | protocol | daemon | suggested chunk_type |
|---|---|---|---|---|
| 1 | "<query>" | <ospf> | <ospfd> | <command_reference|concept> |

## 7. Assumptions / gaps
- <assumption 1, if any>
```

## Full example

**User question** (submitted in Portuguese — kept verbatim in the "Original
query" field; everything else in the output is written in English, per this
skill's convention):

> "meu ospf não converge entre o core (R1) e o edge (R2), os dois rodando
> FRR, ligados pela Gi0/1 em area 0. o R2 também tem uma sessão eBGP com o
> upstream, AS 65010, pela Gi0/2 — isso pode ter alguma relação?"

**File produced** (`./.agent/tmp/frr_intent/meu-ospf-nao-converge-entre-60becac1.md`):

```markdown
# Intent Analysis — FRR RAG

**Timestamp:** 2026-07-14T15:20:00-03:00
**Original query:** "meu ospf não converge entre o core (R1) e o edge (R2), os dois rodando FRR, ligados pela Gi0/1 em area 0. o R2 também tem uma sessão eBGP com o upstream, AS 65010, pela Gi0/2 — isso pode ter alguma relação?"

## 1. Classification
- **Intent type:** troubleshooting
- **Protocol(s):** ospf (primary), bgp (context/possible interaction)
- **Daemon(s):** ospfd, bgpd

## 2. Devices
| Label | Role | Platform | Daemon(s)/Protocol(s) |
|---|---|---|---|
| R1 | core | FRR | ospfd |
| R2 | edge | FRR | ospfd, bgpd |
| upstream (AS 65010) | eBGP peer | unknown/non-FRR | bgp |

## 3. Connections
| From | To | Interface/Link | Protocol/Session |
|---|---|---|---|
| R1 | R2 | Gi0/1 | OSPF area 0 |
| R2 | upstream | Gi0/2 | eBGP AS 65010 |

## 4. Problem/goal summary
The OSPF adjacency between R1 (core) and R2 (edge) isn't converging in area
0, over interface Gi0/1, both running FRR. R2 also holds a separate eBGP
session with an upstream (AS 65010) over Gi0/2. The user wants to know the
cause of the OSPF non-convergence and whether the eBGP session on R2 could
be related.

## 5. Resolution plan
1. Confirm ospfd is enabled and running on R1 and R2.
2. Check basic L3 connectivity between R1 and R2 on Gi0/1 (ping, MTU).
3. Verify R1 and R2 are in the same area with matching interface parameters
   (network type, cost, authentication).
4. Check which state the adjacency is stuck in (Init, 2-Way,
   Exstart/Exchange, Loading) — the state points to the likely cause.
5. Assess whether the eBGP session on R2 (Gi0/2) has any plausible relation
   to OSPF on R1/R2 — generally independent unless there's redistribution
   or a shared resource/CPU issue on R2.
6. If stuck in Exstart/Exchange, check for an MTU mismatch between the
   interfaces — a common cause of that specific state.

## 6. Rewritten queries for RAG
| # | Rewritten query | protocol | daemon | suggested chunk_type |
|---|---|---|---|---|
| 1 | "OSPF neighbor stuck Exstart Exchange state" | ospf | ospfd | concept |
| 2 | "OSPF interface authentication area configuration" | ospf | ospfd | command_reference |
| 3 | "OSPF interface cost network type mismatch" | ospf | ospfd | concept |
| 4 | "OSPF MTU mismatch neighbor adjacency" | ospf | ospfd | concept |
| 5 | "BGP OSPF redistribution interaction" | bgp | bgpd | concept |

## 7. Assumptions / gaps
- Assumed a recent FRR version (user didn't state one) — if behavior varies
  by version, confirm before applying the final answer.
- Not stated whether the adjacency partially forms or never leaves Down —
  this changes which plan item is the most likely first check; without that
  information, the plan covers from most generic to most specific.
```

### 8. Save the file
 
Use the `WriteFile` tool to save the result to:
 
```
./.agent/tmp/<slug>-<hash8>.md
```
 
- `slug`: up to 5 words from the original query, lowercase, hyphen-separated
  (same scheme as `chunk_id` — no accents, only `[a-z0-9-]`).
- `hash8`: first 8 chars of the SHA1 of the original query + timestamp, so
  it never collides between calls.
 
Create the directory if it doesn't exist. After saving, return the file's
path — the retrieval step consumes this file, not loose text in the
conversation.

## Design notes

- **Don't answer the question here.** This skill only structures; the final
  answer uses the results of the search run with the queries from step 6.
- **`protocol`/`daemon`/`chunk_type` in step 6 must match the real columns
  of the `frr_docs` table** (see the project's `schema.sql`) — they're used
  as direct filters in the hybrid query, not decorative metadata.
- Output is always written in English, regardless of the language the user
  asked in — the underlying FRR docs are English-only, so normalizing here
  keeps every downstream query consistent. The **"Original query" field is
  the one exception**: keep it verbatim, in whatever language the user
  actually used, for traceability.
- If the question spans multiple protocols, it's normal to have queries
  with different `protocol` values in the same step-6 table — the retrieval
  step runs one search per query.
- Keep the step-4 summary faithful to what the user actually said — this
  text, not the raw question, is what should show up if a human reviews the
  interaction later.
