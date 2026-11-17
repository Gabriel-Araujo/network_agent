# iBGP Route Reflector

Removes the full mesh of N x (N-1)/2 sessions. Clients talk only to the
reflectors; the reflectors reflect between clients, preserving
`originator-id` and `cluster-list` to prevent loops.

```
! ---- on the reflector ----
router bgp 65000
 bgp router-id 10.0.0.254
 bgp cluster-id 10.0.0.254
 no bgp default ipv4-unicast
 !
 neighbor CLIENTS peer-group
 neighbor CLIENTS remote-as 65000
 neighbor CLIENTS update-source lo
 neighbor 10.0.0.1 peer-group CLIENTS
 neighbor 10.0.0.2 peer-group CLIENTS
 neighbor 10.0.0.3 peer-group CLIENTS
 !
 address-family ipv4 unicast
  neighbor CLIENTS activate
  neighbor CLIENTS route-reflector-client
  ! preserve multiple paths towards the clients (RFC 7911)
  neighbor CLIENTS addpath-tx-all-paths
 exit-address-family
exit
!
! ---- on the client: an ordinary iBGP session, nothing special ----
router bgp 65000
 neighbor 10.0.0.254 remote-as 65000
 neighbor 10.0.0.254 update-source lo
 address-family ipv4 unicast
  neighbor 10.0.0.254 activate
 exit-address-family
exit
```

What the shape demonstrates:

- `route-reflector-client` inside the address family, never at the global
  level
- an explicit `bgp cluster-id`, so a second reflector can share it
- the client side is a plain iBGP session: the asymmetry lives entirely on
  the reflector
- add-path advertised deliberately, because reflection otherwise hides every
  path but the best one
