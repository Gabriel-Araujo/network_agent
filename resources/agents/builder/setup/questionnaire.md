# Onboarding

Answers configure the agent, not a run. Asked once. Per run facts (ASNs,
prefixes, peers) are collected by stage 01, never here.

Answer everything in one message. Skip anything you do not care about and the
default applies.

1. What FRR version runs in production?
   Default: unknown, which forces long stable syntax only.
   Derived: which version gated features stages 03 and 04 may propose.

2. What is your organization's ASN, or ASN range if you run several?
   Default: none, stage 01 asks per run.
   Derived: whether a session is eBGP or iBGP without asking.

3. Which address blocks does your organization own?
   Default: none, stage 01 asks per run.
   Derived: whether an advertisement is your own space or a leak.

4. Naming convention for policy objects (prefix-lists, route-maps)?
   Example: `PEER-NAME-IN` / `PEER-NAME-OUT`, uppercase with hyphens.
   Default: uppercase with hyphens, direction suffix.

5. Do you run an RPKI validator? Which, and reachable from where?
   Default: no, so stage 03 marks origin validation not applicable.

6. Session authentication preference?
   Options: TCP-AO where supported, MD5 only, keychain by name.
   Default: TCP-AO where the version of record supports it, MD5 otherwise
   with a `SECURITY NOTE`.

7. Where do configs get written relative to the working directory?
   Default: the render stage folder in the agent workspace, never `/etc/frr`.

8. Should stage 03 always pause for approval, even for single parameter
   changes?
   Default: yes. Turning this off removes the only human checkpoint in the pipeline.
