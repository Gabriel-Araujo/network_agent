# Required Facts

What stage 01 must ask about, and what it may fill in. An identity fact is
never guessed; an operational preference may take a default.

## Always Ask

| Missing | Question | Why |
|---------|----------|-----|
| Local ASN | What is the ASN of the router receiving this config? | Without it there is no `router bgp` |
| Peer ASN | What is the ASN of each peer? | Decides eBGP vs iBGP and the applicable policy |
| Peer address | What is the address of each session? | Without it there is no `neighbor` |
| Own prefixes | Which blocks does this AS advertise? | Advertising space you do not own is an incident |
| Authentication | Does the session use a password or keychain? Under what name? | Doctrine requires authentication, and the name is a reference, not the value |
| FRR version | Which version runs in production? | Governs version gated features |

Skip a row when `setup/questionnaire.md` already answered it at the workspace
level. Ask only for what is still unknown at run time.

## Ask When the Request Implies It

| Missing | Question |
|---------|----------|
| iBGP role | Is this router a route reflector, a client, or full mesh? |
| Multihoming | Is there more than one upstream? Which is primary? |
| VRF | Does the session live in a VRF? Which one? |
| Interface | Is the session over a physical interface, a loopback, or unnumbered? |

## May Become an Assumption

Record in `brief.json.assumptions` with the default adopted, then proceed.
Nothing here carries security impact or invents an identity.

- prefix-list, route-map, and community-list names
- `description` text
- timer values, where doctrine does not fix one
- `bgp router-id`, when an obvious loopback appears in material already read

## Question Form

One question per item, direct, with the reason attached.

```
What is the ASN of `peer-upstream`?
why: decides whether the outbound policy is transit or peering, and stage 04
cannot emit `remote-as` without it.
```

Never ask in bulk ("send me the network details").
