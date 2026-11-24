# EVPN Doctrine

Default posture is opt-in: EVPN is enabled in one BGP instance, the underlay's, and carried only
by the sessions the request names; a route target or a VNI is written only with the value the
request gives: FRR derives the route targets it is not given, and zebra learns the VNIs from the
kernel. Each rule below is met, not applicable for a stated reason, or a regression. The Source
column names the `.rst` section that carries the command or the behaviour the rule rests on. A
behaviour the corpus does not state, proved on a live `frr:10.5.0`, says so.

## Overlay Policy

What every EVPN configuration carries, whatever else the request asks for.

| Rule | Source |
|------|--------|
| On a VTEP, `advertise-all-vni` in `address-family l2vpn evpn` of exactly one BGP instance, the underlay VRF's: the default VRF unless the request names another. It defines the VRF used as the underlay for EVPN-VXLAN and enables EVPN, and it is mandatory for EVPN-VXLAN to work | bgp › Enabling EVPN |
| A router that only reflects EVPN routes between VTEPs has no `advertise-all-vni`. On `frr:10.5.0`, a route reflector without it reflects the Type-3 and Type-5 routes of its clients, and each client lists the other as a remote VTEP of their common VNI | bgp › Route Reflector; bgp › Enabling EVPN |
| `neighbor <peer> activate` in `address-family l2vpn evpn` for each session the request names as carrying EVPN, and for no other: it lets that session advertise and receive EVPN routes | evpn › Sample Configuration; bgp › IPv6 Support |
| No `bgp default l2vpn-evpn`, which turns the L2VPN EVPN address family on by default: each EVPN session is activated by its own line | bgp › Configuring Peers |
| Every session that carries EVPN is iBGP. `bgp ebgp-requires-policy`, on by default for the traditional configuration, requires incoming and outgoing filters on an eBGP session; on `frr:10.5.0` an eBGP session without them in `address-family l2vpn evpn` exchanges no EVPN route, and this pack writes no such filter | bgp › Require policy on EBGP |
| On a VTEP, the other VTEPs reach its VTEP address through the underlay the request names: BGP, static routes or an IGP, each written by its own pack. The VTEP address needs to be reachable by the other VTEPs for VXLAN decapsulation to function | evpn › Sample Configuration |
| A VNI is named only to carry a setting the request gives: zebra discovers the VNIs from netlink. On `frr:10.5.0`, `advertise-all-vni` advertises a VNI that no line of the file names | evpn › Sample Configuration; bgp › Enabling EVPN |
| `vni <VNI>` in the `vrf` block of each IP-VRF the request routes over EVPN, with the L3VNI the request gives. Until this line exists, zebra considers that VNI an L2VNI | evpn › Sample Configuration |
| A route target only with the value the request gives. Without one, FRR derives an automatic route target per direction for every IP-VRF with an L3VNI and every L2VNI, and EVPN mostly works out of the box. A manual route target removes the automatic one of its direction. On `frr:10.5.0`, two VTEPs of one AS import each other's routes with the automatic ones, but `show bgp l2vpn evpn vni` lists the automatic import route target as `<AS>:<VNI>`, where evpn › Automatic Route Targets derives the wildcard `*:VNI` | evpn › Automatic Route Targets |
| Above the first `address-family l2vpn evpn` of a VTEP, the warning `! WARNING: FRR does not create the VXLAN devices, bridges and SVIs of these VNIs; zebra learns them from the kernel`. FRR learns the Linux interface configuration from the kernel via Netlink, and does not manage network interfaces directly | evpn › FRR Configuration; evpn › Sample Configuration |

The rest of the tables below apply when the request asks for the feature or gives what it
needs; absent, they add neither a line nor a warning.

## Security

