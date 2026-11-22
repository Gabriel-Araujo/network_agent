# IS-IS

IS-IS for IPv4 and IPv6: levels and circuit types, passive and transit interfaces,
point-to-point and broadcast circuits, the attached and overload bits, default origination,
local LFA, IS-IS in a VRF, BFD bound to an IS-IS interface.

## Scope

Out of this pack: OpenFabric (pack `openfabric`), multi-topology (`topology`), redistribution
of a routing table (`redistribute <ipv4|ipv6> table`), Remote LFA and TI-LFA, Traffic
Engineering, Segment Routing, SRv6, Flex-Algo and its `affinity-map`, prefix priorities.

Read with: `bfd`, when an interface carries `isis bfd`.

## Daemons

| Daemon | When |
|--------|------|
| `zebra` | Always: *isisd* gets its interface information from it |
| `isisd` | Always |
| `bfdd` | Only if an interface carries `isis bfd` |

Anything beyond this table needs an explicit justification.

## Vocabulary

Use these terms with exactly these meanings, in config comments and in what is said about the
configuration.

| Term | Means | Not |
|------|-------|-----|
| adjacency | Two IS-IS routers on one link that exchange hellos at a level | session, peering |
| transit interface | An interface where IS-IS forms adjacencies | active link, uplink |
| passive interface | An IS-IS interface in passive mode, `isis passive`, whose prefix IS-IS advertises | silent, edge interface |
| shared segment | A broadcast circuit with more than two IS-IS routers, which holds a designated router election | LAN link |
| NET | The network entity title `net` sets, in ISO format: area address, system ID and the `00` selector | router-id |
| level-1 | Routing inside one area: a level-1 router is a station router | area 0, intra-area level |
| level-2 | Routing between areas: a level-2 router is an area router | backbone area |
| level-1-2 router | A router that is both a station router and an area router | ABR |
| circuit type | The level at which one interface of a level-1-2 router forms adjacencies | interface level |
| attached bit | The bit a level-1-2 router sets in the LSPs it sends to level-1 routers, which then create a default route toward it | default flag |
| overload bit | The bit that keeps transit traffic off the router that sets it | maintenance bit |

## Required Facts

An identity fact is never guessed; an operational preference may take a default. A fact the
request, or a file already read, gives is not asked.

### Always Ask

Each of these blocks the build while missing.

| Missing | Question | Why |
|---------|----------|-----|
| NET | What is the NET of this router: its area address and system ID? | The NET is the router's identity in IS-IS; an invented one can collide with another router's |
| Level | Does this router run level-1, level-2-only, or level-1-2? | `is-type` decides which adjacencies it forms |
| Interfaces | Which interfaces run IS-IS? | Without them no interface joins IS-IS |
| Transit interfaces | On which of those interfaces must this router form adjacencies? | Every other IS-IS interface is passive |

### Ask When the Request Implies It

| Missing | Question |
|---------|----------|
| Link type | Does `<interface>` connect exactly two routers, or is it a shared segment? |
| Circuit type | On this level-1-2 router, do the neighbors on `<interface>` run level-1, level-2, or both? |
| Address families | Does IS-IS carry IPv6 as well as IPv4, and on which interfaces? |
| Metric | Which metric does `<interface>` carry, and at which level? |
| Designated router priority | Which `isis priority` does `<interface>` carry for the designated router election, and at which level? |
| Default route | Does this router originate a default into level `<level>`, and only while it has one itself? |
| Fast reroute | Which transit interfaces does local LFA protect? |
| VRF | Does IS-IS run in a VRF? Which one, and which interfaces are in it? |

### Warn, Never Ask

Missing, these do not block: the configuration carries the warning that `doctrine.md` gives for
each, and no invented value.

- Authentication: `isis password` on each transit interface, `area-password` at level-1 and
  `domain-password` at level-2. The argument of each is the password itself, so none is ever
  written and none is asked

### Never Ask

- FRR version: `frr version` is never written
- A password: the line is left out and the warning names it

### May Be Filled In

Nothing here carries security impact or invents an identity.

- the `router isis` name, unless a file already read names one
- timer values, where doctrine does not fix one

### Question Form

One question per item, direct, with the reason attached.

```
Does `eth2` connect exactly two routers, or is it a shared segment?
why: a two-router link gets `isis network point-to-point`; a shared segment keeps the
broadcast default and elects a designated router.
```

Never ask in bulk ("send me the network details").

## Examples

Canonical shapes of `frr.conf`, one file each under `references/isis/examples/`. Read the one
closest to the request; copy its structure, never its values.

| File | Shape |
|------|-------|
| `references/isis/examples/isis-l2-core-p2p.md` | Level-2 core router: two point-to-point transit links, a passive loopback, BFD, local LFA, the overload bit on startup |
| `references/isis/examples/isis-l1l2-area-border.md` | Level-1-2 router between its area and level-2: a circuit type per interface and the attached bit |
| `references/isis/examples/isis-l1-area-router.md` | Level-1 router inside an area: a passive LAN and a default route from the attached bit |
| `references/isis/examples/isis-shared-segment.md` | Router on a shared segment: broadcast circuit and the designated router priority the request gives |
| `references/isis/examples/isis-dual-stack.md` | IS-IS for IPv4 and IPv6 on the same interfaces, with BFD |
| `references/isis/examples/isis-default-origination.md` | Edge router: a default into level-2 only while its routing table has one |
| `references/isis/examples/isis-maintenance-drain.md` | Maintenance: high metrics shift traffic off the router |
| `references/isis/examples/isis-vrf.md` | IS-IS in a VRF toward a customer router |

## Verification Commands

What an operator runs to verify the configuration.

```
vtysh -c "show isis summary"
vtysh -c "show isis interface detail"
vtysh -c "show isis neighbor"
vtysh -c "show isis database"
vtysh -c "show isis route"
vtysh -c "show isis topology"
vtysh -c "show isis fast-reroute summary"
vtysh -c "show isis vrf <vrf> neighbor"
vtysh -c "show bfd peers"
```
