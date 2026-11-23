# BFD

The `bfd` block that other protocols and static routes use to detect a failed path: its
profiles and peers, detection timers, echo mode, passive mode, single-hop and multihop
sessions.

## Scope

A Binding line belongs to its client's pack: `ip ospf bfd` to `ospf`, `ipv6 ospf6 bfd` to
`ospf6`, `isis bfd` to `isis`, `neighbor <peer> bfd` to `bgp`. A static route monitored by BFD
is a static route of the skill body. This pack writes the `bfd` block those lines name, and
decides what goes in it.

Out of this pack: authentication (`authentication key-chain`, `authentication algorithm
meticulous`) and `demand-mode`, which `bfd.rst` documents and `vtysh` of `frr:10.5.0`, the image
every example is checked against, rejects; distributed BFD and the `--dplaneaddr` and `--vrfs`
options of *bfdd*, which are not `frr.conf` lines; PTM; the PIM and RIP bindings.

## Daemons

| Daemon | When |
|--------|------|
| `bfdd` | Only if the file has a Binding line or a `peer` |
| `zebra` | Always: *bfdd* learns interfaces, local addresses and VRFs from it |
| `staticd` | Only if a static route is monitored by BFD |

Each client runs the daemon its own pack names. Anything beyond this table needs an explicit
justification.

## Vocabulary

Use these terms with exactly these meanings, in config comments and in what is said about the
configuration.

| Term | Means | Not |
|------|-------|-----|
| session | The BFD exchange between this router and one peer address | adjacency, neighbor |
| client | The protocol, or the static route, whose Binding line opens a session or uses one that exists, and acts on its state | consumer, user |
| profile | A named set of session settings in the `bfd` block, which Binding lines and peers name | template, policy |
| configured peer | A session the `bfd` block opens itself, with `peer` | static peer, manual peer |
| single-hop | A session to a peer on a directly connected link | direct session |
| multihop | A session to a peer more than one hop away, on port 4784 | remote session |
| detection time | How long the peer waits without packets before it declares the session down: this detect-multiplier times the larger of this transmit-interval and the peer's receive-interval | hold time, timeout |
| echo mode | Echo packets that the peer loops back, next to the control packets | echo test, ping |
| passive mode | A session that does not start the connection and waits for the peer's control packets | silent mode, listen mode |

## Required Facts

An identity fact is never guessed; an operational preference may take a default. A fact the
request, or a file already read, gives is not asked.

### Always Ask

Each of these blocks the build while missing.

| Missing | Question | Why |
|---------|----------|-----|
| Clients | Which adjacencies, sessions or static routes does BFD watch? | Each one gets its own Binding line, and BFD watches only what the request names |

### Ask When the Request Implies It

| Missing | Question |
|---------|----------|
| Timers | Which detect-multiplier, receive-interval and transmit-interval do the sessions of `<client>` use? |
| Hops | Is `<peer>` on a directly connected link, or more than one hop away? |
| Minimum TTL | What minimum TTL do the control packets of the configured multihop peer `<peer>` arrive with? The default, 254, expects one hop |
| Peer implementation | Is the router at the other end of `<link>` an FRR router too? |
| Star hub | Is this router the central node of a star network, waiting for its spokes to start the sessions? |
| Local address | Which local address does the configured peer `<peer>` use? A multihop or IPv6 peer needs one |
| VRF | Is the configured peer `<peer>` in a VRF? Which one? |

### Warn, Never Ask

Nothing. Authentication, which `bfd.rst` gives to keep sessions between trusted routers, is out
of this pack (see Scope).

### Never Ask

- FRR version: `frr version` is never written

### May Be Filled In

Nothing here carries security impact or invents an identity.

- profile names
- `transmit-interval` of a profile with echo mode, where doctrine fixes one

### Question Form

One question per item, direct, with the reason attached.

```
Which detect-multiplier and intervals do the BFD sessions of the OSPF links use?
why: the profile carries only values the request gives; with none, the sessions keep the
defaults, a detect-multiplier of 3 and 300 ms intervals.
```

Never ask in bulk ("send me the network details").

## Examples

Canonical shapes of `frr.conf`, one file each under `references/bfd/examples/`. Read the one
closest to the request; copy its structure, never its values.

| File | Shape |
|------|-------|
| `references/bfd/examples/bfd-single-hop-profile.md` | One profile with faster timers for the sessions of directly connected clients |
| `references/bfd/examples/bfd-echo-mode.md` | Echo mode between two FRR routers on one link, control packets slowed down |
| `references/bfd/examples/bfd-multihop-profile.md` | A profile for multihop sessions, apart from the profiles with echo mode |
| `references/bfd/examples/bfd-passive-hub.md` | Hub of a star network: passive sessions that log their state changes |
| `references/bfd/examples/bfd-static-route.md` | IPv4 and IPv6 static routes installed only while BFD sees their gateway |

## Verification Commands

What an operator runs to verify the configuration.

```
vtysh -c "show bfd peers"
vtysh -c "show bfd peers brief"
vtysh -c "show bfd peer <peer>"
vtysh -c "show bfd vrf <vrf> peers"
vtysh -c "show bfd static route"
```
