---
status: completed
goal_ready: false
---

# Feature: Grow Interface Toward Stackable-Trait Capability

## Goal

Grow `interface` into Tya's stackable behavior composition mechanism while
keeping `interface` as the only user-facing keyword. Implement default methods,
interface-contributed fields, zero-arity interface initializers, deterministic
diamond handling, and `super()` across class inheritance and stacked interface
defaults.

## Context

`docs/v0.61/SPEC.md` is the implementation specification for this feature. Tya
already has explicit interfaces and single class inheritance. This feature
extends interfaces from body-free contracts into reusable behavior units without
adding a separate `trait` keyword.

Some v0.61 behavior may already exist in the current codebase. This PRD defines
the whole acceptance surface for the feature, not only the remaining gaps.

## Behavior

- Keep `interface` as the single public concept and syntax.
- Do not add `trait` as a keyword or alias.
- Preserve explicit `implements`; no implicit conformance.
- Preserve single class inheritance.
- Allow interface members:
  - body-free instance method requirements;
  - default instance methods;
  - instance field declarations;
  - zero-arity `initialize` hooks.
- Reject interface members that are out of scope:
  - static fields;
  - static methods;
  - private members;
  - nested classes;
  - nested interfaces;
  - interface `initialize` hooks with parameters.
- A class method overrides interface defaults with the same name.
- A default method can satisfy a compatible body-free requirement from another
  interface.
- Interface fields are composed into implementing class instances.
- A class field declaration resolves same-name interface field conflicts.
- Interface field initializers may not reference `self`.
- Interface initializer hooks may reference `self`.
- Interface initializer hooks run once per interface identity.
- If a class declares `initialize` and implements interfaces with fields or
  initializer hooks, the class constructor must call `super()` so interface
  initialization runs.
- In a root class constructor, `super()` means "run interface initialization".
- In a subclass constructor, `super(args...)` runs parent construction first,
  then the subclass's newly implemented interface initialization.
- Effective interface initialization order is depth-first, left-to-right,
  postorder, with duplicate interface identities removed.
- `super()` in an interface default method calls the next implementation of the
  same method in the effective interface stack.
- Method stack order is Scala-like: in `implements A, B`, the rightmost
  interface wraps the interfaces to its left.
- Class inheritance takes precedence over interface defaults.
- `super()` in a class override calls the parent class method first; if no
  parent class method exists, it enters the interface default stack.
- Duplicate body-free requirements with the same arity are compatible.
- Same-name methods with different arities are conflicts unless resolved by an
  explicit class or child-interface method.
- Unrelated same-name defaults are conflicts unless the implementing class or a
  combining child interface declares the method explicitly.
- Duplicate inheritance of the same source default through a diamond is
  compatible and contributes once.
- Conflicts are diagnosed before C emission.

## Scope

- Parser and AST support for interface default methods, fields, and
  `initialize` hooks.
- Formatter support for the new interface member shapes.
- Checker interface model, effective interface graph, conflict resolution, and
  stable diagnostics.
- C codegen for interface default methods, field initialization, initializer
  hooks, and `super()` lowering through interface stacks.
- Runtime helpers only where codegen cannot encode next-method targets
  statically.
- Self-host parser/checker/codegen parity where required to preserve the
  fixed-point invariant.
- Tests for old interface behavior and new v0.61 behavior.
- `docs/SPEC.md`, `docs/STDLIB.md` or version docs as needed.
- `ROADMAP.md`.

## Out of Scope

- Adding a `trait` keyword.
- Supporting both `interface` and `trait` as synonyms.
- Implicit interfaces generated from classes.
- Implementing a class as if it were an interface.
- Multiple class inheritance.
- Method overloading.
- Generics.
- Operator-overload traits.
- Static or private interface members.
- Public runtime reflection APIs for interface composition.

## Acceptance Criteria

- Existing v0.11 and v0.12 interface behavior remains compatible.
- Interface methods may be body-free requirements or default methods.
- Interface fields are composed into implementing classes.
- Interface field initializers run per instance and cannot reference `self`.
- Interface initializer hooks are zero-arity and run in deterministic order.
- Class constructors that must run interface initialization are rejected unless
  they call `super()`.
- Root-class constructor `super()` runs the interface initialization chain.
- `super()` works through class inheritance and stacked interface defaults.
- Class members explicitly override interface methods and fields.
- Interface inheritance diamonds are deterministic and de-duplicated.
- Same-name method arity conflicts produce stable diagnostics.
- Same-name field conflicts produce stable diagnostics.
- Invalid interface members produce stable diagnostics.
- `super()` with no next method in the interface stack is rejected.
- Generated C runs examples for:
  - default methods;
  - default satisfying a requirement;
  - stateful interfaces;
  - initializer order;
  - class override calling interface `super()`;
  - stacked interface `super()`;
  - diamond de-duplication;
  - conflict diagnostics.
- The self-host fixed point remains green.

## Verification

Focused checks for old and new interface behavior:

```sh
go test ./tests -run 'TestV(11|12|61).*Script' -count=1
```

Self-host invariant:

```sh
go test ./tests -run TestSelfhostV01Scripts -count=1
```

Full project check:

```sh
go test ./... -count=1
```

## Dependencies

- Use `docs/v0.61/SPEC.md` as the implementation spec.
- Preserve compatibility with the existing self-host fixed-point gate.
- Keep the `interface` naming decision aligned with Canonical Syntax: one
  concept, one spelling.

## Open Questions

None.
