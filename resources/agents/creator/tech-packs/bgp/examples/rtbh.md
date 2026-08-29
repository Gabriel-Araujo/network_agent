# Remote Triggered Black Hole

Propagates a drop by destination using the BLACKHOLE community (RFC 7999),
so the discard happens at the edge instead of at the victim.

```
! ---- on the trigger node ----
ip route 198.51.100.66/32 blackhole
ip prefix-list PL-BLACKHOLE seq 10 permit 198.51.100.66/32
!
route-map RM-RTBH-OUT permit 10
 match ip address prefix-list PL-BLACKHOLE
 set community 65535:666 no-export additive
exit
!
! ---- on the borders ----
ip route 192.0.2.1/32 blackhole
bgp community-list standard CL-BLACKHOLE permit 65535:666
!
route-map RM-EDGE-IN permit 10
 match community CL-BLACKHOLE
 set ip next-hop 192.0.2.1
exit
route-map RM-EDGE-IN permit 20
exit
```

What the shape demonstrates:

- the trigger and the border are two different configurations of one
  mechanism
- `no-export` on the RTBH advertisement, so the drop does not escape the AS
- the discard next-hop resolved by a local blackhole static, which is what
  actually drops the traffic
