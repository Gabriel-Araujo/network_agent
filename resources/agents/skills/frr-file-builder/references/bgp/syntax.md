# BGP Syntax

How the BGP part of a configuration is written. The file header and the block order are the
generic rules of the skill body; the policy objects and the warnings come from `doctrine.md`.

## Shape

One eBGP session advertising one prefix, with no inbound policy, authentication or
`maximum-prefix` given by the request. Replace every `<...>` with a value from the request,
and keep the order, the indentation and the `!` separator lines:

```
ip prefix-list <OUT-NAME> seq 5 permit <prefix to advertise>
ip prefix-list <OUT-NAME> seq 10 deny any
!
! WARNING: nothing is accepted from <peer address> until an inbound policy says what to accept
ip prefix-list <IN-NAME> seq 5 deny any
!
router bgp <local AS>
 bgp router-id <router-id>
 neighbor <peer address> remote-as <peer AS>
 neighbor <peer address> description <who the peer is and why the session exists>
 ! WARNING: no authentication on the session with <peer address>
 !
 address-family ipv4 unicast
  neighbor <peer address> activate
  neighbor <peer address> prefix-list <IN-NAME> in
  neighbor <peer address> prefix-list <OUT-NAME> out
  ! WARNING: no maximum-prefix on <peer address>
  ! WARNING: <prefix to advertise> must be in the RIB, or BGP will not advertise it
  network <prefix to advertise>
 exit-address-family
!
```

## Rules

The shape generalizes by these rules:

- every line that configures a neighbor repeats `neighbor <peer address>`. FRR has no
  neighbor sub-mode: a bare `description`, `activate` or `prefix-list` line is not a valid
  command;
- `remote-as` and `description` go at the `router bgp` level; `activate`, `prefix-list`,
  `maximum-prefix` and every `network` line go inside the address-family. Policy written at
  the global level silently does nothing, and a `network` line outside an address-family is
  not a valid command;
- every neighbor gets its own `activate` in the address-family it uses: without it the session
  comes up and carries no routes;
- every neighbor gets a `description` naming the peer and why the session exists;
- one `address-family` block per address family, never two, and every block closes with
  `exit-address-family`;
- three or more peers sharing one policy go in a peer-group; below three, configure each
  neighbor directly;
- every `network` line gets a warning right above it that the prefix must be in the RIB for
  BGP to advertise it, because `bgp network import-check` is on by default.
