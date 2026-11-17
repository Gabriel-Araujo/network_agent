# Maintenance With Graceful Restart And Graceful Shutdown

Two mechanisms with distinct purposes: restart preserves forwarding while
the daemon restarts; shutdown (RFC 8326) drains traffic before a planned
stop, avoiding loss of in-flight connections.

```
router bgp 65001
 ! preserve the FIB while bgpd restarts
 bgp graceful-restart
 bgp graceful-restart preserve-fw-state
 bgp graceful-restart restart-time 120
 bgp graceful-restart stalepath-time 360
exit
!
! ---- maintenance procedure ----
! 1) drain: marks the routes with GRACEFUL_SHUTDOWN and local-pref 0
!    vtysh -c "conf t" -c "router bgp 65001" -c "bgp graceful-shutdown"
! 2) wait for the existing connections to empty
! 3) stop the service and carry out the maintenance
! 4) revert:
!    vtysh -c "conf t" -c "router bgp 65001" -c "no bgp graceful-shutdown"
! 5) verify: show bgp ipv4 unicast neighbors <peer> advertised-routes
!
! honour the community on ingress, when the neighbor sends it:
bgp community-list standard CL-GSHUT permit graceful-shutdown
!
route-map RM-PEER-IN permit 10
 match community CL-GSHUT
 set local-preference 0
exit
route-map RM-PEER-IN permit 20
exit
```

What the shape demonstrates:

- the two mechanisms separated: one is config, the other is a procedure
- `long-lived-graceful-restart` deliberately not used, since it is version
  gated; plain graceful restart is the stable alternative
- the drain honoured on ingress as well as signalled on egress, since only
  the receiving side can act on the community
- a trailing `permit` after the match, so the policy drains rather than
  filters
