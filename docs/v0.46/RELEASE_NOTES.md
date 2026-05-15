---
layout: doc
title: Release Notes
permalink: /v0.46/release-notes/
---

# Tya v0.46 Release Notes

> **Status:** shipped (transitional). The `tya version` constant
> is `0.46.0` and `ROADMAP.md` carries the matching `Released`
> entry.

## TL;DR

v0.46 is a **transitional release** that introduces a **sigil-free,
keyword-based class-member surface** modelled on Swift. The new
keywords are the canonical form for new code; the legacy v0.45
surface (`@`, `@@`, `_`-prefix, `init`) still parses unchanged for
backward compatibility. A future minor (v0.47) lands the clean cut
that removes legacy syntax.

Five Goals shipped:

- **G1.** `private` keyword replaces the `_`-prefix privacy
  convention on class members.
- **G2.** `self` (lowercase) refers to the instance; `Self`
  (capital S) refers to the declaring class. Both replace `@`/`@@`
  sigils.
- **G3.** `static` keyword replaces the `@@` declaration prefix.
- **G5.** Constructor renamed from `init` to `initialize`.

(G4 — strict bare-name receivers — is deferred to v0.47 with the
clean cut.)

## What's new

### `private` keyword

```tya
class User
  name = ""                # public instance field
  private id = 0           # private instance field
  static count = 0         # public class field
  private static seed = 1  # private class field

  initialize = name ->
    self.name = name

  private normalize = ->
    self.name = trim(self.name)

  static build = name ->
    Self(name)

  private static next_id = ->
    Self.seed + 1
```

Canonical modifier order: `[private] [static] [abstract|override]
<name> = <body>`.

### `self` and `Self`

- `self` (lowercase) is the **instance**. Valid in instance
  methods and constructors.
- `Self` (capital S) is the **declaring class**, statically
  resolved. Valid in all class-body contexts (instance methods,
  static methods, field initializers).
- `Self(args)` constructs an instance of the declaring class —
  rename-safe in lieu of writing the class name literally.

```tya
class User
  static count = 0

  initialize = name ->
    self.name = name              # instance field write
    Self.count = Self.count + 1   # class field write

  greeting = ->
    "Hello, {self.name}"

  static build = name ->
    Self(name)                    # rename-safe construction
```

### `static` keyword

```tya
class Counter
  static value = 0                # class field (was: @@value)

  static increment = ->           # class method
    Self.value = Self.value + 1
```

### `initialize` constructor

```tya
class User
  initialize = name ->            # public constructor (was: init)
    self.name = name

  private initialize_with_id = name, id ->   # private variant
    self.name = name
    self.id = id
```

The legacy `init` / `_init` keep working during the transition.

## Migration

Mechanical rewrites:

| Old (v0.45)            | New (v0.46)                       |
| ---------------------- | --------------------------------- |
| `@name`                | `self.name`                       |
| `@name = x`            | `self.name = x`                   |
| `@@count` (read)       | `Self.count`                      |
| `@@count = x`          | `Self.count = x`                  |
| `@@count = 0` (decl)   | `static count = 0`                |
| `@@build = -> ...`     | `static build = -> ...`           |
| `_name = ...`          | `private name = ...`              |
| `_method = -> ...`     | `private method = -> ...`         |
| `init = name -> ...`   | `initialize = name -> ...`        |
| `_init = name -> ...`  | `private initialize = name -> ...` |

The official stdlib, examples, and documentation have been migrated
to the new surface. User code using the legacy syntax continues to
work; migration is encouraged before v0.47 ships the clean cut.

## Transitional rules

- Legacy `@field` and `@@field` parse correctly.
- Legacy `_foo` (class member) is still treated as private.
- Legacy `init` / `_init` are still recognized as constructors.
- Mixing legacy and new in the same class is allowed (`@count`
  reading next to `Self.count`); the formatter will canonicalize
  later.
- `self` in **class methods** retains its v0.45 meaning (the class
  object) AS WELL AS the v0.46 meaning. To avoid ambiguity in new
  code, write `Self.foo` for class-level access.

## Self-host

`TestSelfhostV01Scripts` remains green: the v01 self-host compiler
keeps using v0.43 surface unchanged. The Go reference implementation
ships v0.46 surface support; the v02 self-host (in progress under
the v0.4x Epic) will adopt v0.46 directly.

## Verification

```sh
go test ./... -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./tests -run TestV46Scripts -count=1
```

All green at the v0.46.0 tag.

## Deferred to v0.47

- **G4** strict bare-name receivers. v0.47 will require an explicit
  receiver (`self.x`, `Self.x`, `<ClassName>.x`) for every class
  member access; bare identifiers will resolve to locals / params /
  imports only.
- **Clean cut** of the legacy v0.45 surface. v0.47 will emit
  diagnostics for `@`, `@@`, `_`-prefix privacy, and `init` /
  `_init` constructor names ([TYA-E0407], [TYA-E0410], [TYA-E0411],
  [TYA-E0412], [TYA-E0414]).
- **Formatter rewriter** (`tya format --migrate`) to apply the
  mechanical rewrites above.

## Cross-References

- [`docs/v0.46/SPEC.md`](SPEC.md) — frozen v0.46 specification.
- [`docs/v0.45/SPEC.md`](../v0.45/SPEC.md) — the surface v0.46
  builds on.
- [`ROADMAP.md`](../../ROADMAP.md) — `Released` § v0.46.
