# EVPN Syntax

How the EVPN part of a configuration is written. The file header, the block order, the `vrf`
and `interface` headers and the Shared objects are rules of the skill body; the `router bgp`
blocks, the sessions and every `neighbor` line follow the `bgp` pack; the warnings come from
`doctrine.md`.

## Shape

One VTEP with one iBGP session that carries EVPN, its VNIs learned from the kernel, and no
authentication given by the request. Replace every `<...>` with a value from the request, and
keep the order, the indentation and the `!` separator lines:

```
router bgp <local AS>
 bgp router-id <router-id>
 neighbor <peer address> remote-as <local AS>
 neighbor <peer address> description <who the peer is and why the session exists>
 ! WARNING: no authentication on the session with <peer address>
 !
 ! WARNING: FRR does not create the VXLAN devices, bridges and SVIs of these VNIs; zebra learns them from the kernel
 address-family l2vpn evpn
  neighbor <peer address> activate
  advertise-all-vni
 exit-address-family
exit
!
```

## Rules

The shape generalizes by these rules. Each cites the `.rst` section it rests on.

- Every EVPN line of a BGP instance goes in its one `address-family l2vpn evpn` block, which
  closes with `exit-address-family`. [bgp › Enabling EVPN; evpn › Sample Configuration]
- `advertise-all-vni` goes in `address-family l2vpn evpn` of the BGP instance of the underlay
  VRF, the default VRF unless the request names another, and in no other instance: it can only
  be configured in a single VRF. [bgp › Enabling EVPN]
- Each session that carries EVPN has `neighbor <peer> activate` in `address-family l2vpn evpn`.
  The session, its `description`, its warnings and every other `neighbor` line follow the `bgp`
  pack. [evpn › Sample Configuration; bgp › IPv6 Support]
- A VNI has no line of its own unless the request gives it a setting: zebra learns the VNIs from
  the kernel. A VNI with a setting opens `vni <VNI>` in `address-family l2vpn evpn` of the
  underlay instance, holds the setting, and closes with `exit-vni`.
  [evpn › Sample Configuration; bgp › EVPN MAC-VRF / L2VNI Route Targets]
- An L3VNI is `vni <VNI>` in the `vrf <vrf>` block of its IP-VRF.
  [evpn › Sample Configuration]
- Type-5 export is `advertise ipv4 unicast` or `advertise ipv6 unicast`, one line per address
  family, in `address-family l2vpn evpn` of the VRF's own BGP instance,
  `router bgp <local AS> vrf <vrf>`. [evpn › Sample Configuration; bgp › EVPN Overlay Index Gateway IP]
- A route target is `route-target <import|export|both> <route target>`, one route target per
  line. An IP-VRF's go in `address-family l2vpn evpn` of the VRF's BGP instance; an L2VNI's go
  in its `vni` block of the underlay instance, the only instance where FRR accepts the L2VNI
  `route-target` command.
  [bgp › EVPN IP-VRF Route Targets; bgp › EVPN MAC-VRF / L2VNI Route Targets]
- A route target is `<AS>:<N>` or `<A.B.C.D>:<N>`. With a 4-byte AS or an IPv4 address in front,
  `<N>` is a 2-byte local administrator, and a VNI such as 70000 does not fit in it.
  [evpn › Wildcard Route Targets]
- A wildcard route target `*:<N>` goes only in `route-target import` of an IP-VRF: wildcards
  apply only to import, and an L2VNI accepts none.
  [bgp › EVPN IP-VRF Route Targets; bgp › EVPN MAC-VRF / L2VNI Route Targets]
- `flooding disable` handles BUM traffic for every VNI, or for one VNI from inside its `vni`
  block. On `frr:10.5.0`, the daemon takes it in `address-family l2vpn evpn` next to
  `advertise-all-vni`, or in the `vni` block of one VNI, and `show running-config` keeps it
  there. [bgp › EVPN BUM Handling; bgp › EVPN MAC-VRF / L2VNI Route Targets]
- `advertise-svi-ip` goes in `address-family l2vpn evpn` of the underlay instance.
  [bgp › EVPN advertise-svi-ip; evpn › Sample Configuration]
- An Ethernet Segment is configured in the interface block of its bond, with either a local
  discriminator and a system MAC (`evpn mh es-id <discriminator>` and `evpn mh es-sys-mac
  <system MAC>`) or a full 10-byte Type-0 ESI alone (`evpn mh es-id <ESI>`). `evpn mh es-df-pref
  <preference>` goes in the same block when the request gives one.
  [bgp › Ethernet Segments; bgp › BUM handling]
- `evpn mh uplink` goes in the interface block of each underlay interface.
  [bgp › Uplink/Core tracking]
- `evpn mh startup-delay <seconds>` stands outside any block, right after the header and before
  every object: on `frr:10.5.0`, `show running-config` writes it there, and `show evpn` lists
  the delay in seconds. [bgp › Startup delay]
