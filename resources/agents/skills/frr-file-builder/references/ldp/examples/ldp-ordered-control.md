# Ordered Label Distribution Control

The LSRs of this network bind labels in ordered control: the request prefers no blackholing
during LDP convergence over a faster convergence. An LSR then binds a label to a FEC only as its
egress, or once its next hop has sent a label for it. The IGP that carries the transport
addresses is left out here.

A reference for shape. Copy the structure, never the router-id, addresses or interfaces.

```
! WARNING: label forwarding needs MPLS in the kernel; without it the LDP sessions come up and no label is installed
mpls ldp
 router-id 10.255.0.2
 ! WARNING: no authentication on the LDP sessions of this router
 ordered-control
 !
 address-family ipv4
  discovery transport-address 10.255.0.2
  !
  interface eth1
  exit
  !
  interface eth2
  exit
  !
 exit-address-family
 !
exit
!
```

What the shape demonstrates:

- `ordered-control` in `mpls ldp`, only because the request asks to prevent blackholing during convergence
- the rest of the block as in any LSR: discovery per interface, the transport address per address family

Sources: ldpd › Understanding LDP principles; ldpd › LDP Configuration; ldpd › Sample configuration; ldpd › LDP.
