# iBGP Over An OSPF Underlay

The classic split of roles: the IGP carries only loopbacks and internal
links, which is what resolves the next-hop; iBGP carries the external
routes. `update-source` and `next-hop-self` are what make the pair work.

```
! ---- underlay: OSPF carries infrastructure only ----
interface lo
 ip address 10.0.0.1/32
 ip ospf area 0.0.0.0
exit
!
interface eth1
 ip ospf area 0.0.0.0
 ip ospf network point-to-point
 no ip ospf passive
 ! WARNING: no authentication on the adjacencies of eth1
exit
!
router ospf
 ospf router-id 10.0.0.1
 passive-interface default
exit
!
! ---- overlay: iBGP between loopbacks ----
router bgp 65000
 bgp router-id 10.0.0.1
 no bgp default ipv4-unicast
 neighbor IBGP peer-group
 neighbor IBGP remote-as 65000
 neighbor IBGP update-source lo
 neighbor 10.0.0.2 peer-group IBGP
 neighbor 10.0.0.3 peer-group IBGP
 !
 address-family ipv4 unicast
  neighbor IBGP activate
  ! mandatory on the border: the external next-hop is not reachable via OSPF
  neighbor IBGP next-hop-self
 exit-address-family
exit
```

What the shape demonstrates:

- the IGP redistributes nothing from BGP; the two tables stay separate
- sessions sourced from the loopback, so one link failure does not drop the
  session
- a peer-group used at the point where peers share one policy
- `next-hop-self` on the border router, where the external next-hop is
  otherwise unresolvable
