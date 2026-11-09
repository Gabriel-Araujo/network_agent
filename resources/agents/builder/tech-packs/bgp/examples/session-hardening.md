# Session Hardening

Four independent layers: TCP authentication, TTL restriction, prefix ceiling
and flap damping. None of them substitutes for the others.

```
router bgp 65001
 neighbor 203.0.113.1 remote-as 64512
 neighbor 203.0.113.1 description upstream / primary transit
 ! TCP-MD5 authentication (a vault reference, never a literal)
 neighbor 203.0.113.1 password {{BGP_SECRET}}
 ! GTSM: accept only packets whose TTL is consistent with one hop
 neighbor 203.0.113.1 ttl-security hops 1
 neighbor 203.0.113.1 timers 10 30
 !
 address-family ipv4 unicast
  neighbor 203.0.113.1 activate
  ! tear down above 1M prefixes, warn at 90%, retry in 30 min
  neighbor 203.0.113.1 maximum-prefix 1000000 90 restart 30
  ! penalise unstable prefixes: half-life 15min, reuse 750, suppress 2000, max 60min
  bgp dampening 15 750 2000 60
 exit-address-family
exit
!
! dampening is contested on a full-table transit session: evaluate before enabling
! verification: show bgp ipv4 unicast dampening dampened-paths
```

What the shape demonstrates:

- a secret referenced by name, so the artifact never carries the value
- `ttl-security` as a cheap, per-session spoofing guard on a single-hop eBGP
  session
- `maximum-prefix` inside the address family, with a ceiling stated relative
  to the current full table
- a `!` comment on every line carrying a number
- a contested default flagged in the artifact rather than applied silently
