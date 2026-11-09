# EVPN/VXLAN Over An eBGP Fabric

Outside this pack's scope per `TECH.md`: ask for a run of the matching
technology. Kept here as the canonical shape for the BGP half of it.

BGP distributes MAC addresses (type 2) and IP prefixes (type 5) as routes;
VXLAN encapsulates. Replaces flood-and-learn and removes STP from the
fabric.

```
! the VXLAN interface and bridge are created outside FRR (ip link / netplan)
! ip link add vxlan100 type vxlan id 100 dstport 4789 local 10.0.0.1 nolearning
!
router bgp 65001
 bgp router-id 10.0.0.1
 no bgp default ipv4-unicast
 bgp bestpath as-path multipath-relax
 !
 neighbor FABRIC peer-group
 neighbor FABRIC remote-as external
 neighbor eth1 interface peer-group FABRIC
 neighbor eth2 interface peer-group FABRIC
 !
 address-family ipv4 unicast
  network 10.0.0.1/32
  neighbor FABRIC activate
 exit-address-family
 !
 address-family l2vpn evpn
  neighbor FABRIC activate
  ! advertise every local VNI automatically
  advertise-all-vni
  advertise-default-gw
 exit-address-family
exit
!
! verification: show bgp l2vpn evpn / show evpn vni / show evpn mac vni 100
```

What the shape demonstrates:

- the underlay (`ipv4 unicast`, loopback reachability) and the overlay
  (`l2vpn evpn`) on the same peer-group, each activated separately
- `multipath-relax`, without which ECMP across distinct spine ASNs does not
  happen
- data-plane objects created outside FRR recorded as comments
- unnumbered interface peers, which is what makes the fabric config
  identical on every leaf
