# Feature: Unambiguous Dynamic Semantics

## Goal

Clarify the remaining dynamic Tya semantics so users have one obvious way to
write and reason about common operations. Tya remains dynamically typed; this
spec only tightens runtime and checker behavior that was ambiguous.

## Context

The current language direction is "a language without hesitation". The user
approved these choices for the current dynamic language surface, not for the
static typing discussion. This spec intentionally includes the tests a future
implementation must add, but it does not implement the behavior itself.

Self-host compatibility is required. Any implementation must preserve the v01
and v02 self-host fixed point, updating self-host source first when a stricter
rule would reject existing self-host code.

## Behavior

- Dictionary key member access is prohibited.
- Dictionary keys are read and written with string indexes, for example
  `user["name"]`.
- Dictionary receiver methods remain valid dot calls, for example
  `user.keys()`, `user.has?("name")`, `user.delete("age")`, and
  `user.equal?(other)`.
- Missing dictionary keys through index access return `nil`.
- Unknown member or method access on values without a documented member surface
  is an error.
- `and` and `or` short-circuit.
- `and` and `or` return `Bool`, not operand values.
- Truthiness remains: only `nil` and `false` are falsey; every other value is
  truthy.
- A function without an explicit `return` returns the final evaluated statement
  or expression.
- Same-scope variable reassignment remains allowed.
- Function bindings may be reassigned like ordinary variable bindings.
- Reassignment may move to or from `nil`, but `nil` does not erase the last
  known concrete kind. `x = 1; x = nil; x = "one"` is invalid.
- Direct reassignment between two different concrete non-`nil` kinds remains
  invalid.
- Block-local bindings created inside `if`, `while`, `for`, `catch`,
  `match case`, `scope`, and `select` bodies do not leak outside the body.
- Assigning to an existing outer non-function binding from a nested block
  updates the outer binding.
- `for` value and index bindings are local to the loop body.
- `catch` bindings are local to the catch body.
- `match case` pattern bindings are local to the case body.
- Standard-library invalid argument kinds and arity are runtime errors.
- Standard-library absence and lookup APIs return `nil` only where documented.
- Standard-library outside-world failures, such as file, process, network,
  compression, digest, time, and random failures, must either raise a non-`nil`
  error value or return a documented `value, err` pair. Each API must document
  which style it uses.

## Scope

- Update `docs/SPEC.md` with the accepted dynamic semantics.
- Update `docs/STRICT_SEMANTICS.md` only after active tests exist.
- Update checker rules for source-validity cases visible before execution.
- Update eval interpreter behavior for runtime cases.
- Update generated C/runtime behavior where the compiled path differs from the
  interpreter.
- Update self-host Tya sources before enabling stricter rules that would reject
  current self-host code.
- Add focused unit tests and testscript coverage listed below.

## Out of Scope

- Static typing syntax, generics, typed nil, method overloads, or typed function
  signatures.
- Changing string `+` formatting behavior; that is covered by the separate
  strict string plus spec.
- Removing documented dictionary receiver methods.
- Making `print` non-special or changing builtin availability.
- Reworking stdlib APIs beyond documenting and testing their failure style.

## Acceptance Criteria

- `user.name` where `user` is known to be a dictionary is rejected with a clear
  diagnostic such as `cannot use . access on dictionary; use index access`.
- `user["name"]`, `user.keys()`, `user.has?("name")`, and
  `user.delete("age")` remain valid.
- `false and boom()` does not evaluate `boom`.
- `true or boom()` does not evaluate `boom`.
- `"value" and "right"` prints or returns `true`, not `"right"`.
- `nil or "fallback"` prints or returns `true`, not `"fallback"`.
- `x = 1; x = nil; x = "one"` is rejected as number-to-string reassignment.
- `x = nil; x = "one"` remains valid.
- A block-local variable created inside an `if` body is unavailable after the
  `if`.
- Assignment to a pre-existing outer variable inside an `if` body updates the
  outer variable.
- `for` item/index bindings are unavailable after the loop.
- `catch` bindings are unavailable after the catch body.
- `match case` pattern bindings are unavailable after the case body.
- Function bindings can be reassigned without a special prohibition.
- Existing self-host tests still pass after any required self-host source
  migration.

## Tests To Add

Checker tests:

- `TestCheckRejectsDictionaryKeyMemberAccess`
  - `user = { name: "komagata" }; print(user.name)` rejects.
- `TestCheckAllowsDictionaryIndexAndMethodAccess`
  - `user["name"]`, `user.keys()`, `user.has?("name")`, and `user.delete("age")`
    pass.
- `TestCheckRejectsKindChangingReassignmentThroughNil`
  - `value = 1; value = nil; value = "one"` rejects.
- `TestCheckAllowsNilToConcreteWhenNoPriorConcreteKind`
  - `value = nil; value = "one"` passes.
- `TestCheckKeepsBlockLocalBindingsLocal`
  - `if true; local = 1; print(local)` rejects after the block.
- `TestCheckAllowsBlockAssignmentToOuterBinding`
  - `count = 1; if true; count = 2; print(count)` passes.
- `TestCheckKeepsForBindingsLocal`
  - loop item and index are rejected after the loop.
- `TestCheckKeepsCatchBindingLocal`
  - catch binding is rejected after the catch body.
- `TestCheckKeepsMatchBindingsLocal`
  - pattern binding is rejected after the case body.
- `TestCheckAllowsFunctionReassignment`
  - assigning a function name to another function or value follows ordinary
    reassignment rules.

Eval tests:

- `TestRunLogicShortCircuitsAndReturnsBool`
  - covers `false and boom()`, `true or boom()`, truthy `and`, and falsey `or`.
- `TestRunRejectsDictionaryKeyMemberAccess`
  - runtime-known dictionary `user.name` errors when checker cannot prove it.
- `TestRunRejectsKindChangingReassignmentThroughNil`
  - runtime path mirrors checker behavior.
- `TestRunBlockScopeAndOuterAssignment`
  - outer updates work and block locals do not leak.

Testscript coverage:

- Extend `tests/testdata/v65_strict/strict_semantics.txtar` with the same
  valid and invalid snippets above.
- Include a self-host migration guard by running:

```sh
go test ./tests -run 'TestSelfhostV01Scripts|TestSelfhostV02Scripts' -count=1 -timeout=20m
```

## Verification

```sh
gofmt -w internal/checker/checker.go internal/checker/checker_test.go internal/eval/eval.go internal/eval/eval_test.go internal/codegen/c.go
go test ./internal/checker ./internal/eval ./internal/codegen -count=1
go test ./tests -run TestV65Scripts -count=1
go test ./tests -run 'TestExamplesGolden|TestV02Scripts|TestStdlibBinaryScript|TestSelfhostV01Scripts|TestSelfhostV02Scripts' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
