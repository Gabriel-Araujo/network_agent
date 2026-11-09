# RPKI Origin Validation

BGP does not authenticate who originated a prefix. RPKI adds that layer
through an external validating cache (routinator, rpki-client, fort), and
the result becomes a route-map condition.

```
! daemon: bgpd built with RPKI support + enabled in /etc/frr/daemons
rpki
 rpki polling_period 300
 rpki cache 192.0.2.10 3323 preference 1
 rpki cache 192.0.2.11 3323 preference 2
exit
!
route-map RM-TRANSIT-IN deny 10
 ! invalid origin: drop
 match rpki invalid
exit
route-map RM-TRANSIT-IN permit 20
 match rpki valid
 set local-preference 200
exit
route-map RM-TRANSIT-IN permit 30
 ! no ROA published: accept, at a lower preference
 match rpki notfound
 set local-preference 100
exit
!
router bgp 65001
 address-family ipv4 unicast
  neighbor 203.0.113.1 route-map RM-TRANSIT-IN in
 exit-address-family
exit
!
! the rpki node syntax changed between FRR releases: confirm against yours
! verification: show rpki prefix-table / show bgp ipv4 unicast rpki invalid
```

What the shape demonstrates:

- RPKI syntax is version gated: confirm against the version of record or
  fall back to prefix-list filtering with a `VERSION NOTE`
- two caches configured with a preference, so validation survives one being
  down
- `notfound` accepted rather than dropped, which is what keeps the session
  useful during partial ROA coverage
- validation state expressed as local-preference, not as a hard accept/deny
  outside the invalid case