| Rule | Note | Source |
|------|------|--------|
| An import route target, of an IP-VRF or of an L2VNI, only with the value the request gives | The route targets alone decide into which IP-VRF or MAC-VRF a received route is imported, never the VNI it carries: a router imports a route into exactly those IP-VRFs and MAC-VRFs whose import route targets match one the route carries | evpn › EVPN Route Targets; bgp › EVPN IP-VRF Route Targets; bgp › EVPN MAC-VRF / L2VNI Route Targets |
| A wildcard `route-target import *:<N>` only when the request asks to import every route target whose local administrator is `<N>`, whatever AS number or IPv4 address is in front | It matches any fully qualified route target whose local administrator is `<N>`, regardless of the global administrator | evpn › Wildcard Route Targets; bgp › EVPN IP-VRF Route Targets |

## Control Plane Minimization

The `daemons` file enables `bgpd` the way `ospfd=yes` enables *ospfd*, with *zebra*; the
underlay's daemon follows its own pack, and any other daemon needs the explicit justification
`TECH.md` › Daemons asks for. EVPN is activated only on the sessions the request names, and only
a VTEP has `advertise-all-vni`.
[ospfd › Multi-instance Support; evpn › Sample Configuration; bgp › Enabling EVPN]

## Security Regression

Activating `address-family l2vpn evpn` on a session the request did not name, or adding an
import route target to an IP-VRF or an L2VNI the file already has, a wildcard one on an IP-VRF
included, is built only after a `SECURITY WARNING` and explicit user confirmation. Without the
confirmation, nothing is built.
[evpn › Sample Configuration; evpn › EVPN Route Targets; evpn › Wildcard Route Targets; bgp › EVPN IP-VRF Route Targets; bgp › EVPN MAC-VRF / L2VNI Route Targets]

## Performance

| Rule | Note | Source |
|------|------|--------|
| `advertise ipv4 unicast` or `advertise ipv6 unicast` in the BGP instance of an IP-VRF only for the address families the request exports to EVPN | An IP-VRF's BGP prefixes are not exported as Type-5 routes until the command is configured; then every route of the VRF's BGP RIB that is valid and selected is eligible, except a route imported directly from EVPN, and only the best path of each prefix is exported | evpn › Sample Configuration; bgp › EVPN Overlay Index Gateway IP |
| `advertise-svi-ip` only on a VTEP whose SVIs carry unique addresses the request wants reachable | It advertises the SVI IP/MAC address as a Type-2 route, so no flooding over VXLAN is needed to reach it; typically the SVI address is reused on VTEPs across racks. `advertise-svi-ip` and `advertise-default-gw` should not be enabled at the same time | bgp › EVPN advertise-svi-ip |
| `flooding disable` only for the VNIs the request names, or for every VNI when it says so | By default BUM traffic is flooded across all VTEPs of the EVPN instance; with `disable`, it is not flooded for that VNI, or globally | bgp › EVPN BUM Handling |
| An Ethernet Segment only on a VTEP whose VTEP address is IPv4 | The corpus disagrees with itself: bgp › EVPN Multihoming says IPv6 VTEP addresses are not supported with EVPN Multihoming, and evpn › Sample Configuration says the VTEP-IP can be IPv4 or IPv6 for both singlehomed and multihomed deployments. This pack keeps to the narrower one | bgp › EVPN Multihoming; evpn › Sample Configuration |
| On a VTEP with an Ethernet Segment, `evpn mh uplink` on every interface the request names as underlay | When all the underlay links go down, the VTEP protodowns the server links of its Ethernet Segments, to prevent blackholing of traffic | bgp › Uplink/Core tracking |
| `evpn mh es-df-pref` only with the value the request gives | The DF election is preference based, and only the DF forwards the BUM traffic it receives from the overlay to the access port | bgp › BUM handling |
| `evpn mh startup-delay` only with the value the request gives | After a reboot the Ethernet Segment bonds are held protodown for this time, so the underlay and EVPN converge before the segments are enabled. On `frr:10.5.0`, `show evpn` lists the delay in seconds | bgp › Startup delay |
| Every timer carries a `!` comment stating its expected effect | | bgp › Startup delay |
