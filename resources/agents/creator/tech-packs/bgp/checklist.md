# BGP Checklist

Stage 06 answers every item against the artifact as it stands: met, not
applicable with a reason, or violated.

## Security

| # | Check |
|---|-------|
| 1 | Every session has authentication configured |
| 2 | Where MD5 is used instead of TCP-AO, a `SECURITY NOTE` justifies it |
| 3 | Every eBGP session has an inbound prefix-list |
| 4 | Every eBGP session has an outbound prefix-list |
| 5 | Every eBGP session has `maximum-prefix`, sized to growth |
| 6 | GTSM applied where the topology allowed it, or a reason stated |
| 7 | No `network` statement advertises space the operator does not own |
| 8 | No password or key value appears in the artifact |
| 9 | `daemons` enables only what the plan justified |

## Performance

| # | Check |
|---|-------|
| 10 | Every changed timer carries a `!` comment with its expected effect |
| 11 | Every changed timer has a numeric justification in the plan |
| 12 | BFD present where the topology allowed it, or a reason stated |
| 13 | Aggregation applied where a contiguous advertisable block existed |
| 14 | Every aggregate has an anchor in the local RIB |

## Integrity

| # | Check |
|---|-------|
| 15 | Every referenced prefix-list, route-map, community-list, and keychain is defined |
| 16 | Every neighbor has `activate` in the address family it uses |
| 17 | Every neighbor has a `description` |
| 18 | Every block in the plan appears in the artifact |
| 19 | No block outside the plan appears in the artifact |
