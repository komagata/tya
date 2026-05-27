---
layout: doc
title: Spec
permalink: /v0.46/spec/
---

# Tya v0.46 Specification

> **Status:** shipped, transitional. The new keyword surface
> (`private`, `static`, `self`, `Self`, `initialize`) is fully
> implemented and is the canonical form for new code. The legacy
> v0.45 surface (`@`, `@@`, `_`-prefix, `init`) still parses for
> backward compatibility — the clean-cut removal of legacy syntax
> is deferred to v0.47. G4 (strict bare-name receivers) is also
> deferred: bare identifiers in method bodies continue to resolve
> through the v0.45 rules during the transition.

## Theme

v0.44 / v0.45 settled the **shape** of classes and packages
(directory-as-package, class files, cross-file private). v0.46
replaces the **sigil-based member syntax** (`@`, `@@`, `_`-prefix)
with **keyword-based syntax** modelled on Swift: `self` for the
instance, `Self` for the declaring class, `private` for visibility,
`static` for class-level members.

The result reads like Swift/Java but eliminates Java's bare-name
ambiguity (a real footgun in a dynamically-typed language) by
**requiring an explicit receiver** for every field/static access.

## Goals

- **G1.** Replace the leading-underscore private convention on class
  members with an explicit `private` keyword.
- **G2.** Replace the `@field` / `@@field` sigils with the Swift
  keyword pair: `self.field` (instance) and `Self.field` (class).
- **G3.** Replace the `@@field` declaration prefix with a `static`
  modifier keyword.
- **G4.** Enforce explicit receivers — bare identifiers in method
  bodies resolve to locals / parameters / imports only, never to
  class members.
- **G5.** Rename the constructor method from `init` to `initialize`.

## Non-Goals

- Type annotations (Tya stays dynamically typed).
- Method overloading.
- Polymorphic / dynamic static dispatch. `Self` is **statically
  resolved** to the lexically enclosing class — same as a hardcoded
  class name, just rename-safe. (Swift's dynamic-Self semantics are
  not adopted; see Q4.)
- Self-host upgrade work — that lives in v0.4x (M8 onward).
- Module-level private bindings (`module foo; _helper`) keep the
  underscore convention until v0.4x M9 retires the module keyword
  entirely.

## G1 — `private` keyword

`private` is a prefix modifier on a class member declaration. The
absence of `private` means public. Leading-underscore on a class
member is **no longer a visibility marker** in v0.46.

```tya
class User
  name = ""                # public instance field
  private id = 0           # private instance field
  static count = 0         # public class field
  private static seed = 1  # private class field

  initialize = name ->           # public constructor
    self.name = name

  private initialize_with_id = name, id ->   # private constructor
    self.name = name
    self.id = id

  greeting = ->            # public instance method
    "Hello, {self.name}"

  private normalize = ->   # private instance method
    self.name = trim(self.name)

  static build = name ->   # public class method
    Self(name)

  private static next_id = ->   # private class method
    Self.seed + 1
```

Canonical modifier order:

```
[private] [static] [abstract|final|override] <name> = <body>
```

Other orders are rejected by the parser (canonical syntax: every
program has exactly one source representation).

`_init` is no longer special-cased. The v0.46 private constructor
is spelled `private initialize` (see G5 for the constructor rename).

## G2 — `self` and `Self` keywords

### Reserved words

- **`self`** (lowercase) — refers to the **instance**.
  - Valid inside instance methods and constructors.
  - Invalid in `static` methods (the receiver is the class, not an
    instance) — `[TYA-E0411]`.
  - This is a **breaking change** to the existing `self` semantics
    (v0.45 reserved `self` as a class-method-only reference to the
    class object; v0.46 reverses it).

- **`Self`** (uppercase) — refers to the **class itself**.
  - Valid inside instance methods, static methods, field
    initializers, and constructors.
  - Invalid outside a class body — `[TYA-E0412]`.
  - **Statically resolved** to the lexically enclosing class. The
    `Self` written inside `class User` always means `User`,
    regardless of how the method is invoked. `Self` inside
    `class Admin extends User` means `Admin`.
  - `Self(args)` is equivalent to `<DeclaringClass>(args)` — i.e.
    instance construction.

`Self` is the only PascalCase reserved word in Tya. Its visual
form mirrors the language's existing class-name convention
(`PascalCase` = type-like), matching Swift's design.

### G2 sample

