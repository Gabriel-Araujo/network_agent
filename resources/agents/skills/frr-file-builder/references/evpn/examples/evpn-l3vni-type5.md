# IP-VRF With An L3VNI And Type-5 Export

A VTEP routes the hosts of IP-VRF `TENANT-A`, whose L3VNI the request gives as 5000. The VRF's
own BGP instance holds the tenant prefix 10.1.10.0/24, which the request gives, and exports it
to EVPN as a Type-5 route. On `frr:10.5.0`, another VTEP with the same L3VNI installs the prefix
in its own `TENANT-A`, through the SVI of the L3VNI. The underlay that carries the VTEP address
comes from its own pack and is left out here.

A reference for shape. Copy the structure, never the AS, addresses, VRF, VNI or prefix.

```
vrf TENANT-A
 vni 5000
exit-vrf
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
router bgp 65000 vrf TENANT-A
 !
 address-family ipv4 unicast
  ! WARNING: 10.1.10.0/24 must be in the RIB, or BGP will not advertise it
  network 10.1.10.0/24
 exit-address-family
 !
 address-family l2vpn evpn
  advertise ipv4 unicast
 exit-address-family
exit
!
```

What the shape demonstrates:

- the L3VNI set by `vni` in the `vrf` block, which the skill body opens
- `advertise ipv4 unicast` in the VRF's own BGP instance, not in the underlay instance
- `advertise-all-vni` only in the underlay instance
- the automatic route targets of the IP-VRF, with no `route-target` line

Sources: evpn › Sample Configuration; bgp › EVPN Overlay Index Gateway IP; bgp › Enabling EVPN; evpn › Automatic Route Targets; evpn › FRR Configuration; bgp › Networks; bgp › ASN and Router ID; bgp › Defining Peers; bgp › Configuring Peers.
