---
status: completed
goal_ready: false
---

# Feature: TOML Stdlib Class-Style Documentation

## Goal

Keep the existing public `toml` stdlib package, but document it consistently as
a class-style package (`toml.Toml.parse` / `toml.Toml.dump`) and clarify that
the toolchain's internal TOML parser is a separate private implementation
detail for `tya.toml` and `tya.lock`.

## Context

Tya already has a public TOML stdlib implementation at
`lib/toml/Toml.tya`. Existing tests use the class-style API:

```tya
import toml

config = toml.Toml.parse("name = \"tya\"\n")
text = toml.Toml.dump(config)
```

The toolchain also has an internal Go TOML subset parser in
`internal/pkg/toml.go` for project manifests and lockfiles. That internal
parser is not the public stdlib API.

Current editable docs still include older module-style examples such as
`toml.parse` and `toml.dump`, which makes the public surface look inconsistent
with the class-file package layout.

## Behavior

- Keep `lib/toml/Toml.tya` as the public bundled stdlib TOML package.
- Document the public API as:
  - `toml.Toml.parse(text)`
  - `toml.Toml.dump(value)`
- Remove editable current-doc references that advertise `toml.parse` or
  `toml.dump` as top-level module functions.
- Clarify in docs that `tya.toml` / `tya.lock` parsing uses private toolchain
  code and does not depend on importing the public `toml` package.
- Keep public stdlib TOML behavior aligned with existing tests unless a release
  spec intentionally expands it.
- Do not expose `internal/pkg/toml.go` as user API.

## Scope

- `docs/STDLIB.md`
- `docs/API.md` only if it lists `toml` APIs
- `docs/TERMINOLOGY.md` only if internal-vs-stdlib wording needs clarification
- next release `docs/vX.Y/SPEC.md` and `docs/vX.Y/RELEASE_NOTES.md`
- generated docs HTML if release prep rebuilds it
- `tests/stdlib_toml_test.tya` only if examples or API names need class-style
  updates
- `ROADMAP.md`

## Out of Scope

- Removing the public `toml` stdlib package.
- Replacing the internal Go TOML parser with the Tya stdlib implementation.
- Expanding TOML support beyond the existing public parser/dumper behavior.
- Adding TOML 1.1 features, datetime values, dotted-key assignments, or
  multiline strings.
- Adding a native or third-party TOML dependency.

## Acceptance Criteria

- Editable current docs show `toml.Toml.parse` and `toml.Toml.dump` as the
  public API.
- Editable current docs no longer advertise `toml.parse` or `toml.dump` as
  public top-level module functions.
- Documentation explicitly states that `tya.toml` / `tya.lock` support is
  provided by private toolchain code, separate from the stdlib package.
- Existing `tests/stdlib_toml_test.tya` coverage remains green.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```

## Open Questions

None.
