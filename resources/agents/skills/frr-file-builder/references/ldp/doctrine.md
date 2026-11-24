# LDP Doctrine

Default posture is opt-in: LDP discovers neighbors only on the interfaces the request names, in
the address families it names, over a transport address the request's IGP carries; a timer,
the label control mode or the transport preference changes only with the value the request
gives. Each rule below is met, not applicable for a stated reason, or a regression. The Source
column names the `.rst` section that carries the command or the behaviour the rule rests on. A
behaviour the corpus does not state, proved on a live `frr:10.5.0`, says so.

## Discovery Policy

What every LDP configuration carries, whatever else the request asks for.

| Rule | Source |
|------|--------|
| `router-id` in `mpls ldp` with the value the request gives | ldpd › LDP Configuration |
| `interface <name>` in an address-family block only for the interfaces the request names: discovery is disabled on every interface by default | ldpd › LDP Configuration |
| `discovery transport-address` in every address-family block, with the address the request gives, where the sample configurations and `show running-config` on `frr:10.5.0` keep it (the `clicmd` places it under the address-family interface node). On `frr:10.5.0`, an address family without one brings up none of its interfaces and forms no session, and `show running-config` marks it `! Incomplete config, specify a discovery transport-address` | ldpd › LDP Configuration; ldpd › Sample configuration |
| The transport address is one the IGP the request names advertises, and the IGP lines follow that IGP's pack: LDP is typically deployed with an IGP like OSPF, that helps discover the remote IPs. On `frr:10.5.0`, a neighbor whose transport address this router has no route to forms no session, though its hellos arrive | ldpd › Sample configuration |
| No `neighbor <lsr-id> password`: PASSWORD is a clear text password, the secret itself. Each neighbor the request asks to authenticate gets `! WARNING: neighbor <lsr-id> password is left out; add it on the router, never in this file`; with none asked, `mpls ldp` carries `! WARNING: no authentication on the LDP sessions of this router` below `router-id` | ldpd › LDP Configuration |
| Above `mpls ldp`, the warning `! WARNING: label forwarding needs MPLS in the kernel; without it the LDP sessions come up and no label is installed`. *ldpd* stores the label information in the MPLS table of *zebra*, which injects it in the underlying system. On `frr:10.5.0` over a kernel without MPLS forwarding, the sessions are OPERATIONAL and their bindings in use, and `show mpls table` is empty | ldpd › LDP |

The rest of the tables below apply when the request asks for the feature or gives what it
needs; absent, they add neither a line nor a warning.

## Security

| Rule | Note | Source |
|------|------|--------|
| GTSM stays enabled: `ttl-security disable` in an address-family block, or `neighbor <lsr-id> ttl-security disable`, only for an LSR implementation the request names as needing it, with a `SECURITY NOTE` naming it. The corpus gives this reason for the IPv6 address family; for IPv4 and for one neighbor it is this pack's policy | GTSM requires incoming packets to have a TTL/hop limit of 256 minus the hops allowed, 1 by default, ensuring they have not passed through more than the expected number of hops. GTSM is mandatory for LDPv6: disabling it for IPv6 only stops *ldpd* from discarding packets with a hop limit below 255, which may be necessary to interoperate with older implementations; outgoing packets are still sent with a hop limit of 255. The corpus gives multi-hop neighbors, out of this pack, as the reason to disable it for one neighbor | ldpd › LDP Configuration |

## Control Plane Minimization

The `daemons` file enables `ldpd` the way `ospfd=yes` enables *ospfd*, with *zebra*, which must
be running before *ldpd* is invoked; the IGP's daemon follows its own pack, and any other daemon
needs the explicit justification `TECH.md` › Daemons asks for. Discovery runs on no interface
the request did not name. [ospfd › Multi-instance Support; ldpd › Running Ldpd; ldpd › LDP Configuration]

## Security Regression

Disabling GTSM, for an address family or for one neighbor, or adding to discovery an interface
the request did not name, is built only after a `SECURITY WARNING` and explicit user
confirmation. Without the confirmation, nothing is built. [ldpd › LDP Configuration]

## Performance

| Rule | Note | Source |
|------|------|--------|
| `ordered-control` only when the request asks for ordered control or to prevent blackholing during convergence | The implementation supports Liberal Label Retention, Downstream Unsolicited and Independent Control, where each LSR binds labels with no dependency on another router; `ordered-control` configures Ordered Control instead. Ordered Control prevents blackholing during the LDP convergence process, at the cost of slowing down the convergence itself. Both modes are interoperable | ldpd › Understanding LDP principles; ldpd › LDP Configuration |
| `disable-establish-hello` only on an interface the request places on a network with many LDP routers | By default *ldpd* sends additional multicast hellos on TCP session establishment, which should speed up the connection but can produce massive multicast traffic when a lot of ldpd services share one network. With it, hellos are sent only by interval | ldpd › LDP Configuration |
| `discovery hello interval` and `discovery hello holdtime` only with the values the request gives | From 1 to 65535 seconds. The interval, 5 seconds by default, is the time between hello messages; the holdtime, 15 seconds by default, is added as a TLV in the LDP messages. On `frr:10.5.0`, an adjacency takes the smaller holdtime of its two LSRs, so a longer one takes effect only where the neighbors announce as much, and an adjacency that gets no hello for its holdtime goes down, its session with it when it is the last one | ldpd › LDP Configuration |
| `neighbor <lsr-id> session holdtime` only with the value the request gives, from 15 to 65535 seconds | It triggers a keepalive mechanism: after this time of non response, the session is considered down | ldpd › LDP Configuration |
| `dual-stack transport-connection prefer ipv4` only on a dual-stack LSR whose request asks for TCP over IPv4 | By default a dual-stack LSR prefers IPv6 for the transport connection (RFC 7552) and refuses TCP connections over IPv4; with it, label mappings for IPv6 FECs can go over TCPv4 | ldpd › LDP Configuration |
| Every timer carries a `!` comment stating its expected effect | | ldpd › LDP Configuration |
