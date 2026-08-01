# System Prompt

## V1

- Adição de Skills

## V1b

1. Fixed duplicate `3.1` heading (Tools vs. RAG grounding); RAG is now `3.2`, Skills is `3.3`.
2. Unified the "only permitted stop" language — BLOCKED and the §7 security carve-out are now both named together, resolving the earlier contradiction between §1.1.5 and §4.
3. Grounded the previously-dangling "steps marked OPTIONAL" clause with a concrete rule: single-parameter, no-impact changes may abbreviate P3 PLAN, with mandatory inline declaration.
4. Added §1.1 request classification and two new pipelines — §4.2 (diagnostics) and §4.3 (audit/review) — plus explicit exemption for pure informational questions (Classification D) and a rule for mixed requests (§4.4).
5. Added an explicit no-execution boundary in §3.1: the agent never runs commands on a live device; all operator commands are text output only.
6. Added a precedence hierarchy in §3.2 for conflicts between user-stated environment facts, RAG, and prior knowledge.
7. Differentiated rollback guidance in §4.1 P6 between `frr.conf` hot-reload and `daemons`/`vtysh.conf` restart (session-drop impact differs).
8. Added a citation format standard (`CITATIONS:` block) and an explicit note that pipeline verbosity is exempt from the "terse" response-style rule, plus a worked example of the TODO/status format.
