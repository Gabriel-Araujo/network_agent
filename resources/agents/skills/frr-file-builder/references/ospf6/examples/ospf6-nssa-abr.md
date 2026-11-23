# ABR Of A Not-So-Stubby Area

A branch area the request makes an NSSA imports routes of its own: a router inside it
redistributes a partner network. The NSSA carries those imports as Type-7 LSAs, and this ABR
translates them into Type-5 for the backbone, as one summary instead of every partner
prefix. This ABR also originates a Type-7 default into the area with
`nssa default-information-originate`.

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
 ipv6 ospf6 area 0.0.0.20
 ipv6 ospf6 network point-to-point
 ipv6 ospf6 authentication keychain OSPF6-AUTH
exit
!
router ospf6
 ospf6 router-id 10.0.0.3
 ! NSSA: external routes of the area enter as Type-7; a Type-7 default goes in
 area 0.0.0.20 nssa default-information-originate
 ! one Type-7 summary for the partner routes, translated to one Type-5 for the backbone
 area 0.0.0.20 nssa range 2001:db8:20::/48
exit
!
```

What the shape demonstrates:

- the area type written in `router ospf6` of the ABR, with the area in dotted form
- the Type-7 default into the NSSA, with `default-information-originate` on the `nssa` line
- `nssa range` summarizing the external routes of the area, which only the NSSA ABR does
- `no-summary` left out: the area still receives the inter-area summaries

Sources: ospf6d › OSPF6 area; ospf6d › OSPF6 interface; ospf6d › Configuration Commands; ospf6d › Example configuration of keychain:; ospfd › Interfaces.
