# Route Reflector Of The EVPN Routes

A spine reflects the EVPN routes of two VTEPs, its route reflector clients, over iBGP sessions
to their loopbacks. The spine is no VTEP: it has no VNI of its own and carries no
`advertise-all-vni`. On `frr:10.5.0` it still reflects the Type-3 and Type-5 routes of each
client to the other, and each client lists the other as a remote VTEP of their common VNI. The
underlay that carries the loopbacks comes from its own pack and is left out here.

A reference for shape. Copy the structure, never the AS, addresses or names.

```
router bgp 65000
 bgp router-id 10.255.0.1
 neighbor 10.255.0.11 remote-as 65000
 neighbor 10.255.0.11 description leaf1, a VTEP and route reflector client
 ! WARNING: no authentication on the session with 10.255.0.11
 neighbor 10.255.0.11 update-source lo
 neighbor 10.255.0.12 remote-as 65000
 neighbor 10.255.0.12 description leaf2, a VTEP and route reflector client
 ! WARNING: no authentication on the session with 10.255.0.12
 neighbor 10.255.0.12 update-source lo
 !
 address-family l2vpn evpn
  neighbor 10.255.0.11 activate
  neighbor 10.255.0.11 route-reflector-client
  neighbor 10.255.0.12 activate
  neighbor 10.255.0.12 route-reflector-client
 exit-address-family
exit
!
```

What the shape demonstrates:

- `address-family l2vpn evpn` with no `advertise-all-vni` on a router that only reflects EVPN
  routes
- each client session activated in `address-family l2vpn evpn`
- no warning about the VXLAN devices: the spine has no VNI

Sources: bgp › Route Reflector; bgp › Enabling EVPN; evpn › Sample Configuration; bgp › ASN and Router ID; bgp › Defining Peers; bgp › Configuring Peers.
