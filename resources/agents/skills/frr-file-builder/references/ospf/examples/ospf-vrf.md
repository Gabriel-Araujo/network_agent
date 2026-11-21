# OSPF In A VRF Toward A Customer Router

A provider edge router runs a separate OSPF instance inside a customer VRF, over one link to
the customer router. The instance has its own router-id, and runs on the interfaces the
request puts in that VRF.

A reference for shape. Copy the structure, never the VRF, router-id, names or interfaces.

```
key chain CUST-A-OSPF
 key 1
  cryptographic-algorithm hmac-sha-256
  ! WARNING: key 1 of CUST-A-OSPF has no key-string; set it on the router, never in this file
 exit
exit
!
interface eth3 vrf CUST-A
 ip ospf area 0.0.0.0
 ip ospf network point-to-point
 no ip ospf passive
 ip ospf authentication key-chain CUST-A-OSPF
exit
!
router ospf vrf CUST-A
 ospf router-id 10.255.0.1
 passive-interface default
exit
!
```

What the shape demonstrates:

- `router ospf vrf <vrf>` opens the instance of the VRF, with its own router-id
- the interface named with its VRF, and enabled with `ip ospf area` like any other
- passive by default and authenticated, the same as an instance in the default VRF

Sources: ospfd › Routers; ospfd › Interfaces; isisd › ISIS Vrf Configuration Examples.
