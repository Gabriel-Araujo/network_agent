# Core Router On Point-To-Point Links

A core router joins the backbone over two links, each to exactly one other router. Only those
two links speak OSPF; the loopback, which iBGP and management use, is advertised as a stub
link. The adjacencies are authenticated through a key chain and watched by BFD, so a dead link
is detected before the dead interval expires.

A reference for shape. Copy the structure, never the router-id, names or interfaces.

```
key chain OSPF-CORE
 key 1
  cryptographic-algorithm hmac-sha-256
  ! WARNING: key 1 of OSPF-CORE has no key-string; set it on the router, never in this file
 exit
exit
!
interface lo
 ip ospf area 0.0.0.0
exit
!
interface eth1
 ip ospf area 0.0.0.0
 ip ospf network point-to-point
 no ip ospf passive
 ip ospf authentication key-chain OSPF-CORE
 ip ospf bfd
exit
!
interface eth2
 ip ospf area 0.0.0.0
 ip ospf network point-to-point
 no ip ospf passive
 ip ospf authentication key-chain OSPF-CORE
 ip ospf bfd
exit
!
router ospf
 ospf router-id 10.0.0.1
 passive-interface default
 ! 5 min as a stub router after boot, until the LSDB settles
 max-metric router-lsa on-startup 300
exit
!
```

What the shape demonstrates:

- OSPF enabled per interface, never with `network` under `router ospf`
- passive by default, with transit opted in per interface
- the loopback advertised as a stub link, not redistributed
- one key chain shared by both links, referenced by name
- BFD bound per interface, so it covers every adjacency on the link
- `max-metric router-lsa on-startup` on a router that carries transit traffic

Sources: ospfd › Routers; ospfd › Interfaces; bfd › OSPF BFD Configuration.
