# Tech pack router: enum, prompt and UNROUTABLE

Type: grilling
Status: open

## Question

Plan §1.6: one LLM call at run start, the enum from `ls tech-packs/`, the
descriptions from the first line of each `TECH.md`, following the pattern in
`skills.BuildTools()`. Returns one slug, or `UNROUTABLE`.

Three things are undecided.

Where its own prompt lives. `resources/agents/router/` is now taken by the
agent router, which routes between `creator` and `general` and is out of scope
for this map. Two routers, two layers, one name.

What `UNROUTABLE` does. The run cannot proceed without a bound `{{TECH}}`,
since four contracts name it. Refuse, ask the human, or fall back to the only
installed pack.

Whether it runs at all with one pack installed. The plan says ship it anyway
so Phase 3 needs no code change, which is defensible, but a model call that
can only return one value is also a model call that can return the wrong one.
