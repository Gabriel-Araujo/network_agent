# LDP

MPLS label distribution by LDP: the `mpls ldp` block, discovery interfaces and transport
addresses, IPv4 and IPv6 address families on a dual-stack LSR, session holdtime and hello
timers, ordered label distribution control, GTSM.

## Scope

This pack owns the `mpls ldp` block: no other pack writes a line in it, and this pack writes
there every line not listed out of it below. The `interface` lines of its address families are
this pack's too: they open LDP's own interface nodes, not the interface blocks of the skill body.
The IGP that carries the transport addresses is written by its own pack.

Out of this pack:

- Layer 2 VPNs over LDP: `l2vpn ... type vpls`, its bridge, members and pseudowires, which
  `ldpd.rst` shows only in a sample with no `clicmd`.
- Targeted LDP sessions to neighbors more than one hop away, and the `neighbor <lsr-id>
  ttl-security hops` that serves them: the corpus has no `clicmd` that opens or accepts a
  targeted session.
- Label policy (`label local advertise explicit-null`) and `dual-stack cisco-interop`, which the
  sample shows with no `clicmd`.
- LDP-IGP synchronization and the Remote LFA tunnels of IS-IS, which the corpus names only in
  debug and IS-IS commands.
- The MPLS forwarding of the kernel, which this file does not configure.

## Daemons

| Daemon | When |
|--------|------|
| `zebra` | Always: it must be running before *ldpd* starts, and it holds the MPLS table *ldpd* fills |
| `ldpd` | Always |

The IGP runs the daemon its own pack names. Anything beyond this table needs an explicit
justification.

## Vocabulary

Use these terms with exactly these meanings, in config comments and in what is said about the
configuration.

| Term | Means | Not |
|------|-------|-----|
| LSR | A router that forwards traffic between and through MPLS routers by label | P router, MPLS node |
| LER | A router at the edge of an MPLS network, generally between an IP network and an MPLS network | PE, edge router |
| LSR-ID | The `router-id` of `mpls ldp`, by which every neighbor names this LSR | transport address |
| transport address | The address of this LSR that its LDP sessions use, one per address family | LSR-ID, update-source |
| discovery | The hello messages on UDP port 646 by which LDP finds neighbors on an interface | adjacency |
| LDP session | The TCP connection on port 646 between two LSRs, over which they exchange label information | adjacency, peering |
| label binding | A label an LSR binds to a FEC and advertises to its neighbors | label route |
| FEC | The destination a label is bound to, a prefix of the routing table | LSP |
| independent control | The mode the implementation supports without `ordered-control`: each LSR binds labels to FECs with no dependency on another router | unordered mode |
| ordered control | An LSR binds a label to a FEC only as egress or once its next hop has sent one for it | conservative mode |
| GTSM | The TTL check of RFC 6720 (IPv4) and RFC 7552 (IPv6): incoming packets need a TTL of 256 minus the hops allowed | TTL filter |
| dual-stack LSR | An LSR with LDP in both address families | |

## Required Facts

An identity fact is never guessed; an operational preference may take a default. A fact the
request, or a file already read, gives is not asked.

### Always Ask

Each of these blocks the build while missing.

| Missing | Question | Why |
|---------|----------|-----|
| LSR-ID | What is the LDP router-id of this router? | Neighbors name this LSR by it in their `neighbor` lines, and `show mpls ldp neighbor` lists it as the ID |
| Interfaces | On which interfaces does LDP discover neighbors? | Discovery is disabled on every interface by default |
| Transport address | Which address of this router do its LDP sessions use, one the IGP advertises? | `discovery transport-address` cannot be written without it, and on `frr:10.5.0` an address family without one forms no session |

### Ask When the Request Implies It

| Missing | Question |
|---------|----------|
| IGP | The request names no IGP and no file read carries one: which IGP carries the transport address of every LSR of this network? |
| IPv6 | Does LDP also run for IPv6? Which IPv6 transport address, and on which interfaces? |
| Transport preference | This LSR is dual-stack: do its sessions use TCP over IPv6, the default, or over IPv4? |
| Authenticated neighbors | Which neighbors, by LSR-ID, must authenticate their LDP sessions? |
| Session holdtime | Which session holdtime does the session with `<lsr-id>` use, between 15 and 65535 seconds? |
| Hello timers | Which hello interval and hello holdtime does `<interface>` use, or every interface? |
| Crowded segment | Does `<interface>` reach many LDP routers on one network? |
| Label control | Does this network use ordered control, which prevents blackholing during convergence and slows it down? |
| Older implementation | Is an LSR on these links an older implementation that needs GTSM disabled, for the IPv6 address family or for its LSR-ID? Which one? |

### Warn, Never Ask

Missing, these do not block: the configuration carries the warning that `doctrine.md` gives for
each, and no invented value.

- Authentication: the password of a neighbor is a secret, never written
- The MPLS forwarding of the kernel, which this file does not configure

### Never Ask

- FRR version: `frr version` is never written
- Labels: *ldpd* binds them to the FECs of the routing table

### May Be Filled In

Nothing. No LDP line carries a local name, and a value with operational effect (a timer, a
holdtime) is never filled in: without one in the request, its line is not written and the FRR
default holds.

### Question Form

One question per item, direct, with the reason attached.

```
Which address of this router do its LDP sessions use as transport address?
why: `discovery transport-address` cannot be written without it; on `frr:10.5.0` the address
family forms no LDP session without it, and a neighbor with no route to that address forms none
with this router, so the IGP must advertise it.
```

Never ask in bulk ("send me the network details").

## Examples

Canonical shapes of `frr.conf`, one file each under `references/ldp/examples/`. Read the one
closest to the request; copy its structure, never its values.

| File | Shape |
|------|-------|
| `references/ldp/examples/ldp-core-lsr.md` | Core LSR: LDP on two transit links, the transport address on the loopback, an OSPF IGP |
| `references/ldp/examples/ldp-authenticated-neighbors.md` | LSR whose named neighbors authenticate their sessions and take a session holdtime |
| `references/ldp/examples/ldp-ordered-control.md` | LSR of a network that binds labels in ordered control |
| `references/ldp/examples/ldp-crowded-segment.md` | LSR on a network with many LDP routers: no extra hellos at session establishment, slower hellos |
| `references/ldp/examples/ldp-dual-stack.md` | Dual-stack LSR: both address families, sessions over TCP on IPv4 |

## Verification Commands

What an operator runs to verify the configuration.

```
vtysh -c "show mpls ldp neighbor"
vtysh -c "show mpls ldp neighbor detail"
vtysh -c "show mpls ldp discovery detail"
vtysh -c "show mpls ldp ipv4 interface"
vtysh -c "show mpls ldp ipv6 interface"
vtysh -c "show mpls ldp ipv4 binding"
vtysh -c "show mpls table"
vtysh -c "show ip route"
```
