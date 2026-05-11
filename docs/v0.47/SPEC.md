# Tya v0.47 Specification

> **Status:** shipped. The clean cut of the legacy v0.45 class-
> member surface is in place. The `tya version` constant is
> `0.47.0`. `selfhost/v01/` keeps the v0.43 surface via a path-
> based permissive-legacy mode on the checker.
>
> Not all SPEC goals shipped in v0.47.0 — G1 (strict bare-name
> receivers) is partially deferred: bare names that previously
> resolved through legacy paths now raise E0410 at the parse site,
> but the checker's class-member fallback elimination is a future
> step. G6 (canonical `Self.foo` formatter rewrite) is deferred to
> a follow-up minor.

## Theme

v0.46 introduced the keyword-based class-member surface (`private`,
`static`, `self`, `Self`, `initialize`) **alongside** the legacy
sigil-based surface (`@`, `@@`, `_`-prefix, `init`). v0.47 is the
**clean-cut release** that:

1. Lands G4 — bare identifiers in method bodies resolve to locals /
   parameters / imports only, never to class members.
2. Retires the legacy v0.45 syntax. Parser and checker emit
   structured diagnostics for every legacy form.
3. Migrates `tests/testdata/v0[6-9]…v45/` fixtures that exercise
   the legacy surface either to the v0.46 form (for tests that just
   happen to use legacy syntax) or to dedicated "legacy diagnostic"
   fixtures (for tests that pin the legacy error messages).

After v0.47 the v0.46 keyword surface is the **only** way to write
a class.

## Goals

- **G1.** Implement **G4** from v0.46: bare identifiers in class
  method bodies resolve to locals / parameters / imports only.
  `count = count + 1` always touches a local, never `Self.count`.
- **G2.** Parser rejects `@field` and `@@field` with `[TYA-E0410]`.
- **G3.** Checker rejects `_`-prefix as a privacy marker on class
  members with `[TYA-E0407]`. (Module-level `_` privacy is
  unaffected — it stays until M9 retires modules entirely.)
- **G4.** Checker rejects `init` / `_init` as constructor names
  with `[TYA-E0414]`; only `initialize` is recognized.
- **G5.** Checker rejects `self` inside `static` methods with
  `[TYA-E0411]` (was: warning during v0.46 transition).
- **G6.** Formatter rewrites `<DeclaringClass>.foo` → `Self.foo`
  inside the declaring class body (canonical receiver rule);
  emits `[TYA-E0413]` under `--check-unused` strict mode.
- **G7.** Migrate every `tests/testdata/v*/` fixture using legacy
  syntax to either the v0.46 form (when the test does not pin
  legacy-specific behavior) or to `tests/testdata/legacy/`
  (when the test exists specifically to verify a legacy diagnostic).

## Non-Goals

- Module-level `_` privacy (`module foo; _helper`) — stays until
  v0.4x M9 retires modules.
- Self-host v02 upgrade — still part of the v0.4x Epic.
- Module keyword removal — still v0.4x M9.

## G1 — Strict bare-name receivers (former v0.46 G4)

### Rule

Inside a class method body (instance method, static method, or
constructor), a bare identifier resolves only to:

1. A local variable bound earlier in the method.
2. A parameter.
3. An imported binding visible at module scope (e.g. `print`,
   `len`, a built-in, an `import`-ed package alias, etc.).

Bare identifiers **never** resolve to a class member, even when a
method of the same name exists on `self` or `Self`. To access a
class member, write an explicit receiver: `self.x`, `Self.x`, or
`<OtherClass>.x`.

### Why

Java falls back from bare name to `this.field` and class statics
because the type system can disambiguate. Tya is dynamically typed
and cannot. The bare-name fallback silently turns typos into new
locals (or vice versa). G1 makes the fallback impossible so the
hazard cannot happen.

### Symmetric reads / writes

`count = count + 1` inside any method body:
- creates or updates a local `count`,
- never reads or writes `Self.count`,
- raises `undefined variable count` if no local or import named
  `count` exists in scope.

To update a class field, write `Self.count = Self.count + 1`. To
read it, write `Self.count`.

### Implementation

Checker `checkExpr` for `*ast.Ident` already walks scope to find
the binding. The current code falls back to "<currentModule>.<Name>"
for class-name resolution within a package (v0.44 surface). G1
keeps that fallback for **class names** (PascalCase identifiers
that resolve to a class), but drops any "is this a method or
field?" fallback. Concretely:

