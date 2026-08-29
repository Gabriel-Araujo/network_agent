# The shape of an assembled prompt

Type: prototype
Status: open

## Question

Plan §1.2 gives `SystemPrompt(shared)` and `UserTurn(wsDir)` twenty lines each
and one instruction about form: Layer 4 artifacts get "wrapped in a named tag
so the model can tell the artifacts apart". Nothing else is decided. Order,
delimiters, whether Layer 3 files are labelled at all, how a section-scoped
excerpt announces that it is an excerpt.

`.agent/config/workspace.json` is special-cased into the system prompt next to
`shared/execution-rules.md`, per plan Decision 1, so it needs a place in the
shape too.

Build the concrete artifact: the fully assembled system prompt and user turn
for `04-render`, which exercises every scope token that matters, and for
`01-intent`, which is the simplest. Real file contents, not a sketch. React to
those, then the byte-for-byte table test in the destination has something to
pin.
