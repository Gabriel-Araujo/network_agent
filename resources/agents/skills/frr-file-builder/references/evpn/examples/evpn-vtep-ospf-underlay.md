# VTEP Over An OSPF Underlay

A leaf is a VTEP. Its VTEP address, the one the VXLAN devices on the host use as
their local tunnel address, is on the loopback, and OSPF advertises it to the other VTEPs over
the one point-to-point uplink to the spine. One iBGP session, to the spine's loopback, carries
EVPN: the spine is the route reflector of the EVPN routes. The leaf's L2VNIs are the ones its
VXLAN devices define on the host.

A reference for shape. Copy the structure, never the AS, addresses, names or interfaces.

```
interface lo
 ip address 10.255.0.11/32
 ip ospf area 0.0.0.0
exit
!
interface eth1
 ip ospf area 0.0.0.0
 ip ospf network point-to-point
 no ip ospf passive
 ! WARNING: no authentication on the adjacencies of eth1
exit
!
router ospf
 ospf router-id 10.255.0.11
 passive-interface default
exit
!
router bgp 65000
 bgp router-id 10.255.0.11
 neighbor 10.255.0.1 remote-as 65000
 neighbor 10.255.0.1 description spine1, the route reflector of the EVPN routes
 ! WARNING: no authentication on the session with 10.255.0.1
 neighbor 10.255.0.1 update-source lo
 !
 ! WARNING: FRR does not create the VXLAN devices, bridges and SVIs of these VNIs; zebra learns them from the kernel
 address-family l2vpn evpn
  neighbor 10.255.0.1 activate
  advertise-all-vni
 exit-address-family
exit
!
```

What the shape demonstrates:

- EVPN enabled by `advertise-all-vni` in the BGP instance of the default VRF, the underlay
- the one EVPN session activated in `address-family l2vpn evpn`
- no VNI named: zebra learns the VNIs from the kernel and, on `frr:10.5.0`, `advertise-all-vni` advertises them
- the VTEP address reachable through the underlay the request names, here OSPF
- the warning that the VXLAN devices, bridges and SVIs live on the host

Sources: bgp › Enabling EVPN; evpn › Sample Configuration; evpn › FRR Configuration; evpn › Linux Interface Configuration; bgp › ASN and Router ID; bgp › Defining Peers; bgp › Configuring Peers; ospfd › Routers; ospfd › Interfaces.
