# Flowchart — topologies and configuration structure

## Header and direction

The first line is `flowchart` plus a direction (`graph` is an alias):

- `LR` left to right — topologies, chains of routers.
- `TB` (same as `TD`) top to bottom — hierarchies, configuration structure.
- `RL`, `BT` — the reverse directions.

## Nodes

An ID, optionally followed by a shape holding the label. The ID is what links refer to; the
label is what is drawn. Declare a node's label once, then use the bare ID.

| Shape | Syntax | Use it for |
|-------|--------|-----------|
| Rectangle | `R1["R1"]` | Routers, hosts, files, configuration blocks |
| Rounded | `bgpd("bgpd")` | Daemons, processes |
| Stadium | `lan(["LAN 192.168.10.0/24"])` | Networks, segments, prefixes |
| Circle | `inet(("Internet"))` | External clouds |
| Diamond | `rm10{"seq 10 match?"}` | Decisions in a policy chain |
| Hexagon | `rr{{"Route reflector"}}` | A role worth singling out |

Quote any label with spaces or punctuation: `R2["R2 (AS 65001)"]`. Inside the quotes anything
goes except a literal `"`, which is `#quot;`.

## Links

| Link | Syntax | Use it for |
|------|--------|-----------|
| Line | `R1 --- R2` | A physical link (no direction) |
| Arrow | `R1 --> R2` | A direction: advertisement, policy flow, dependency |
| Dotted | `R1 -.- R2`, `R1 -.-> R2` | A logical session (BGP over the IGP), a backup path |
| Thick | `R1 === R2`, `R1 ==> R2` | The primary or highlighted path |
| Both ways | `R1 <--> R2` | A two-way exchange |

Label a link between pipes, quoted: `R1 ---|"eth0 10.0.0.1/30"| R2`,
`R1 -->|"UPDATE 10.0.0.0/8"| R2`. Chain links on one line (`R1 --> R2 --> R3`) or fan out with
`&` (`R1 & R2 --> RR1`).

**Always put spaces around a link.** A node ID starting with `o` or `x` glued to a link turns
the link into a circle or cross edge: `zebra---ospfd` draws `zebra` linked to `spfd`. Write
`zebra --- ospfd`.

## Subgraphs

Group nodes into autonomous systems, areas, sites or daemons:

```
subgraph AS65001 ["AS 65001"]
    direction TB
    R1["R1"]
    R2["R2"]
end
```

- `subgraph ID ["Title"]` … `end`. Subgraphs nest.
- Links can point at a subgraph ID (`AS65001 --> AS65002`).
- `direction` inside a subgraph is ignored as soon as one of its nodes links to a node outside
  it — the subgraph then follows the diagram's direction.

## Example — topology

```mermaid
flowchart LR
    subgraph AS65001 ["AS 65001 — OSPF area 0"]
        R1["R1<br/>10.255.0.1"]
        R2["R2<br/>10.255.0.2"]
        R1 ---|"eth1 10.0.12.0/30"| R2
        R1 -.-|"iBGP (loopbacks)"| R2
    end
    subgraph AS65002 ["AS 65002"]
        R3["R3<br/>10.255.0.3"]
    end
    R2 ---|"eth0 203.0.113.0/31 · eBGP"| R3
```

## Example — configuration structure

One router's `frr.conf`, from daemons down to the policy applied to a neighbor:

```mermaid
flowchart TB
    conf["frr.conf"]
    conf --> zebra("zebra")
    conf --> ospfd("ospfd")
    conf --> bgpd("bgpd")
    zebra --> ifaces["interface eth0, eth1, lo"]
    ospfd --> ospf["router ospf<br/>network 10.0.12.0/30 area 0"]
    bgpd --> bgp["router bgp 65001"]
    bgp --> nbr["neighbor 203.0.113.1 remote-as 65002"]
    nbr -->|"route-map EXPORT out"| rm["route-map EXPORT"]
    rm --> pl["ip prefix-list OWN seq 5 permit 198.51.100.0/24"]
```

## Never write these

Styling (`classDef`, `style`, `linkStyle`, `:::`), icons, click events, Markdown strings and
the newer `@{ shape: … }` node syntax. Most of it renders — it is banned because colour
carries no meaning a label cannot, and because two of its forms are parse errors that kill the
whole diagram:

- `linkStyle 0 stroke:#FF4500;` — a hex value followed by `;`. Measured: `stroke:red;` parses
  and `stroke:#FF4500` without the `;` parses, the two together do not.
- `class R1--R2 name` — `class` takes nodes, never links.

Carry the meaning in the label and in the link styles this reference already teaches.
