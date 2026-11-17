# Anycast With ExaBGP And A Health Check

The advertisement is tied to the result of a service health check: when the
check fails the process withdraws the prefix and traffic falls back to the
other replicas in under a second.

Not FRR syntax. ExaBGP configuration.

```
# /etc/exabgp/exabgp.conf
process healthcheck {
    run /usr/bin/python3 -m exabgp healthcheck --cmd "curl -sf http://localhost:8080/health" --label anycast --interval 2 --fast-interval 1 --rise 3 --fall 2;
    encoder text;
}

neighbor 10.0.0.100 {
    router-id 10.1.0.7;
    local-address 10.1.0.7;
    local-as 65107;
    peer-as 65100;
    hold-time 9;

    family {
        ipv4 unicast;
    }

    api {
        processes [ healthcheck ];
    }
}

# the anycast VIP 10.1.0.53/32 lives on lo; the healthcheck advertises/withdraws the /32
# FRR equivalent: "network 10.1.0.53/32" + ip addr add/del 10.1.0.53/32 dev lo
```

What the shape demonstrates:

- the advertisement follows service health, not interface state, which is
  the whole point of the pattern
- `rise`/`fall` asymmetric: slow to advertise, fast to withdraw
- the VIP on the loopback, so the route is the only thing that moves
- the FRR equivalent named, since the pack renders FRR: this file is the
  reference for what the FRR side has to reproduce
