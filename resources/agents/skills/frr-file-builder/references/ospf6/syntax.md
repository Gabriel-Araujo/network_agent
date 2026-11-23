# OSPFv3 Syntax

How the OSPFv3 part of a configuration is written. The file header, the block order, the
interface headers and the Shared objects are rules of the skill body; authentication and the
warnings come from `doctrine.md`.

## Shape

One router with a loopback and one transit interface to one other router, in one area, with no
authentication given by the request. Replace every `<...>` with a value from the request, and
keep the order, the indentation and the `!` separator lines:

```
interface lo
 ipv6 ospf6 area <area>
 ipv6 ospf6 passive
exit
!
interface <transit interface>
 ipv6 ospf6 area <area>
 ipv6 ospf6 network point-to-point
 ! WARNING: no authentication on the adjacencies of <transit interface>
exit
!
router ospf6
 ospf6 router-id <router-id>
exit
!
```

## Rules

The shape generalizes by these rules. Each cites the `.rst` section it rests on.

- OSPFv3 is enabled per interface with `ipv6 ospf6 area <area>` in the interface block, which
  adds the interface to that area. [ospf6d › OSPF6 interface]
- Every OSPFv3 interface the request does not name as transit carries `ipv6 ospf6 passive`, the
  loopback included. A transit interface has no passive line. [ospf6d › OSPF6 interface]
- An area is written dotted (`0.0.0.0`, `0.0.0.10`) everywhere it appears: `ipv6 ospf6 area`,
  `area <area> range`, `area <area> nssa`. The commands accept the integer form too; this pack
  writes only the dotted one. [ospf6d › OSPF6 interface; ospf6d › OSPF6 area]
- Per-interface lines (`ipv6 ospf6 ...`) go in the interface block; per-area lines
  (`area <area> ...`) and router-wide lines go in `router ospf6`.
  [ospf6d › OSPF6 interface; ospf6d › OSPF6 area; ospf6d › Configuration Commands; ospf6d › Sample configuration]
- A link with exactly two routers gets `ipv6 ospf6 network point-to-point`, on the routers at
  both ends: routers must be configured for the same mode to form adjacencies. A shared
  segment keeps the default broadcast mode and gets no `ipv6 ospf6 network` line.
  [ospf6d › OSPF6 interface; ospf6d › OSPF6 point-to-point and point-to-multipoint operation]
- No `ipv6 address` is written for OSPFv3's sake: OSPFv3 relies exclusively on link-local
  addresses to do its signaling. An address line comes only from the request, by the rule of
  the skill body. [ospf6d › OSPF6 point-to-point and point-to-multipoint operation]
- `ipv6 ospf6 hello-interval` and `ipv6 ospf6 dead-interval` are written only with values the
  request gives; the hello interval must be the same on all routers of the link.
  [ospf6d › OSPF6 interface; ospf6d › OSPF6 point-to-point and point-to-multipoint operation]
- Authentication is `ipv6 ospf6 authentication keychain <key chain>` in the interface block,
  naming a key chain the file defines. The keychain form and the manual key form are mutually
  exclusive, and this pack writes only the keychain form.
  [ospf6d › CLI Configuration:; ospf6d › Example configuration of keychain:]
- BFD is bound in the interface block with `ipv6 ospf6 bfd`, or `ipv6 ospf6 bfd profile
  <profile>` only with a profile the `bfd` block defines. [bfd › OSPF6 BFD Configuration]
- A `redistribute` line takes its options in this order: `redistribute <source> metric-type
  <1|2> metric <cost> route-map <name>`, each option only when used.
  [ospf6d › Redistribute routes to OSPF6]
- `summary-address <prefix>` goes in `router ospf6` of an ASBR only: external route
  summarization must be configured on the ASBR. [ospf6d › ASBR Summarisation Support in OSPFv3]
- OSPFv3 in a VRF is `router ospf6 vrf <vrf>`. Each interface the request puts in that VRF is
  opened as `interface <name> vrf <vrf>` and enabled with its own `ipv6 ospf6 area`.
  [ospf6d › Configuration Commands; isisd › ISIS Vrf Configuration Examples]
