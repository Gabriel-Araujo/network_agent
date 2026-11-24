# LSR On A Network With Many LDP Routers

An LSR has one point-to-point link, `eth1`, and one interface, `eth3`, on a network the request
says many LDP routers share. On `eth3` the additional multicast hellos at session establishment
can produce massive traffic, so they are disabled and hellos go only by interval. The request
also asks for slower hellos on that network, the same on every LDP router of it: one every 10
seconds, with a 30-second holdtime. An adjacency takes the smaller holdtime of its two LSRs, so
the 30 seconds hold only where the neighbors announce 30 too. The IGP that carries the transport
addresses is left out here.

A reference for shape. Copy the structure, never the router-id, addresses, interfaces or timers.

```
! WARNING: label forwarding needs MPLS in the kernel; without it the LDP sessions come up and no label is installed
mpls ldp
 router-id 10.255.0.2
 ! WARNING: no authentication on the LDP sessions of this router
 !
 address-family ipv4
  discovery transport-address 10.255.0.2
  !
  interface eth1
  exit
  !
  interface eth3
   ! a hello every 10 s instead of 5 s: fewer multicast messages on this network
   discovery hello interval 10
   ! adjacencies on eth3 go down after 30 s without a hello instead of 15 s, where the neighbors announce 30 s too
   discovery hello holdtime 30
   ! hellos only by interval, none extra when a session is established
   disable-establish-hello
  exit
  !
 exit-address-family
 !
exit
!
```

What the shape demonstrates:

- `disable-establish-hello` only on the interface the request places on a network with many LDP routers
- hello timers in the `interface` block of that one interface, with the values the request gives
- every timer with a comment stating its expected effect
- the other interface keeping the defaults

Sources: ldpd › LDP Configuration; ldpd › Sample configuration; ldpd › LDP.
