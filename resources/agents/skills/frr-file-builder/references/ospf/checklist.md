# OSPF Checklist

Answer every item against the artifact as it stands: met, not applicable with a reason, or
violated.

## Security

| # | Check | Source |
|---|-------|--------|
| 1 | Every transit interface has `ip ospf authentication key-chain`, or the no-authentication warning | ospfd › Interfaces |
| 2 | No `ip ospf authentication-key`, no `ip ospf message-digest-key`, no `area <area> authentication` | ospfd › Interfaces; ospfd › Areas |
| 3 | No secret appears in the artifact: every key chain it defines has no `key-string`, and carries the key string warning instead | ospfd › Interfaces |
| 4 | `passive-interface default` is present, and `no ip ospf passive` appears only on interfaces the plan names as transit | ospfd › Routers; ospfd › Interfaces |
| 5 | No `redistribute connected` | ospfd › Redistribution; ospfd › Interfaces |
| 6 | Every `redistribute` line carries a route-map | ospfd › Redistribution |
| 7 | `default-information originate always` appears only with a stated reason | ospfd › Redistribution |
| 8 | `ospf rfc1583compatibility` appears only with a stated reason; `no compatible rfc7474` only with a `SECURITY NOTE` naming the peer implementation | ospfd › Routers |
| 9 | `daemons` enables `zebra` and `ospfd`, and any other daemon only with a justification in the plan | ospfd › Configuring OSPF; ospfd › Multi-instance Support |

## Performance

| # | Check | Source |
|---|-------|--------|
| 10 | Every changed timer carries a `!` comment with its expected effect | ospfd › Interfaces; ospfd › Routers |
| 11 | Every changed timer has a numeric justification in the plan | ospfd › Interfaces; ospfd › Routers |
| 12 | BFD present on transit interfaces where the topology allowed it, or a reason stated | bfd › OSPF BFD Configuration |
| 13 | Every two-router link has `ip ospf network point-to-point` | ospfd › Interfaces; ospf_fundamentals › Link-State LSAs |
| 14 | `auto-cost reference-bandwidth` is absent or equal to the domain value the request gave | ospfd › Routers |

## Integrity

| # | Check | Source |
|---|-------|--------|
| 15 | Every key chain named by `ip ospf authentication key-chain`, prefix-list named by `ip ospf neighbor-filter` and route-map named by `redistribute` or `default-information originate` is defined | ospfd › Interfaces; ospfd › Redistribution |
| 16 | Every OSPF interface has `ip ospf area` | ospfd › Interfaces |
| 17 | No `network ... area` line in `router ospf` | ospfd › Routers |
| 18 | `ospf router-id` is present | ospfd › Routers |
| 19 | Every area is written dotted, and every `area <area> range` or `area <area> nssa range` names an area this router borders as its ABR | ospfd › Areas |
| 20 | `ip ospf bfd profile` appears only on an interface that also has `ip ospf bfd` | bfd › OSPF BFD Configuration |
