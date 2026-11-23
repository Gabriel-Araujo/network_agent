# OSPFv3

OSPFv3 for IPv6: interfaces in areas, passive and transit interfaces, the authentication
trailer, NSSA and ABR summarization, redistribution and ASBR summarization, graceful restart,
stub router for maintenance, OSPFv3 in a VRF, BFD bound to an OSPFv3 interface.

## Scope

Out of this pack: OSPFv2 (pack `ospf`); stub and totally stubby areas, for which `ospf6d.rst`
documents no command; point-to-multipoint mode; the `ipv6 ospf6 p2p-p2mp` options and
`ipv6 ospf6 neighbor`, which the corpus calls advanced configuration; NBMA, which FRR does not
support for IPv6; inter-area filtering (`area <area> export-list`, `import-list`,
`filter-list`); `ipv6 ospf6 instance-id`, which only appears in sample blocks; IPsec.

Read with: `bfd`, when an interface carries `ipv6 ospf6 bfd`.

## Daemons

| Daemon | When |
|--------|------|
| `zebra` | Always: *ospf6d* exchanges its routes and redistribution with it |
| `ospf6d` | Always |
| `bfdd` | Only if an interface carries `ipv6 ospf6 bfd` |
| `staticd` | Only if static routes are redistributed into OSPFv3 |

A protocol redistributed into OSPFv3 runs the daemon its own pack names. Anything beyond this
table needs an explicit justification.

## Vocabulary

Use these terms with exactly these meanings, in config comments and in what is said about the
configuration.

| Term | Means | Not |
|------|-------|-----|
| adjacency | Two OSPFv3 routers on one link that exchange their LSDBs | session, peering |
| transit interface | An OSPFv3 interface without `ipv6 ospf6 passive`, where this router forms adjacencies | active link, uplink |
| passive interface | An OSPFv3 interface in its area with `ipv6 ospf6 passive`: it sends no hellos and forms no adjacency, and its prefixes are still advertised | silent, edge, stub interface |
| shared segment | A link with more than two OSPFv3 routers, left in broadcast mode, where a DR and a BDR are elected | LAN link |
| link-local address | The `fe80::/10` address of an interface; OSPFv3 does all its signaling over it | global address, ULA |
| area | The OSPFv3 area an interface belongs to, written dotted (`0.0.0.0`) | zone, region |
| NSSA | An area that imports external routes as Type-7 LSAs, which its ABR translates to Type-5 for the backbone | stub area |
| ABR | A router with interfaces in the backbone and in another area | border router |
| ASBR | A router that redistributes routes external to OSPFv3, and so originates its external LSAs | edge router |
| stub router | A router that `stub-router administrative` holds as a stub router until the line is removed: no route transits it | overloaded router |
| authentication trailer | The per-packet OSPFv3 authentication of RFC 7166, keyed here by a key chain | IPsec |
| redistribution policy | The route-map on one `redistribute` line | filter, import rule |

## Required Facts

An identity fact is never guessed; an operational preference may take a default. A fact the
request, or a file already read, gives is not asked.

### Always Ask

Each of these blocks the build while missing.

| Missing | Question | Why |
|---------|----------|-----|
| Router-id | What is the router-id of this router? | It identifies this router in the OSPFv3 domain, written as `A.B.C.D` even on IPv6; a change takes effect after `clear ipv6 ospf6 process` or a restart of *ospf6d* |
| Interfaces and areas | Which interfaces run OSPFv3, and in which area is each one? | Without them no interface joins OSPFv3 |
| Transit interfaces | On which of those interfaces must this router form adjacencies? | Every other OSPFv3 interface is passive |

### Ask When the Request Implies It

| Missing | Question |
|---------|----------|
| Link type | Does `<interface>` connect exactly two routers, or is it a shared segment? |
| Area type | Is area `<area>` a normal area or an NSSA, and does an NSSA get inter-area summaries or only a default? |
| Summarization | Which IPv6 ranges does this ABR summarize out of area `<area>`? |
| Redistribution | Which routes are redistributed into OSPFv3, and which of them are allowed? |
| External summarization | Which IPv6 ranges does this ASBR advertise as one external LSA each? |
| Default route | Does this router originate a default into OSPFv3, and only while it has one itself? |
| DR priority | On `<interface>`, which router priority does this router take in the DR election? |
| Reference bandwidth | What reference bandwidth does the rest of the OSPFv3 domain use? |
| VRF | Does OSPFv3 run in a VRF? Which one, and which interfaces are in it? |

### Warn, Never Ask

Missing, these do not block: the configuration carries the warning that `doctrine.md` gives for
each, and no invented value.

- Authentication, when the request does not ask for it on a transit interface

### Never Ask

- FRR version: `frr version` is never written
- An IPv6 address for an OSPFv3 interface: OSPFv3 signals over link-local addresses, so an
  address is written only when the request gives one

### May Be Filled In

Nothing here carries security impact or invents an identity. A value with operational effect
(a timer, a metric, a cost, a priority) is never filled in: without one in the request, its line
is not written and the FRR default holds.

- key chain name, only when the request asks for authentication without naming one
- prefix-list, route-map and `bfd` profile names

### Question Form

One question per item, direct, with the reason attached.

```
Which area does `eth2` belong to?
why: `ipv6 ospf6 area` cannot be written without it, and the interface does not join OSPFv3
until it has one.
```

Never ask in bulk ("send me the network details").

## Examples

Canonical shapes of `frr.conf`, one file each under `references/ospf6/examples/`. Read the one
closest to the request; copy its structure, never its values.

| File | Shape |
|------|-------|
| `references/ospf6/examples/ospf6-p2p-core.md` | Core router: two point-to-point transit links, a passive loopback, the authentication trailer and BFD |
| `references/ospf6/examples/ospf6-abr-range.md` | ABR between the backbone and a branch area, with one range for the branch |
| `references/ospf6/examples/ospf6-nssa-abr.md` | ABR of an NSSA: a Type-7 default into the area, one translated summary out |
| `references/ospf6/examples/ospf6-asbr-redistribution.md` | ASBR: routes from another protocol through a route-map, one summary-address for them, a conditional default |
| `references/ospf6/examples/ospf6-shared-segment.md` | Router on a shared segment: broadcast mode and the router priority the request gives |
| `references/ospf6/examples/ospf6-graceful-restart.md` | Maintenance: graceful restart and helper mode |
| `references/ospf6/examples/ospf6-stub-router.md` | Maintenance: a router held as a stub router while it is drained |
| `references/ospf6/examples/ospf6-vrf.md` | OSPFv3 in a VRF toward a customer router |

## Verification Commands

What an operator runs to verify the configuration.

```
vtysh -c "show ipv6 ospf6"
vtysh -c "show ipv6 ospf6 interface"
vtysh -c "show ipv6 ospf6 neighbor"
vtysh -c "show ipv6 ospf6 database"
vtysh -c "show ipv6 ospf6 route"
vtysh -c "show ipv6 ospf6 redistribute"
vtysh -c "show ipv6 ospf6 summary-address detail"
vtysh -c "show ipv6 ospf6 graceful-restart helper"
vtysh -c "show ipv6 ospf6 vrf <vrf> neighbor"
vtysh -c "show bfd peers"
```
