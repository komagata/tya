# Feature: Implicit Self Method Calls

## Goal

Instance and static methods should be callable from the same class context
without spelling `self.` or `Self.`, while field storage becomes explicit and
statically declared.

## Context

Current Tya code must call methods on the current instance as `self.foo(...)`
and current-class static methods as `Self.foo(...)`. This is noisier than the
intended class style, especially in stdlib classes with many helper methods.

Java-like lookup is the intended model for calls: an unqualified method call in
an instance method can target the receiver, and an unqualified method call in a
static method can target the class. Tya should differ from Java by rejecting
local or parameter names that collide with instance field names, keeping field
access unambiguous.

The current language also permits fields to be created dynamically through
`self.<name> = ...` inside `initialize` or instance methods. That behavior is
no longer desired. Instance fields should be declared in the class body before
they can be assigned through `self.<name>`.

## Behavior

- Inside `initialize` and instance methods, `foo(args)` resolves to
  `self.foo(args)` when `foo` is an instance method on the current class, an
  inherited parent class, or an implemented interface default method.
- Inside static methods, `foo(args)` resolves to `Self.foo(args)` when `foo` is
  a static method on the current class or an inherited parent class.
- Unqualified calls keep existing local/top-level/function/class lookup first.
  The implicit receiver lookup applies only when ordinary callable lookup does
  not resolve the callee.
- Bare method references are not introduced. `value = foo` must not become
  `value = self.foo` or `value = Self.foo`.
- Field reads and writes remain explicit. Code must continue to use
  `self.field` for instance fields and `Self.NAME` or other existing class
  member forms for class-level data.
- Assigning `self.<field> = value` is valid only when `<field>` is declared as
  an instance field in the class body or contributed by an interface field.
- Assigning `self.<field> = value` to an undeclared field is a checker error,
  including inside `initialize`.
- Parameters and local bindings inside instance methods must not use the same
  name as any instance field declared on the current class, inherited from a
  parent class, or contributed by implemented interfaces.
- Method names and field names remain separate member namespaces for call
  syntax, but field/local name collisions are rejected to keep unqualified
  names predictable.
- Static methods do not gain `self`; they only gain implicit `Self` calls to
  static methods.

## Scope

- Parser changes only if the existing AST cannot represent the needed call
  shape. Prefer preserving the current `CallExpr`/`Ident` surface and resolving
  during checking/emission.
- Checker lookup for unqualified method calls in instance and static class
  contexts.
- Checker validation for `self.<field> = ...` assignments against declared or
  interface-contributed instance fields.
- Checker validation that method parameters and local bindings do not collide
  with effective instance field names.
- Evaluator support for implicitly resolved receiver calls.
- C emitter/runtime support for the same behavior as evaluator execution.
- Formatter behavior only if generated or rewritten source would otherwise
  produce a non-canonical representation.
- `docs/SPEC.md`, `docs/GUIDE.md`, and diagnostics docs where the new field
  declaration and implicit call rules need to be documented.
- `lib/**/*.tya` migration:
  - replace same-class instance method calls like `self.foo(...)` with
    `foo(...)`;
  - replace same-class static method calls like `Self.foo(...)` with `foo(...)`
    when the target is a static method;
  - keep `self.field` reads and writes explicit;
  - add class-body field declarations for stdlib fields currently first
    created through `self.<field> = ...`.
- Relevant tests, examples, and stdlib tests that need updating after the
  stricter field rules.

## Out of Scope

- Implicit receiver lookup for bare method references such as `f = foo`.
- Implicit receiver lookup outside class methods.
- Implicit `self` in static methods.
- Implicit `Self` for class constants, class variables, or class metadata.
- Allowing dynamic instance field creation through assignment.
- Allowing Java-style parameter or local shadowing of instance fields.
- Changing public method dispatch, method override rules, interface arity
  rules, or visibility semantics except where needed for implicit call lookup.
- Rewriting archived examples under `docs/archive/pre-v0.1/` or other
  historical material.

## Acceptance Criteria

- An instance method can call another instance method in the same class as
  `foo(args)` without `self.`.
- An instance method can call an inherited parent instance method as
  `foo(args)` without `self.`.
- An instance method can call an implemented interface default method as
  `foo(args)` without `self.`.
- A static method can call another static method in the same class as
  `foo(args)` without `Self.`.
- A static method can call an inherited parent static method as `foo(args)`
  without `Self.`.
- `foo(args)` does not resolve to an implicit receiver when an ordinary local,
  top-level, function, class, or module callable named `foo` is in scope.
- `value = foo` remains an ordinary identifier reference and does not become a
  receiver method reference.
- `self.undeclared = value` is rejected in `initialize` and ordinary instance
  methods.
- Assigning to a declared class-body field through `self.field = value`
  remains valid.
- Assigning to an interface-contributed field through `self.field = value`
  remains valid for implementing classes.
- A method parameter with the same name as an effective instance field is
  rejected.
- A local binding with the same name as an effective instance field is
  rejected.
- `lib/**/*.tya` no longer contains unnecessary `self.` or `Self.` prefixes
  for same-class or inherited method calls covered by this feature.
- `lib/**/*.tya` no longer creates undeclared instance fields through
  `self.<field> = ...`.
- Evaluator and generated-C execution agree for implicit instance calls,
  implicit static calls, inherited calls, interface default calls, and rejected
  field/local collisions.
- The maintained self-host fixed-point invariant is not regressed.

## Verification

```sh
gofmt -w <changed-go-files>
go test ./internal/checker ./internal/eval ./internal/codegen -count=1
go test ./tests -run 'TestV44Scripts|TestV61Scripts|TestSelfhostV01Scripts' -count=1
go test ./... -count=1
```
