# Fast Detection On Directly Connected Links

The adjacencies and sessions of this router run over directly connected links, and the request
gives the detection their BFD sessions use: a detect-multiplier of 3 and 100 ms intervals, where
the defaults are 3 and 300 ms. One profile carries those values, and each client names it on its
own Binding line, written by the client's pack: `ip ospf bfd profile LINKS` below `ip ospf bfd`,
`isis bfd profile LINKS` below `isis bfd`, `neighbor <peer> bfd profile LINKS`. An update to the
profile later applies to all peers with the profile, without interruptions.

A reference for shape. Copy the structure, never the values or names.

```
bfd
 profile LINKS
  ! the peer declares a session down after 3 x 100 ms = 300 ms without packets,
  ! while its own receive-interval is 100 ms or less
  detect-multiplier 3
  receive-interval 100
  transmit-interval 100
 exit
exit
!
```

What the shape demonstrates:

- one named profile for every session that wants the same detection, named on each Binding line
- only the values the request gives, the timers under a comment with the detection time they give

Sources: bfd › Peer / Profile Configuration; bfd › BFDd Commands; bfd › BFD Peer Specific Commands; bfd › OSPF BFD Configuration; bfd › IS-IS BFD Configuration; bfd › BGP BFD Configuration.
