# IS-IS Checklist

Answer every item against the artifact as it stands: met, not applicable with a reason, or
violated.

## Security

| # | Check | Source |
|---|-------|--------|
| 1 | Every transit interface carries the `isis password md5` warning | isisd › ISIS interface |
| 2 | `router isis` carries the `area-password md5` warning when it runs level-1, and the `domain-password md5` warning when it runs level-2 | isisd › ISIS router; isisd › ISIS Timer |
| 3 | No secret appears in the artifact: no `isis password`, `area-password` or `domain-password` line | isisd › ISIS interface; isisd › ISIS router |
| 4 | `isis passive` is on every IS-IS interface the plan does not name as transit, and on none it names as transit | isisd › ISIS interface |
| 5 | `default-information originate ... always` appears only with a stated reason | isisd › ISIS Fast-Reroute |
| 6 | `daemons` enables `isisd` and *zebra* runs with it; any other daemon only with a justification in the plan | isisd › Configuring isisd; ospfd › Multi-instance Support |

## Performance

| # | Check | Source |
|---|-------|--------|
| 7 | Every changed timer carries a `!` comment with its expected effect | isisd › ISIS interface; isisd › ISIS Timer; isisd › ISIS router |
| 8 | Every changed timer has a numeric justification in the plan | isisd › ISIS interface; isisd › ISIS Timer |
| 9 | BFD present on transit interfaces where the topology allowed it, or a reason stated | bfd › IS-IS BFD Configuration |
| 10 | Every two-router link has `isis network point-to-point` | isisd › ISIS interface |
| 11 | `metric-style` is absent, or `transition` with the move between styles stated in the plan | isisd › ISIS router |
| 12 | `set-overload-bit` without `on-startup` appears only for a drain where no transit traffic may cross the router | isisd › ISIS router |

## Integrity

| # | Check | Source |
|---|-------|--------|
| 13 | Every `ip router isis` and `ipv6 router isis` names the instance of a `router isis` block in the VRF of its interface | isisd › ISIS interface; isisd › ISIS router; isisd › ISIS Vrf Configuration Examples |
| 14 | At most one `router isis` in the default VRF: as of this writing, *isisd* does not support multiple IS-IS processes | isisd › ISIS router |
| 15 | Every `router isis` has `net` and `is-type` | isisd › ISIS router; isisd › ISIS region |
| 16 | `isis circuit-type` appears only on a level-1-2 router | isisd › ISIS interface; isisd › ISIS region |
| 17 | `isis bfd profile` appears only on an interface that also has `isis bfd` | bfd › IS-IS BFD Configuration; isisd › ISIS interface |
| 18 | Every route-map named by `default-information originate` is defined | isisd › ISIS Fast-Reroute |
