# ABR Of A Not-So-Stubby Area

A branch area imports routes of its own: a router inside it redistributes a partner network,
which a stub area would refuse. As an NSSA the area takes those imports as Type-7 LSAs, and
this ABR translates them into Type-5 for the backbone, as one summary instead of every
partner prefix. The area receives no Type-5 from the rest of the domain, and this ABR
originates a Type-7 default into it with `nssa default-information-originate`.

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
 ip ospf area 0.0.0.20
 ip ospf network point-to-point
 no ip ospf passive
 ip ospf authentication key-chain OSPF-AUTH
exit
!
router ospf
 ospf router-id 10.0.0.3
 passive-interface default
 area 0.0.0.20 nssa
 area 0.0.0.20 nssa default-information-originate
 ! the partner routes leave the area as one Type-5
 area 0.0.0.20 nssa range 172.16.0.0/16
exit
!
```

What the shape demonstrates:

- the NSSA declared once in `router ospf`, with the area in dotted form
- the Type-7 default originated by the ABR into the NSSA
- `nssa range` summarizing the external routes of the area at the ABR, where the Type-7 to
  Type-5 translation happens; `area <area> range` would only summarize intra-area routes
- the redistribution itself lives on the router inside the area, not here

Sources: ospfd › Areas; ospfd › Redistribution; ospfd › Interfaces; ospfd › Routers.
