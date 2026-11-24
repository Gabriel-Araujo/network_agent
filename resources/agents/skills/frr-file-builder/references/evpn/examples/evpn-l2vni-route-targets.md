# L2VNI With Route Targets Fixed By The Request

A VTEP joins VNI 200 of an EVPN domain whose VTEPs import and export that MAC-VRF's routes with
the route target 65000:2000, which the request gives, in place of the automatic one. The route
target goes in the `vni` block of the underlay instance; every other VNI of the VTEP keeps its
automatic route targets and has no line. The underlay that carries the VTEP address comes from
its own pack and is left out here.

A reference for shape. Copy the structure, never the AS, addresses, VNIs or route targets.

```
router bgp 65000
 bgp router-id 10.255.0.12
 neighbor 10.255.0.1 remote-as 65000
 neighbor 10.255.0.1 description spine1, the route reflector of the EVPN routes
 ! WARNING: no authentication on the session with 10.255.0.1
 neighbor 10.255.0.1 update-source lo
 !
 ! WARNING: FRR does not create the VXLAN devices, bridges and SVIs of these VNIs; zebra learns them from the kernel
 address-family l2vpn evpn
  neighbor 10.255.0.1 activate
  advertise-all-vni
  vni 200
   route-target both 65000:2000
  exit-vni
 exit-address-family
exit
!
```

What the shape demonstrates:

- the `vni` block only for the VNI that has a setting of its own
- `both`, for one route target the request gives for import and export
- the L2VNI route target inside the instance that has `advertise-all-vni`, the only one that
  accepts it

Sources: bgp › EVPN MAC-VRF / L2VNI Route Targets; bgp › EVPN IP-VRF Route Targets; evpn › EVPN Route Targets; evpn › Automatic Route Targets; bgp › Enabling EVPN; evpn › Sample Configuration; evpn › FRR Configuration; bgp › ASN and Router ID; bgp › Defining Peers; bgp › Configuring Peers.
