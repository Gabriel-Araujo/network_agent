# Level-1-2 Router Between Its Area And Level-2

A router is both a station router and an area router: it forms level-1 adjacencies with the
routers of its own area and level-2 adjacencies toward the rest of the domain. Each link has
neighbors of one level only, so each interface is limited to that level. The attached bit in
the LSPs it sends to the level-1 routers lets them create a default route toward it.

A reference for shape. Copy the structure, never the NET, names or interfaces.

```
interface lo
 ip router isis CORE
 isis passive
exit
!
interface eth1
 ip router isis CORE
 isis network point-to-point
 isis circuit-type level-2-only
 ! WARNING: isis password md5 is left out; add it on the router, never in this file
exit
!
interface eth2
 ip router isis CORE
 isis network point-to-point
 isis circuit-type level-1
 ! WARNING: isis password md5 is left out; add it on the router, never in this file
exit
!
router isis CORE
 net 49.0002.0000.0000.0002.00
 is-type level-1-2
 ! WARNING: area-password md5 is left out; add it on the router, never in this file
 ! WARNING: domain-password md5 is left out; add it on the router, never in this file
 attached-bit send
exit
!
```

What the shape demonstrates:

- `is-type level-1-2` for a router that routes inside its area and between areas
- `isis circuit-type level-2-only` toward the domain and `isis circuit-type level-1` toward
  the area, the keywords of the command's description and of the isisd examples
- both password warnings in `router isis`, one per level the router runs
- `attached-bit send`, so the level-1 routers of the area get a default route toward it

Sources: isisd › ISIS region; isisd › ISIS interface; isisd › ISIS router; isisd › ISIS Timer; isisd › ISIS Configuration Examples.
