# ASBR Handing External Routes To OSPFv3

An edge router learns a default route and a customer block from another protocol, here BGP,
and hands both to the OSPFv3 domain. The customer routes go in through a route-map that names
the block, never the whole table of the other protocol, and the domain sees them as one
external LSA. The default goes in only while this router has one. The BGP session that learns
them is built by the `bgp` pack and is left out of this shape.

A reference for shape. Copy the structure, never the router-id, prefixes, names or interfaces.

```
ipv6 prefix-list CUSTOMER-BLOCK seq 5 permit 2001:db8:100::/40 le 48
!
route-map BGP-TO-OSPF6 permit 10
 match ipv6 address prefix-list CUSTOMER-BLOCK
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
 ! WARNING: no authentication on the adjacencies of eth1
exit
!
router ospf6
 ospf6 router-id 10.0.0.4
 redistribute bgp route-map BGP-TO-OSPF6
 ! one external LSA for every customer prefix inside the block
 summary-address 2001:db8:100::/40
 default-information originate
exit
!
```

What the shape demonstrates:

- a router becomes an ASBR by originating external LSAs, here by redistributing; `ospf6d.rst`
  documents no command that declares the role
- the redistribution policy is a route-map whose only entry matches a prefix-list that permits
  the customer block; the prefix-list denies the rest by default, and on `frr:10.5.0` a route
  that matches no entry of the route-map is not redistributed
- `summary-address` on the ASBR, advertised only while one route inside the range exists, and
  installed locally with Null0 as next-hop
- `default-information originate` without `always`, so the default is injected only while this
  router has one
- the no-authentication warning, since the request names no key chain for `eth1`

Sources: ospf6d › Redistribute routes to OSPF6; ospf6d › ASBR Summarisation Support in OSPFv3; ospf6d › OSPF6 interface; ospf6d › Sample configuration; ospf_fundamentals › External LSAs; filter › IP Prefix List.
