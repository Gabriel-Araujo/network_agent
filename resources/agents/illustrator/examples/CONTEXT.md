# Diagram Examples

Reference diagrams for the illustrator. One subject per file, `NN-<slug>.mmd`, which the
illustrator reads with `ReadFile` when a request touches that subject.

Every file here is MermaidJS that renders as written: the shape is the lesson — node ids, label
quoting, direction, amount of detail. The subject is not the answer. A request gets the diagram it
asked for, drawn the way the closest example draws its own.

## Contents of an Example

The first line is a `%%` comment holding the title, the second line declares the Mermaid type, and
the rest is the diagram. No frontmatter, no prose: nothing addresses an example by section, since
the illustrator reads the file whole.

Every file is under 1 KB, so the `ReadFile` cap (`READ_MAX_CHARS_TO_READ`,
`internal/tools/file/prompts.go`) is never in play. `TestAgentResourcesFitTheReadCap`
(`tests/internal/llm`) walks `.md` only, so an example past the cap would fail no gate: keep them
small.

## Available Examples

| File | Shows | Type |
|------|-------|------|
| `01-frr-daemon-architecture.mmd` | FRRouting daemon architecture and zebra | `flowchart TD` |
| `02-fib-route-installation.mmd` | Route installation into the FIB | `flowchart TD` |
| `03-bgp-best-path-selection.mmd` | BGP best path selection, simplified | `flowchart TD` |
| `04-route-map-processing.mmd` | route-map processing | `flowchart TD` |
| `05-bgp-finite-state-machine.mmd` | BGP finite state machine | `flowchart LR` |
| `06-ospf-neighbor-states.mmd` | OSPF neighbor states | `flowchart LR` |
| `07-bgp-session-troubleshooting.mmd` | BGP session troubleshooting, as a decision tree | `flowchart TD` |
| `08-bgp-route-processing-pipeline.mmd` | Adj-RIB-In to Loc-RIB to Adj-RIB-Out | `flowchart LR` |
| `09-ebgp-ibgp-propagation-rules.mmd` | eBGP vs iBGP propagation rules | `flowchart TD` |
| `10-route-reflector-rules.mmd` | Route reflector reflection rules | `flowchart TD` |
| `11-leaf-spine-ebgp-unnumbered.mmd` | Leaf-spine topology with eBGP unnumbered | `flowchart TB` |
| `12-ecmp-multipath-decision.mmd` | BGP ECMP (multipath) decision | `flowchart TD` |
| `13-inbound-filter-prefix-list-route-map.mmd` | Inbound filtering with prefix-list and route-map | `flowchart TD` |
| `14-rtbh-blackhole-community.mmd` | RTBH through a blackhole community | `flowchart TD` |
| `15-bgp-graceful-restart.mmd` | BGP Graceful Restart, restarter and helper | `sequenceDiagram` |
| `16-bgp-evpn-route-types.mmd` | BGP EVPN (VXLAN) route types | `flowchart TD` |
| `17-link-failure-convergence.mmd` | BGP convergence after a link failure | `flowchart TD` |

`05` and `06` draw state machines as flowcharts, while the `mermaid` skill routes a state machine
to `stateDiagram-v2`: take those two as examples of the state vocabulary, not of the type choice.

## Adding an Example

Create `NN-<slug>.mmd` with the `%%` title line and add a row to the table above. No code reads
this folder: it reaches the illustrator only because `../AGENT.md` names it, so an example that
does not fit the shape described here is an example nothing can use.
