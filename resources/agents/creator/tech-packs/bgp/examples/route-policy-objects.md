# Route Policy: Prefix-List, AS-Path And Community

The three filters that hold up any border, combined: prefix-list for what,
as-path access-list for where from, community for why. The implicit `deny`
at the end of a route-map is the real protection against a leak.

```
! ---- what I accept ----
ip prefix-list PL-CUSTOMER seq 10 permit 198.51.100.0/24 le 24
ip prefix-list PL-BOGONS seq 10 permit 10.0.0.0/8 le 32
ip prefix-list PL-BOGONS seq 20 permit 172.16.0.0/12 le 32
ip prefix-list PL-BOGONS seq 30 permit 192.168.0.0/16 le 32
ip prefix-list PL-BOGONS seq 40 permit 0.0.0.0/0 ge 25
!
! ---- where I accept it from ----
bgp as-path access-list AS-CUSTOMER permit ^65010$
!
! ---- semantic labels ----
bgp community-list standard CL-CUSTOMER permit 65000:100
bgp community-list standard CL-PEER     permit 65000:200
bgp community-list standard CL-TRANSIT  permit 65000:300
!
route-map RM-CUSTOMER-IN deny 10
 match ip address prefix-list PL-BOGONS
exit
route-map RM-CUSTOMER-IN permit 20
 match ip address prefix-list PL-CUSTOMER
 match as-path AS-CUSTOMER
 set community 65000:100 additive
 set local-preference 300
exit
! implicit deny after seq 20: nothing else gets in
!
route-map RM-TRANSIT-OUT permit 10
 ! to transit, advertise customers and own prefixes - never peer routes
 match community CL-CUSTOMER
exit
route-map RM-TRANSIT-OUT permit 20
 match ip address prefix-list PL-OWN
exit
```

What the shape demonstrates:

- bogons denied in a low sequence, before any permit can match them
- `le` and `ge` used to bound prefix length instead of listing every length
- an inbound community `set` used as the label the outbound policy matches
  on, which is what keeps a peer route out of a transit advertisement
- the implicit deny relied on deliberately, not patched over with a trailing
  `permit`
