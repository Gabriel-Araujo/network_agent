# Conditional Advertisement For Backup Transit

The prefix is advertised to the secondary provider only once an anchor route
from the primary disappears. Avoids paying for backup transit while the main
link is healthy.

```
! anchor route that exists only while the primary is alive
ip prefix-list PL-ANCHOR seq 10 permit 192.0.2.0/24
route-map RM-CHECK-PRIMARY permit 10
 match ip address prefix-list PL-ANCHOR
exit
!
! what to advertise once the anchor is gone
ip prefix-list PL-OWN seq 10 permit 198.51.100.0/24
route-map RM-ADVERTISE-BACKUP permit 10
 match ip address prefix-list PL-OWN
exit
!
router bgp 65001
 neighbor 203.0.113.5 remote-as 64513
 neighbor 203.0.113.5 description backup transit / paid per volume
 !
 address-family ipv4 unicast
  neighbor 203.0.113.5 activate
  ! advertise RM-ADVERTISE-BACKUP only while RM-CHECK-PRIMARY matches nothing
  neighbor 203.0.113.5 advertise-map RM-ADVERTISE-BACKUP non-exist-map RM-CHECK-PRIMARY
 exit-address-family
exit
!
! verification: show bgp ipv4 unicast neighbors 203.0.113.5 advertised-routes
```

What the shape demonstrates:

- two route-maps that never filter traffic: one is a condition, the other is
  the payload
- the anchor chosen as a route the primary always sends, so its absence is a
  reliable failure signal
- `advertise-map`/`non-exist-map` inside the address family
- the verification command that actually proves the condition, since the
  session looks identical either way
