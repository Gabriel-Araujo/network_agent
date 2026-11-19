# Own surfaces take their tokens from S2, not from the prototype's CSS

The frontend's visual reference is the prototype (`docs/FRONTEND_PROTOTIPO.html`), but only its
**layout** is carried over: structure and measures (titlebar 48, sidebar 244, collapsed strip 56,
Logs 360, status bar 30). Colour, type, radius, spacing and motion on every own surface — the shell,
composer, tab bar, conversation list, Markdown, code block, diagram canvas, Logs drawer, status bar
— come from React Spectrum S2's tokens through the `style` macro. The prototype's custom properties
(`--bg-panel`, `--text-dim`, its `html[data-theme="dark"]` palette) and its keyframes are not used.
`App.css` and `Settings.css` are deleted with no replacement stylesheet for surfaces.

We chose this so the app has one palette, not two. S2 compiles its colours to literal `light-dark()`
values at build time and exposes no palette variables, so a second palette for own surfaces would
have to be kept in step with S2 by hand, in both themes, beside S2 controls it sits next to. With
the macro, light/dark is just the `Provider`'s `colorScheme`, and an own surface cannot drift from
the controls around it.

## Considered Options

**The prototype's CSS for own surfaces, S2 only for standard controls.** It is what the map first
said, and it is pixel-faithful to the prototype. Rejected because it means two palettes and two
dark themes: an own panel at the prototype's `#202125` next to an S2 popover at `layer-1`
`rgb(27, 27, 27)`, 13 px own text next to 14 px S2 text — every mismatch visible, and each S2
upgrade a manual re-sync.

## Consequences

- The app does not match the prototype's hex values or its 13 px base size; each prototype colour
  maps to the nearest S2 token, and text is S2's 14 px.
- The only global CSS left holds the `@font-face` rules, the layer order and the reduced-motion
  rule. Markdown elements are styled one macro class per element through `react-markdown`'s
  `components`.
- Motion follows the same rule: S2 controls keep S2's transitions, own surfaces imitate the nearest
  S2 component, and the prototype's keyframes are dropped.
