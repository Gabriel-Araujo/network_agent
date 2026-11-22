# IS-IS Doctrine

Default posture is passive: every IS-IS interface the request does not name as transit is in
passive mode. Each rule below is met, not applicable for a stated reason, or a regression. The
Source column names the `.rst` section that carries the command or the behaviour the rule
rests on.

## Adjacency Policy

What every IS-IS configuration carries, whatever else the request asks for.

| Rule | Source |
|------|--------|
| `net` with the NET the request gives, and `is-type` with the level the request gives | isisd › ISIS router; isisd › ISIS region |
| `isis passive` on every IS-IS interface the request does not name as transit, the loopback included | isisd › ISIS interface |
| `isis password` is never written: its argument is the password itself. Every transit interface carries in its place `! WARNING: isis password md5 is left out; add it on the router, never in this file` | isisd › ISIS interface |
| `area-password` and `domain-password` are never written: their argument is the password itself. `router isis` carries in their place `! WARNING: area-password md5 is left out; add it on the router, never in this file` when it runs level-1, the area, and the same warning for `domain-password` when it runs level-2, the domain | isisd › ISIS router; isisd › ISIS Timer |
| The warnings name the `md5` form: the `clear` form configures the password as clear text | isisd › ISIS router; isisd › ISIS interface |

The rest of the tables below apply when the request asks for the feature or gives what it
needs; absent, they add neither a line nor a warning.

## Security

| Rule | Note | Source |
|------|------|--------|
| `default-information originate` without `always`, unless the request states a reason | `always` originates the default even if there is no default route in the system | isisd › ISIS Fast-Reroute |
| A route-map on `default-information originate` is defined in the file | | isisd › ISIS Fast-Reroute |

## Control Plane Minimization

The `daemons` file enables `isisd` the way `ospfd=yes` enables *ospfd*, and *zebra* runs with
it: *zebra* must be running before *isisd* is invoked. Any other daemon runs only per the table
in `TECH.md`. Every IS-IS interface the request did not name as transit is in passive mode.
[isisd › Configuring isisd; ospfd › Multi-instance Support; isisd › ISIS interface]

## Security Regression

Removing `isis password`, `area-password` or `domain-password` from a file that has them, or
making a passive interface transit, is built only after a `SECURITY WARNING` and explicit user
confirmation. Without the confirmation, nothing is built. [isisd › ISIS interface; isisd › ISIS router]

## Performance

| Rule | Note | Source |
|------|------|--------|
| BFD on every transit interface where the topology allows, with `isis bfd` | BFD provides faster failure detection than the IS-IS hello timers | isisd › ISIS interface; bfd › IS-IS BFD Configuration |
| Point-to-point type on links with exactly two routers | Broadcast is the default type | isisd › ISIS interface |
| On a shared segment, `isis priority` only with the value and level the request gives | It sets the priority for the designated router election | isisd › ISIS interface |
| `is-type` follows the role: `level-1` for a station router only, `level-2-only` for an area router only, `level-1-2` for both | | isisd › ISIS region |
| On a level-1-2 router, `isis circuit-type` on each interface whose neighbors run one level | Only the adjacencies of that level are formed on the interface | isisd › ISIS interface |
| `attached-bit send` on a level-1-2 router with level-1 neighbors | It sets the attached bit in the LSPs sent to level-1 routers, which create a default route toward it on receipt | isisd › ISIS router |
| `set-overload-bit on-startup` on routers that carry transit traffic | The overload bit avoids transit traffic for the given number of seconds after startup | isisd › ISIS router |
| A router drained for maintenance gets `advertise-high-metrics`; `set-overload-bit` only when no transit traffic may cross it at all | High metrics gracefully shift traffic off the router; the overload bit avoids any transit traffic | isisd › ISIS router |
| `isis fast-reroute lfa` on the transit interfaces the request asks to protect | Per-prefix local LFA link protection | isisd › ISIS interface |
| `metric-style` stays at its `wide` default; `transition` only while the domain moves between styles | `transition` sends and accepts both styles of TLVs during transition | isisd › ISIS router |
| `isis metric` only with the value the request gives | Its maximum depends on the metric style: 63 narrow, 16777215 wide | isisd › ISIS interface |
| Hello interval and multiplier changed only with numeric justification | The multiplier sets the hello holding time | isisd › ISIS interface |
| LSP and SPF timers changed only with numeric justification | State the trade-off, not just the value | isisd › ISIS Timer |
| Every timer, threshold, or limit carries a `!` comment stating its expected effect | | isisd › ISIS interface; isisd › ISIS Timer; isisd › ISIS router |
