# MPLS L3VPN With VPNv4 And VRF

Outside this pack's scope per `TECH.md`: ask for a run of the matching
technology. Kept here as the canonical shape for the BGP half of it.

BGP carries the customer routes with a route distinguisher and route target;
MPLS carries the packets. The core IGP knows no customer route at all.

```
! customer VRF in the data plane
vrf CLIENTE-A
exit-vrf
!
interface eth2
 ! interface facing the CE
 ip address 10.20.0.1/30
exit
!
! ---- BGP instance for the VRF ----
router bgp 65000 vrf CLIENTE-A
 no bgp default ipv4-unicast
 neighbor 10.20.0.2 remote-as 65100
 neighbor 10.20.0.2 description CE / customer A
 !
 address-family ipv4 unicast
  neighbor 10.20.0.2 activate
  ! export into the VPN table with RD/RT and a label
  rd vpn export 65000:100
  rt vpn both 65000:100
  label vpn export auto
  export vpn
  import vpn
 exit-address-family
exit
!
! ---- global instance: VPNv4 session with the remote PE ----
router bgp 65000
 bgp router-id 10.0.0.1
 no bgp default ipv4-unicast
 neighbor 10.0.0.2 remote-as 65000
 neighbor 10.0.0.2 update-source lo
 neighbor 10.0.0.2 description remote PE / VPNv4
 !
 address-family ipv4 vpn
  neighbor 10.0.0.2 activate
 exit-address-family
exit
```

What the shape demonstrates:

- two BGP instances in one file: the VRF one peers with the CE, the global
  one peers with the remote PE
- RD, RT and label export declared inside the VRF address family
- `export vpn` / `import vpn` as the pair that connects the VRF table to the
  VPN table
- the PE-PE session activated in `ipv4 vpn`, not in `ipv4 unicast`
