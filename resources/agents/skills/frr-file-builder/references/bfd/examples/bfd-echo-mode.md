# Echo Mode Between FRR Routers

Two FRR routers share a directly connected link, and the request asks for echo mode on the BFD
sessions their IGP creates there: the interface of that link names the profile, with
`ip ospf bfd profile ECHO` or `isis bfd profile ECHO`. Without distributed BFD, echo mode works
only when the peer is also FRR, and it is not supported on multihop sessions, so this profile
serves single-hop sessions toward FRR peers only. With echo mode on, the control packets slow
down to 2000 ms to reduce bandwidth usage. The request gives 100 ms for the echo packets.

A reference for shape. Copy the structure, never the values or names.

```
bfd
 profile ECHO
  echo-mode
  ! echo packets sent no faster than every 100 ms, and received no faster than every 100 ms
  echo transmit-interval 100
  echo receive-interval 100
  ! control packets every 2000 ms at least, to reduce bandwidth while echo mode is on: the peer
  ! declares the session down after 3 x 2000 ms = 6000 ms without control packets,
  ! while its own receive-interval is 2000 ms or less
  transmit-interval 2000
 exit
exit
!
```

What the shape demonstrates:

- echo mode in a profile of its own, apart from any profile of multihop sessions
- the control `transmit-interval` raised once echo mode is on
- the echo intervals written because the request gives them

Sources: bfd › Peer / Profile Configuration; bfd › OSPF BFD Configuration; bfd › IS-IS BFD Configuration.
