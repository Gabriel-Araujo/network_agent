# BFD Checklist

Answer every item against the artifact as it stands: met, not applicable with a reason, or
violated.

## Security

| # | Check | Source |
|---|-------|--------|
| 1 | `multihop` appears only on a `peer` the plan places more than one hop away | bfd › BFDd Commands |
| 2 | `minimum-ttl` appears only for configured multihop peers that no Binding line also uses, with the value the plan gives | bfd › Peer / Profile Configuration |
| 3 | `daemons` enables `bfdd` only when the artifact has a Binding line or a `peer`, and any other daemon only per the table in `TECH.md` or with a justification in the plan | ospfd › Multi-instance Support; bfd › BFDd Commands |

## Performance

| # | Check | Source |
|---|-------|--------|
| 4 | Every changed control-packet timer carries a `!` comment with the detection time it gives, and every changed echo timer one with its interval | bfd › Peer / Profile Configuration |
| 5 | Every changed timer has a numeric justification in the plan | bfd › Peer / Profile Configuration |
| 6 | `echo-mode` appears only for single-hop sessions toward FRR peers | bfd › Peer / Profile Configuration |
| 7 | Every profile or `peer` with `echo-mode` carries a raised `transmit-interval` | bfd › Peer / Profile Configuration |
| 8 | `passive-mode` appears only on the central node of a star network | bfd › Peer / Profile Configuration |

## Integrity

| # | Check | Source |
|---|-------|--------|
| 9 | Every profile a Binding line or a `peer` names is defined in the `bfd` block | bfd › BFD Peer Specific Commands |
| 10 | Every setting of a profile or `peer` carries a value the plan gives or doctrine fixes, and no profile is empty | bfd › Peer / Profile Configuration |
| 11 | No `ip ospf bfd` line carries a multiplier or intervals | bfd › OSPF BFD Configuration; ospfd › Interfaces |
| 12 | Every `peer` is one the plan names; every multihop or IPv6 `peer` carries `local-address` | bfd › BFDd Commands; bfd › Configuration |
