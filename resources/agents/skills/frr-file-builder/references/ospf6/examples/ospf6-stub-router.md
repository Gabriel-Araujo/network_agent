# Maintenance As A Stub Router

A core router is drained before work on it, and the traffic through it has another path. As a
stub router its router-LSA clears the R and V6 bits: on `frr:10.5.0` the other routers keep
their adjacencies with it and reach its own prefixes, but no route transits it, even one with
no other path. The state holds for an indefinite period: the operator removes the line when the
router returns to service.

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
exit
!
interface eth2
 ipv6 ospf6 area 0.0.0.0
 ipv6 ospf6 network point-to-point
 ipv6 ospf6 authentication keychain OSPF6-CORE
exit
!
router ospf6
 ospf6 router-id 10.0.0.7
 ! drained for maintenance until this line is removed
 stub-router administrative
exit
!
```

What the shape demonstrates:

- the drain is one router-wide line: the interfaces and their authentication lines are left
  unchanged, and the adjacencies stay Full
- a `!` comment saying when the line comes out

Sources: ospf6d › Configuration Commands; ospf6d › OSPF6 interface; ospf6d › Example configuration of keychain:; ospfd › Interfaces.
