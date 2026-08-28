# MetalLB In BGP Mode On Kubernetes

Cluster nodes hold BGP sessions with the ToRs and advertise LoadBalancer
`Service` IPs as /32s, only from the nodes that have healthy endpoints.

Not FRR syntax. Kubernetes manifests.

```yaml
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: pool-servicos
  namespace: metallb-system
spec:
  addresses:
    - 10.1.0.0/24
---
apiVersion: metallb.io/v1beta2
kind: BGPPeer
metadata:
  name: tor1
  namespace: metallb-system
spec:
  myASN: 65200
  peerASN: 65100
  peerAddress: 10.0.0.101
  # BFD speeds up node failure detection
  bfdProfile: fabric
---
apiVersion: metallb.io/v1beta1
kind: BGPAdvertisement
metadata:
  name: adv-servicos
  namespace: metallb-system
spec:
  ipAddressPools:
    - pool-servicos
  aggregationLength: 32
  communities:
    - 65000:100
# on the ToR: maximum-prefix and an inbound filter restricted to 10.1.0.0/24
```

What the shape demonstrates:

- the peer is unmanaged by this artifact: the ToR side still needs its own
  inbound policy and `maximum-prefix`
- a community set at the source, so the ToR policy has something to match
- `aggregationLength: 32`, so service IPs are advertised individually and
  follow endpoint health
- the pool bounded to one block, which is what the ToR filter mirrors
