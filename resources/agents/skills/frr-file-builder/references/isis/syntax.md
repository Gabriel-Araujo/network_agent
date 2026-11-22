# IS-IS Syntax

How the IS-IS part of a configuration is written. The file header, the block order, the
interface headers and the Shared objects are rules of the skill body; authentication and the
warnings come from `doctrine.md`.

## Shape

One level-2-only router with a loopback and one transit interface to one other router, over
IPv4. Replace every `<...>` with a value from the request, and keep the order, the indentation
and the `!` separator lines:

```
interface lo
 ip router isis <name>
 isis passive
exit
!
interface <transit interface>
 ip router isis <name>
 isis network point-to-point
 ! WARNING: isis password md5 is left out; add it on the router, never in this file
exit
!
router isis <name>
 net <NET>
 is-type level-2-only
 ! WARNING: domain-password md5 is left out; add it on the router, never in this file
exit
!
```

## Rules

The shape generalizes by these rules. Each cites the `.rst` section it rests on.

- IS-IS is enabled per interface: `ip router isis <name>` for IPv4 and `ipv6 router isis
  <name>` for IPv6, each naming the instance of `router isis <name>`. [isisd › ISIS interface; isisd › ISIS router]
- The default VRF has one `router isis <name>`: as of this writing, *isisd* does not support
  multiple IS-IS processes. [isisd › ISIS router]
- `router isis` carries `net` with the NET the request gives, in ISO format, and `is-type` with
  the level the request gives: `level-1`, `level-1-2` or `level-2-only`. [isisd › ISIS router; isisd › ISIS region]
- Every IS-IS interface the request does not name as transit carries `isis passive`, the
  loopback included. [isisd › ISIS interface]
- A link with exactly two routers gets `isis network point-to-point`; a shared segment keeps
  the broadcast default and gets no `isis network` line. [isisd › ISIS interface]
- On a level-1-2 router, an interface whose neighbors run one level carries `isis circuit-type
  level-1` or `isis circuit-type level-2-only`, the keywords of the command's description and
  of the isisd examples. A level-1 or level-2-only router carries no `isis circuit-type` line.
  [isisd › ISIS interface; isisd › ISIS region; isisd › ISIS Configuration Examples]
- `metric-style` is written only as `metric-style transition`, while the domain moves between
  styles: `wide` is the FRR default. [isisd › ISIS router]
- A per-level command (`isis metric`, `isis priority`, `isis hello-interval`,
  `lsp-gen-interval`) takes `level-1` or `level-2` when the value is for one level; without
  it, the value is set globally. [isisd › ISIS interface; isisd › ISIS Timer]
- `isis metric`, `isis priority`, `isis hello-interval`, `isis hello-multiplier` and the
  timers of `router isis` are written only with values the request gives. [isisd › ISIS interface; isisd › ISIS Timer]
- BFD is bound in the interface block with `isis bfd`. `isis bfd profile <profile>` goes right
  below it, only with a profile the request names. [bfd › IS-IS BFD Configuration; isisd › ISIS interface]
- `default-information originate` takes the family and the level, then `always`, then
  `metric` or `route-map`, one of the two; each only when used:
  `default-information originate ipv4 level-2`.
  [isisd › ISIS Fast-Reroute]
- IS-IS in a VRF is `router isis <name> vrf <vrf>`. Each interface the request puts in that
  VRF is opened as `interface <name> vrf <vrf>` and enabled with `ip router isis <name>`,
  naming that instance. [isisd › ISIS router; isisd › ISIS Vrf Configuration Examples]
