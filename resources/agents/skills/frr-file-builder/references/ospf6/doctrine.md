# OSPFv3 Doctrine

Default posture is passive: every OSPFv3 interface the request does not name as transit carries
`ipv6 ospf6 passive`, which suppresses routing updates on it. Each rule below is met, not
applicable for a stated reason, or a regression. The Source column names the `.rst` section
that carries the command or the behaviour the rule rests on. A behaviour the corpus does not
state, proved on a live *ospf6d* of `frr:10.5.0`, says so.

## Adjacency Policy

What every OSPFv3 configuration carries, whatever else the request asks for.

| Rule | Source |
|------|--------|
| `ospf6 router-id` with the value the request gives | ospf6d › Configuration Commands |
| `ipv6 ospf6 passive` on every OSPFv3 interface the request does not name as transit, the loopback included: it suppresses routing updates on the interface. On `frr:10.5.0` a passive interface sends no hellos and forms no adjacency, and its prefixes stay in the router's Intra-Area-Prefix LSA | ospf6d › OSPF6 interface |
| A connected prefix enters OSPFv3 through `ipv6 ospf6 area` on its interface, passive or transit, never through `redistribute connected`, which carries it as an external route in a Type-5 LSA, the kind an ASBR generates. On `frr:10.5.0`, `redistribute connected` re-originates the prefixes of the OSPFv3 interfaces as AS-scoped external LSAs | ospf6d › OSPF6 interface; ospf6d › Redistribute routes to OSPF6; ospf6d › ASBR Summarisation Support in OSPFv3 |
| Every transit interface carries `ipv6 ospf6 authentication keychain <name>` with the key chain the request names, or a filled-in name when the request asks for authentication without naming one: the authentication trailer provides replay protection via sequence number and IPv6 source address protection. When the request does not ask for authentication, write no authentication line and put a warning in the interface block: `! WARNING: no authentication on the adjacencies of <interface>` | ospf6d › Example configuration of keychain:; ospf6d › Authentication trailer support: |
| Never `ipv6 ospf6 authentication key-id`: its `key` argument is the key itself, which the running configuration shows as written, and a secret is never written | ospf6d › Manual key: |

The rest of the tables below apply when the request asks for the feature or gives what it
needs; absent, they add neither a line nor a warning.

## Security

| Rule | Note | Source |
|------|------|--------|
| A key chain this file defines carries `cryptographic-algorithm hmac-sha-256` and no key string | FRR supports MD5 and SHA256 internally and relies on openssl for the other hash algorithms. The key string warning of the skill body takes its place | ospf6d › Authentication trailer support:; ospfd › Interfaces |
| Every `redistribute` line carries a route-map permitting exactly what the request allows | The route-map filters the redistributed routes. On `frr:10.5.0` a route that matches no entry of the route-map is not redistributed | ospf6d › Redistribute routes to OSPF6 |
| `default-information originate always` only with a stated reason | `always` injects the default route regardless of it being present in the router | ospf6d › Redistribute routes to OSPF6 |

## Control Plane Minimization

The `daemons` file enables `ospf6d` the way `ospfd=yes` enables *ospfd*, with *zebra*, which
it exchanges redistributed routes with, and any other daemon only per the table in `TECH.md`.
Every OSPFv3 interface the request did not name as transit is passive.
[ospfd › Multi-instance Support; ospf6d › Showing OSPF6 information; ospf6d › OSPF6 interface]

## Security Regression

Removing authentication from a transit interface, making a passive interface transit, or adding
a `redistribute` line without a route-map is built only after a `SECURITY WARNING` and explicit
user confirmation. Without the confirmation, nothing is built.
[ospf6d › Authentication trailer support:; ospf6d › OSPF6 interface; ospf6d › Redistribute routes to OSPF6]

## Performance

| Rule | Note | Source |
|------|------|--------|
| `ipv6 ospf6 bfd` only when the request asks for BFD or fast failure detection, on the interfaces it asks it for | A BFD peer is created for every new neighbor, to monitor the link status for fast convergence | bfd › OSPF6 BFD Configuration |
| Point-to-point mode on links with exactly two routers, at both ends | No DR/BDR election; routers must be configured for the same mode to form adjacencies | ospf6d › OSPF6 interface; ospf6d › OSPF6 point-to-point and point-to-multipoint operation |
| On a shared segment, `ipv6 ospf6 priority` only with the value the request gives | It sets the router priority of the interface, which the DR and BDR election uses; the default is 1 | ospf6d › OSPF6 interface; ospf_fundamentals › The Hello Protocol |
| Hello and dead intervals only with the values the request gives, and their numeric justification | The hello interval must be the same on all routers of the link; the defaults are 10 and 40 | ospf6d › OSPF6 interface; ospf6d › OSPF6 point-to-point and point-to-multipoint operation |
| On a router with `graceful-restart`, `ipv6 ospf6 graceful-restart hello-delay` follows the hello interval: when the request changes the hello interval, the hello delay takes the same value | The corpus recommends the same values as a best practice; too high a hello delay can time out the adjacencies | ospf6d › OSPF6 interface |
| SPF throttle only with the values the request gives, and their numeric justification | State the trade-off of the delay, the initial holdtime and the maximum holdtime, not just the values | ospf6d › Configuration Commands |
| `auto-cost reference-bandwidth` only with the value the whole domain uses | It must be consistent across all routers within the OSPF domain | ospf6d › Configuration Commands |
| Summarize at the ABR with `area <area> range` | It is used only at the area boundary, and advertises one Inter-Area-Prefix LSA for a group of internal subnets | ospf6d › OSPF6 area |
| Summarize external routes at the ASBR with `summary-address` | One external LSA stands for every prefix in the range, and the summary is installed locally with Null0 as next-hop to avoid leaking traffic | ospf6d › ASBR Summarisation Support in OSPFv3 |
| An area the request makes an NSSA takes the external routes of a router inside it as Type-7 LSAs, and its ABR summarizes them with `area <area> nssa range` | Type-7 LSAs are advertised into the NSSA when an external route is redistributed; the NSSA ABR translates the Type-7 summary to a Type-5 LSA for the backbone | ospf6d › OSPF6 area |
| `area <area> nssa no-summary` only for an NSSA the request wants without inter-area summaries | A single Type-3 LSA with a default route is originated into the NSSA instead | ospf6d › OSPF6 area |
| `graceful-restart` on a router whose *ospf6d* restarts while it keeps forwarding, and `graceful-restart helper enable` on the routers that help a neighbor restart | Helper support is disabled by default. *ospf6d* must be restarted within the grace period, or the graceful restart fails. `shutdown` with graceful restart continues forwarding traffic while shutting down | ospf6d › Graceful Restart; ospf6d › Configuration Commands |
| `stub-router administrative` on a router drained for maintenance, only when the traffic through it has another path, and removed when it returns to service | It holds the router as a stub router for an indefinite period. On `frr:10.5.0` its router-LSA clears the R and V6 bits: the adjacencies stay Full and its own prefixes stay reachable, but no route transits it, even one with no other path | ospf6d › Configuration Commands |
| Every timer, threshold, or limit carries a `!` comment stating its expected effect | | ospf6d › OSPF6 interface; ospf6d › Configuration Commands; ospf6d › Graceful Restart |
