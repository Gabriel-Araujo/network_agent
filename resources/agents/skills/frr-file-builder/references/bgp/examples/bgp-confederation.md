# BGP Confederation

The alternative to a route reflector: the AS is split into private sub-ASes
that see each other as eBGP internally, while the outside world sees a
single AS.

```
router bgp 65001
 ! identity as seen from outside
 bgp confederation identifier 65000
 ! sub-ASes belonging to the same confederation
 bgp confederation peers 65002 65003
 bgp router-id 10.0.0.1
 no bgp default ipv4-unicast
 !
 ! neighbor inside the same sub-AS = iBGP
 neighbor 10.0.0.2 remote-as 65001
 neighbor 10.0.0.2 update-source lo
 !
 ! neighbor in another sub-AS = confederation eBGP
 neighbor 10.0.1.1 remote-as 65002
 neighbor 10.0.1.1 update-source lo
 neighbor 10.0.1.1 ebgp-multihop 2
 !
 ! external transit sees AS 65000 only
 neighbor 203.0.113.1 remote-as 64512
 !
 address-family ipv4 unicast
  neighbor 10.0.0.2 activate
  neighbor 10.0.1.1 activate
  neighbor 10.0.1.1 next-hop-self
  neighbor 203.0.113.1 activate
 exit-address-family
exit
```

What the shape demonstrates:

- three session types in one block, each giving `remote-as` a different
  meaning: iBGP, confederation eBGP, real eBGP
- `bgp confederation identifier` and `bgp confederation peers` as a pair,
  neither of any use alone
- `ebgp-multihop` needed because the confederation session runs between
  loopbacks
- `next-hop-self` on the confederation border, for the same reason as on a
  real one
