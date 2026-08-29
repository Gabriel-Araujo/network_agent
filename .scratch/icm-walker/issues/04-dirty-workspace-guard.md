# Dirty workspace guard

Type: grilling
Status: open
Blocked by: 03

## Question

Open since the plan was written, §1.3 "Consequence to decide". One workspace
per agent, no run identity in the path, so a second request overwrites the
first run's artifacts, an approved plan included.

Three options are on the table there: refuse to start while the workspace is
non-empty and make the human clear it, wipe silently, or archive to
`.agent/workspace/creator/.archive/<timestamp>/`. The plan recommends refusing
and pointing at `status`, on the grounds that losing an approved plan without
warning is the worst outcome of the three.

`status` is itself unimplemented (Phase 2.1), so decide whether the guard
depends on it or stands alone.
