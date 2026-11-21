# Router On A Shared Segment

Three routers share one Ethernet segment, and this one should be its DR. Adjacencies are
wanted with only the two other routers among those reachable on the segment, so a
neighbor-filter drops OSPF packets from every other source address, which also mitigates OSPF
denial of service.

A reference for shape. Copy the structure, never the router-id, addresses, names or interfaces.

```
ip prefix-list LAN1-ROUTERS seq 5 permit 10.1.0.2/32
ip prefix-list LAN1-ROUTERS seq 10 permit 10.1.0.3/32
!
key chain OSPF-LAN
 key 1
  cryptographic-algorithm hmac-sha-256
  ! WARNING: key 1 of OSPF-LAN has no key-string; set it on the router, never in this file
 exit
exit
!
interface lo
 ip ospf area 0.0.0.0
exit
!
interface eth1
 ip address 10.1.0.1/24
 ip ospf area 0.0.0.0
 no ip ospf passive
 ip ospf authentication key-chain OSPF-LAN
 ! highest priority on the segment: the most eligible router to become DR
 ip ospf priority 200
 ip ospf neighbor-filter LAN1-ROUTERS
exit
!
router ospf
 ospf router-id 10.0.0.4
 passive-interface default
exit
!
```

What the shape demonstrates:

- a shared segment is a multi-access link with no `ip ospf network` line, where the routers
  elect a DR
- `ip ospf priority` set only because the request gives the DR role
- the neighbor-filter prefix-list permits the source address of each router that belongs;
  every other source is denied
- the interface address the request gives comes first in the block, before the OSPF lines

Sources: ospfd › Interfaces; ospf_fundamentals › Link-State LSAs; filter › IP Prefix List.