```tya
class User
  name = ""
  static count = 0

  initialize = name ->
    self.name = name              # instance field write
    Self.count = Self.count + 1   # class field write

  greeting = ->
    "Hello, {self.name} (#{Self.count})"   # both forms in interpolation

  static build = name ->
    Self(name)                    # constructs an instance of User

class Admin extends User
  initialize = name ->
    super(name)                   # parent constructor
    self.role = "admin"
    Self.count = Self.count + 1   # Admin's own count (if redeclared);
                                  # otherwise inherits User.count

  greeting = ->
    "Hello, {self.name} (admin)"
```

## G3 — `static` keyword

`static` is a prefix modifier marking a class-level
(class-bound, not instance-bound) member declaration.

```tya
class Counter
  static value = 0                   # class field
  static increment = ->              # class method
    Self.value = Self.value + 1
  static reset = ->                  # class method
    Self.value = 0
```

`static` composes with `private`, in canonical order:

```tya
class Counter
  private static seed = 42           # private class field
  private static reseed = n ->       # private class method
    Self.seed = n
```

The `@@field` declaration prefix is **removed** (`[TYA-E0410]`).

## G4 — Explicit receivers

In v0.46 a class-member access **always requires a receiver**:

| Target              | Form                  |
| ------------------- | --------------------- |
| Local / param       | bare name `x`         |
| Instance member     | `self.x`              |
| Class member (own)  | `Self.x`              |
| Class member (other) | `<ClassName>.x`      |

A bare identifier in a method body never resolves to a class
member. This is the deliberate divergence from Java, where
bare-name resolution falls back to `this.field` and class statics
when no local matches — a hazard that requires static type
checking to disambiguate. In a dynamically-typed language, the
bare-name fallback silently turns typos into new locals (or vice
versa).

The rule is symmetric for reads and writes. `count = count + 1`
inside a method body always creates / updates a local `count` and
never touches `Self.count`. To update a class field, write
`Self.count = Self.count + 1`.

### Why this beats Java

| Java pain                       | v0.46 resolution                         |
| ------------------------------- | ---------------------------------------- |
| `count = count + 1` ambiguous   | Bare `count` is **always** a local.      |
| `this` works only in instance   | `Self` is symmetric and works in static. |
| `ClassName.foo` hardcoded       | `Self.foo` is rename-safe.               |
| `_init` / `_foo` underscore     | `private initialize` / `private foo`.    |
| `public` is a verbose default   | Public is implicit (no marker).          |
| Static access bare or qualified | One canonical form: `Self.foo`.          |

### Canonical receiver rule

When accessing a member of the declaring class from inside that
class, the canonical form is **`Self.foo`**, not
`<DeclaringClass>.foo`. The formatter rewrites the latter to the
former.

`<OtherClass>.foo` (a name other than the declaring class) is
permitted **only when the access is to a different class** — for
example reaching a parent's class field from a subclass:

```tya
class User
  static count = 0

  static record_birth = ->
    Self.count = Self.count + 1   # canonical: Self

class Admin extends User
  static admin_count = 0

  static record_birth = ->
    User.count = User.count + 1   # explicit: reaching parent's count
    Self.admin_count = Self.admin_count + 1   # own static
```

If the access target is `Self` (the declaring class) but written
as a class name, the formatter rewrites it; the checker emits a
warning under `--check-unused` strict mode (`[TYA-E0413]`).

## G5 — `initialize` constructor name

The constructor method name changes from `init` to `initialize`.
Tya joins the Ruby tradition (`initialize`) rather than Swift's
abbreviation (`init`); the longer name is unambiguous and stops
reading as a regular noun.

```tya
class User
  name = ""

  initialize = name ->         # public constructor (was: init)
    self.name = name

  private initialize_with_id = name, id ->   # private constructor variant
    self.name = name
    self.id = id

  greeting = ->
    "Hello, {self.name}"
```

The `init` identifier becomes a regular name with no special
status. Code that previously declared `init = ...` either
(a) renames to `initialize` (typical migration) or (b) keeps the
name `init` if it was a non-constructor helper method (unusual but
allowed — `init` is no longer reserved).

Combined with G1: the v0.45 `_init` private constructor becomes
`private initialize` in v0.46, not `private init`.

Diagnostic: `[TYA-E0414]` — a class declares `init = ...` (or
`private init = ...`) where `initialize` is expected. The error
points at the rename. Wired in v0.46 as a clean cut; there is no
deprecation window.

## Static dispatch semantics

`Self` is **statically resolved**, like Java's static methods:

