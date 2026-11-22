# IS-IS In A VRF Toward A Customer Router

A provider edge router runs an IS-IS instance inside a customer VRF, over one link to the
customer router. The instance has its own name and its own `net` line, and runs on the interfaces the request
puts in that VRF.

A reference for shape. Copy the structure, never the VRF, NET, names or interfaces.

```
interface eth3 vrf CUST-A
 ip router isis CUST-A
 isis network point-to-point
 ! WARNING: isis password md5 is left out; add it on the router, never in this file
exit
!
router isis CUST-A vrf CUST-A
 net 49.0100.0000.0000.0001.00
 is-type level-2-only
 ! WARNING: domain-password md5 is left out; add it on the router, never in this file
exit
!
```

What the shape demonstrates:

- `router isis <name> vrf <vrf>` opens the instance of the VRF, with its own `net` line
- the interface named with its VRF, and enabled with `ip router isis` naming that instance
- the same password warnings as an instance in the default VRF

Sources: isisd › ISIS router; isisd › ISIS interface; isisd › ISIS Vrf Configuration Examples.
