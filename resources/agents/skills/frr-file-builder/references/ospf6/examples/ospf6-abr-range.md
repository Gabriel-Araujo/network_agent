# ABR Between The Backbone And A Branch Area

A branch area reaches the backbone through this router. The backbone sees the whole branch as
one Inter-Area-Prefix LSA for its range instead of one per subnet inside it.

A reference for shape. Copy the structure, never the router-id, areas, ranges or interfaces.

```
key chain OSPF6-AUTH
 key 1
  cryptographic-algorithm hmac-sha-256
  ! WARNING: key 1 of OSPF6-AUTH has no key-string; set it on the router, never in this file
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
 ipv6 ospf6 authentication keychain OSPF6-AUTH
exit
!
interface eth2
 ipv6 ospf6 area 0.0.0.10
 ipv6 ospf6 authentication keychain OSPF6-AUTH
exit
!
router ospf6
 ospf6 router-id 10.0.0.2
 ! one Inter-Area-Prefix LSA into the backbone for the whole branch
 area 0.0.0.10 range 2001:db8:10::/48
exit
!
```

What the shape demonstrates:

- an ABR is a router with interfaces in area `0.0.0.0` and in another area; `ospf6d.rst`
  documents no command that declares the role
- `eth2` is a shared segment of the branch, so it keeps the default broadcast mode
- the range written once, in `router ospf6`, with the area in dotted form
- the range summarizes the internal subnets of the area, with the highest metric among them

Sources: ospf6d › OSPF6 area; ospf6d › OSPF6 interface; ospf6d › OSPF6 point-to-point and point-to-multipoint operation; ospf6d › Configuration Commands; ospf_fundamentals › Areas; ospf6d › Example configuration of keychain:; ospfd › Interfaces.
