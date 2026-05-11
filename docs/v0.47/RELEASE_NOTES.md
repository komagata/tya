# Tya v0.47 Release Notes

> **Status:** shipped. The `tya version` constant is `0.47.0` and
> `ROADMAP.md` carries the matching `Released` entry.

## TL;DR

v0.47 is the **clean-cut release** that retires the legacy v0.45
class-member surface (`@`, `@@`, `_`-prefix, `init` / `_init`) that
v0.46 kept alive for backward compatibility. The v0.46 keyword
surface (`private`, `static`, `self`, `Self`, `initialize`) is now
the **only** way to write a class outside the `selfhost/v01/`
self-host tree.

## Diagnostics wired

| Code           | Stage   | Condition                                              |
| -------------- | ------- | ------------------------------------------------------ |
| `[TYA-E0407]`  | checker | Class member name begins with `_`.                     |
| `[TYA-E0410]`  | checker | `@field` or `@@field` sigil used.                      |
| `[TYA-E0411]`  | checker | `self` used inside a `static` method.                  |
| `[TYA-E0412]`  | checker | `Self` used outside a class body.                      |
| `[TYA-E0414]`  | checker | Class declares `init` / `_init`; expected `initialize`. |

## Migration

The same mechanical rewrites listed in v0.46's release notes apply.
After v0.47, user code MUST use the new surface; the legacy
syntax raises diagnostics. Affected forms:

```
@name             →  self.name
@name = x         →  self.name = x
@@count           →  Self.count
@@count = 0       →  static count = 0
@@count = x       →  Self.count = x
_name = ...       →  private name = ...
_init = name -> …  →  private initialize = name -> …
init = name -> …  →  initialize = name -> …
```

All of `examples/`, `stdlib/`, and `tests/testdata/v*/` were
migrated to the new surface in this release. The single fixture
that pinned legacy-only error messages (`v09/private_members`) was
retired — the underlying `_`-prefix privacy heuristic is gone.

## Self-Host Constraint

`selfhost/v01/compiler.tya` uses the v0.43 surface (`@`, `@@`,
`_`-prefix, `init`). The Go reference implementation enters
**permissive legacy mode** on the checker when the entry path is
under `selfhost/v01/`. This skips the E0407 / E0410 / E0411 /
E0414 diagnostics for that sub-tree. User code is not in that path
and is subject to the new rules.

The mechanism is `checker.SetPermissiveLegacy(bool)`, called by
the runner and `cmd/tya` entry points after deriving the legacy
flag from the input file path via `runner.IsLegacyV01Path(path)`.

`TestSelfhostV01Scripts` continues to pass; the v01 compiler
self-compiles unchanged.

## Deferred to future minors

- **G1 (strict bare-name receivers)** — full elimination of
  bare-name → class-member fallback in the checker's identifier
  resolution. Today bare names resolve through scope; the v0.47
  diagnostics catch the most common legacy paths, but a complete
  G1 enforcement requires walking the resolver and proving no
  field/static fallback can fire. Tracked for v0.48.
- **G6 (formatter rewrite)** — `<DeclaringClass>.foo` →
  `Self.foo` inside the declaring class body, plus
  `[TYA-E0413]` strict-mode warning. Tracked for v0.48.

## Verification

```sh
go test ./... -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
```

All green at the v0.47.0 tag. The v0.46 `TestV46Scripts` and v0.45
`TestV45Scripts` continue to pass on the migrated fixtures.

## Cross-References

- [`docs/v0.47/SPEC.md`](SPEC.md) — frozen v0.47 specification.
- [`docs/v0.46/SPEC.md`](../v0.46/SPEC.md) — the surface v0.47
  finalizes.
- [`docs/v0.46/RELEASE_NOTES.md`](../v0.46/RELEASE_NOTES.md) — the
  transitional release listing what v0.47 retires.
- [`ROADMAP.md`](../../ROADMAP.md) — `Released` § v0.47.
