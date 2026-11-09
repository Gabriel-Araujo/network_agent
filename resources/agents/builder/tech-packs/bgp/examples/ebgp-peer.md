# eBGP Session With Policy Both Ways

A reference for shape. Copy the structure, never the ASNs, prefixes, or
names. Addresses here come from RFC 5737 and RFC 5398 documentation ranges.

```
! policy objects before the block that references them
ip prefix-list UPSTREAM-IN seq 5 deny 10.0.0.0/8 le 32
ip prefix-list UPSTREAM-IN seq 10 permit 0.0.0.0/0 le 24
!
ip prefix-list OWN-BLOCKS-OUT seq 5 permit 198.51.100.0/24
ip prefix-list OWN-BLOCKS-OUT seq 10 deny any
!
router bgp 65001
 bgp router-id 198.51.100.1
 neighbor 203.0.113.2 remote-as 65010
 neighbor 203.0.113.2 description upstream-A / primary transit
 neighbor 203.0.113.2 password TRANSIT-A           ! reference, value lives outside the artifact
 neighbor 203.0.113.2 ttl-security hops 1          ! GTSM, drops packets with a forged TTL
 neighbor 203.0.113.2 bfd
 !
 address-family ipv4 unicast
  neighbor 203.0.113.2 activate
  neighbor 203.0.113.2 prefix-list UPSTREAM-IN in
  neighbor 203.0.113.2 prefix-list OWN-BLOCKS-OUT out
  neighbor 203.0.113.2 maximum-prefix 1000000 restart 15   ! ceiling about 2x current full table
 exit-address-family
!
```

What the shape demonstrates:

- a policy object declared before use
- `remote-as` at the global level, `activate` and policy and `maximum-prefix`
  inside the address family
- both directions filtered. The outbound one is the most often forgotten and
  the one that causes route leaks
- a password referenced by name
- a `!` comment on every line carrying a number
- `bgp ebgp-requires-policy` left at its default, which is on. The artifact
  has policy both ways, so there is no reason to disable the guard
