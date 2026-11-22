# IS-IS For IPv4 And IPv6

A level-2 core router carries IPv4 and IPv6 in IS-IS over the same point-to-point link, and
its passive loopback in both families. Each family is enabled on each interface with its own line.
BFD watches the adjacency; with IPv6 enabled on the circuit, its session runs over the IPv6
link-local address.

A reference for shape. Copy the structure, never the NET, names or interfaces.

```
interface lo
 ip router isis CORE
 ipv6 router isis CORE
 isis passive
exit
!
interface eth1
 ip router isis CORE
 ipv6 router isis CORE
 isis network point-to-point
 isis bfd
 ! WARNING: isis password md5 is left out; add it on the router, never in this file
exit
!
router isis CORE
 net 49.0001.0000.0000.0005.00
 is-type level-2-only
 ! WARNING: domain-password md5 is left out; add it on the router, never in this file
exit
!
```

What the shape demonstrates:

- `ip router isis` and `ipv6 router isis` on each interface that carries both families,
  naming the same instance
- one set of interface lines (`isis passive`, `isis network`, `isis bfd`) for both families
- BFD bound once per interface, whatever the family of its session

Sources: isisd › ISIS interface; isisd › ISIS router; bfd › IS-IS BFD Configuration.
