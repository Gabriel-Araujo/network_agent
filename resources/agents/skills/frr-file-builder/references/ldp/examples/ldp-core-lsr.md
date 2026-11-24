# Core LSR Over An OSPF IGP

A core LSR of an MPLS backbone runs LDP on its two transit links, each to one other LSR. Its
transport address is on the loopback, and OSPF, the IGP the request names, advertises it to the
other LSRs, so each LDP session can reach it. The request asks for no authentication.

A reference for shape. Copy the structure, never the router-id, addresses or interfaces.

```
interface lo
 ip address 10.255.0.2/32
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
interface eth2
 ip ospf area 0.0.0.0
 ip ospf network point-to-point
 no ip ospf passive
 ! WARNING: no authentication on the adjacencies of eth2
exit
!
router ospf
 ospf router-id 10.255.0.2
 passive-interface default
exit
!
! WARNING: label forwarding needs MPLS in the kernel; without it the LDP sessions come up and no label is installed
mpls ldp
 router-id 10.255.0.2
 ! WARNING: no authentication on the LDP sessions of this router
 !
 address-family ipv4
  discovery transport-address 10.255.0.2
  !
  interface eth1
  exit
  !
  interface eth2
  exit
  !
 exit-address-family
 !
exit
!
```

What the shape demonstrates:

- discovery enabled only on the two transit interfaces, inside the address family
- the transport address on the loopback, which the IGP advertises
- the IGP lines written by the `ospf` pack, its warnings included
- the no-authentication warning in place of any `password` line
- the warning that label forwarding needs MPLS in the kernel

Sources: ldpd › LDP Configuration; ldpd › Sample configuration; ldpd › LDP; ospfd › Routers; ospfd › Interfaces.
