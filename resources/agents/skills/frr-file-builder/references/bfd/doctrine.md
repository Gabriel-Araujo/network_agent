# BFD Doctrine

Default posture is bound: a BFD session runs only for a client the request names, created by
that client's Binding line, and every setting the request gives lives in a named profile. Each
rule below is met, not applicable for a stated reason, or a regression. The Source column names
the `.rst` section that carries the command or the behaviour the rule rests on.

## Profile Policy

What every BFD configuration carries, whatever else the request asks for.

| Rule | Source |
|------|--------|
| A session comes from the Binding line of a client the request names, written by the client's pack: OSPF and IS-IS create one for each neighbor they find on the interface, `neighbor <peer> bfd profile` applies its profile to the sessions it creates, and a monitored static route uses its gateway address as the BFD peer. bfd › Configuration still creates the peers in the `bfd` node first; on `frr:10.5.0` a BGP neighbor and a static route with `bfd` open their sessions with no `peer` | bfd › OSPF BFD Configuration; bfd › IS-IS BFD Configuration; bfd › BGP BFD Configuration; bfd › BFD Static Route Monitoring Configuration; bfd › Configuration |
| A `peer` in the `bfd` block only for a session the request asks the block itself to open | bfd › BFDd Commands |
| Every setting the request gives goes in a named `profile`, and each Binding line that wants it names the profile. A profile can be configured in multiple peers, and an update to it applies to all peers with the profile without interruptions. `ip ospf bfd` carries no multiplier or intervals of its own: they go in the profile | bfd › BFDd Commands; bfd › BFD Peer Specific Commands; bfd › OSPF BFD Configuration; ospfd › Interfaces |
| A `peer` carries a setting of its own only to override its profile for that peer | bfd › BFD Peer Specific Commands |
| A setting is written only with the value the request gives or a rule below fixes. With none, no profile is written and the Binding line names none: the session keeps the defaults, a detect-multiplier of 3 and 300 ms intervals | bfd › Peer / Profile Configuration |
| Every profile a Binding line or a `peer` names is defined in the `bfd` block: a `peer` accepts a profile that does not exist yet, and the profile applies only once it exists | bfd › BFD Peer Specific Commands |

The rest of the tables below apply when the request asks for the feature or gives what it
needs; absent, they add neither a line nor a warning.

## Security

| Rule | Note | Source |
|------|------|--------|
| `multihop` on a `peer` only for a peer the request places more than one hop away | A multihop session expects packets with a TTL under 254 | bfd › BFDd Commands |
| `minimum-ttl` only for a configured multihop `peer` that no Binding line also uses, with the value the request gives | The default, 254, expects only one hop between this system and the peer, and the value tightens packet validation against control packets of other sessions. On `frr:10.5.0` a client's multihop session keeps the minimum TTL its client sets, whatever its profile or a matching `peer` says | bfd › Peer / Profile Configuration |

## Control Plane Minimization

The `daemons` file enables `bfdd` the way `ospfd=yes` enables *ospfd*, only when the file has a
Binding line or a `peer`, and any other daemon only per the table in `TECH.md`. No session runs
for a client the request did not name. [ospfd › Multi-instance Support; bfd › BFDd Commands]

## Security Regression

Removing `minimum-ttl` from a configured multihop `peer`, or making a single-hop `peer`
multihop, is built only after a `SECURITY WARNING` and explicit user confirmation. Without the
confirmation, nothing is built. [bfd › Peer / Profile Configuration; bfd › BFDd Commands]

## Performance

| Rule | Note | Source |
|------|------|--------|
| Timers changed only with numeric justification | State the detection time, not just the values | bfd › Peer / Profile Configuration |
| `echo-mode` only in a profile or `peer` of single-hop sessions toward a peer that is also FRR | Echo mode is not supported on multihop sessions, and without distributed BFD, which is out of this pack, it works only when the peer is also FRR | bfd › Peer / Profile Configuration |
| A profile or `peer` with `echo-mode` raises `transmit-interval`: 2000 unless the request gives another value | Fewer control packets reduce bandwidth usage | bfd › Peer / Profile Configuration |
| `passive-mode` only on the central node of a star network | It waits for the peer's control packets before it begins replying, and sends none it does not need. Each spoke keeps the default active mode: a passive session does not start the connection | bfd › Peer / Profile Configuration |
| Every timer, threshold, or limit carries a `!` comment stating its expected effect: a control-packet timer (`detect-multiplier`, `receive-interval`, `transmit-interval`) the detection time it gives, an echo timer its interval | The peer declares the session down after this detect-multiplier times the larger of this transmit-interval and its own receive-interval | bfd › Peer / Profile Configuration |
