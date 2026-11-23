# Router On A Shared Segment

Three routers share one Ethernet segment, and the request gives this one its router priority for
the DR election. The segment keeps the default broadcast mode, where a DR and a BDR are elected,
and the authentication trailer gives its adjacencies IPv6 source address protection.

A reference for shape. Copy the structure, never the router-id, priority, names or interfaces.

```
key chain OSPF6-LAN
 key 1
  cryptographic-algorithm hmac-sha-256
  ! WARNING: key 1 of OSPF6-LAN has no key-string; set it on the router, never in this file
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
 ipv6 ospf6 authentication keychain OSPF6-LAN
 ! router priority the request gives for the DR election; the default is 1
 ipv6 ospf6 priority 200
exit
!
router ospf6
 ospf6 router-id 10.0.0.5
exit
!
```

What the shape demonstrates:

- a shared segment has no `ipv6 ospf6 network` line, so it stays in broadcast mode
- `ipv6 ospf6 priority` set only because the request gives the value
- no `ipv6 address` line: OSPFv3 signals over link-local addresses, and the request gives none

Sources: ospf6d › OSPF6 interface; ospf6d › OSPF6 point-to-point and point-to-multipoint operation; ospf6d › Authentication trailer support:; ospf_fundamentals › The Hello Protocol; ospf6d › Example configuration of keychain:; ospfd › Interfaces.
