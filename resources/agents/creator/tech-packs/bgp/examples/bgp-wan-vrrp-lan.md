# BGP On The WAN With VRRP On The LAN

Two border routers hold independent eBGP sessions to different providers,
but present one virtual gateway to the hosts. VRRP solves the host side;
BGP solves the world side.

```
! ---- LAN: virtual gateway 10.10.10.1 ----
interface eth0
 ip address 10.10.10.2/24
 vrrp 10
 vrrp 10 ip 10.10.10.1
 vrrp 10 priority 200
 vrrp 10 advertisement-interval 1000
exit
!
! ---- WAN: eBGP session with provider A ----
interface eth1
 ip address 203.0.113.2/30
exit
!
router bgp 65001
 bgp router-id 10.0.0.1
 no bgp default ipv4-unicast
 neighbor 203.0.113.1 remote-as 64512
 neighbor 203.0.113.1 description provider-A / primary transit
 !
 ! iBGP with the peer router, to sync external routes
 neighbor 10.10.10.3 remote-as 65001
 neighbor 10.10.10.3 description border-pair / VRRP backup
 !
 address-family ipv4 unicast
  network 198.51.100.0/24
  neighbor 203.0.113.1 activate
  neighbor 203.0.113.1 route-map RM-UP-OUT out
  neighbor 10.10.10.3 activate
  neighbor 10.10.10.3 next-hop-self
 exit-address-family
exit
!
! on the peer: same config with "vrrp 10 priority 100" and provider B
! enable vrrpd in /etc/frr/daemons
```

What the shape demonstrates:

- two failure domains handled by two protocols, each on the side it fits
- the iBGP session between the pair, without which each router knows only
  its own provider's routes
- `next-hop-self` towards the peer router
- a daemon outside the pack's default set (`vrrpd`) called out in a comment
- a `description` on every neighbor, naming the peer and why the session
  exists
