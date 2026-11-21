# ABR Between The Backbone And A Totally Stubby Area

A branch area has a single way out, through this router. The area gets no external and no
inter-area routes, only a default, and the backbone sees the whole branch as one summary
instead of every link inside it.

A reference for shape. Copy the structure, never the router-id, areas, ranges or interfaces.

```
key chain OSPF-AUTH
 key 1
  cryptographic-algorithm hmac-sha-256
  ! WARNING: key 1 of OSPF-AUTH has no key-string; set it on the router, never in this file
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
 ip ospf authentication key-chain OSPF-AUTH
exit
!
interface eth2
 ip ospf area 0.0.0.10
 no ip ospf passive
 ip ospf authentication key-chain OSPF-AUTH
exit
!
router ospf
 ospf router-id 10.0.0.2
 passive-interface default
 ! totally stubby: no inter-area summaries enter area 0.0.0.10, only the default
 area 0.0.0.10 stub no-summary
 ! one Type-3 into the backbone for the whole branch
 area 0.0.0.10 range 10.10.0.0/16
exit
!
```

What the shape demonstrates:

- an ABR is a router with interfaces in area `0.0.0.0` and in another area; nothing else
  declares the role
- `eth2` is a shared segment of the branch, so it keeps the default network type
- the area type and the range written once, in `router ospf`, with the area in dotted form
- the range summarizes intra-area routes only; external routes are not summarized here

Sources: ospfd › Areas; ospfd › Interfaces; ospfd › Routers; ospf_fundamentals › Areas.
