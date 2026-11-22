# Multihop Sessions

The request runs BFD toward peers more than one hop away: a static route whose gateway is
several hops away, with `bfd multi-hop profile MULTIHOP`, and a BGP neighbor that is not directly
connected, with `neighbor <peer> bfd profile MULTIHOP`. Their sessions are multihop sessions,
which expect packets with a TTL under 254. Echo mode is not supported on them, so their profile
stays apart from any profile with `echo-mode`. The request gives a detect-multiplier of 3 and
500 ms intervals for these sessions.

A reference for shape. Copy the structure, never the values or names.

```
bfd
 profile MULTIHOP
  ! the peer declares a session down after 3 x 500 ms = 1500 ms without packets,
  ! while its own receive-interval is 500 ms or less
  detect-multiplier 3
  receive-interval 500
  transmit-interval 500
 exit
exit
!
```

What the shape demonstrates:

- a profile for multihop sessions only, apart from the profiles with `echo-mode`
- no `minimum-ttl`: a client's multihop session keeps the minimum TTL its client sets

Sources: bfd › Peer / Profile Configuration; bfd › BFDd Commands; bfd › BGP BFD Configuration; bfd › BFD Static Route Monitoring Configuration.
