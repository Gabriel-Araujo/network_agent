---
name: frr-file-builder
description: "Produces the content of an FRRouting configuration for any FRR technology: the whole integrated file for a new configuration, or the new lines for an existing one, built from the retrieved FRR documentation and the technology's pack."
---

# FRR configuration content

Following this body produces text and nothing else: the content of an FRR configuration. For a
new configuration it is the whole integrated file; for an existing one it is only the lines to
add or change.

## Sources

Every build uses each of these that exists:

- **The `<recovered_context>` block of the turn**: FRR documentation passages retrieved for
  this request. It says which commands exist and how they are written. Read it on every build.
  When the turn carries no such block, the retrieval did not run or failed: build from the
  pack alone.
- **The technology's pack**: a folder `references/<slug>/` of this skill. It says how this
  agent writes the technology: shape, syntax rules, policy, and the facts a request must give.
  The pack of the turn's technology comes after this body. Every other file of a pack is
  found with `read_skill_directory` and read with `read_skill_file` (`skill_name`:
  `frr-file-builder` in both).

The two do not overlap: the documentation names the commands, the pack decides how they are
used here.

| Slug | Covers |
|------|--------|
| `bgp` | eBGP/iBGP, route reflectors, policy, communities, aggregation, multihoming, BFD bound to a BGP session, MP-BGP IPv4/IPv6 unicast |
| `ospf` | OSPFv2: interfaces in areas, passive and transit interfaces, authentication, area types and ABR summarization, redistribution, graceful restart, OSPF in a VRF, BFD bound to an OSPF interface |

## Procedure

1. **Name the technologies.** List every technology the configuration needs: the ones the
   request names and the ones it implies (BGP over an OSPF underlay needs OSPF too; a session
   over an interface the request describes needs that interface).
2. **Read the packs.** The pack of the turn's technology already follows this body:
   `references/<slug>/TECH.md`, `syntax.md` and `doctrine.md`, each inside a `<file>` block
   named by its path. When the turn has no pack, a line after this body says why: no
   technology was classified, or no pack covers it (with the warning to write). A last line
   may say which technology the `<recovered_context>` was retrieved for. `TECH.md`'s
   "Required Facts" says what the request must give. For any other technology of the request
   with a row in the table above, read its `TECH.md`, `syntax.md` and `doctrine.md` with
   `read_skill_file`. A `Read with:` line in a pack's Scope names the packs it depends on: read
   each one named there that has a row in the table, the same way. A technology with no row
   has no pack: the `<recovered_context>` is its only source.
3. **Find the closest shape.** For each pack read, list its folder `references/<slug>/` with
   `read_skill_directory`, then read with `read_skill_file` the file closest to the request,
   from `examples/` or `reference/`. `examples/` holds the shapes `TECH.md`'s "Examples"
   describes; `reference/` holds broader ones. A file that opens saying it is outside the
   pack's scope lends its structure only: that technology still has no pack, so the
   `<recovered_context>` must cover it and its no-pack warning applies. A file that opens
   saying it is not FRR syntax is never a shape for the configuration.
4. **Check the facts.** Collect every identity value the configuration needs and the request
   does not give: each one in a pack's "Required Facts" that blocks the build, and, for any
   technology, any AS number, address, prefix, router-id, interface name, area or VRF name the
   lines would carry. When the content is new lines for an existing file, a value the file
   already states counts as given. If any value is missing, ask for each one by name, one
   question per value with the reason, and stop: produce no configuration. Never invent an
   identity value, and never borrow one from a shape or from the documentation.
5. **Check the sources.** Every part of the request must be covered by its pack or by the
   `<recovered_context>`. If a part is covered by neither, say which part and what was missing
   (no pack and no retrieved documentation for it), and stop: produce no configuration. Never
   build a part from memory.
6. **Build.** Apply the rules below, then the syntax and doctrine of each pack. Copy the
   shape's structure, never its values.
7. **Warn.** Add every warning the next section and the packs call for.

## Rules for every configuration