- `Self` inside `class User`'s body is `User`. Always.
- `Self` inside `class Admin extends User`'s body is `Admin`. Always.
- If `Admin` does not redeclare a static method declared on `User`,
  calling `Admin.foo()` invokes the inherited body — which still
  reads `User`'s statics (because that body was lexically inside
  `User` when it wrote `Self`).

Concretely:

```tya
class User
  static count = 0
  static record_birth = ->
    Self.count = Self.count + 1   # Self is User here

class Admin extends User
  static count = 0                # Admin's own (shadows User.count)
  # record_birth not redeclared

Admin.record_birth()
# Calls User.record_birth (inherited).
# Inside that body, Self == User → User.count is updated.
# Admin.count stays 0.
```

If you want the subclass's `count` updated, redeclare
`record_birth` on `Admin`. This matches Java's static-not-virtual
rule and avoids Swift's dynamic-Self subtleties.

## Full surface (combined sample)

```tya
abstract class Shape
  private name = ""
  static species_count = 0

  initialize = name ->
    self.name = name
    Self.species_count = Self.species_count + 1

  abstract area = ->
    nil

  describe = ->
    "{self.name} (area={self.area()})"

  private normalize = ->
    self.name = trim(self.name)

  static count = ->
    Self.species_count

final class Circle extends Shape
  private radius = 0

  initialize = name, radius ->
    super(name)
    self.radius = radius

  override area = ->
    self.radius * self.radius * 3.14
```

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
| `@@_count = 0`         | `private static count = 0`        |
| `@@_helper = -> ...`   | `private static helper = -> ...`  |
| `User.foo` (inside User) | `Self.foo`                      |
| `User.foo` (outside)   | `User.foo` (unchanged)            |

The migration is purely mechanical and can be automated. Tya does
not ship a deprecation window: v0.46 is a clean cut, with the
diagnostics table below pointing at every site that needs editing.

## Diagnostics

| Code           | Stage   | Wired in | Condition                                              |
| -------------- | ------- | -------- | ------------------------------------------------------ |
| `[TYA-E0407]`  | checker | v0.46    | A class member name begins with `_`. The leading underscore is no longer a privacy marker; rename or add `private`. |
| `[TYA-E0408]`  | parser  | v0.46    | `private` keyword used outside a class body.           |
| `[TYA-E0409]`  | parser  | v0.46    | Modifier order is non-canonical. Expected: `private static abstract|final|override <name>`. |
| `[TYA-E0410]`  | parser  | v0.46    | `@` or `@@` sigil used. Removed in v0.46; use `self.name` / `Self.name` / `static name = ...`. |
| `[TYA-E0411]`  | checker | v0.46    | `self` used inside a `static` method. `self` refers to the instance and is unavailable here; use `Self`. |
| `[TYA-E0412]`  | checker | v0.46    | `Self` used outside a class body.                      |
| `[TYA-E0413]`  | checker | v0.46    | `<DeclaringClass>.foo` written inside its own class body. Canonical form is `Self.foo`. (Strict-mode warning; the formatter auto-rewrites.) |
| `[TYA-E0414]`  | checker | v0.46    | Class declares `init = ...` (or `private init = ...`) where `initialize` is expected. Constructor renamed to `initialize` in v0.46. |

## Implementation Sketch

- **Lexer:** add `private`, `static`, `Self`, `this`-reserve (for
  future use, but `this` is not exposed in v0.46) to the reserved-
  word table. The existing `self` keyword stays reserved but
  changes meaning (see Checker).
- **Parser:** class-body statement dispatch grows a modifier
  prefix loop accepting `private` and `static` in canonical order.
  AST `ClassMember` nodes get `Private` and `Static` flags. The
  `_init` special case is removed.
- **AST:** `SelfExpr` (existing) is renamed `InstanceSelfExpr`
  and gains a new sibling `ClassSelfExpr` for `Self`. Or simpler:
  one `SelfExpr` with a `Class bool` field. Field-access nodes
  walk through `self.foo` / `Self.foo` parsing as `MemberExpr`
  with the reserved-word target.
- **Checker:** rewire `isPrivateName(name)` to read the new
  `Private` flag. New checks:
  - `self` outside instance method → `[TYA-E0411]`
  - `Self` outside class body → `[TYA-E0412]`
  - `<DeclaringClass>.foo` inside own class body → `[TYA-E0413]`
    (warn).
  - bare identifier that matches an instance / class member but
    not a local → undefined-variable error pointing at
    `self.foo` / `Self.foo` as the fix.
