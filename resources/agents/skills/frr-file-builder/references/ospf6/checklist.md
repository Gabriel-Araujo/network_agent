# OSPFv3 Checklist

Answer every item against the artifact as it stands: met, not applicable with a reason, or
violated.

## Security

| # | Check | Source |
|---|-------|--------|
| 1 | Every transit interface has `ipv6 ospf6 authentication keychain`, or the no-authentication warning | ospf6d › Example configuration of keychain:; ospf6d › Authentication trailer support: |
| 2 | No `ipv6 ospf6 authentication key-id` | ospf6d › Manual key: |
| 3 | No secret appears in the artifact: every key chain it defines has no `key-string`, and carries the key string warning instead | ospf6d › Example keychain config: |
| 4 | `ipv6 ospf6 passive` is on every OSPFv3 interface the plan does not name as transit, and on none it names as transit | ospf6d › OSPF6 interface |
| 5 | No `redistribute connected` | ospf6d › Redistribute routes to OSPF6; ospf6d › OSPF6 interface |
| 6 | Every `redistribute` line carries a route-map | ospf6d › Redistribute routes to OSPF6 |
| 7 | `default-information originate always` appears only with a stated reason | ospf6d › Redistribute routes to OSPF6 |
| 8 | `daemons` enables `zebra`, `ospf6d` and the daemons the table in `TECH.md` gives for what the file uses, and any other daemon only with a justification in the plan | ospfd › Multi-instance Support; ospf6d › Showing OSPF6 information |

## Performance

| # | Check | Source |
|---|-------|--------|
| 9 | Every timer, threshold or limit the artifact writes carries a `!` comment with its expected effect | ospf6d › OSPF6 interface; ospf6d › Configuration Commands |
| 10 | Every timer the artifact changes has the value the request gives and a numeric justification in the plan | ospf6d › OSPF6 interface; ospf6d › Configuration Commands |
| 11 | `ipv6 ospf6 bfd` appears only on interfaces where the plan asks for BFD or fast failure detection | bfd › OSPF6 BFD Configuration |
| 12 | Every two-router link has `ipv6 ospf6 network point-to-point` | ospf6d › OSPF6 interface; ospf6d › OSPF6 point-to-point and point-to-multipoint operation |
| 13 | `auto-cost reference-bandwidth` is absent or equal to the domain value the request gave | ospf6d › Configuration Commands |
| 14 | `stub-router administrative` appears only on a router the plan drains for maintenance, and whose transit traffic the plan gives another path | ospf6d › Configuration Commands |

## Integrity

| # | Check | Source |
|---|-------|--------|
| 15 | Every key chain named by `ipv6 ospf6 authentication keychain` and route-map named by `redistribute` or `default-information originate` is defined | ospf6d › Example configuration of keychain:; ospf6d › Redistribute routes to OSPF6 |
| 16 | Every OSPFv3 interface has `ipv6 ospf6 area` | ospf6d › OSPF6 interface |
| 17 | `ospf6 router-id` is present | ospf6d › Configuration Commands |
| 18 | Every area is written dotted, and every `area <area> range` or `area <area> nssa range` names an area this router borders as its ABR | ospf6d › OSPF6 area |
| 19 | `summary-address` appears only on a router that redistributes into OSPFv3 | ospf6d › ASBR Summarisation Support in OSPFv3 |
