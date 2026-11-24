# LDP Checklist

Answer every item against the artifact as it stands: met, not applicable with a reason, or
violated.

## Security

| # | Check | Source |
|---|-------|--------|
| 1 | No `neighbor <lsr-id> password` line: each neighbor the plan authenticates has the left-out warning, or `mpls ldp` has the no-authentication warning | ldpd › LDP Configuration |
| 2 | No secret appears in the artifact | ldpd › LDP Configuration |
| 3 | `ttl-security disable` and `neighbor <lsr-id> ttl-security disable` appear only with a `SECURITY NOTE` naming the implementation that needs them | ldpd › LDP Configuration |
| 4 | `interface <name>` appears in an address-family block only for the interfaces the plan names | ldpd › LDP Configuration |
| 5 | `daemons` enables `zebra`, `ldpd` and the daemons the IGP's pack names, and any other daemon only with a justification in the plan | ospfd › Multi-instance Support; ldpd › Running Ldpd |

## Performance

| # | Check | Source |
|---|-------|--------|
| 6 | `ordered-control` appears only when the plan asks for ordered control or to prevent blackholing during convergence | ldpd › Understanding LDP principles; ldpd › LDP Configuration |
| 7 | `disable-establish-hello` appears only on interfaces the plan places on a network with many LDP routers | ldpd › LDP Configuration |
| 8 | `discovery hello interval`, `discovery hello holdtime` and `neighbor <lsr-id> session holdtime` appear only with values the request gives, each with a `!` comment stating its expected effect | ldpd › LDP Configuration |
| 9 | `dual-stack transport-connection prefer ipv4` appears only on a dual-stack LSR the plan gives TCP over IPv4 | ldpd › LDP Configuration |

## Integrity

| # | Check | Source |
|---|-------|--------|
| 10 | `mpls ldp` has `router-id` | ldpd › LDP Configuration |
| 11 | Every address-family block has a `discovery transport-address` of its own family: on `frr:10.5.0` a family without one forms no session | ldpd › LDP Configuration; ldpd › Sample configuration |
| 12 | The plan names the IGP that carries the transport address, and the artifact, or the file it adds to, carries that IGP | ldpd › Sample configuration |
| 13 | Every `neighbor` line names a neighbor by its LSR-ID | ldpd › Sample configuration |
| 14 | `disable-establish-hello` is in the `interface` block of an address family | ldpd › LDP Configuration |
| 15 | The warning that label forwarding needs MPLS in the kernel stands above `mpls ldp` | ldpd › LDP |
