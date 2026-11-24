# Dual-Stack LSR

An LSR runs LDP for IPv4 and for IPv6 on both its transit links. Each address family has its own
transport address. By default a dual-stack LSR makes its transport connections over IPv6 and
refuses them over IPv4; the request asks for TCP over IPv4, so label mappings for IPv6 FECs can also
go over TCPv4. The IGPs that carry the transport addresses are left out here.

A reference for shape. Copy the structure, never the router-id, addresses or interfaces.

```
! WARNING: label forwarding needs MPLS in the kernel; without it the LDP sessions come up and no label is installed
mpls ldp
 router-id 10.255.0.2
 ! WARNING: no authentication on the LDP sessions of this router
 dual-stack transport-connection prefer ipv4
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
 address-family ipv6
  discovery transport-address 2001:db8::2
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

- both address-family blocks, each with a transport address of its own family and its own `interface` lines
- `dual-stack transport-connection prefer ipv4` only because the request asks for TCP over IPv4
- GTSM left enabled in both families

Sources: ldpd › LDP Configuration; ldpd › Sample configuration; ldpd › LDP.
