# BGP Doctrine

Default posture is deny. Each rule below is met, not applicable for a stated reason, or a
regression.

## Session Policy

What every BGP configuration carries, whatever else the request asks for.

- A neighbor whose `remote-as` differs from the local AS is **eBGP**, and gets a prefix-list in
  each direction.
  - The outbound list permits each prefix the request advertises and ends in `deny any`.
  - The inbound list permits what the request says to accept. When the request says nothing,
    it is an explicit `deny any`, with a warning above it:
    `! WARNING: nothing is accepted from <peer address> until an inbound policy says what to accept`.
- Never write `no bgp ebgp-requires-policy`.
- A neighbor inside the local AS is **iBGP**: write no prefix-list for it. `activate` and
  `description` apply to every neighbor, eBGP or iBGP.
- Authentication: a password or keychain the request names is referenced by that name. When
  the request names none, write no password and put a warning under the neighbor's
  `description`: `! WARNING: no authentication on the session with <peer address>`.
- `maximum-prefix` on every eBGP session: a limit the request gives is written with a `!`
  comment stating its effect. When the request gives none, write no number and put a warning
  in the address-family: `! WARNING: no maximum-prefix on <peer address>`.

The rest of the tables below apply when the request asks for the feature or gives what it
needs; absent, they add neither a line nor a warning.

## Security

| Rule | Note |
|------|------|
| Authentication on every session | TCP-AO where the FRR version in use supports it, otherwise MD5 as a legacy fallback flagged with a `SECURITY NOTE` |
| Inbound prefix-list on every eBGP session | No exceptions |
| Outbound prefix-list on every eBGP session | A session with no outbound filter is a regression even if the user only asked about inbound |
| `maximum-prefix` on every eBGP session | Sized to expected table growth, not to today's table |
| GTSM where the topology allows | TTL security |
| RPKI origin validation where a validator exists | |
| RFC 9234 roles and OTC | Only when the FRR version in use supports them |
| No `network` for space the operator does not own | |
| Secrets by reference | Keychain or password name. The value never appears in the artifact or the conversation |

## Control Plane Minimization

Only the daemons the task requires, per the table in `TECH.md`. Unused
services stay disabled.

## Security Regression

Removing authentication, opening a filter to permit-any, or disabling control
plane protection is built only after a `SECURITY WARNING` and explicit user
confirmation. Without the confirmation, nothing is built.

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
