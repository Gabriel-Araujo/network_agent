# eBGP With BFD

BGP detects failure by hold timer, which is tens of seconds. BFD brings that
under a second without aggressive BGP keepalives. The two complement each
other: BFD detects, BGP reacts.

```
bfd
 profile FABRIC
  detect-multiplier 3
  receive-interval 300
  transmit-interval 300
 exit
exit
!
router bgp 65001
 bgp router-id 10.0.0.1
 no bgp default ipv4-unicast
 !
 neighbor FABRIC peer-group
 neighbor FABRIC remote-as external
 ! bind the BGP session to the BFD profile
 neighbor FABRIC bfd profile FABRIC
 neighbor eth1 interface peer-group FABRIC
 neighbor eth2 interface peer-group FABRIC
 !
 address-family ipv4 unicast
  network 10.0.0.1/32
  neighbor FABRIC activate
  maximum-paths 8
 exit-address-family
exit
!
! verification: show bfd peers brief / show bgp neighbor
```

What the shape demonstrates:

- `bfdd` added to the daemon set, which the plan has to justify
- a named BFD profile reused by the peer-group instead of per-neighbor
  timers
- `remote-as external`, valid with interface-based unnumbered peers
- `maximum-paths` inside the address family, which is where ECMP is decided
- verification commands named next to the feature that needs them
