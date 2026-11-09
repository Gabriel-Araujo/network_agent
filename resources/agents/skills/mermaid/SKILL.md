---
name: mermaid
description: "MermaidJS syntax for diagrams written as fenced mermaid blocks in the reply: FRR topologies and configuration structure (flowchart), protocol message exchanges (sequence), protocol state machines (state). Use before writing any mermaid block."
---

# Mermaid diagrams

A diagram is text: a fenced code block tagged `mermaid` inside the reply. Whoever reads the
reply renders it (MermaidJS 11). Nothing here writes, renders or validates a file.

## Fence convention

- One diagram per fence. The first line inside the fence names the diagram type.
- The fence goes in the reply text — never write a diagram to a file.

## Pick the diagram type

| To show | Type | First line | Reference |
|---------|------|------------|-----------|
| Topology: routers, links, autonomous systems, areas, sites | flowchart | `flowchart LR` | `references/flowchart.md` |
| Configuration structure: daemons, interfaces, protocol blocks, policy chains | flowchart | `flowchart TB` | `references/flowchart.md` |
| Messages over time: BGP OPEN/KEEPALIVE/UPDATE, OSPF Hello/DBD/LSR/LSU | sequence | `sequenceDiagram` | `references/sequence.md` |
| A protocol state machine: BGP FSM, OSPF neighbor states | state | `stateDiagram-v2` | `references/state.md` |

Other diagram types (class, gantt, pie, ...) are not covered: draw with these three.

## Rules that break rendering when ignored

- **IDs are plain words:** letters, digits and underscore only (`R1`, `AS65001`, `spine_1`).
  No spaces, hyphens, dots or colons in an ID — the human text goes in a label.
- **Never write `end` in lowercase** as an ID or as a bare word in a label: it closes a block.
  Write `End`/`END`, or put it in quotes or brackets.
- **Flowchart text with spaces or punctuation goes in double quotes — node labels, link labels
  and subgraph titles:** `R2["R2 (AS 65001)"]`, `R1 ---|"eth0 10.0.0.1/30"| R2`,
  `subgraph AS65001 ["AS 65001 (R1)"]`. Parentheses outside quotes break the diagram:
  `subgraph AS 65001 (R1)` does not render. A state with such text is declared
  `state "2-Way" as TwoWay`. Sequence participant and message text takes no quotes.
- **Escape with entity codes:** a literal `"` is `#quot;`, `#` is `#35;`, `;` is `#59;`. In
  sequence and state text (after `:`), never write a raw `;` or `#`: depending on the type it
  breaks the diagram, cuts the text short or spawns stray states.
- **Close every block:** `subgraph`, `loop`, `alt`, `opt`, `par`, `critical` close with `end`;
  a composite state closes with `}`; a multi-line note closes with `end note`.
- **Comments** start with `%%` on their own line.
- **One idea per diagram.** A large network reads better as two or three small diagrams than
  as one crowded one.

## References

Read the one for the chosen type with `read_skill_file` (`skill_name`: `mermaid`):

- `references/flowchart.md` — directions, node shapes, link styles and labels, subgraphs; a
  topology example and a configuration-structure example.
- `references/sequence.md` — participants, arrows, notes, `loop`/`alt`/`opt`/`par`; a BGP
  session establishment example.
- `references/state.md` — states with descriptions, transitions with events, composite
  states, choices, notes; BGP FSM and OSPF neighbor state examples.
