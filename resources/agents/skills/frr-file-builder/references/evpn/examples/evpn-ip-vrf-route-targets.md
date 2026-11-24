# IP-VRF With Route Targets Fixed By The Request

IP-VRF `TENANT-B`, L3VNI 6000, shares an EVPN domain whose VTEPs identify that tenant by the
local administrator 9000, not by its VNI, each with its own AS number or IPv4 address in front.
The request imports every route target whose local administrator is 9000, whatever is in front,
and exports with 65000:9000. A manual route target removes the automatic one of its direction:
on `frr:10.5.0`, `show bgp l2vpn evpn vni` then lists `0:9000` as the import route target of
VNI 6000 and `65000:9000` as its export one. The underlay that carries the VTEP address comes
from its own pack and is left out here.

A reference for shape. Copy the structure, never the AS, addresses, VRF, VNI or route targets.

```
vrf TENANT-B
 vni 6000
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
router bgp 65000 vrf TENANT-B
 !
 address-family l2vpn evpn
  route-target import *:9000
  route-target export 65000:9000
 exit-address-family
exit
!
```

What the shape demonstrates:

- the IP-VRF's route targets in `address-family l2vpn evpn` of the VRF's own BGP instance
- one route target per line and per direction
- the wildcard only on the import side, where the request asks for every global administrator
- route targets that EVPN attaches to the IP-VRF's Type-2 and Type-5 routes alike

Sources: bgp › EVPN IP-VRF Route Targets; evpn › Wildcard Route Targets; evpn › Automatic Route Targets; evpn › Sample Configuration; evpn › FRR Configuration; bgp › Enabling EVPN; bgp › ASN and Router ID; bgp › Defining Peers; bgp › Configuring Peers.
