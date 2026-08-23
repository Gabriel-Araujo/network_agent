# BGP Doctrine

Default posture is deny. Stage 03 walks this list item by item and gives each
one a verdict: met, not applicable with a reason, or regression.

## Security

| Rule | Note |
|------|------|
| Authentication on every session | TCP-AO where the version of record supports it, otherwise MD5 as a legacy fallback flagged with a `SECURITY NOTE` |
| Inbound prefix-list on every eBGP session | No exceptions |
| Outbound prefix-list on every eBGP session | A session with no outbound filter is a regression even if the user only asked about inbound |
| `maximum-prefix` on every eBGP session | Sized to expected table growth, not to today's table |
| GTSM where the topology allows | TTL security |
| RPKI origin validation where a validator exists | See the onboarding answer |
| RFC 9234 roles and OTC | Only when the version of record supports them |
| No `network` for space the operator does not own | |
| Secrets by reference | Keychain or password name. The value never appears in the artifact or the conversation |

## Control Plane Minimization

Only the daemons the task requires, per the table in `TECH.md`. Unused
services stay disabled.

## Security Regression

Removing authentication, opening a filter to permit-any, or disabling control
plane protection is fulfilled only after a `SECURITY WARNING` banner and
explicit user confirmation. It is the one thing that halts the pipeline
outside a declared checkpoint.

## Performance

| Rule | Note |
|------|------|
| BFD for liveness where the topology allows | |
| Timer tuning only with numeric justification | State the trade-off, not just the value |
| Graceful restart where appropriate | |
| Prefix aggregation | Each aggregate needs an anchor in the local RIB |
| Route reflector design over full mesh at scale | |
| ECMP via `maximum-paths` | |
| Prefix-lists rather than per-prefix ACLs | |
| Conservative dampening per RFC 7196 | |

Every timer, threshold, or limit carries a `!` comment stating its expected
effect.
