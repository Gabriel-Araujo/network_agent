# EVPN

BGP EVPN over VXLAN: EVPN enabled in the underlay BGP instance, L2VNIs and L3VNIs, route
targets of a MAC-VRF and of an IP-VRF, Type-5 export from an IP-VRF, BUM flooding, SVI
addresses advertised as Type-2 routes, route reflectors that carry EVPN, EVPN multihoming with
Ethernet Segments.

## Scope

EVPN is an extension of BGP. This pack writes the EVPN lines: `address-family l2vpn evpn` and
what goes in it, the `vni` of a `vrf` block, and the `evpn mh` lines. The `router bgp` blocks,
the sessions and every `neighbor` line are the `bgp` pack's, also inside
`address-family l2vpn evpn`. The `vrf` and `interface` headers are the skill body's.

Out of this pack:

- eBGP sessions that carry EVPN. On `frr:10.5.0` with `frr defaults traditional`, such a
  session exchanges no EVPN route until it has an inbound and an outbound filter in
  `address-family l2vpn evpn`, and the corpus documents no EVPN match for that filter.
- The Linux dataplane (VXLAN devices, bridges, SVIs, VRF devices), which FRR does not manage.
- EVPN over MPLS; Downstream VNI; the VRF network namespace backend.
- `advertise-pip`, MAC-VRF Site-of-Origin and the MLAG anycast VTEPs they serve.
- The gateway IP overlay index: `advertise <afi> unicast gateway-ip`, `set evpn gateway-ip`,
  `enable-resolve-overlay-index`.
- `advertise-default-gw` and `auto-route-target`, which the corpus names without a `clicmd`.
- The EVPN multihoming knobs for interop and fast failover (`disable-ead-evi-rx`,
  `disable-ead-evi-tx`, `use-es-l3nhg`, `evpn mh redirect-off`), the proxy holdtimes, and the
  EAD-per-ES fragmentation and route targets.

Read with: `bgp`, always: every `address-family l2vpn evpn` block of this pack lives in a
`router bgp` block whose sessions that pack writes.

## Daemons

| Daemon | When |
|--------|------|
| `zebra` | Always: it learns the VNIs from the kernel, and holds the `vni` of a `vrf` and the `evpn mh` lines |
| `bgpd` | Always: EVPN is configured inside `router bgp` |

The underlay protocol runs the daemon its own pack names. Anything beyond this table needs an
explicit justification.

## Vocabulary

Use these terms with exactly these meanings, in config comments and in what is said about the
configuration.

| Term | Means | Not |
|------|-------|-----|
| VTEP | A router that encapsulates the overlay traffic in VXLAN toward the other VTEPs and decapsulates what they send | leaf, NVE |
| VTEP address | The local tunnel address of the VXLAN devices of a VTEP, which every other VTEP must reach | router-id, loopback |
| underlay | The network that carries the VXLAN traffic between VTEPs, and the one VRF whose BGP instance has `advertise-all-vni` | fabric, transport |
| overlay | The virtual networks EVPN carries over the underlay | tenant network |
| VNI | The VXLAN Network Identifier of one virtual network | VLAN, segment ID |
| L2VNI | A VNI tied to a MAC-VRF: a bridging table of MACs and ARP/ND entries | L2 segment |
| L3VNI | A VNI tied to an IP-VRF: a routing table. A VNI is an L2VNI until `vni` in a `vrf` block makes it that VRF's L3VNI | transit VNI |
| route target | The extended community whose value alone decides which IP-VRFs and MAC-VRFs import a route | route distinguisher, label |
| Type-2 route | The MAC/IP advertisement of one host or SVI | host route |
| Type-5 route | The IP prefix route an IP-VRF exports to EVPN | prefix advertisement |
| SVI | A Layer-3 interface bound to a bridging domain | VLAN interface |
| Ethernet Segment | The bonded links of one server attached to two or more VTEPs | MLAG, port-channel |
| DF | The designated forwarder: the one VTEP of an Ethernet Segment that forwards BUM traffic from the overlay to it | master |
| BUM traffic | Broadcast, unknown unicast and multicast traffic | flood traffic |

## Required Facts

An identity fact is never guessed; an operational preference may take a default. A fact the
request, or a file already read, gives is not asked.

### Always Ask

Each of these blocks the build while missing. The `bgp` pack asks what each session needs.

