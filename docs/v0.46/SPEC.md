# Tya v0.46 Specification (draft)

> **Status:** draft — under design review. Two Goals, both targeting
> class-member declaration syntax. G1 has a settled design. G2
> presents three options with a recommendation and is open for
> discussion before being committed.

## Theme

v0.44 / v0.45 settled the **shape** of classes and packages
(directory-as-package, class files, cross-file private). v0.46
revisits the **declaration syntax for class members** so the surface
reads like a typical OO language rather than a Tya-internal mix of
`_` prefixes and `@@` sigils:

1. Member visibility uses a **`private` keyword**, not a `_` prefix.
2. The class-level vs instance-level distinction uses a **`static`
   keyword**, replacing the `@@` declaration prefix.

These are surface changes only. Class semantics (inheritance,
override, abstract/final, interface conformance, GC, dispatch) stay
identical.

## Goals

- **G1.** Replace the leading-underscore private convention with an
  explicit `private` keyword on class members.
- **G2.** Replace the `@@` declaration prefix with a `static` keyword.
  Access-site syntax is one of three options below; the choice is
  the open design question.

## Non-Goals

v0.46 explicitly does **not** include:

- A new language feature beyond visibility/staticness keywords.
- Type annotations (Tya stays dynamically typed).
- Method overloading.
- A change to how instance fields are **read** (`@field` stays — see
  G2 rationale).
- Removal of any other v0.44 surface element.
- Self-host upgrade work (that lives in v0.4x).
- A formatter rewrite. The formatter just learns the two new
  keywords.

## G1 — `private` keyword

### Current

Today a class member is private iff its name starts with `_`. The
checker treats `_foo`, `_init`, `_count`, etc. as private. The same
convention is used for module-level private bindings.

```tya
class User
  _name = ""               # private instance field
  @@_count = 0             # private class field (note awkward @@_)

  _init = name ->          # private constructor
    @_name = name
    @@_count = @@_count + 1

  _normalize_name = ->     # private instance method
    @_name = trim(@_name)
```

### Problems

- `@@_count` reads as a sigil soup.
- `_init` is special-cased in the checker as "private constructor",
  which is opaque from the surface.
- Underscore conflates two unrelated concerns: name shape and
  visibility. A user who likes leading-underscore for *style*
  reasons cannot have a public `_helper`.
- Other modifiers (`abstract`, `final`, `override`, `static`) are
  keywords; `private` should be too, for consistency.

### Proposal

`private` is a prefix modifier on a class member declaration. The
absence of `private` means public. The leading-underscore
convention is **removed** from classes (it stays for module-level
private bindings, where it is a less load-bearing distinction —
see Migration / open questions).

```tya
class User
  private name = ""              # private instance field
  private static count = 0       # private class field
  static built_count = 0         # public class field

  private init = name ->         # private constructor
    @name = name
    User.count = User.count + 1  # (see G2 for access syntax)

  greeting = ->                  # public instance method
    "Hello, {@name}"

  private normalize_name = ->    # private instance method
    @name = trim(@name)

  static build = name ->         # public class method
    User(name)
```

Modifier order: `private` comes first, then `static` / `abstract` /
`final` / `override`. Other orders are syntax errors so the
canonical form is unambiguous.

### Diagnostics

- `[TYA-E0407]` — checker. A `_`-prefixed class member name is no
  longer recognized as private in v0.46. Either the member is
  intentionally public and the underscore is stylistic
  (allowed), or it should be marked `private` (the typical
  migration). Wired as a strict-mode warning during the transition,
  upgrades to error at v0.47.

  Actually — see the **Migration** section: v0.46 ships a clean
  cut (no warning period). The diagnostic table below reflects the
  clean-cut choice; remove the code if a deprecation window is
  preferred instead.

- `[TYA-E0408]` — parser. `private` keyword used outside a class
  body. (Module-level private stays `_`-prefixed in v0.46; see open
  question Q1.)

## G2 — `static` keyword for class-level members *(open)*

### Current

