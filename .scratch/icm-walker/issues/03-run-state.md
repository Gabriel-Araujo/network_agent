# Where run state lives and what it holds

Type: grilling
Status: open

## Question

Plan §1.5 says the walker persists pipeline state so the next user message
resumes at the following stage. It does not say where, or what is in it.

Candidates: alongside the artifacts in `.agent/workspace/creator/`, or in
`.agent/config/`, which plan Decision 1 already created to hold the `setup`
answers as `workspace.json`, or not persisted at all and derived by reading
the stage folders.

At stake in the contents: the stage to resume at, the bound `{{TECH}}` slug,
the 05 to 04 retry counter, and whether the run is paused at a gate.

This one unblocks three other things: the dirty workspace guard, the retry
loop, and what the REPL prints when a run pauses.
