# eBGP With Aggregate Origination And A Static Anchor

An aggregate is only advertised while it exists in the local RIB, so the
blackhole static route is the anchor: it keeps the block advertised
permanently and stops traffic for unused addresses from looping.

```
! aggregate anchor - without it the prefix can disappear from the advertisement
ip route 198.51.100.0/24 blackhole
!
router bgp 65001
 bgp router-id 203.0.113.2
 no bgp default ipv4-unicast
 neighbor 203.0.113.1 remote-as 64512
 neighbor 203.0.113.1 description upstream / transit
 !
 address-family ipv4 unicast
  network 198.51.100.0/24
  ! advertise the aggregate only, suppressing the internal more-specifics
  aggregate-address 198.51.100.0/24 summary-only
  neighbor 203.0.113.1 activate
  neighbor 203.0.113.1 route-map RM-OUT out
  neighbor 203.0.113.1 route-map RM-IN in
 exit-address-family
exit
!
ip prefix-list PL-OWN seq 10 permit 198.51.100.0/24
route-map RM-OUT permit 10
 match ip address prefix-list PL-OWN
exit
route-map RM-IN permit 10
exit
```

What the shape demonstrates:

- a `staticd` blackhole route used as an origination anchor, not as a filter
- `aggregate-address ... summary-only` paired with `network`, so only the
  covering block leaves the router
- `no bgp default ipv4-unicast`, so no family is carried by accident
- policy objects declared before the block that references them
- both directions carry a route-map, even where the inbound one is a no-op
