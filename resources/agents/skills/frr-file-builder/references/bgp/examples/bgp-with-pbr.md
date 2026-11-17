# BGP With PBR For Selective Egress

BGP picks the best path per destination prefix. PBR overrides that decision
for traffic that has to leave somewhere else - by source, by port, or by
marking.

```
! enable pbrd in /etc/frr/daemons
!
! alternate next-hop group
nexthop-group NHG-PROVIDER-B
 nexthop 203.0.113.5
exit
!
pbr-map PBR-EGRESS seq 10
 ! traffic from this source leaves via provider B, whatever BGP decided
 match src-ip 10.10.50.0/24
 set nexthop-group NHG-PROVIDER-B
exit
!
pbr-map PBR-EGRESS seq 20
 match dst-ip 192.0.2.0/24
 match dst-port 443
 set nexthop 203.0.113.1
exit
!
interface eth0
 pbr-policy PBR-EGRESS
exit
!
! verification: show pbr map / show pbr interface
```

What the shape demonstrates:

- PBR is applied to an ingress interface, so the scope is the traffic
  entering there, not the whole router
- a nexthop-group named and declared before the map that references it
- the override stated as an override: the comment says what BGP would have
  done
- `pbrd` outside the pack's default daemon set, called out in the artifact
