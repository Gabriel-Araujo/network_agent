# BGP Labeled-Unicast

Outside this pack's scope per `TECH.md`: ask for a run of the matching
technology. Kept here as the canonical shape for the BGP half of it.

BGP distributes the prefix and its MPLS label in the same NLRI, allowing an
end-to-end LSP across domains without extending the IGP or LDP over the
boundary.

```
router bgp 65001
 bgp router-id 10.0.0.1
 no bgp default ipv4-unicast
 !
 neighbor 10.0.0.2 remote-as 65001
 neighbor 10.0.0.2 update-source lo
 neighbor 10.0.0.2 description internal LSR
 neighbor 203.0.113.1 remote-as 65002
 neighbor 203.0.113.1 description inter-domain ASBR
 !
 ! labeled family: every prefix carries its label
 address-family ipv4 labeled-unicast
  network 10.0.0.1/32
  neighbor 10.0.0.2 activate
  neighbor 10.0.0.2 next-hop-self
  neighbor 203.0.113.1 activate
 exit-address-family
exit
!
! requires kernel MPLS support: modprobe mpls_router
! sysctl net.mpls.platform_labels=100000
! sysctl net.mpls.conf.eth1.input=1
```

What the shape demonstrates:

- a family other than `ipv4 unicast` selected deliberately, with every
  neighbor activated in it
- `next-hop-self` on the iBGP side, so the label stack resolves internally
- kernel prerequisites recorded in the artifact, because the config is
  accepted and silently does nothing without them
