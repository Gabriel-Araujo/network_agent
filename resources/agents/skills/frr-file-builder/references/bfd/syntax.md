# BFD Syntax

How the BFD part of a configuration is written. The file header, the block order and the
static routes are rules of the skill body; each Binding line is written by its client's pack.

## Shape

One profile with the timers the request gives, for the sessions its clients create. Replace
every `<...>` with a value from the request, and keep the order, the indentation and the `!`
separator lines:

```
bfd
 profile <profile>
  ! the peer declares a session down after <detect-multiplier> x <interval> ms without packets,
  ! while its own receive-interval is <interval> ms or less
  detect-multiplier <detect-multiplier>
  receive-interval <interval>
  transmit-interval <interval>
 exit
exit
!
```

## Rules

The shape generalizes by these rules. Each cites the `.rst` section it rests on.

- The file has one `bfd` block, which opens the BFD daemon configuration node. [bfd › BFDd Commands]
  It, and each `profile <name>` and `peer` inside it, closes with `exit` the way the skill body
  closes every block, and the settings of each are indented one level deeper.
- The `bfd` block holds named objects, its profiles, so it goes where the skill body puts the
  named objects of a pack: after the Shared objects and before the static routes. Every profile
  then comes before the Binding lines that name it.
- The session settings are `detect-multiplier`, `receive-interval`, `transmit-interval`,
  `echo-mode`, `echo transmit-interval`, `echo receive-interval`, `passive-mode`, `minimum-ttl`,
  `shutdown` and `log-session-changes`. Profiles and peers share them. [bfd › Peer / Profile Configuration]
- Intervals are in milliseconds: `receive-interval`, `transmit-interval` and
  `echo transmit-interval` take 10 to 4294967, `echo receive-interval` the same or `disabled`,
  and `detect-multiplier` takes 1 to 255. [bfd › Peer / Profile Configuration]
- A client names a profile with its own `profile` form: `ip ospf bfd profile <profile>`,
  `isis bfd profile <profile>`, `neighbor <peer> bfd profile <profile>`, and `bfd profile
  <profile>` or `bfd multi-hop profile <profile>` at the end of a static route.
  [bfd › OSPF BFD Configuration; bfd › IS-IS BFD Configuration; bfd › BGP BFD Configuration; bfd › BFD Static Route Monitoring Configuration]
- A configured peer is `peer <address>`, with `profile <profile>` inside when it uses one. A peer
  more than one hop away is `peer <address> multihop local-address <address>`, and an IPv6 peer
  always carries `local-address`. `interface <name>` selects the interface the peer uses, and
  `vrf <vrf>` its VRF. [bfd › BFDd Commands; bfd › BFD Peer Specific Commands; bfd › Configuration]