Class-level members (Java's `static`) are declared with `@@`:

```tya
class User
  @@count = 0
  @@build = name ->
    User(name)
```

Access from inside the class also uses `@@`:

```tya
class User
  @@count = 0

  init = ->
    @@count = @@count + 1     # read + write to class field
```

Access from outside uses dotted form:

```tya
n = User.count
u = User.build("komagata")
```

### Problem

`@@` is unfamiliar to users coming from Java/Kotlin/Swift/Python.
The visual weight is also high — `@@build`, `@@count`, `@@assert_*`
look like decorators or annotations in those languages, not
"class-level".

### Proposal: `static` keyword on declaration

This part is uncontroversial:

```tya
class User
  static count = 0                 # class field
  static build = name ->           # class method
    User(name)
```

The **open question is what replaces the `@@` access form inside the
class body**. Three options, each consistent with the declaration
syntax above but with different ergonomic / semantic trade-offs.

---

### Option A — Explicit `ClassName.member` access *(recommended)*

From inside a class body, reach class-level members via the class's
own name. This is exactly how external access works today, so the
"inside" and "outside" surfaces unify.

```tya
class User
  static count = 0
  name = ""

  init = name ->
    @name = name
    User.count = User.count + 1   # explicit, same as external access

  greeting = ->
    "Hello, {@name} (#{User.count})"

  static build = name ->
    User(name)                    # constructor call, unchanged
```

**Pros:**
- One mental model: a class member is always `ClassName.foo`,
  whether you're inside or outside. No "inside the class you have a
  shortcut" rule to remember.
- Renaming the class is the only thing that touches class-member
  access — and a rename-aware formatter / IDE refactor catches every
  site.
- Migration is mechanical and unambiguous (1:1 token rewrite).
- The `@field` rule for instance fields stays, which keeps Tya's
  "is this state or a local?" visual cue.

**Cons:**
- Slightly more verbose than `@@count` (one extra token).
- Inside a deep inheritance chain you have to name the class that
  declared the member. Subclass code that references a parent's
  static field writes `Parent.foo`, not `User.foo`. (Java behaves
  the same: `Parent.foo` is the explicit canonical reference; `foo`
  bare-name resolution happens to also work for inherited statics.)

**Self-keyword variant:** if `Parent.foo` from a subclass feels
brittle, introduce `Self.foo` (capital S to distinguish from the
lowercase per-instance `self`) which always resolves to the class
the method was *declared in*. This keeps the "rename the class
freely" property for the declaring class's own static members.

---

### Option B — Bare-name resolution inside the class

Inside a class body, an unqualified name first looks at locals /
params, then at the class's own static members.

```tya
class User
  static count = 0
  name = ""

  init = name ->
    @name = name
    count = count + 1            # resolves to User.count
                                 # NOT to a new local "count"

  static build = name ->
    User(name)
```

**Pros:**
- Most "Java-like" in the natural sense — Java lets you write `foo`
  bare inside a method to reach a static `foo` declared in the same
  class.
- Shortest at the call site.

**Cons:**
- Assignment ambiguity: `count = count + 1` could be "create a new
  local named `count`" or "update `User.count`". Tya's existing
  rule is "assignment to an undefined name creates a local";
  v0.44's `[TYA-E0307]` already rejects outer-assign in lambdas to
  prevent silent shadowing. Bare static-write reopens the same
  hazard inside class methods.
- The fix — "make all writes to a static go through `User.count = `
  but reads can be bare" — is asymmetric and surprising. Worse, in
  Java the same asymmetry exists but type checking catches the
  "did you mean a field or a local?" mistake. Tya is dynamic and
  would not.
- Renaming a method param to match a static silently shadows the
  static.
- Hidden interaction with the v0.44 within-package bare-Ident class
  reference (`Foo()` where `Foo` is a sibling class in the same
  module). Now `count` bare can mean local, static, *or* sibling
  class.

**Conclusion:** Tya is dynamically typed and already has the
outer-assign hazard. Option B compounds it. Not recommended unless
there is a separate decision to ban bare-name writes to class
members.

---

### Option C — `static.member` access prefix

Keep the spirit of `@@` but spell it with a keyword. Inside a class
body, `static.foo` reads / writes the class-level `foo`.

```tya
class User
  static count = 0
  name = ""

  init = name ->
    @name = name
    static.count = static.count + 1

  static build = name ->
    User(name)
```

**Pros:**
- Visually mirrors `@field` for instance access. Both prefixes mean
  "not a local".
- Symmetric with the `static` keyword in declarations.
- No name-resolution ambiguity.
- Works uniformly even if the class is renamed.

**Cons:**
- Java doesn't have this. Python (`cls.foo` inside `@classmethod`),
  Ruby (`@@foo` — what we're moving *away* from), Swift
  (`Self.foo`), and Kotlin (`Companion.foo`) all use some named
  receiver. Calling it `static.` is a Tya invention.
- `static` reads as a modifier in declarations; using the same
  word as a receiver in expression position is a context overload
  and might confuse readers.

---

### Recommendation

**Option A** with the **`Self.` variant** for subclass-cleanliness:

```tya
class User
  static count = 0

  init = ->
    Self.count = Self.count + 1   # always means "this class's count"

  static build = name ->
    User(name)

class Admin extends User
  init = ->
    super()
    Self.count = Self.count + 1   # Admin.count, not User.count
                                  # (would be a separate static)
```

Reasoning:

1. **Closest to Java/Kotlin without the dynamic-typing footgun of
   Option B.** Bare-name writes never silently retarget a class
   field. Java escapes Option B's hazard via static type-checking
   that Tya lacks.
2. **Inside-the-class and outside-the-class are the same form.**
   Anyone who can read external call sites (`User.count`) can read
   internal ones too.
3. **`Self.` keeps subclasses honest.** A subclass's `Self.count`
   doesn't accidentally read the parent's class field unless it
   inherits or explicitly forwards via `User.count`. (`self` —
   lowercase — keeps referring to the instance.)
4. **`@field` for instance access stays.** The visual difference
   between "state of this object" (`@field`) and "class-level
   state" (`Self.field`) is preserved.

Option B is rejected. Option C is the second-best — pick it if you
want a class-local prefix that mirrors `@field` for instance access
and the `static` keyword reuse doesn't feel jarring after a week of
reading code.

## Combined surface (proposal, assuming G1 + G2 Option A)

```tya
class User
  name = ""                       # public instance field (default value)
  private id = 0                  # private instance field
  static count = 0                # public class field
  private static seed = 1         # private class field

  init = name ->                  # public constructor
    @name = name
    @id = Self.seed                # private class field, internal access
    Self.count = Self.count + 1
    Self.seed = Self.seed + 1

  private init_with_id = name, id ->  # private constructor variant
    @name = name
    @id = id

  greeting = ->                   # public instance method
    "Hello, {@name} (id={@id})"

  private normalize = ->          # private instance method
    @name = trim(@name)

  static build = name ->          # public class method
    User(name)

  private static seed_next = ->   # private class method
    Self.seed + 1

abstract class Animal
  private static species_count = 0

  abstract speak = ->
    nil

final class Cat extends Animal
  override speak = ->
    "meow"
```

Order of modifiers (canonical):

```
[private] [static] [abstract|final|override] <name> = <body>
```

Where applicable. `abstract` / `final` are class-level concepts on
methods (abstract methods exist on the type); `override` is method-
level; the formatter enforces this exact order.

## Migration

Mechanical rewrites:

| Old                          | New                              |
| ---------------------------- | -------------------------------- |
| `_name = ...`                | `private name = ...`             |
| `_method = -> ...`           | `private method = -> ...`        |
| `_init = name -> ...`        | `private init = name -> ...`     |
| `@@count = 0`                | `static count = 0`               |
| `@@build = -> ...`           | `static build = -> ...`          |
| `@@_count = 0`               | `private static count = 0`       |
| `@@_helper = -> ...`         | `private static helper = -> ...` |
| `@_name` inside body         | `@name` (name no longer carries `_`) |
| `@@count` inside body (read) | `Self.count`                     |
| `@@count = ...` inside body  | `Self.count = ...`               |
| External `User.count`        | `User.count` (unchanged)         |

Module-level private bindings (`module foo; _helper = ...`) are
**out of scope** for v0.46 — see Q1. They keep the `_` prefix.

A `tya migrate` subcommand could automate the rewrite, but is
non-blocking; the migrations are simple enough for grep+sed.

## Diagnostics

| Code           | Stage   | Wired in | Condition                                              |
| -------------- | ------- | -------- | ------------------------------------------------------ |
| `[TYA-E0407]`  | checker | v0.46    | Class member name begins with `_`. The leading underscore is no longer a privacy marker; rename or add `private`. |
| `[TYA-E0408]`  | parser  | v0.46    | `private` keyword used outside a class body.           |
| `[TYA-E0409]`  | parser  | v0.46    | Modifier order is non-canonical (e.g. `static private`). Expected order: `private static abstract|final|override <name>`. |
| `[TYA-E0410]`  | parser  | v0.46    | Bare `@@name` declaration or access inside a class. The `@@` form is removed in v0.46; use `static name = ...` (declaration) and `Self.name` / `ClassName.name` (access). |

`_init` is no longer special. `private init = ...` is the v0.46
spelling for private constructors and is enforced by the same
`init`/`_init` mutex check (now `init` / `private init`).

## Implementation Sketch

- **Lexer:** add `private` and `static` to a reserved-word table.
  `Self` is a new IDENT — it tokenizes as a regular IDENT, the
  parser dispatches on its name. (Avoids a hard reserved word for
  Self until v0.47, when bare uses of `Self` outside class context
  can be diagnosed cleanly.)
- **Parser:** class-body statement dispatch grows a modifier prefix
  loop accepting `private` and `static` in canonical order. AST
  ClassMember nodes already have boolean flags for `Class` (we
  already use this for `@@`); rename to `Static` and add `Private`.
  Drop the implicit `_`-prefix → `Private` derivation in the
  checker.
- **Checker:** replace `isPrivateName(name)` calls with reads of
  the new `Private` flag on the AST node / classInfo entry.
  `Self.foo` resolves to `<current class>.foo`. Outside class
  scope, `Self` raises `[TYA-E0411]` (reserve the code).
- **C emitter:** unchanged in spirit. `static` declarations lower
  to the same C function/global as `@@` does today. `Self.foo`
  emits the same C as `<ClassName>.foo` does today.
- **Formatter:** emit `private` / `static` keywords in canonical
  order. Reject `_`-prefixed class members per [TYA-E0407].

## Acceptance

- `go test ./... -count=1` green; both self-host gates green.
- New fixtures under `tests/testdata/v46/`:
  - `private_keyword.txtar` — every class-member shape with and
    without `private`.
  - `static_keyword.txtar` — every class-level declaration and
    `Self.` access shape.
  - `combined_modifiers.txtar` — `private static`, `private
    abstract`, `override`, `final` combinations.
  - `migration_diagnostics.txtar` — every E0407 / E0410 case.
- `examples/`, `stdlib/`, and the v01 self-host **(see
  Self-Host Constraint)** carry no `_`-prefixed class members and
  no `@@` declarations after v0.46 lands.

## Self-Host Constraint

`selfhost/v01/compiler.tya` uses `_`-prefixed class members (for
example `_v01_reserved_name`) and a small amount of `@@`. Two
options:

1. Pre-migrate v01 to the new syntax (v01 stays the v0.43-surface
   fixed point, but with the v0.46 visibility / static keywords).
   Self-host gate runs against v01 unchanged in semantics, just
   different syntax.
2. Defer v0.46 surface adoption in v01 to the v0.4x self-host
   upgrade, where v02 lands directly on the v0.44 + v0.46 surface.

Option 2 is cleaner — v01 stays frozen at the syntax it was written
in. The v01 lexer/parser keeps recognizing `_foo` as private and
`@@foo` as static at the v01 surface only, while the Go reference
implementation moves to the v0.46 surface for everything else.

## Open Questions

**Q1.** Should module-level private bindings (`module foo; _helper`)
also migrate to a `private` keyword in v0.46, or does that ride
along with v0.4x M9 (module keyword removal)?

Recommendation: ride along with M9. Module-level `_` prefix is
disappearing anyway when modules go away.

**Q2.** `Self.` capital S — too close to `self`? Alternatives:
`@@`-as-receiver (`@@.count`), `class.count` (lowercase `class`
keyword as receiver — but `class` is already taken in declaration
position), `this_class.count` (verbose). Decision: stick with
`Self.` unless usability testing on real code says otherwise.

**Q3.** Do we keep `_init` working as a deprecated alias for
`private init` for one minor? Recommendation: no. Clean cut at
v0.46 with `[TYA-E0407]` pointing at every site. Tya's track
record is clean cuts, not deprecation windows.

**Q4.** `_`-prefixed names as **public** members — allowed?
Recommendation: yes. `_` becomes purely a name shape, not a
visibility marker, inside classes. (Module-level keeps the
existing rule until M9.)

## Cross-References

- [`docs/v0.44/SPEC.md`](../v0.44/SPEC.md) — the class model.
- [`docs/v0.45/SPEC.md`](../v0.45/SPEC.md) — cross-file private
  enforcement (defines `[TYA-E0406]` which v0.46 builds on).
- [`docs/NAMING.md`](../NAMING.md) — file/identifier shape rules.
- [`ROADMAP.md`](../../ROADMAP.md) — v0.46 entry will be added
  under `Scheduled` once G2 is decided.
