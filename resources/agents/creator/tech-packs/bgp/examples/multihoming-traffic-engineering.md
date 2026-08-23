# Multihoming With Traffic Engineering

Outbound is a local decision (`local-preference`). Inbound is persuasion
(`as-path prepend`, MED, provider communities). The two directions need
different mechanisms.

```
! ---- OUTBOUND: prefer leaving through provider A ----
route-map RM-A-IN permit 10
 set local-preference 200
exit
route-map RM-B-IN permit 10
 set local-preference 100
exit
!
! ---- INBOUND: discourage entry through provider B ----
route-map RM-B-OUT permit 10
 match ip address prefix-list PL-OWN
 set as-path prepend 65001 65001 65001
 ! many providers publish control communities; generic example:
 set community 64513:120 additive
exit
route-map RM-A-OUT permit 10
 match ip address prefix-list PL-OWN
exit
!
router bgp 65001
 neighbor 203.0.113.1 remote-as 64512
 neighbor 203.0.113.1 description provider-A / preferred
 neighbor 203.0.113.5 remote-as 64513
 neighbor 203.0.113.5 description provider-B / secondary
 !
 address-family ipv4 unicast
  network 198.51.100.0/24
  neighbor 203.0.113.1 activate
  neighbor 203.0.113.1 route-map RM-A-IN in
  neighbor 203.0.113.1 route-map RM-A-OUT out
  neighbor 203.0.113.5 activate
  neighbor 203.0.113.5 route-map RM-B-IN in
  neighbor 203.0.113.5 route-map RM-B-OUT out
 exit-address-family
exit
```

What the shape demonstrates:

- the asymmetry stated explicitly: local-preference inbound-applied for
  outbound traffic, prepend outbound-applied for inbound traffic
- an outbound policy on both sessions bounded by `PL-OWN`, which is what
  stops the site becoming transit between the two providers
- prepend and provider community used together, because neither alone is
  reliable
- direction named in every policy object, per the run vocabulary
