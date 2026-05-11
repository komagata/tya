# Tya v0.48 Specification

> **Status:** shipped. The `tya version` constant is `0.48.0`.
> v0.48 lands the two items deferred from v0.47:
>
> - **G1 strict bare-name receivers** — already enforced as a side
>   effect of v0.46's G2/G4. v0.48 documents this and confirms the
>   behavior is intentional. No code change required.
> - **G6 canonical receiver rule** — formatter rewrites
>   `<DeclaringClass>.foo` → `Self.foo` inside the declaring class
>   body. Checker emits `[TYA-E0413]` under `--check-unused` strict
>   mode for the same shape.

## Theme

v0.47 retired the legacy v0.45 class-member sigils with structured
diagnostics. v0.48 closes the v0.46/v0.47 arc by:

1. Codifying the bare-name receiver rule (G1) — bare identifiers
   resolve to locals / params / imports only, never to class
   members.
2. Making `Self.foo` the canonical receiver form inside the
   declaring class body (G6) via formatter rewrite + strict-mode
   warning.

After v0.48 the class-member surface is fully canonical: one way
to spell every form.

## G1 — strict bare-name receivers (codified)

Inside any class method body (instance method, static method, or
constructor), a bare identifier resolves only to:

1. A local variable bound earlier in the method.
2. A parameter.
3. An imported binding visible at module scope (e.g. `print`,
   `len`, an `import`-ed package alias).

Bare identifiers **never** resolve to a class member, even when a
method or field of the same name exists on `self` or `Self`. To
access a class member, write an explicit receiver: `self.x`,
`Self.x`, or `<OtherClass>.x`.

This was already enforced from v0.46 onward — the v0.46 G2
keyword surface added explicit `self.`/`Self.` receivers and
v0.46 G4 (which v0.47 confirmed) eliminated bare-name → class
member fallback. v0.48 simply records that the rule is settled.

```tya
class Counter
  count = 0       # instance field

  bump = ->
    count = count + 1   # error: undefined variable count
                        # (bare `count` is local-only; no field fallback)
    self.count = self.count + 1   # canonical write
```

## G6 — canonical receiver rule

### Rule

Inside the body of `class C`, an access to one of C's own members
written as `C.foo` is **non-canonical**. The canonical form is
`Self.foo`. References to *other* classes from inside C
(e.g. `User.foo` from inside `class Admin`) remain unchanged —
the rule only fires for the lexically declaring class name.

### Formatter rewrite

`tya format` rewrites every `MemberExpr{Target: Ident("C"), Name: x}`
inside `class C` to `Self.x`. Combined with the existing v0.48
formatter passes for the v0.46 keyword surface, the output is
fully canonical:

```tya
# input
class Counter
  @@count = 0

  @@bump = ->
    Counter.count = Counter.count + 1
```

```tya
# output of `tya format`
class Counter
  static count = 0

  static bump = () ->
    Self.count = Self.count + 1
```

### Strict-mode diagnostic

`tya check --check-unused` (strict mode) emits
`[TYA-E0413]` for every non-canonical access at WARNING severity:

```
4:5: [TYA-E0413] `Counter.count` inside its own class body is
                 non-canonical; write `Self.count`.
```

CI can detect drift even without running the formatter.

## Formatter v0.46+ keyword surface support

v0.48 also completes the formatter's support for emitting the
v0.46 keyword surface (was incomplete in v0.46/v0.47 — the
formatter still emitted `@@`/`@` in some paths). The formatter
now consistently emits:

- `private name = value` instead of `_name = value`
- `static name = value` instead of `@@name = value`
- `self.name` instead of `@name`
- `Self.name` instead of `@@name` (access)
- `initialize` instead of `init` / `_init` (constructor)
- `private static`, `static override`, `static abstract`, etc. in
  canonical modifier order

## Diagnostics

| Code           | Stage   | Wired in | Condition                                              |
| -------------- | ------- | -------- | ------------------------------------------------------ |
| `[TYA-E0413]`  | checker | v0.48    | `<DeclaringClass>.foo` written inside its own class body (strict mode warning). |

The v0.47 codes (E0407, E0410, E0411, E0412, E0414) remain in
place unchanged.

## Self-Host

The selfhost/v01/ permissive-legacy path exemption introduced in
v0.47 continues to apply; v01 keeps its v0.43 surface. v0.48
diagnostics are gated by `permissiveLegacy` only where
behaviorally appropriate (E0413 is a strict-mode warning that
points at legacy-style references — it remains active in
permissive mode as well since the rewrite hint is useful even for
v01 readers).

## Verification

```sh
go test ./... -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./tests -run TestV46Scripts -count=1
```

All green at the v0.48.0 tag.

## Cross-References

- [`docs/v0.47/SPEC.md`](../v0.47/SPEC.md) — clean-cut release.
- [`docs/v0.46/SPEC.md`](../v0.46/SPEC.md) — the keyword surface
  that v0.48 canonicalizes.
- [`ROADMAP.md`](../../ROADMAP.md) — `Released` § v0.48.
