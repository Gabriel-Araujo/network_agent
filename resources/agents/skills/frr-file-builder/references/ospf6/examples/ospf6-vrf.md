# OSPFv3 In A VRF Toward A Customer Router

A provider edge router runs a separate OSPFv3 instance inside a customer VRF, over one link to
the customer router. The instance has its own router-id, and runs on the interfaces the request
puts in that VRF.

A reference for shape. Copy the structure, never the VRF, router-id, names or interfaces.

```
key chain CUST-A-OSPF6
 key 1
  cryptographic-algorithm hmac-sha-256
  ! WARNING: key 1 of CUST-A-OSPF6 has no key-string; set it on the router, never in this file
 exit
exit
!
interface eth3 vrf CUST-A
 ipv6 ospf6 area 0.0.0.0
 ipv6 ospf6 network point-to-point
 ipv6 ospf6 authentication keychain CUST-A-OSPF6
exit
!
router ospf6 vrf CUST-A
 ospf6 router-id 10.255.0.1
exit
!
```

What the shape demonstrates:

- `router ospf6 vrf <vrf>` opens the instance of the VRF, with its own router-id
- the interface named with its VRF, and enabled with `ipv6 ospf6 area` like any other
- authenticated the same as an instance in the default VRF

Sources: ospf6d › Configuration Commands; ospf6d › OSPF6 interface; isisd › ISIS Vrf Configuration Examples; ospf6d › Example configuration of keychain:; ospfd › Interfaces.
