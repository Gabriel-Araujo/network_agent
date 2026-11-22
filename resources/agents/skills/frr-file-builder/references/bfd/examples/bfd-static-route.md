# Static Routes Monitored By BFD

The request routes one IPv4 and one IPv6 prefix, each through a directly connected gateway, and
wants each route in the RIB only while BFD sees its gateway. A monitored static route uses its
gateway address as the BFD peer: while the session is up the route is installed in the RIB, and
while it is down the route is removed. The request gives a detect-multiplier of 3 and 200 ms
intervals for these sessions. A route removed this way no longer shows in `show ip route` or
`show ipv6 route`; `show bfd static route` shows it.

A reference for shape. Copy the structure, never the prefixes, gateways or names.

```
bfd
 profile GATEWAY
  ! the gateway declares a session down after 3 x 200 ms = 600 ms without packets,
  ! while its own receive-interval is 200 ms or less
  detect-multiplier 3
  receive-interval 200
  transmit-interval 200
 exit
exit
!
ip route 192.0.2.0/24 10.0.12.2 bfd profile GATEWAY
ipv6 route 2001:db8:100::/48 2001:db8:12::2 bfd profile GATEWAY
!
```

What the shape demonstrates:

- the `bfd` block before the static routes that name its profile
- one profile for the IPv4 and the IPv6 sessions, which want the same detection

Sources: bfd › BFD Static Route Monitoring Configuration; bfd › Peer / Profile Configuration.
