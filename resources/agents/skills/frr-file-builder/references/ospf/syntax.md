# OSPF Syntax

How the OSPF part of a configuration is written. The file header, the block order, the
interface headers and the Shared objects are rules of the skill body; authentication and the
warnings come from `doctrine.md`.

## Shape

One router with a loopback and one transit interface to one other router, in one area, with no
authentication given by the request. Replace every `<...>` with a value from the request, and
keep the order, the indentation and the `!` separator lines:

```
interface lo
 ip ospf area <area>
exit
!
interface <transit interface>
 ip ospf area <area>
 ip ospf network point-to-point
 no ip ospf passive
 ! WARNING: no authentication on the adjacencies of <transit interface>
exit
!
router ospf
 ospf router-id <router-id>
 passive-interface default
exit
!
```

## Rules

The shape generalizes by these rules. Each cites the `.rst` section it rests on.

- OSPF is enabled per interface with `ip ospf area <area>` in the interface block, never with
  `network <prefix> area <area>` under `router ospf`. The two are not supported together, and
  with any `ip ospf area` present every `network` command fails. [ospfd › Interfaces; ospfd › Routers]
- `router ospf` carries `passive-interface default`, its only `passive-interface` line. Each
  transit interface gets `no ip ospf passive` in its own block, which takes precedence over the
  default. [ospfd › Routers; ospfd › Interfaces]
- An area is written dotted (`0.0.0.0`, `0.0.0.10`) everywhere it appears: `ip ospf area`,
  `area <area> range`, `area <area> stub`, `area <area> nssa`. The commands accept the integer
  form too; this pack writes only the dotted one. [ospfd › Areas; ospfd › Interfaces]
- Per-interface lines (`ip ospf ...`) go in the interface block; per-area lines
  (`area <area> ...`) and router-wide lines go in `router ospf`. [ospfd › Interfaces; ospfd › Areas; ospfd › Routers]
- A link with exactly two routers gets `ip ospf network point-to-point`; a shared segment keeps
  the default type and gets no `ip ospf network` line. [ospfd › Interfaces; ospf_fundamentals › Link-State LSAs]
- A point-to-point interface whose address is a /32 is unnumbered: put a warning above its
  `ip ospf network` line that `net.ipv4.conf.<interface>.rp_filter` must be `0`, or the kernel
  drops the OSPF multicast. [ospfd › Interfaces]
- `ip ospf hello-interval` and `ip ospf dead-interval` are written only with values the
  request gives, and are equal on every router of the link. [ospfd › Interfaces]
- BFD is bound in the interface block with `ip ospf bfd`. `ip ospf bfd profile <profile>`
  goes right below it, only with a profile the `bfd` block defines: alone, it does nothing.
  [bfd › OSPF BFD Configuration]
- A `redistribute` line takes its options in this order: `redistribute <source> metric-type
  <1|2> metric <cost> route-map <name>`, each option only when used. [ospfd › Redistribution]
- OSPF in a VRF is `router ospf vrf <vrf>`. Each interface the request puts in that VRF is
  opened as `interface <name> vrf <vrf>` and enabled with its own `ip ospf area`.
  [ospfd › Routers; isisd › ISIS Vrf Configuration Examples]