| Missing | Question | Why |
|---------|----------|-----|
| EVPN sessions | Which BGP sessions carry EVPN routes? | Each one is activated in `address-family l2vpn evpn`; the address family is off by default, and a session not activated in it carries no EVPN route |
| Role | Is this router a VTEP with VNIs of its own, or does it only reflect EVPN routes between VTEPs? | A VTEP carries `advertise-all-vni`; a route reflector does not |

### Ask When the Request Implies It

| Missing | Question |
|---------|----------|
| VTEP reachability | This router is a VTEP: how do the other VTEPs reach its VTEP address, advertised in BGP, by static routes, or through an IGP? |
| VTEP address family | Is the VTEP address of this router IPv4? An Ethernet Segment needs an IPv4 VTEP address |
| L3VNI | Which VNI is the L3VNI of IP-VRF `<vrf>`? |
| Type-5 export | Which address families of IP-VRF `<vrf>` are exported to EVPN as Type-5 routes, and which routes does its BGP instance hold? |
| Route targets | Which import and export route targets does `<VNI or IP-VRF>` use in place of the automatic ones? |
| BUM flooding | Is BUM traffic flooded to every VTEP, or is flooding disabled, for every VNI or for which one? |
| SVI addresses | Does each SVI of this VTEP carry an address of its own, unique among the VTEPs, that the other VTEPs must reach? |
| Ethernet Segment | Which Ethernet Segment does `<bond>` belong to: a local discriminator with a system MAC, or a full 10-byte Type-0 ESI? |
| DF preference | Which DF preference does this router take on the Ethernet Segment of `<bond>`? |
| Uplinks | Which interfaces of this router carry the underlay? |
| Startup delay | How long are the Ethernet Segments held down after a reboot, if not the default? |

### Warn, Never Ask

Missing, these do not block: the configuration carries the warning that `doctrine.md` gives for
each, and no invented value.

- The VXLAN devices, bridges and SVIs of the VNIs, which FRR does not create

### Never Ask

- FRR version: `frr version` is never written
- The L2VNIs of a VTEP: zebra learns them from the kernel, and an L2VNI is named only to carry
  a setting the request gives
- Route targets, when the request gives none: FRR derives them

### May Be Filled In

Nothing. No EVPN line carries a local name, and a value with operational effect (a DF
preference, a startup delay) is never filled in: without one in the request, its line is not
written and the FRR default holds.

### Question Form

One question per item, direct, with the reason attached.

```
Which VNI is the L3VNI of VRF `TENANT-A`?
why: `vni` in the `vrf` block cannot be written without it, and until it is, the VNI is an
L2VNI and the VRF exports no Type-5 route.
```

Never ask in bulk ("send me the network details").

## Examples

Canonical shapes of `frr.conf`, one file each under `references/evpn/examples/`. Read the one
closest to the request; copy its structure, never its values.

| File | Shape |
|------|-------|
| `references/evpn/examples/evpn-vtep-ospf-underlay.md` | VTEP with L2VNIs: an OSPF underlay for the VTEP address, one iBGP session to a route reflector |
| `references/evpn/examples/evpn-route-reflector.md` | Route reflector of the EVPN routes of two VTEPs, with no VNI of its own |
| `references/evpn/examples/evpn-l2vni-route-targets.md` | VTEP with one L2VNI whose route targets the request fixes |
| `references/evpn/examples/evpn-l3vni-type5.md` | VTEP with an IP-VRF: its L3VNI and Type-5 export of the VRF's routes |
| `references/evpn/examples/evpn-ip-vrf-route-targets.md` | IP-VRF whose route targets the request fixes, with a wildcard import |
| `references/evpn/examples/evpn-unique-svi-addresses.md` | VTEP whose SVIs carry unique addresses, advertised as Type-2 routes |
| `references/evpn/examples/evpn-multihoming.md` | VTEP of an Ethernet Segment: the bond's ESI and DF preference, uplink tracking |

## Verification Commands

What an operator runs to verify the configuration.

```
vtysh -c "show bgp l2vpn evpn summary"
vtysh -c "show bgp l2vpn evpn route"
vtysh -c "show bgp l2vpn evpn route type multicast"
vtysh -c "show bgp l2vpn evpn route type prefix"
vtysh -c "show bgp l2vpn evpn vni"
vtysh -c "show evpn vni <VNI>"
vtysh -c "show evpn mac vni <VNI> detail"
vtysh -c "show vrf vni"
vtysh -c "show evpn es detail"
```
