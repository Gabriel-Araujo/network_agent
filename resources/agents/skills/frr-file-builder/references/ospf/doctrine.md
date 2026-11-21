# OSPF Doctrine

Default posture is passive: OSPF speaks only on the interfaces the request names as transit,
and every other OSPF interface is advertised as a stub link. Each rule below is met, not
applicable for a stated reason, or a regression. The Source column names the `.rst` section
that carries the command or the behaviour the rule rests on.

## Adjacency Policy

What every OSPF configuration carries, whatever else the request asks for.

| Rule | Source |
|------|--------|
| `ospf router-id` with the value the request gives | ospfd › Routers |
| `passive-interface default` in `router ospf`, and `no ip ospf passive` only on the interfaces the request names as transit | ospfd › Routers; ospfd › Interfaces |
| A connected prefix enters OSPF through an interface in OSPF, passive or transit, never through `redistribute connected`: a passive interface is a stub link in the router-LSA, a redistributed prefix is a Type-5 LSA with global flooding scope | ospfd › Interfaces; ospfd › Redistribution; ospf_fundamentals › Link-State LSAs |
| Every transit interface carries `ip ospf authentication key-chain <name>` with the key chain the request names. When the request names none, write no authentication line and put a warning in the interface block: `! WARNING: no authentication on the adjacencies of <interface>` | ospfd › Interfaces |
| Never `ip ospf authentication-key` nor `area <area> authentication`: simple password authentication is insecure and deprecated | ospfd › Interfaces; ospfd › Areas |
| Never `ip ospf message-digest-key`: its argument is the key itself, and a secret is never written | ospfd › Interfaces |

The rest of the tables below apply when the request asks for the feature or gives what it
needs; absent, they add neither a line nor a warning.

## Security

| Rule | Note | Source |
|------|------|--------|
| HMAC-SHA through a key chain on every transit interface | The host clock must never go backwards, even across reboots (NTP or a battery-backed clock), or adjacencies do not come back promptly after a restart | ospfd › Interfaces |
| A key chain this file defines carries `cryptographic-algorithm hmac-sha-256` and no key string | The key string warning of the skill body takes its place | ospfd › Interfaces |
| Every `redistribute` line carries a route-map permitting exactly what the request allows | | ospfd › Redistribution |
| `default-information originate always` only with a stated reason | It advertises a default even when the routing table has none | ospfd › Redistribution |
| `ip ospf neighbor-filter` on a shared segment where only some of the reachable routers belong | Also mitigates OSPF denial of service. The prefix-list permits the source address of each router that belongs | ospfd › Interfaces |
| `ospf rfc1583compatibility` only with a reason the request states | It should not be set normally | ospfd › Routers |
| `compatible rfc7474` stays at its default; `no compatible rfc7474` only for a named peer implementation that reuses sequence numbers, with a `SECURITY NOTE` naming it | Packets with equal sequence numbers are then accepted, not only strictly increasing ones | ospfd › Routers |

## Control Plane Minimization

The `daemons` file enables `zebra` and `ospfd`, and any other daemon only per the table in
`TECH.md`. OSPF speaks on no interface the request did not name as transit.
[ospfd › Configuring OSPF; ospfd › Multi-instance Support; ospfd › Routers; ospfd › Interfaces]

## Security Regression

Removing authentication from a transit interface, making a passive interface transit, or adding
a `redistribute` line without a route-map is built only after a `SECURITY WARNING` and explicit
user confirmation. Without the confirmation, nothing is built. [ospfd › Interfaces; ospfd › Redistribution]

## Performance

| Rule | Note | Source |
|------|------|--------|
| BFD on every transit interface where the topology allows, with `ip ospf bfd` | Detection faster than the dead interval | bfd › OSPF BFD Configuration; ospfd › Interfaces |
| Point-to-point type on links with exactly two routers | No DR is elected and no network-LSA is originated for the link | ospfd › Interfaces; ospf_fundamentals › Link-State LSAs |
| On a shared segment, `ip ospf priority` only with the DR role the request gives | The highest priority is the most eligible DR; `0` is never DR | ospfd › Interfaces |
| Hello and dead intervals changed only with numeric justification | Equal on every router of the link | ospfd › Interfaces |
| SPF throttle changed only with numeric justification | State the trade-off, not just the value | ospfd › Routers |
| `auto-cost reference-bandwidth` only with the value the whole domain uses | It must be consistent across the domain | ospfd › Routers |
| Summarize at the ABR with `area <area> range` | Only intra-area routes summarize; Type-5 cannot | ospfd › Areas |
| An area where no router originates external routes is a stub, or totally stubby with `no-summary` when it needs no inter-area summaries either | All its external routes are via the ABR, which passes it summaries and a default instead | ospfd › Areas |
| A stub-like area that must import external routes of its own is an NSSA, and its ABR summarizes them with `area <area> nssa range` | The ABR translates Type-7 into Type-5 for the backbone | ospfd › Areas |
| `max-metric router-lsa on-startup` on routers that carry transit traffic | Lets OSPF converge before traffic goes through the router | ospfd › Routers |
| `graceful-restart` on a router whose *ospfd* restarts while it keeps forwarding, and `graceful-restart helper enable` on the routers that help a neighbor restart | Helper support is disabled by default. *ospfd* must be back within the grace period, or the restart fails | ospfd › Graceful Restart; ospfd › Routers |
| Every timer, threshold, or limit carries a `!` comment stating its expected effect | | ospfd › Interfaces; ospfd › Routers |
