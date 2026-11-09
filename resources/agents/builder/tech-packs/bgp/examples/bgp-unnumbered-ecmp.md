# BGP Unnumbered With ECMP On A Linux Host

Interface sessions over IPv6 link-local with an IPv4 next-hop (RFC 5549), no
point-to-point addressing at all. The kernel has to be tuned for multipath
and forwarding to actually work.

```
! ---- /etc/sysctl.d/90-frr.conf ----
! net.ipv4.conf.all.forwarding=1
! net.ipv6.conf.all.forwarding=1
! net.ipv4.fib_multipath_hash_policy=1   (hash on the 5-tuple, not the IP alone)
! net.ipv4.conf.all.rp_filter=0
!
interface lo
 ip address 10.0.0.7/32
exit
!
router bgp 65107
 bgp router-id 10.0.0.7
 no bgp default ipv4-unicast
 bgp bestpath as-path multipath-relax
 !
 neighbor TOR peer-group
 neighbor TOR remote-as external
 neighbor TOR bfd
 neighbor eth0 interface peer-group TOR
 neighbor eth1 interface peer-group TOR
 !
 address-family ipv4 unicast
  network 10.0.0.7/32
  neighbor TOR activate
  neighbor TOR route-map RM-OUT out
  neighbor TOR route-map RM-IN in
  maximum-paths 2
 exit-address-family
exit
!
ip prefix-list PL-ME seq 10 permit 10.0.0.7/32
route-map RM-OUT permit 10
 match ip address prefix-list PL-ME
exit
route-map RM-IN permit 10
exit
```

What the shape demonstrates:

- sysctls recorded in the artifact, because BGP installs correct multipath
  routes that the kernel then ignores without them
- `multipath-relax` plus `maximum-paths`, the pair that produces ECMP across
  distinct peer ASNs
- an outbound policy bounded to the host's own /32, so the host never
  becomes transit between two ToRs
- `rp_filter=0`, required once traffic can legitimately arrive asymmetrically
