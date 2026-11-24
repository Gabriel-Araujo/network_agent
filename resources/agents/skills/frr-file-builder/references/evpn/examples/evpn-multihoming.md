# VTEP Of An Ethernet Segment

A server is attached to two VTEPs over a bond, and the bonded links form one Ethernet Segment.
On this VTEP the bond, `bond1`, takes the Ethernet Segment the request gives: local
discriminator 1 and system MAC 44:38:39:ff:00:01, from which FRR derives a Type-3 ESI
(`03:44:38:39:ff:00:01:00:00:01` in `show evpn es detail` on `frr:10.5.0`). The request gives a
DF preference of 50000 and a startup delay of 300 seconds (`show evpn` on `frr:10.5.0` lists
the delay in seconds). `eth1` and `eth2` carry the underlay,
so they track the uplinks: with both down, the server links of the segment are protodowned. The
VTEP address is IPv4. The underlay's own lines on `eth1` and `eth2` come from its pack and are
left out here.

A reference for shape. Copy the structure, never the AS, addresses, interfaces, ESI or values.

```
! the Ethernet Segment bonds stay protodown for 300 s after a reboot, while the underlay and EVPN converge
evpn mh startup-delay 300
!
interface bond1
 evpn mh es-id 1
 evpn mh es-sys-mac 44:38:39:ff:00:01
 evpn mh es-df-pref 50000
exit
!
interface eth1
 evpn mh uplink
exit
!
interface eth2
 evpn mh uplink
exit
!
router bgp 65000
 bgp router-id 10.255.0.11
 neighbor 10.255.0.1 remote-as 65000
 neighbor 10.255.0.1 description spine1, the route reflector of the EVPN routes
 ! WARNING: no authentication on the session with 10.255.0.1
 neighbor 10.255.0.1 update-source lo
 !
 ! WARNING: FRR does not create the VXLAN devices, bridges and SVIs of these VNIs; zebra learns them from the kernel
 address-family l2vpn evpn
  neighbor 10.255.0.1 activate
  advertise-all-vni
 exit-address-family
exit
!
```

What the shape demonstrates:

- the Ethernet Segment in the interface block of its bond: a local discriminator with its
  system MAC
- the DF preference and the startup delay only because the request gives them
- `evpn mh startup-delay` outside any block, before every object, with its effect in a comment
- uplink tracking on every underlay interface

Sources: bgp › Ethernet Segments; bgp › BUM handling; bgp › Uplink/Core tracking; bgp › Startup delay; bgp › EVPN Multihoming; bgp › Enabling EVPN; evpn › Sample Configuration; evpn › FRR Configuration; bgp › ASN and Router ID; bgp › Defining Peers; bgp › Configuring Peers.
