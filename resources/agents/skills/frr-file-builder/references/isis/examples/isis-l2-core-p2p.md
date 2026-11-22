# Level-2 Core Router On Point-To-Point Links

A core router routes only between areas, at level-2, over two links that each connect it to
exactly one other router. Those two links are transit; the loopback, which iBGP and
management use, is a passive interface whose prefix IS-IS advertises. BFD watches the
adjacencies, faster than the IS-IS hello timers, and per-prefix local LFA protects each link.
After startup the overload bit keeps transit traffic off the router for five minutes.

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
 isis bfd
 isis fast-reroute lfa level-2
 ! WARNING: isis password md5 is left out; add it on the router, never in this file
exit
!
interface eth2
 ip router isis CORE
 isis network point-to-point
 isis bfd
 isis fast-reroute lfa level-2
 ! WARNING: isis password md5 is left out; add it on the router, never in this file
exit
!
router isis CORE
 net 49.0001.0000.0000.0001.00
 is-type level-2-only
 ! WARNING: domain-password md5 is left out; add it on the router, never in this file
 ! 5 min with the overload bit set after startup: no transit traffic through this router
 set-overload-bit on-startup 300
exit
!
```

What the shape demonstrates:

- IS-IS enabled per interface with `ip router isis`, naming the one instance
- the loopback passive, the two transit links point-to-point
- a level-2-only router: no `isis circuit-type` line, and only the `domain-password` warning
- the password warnings in place of every password line
- BFD bound per interface, and local LFA per interface at the level the router runs
- `set-overload-bit on-startup` on a router that carries transit traffic

Sources: isisd › ISIS router; isisd › ISIS region; isisd › ISIS interface; bfd › IS-IS BFD Configuration.
