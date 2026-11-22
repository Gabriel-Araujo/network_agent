# Router On A Shared Segment

Three level-2 routers share one Ethernet segment, a broadcast circuit. The segment keeps the
broadcast default. The request gives the address of the interface and the level-2 priority
this router carries into the designated router election.

A reference for shape. Copy the structure, never the NET, addresses, names or interfaces.

```
interface lo
 ip router isis CORE
 isis passive
exit
!
interface eth1
 ip address 10.1.0.1/24
 ip router isis CORE
 ! level-2 priority the request gives, for the designated router election
 isis priority 100 level-2
 ! WARNING: isis password md5 is left out; add it on the router, never in this file
exit
!
router isis CORE
 net 49.0001.0000.0000.0004.00
 is-type level-2-only
 ! WARNING: domain-password md5 is left out; add it on the router, never in this file
exit
!
```

What the shape demonstrates:

- a shared segment keeps the broadcast default, with no `isis network` line
- `isis priority` set only with the value and the level the request gives
- the interface address the request gives comes first in the block, as the skill body orders
  it

Sources: isisd › ISIS interface; isisd › ISIS router; isisd › ISIS region; fabricd › Sample configuration.