- **C emitter:** `self.foo` lowers to the same C as the v0.45
  `@foo` did (`tya_member(__this, "@" + name)`). `Self.foo`
  lowers to the same C as v0.45 `@@foo` did. The runtime layer
  is unchanged.
- **Formatter:** emit `private` / `static` keywords in canonical
  order. Rewrite `<DeclaringClass>.foo` → `Self.foo` inside the
  declaring class. Rewrite legacy `@`/`@@`/`_foo` shapes during
  migration (offered behind `tya format --migrate` for one minor).

## Acceptance

- `go test ./... -count=1` green; both self-host gates green.
- New fixtures under `tests/testdata/v46/`:
  - `private_keyword.txtar` — every class-member shape with and
    without `private`.
  - `self_keyword.txtar` — `self.field` and `self.method()` reads
    and writes inside instance methods; `[TYA-E0411]` from inside
    a static method.
  - `Self_keyword.txtar` — `Self.field`, `Self.method()`,
    `Self(args)`, both from instance and static methods;
    `[TYA-E0412]` from outside a class.
  - `static_keyword.txtar` — every static-declaration shape.
  - `combined_modifiers.txtar` — `private static`, `private
    abstract`, `override`, `final` combinations.
  - `migration_diagnostics.txtar` — every E0407 / E0410 / E0411 /
    E0412 / E0413 case.
- `examples/`, `lib/`, and the v0.46-surface portion of the
  Go reference impl carry no `@`/`@@` sigils and no `_`-prefixed
  class members after v0.46 lands.

## Self-Host Constraint

`selfhost/v01/compiler.tya` uses the v0.43 surface (`@`, `@@`,
`_foo` privacy). v0.46 ships the new surface in the **Go reference
implementation** only; the v01 self-host stays on its frozen
surface and is exempt from the new diagnostics when invoked as a
v0.1-surface source.

Concretely:

- The Go lexer/parser/checker accept the v0.46 surface AND the
  v0.43 surface during a transition window.
- Files identified as part of `selfhost/v01/` keep the v0.43
  rules. (Path-based exemption is acceptable for a self-host
  gate; user code is not in that path.)
- `selfhost/v02/` is reset to track the v0.46 surface as it
  develops in the v0.4x Epic. The M8.0–M8.2d scaffolding lands as
  before, but the M8.x work going forward targets the v0.46
  surface directly.

The clean alternative — port v01 to the v0.46 surface — is also
acceptable but adds a sub-task that the v0.46 Epic does not
currently scope. Pick at implementation time.

## Open Questions

**Q1.** Should `Self` be allowed at the very top level of a class
body (in field initializer position)? Yes:

```tya
class Sequence
  static current = 0
  static next = Self.current + 1   # OK at decl time? checker order?
```

Implementation concern: declaration order matters. `Self.current`
referenced before `current` is declared raises an "undefined
class member" error. Tya does not promise forward references in
class bodies. The user must order declarations top-down.

**Q2.** `super` for static methods? Current Tya `super(args)`
works for parent constructor / parent instance method. v0.46 does
not extend `super` to static methods; reach the parent's static
explicitly via `<ParentName>.foo` (the `<OtherClass>.foo`
permitted form). Revisit if subclasses need to call shadowed
parent statics frequently.

**Q3.** Reserve `this` even though v0.46 does not use it? Yes,
for future-proofing. `[TYA-E0415]` "`this` is reserved" if a user
tries to use it as an identifier — prevents future surprises.

**Q4.** Adopt Swift's dynamic-`Self` (polymorphic statics) in a
later minor? Open. The v0.46 spec is consistent with Swift in
naming but conservative in semantics (static resolution). If a
future minor adopts dynamic resolution it can do so without
renaming the keyword; the change is purely in `Self`-lookup at
codegen time.

## Cross-References

- [`docs/v0.44/SPEC.md`](../v0.44/SPEC.md) — the class model that
  v0.46 reshapes.
- [`docs/v0.45/SPEC.md`](../v0.45/SPEC.md) — cross-file private
  enforcement (defines `[TYA-E0406]` which v0.46 keeps unchanged).
- [`docs/NAMING.md`](../NAMING.md) — file/identifier shape rules;
  v0.46 adds `Self` as the sole PascalCase keyword.
- [`docs/CANONICAL_SYNTAX.md`](../CANONICAL_SYNTAX.md) — v0.46
  adds: (a) canonical modifier order on class members, (b) the
  `Self.foo` over `<DeclaringClass>.foo` rule.
- [`ROADMAP.md`](../../ROADMAP.md) — v0.46 entry to be added under
  `Scheduled`.
