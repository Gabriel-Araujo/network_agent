# OSPF

OSPFv2 for IPv4: interfaces in areas, passive and transit interfaces, authentication, area
types and ABR summarization, redistribution into OSPF, graceful restart, OSPF in a VRF, BFD
bound to an OSPF interface.

## Scope

Out of this pack: OSPFv3 (pack `ospf6`), multi-instance `router ospf (1-65535)`, NBMA and
point-to-multipoint `neighbor` statements, virtual links, Opaque LSA applications, Traffic
Engineering, Router Information, Segment Routing, TI-LFA.

Read with: `bfd`, when an interface carries `ip ospf bfd`.

## Daemons

| Daemon | When |
|--------|------|
| `zebra` | Always: *ospfd* gets its interface information from it |
| `ospfd` | Always |
| `bfdd` | Only if an interface carries `ip ospf bfd` |
| `staticd` | Only if static routes are redistributed into OSPF |

A protocol redistributed into OSPF runs the daemon its own pack names. Anything beyond this
table needs an explicit justification.

## Vocabulary

Use these terms with exactly these meanings, in config comments and in what is said about the
configuration.

| Term | Means | Not |
|------|-------|-----|
| adjacency | Two OSPF routers on one link that exchange their LSDBs | session, peering |
| transit interface | An interface where OSPF speaks and forms adjacencies | active link, uplink |
| passive interface | An interface whose prefix is advertised as a stub link, with no OSPF spoken on it | silent, edge, stub interface |
| shared segment | A link with more than two OSPF routers, where a DR is elected | LAN link, broadcast link |
| area | The OSPF area an interface belongs to, written dotted (`0.0.0.0`) | zone, region |
| stub area | An area no external route enters; its ABR sends a default instead | edge area |
| totally stubby area | A stub area that receives no inter-area summaries either, only the default | stub (alone) |
| NSSA | An area that imports external routes as Type-7, which its ABR translates to Type-5 | stub area |
| ABR | A router with interfaces in the backbone and in another area | border router |
| ASBR | A router that redistributes routes external to OSPF | edge router |
| redistribution policy | The route-map on one `redistribute` line | filter, import rule |

## Required Facts

An identity fact is never guessed; an operational preference may take a default. A fact the
request, or a file already read, gives is not asked.

### Always Ask

Each of these blocks the build while missing.

| Missing | Question | Why |
|---------|----------|-----|
| Router-id | What is the router-id of this router? | It must be unique in the whole OSPF domain; left out, *ospfd* takes one from zebra and it can change |
| Interfaces and areas | Which interfaces run OSPF, and in which area is each one? | Without them no interface joins OSPF |
| Transit interfaces | On which of those interfaces must this router form adjacencies? | Every other OSPF interface is passive |

### Ask When the Request Implies It

| Missing | Question |
|---------|----------|
| Link type | Does `<interface>` connect exactly two routers, or is it a shared segment? |
| Area type | Is area `<area>` a normal area, a stub, a totally stubby area, or an NSSA? |
| Summarization | Which ranges does this ABR summarize out of area `<area>`? |
| Redistribution | Which routes are redistributed into OSPF, and which of them are allowed? |
| Default route | Does this router originate a default into OSPF, and only while it has one itself? |
| Neighbors on a shared segment | Which routers on `<interface>` may form adjacencies with this one? |
| DR role | On `<interface>`, should this router be the DR, a candidate, or never the DR? |
| Reference bandwidth | What reference bandwidth does the rest of the OSPF domain use? |
| VRF | Does OSPF run in a VRF? Which one, and which interfaces are in it? |

### Warn, Never Ask

Missing, these do not block: the configuration carries the warning that `doctrine.md` gives for
each, and no invented value.

- Authentication: a key chain name for each transit interface

### Never Ask

- FRR version: `frr version` is never written

### May Be Filled In

Nothing here carries security impact or invents an identity.

- key chain, prefix-list and route-map names
- timer values, where doctrine does not fix one

### Question Form

One question per item, direct, with the reason attached.

```
Which area does `eth2` belong to?
why: `ip ospf area` cannot be written without it, and two routers on the same link in
different areas never form an adjacency.
```

Never ask in bulk ("send me the network details").

## Examples

Canonical shapes of `frr.conf`, one file each under `references/ospf/examples/`. Read the one
closest to the request; copy its structure, never its values.

| File | Shape |
|------|-------|
| `references/ospf/examples/ospf-p2p-core.md` | Core router: two point-to-point transit links, a loopback, authentication and BFD |
| `references/ospf/examples/ospf-abr-stub-area.md` | ABR between the backbone and a totally stubby area, with a range |
| `references/ospf/examples/ospf-nssa-abr.md` | ABR of an NSSA: Type-7 default into the area, one translated summary out |
| `references/ospf/examples/ospf-asbr-redistribution.md` | ASBR: a conditional default and routes from another protocol through a route-map |
| `references/ospf/examples/ospf-shared-segment.md` | Router on a shared segment: DR priority and a neighbor-filter |
| `references/ospf/examples/ospf-graceful-restart.md` | Maintenance: graceful restart and helper mode |
| `references/ospf/examples/ospf-vrf.md` | OSPF in a VRF toward a customer router |

## Verification Commands

What an operator runs to verify the configuration.

```
vtysh -c "show ip ospf"
vtysh -c "show ip ospf interface"
vtysh -c "show ip ospf neighbor"
vtysh -c "show ip ospf database"
vtysh -c "show ip ospf route"
vtysh -c "show ip ospf border-routers"
vtysh -c "show ip ospf graceful-restart helper"
vtysh -c "show ip ospf vrf <vrf> neighbor <interface>"
vtysh -c "show bfd peers"
```