- `User()` — bare PascalCase resolves to a class for construction.
- `length` (lowercase) — bare snake_case resolves only to local /
  param / import. If a method or field `length` exists on the
  current class, it is invisible from a bare reference.

## G2 — Reject `@field` / `@@field`

Parser path that consumes `AT` token in expression position emits
`[TYA-E0410]`:

```
3:5: [TYA-E0410] @name is removed; use self.name (was: @name)
```

For `@@name` (two AT tokens) the message points at the lexeme:

```
3:5: [TYA-E0410] @@name is removed; use Self.name (was: @@name)
```

`AssignStmt` `@field = value` and `@@field = value` produce the
same diagnostic.

### Implementation

- Remove the AT-prefix branches from parser.primary() and
  parser.assignTarget() in `internal/parser/parser.go`.
- Lex remains unchanged — AT is still a token, the parser just
  always treats it as a syntax error.

## G3 — Reject `_`-prefix privacy on class members

Checker, when registering a class member:

- A member name starting with `_` is **not** marked private.
- The checker emits `[TYA-E0407]` pointing at the declaration:

```
5:3: [TYA-E0407] _id is no longer a privacy marker on class
                  members; rename to `private id` or `id`
```

The leading underscore on a class member name is now purely
stylistic. (Module-level `_` privacy continues to work until M9.)

### Implementation

- Parser drops the transitional rule that maps `_`-prefix → Private
  on class members.
- Checker `predeclareModuleClass` / `predeclareClass` /
  `collectPrivateClassMembers` already read the AST's Private flag;
  no further change there.
- Add the new diagnostic emission in `checker.CheckAll` or during
  class predeclaration.

## G4 — Reject `init` / `_init` as constructor names

Checker, when registering a class method:

- `init` and `_init` are accepted as regular method names (no
  constructor semantics).
- If the class contains an `init` or `_init` method, emit
  `[TYA-E0414]` pointing at the declaration:

```
5:3: [TYA-E0414] `init` is removed as a constructor name; rename
                  to `initialize`
```

`initialize` remains the only constructor.

### Implementation

- Drop `init` / `_init` from the constructor-detection sites listed
  in v0.46's commit log:
  - `internal/checker/checker.go` (4 sites)
  - `internal/codegen/c.go` (5 sites)
- `inheritedMethodSym`'s alias list reduces to just `initialize`.
- Add E0414 diagnostic at the class-predeclaration site.

## G5 — Reject `self` inside `static` methods

Checker, when seeing `*ast.SelfExpr{Class:false}` inside a
`static`-method scope:

```
4:5: [TYA-E0411] self is not available in static methods (no
                  instance receiver); use Self for the class
```

This was a warning during the v0.46 transition; v0.47 makes it
an error.

### Implementation

- The transitional checker accepts `self` in any class-method or
  instance-method context. Tighten to instance methods +
  constructors only.

## G6 — Canonical receiver rule

When the formatter sees a `MemberExpr{Target: Ident("<C>"), Name: m}`
where `<C>` is the **declaring class name** of the surrounding
class body, rewrite it to `Self.m`.

Checker emits `[TYA-E0413]` under `--check-unused` strict mode for
the pre-rewrite form so CI can detect drift even without invoking
the formatter.

### Implementation

- Formatter pass tracks the current class name as it walks. On
  every `MemberExpr` whose Target is the current class name, swap
  to a SelfExpr{Class:true}.
- Checker tracks the same context and emits E0413 when the rewrite
  would apply.

## G7 — Test fixture migration

`tests/testdata/v0[6-9]` through `tests/testdata/v45/` contain
roughly 33 fixtures that use legacy class syntax.

Classification:

- **Migrate to v0.46 syntax** (when the test happens to use legacy
  syntax but is not testing legacy behavior). Most fixtures fall
  here.
- **Move to `tests/testdata/legacy/`** (when the test pins a legacy
  diagnostic message or asserts a legacy-only behavior). E.g.
  `v09/private_members.txtar` tests external `_name` access
  rejection — the v0.47 form `private name` cannot be statically
  rejected externally (dynamic typing, no receiver type), so the
  test as-written has no v0.46 equivalent. Move it to legacy/ and
  add an assertion that the source file produces `[TYA-E0407]`.
