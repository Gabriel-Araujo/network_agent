# MP-BGP Dual-Stack Over One Session

A single BGP session over IPv6 transport carries both the IPv4 and IPv6
families. Halves the number of sessions to operate, at the cost of tying
both families to one failure domain.

```
router bgp 65001
 bgp router-id 10.0.0.1
 no bgp default ipv4-unicast
 !
 neighbor 2001:db8::1 remote-as 64512
 neighbor 2001:db8::1 description upstream / dual-stack transit
 ! allows an IPv4 next-hop over IPv6 transport (RFC 5549)
 neighbor 2001:db8::1 capability extended-nexthop
 !
 address-family ipv4 unicast
  network 198.51.100.0/24
  neighbor 2001:db8::1 activate
  neighbor 2001:db8::1 route-map RM-V4-IN in
  neighbor 2001:db8::1 route-map RM-V4-OUT out
 exit-address-family
 !
 address-family ipv6 unicast
  network 2001:db8:100::/48
  neighbor 2001:db8::1 activate
  neighbor 2001:db8::1 route-map RM-V6-IN in
  neighbor 2001:db8::1 route-map RM-V6-OUT out
 exit-address-family
exit
```

What the shape demonstrates:

- one neighbor explicitly activated in each family it uses, with its own
  policy per family
- `capability extended-nexthop` at the global level, where capabilities are
  negotiated
- `no bgp default ipv4-unicast`, which is what makes the two `activate`
  lines meaningful instead of decorative
- the tradeoff stated in the framing, so the plan can reject it
