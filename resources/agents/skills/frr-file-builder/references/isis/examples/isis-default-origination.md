# Edge Router Originating A Default

An edge router learns a default route from an upstream, through another protocol, and hands
it to level-2 of the IS-IS domain. The line leaves out `always`, which would originate the
default even if there is no default route in the system. The upstream interface does not run IS-IS, and the session that
learns the default is built by its own pack and left out of this shape.

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
router isis CORE
 net 49.0001.0000.0000.0006.00
 is-type level-2-only
 ! WARNING: domain-password md5 is left out; add it on the router, never in this file
 default-information originate ipv4 level-2
exit
!
```

What the shape demonstrates:

- `default-information originate` with the family and the level the default goes into
- no `always`, which would originate the default even with no default route in the system
- the upstream interface left out of IS-IS: no `ip router isis` on it

Sources: isisd › ISIS Fast-Reroute; isisd › ISIS interface; isisd › ISIS router.
