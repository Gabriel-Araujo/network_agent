# BGP Artifact Shape

Governs shape, not catalog. The exact syntax of each command comes from the
version of record via `evidence.json`. Where the two diverge, the evidence
wins and you emit a `VERSION NOTE`.

## frr.conf Structure

One integrated file, in this order.

1. Header: `frr version`, `frr defaults`, `hostname`, `log`.
2. Interfaces and static routes, if any.
3. Policy objects (prefix-lists, community-lists, route-maps) before the
   block that references them. A dangling reference fails stage 05.
4. `router bgp <ASN>`: `bgp router-id`, global options,
   `neighbor ... remote-as`, then each `address-family`.
5. A `!` separator between blocks.

## Rules

| Rule | Reason |
|------|--------|
| `remote-as` at the `router bgp` level, `activate` and policy and `maximum-prefix` inside the address family | Placing policy at the global level silently does nothing |
| Every neighbor gets an explicit `activate` in the address family it uses | Without it the session comes up and carries no routes |
| Every address family closes with `exit-address-family` | |
| Peer-group at three or more peers sharing one policy, direct config below that | |
| Every neighbor gets a `description` naming the peer and why the session exists | |
| Never a per-daemon file, never `bgpd.conf` | |

## Version Gated Features

Confirm against the evidence before emitting. If you cannot confirm, emit the
stable alternative with a `VERSION NOTE`.

| Feature | Stable alternative |
|---------|-------------------|
| TCP-AO | MD5 with a `SECURITY NOTE` |
| BGP roles and OTC (RFC 9234) | Explicit route-map policy |
| RPKI syntax and validator daemon name | Prefix-list filtering only |
| `bgp long-lived-graceful-restart` | Plain graceful restart |
| Large community field naming | Standard communities |
