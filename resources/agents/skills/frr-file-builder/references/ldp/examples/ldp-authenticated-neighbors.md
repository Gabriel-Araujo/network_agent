# Authenticated LDP Neighbors

An LSR forms LDP sessions with two neighbors the request names by their LSR-IDs, `10.255.0.1` and
`10.255.0.3`, and asks to authenticate both. The password is a secret, so each `password` line is
left out and a warning names the line the operator adds on the router. The request also gives
both sessions a holdtime of 90 seconds. The IGP that carries the transport addresses is left out
here.

A reference for shape. Copy the structure, never the router-id, LSR-IDs, addresses or interfaces.

```
! WARNING: label forwarding needs MPLS in the kernel; without it the LDP sessions come up and no label is installed
mpls ldp
 router-id 10.255.0.2
 ! WARNING: neighbor 10.255.0.1 password is left out; add it on the router, never in this file
 ! the session with 10.255.0.1 is considered down after 90 s of non response
 neighbor 10.255.0.1 session holdtime 90
 ! WARNING: neighbor 10.255.0.3 password is left out; add it on the router, never in this file
 ! the session with 10.255.0.3 is considered down after 90 s of non response
 neighbor 10.255.0.3 session holdtime 90
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

- each neighbor named by its LSR-ID, the router-id of its `mpls ldp`
- one left-out warning per authenticated neighbor, never the password itself
- the session holdtime only with the value the request gives, with a comment stating its effect

Sources: ldpd › LDP Configuration; ldpd › Sample configuration; ldpd › Show LDP Information; ldpd › LDP.
