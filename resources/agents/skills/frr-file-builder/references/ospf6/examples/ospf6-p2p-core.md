# Core Router On Point-To-Point Links

A core router joins the OSPFv3 backbone over two links, each to exactly one other router. Only
those two links form adjacencies; the loopback is passive. The adjacencies carry the
authentication trailer through a key chain and are watched by BFD, for fast convergence when a
link fails. No interface carries an IPv6 address for OSPFv3's sake: it signals over link-local
addresses.

A reference for shape. Copy the structure, never the router-id, names or interfaces.

```
key chain OSPF6-CORE
 key 1
  cryptographic-algorithm hmac-sha-256
  ! WARNING: key 1 of OSPF6-CORE has no key-string; set it on the router, never in this file
 exit
exit
!
interface lo
 ipv6 ospf6 area 0.0.0.0
 ipv6 ospf6 passive
exit
!
interface eth1
 ipv6 ospf6 area 0.0.0.0
 ipv6 ospf6 network point-to-point
 ipv6 ospf6 authentication keychain OSPF6-CORE
 ipv6 ospf6 bfd
exit
!
interface eth2
 ipv6 ospf6 area 0.0.0.0
 ipv6 ospf6 network point-to-point
 ipv6 ospf6 authentication keychain OSPF6-CORE
 ipv6 ospf6 bfd
exit
!
router ospf6
 ospf6 router-id 10.0.0.1
exit
!
```

What the shape demonstrates:

- OSPFv3 enabled per interface with `ipv6 ospf6 area`
- the loopback passive, and the transit links with no passive line
- point-to-point mode on each two-router link, which the router at the other end shares
- one key chain shared by both links, referenced by name from the authentication trailer
- BFD bound per interface, so every neighbor found on the link gets a BFD peer
- the router-id written as `A.B.C.D`, as OSPFv3 takes it

Sources: ospf6d › OSPF6 interface; ospf6d › OSPF6 point-to-point and point-to-multipoint operation; ospf6d › Example configuration of keychain:; ospfd › Interfaces; ospf6d › Configuration Commands; bfd › OSPF6 BFD Configuration.
