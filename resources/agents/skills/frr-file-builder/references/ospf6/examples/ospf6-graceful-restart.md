# Maintenance With Graceful Restart

*ospf6d* on this router is restarted for an upgrade while the router keeps forwarding traffic.
With graceful restart (RFC 5187) it must be back within the grace period, or the restart fails.
Helper mode, disabled by default, lets this router support the same restart of a neighbor.
Before a planned restart the operator runs `graceful-restart prepare ipv6 ospf`, an EXEC
command that is never part of the file.

A reference for shape. Copy the structure, never the router-id, names or interfaces.

```
key chain OSPF6-CORE
 key 1
  cryptographic-algorithm hmac-sha-256
  ! WARNING: key 1 of OSPF6-CORE has no key-string; set it on the router, never in this file
 exit
exit
!
interface lo
 ipv6 ospf6 area 0.0.0.0
 ipv6 ospf6 passive
exit
!
interface eth1
 ipv6 ospf6 area 0.0.0.0
 ipv6 ospf6 network point-to-point
 ipv6 ospf6 authentication keychain OSPF6-CORE
exit
!
router ospf6
 ospf6 router-id 10.0.0.6
 ! default grace period, 120 s: ospf6d must be back within it
 graceful-restart
 graceful-restart helper enable
exit
!
```

What the shape demonstrates:

- restarting support for this router, and helper support for the restarts of its neighbors,
  each its own line
- the default grace period kept, with a `!` comment saying what it bounds
- `ipv6 ospf6 graceful-restart hello-delay` left at its default, equal to the default hello
  interval

Sources: ospf6d › Graceful Restart; ospf6d › OSPF6 interface; ospf6d › Configuration Commands; ospf6d › Example configuration of keychain:; ospfd › Interfaces.
