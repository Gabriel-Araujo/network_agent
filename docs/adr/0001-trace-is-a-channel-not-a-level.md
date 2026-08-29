# TRACE is a file-only channel, not a severity level

The logger has two sinks (console as text, file as JSON) controlled by a single
`LOG_LEVEL`. We needed a way to send bulky debugging information to the file
without flooding the console, and decided that `TRACE` is a **destination
channel**, not a severity: `log.Trace(...)` ignores `LOG_LEVEL`, always goes to
the file and **never** appears in the console.

## Considered Options

The obvious alternative was to treat TRACE as the lowest severity, below Debug,
with a separate `LOG_FILE_LEVEL` controlling the file. It was rejected because
seeing a single trace would mean dropping the global level to `trace`, which
drags every Debug line into the console with it, exactly the noise TRACE exists
to avoid.

## Consequences

TRACE occupies a `slog.Level` value below `LevelDebug` for routing convenience,
and that **misleads**: the numeric ordering suggests a severity, but dispatch is
by channel. With `LOG_LEVEL=info` the file receives Info and above **plus**
every TRACE, without receiving Debug, a gap that looks like a bug and is not.
Any refactor that "fixes" the routing to respect the threshold breaks the
decision.
