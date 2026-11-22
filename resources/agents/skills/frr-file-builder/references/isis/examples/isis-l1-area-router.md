# Level-1 Router Inside An Area

A router acts as a station router only: it routes inside its own area, at level-1, over one
link to the level-1-2 router of the area. A LAN of hosts hangs off it, on a passive interface
whose prefix IS-IS advertises. Traffic for other areas follows a default route: a router that
receives an LSP with the attached bit set creates a default route to that neighbor.

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
 ! WARNING: isis password md5 is left out; add it on the router, never in this file
exit
!
interface eth2
 ip router isis CORE
 isis passive
exit
!
router isis CORE
 net 49.0002.0000.0000.0003.00
 is-type level-1
 ! WARNING: area-password md5 is left out; add it on the router, never in this file
exit
!
```

What the shape demonstrates:

- `is-type level-1` for a station router, with no `isis circuit-type` line
- the host LAN as a passive interface, its prefix advertised
- only the `area-password` warning, as the router runs level-1 alone
- no `default-information originate`: a router that receives an LSP with the attached bit set
  creates a default route to the neighbor
- no `attached-bit receive ignore`

Sources: isisd › ISIS region; isisd › ISIS interface; isisd › ISIS router; isisd › ISIS Timer.
