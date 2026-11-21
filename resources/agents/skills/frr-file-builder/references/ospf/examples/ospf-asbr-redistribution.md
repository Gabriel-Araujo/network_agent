# ASBR Handing External Routes To OSPF

An edge router learns a default route and a customer block from another protocol, here BGP,
and hands both to the OSPF domain. The default goes in only while the routing table of this
router has one. The customer block goes in through a route-map that names it, never the whole
table of the other protocol. The BGP session that learns them is
built by the `bgp` pack and is left out of this shape.

A reference for shape. Copy the structure, never the router-id, prefixes, names or interfaces.

```
ip prefix-list CUSTOMER-BLOCK seq 5 permit 198.51.100.0/24
!
route-map BGP-TO-OSPF permit 10
 match ip address prefix-list CUSTOMER-BLOCK
 ! external cost 20 for the customer block
 set metric 20
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
 ! WARNING: no authentication on the adjacencies of eth1
exit
!
router ospf
 ospf router-id 10.0.0.5
 passive-interface default
 redistribute bgp route-map BGP-TO-OSPF
 default-information originate
exit
!
```

What the shape demonstrates:

- a router becomes an ASBR by redistributing; nothing else declares the role
- the redistribution policy is a route-map whose only entry matches a prefix-list that permits
  the customer block and, by its default deny, nothing else
- `default-information originate` without `always`, so the default follows the routing table
- the routes enter as Type-5 LSAs, flooded to every area that accepts external routes
- the no-authentication warning, since the request names no key chain for `eth1`

Sources: ospfd › Redistribution; ospfd › OSPF route-map; ospfd › Interfaces; ospf_fundamentals › External LSAs; filter › IP Prefix List.
