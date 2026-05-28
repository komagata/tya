# Feature: Bare instance member access

## Goal

Allow instance methods to read same-class instance fields through bare names
when no local binding or parameter with that name is in scope, so method bodies
can use the same concise receiver style already used for same-class method
calls.

## Context

GitHub issue #27 reports that this class fails to check:

```tya
class Command
  arguments: []

  argument: spec ->
    arguments.push(spec)
    self
```

The desired style is for `arguments.push(spec)` inside an instance method to
resolve to the instance field `self.arguments` when `arguments` is not a local
binding or parameter.

Current `docs/SPEC.md` already allows unqualified same-class instance method
calls such as `helper(args)` to resolve to `self.helper(args)`. Field reads and
writes are currently explicit. This feature extends bare lookup to instance
field reads only, while keeping local/parameter lookup predictable.

Relevant files include:

- `docs/SPEC.md`
- `docs/GUIDE.md`
- `internal/checker/checker.go`
- `internal/eval/eval.go`
- `internal/codegen/c.go`
- `internal/formatter/unparse.go`
- `internal/formatter/unparse_test.go`
- class/member tests under `internal/checker`, `internal/eval`,
  `internal/codegen`, and `tests/`

## Behavior

- Inside `initialize` and instance methods, a bare identifier that names a
  declared effective instance field resolves to `self.<field>` when ordinary
  lexical lookup does not find a local binding, parameter, function, import, or
  other ordinary binding with that name.
- Effective instance fields include fields declared on the current class,
  inherited fields, and interface-contributed fields.
- Local bindings and parameters take precedence over fields with the same name.
  In that case, the field remains accessible as `self.<field>`.
- Bare field reads may be used as a member target, so `arguments.push(spec)`
  resolves like `self.arguments.push(spec)` when `arguments` is an effective
  instance field and no local or parameter named `arguments` is in scope.
- Bare field reads may be used as ordinary value expressions, so `arguments`
  returns the same value as `self.arguments` under the same lookup rule.
- Field writes remain explicit. Assignments must continue to use
  `self.<field> = value`; `field = value` creates or updates an ordinary local
  binding according to existing assignment rules.
- Bare method references are not introduced. A bare identifier that names only
  an instance method still does not become a callable receiver-bound method
  value.
- Static methods do not gain bare instance field access.
- Formatter output should prefer the bare field-read form inside the declaring
  class when the access is unambiguous and semantically equivalent.

Example:

```tya
class Command
  arguments: []

  argument: spec ->
    arguments.push(spec)
    self

  replace_arguments: arguments ->
    self.arguments = arguments
```

In `argument`, `arguments` is the field. In `replace_arguments`, `arguments` is
the parameter and `self.arguments` is the field.

## Scope

- Checker lookup for bare identifiers and member targets inside instance
  methods.
- Evaluator support for bare field reads in instance method execution.
- C emitter/runtime support for the same behavior as evaluator execution.
- Formatter canonicalization from same-class `self.field` reads to bare
  `field` only where local/parameter shadowing does not change meaning.
- Docs updates for class member lookup and field access rules.
- Focused tests for bare field reads, member-call targets, local/parameter
  shadowing, assignment behavior, inherited fields, interface fields, formatter
  canonicalization, evaluator/codegen parity, and self-host invariants.

## Out of Scope

- Bare field writes such as `field = value` assigning to `self.field`.
- Bare instance field access outside instance methods and `initialize`.
- Bare instance field access in static methods.
- Bare class constant or class variable access beyond existing canonical
  `Self.NAME` / bare constant rules.
- Bare method references such as `handler = save`.
- Changing ordinary local/parameter lookup precedence.
- Rejecting local or parameter names that shadow instance fields.
- Introducing properties, getters, setters, or new field declaration syntax.

## Acceptance Criteria

- The issue #27 reproduction checks, runs, and appends to the instance
  `arguments` field.
- A bare field read inside an instance method returns the same value as
  `self.field` when there is no local or parameter with that name.
- `field.method(args)` works as a member call on `self.field` under the same
  lookup rule.
- A parameter with the same name as an instance field takes precedence over the
  field; `self.field` still accesses the field.
- A local binding with the same name as an instance field takes precedence over
  the field after the local is introduced; `self.field` still accesses the
  field.
- `field = value` inside an instance method does not assign to `self.field`.
- Static methods cannot read instance fields through bare names.
- Inherited fields and interface-contributed fields are available through bare
  reads inside instance methods.
- Formatter rewrites unambiguous same-class `self.field` reads to `field`, but
  does not rewrite when a local or parameter named `field` is in scope.
- Evaluator and generated-C execution agree for all accepted cases.
- The maintained self-host fixed-point invariant is not regressed.

## Verification

```sh
gofmt -w internal/checker/checker.go internal/eval/eval.go internal/codegen/c.go internal/formatter/unparse.go
go test ./internal/checker ./internal/eval ./internal/codegen ./internal/formatter -run 'Field|Member|Self|Format' -count=1
go test ./tests -run 'TestV44Scripts|TestV61Scripts|TestSelfhostV01Scripts' -count=1
go test ./... -count=1
```
