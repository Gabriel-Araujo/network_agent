# Maintenance With Graceful Restart

*ospfd* on this router is restarted for an upgrade while the router keeps forwarding traffic.
With graceful restart it announces the restart to its neighbors with Grace-LSAs, and must be
back within the grace period. Helper mode, disabled by default, lets this router support the
same restart of a neighbor. Before a planned restart the operator runs `graceful-restart prepare ip ospf`, an
EXEC command that is never part of the file.

A reference for shape. Copy the structure, never the router-id, names or interfaces.

```
key chain OSPF-CORE
 key 1
  cryptographic-algorithm hmac-sha-256
  ! WARNING: key 1 of OSPF-CORE has no key-string; set it on the router, never in this file
 exit
exit
!
interface lo
 ip ospf area 0.0.0.0
exit
!
interface eth1
 ip ospf area 0.0.0.0
 ip ospf network point-to-point
 no ip ospf passive
 ip ospf authentication key-chain OSPF-CORE
exit
!
router ospf
 ospf router-id 10.0.0.6
 passive-interface default
 ! default grace period, 120 s: ospfd must be back within it
 graceful-restart
 graceful-restart helper enable
exit
!
```

What the shape demonstrates:

- restarting support for this router, and helper support for the restarts of its neighbors,
  each its own line
- the default grace period kept, with a `!` comment saying what it bounds
- no `capability opaque` line: graceful restart signals through opaque LSAs, which are on by
  default

Sources: ospfd › Graceful Restart; ospfd › Opaque LSA; ospfd › Routers; ospfd › Interfaces.
