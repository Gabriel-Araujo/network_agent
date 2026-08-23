# Mitigation With RTBH And FlowSpec

FlowSpec is outside this pack's scope per `TECH.md`: ask for a run of the
matching technology. RTBH is in scope.

RTBH uses the BLACKHOLE community (RFC 7999) to propagate a drop by
destination; FlowSpec (RFC 8955) propagates rules by 5-tuple. Both use BGP
as the bus for a security policy.

```
! ---- RTBH: on the trigger node ----
ip route 198.51.100.66/32 blackhole
ip prefix-list PL-BLACKHOLE seq 10 permit 198.51.100.66/32
!
route-map RM-RTBH-OUT permit 10
 match ip address prefix-list PL-BLACKHOLE
 set community 65535:666 no-export additive
exit
!
! ---- RTBH: on the borders ----
ip route 192.0.2.1/32 blackhole
bgp community-list standard CL-BLACKHOLE permit 65535:666
!
route-map RM-EDGE-IN permit 10
 match community CL-BLACKHOLE
 set ip next-hop 192.0.2.1
exit
route-map RM-EDGE-IN permit 20
exit
!
! ---- FlowSpec: reception and local install ----
router bgp 65001
 address-family ipv4 flowspec
  neighbor 10.0.0.9 activate
  local-install eth1
 exit-address-family
exit
!
! FRR receives and installs FlowSpec; origination usually comes from ExaBGP/GoBGP
! verification: show bgp ipv4 flowspec detail
```

What the shape demonstrates:

- the trigger and the border are two different configurations of one
  mechanism
- `no-export` on the RTBH advertisement, so the drop does not escape the AS
- the discard next-hop resolved by a local blackhole static, which is what
  actually drops the traffic
- FlowSpec received and installed on a named interface, not applied globally
