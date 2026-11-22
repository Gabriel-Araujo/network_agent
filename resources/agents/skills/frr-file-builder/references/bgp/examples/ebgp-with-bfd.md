# eBGP With BFD

BGP detects failure by hold timer, which is tens of seconds. BFD brings that
under a second without aggressive BGP keepalives. The two complement each
other: BFD detects, BGP reacts.

```
bfd
 profile FABRIC
  ! the peer declares a session down after 3 x 100 ms = 300 ms without packets,
  ! while its own receive-interval is 100 ms or less
  detect-multiplier 3
  receive-interval 100
  transmit-interval 100
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

- `bfdd` added to the daemon set
- the peer-group binding every session to the BFD profile on one line
- `remote-as external`, valid with interface-based unnumbered peers
- `maximum-paths` inside the address family, which is where ECMP is decided
- verification commands named next to the feature that needs them
