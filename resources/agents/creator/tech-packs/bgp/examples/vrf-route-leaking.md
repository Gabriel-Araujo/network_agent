# Route Leaking Between VRFs

Outside this pack's scope per `TECH.md`: ask for a run of the matching
technology. Kept here as the canonical shape for the BGP half of it.

Shares common services (DNS, monitoring, internet egress) between isolated
VRFs, without an external firewall hop and without duplicating the table.

```
vrf CLIENTE-A
exit-vrf
vrf SERVICOS
exit-vrf
!
router bgp 65000 vrf CLIENTE-A
 address-family ipv4 unicast
  ! import the routes of the shared services VRF
  import vrf SERVICOS
  ! filter what comes in: the services block only, nothing else
  import vrf route-map RM-IMPORT-SERVICOS
 exit-address-family
exit
!
router bgp 65000 vrf SERVICOS
 address-family ipv4 unicast
  redistribute connected
 exit-address-family
exit
!
ip prefix-list PL-SERVICOS seq 10 permit 10.99.0.0/24
route-map RM-IMPORT-SERVICOS permit 10
 match ip address prefix-list PL-SERVICOS
exit
!
! verification: show ip route vrf CLIENTE-A
```

What the shape demonstrates:

- the leak is one-directional and declared on the importing side
- `import vrf route-map` always present next to `import vrf`: an unfiltered
  import merges the two VRFs and destroys the isolation that justified them
- `redistribute connected` scoped to the services VRF only
- verification done in the routing table of the importing VRF, where the
  result is visible
