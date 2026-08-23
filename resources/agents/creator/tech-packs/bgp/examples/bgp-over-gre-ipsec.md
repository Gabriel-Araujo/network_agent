# BGP Over A GRE Tunnel With IPsec

IPsec provides confidentiality and GRE provides a routable interface; BGP
does the rest. The usual shape for site-to-site and multi-cloud
interconnect.

```
! ---- GRE tunnel (outside FRR) ----
! ip link add gre1 type gre local 203.0.113.2 remote 198.51.100.2 ttl 255
! ip link set gre1 up mtu 1400
! ip addr add 169.254.10.1/30 dev gre1
! IPsec (strongswan) protects the GRE traffic between the public endpoints
!
router bgp 65001
 bgp router-id 10.0.0.1
 no bgp default ipv4-unicast
 !
 neighbor 169.254.10.2 remote-as 65002
 neighbor 169.254.10.2 description remote site / over GRE+IPsec
 neighbor 169.254.10.2 bfd
 ! without this an unstable tunnel drops the session far too fast
 neighbor 169.254.10.2 timers 10 30
 !
 address-family ipv4 unicast
  network 10.10.0.0/16
  neighbor 169.254.10.2 activate
  neighbor 169.254.10.2 route-map RM-SITE-IN in
  neighbor 169.254.10.2 route-map RM-SITE-OUT out
  neighbor 169.254.10.2 maximum-prefix 5000 80 restart 15
 exit-address-family
exit
!
! the tunnel MTU requires MSS clamping on the firewall, or PMTUD breaks
```

What the shape demonstrates:

- the session addressed on the tunnel, never on the public endpoints
- timers relaxed rather than tightened, because the transport is already
  lossy
- BFD over the tunnel, which detects the tunnel dying while the underlay
  stays up
- `maximum-prefix` sized to a site, not to a full table
- the MTU/MSS consequence written down next to the config that causes it