- **Add new v47 fixtures** for each new diagnostic (E0407, E0410,
  E0411, E0412, E0413, E0414).

### Self-host

`selfhost/v01/compiler.tya` continues to use v0.43 surface
(`@`, `@@`, `_`-prefix, `init`). The Go reference implementation
must keep accepting that surface for files under `selfhost/v01/`.

Path-based exemption: parser / checker take a flag (or examine the
file path) and skip G2 / G3 / G4 / G5 diagnostics for paths under
`selfhost/v01/`. The exemption applies to ALL files in that
directory, including any user-supplied test sources that the v01
self-host compiles in its txtar.

The v02 self-host (in progress under the v0.4x Epic) targets the
v0.46+ surface directly and is not exempted.

## Migration matrix

Same as v0.46 — but in v0.47 the legacy column raises diagnostics.

| Legacy (now [TYA-E…])  | v0.46+ canonical               |
| ---------------------- | ------------------------------ |
| `@name` → E0410         | `self.name`                    |
| `@name = x` → E0410     | `self.name = x`                |
| `@@count` → E0410       | `Self.count`                   |
| `@@count = x` → E0410   | `Self.count = x`               |
| `@@count = 0` → E0410   | `static count = 0`             |
| `_name = ...` → E0407   | `private name = ...`           |
| `init = name -> ...` → E0414 | `initialize = name -> ...` |
| `_init = name -> ...` → E0414, E0407 | `private initialize = name -> ...` |

## Diagnostics reference

| Code           | Stage   | Condition                                              |
| -------------- | ------- | ------------------------------------------------------ |
| `[TYA-E0407]`  | checker | Class member name begins with `_`.                     |
| `[TYA-E0408]`  | parser  | `private` keyword used outside a class body.           |
| `[TYA-E0409]`  | parser  | Modifier order is non-canonical.                       |
| `[TYA-E0410]`  | parser  | `@` or `@@` sigil used.                                |
| `[TYA-E0411]`  | checker | `self` used inside a `static` method.                  |
| `[TYA-E0412]`  | checker | `Self` used outside a class body.                      |
| `[TYA-E0413]`  | checker | `<DeclaringClass>.foo` written inside its own class body. |
| `[TYA-E0414]`  | checker | Class declares `init` / `_init`; expected `initialize`. |
| `[TYA-E0415]`  | checker | `this` reserved-name usage (reserved for future).     |

E0408 / E0409 / E0415 are already reserved in v0.46 SPEC and stay
reserved (no wiring change in v0.47 unless their condition arises).

## Acceptance

- `go test ./... -count=1` green; both self-host gates green.
- `tests/testdata/v47/` covers each new diagnostic.
- `tests/testdata/legacy/` houses any preserved legacy-behavior
  fixtures.
- `examples/`, `stdlib/`, and the v0.46+ portion of the Go
  reference implementation carry no legacy class syntax.
- `selfhost/v01/` is untouched and continues to compile itself to
  a stable stage-2/stage-3 fixed point.

## Open Questions

**Q1.** Should v0.47 also enforce **G6 fully** (formatter rewrite +
checker strict-mode E0413), or only emit E0413 as a non-strict
warning and defer the formatter rewrite to a follow-up? The SPEC
above plans full enforcement; the alternative is to ship strict-mode
warning + formatter rewrite as opt-in.

**Q2.** `init` constructor is widely used in user-facing examples
in older Tya documentation. Should v0.47 ship a `tya format
--migrate` subcommand to mechanically rewrite legacy syntax to
v0.46+? Recommended: yes, as a separate Goal G8 if time permits.

**Q3.** `_init` private constructor in legacy code maps to two
diagnostics (E0407 + E0414). Should the parser combine them into a
single E0414 with "rename to `private initialize`" hint? Cleaner
UX; minor extra logic in the diagnostic emitter.

## Cross-References

- [`docs/v0.46/SPEC.md`](../v0.46/SPEC.md) — the surface this clean
  cut finalizes.
- [`docs/v0.46/RELEASE_NOTES.md`](../v0.46/RELEASE_NOTES.md) — the
  transitional release listing what v0.47 retires.
- [`ROADMAP.md`](../../ROADMAP.md) — `Scheduled` § v0.47 entry will
  mirror this document once design is settled.
