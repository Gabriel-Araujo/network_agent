# VTEP Whose SVIs Carry Unique Addresses

Each SVI of this VTEP carries an address of its own, unique among the VTEPs, and the request
wants the other VTEPs to reach those addresses. `advertise-svi-ip` advertises the IP/MAC address
of each SVI as a Type-2 route, so a remote VTEP reaches it with no flooding over VXLAN. VTEPs
that reuse one SVI address, the typical case, leave the line out. The underlay that carries the
VTEP address comes from its own pack and is left out here.

A reference for shape. Copy the structure, never the AS, addresses or names.

```
router bgp 65000
 bgp router-id 10.255.0.13
 neighbor 10.255.0.1 remote-as 65000
 neighbor 10.255.0.1 description spine1, the route reflector of the EVPN routes
 ! WARNING: no authentication on the session with 10.255.0.1
 neighbor 10.255.0.1 update-source lo
 !
 ! WARNING: FRR does not create the VXLAN devices, bridges and SVIs of these VNIs; zebra learns them from the kernel
 address-family l2vpn evpn
  neighbor 10.255.0.1 activate
  advertise-all-vni
  advertise-svi-ip
 exit-address-family
exit
!
```

What the shape demonstrates:

- `advertise-svi-ip` in the instance where EVPN is enabled, next to `advertise-all-vni`
- one line for the SVIs of every VNI of the VTEP

Sources: bgp › EVPN advertise-svi-ip; evpn › Sample Configuration; bgp › Enabling EVPN; evpn › FRR Configuration; bgp › ASN and Router ID; bgp › Defining Peers; bgp › Configuring Peers.
