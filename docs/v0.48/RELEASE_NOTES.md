---
layout: doc
title: Release Notes
permalink: /v0.48/release-notes/
---

# Tya v0.48 Release Notes

> **Status:** shipped. The `tya version` constant is `0.48.0` and
> `ROADMAP.md` carries the matching `Released` entry.

## TL;DR

v0.48 lands the two items deferred from v0.47:

- **G1 strict bare-name receivers** (codified, no code change):
  bare identifiers inside class method bodies resolve to locals /
  params / imports only — never to class members. To access a
  class member write `self.x`, `Self.x`, or `<OtherClass>.x`.
- **G6 canonical receiver rule** (new): `Self.foo` is the
  canonical form for referencing the declaring class's own member
  from inside its body. `<DeclaringClass>.foo` is non-canonical;
  the formatter rewrites it, and the checker emits
  `[TYA-E0413]` under `--check-unused` strict mode.

In addition, the formatter now consistently emits the v0.46
keyword surface for every class shape: `private`, `static`,
`self.`, `Self.`, `initialize`. The legacy sigils (`@`, `@@`,
`_`-prefix on class members, `init`/`_init`) are rewritten to
their v0.46 equivalents on output.

## `tya format` example

```tya
# input
class Counter
  @@count = 0
  _seed = 1

  init = name ->
    @name = name
    Counter.count = Counter.count + 1

  @@bump = ->
    Counter.count = Counter.count + 1
```

```tya
# output of `tya format` after v0.48
class Counter
  static count = 0
  private seed = 1

  initialize = name ->
    self.name = name
    Self.count = Self.count + 1

  static bump = () ->
    Self.count = Self.count + 1
```

## [TYA-E0413] strict-mode warning

```sh
$ tya check --check-unused counter.tya
4:5: [TYA-E0413] `Counter.count` inside its own class body is non-canonical;
                  write `Self.count`.
```

The warning fires at WARNING severity so CI can detect drift
without running the formatter.

## Self-host

`selfhost/v01/` continues to use its v0.43 surface. The
permissive-legacy path exemption from v0.47 is unchanged.
`TestSelfhostV01Scripts` remains green.

## Verification

```sh
go test ./... -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
```

All green at the v0.48.0 tag.

## Cross-References

- [`docs/v0.48/SPEC.md`](SPEC.md) — frozen v0.48 specification.
- [`docs/v0.47/SPEC.md`](../v0.47/SPEC.md) — the clean-cut release.
- [`docs/v0.46/SPEC.md`](../v0.46/SPEC.md) — the keyword surface.
- [`ROADMAP.md`](../../ROADMAP.md) — `Released` § v0.48.
