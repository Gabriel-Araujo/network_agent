# LDP Syntax

How the LDP part of a configuration is written. The file header, the block order, the
interface headers and the Secrets are rules of the skill body; the IGP lines follow the IGP's
pack; the warnings come from `doctrine.md`.

## Shape

One LSR with LDP for IPv4 on one interface, its transport address carried by the IGP, and no
authentication given by the request. Replace every `<...>` with a value from the request, and
keep the order, the indentation and the `!` separator lines:

```
! WARNING: label forwarding needs MPLS in the kernel; without it the LDP sessions come up and no label is installed
mpls ldp
 router-id <router-id>
 ! WARNING: no authentication on the LDP sessions of this router
 !
 address-family ipv4
  discovery transport-address <transport address>
  !
  interface <interface>
  exit
  !
 exit-address-family
 !
exit
!
```

## Rules

The shape generalizes by these rules. Each cites the `.rst` section it rests on.

- LDP is one `mpls ldp` block, the MPLS router node: `router-id` and every `neighbor` line go
  in it. On `frr:10.5.0`, `show running-config` keeps `ordered-control` and
  `dual-stack transport-connection prefer ipv4` there too, and closes the block with `exit`.
  [ldpd › LDP Configuration; ldpd › Sample configuration]
- The lines of one address family go in its `address-family ipv4` or `address-family ipv6`
  block, closed with `exit-address-family`. [ldpd › LDP Configuration; ldpd › Sample configuration]
- Discovery is enabled on an interface by `interface <name>` in the address-family block, which
  opens the address-family interface node. Only the interfaces the request names get one: by
  default discovery is disabled. On `frr:10.5.0`, `show running-config` closes that node with
  `exit`. [ldpd › LDP Configuration]
- Every address-family block carries `discovery transport-address`, an IPv4 address in
  `address-family ipv4` and an IPv6 address in `address-family ipv6`: on `frr:10.5.0`, an
  address family without one forms no session. The `clicmd` places it under the address-family
  interface node; the sample configurations, and `show running-config` on `frr:10.5.0`, keep it
  in the address-family block. [ldpd › LDP Configuration; ldpd › Sample configuration]
- A `neighbor <lsr-id>` line names the neighbor by its LSR-ID, the router-id of its
  `mpls ldp`. [ldpd › Sample configuration; ldpd › Show LDP Information]
- `neighbor <lsr-id> password` is never written: its argument is the secret. In its place goes
  `! WARNING: neighbor <lsr-id> password is left out; add it on the router, never in this file`.
  [ldpd › LDP Configuration]
- `discovery hello interval` and `discovery hello holdtime` take values in seconds. The corpus
  does not place them: on `frr:10.5.0` they go in `mpls ldp` for every interface, or in the
  `interface` block of one interface, and an interface with no value of its own takes the one
  of `mpls ldp`. [ldpd › LDP Configuration]
- `disable-establish-hello` goes in the `interface` block of an address family.
  [ldpd › LDP Configuration]
- `ttl-security disable` for a whole address family goes in its address-family block;
  `neighbor <lsr-id> ttl-security disable` for one neighbor goes in `mpls ldp`.
  [ldpd › LDP Configuration]
- A dual-stack LSR has both address-family blocks, each with its own transport address and its
  own `interface` lines. [ldpd › LDP Configuration; ldpd › Sample configuration]