- A new file opens with the integrated header, then a `!` line:

  ```
  frr defaults traditional
  hostname <router name>
  log syslog informational
  !
  ```

  `hostname` takes the router name the request gives. With no name given, leave the `hostname`
  line out: a router name is never invented.
- One integrated file: never a file per daemon, never `bgpd.conf`.
- Never write `frr version`: the version is a fact about the user's machine that the request
  does not give, and the wrong one changes defaults silently.
- A comment starts with `!`, never with `#`, and takes a line of its own: FRR rejects a `!`
  after a command on the same line. A `!` line separates blocks.
- Order: the header, then the Shared objects (below) and the named objects of a pack such as a
  BGP community-list, then static routes, then `vrf` blocks, then interfaces, then the routing
  protocol blocks. Every object then comes before the block that references it.
- One block per object: an interface, a `vrf` or a `router` block appears once and carries
  the lines of every pack that has lines in it. One `interface eth1` holds its OSPF and BFD
  lines together; one `router bgp` holds every address-family inside it.
- This body writes the headers of the blocks packs share, whenever a pack has a line inside
  them: `interface <name>` … `exit`, `interface <name> vrf <vrf>` for an interface in a VRF,
  and `vrf <name>` … `exit-vrf`. Inside an interface the addresses come first, then the lines
  of each pack, grouped by pack. `ip address` and `ipv6 address` carry only the address the
  request gives for that interface; with none given, write no address line and ask nothing, as
  no protocol line needs it.
- A static route is written only when the request asks for it or a pack's doctrine requires
  one.
- Write only what the request asks for and what a pack's doctrine requires: no interface or
  feature beyond them.
- New lines for an existing file go where these rules place them relative to the file's
  blocks, and reuse the names the file already defines (prefix-lists, route-maps, key chains,
  peer-groups).

## Shared objects

A Shared object configures no protocol, and lines of several packs reference it: a
prefix-list (`ip` or `ipv6`), an access-list, a route-map, a key chain. This body says how
each is written; the pack whose line references it says when to use it. An object only one
protocol uses belongs to that protocol's pack, such as a BGP community-list.

- They come in this order: access-lists and prefix-lists, then route-maps, then key chains.
- Every prefix-list and access-list entry carries an explicit `seq`, and every route-map entry
  its number.
- A prefix-list or an access-list ends in an implicit deny: a prefix no entry permits is
  denied. A list meant to let the rest through ends with an explicit permit.
- A prefix-list entry matches the exact prefix length unless `le` or `ge` bound a range of
  lengths.
- Every object a line references is defined in the file. An object the file already defines
  is reused by its name, never defined twice.

```
access-list <NAME> seq 5 permit <prefix>
ip prefix-list <NAME> seq 5 permit <prefix> le <length>
!
route-map <NAME> permit 10
 match ip address prefix-list <NAME>
exit
!
key chain <NAME>
 key 1
  cryptographic-algorithm <algorithm>
  ! WARNING: key 1 of <NAME> has no key-string; set it on the router, never in this file
 exit
exit
!
```

### Secrets

A secret is never written. FRR cannot point at a secret kept outside `frr.conf`, so any value
the file carried would be the secret itself.

- A key chain is written without `key-string`, and each of its keys carries
  `! WARNING: key <N> of <chain> has no key-string; set it on the router, never in this file`.
- A line whose argument is the secret is left out: `neighbor <peer> password`,
  `ip ospf authentication-key`, `ip ospf message-digest-key`, `isis password`,
  `area-password`, `password`, `enable password`. In its place a warning names the line the
  operator adds on the router: `! WARNING: <line> is left out; add it on the router, never in this file`.
- A secret has no placeholder either (`<SECRET>`, `changeme`): `vtysh` takes the word as the
  secret.

## Warnings

A warning is a comment line that starts with `! WARNING:`, placed right above the line or
block it concerns. Write one:

- for each security measure a pack's doctrine requires that the request leaves out. Never fill
  the measure with an invented value: the warning takes its place;
- for each secret the configuration needs, as Secrets above says;
- for each technology built without a pack, above its first block:
  `! WARNING: no pack covers <technology>; built from the retrieved FRR documentation only`;
- for each case a pack's syntax or doctrine names.
