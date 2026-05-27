---
status: completed
goal_ready: false
---

# Feature: Color Stdlib Library

## Goal

Add a standard `color` library with a reusable `Color` class for RGBA colors,
hex/CSS-style parsing, blending, and simple color adjustments shared by image
processing, raylib bindings, terminal output, charts, and web tooling.

## Context

Planned image and raylib work both need color values. If each package defines
its own `Color`, users will have to convert between nearly identical types. A
small stdlib `color.Color` gives Tya one common color value for general use.

The public value should be a class instance, not a dictionary, matching the
class-style direction for stdlib-owned domain values.

## Behavior

- Add a public `color` stdlib package.
- Import shape:

  ```tya
  import color

  red = color.Color.rgb(255, 0, 0)
  blue = color.Color.hex("#0066ff")
  mixed = color.Color.blend(red, blue, 0.5)
  println mixed.to_hex()
  ```

- Public class:
  - `color.Color`
- `Color` instances expose numeric public fields:
  - `r`
  - `g`
  - `b`
  - `a`
- Channels are integers in `0..255`.
- Constructors and operations return `Color` instances.
- Color instances are immutable by API convention; operations return new
  instances.
- Invalid channel values, invalid hex strings, and invalid blend ratios raise
  clear color errors.

## Constructors

- `Color.rgb(r, g, b)` creates an opaque color with alpha `255`.
- `Color.rgba(r, g, b, a)` creates a color with explicit alpha.
- `Color.gray(value)` creates `r == g == b == value`, alpha `255`.
- `Color.gray(value, alpha)` creates grayscale with explicit alpha.
- `Color.hex(text)` parses:
  - `#rgb`,
  - `#rgba`,
  - `#rrggbb`,
  - `#rrggbbaa`,
  - the same forms without a leading `#`.
- `Color.css(text)` parses the first-version CSS subset:
  - hex forms accepted by `Color.hex`,
  - `rgb(r, g, b)`,
  - `rgba(r, g, b, a)`,
  - named colors listed in this PRD.
- `Color.from_array(values)` accepts `[r, g, b]` or `[r, g, b, a]`.
- `Color.transparent()` returns `rgba(0, 0, 0, 0)`.

## Named Colors

- Add common named color constructors:
  - `Color.black()`
  - `Color.white()`
  - `Color.red()`
  - `Color.green()`
  - `Color.blue()`
  - `Color.yellow()`
  - `Color.cyan()`
  - `Color.magenta()`
  - `Color.gray50()`
  - `Color.transparent()`
- `Color.css` must recognize lowercase CSS names for at least:
  - `black`
  - `white`
  - `red`
  - `green`
  - `blue`
  - `yellow`
  - `cyan`
  - `magenta`
  - `transparent`

## Conversion

- `color.to_hex()` returns `#rrggbb` when alpha is `255`.
- `color.to_hex(true)` returns `#rrggbbaa`.
- `color.to_array()` returns `[r, g, b, a]`.
- `Color.equal?(a, b)` compares all channels exactly.
- `Color.nearly_equal?(a, b, tolerance)` compares channels within tolerance.
- `Color.luminance(color)` returns relative luminance in `0.0..1.0`.
- `Color.contrast_ratio(a, b)` returns the WCAG-style contrast ratio.

## Operations

- `Color.with_alpha(color, alpha)` returns a copy with a new alpha channel.
- `Color.invert(color)` returns the RGB inverse and preserves alpha.
- `Color.grayscale(color)` returns a grayscale color and preserves alpha.
- `Color.blend(a, b, t)` linearly interpolates non-premultiplied RGBA channels.
  `t == 0` returns `a`; `t == 1` returns `b`.
- `Color.over(foreground, background)` alpha-composites `foreground` over
  `background` and returns an opaque color when background is opaque.
- `Color.lighten(color, amount)` mixes toward white.
- `Color.darken(color, amount)` mixes toward black.
- `Color.saturate(color, amount)` and `Color.desaturate(color, amount)` may be
  implemented through simple HSL conversion.
- Amounts and blend factors use `0.0..1.0`; out-of-range values raise.
- Channel results are rounded to the nearest integer and clamped to `0..255`.

## Integration Expectations

- The planned `image` stdlib should accept `color.Color` instances for fill,
  pixels, background, and compositing options.
- The planned `raylib` external package should accept `color.Color` instances
  in drawing calls, even if it also provides `raylib.Color` aliases or
  constants for convenience.
- Terminal/colorized-output features may use `color.Color` for RGB values, but
  ANSI styling itself is out of scope for this PRD.

## Scope

- `lib/color/Color.tya`
- `tests/stdlib_color_test.tya`
- `docs/STDLIB.md`
- Next release `docs/vX.Y/SPEC.md` and `docs/vX.Y/RELEASE_NOTES.md`
- Updates to planned PRDs or docs that mention `image.Color` or `raylib.Color`
  if needed to clarify that `color.Color` is the shared type.

## Out of Scope

- Full CSS Color Level 4 support.
- Color profiles, ICC, CMYK, LAB, LCH, XYZ, wide-gamut color spaces, or HDR.
- Palette generation.
- Gradients.
- Terminal ANSI escape rendering.
- Image pixel storage or image codecs.
- Native code.

## Acceptance Criteria

- `import color` exposes `color.Color`.
- `Color.rgb`, `Color.rgba`, `Color.gray`, `Color.hex`, `Color.css`, and
  `Color.from_array` return `Color` instances.
- Constructed colors expose `r`, `g`, `b`, and `a` fields and have
  `.class == color.Color`.
- Invalid channel values and invalid string forms raise clear color errors.
- Hex parsing supports short and long RGB/RGBA forms with or without `#`.
- `to_hex`, `to_array`, exact equality, and nearly-equal comparison work.
- Named color constructors and the documented `Color.css` named colors work.
- `blend`, `over`, `with_alpha`, `invert`, `grayscale`, `lighten`, and `darken`
  return deterministic colors.
- Amounts outside `0.0..1.0` raise clear errors.
- Existing stdlib tests remain green.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run 'Test.*Color|TestSelfhostV01Scripts' -count=1
go test ./... -count=1
```

Manual smoke after implementation:

```sh
tya run examples/color/palette.tya
```

## Dependencies

- Uses only existing Tya numeric/string helpers.
- Should land before or alongside the planned `image` stdlib and `raylib`
  external package so those APIs can share `color.Color`.

## Open Questions

None.
