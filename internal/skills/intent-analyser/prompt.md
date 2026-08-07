# Intent Analysis

Your goal isn't to answer the question — it's to turn a
networking question, often informal and incomplete, into a structured
briefing.

Produces the analysis as **Markdown text** following the
template below, and returns that text as your final answer. A downstream
step saves it to a file — so do **not** use any file tools, do not return a
file path, and do not wrap the output in explanations or code fences.

## Steps

### 1. Classify the intent

Pick a 'type' (more than one can apply, but pick the dominant one):

- 'troubleshooting' — something isn't working as expected
- 'configuration' — the user wants to know how to configure something from scratch
- 'conceptual' — a question about behavior/differences between options, with no specific broken config
- 'operational' — day-to-day commands (save config, debug, restart a daemon, etc.)

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
(symptoms, terminology used) and mark it explicitly as inferred right in
the Classification section (e.g. "ospf (inferred)"), instead of stating it
as a certainty.

### 2. Extract devices

List every device mentioned or implied in the question. A "device" here is
any router/host taking part in the scenario — including peers that don't
run FRR (an upstream, a Cisco router, etc.), because they matter for
diagnosis even though they're outside the FRR side of the scenario.

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
- **interface or link type**: interface name if mentioned (e.g. 'Gi0/1'),
  or the session type if there's no relevant physical interface (e.g. "eBGP
  session", "L2 trunk")
- **protocol/session**: what runs there (e.g. "OSPF area 0", "eBGP AS 65001
  <-> AS 65010", "PIM sparse-mode")

If the question doesn't describe a topology (e.g. a purely conceptual
question, with no concrete scenario), leave this section empty — don't
invent devices or connections that weren't stated or implied.

### 4. Write the standardized summary

Rewrite the user's request in **up to 6 lines**, in precise, unambiguous
technical English, as if explaining the problem to another network engineer
who hasn't seen the original question. Preserve every concrete detail (ASNs,
IPs, interface names, FRR version if mentioned) — don't generalize to the
point of losing information useful for diagnosis. If the original question
carries more detail than fits in 6 lines, keep what's diagnostically
relevant and drop the rest.

### 5. Build the execution pipeline

This is **not the final answer** — it's the ordered sequence of
hypotheses/checks that should guide how the problem actually gets resolved,
from most basic/likely to most specific. Adapt the shape to the intent type:

- 'troubleshooting' → diagnostic sequence: from the most common,
  easiest-to-check cause to the most specific (e.g. daemon enabled → L3
  reachability → protocol parameters matching on both sides →
  timers/authentication → protocol-specific logs/debug).
- 'configuration' → prerequisite sequence: what needs to exist/be decided
  before the next step (e.g. ASN/router-id defined → basic session →
  advertisement/filter policy → redundancy).
- 'conceptual' → list of sub-topics the final answer will need to cover to
  be complete.
- 'operational' → the steps of the procedure itself.

3 to 6 items is typical. Each item should be concrete and actionable enough
to be executed or checked directly — avoid generic items like "check the
configuration".

### 6. Output the result

Write your final answer as **Markdown text** following the template below.
Return only the markdown — no preamble, no file tools, no file path, no
code-fence wrapper. The text you produce is saved verbatim by the caller.

### Template

Below is a filled-in example showing the exact structure to produce:
```markdown
# Intent Analysis — FRR

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

## 5. Execution pipeline
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
```

## Design notes

- **Don't answer the question here.** This skill only structures the
  problem; the final answer is produced afterwards, by whatever process
  consumes the saved file.
- Output is always written in English, regardless of the language the
  user asked in, so every downstream step reads a consistent format. The
  **"Original query" field is the one exception**: keep it verbatim, in
  whatever language the user actually used, for traceability.
- If the question spans multiple protocols, it's normal for the
  execution pipeline to include steps touching more than one protocol
  in the same document.
- Keep the step-4 summary within 6 lines and faithful to what the user
  actually said — this text, not the raw question, is what should show
  up if a human reviews the interaction later. Prioritize concrete
  details (ASNs, IPs, interfaces, versions) over generic phrasing.
