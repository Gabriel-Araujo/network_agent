# BGP

eBGP and iBGP, route reflectors, policy, communities, aggregation,
multihoming, BFD bound to a BGP session, MP-BGP IPv4/IPv6 unicast.

## Scope

Out of this pack, ask for a run of the matching technology: EVPN/VXLAN,
VPNv4/VPNv6, MPLS/SR, flowspec, multicast.

## Daemons

| Daemon | When |
|--------|------|
| `zebra` | Always |
| `bgpd` | Always |
| `bfdd` | Only if the plan uses BFD |
| `staticd` | Only if the plan uses a static route, such as an aggregate anchor |

Anything beyond this table needs an explicit justification in the plan.

## Run Vocabulary

Use these terms with exactly these meanings in the briefing, the plan, and
the config comments.

| Term | Means | Not |
|------|-------|-----|
| session | The BGP peer pair | connection, link |
| peer | The router on the other side, FRR or not | neighbor device, remote |
| policy | The route-map plus prefix-list set on one direction of one session | filter, rule |
| version of record | The FRR version governing syntax for this run | target version |

Always qualify a policy by direction: inbound policy, outbound policy.

## Verification Commands

Named by stage 06 in the operator delivery.

```
vtysh -c "show bgp summary"
vtysh -c "show bgp ipv4 unicast"
vtysh -c "show bgp neighbors <peer> advertised-routes"
vtysh -c "show bfd peers"
```
