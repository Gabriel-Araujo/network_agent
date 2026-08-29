# The script stage wrapper contract

Type: grilling
Status: open
Blocked by: 03

## Question

Charting Q5 settled that the walker wraps the existing skills instead of
changing their signatures. What the wrapper actually promises is open.

`ragretriever.Do(ctx, briefingPath, cfg) (string, error)` writes to
`.agent/tmp/retrieval` and returns the path. The `02-evidence` contract
declares `evidence.json` in the stage folder, and its Degradation section
requires that a failed or empty retrieval writes
`{"chunks": [], "degraded": true, "reason": "..."}` and lets the run continue.
Today the caller in `cmd/cli/main.go` logs a warning and drops the context.

`05-validate` dispatches to `internal/frr/validate.ValidateConfig`, which
returns an error when `vtysh` is absent. Plan Decision 2 says that degrades:
`report.json` records `syntax: unchecked`, the run continues, and stage 06
carries the risk plus the command the operator runs by hand.

So: what the walker passes in, where it moves the output, and who writes the
degraded artifact when the skill returns an error rather than a result.
