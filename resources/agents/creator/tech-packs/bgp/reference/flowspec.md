# FlowSpec

Outside this pack's scope per `TECH.md`: ask for a run of the matching
technology. Kept here as reference, loaded by no stage.

FlowSpec (RFC 8955) propagates filtering rules by 5-tuple, where RTBH
propagates a drop by destination alone. Both use BGP as the bus for a
security policy. For the RTBH shape, see `examples/rtbh.md`.

```
! ---- reception and local install ----
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

- FlowSpec received and installed on a named interface, not applied globally
