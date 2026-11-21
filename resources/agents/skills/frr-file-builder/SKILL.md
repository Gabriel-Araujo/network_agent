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
- **The technology's pack**: a folder `references/<slug>/` of this skill, read with
  `read_skill_file` (`skill_name`: `frr-file-builder`). It says how this agent writes the
  technology: shape, syntax rules, policy, and the facts a request must give.

The two do not overlap: the documentation names the commands, the pack decides how they are
used here.

| Slug | Covers |
|------|--------|
| `bgp` | eBGP/iBGP, route reflectors, policy, communities, aggregation, multihoming, BFD bound to a BGP session, MP-BGP IPv4/IPv6 unicast |

## Procedure

1. **Name the technologies.** List every technology the configuration needs: the ones the
   request names and the ones it implies (BGP over an OSPF underlay needs OSPF too; a session
   over an interface the request describes needs that interface).
2. **Read the packs.** For each technology with a row in the table above, read
   `references/<slug>/TECH.md` first: its "Required Facts" says what the request must give, and
   its "Examples" lists the example files, since a folder cannot be listed. Then read
   `references/<slug>/syntax.md`, `references/<slug>/doctrine.md` and the one example from that
   list closest to the request. A technology with no row has no pack: the `<recovered_context>`
   is its only source.
3. **Check the facts.** Collect every identity value the configuration needs and the request
   does not give: each one in a pack's "Required Facts" that blocks the build, and, for any
   technology, any AS number, address, prefix, router-id, interface name, area or VRF name the
   lines would carry. When the content is new lines for an existing file, a value the file
   already states counts as given. If any value is missing, ask for each one by name, one
   question per value with the reason, and stop: produce no configuration. Never invent an
   identity value, and never borrow one from an example or from the documentation.
4. **Check the sources.** Every part of the request must be covered by its pack or by the
   `<recovered_context>`. If a part is covered by neither, say which part and what was missing
   (no pack and no retrieved documentation for it), and stop: produce no configuration. Never
   build a part from memory.
5. **Build.** Apply the rules below, then the syntax and doctrine of each pack. Copy an
   example's structure, never its values.
6. **Warn.** Add every warning the next section and the packs call for.

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
- Order: the header, then interfaces and static routes, then policy objects (prefix-lists,
  route-maps, community-lists), then the routing protocol blocks. A policy object always comes
  before the block that references it.
- Write only what the request asks for and what a pack's doctrine requires: no static route,
  interface or feature beyond them.
- New lines for an existing file go where these rules place them relative to the file's
  blocks, and reuse the names the file already defines (prefix-lists, route-maps, peer-groups).

## Warnings

A warning is a comment line that starts with `! WARNING:`, placed right above the line or
block it concerns. Write one:

- for each security measure a pack's doctrine requires that the request leaves out. Never fill
  the measure with an invented value: the warning takes its place;
- for each technology built without a pack, above its first block:
  `! WARNING: no pack covers <technology>; built from the retrieved FRR documentation only`;
- for each case a pack's syntax or doctrine names.
