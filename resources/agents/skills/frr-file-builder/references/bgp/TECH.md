# BGP

eBGP and iBGP, route reflectors, policy, communities, aggregation,
multihoming, BFD bound to a BGP session, MP-BGP IPv4/IPv6 unicast.

## Scope

Out of this pack: EVPN/VXLAN (pack `evpn`, which reads this one), VPNv4/VPNv6, MPLS label
distribution by LDP (pack `ldp`), Segment Routing, flowspec, multicast.

Read with: `bfd`, when a session carries `neighbor <peer> bfd`.

## Daemons

| Daemon | When |
|--------|------|
| `zebra` | Always |
| `bgpd` | Always |
| `bfdd` | Only if the configuration uses BFD |
| `staticd` | Only if the configuration uses a static route, such as an aggregate anchor |

Anything beyond this table needs an explicit justification.

## Vocabulary

Use these terms with exactly these meanings, in config comments and in what
is said about the configuration.

| Term | Means | Not |
|------|-------|-----|
| session | The BGP peer pair | connection, link |
| peer | The router on the other side, FRR or not | neighbor device, remote |
| policy | The route-map plus prefix-list set on one direction of one session | filter, rule |

Always qualify a policy by direction: inbound policy, outbound policy.

## Required Facts

An identity fact is never guessed; an operational preference may take a
default. A fact the request, or a file already read, gives is not asked.

### Always Ask

Each of these blocks the build while missing.

| Missing | Question | Why |
|---------|----------|-----|
| Local AS | What is the AS number of the router receiving this configuration? | Without it there is no `router bgp` |
| Peer AS | What is the AS number of each peer? | Decides eBGP vs iBGP and the policy that applies |
| Peer address | What is the address of each session? | Without it there is no `neighbor` |
| Prefix to advertise | Which prefixes does this AS advertise to each eBGP peer? | The outbound policy permits exactly them, and advertising space you do not own is an incident. An iBGP-only request that advertises nothing needs none |
| Router-id | What is the router-id of this router? | Without it `bgp router-id` would carry an invented value |

### Ask When the Request Implies It

| Missing | Question |
|---------|----------|
| iBGP role | Is this router a route reflector, a client, or full mesh? |
| Multihoming | Is there more than one upstream? Which is primary? |
| VRF | Does the session live in a VRF? Which one? |
| Interface | Is the session over a physical interface, a loopback, or unnumbered? |

### Warn, Never Ask

Missing, these do not block: the configuration carries the warning that
`doctrine.md` gives for each, and no invented value.

- Authentication: a password or keychain name
- `maximum-prefix`: the limit for each eBGP session

### Never Ask

- FRR version: `frr version` is never written

### May Be Filled In

Nothing here carries security impact or invents an identity.

- prefix-list, route-map, and community-list names
- `description` text
- timer values, where doctrine does not fix one

### Question Form

One question per item, direct, with the reason attached.

```
What is the AS number of `peer-upstream`?
why: decides whether the outbound policy is transit or peering, and
`remote-as` cannot be written without it.
```

Never ask in bulk ("send me the network details").

## Examples

Canonical shapes of `frr.conf`, one file each under `references/bgp/examples/`. Read the one
closest to the request; copy its structure, never its values.

| File | Shape |
|------|-------|
| `references/bgp/examples/ebgp-peer.md` | eBGP session with policy both ways |
| `references/bgp/examples/ebgp-aggregate-anchor.md` | eBGP with aggregate origination and a static anchor |
| `references/bgp/examples/ebgp-with-bfd.md` | eBGP with BFD |
| `references/bgp/examples/session-hardening.md` | Session hardening |
| `references/bgp/examples/route-policy-objects.md` | Route policy: prefix-list, AS-path and community |
| `references/bgp/examples/ibgp-route-reflector.md` | iBGP route reflector |
| `references/bgp/examples/ibgp-over-ospf.md` | iBGP over an OSPF underlay |
| `references/bgp/examples/bgp-confederation.md` | BGP confederation |
| `references/bgp/examples/mp-bgp-dual-stack.md` | MP-BGP dual-stack over one session |
| `references/bgp/examples/bgp-unnumbered-ecmp.md` | BGP unnumbered with ECMP on a Linux host |
| `references/bgp/examples/multihoming-traffic-engineering.md` | Multihoming with traffic engineering |
| `references/bgp/examples/conditional-advertisement.md` | Conditional advertisement for backup transit |
| `references/bgp/examples/graceful-restart-and-shutdown.md` | Maintenance with graceful restart and graceful shutdown |
| `references/bgp/examples/rpki-origin-validation.md` | RPKI origin validation |
| `references/bgp/examples/rtbh.md` | Remote triggered black hole |
| `references/bgp/examples/bgp-with-pbr.md` | BGP with PBR for selective egress |
| `references/bgp/examples/bgp-wan-vrrp-lan.md` | BGP on the WAN with VRRP on the LAN |
| `references/bgp/examples/bgp-over-gre-ipsec.md` | BGP over a GRE tunnel with IPsec |

## Verification Commands

What an operator runs to verify the configuration.

```
vtysh -c "show bgp summary"
vtysh -c "show bgp ipv4 unicast"
vtysh -c "show bgp neighbors <peer> advertised-routes"
vtysh -c "show bfd peers"
```
