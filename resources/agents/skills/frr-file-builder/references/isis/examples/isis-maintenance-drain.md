# Maintenance: Draining A Router

A level-2 core router is about to be worked on, and its traffic should move elsewhere first.
It advertises a high metric on all its interfaces, which gracefully shifts traffic off it.
`set-overload-bit` is the stronger choice, for when no transit traffic may cross the router at
all.

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
 net 49.0001.0000.0000.0007.00
 is-type level-2-only
 ! WARNING: domain-password md5 is left out; add it on the router, never in this file
 ! drain: high metric on all interfaces, 16777215 with wide
 advertise-high-metrics
exit
!
```

What the shape demonstrates:

- the drain is one router-wide line, with a `!` comment stating its effect
- the interfaces stay as they were: the high metric is advertised on all of them
- `advertise-high-metrics` rather than `set-overload-bit`: a graceful shift, where the
  overload bit avoids any transit traffic

Sources: isisd › ISIS router; isisd › ISIS interface.
