# EVPN Checklist

Answer every item against the artifact as it stands: met, not applicable with a reason, or
violated.

## Security

| # | Check | Source |
|---|-------|--------|
| 1 | `neighbor <peer> activate` appears in `address-family l2vpn evpn` only for the sessions the plan names as carrying EVPN | evpn › Sample Configuration |
| 2 | No `bgp default l2vpn-evpn` | bgp › Configuring Peers |
| 3 | Every session activated in `address-family l2vpn evpn` is iBGP | bgp › Require policy on EBGP |
| 4 | Every import route target the artifact writes, a wildcard one included, is a value the request gives | evpn › EVPN Route Targets; evpn › Wildcard Route Targets |
| 5 | `daemons` enables `zebra`, `bgpd` and the daemons the underlay's pack names, and any other daemon only with a justification in the plan | ospfd › Multi-instance Support; evpn › Sample Configuration; bgp › Enabling EVPN |

## Performance

| # | Check | Source |
|---|-------|--------|
| 6 | `advertise ipv4 unicast` and `advertise ipv6 unicast` appear only in the BGP instance of an IP-VRF, for the address families the plan exports | evpn › Sample Configuration; bgp › EVPN Overlay Index Gateway IP |
| 7 | `advertise-svi-ip` appears only on a VTEP whose SVIs the plan gives unique addresses to reach, and not together with `advertise-default-gw` | bgp › EVPN advertise-svi-ip |
| 8 | `flooding disable` appears only for the VNIs the plan names, or for every VNI when the plan says so | bgp › EVPN BUM Handling |
| 9 | On a VTEP with an Ethernet Segment, every interface the plan names as underlay has `evpn mh uplink` | bgp › Uplink/Core tracking |
| 10 | `evpn mh es-df-pref` and `evpn mh startup-delay` appear only with the values the request gives, and `evpn mh startup-delay` carries a `!` comment with its expected effect | bgp › BUM handling; bgp › Startup delay |

## Integrity

| # | Check | Source |
|---|-------|--------|
| 11 | On a VTEP, exactly one BGP instance has `advertise-all-vni`; a router that only reflects EVPN routes has none | bgp › Enabling EVPN; bgp › Route Reflector |
| 12 | Every session the plan names as carrying EVPN has `neighbor <peer> activate` in `address-family l2vpn evpn` | evpn › Sample Configuration |
| 13 | On a VTEP, the plan names how the other VTEPs reach the VTEP address, and the artifact, or the file it adds to, carries that underlay | evpn › Sample Configuration |
| 14 | Every IP-VRF the plan routes over EVPN has `vni <L3VNI>` in its `vrf` block | evpn › Sample Configuration |
| 15 | Every `vni` block and every L2VNI route target is in `address-family l2vpn evpn` of the instance with `advertise-all-vni`; every IP-VRF route target is in the BGP instance of its VRF | bgp › EVPN MAC-VRF / L2VNI Route Targets; bgp › EVPN IP-VRF Route Targets |
| 16 | Every wildcard route target is in `route-target import` of an IP-VRF | bgp › EVPN IP-VRF Route Targets; bgp › EVPN MAC-VRF / L2VNI Route Targets |
| 17 | Every route target with a 4-byte AS or an IPv4 address in front has a local administrator that fits in 2 bytes | evpn › Wildcard Route Targets |
| 18 | Every Ethernet Segment has `evpn mh es-id` with either `evpn mh es-sys-mac` or a full 10-byte Type-0 ESI, on a VTEP whose VTEP address is IPv4 | bgp › Ethernet Segments; bgp › EVPN Multihoming; evpn › Sample Configuration |
| 19 | The warning that FRR does not create the VXLAN devices, bridges and SVIs stands above the first `address-family l2vpn evpn` of a VTEP | evpn › FRR Configuration |
