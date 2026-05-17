# Feature: Dynamic Edge Semantics

## Goal

Make the remaining dynamic-language edge cases deterministic and testable so
Tya programs have one obvious behavior for variable lookup, evaluation order,
dictionary literals, string indexing, loop expression results, equality, and
standard-library failures.

## Context

This spec records accepted behavior for the current dynamically typed Tya
surface. It is not a static typing plan. It is intended for a future
implementation pass and therefore includes the tests that should be added with
that implementation.

The behavior here builds on `feature-specs/unambiguous-dynamic-semantics.md`.
If both specs touch the same area, implement them together or make sure their
tests remain consistent.

Self-host compatibility is required. Any implementation must preserve the v01
and v02 self-host fixed point.

## Behavior

- Undefined variables are errors.
- If `tya check` can prove a name is undefined, it reports a checker
  diagnostic.
- If a name can only be proven undefined at runtime, evaluation raises a
  runtime error.
- Undefined variables never silently evaluate to `nil`.
- Top-level functions are ordinary bindings.
- Reassigning a function name to another function is allowed.
- Reassigning a function name to a different concrete kind follows the ordinary
  concrete-kind reassignment rule.
- Dictionary literals allow both identifier keys and string-literal keys.
- Identifier keys are stored as strings.
- String-literal keys allow JSON-style keys such as `"user-name"`,
  `"Content-Type"`, `"$schema"`, `"1"`, and `""`.
- Dictionary literal keys are always strings. Numeric, boolean, `nil`, array,
  dictionary, function, or expression keys are invalid in dictionary literals.
- Duplicate dictionary literal keys are invalid after normalizing identifier
  keys and string-literal keys to strings. For example `{ name: 1, "name": 2 }`
  is invalid.
- Dictionary index access also uses string keys.
- Destructive collection methods return `nil` unless their purpose is to return
  a removed or retrieved value.
- `Array.push`, `Dict.set`, and `Dict.delete` return `nil`.
- `Array.pop` returns the removed value, or `nil` when the array is empty.
- A `for` expression evaluates to the last body value.
- If a `for` body never executes, the loop value is `nil`.
- If a function body ends with an empty `for`, the implicit function return is
  `nil`.
- An `if` expression evaluates to the selected branch value.
- If no `if` branch executes and there is no `else`, the value is `nil`.
- A `match` expression evaluates to the selected case body value.
- If no `match` case matches, the value is `nil`.
- A loop exited with `break` evaluates to the last completed body value before
  the break.
- If there is no completed body value before `break`, the loop value is `nil`.
- `continue` discards the current iteration's partial value and proceeds to the
  next iteration.
- Function call arguments are evaluated left to right.
- Assignment right-hand sides are evaluated left to right before any assignment
  target is updated.
- Multiple assignment targets are then assigned left to right.
- `a, b = b, a` swaps values.
- Dictionary iteration order is intentionally unspecified.
- Tests and user code must not rely on dictionary iteration order. Use an
  explicit sorted key list when order matters.
- String indexing is allowed.
- String indexes must be integers.
- String indexing is by Unicode rune.
- Out-of-range string indexes return `nil`.
- Byte indexing remains byte-based through bytes values.
- `==` and `!=` compare arrays and dictionaries deeply.
- Functions, classes, objects, resources, tasks, channels, and other
  heap-backed non-collection identity values compare by identity unless their
  documented primitive surface says otherwise.
- `catch` catches every non-`nil` value raised by `raise`.
- There is no typed catch or value-filtered catch syntax.
- Branching by raised value is written inside the `catch` body with `if` or
  `match`.
- `raise nil` is invalid.
- `raise` may raise any non-`nil` value.
- Standard-library APIs should raise or return error values consistently:
  programmer errors are runtime errors, external failures are raised errors,
  and absence-only lookups return `nil` or `false` where documented.
- Each standard-library API must document which failure style it uses.

## Scope

- Update `docs/SPEC.md` with these dynamic semantics.
- Update `docs/STRICT_SEMANTICS.md` once active tests exist.
- Update parser/checker support for string-literal dictionary keys and duplicate
  normalized keys.
- Update eval interpreter behavior where current behavior differs.
- Update C codegen/runtime behavior where the compiled path differs from the
  interpreter.
- Update examples only where they depend on behavior clarified here.
- Add focused unit tests and testscript coverage listed below.

## Out of Scope

- Static type syntax, generics, typed nil, overloads, or function type syntax.
- Changing string `+` behavior.
- Requiring a stable dictionary iteration order.
- Adding typed catch clauses.
- Replacing standard-library APIs wholesale; this spec only requires documented
  failure behavior.

## Acceptance Criteria

- Undefined names are rejected or fail at runtime; they never become `nil`.
- Function-to-function reassignment works.
- Function-to-number or function-to-string reassignment follows the ordinary
  kind-change rule.
- `{ "Content-Type": "text/plain", "$schema": "x", "": "empty" }` is valid.
- `{ name: "Tya", "name": "duplicate" }` is rejected as a duplicate key.
- `{ 1: "one" }` is rejected; `{ "1": "one" }` is valid.
- `dict["missing"]` returns `nil`.
- `items.push(value)` returns `nil`.
- `items.pop()` returns the removed value or `nil` for an empty array.
- `dict.set(key, value)` and `dict.delete(key)` return `nil`.
- Empty `for`, unmatched `if`, and unmatched `match` evaluate to `nil`.
- `break` and `continue` produce the loop values described above.
- Function arguments and assignment right-hand sides are evaluated left to
  right.
- `a, b = b, a` swaps.
- Dictionary-order tests avoid ordering assumptions.
- `"あい"[0]` returns `"あ"` and `"あい"[99]` returns `nil`.
- `b"abc"[0]` returns the first byte value.
- Function equality and resource equality do not perform deep comparison.
- `catch` catches any raised non-`nil` value.
- `raise nil` remains invalid.
- Standard-library failure modes are documented and tested for representative
  APIs.

## Tests To Add

Parser/checker tests:

- `TestCheckRejectsUndefinedVariable`
  - Known undefined names fail during `Check`.
- `TestCheckAllowsFunctionReassignmentToFunction`
  - `handler = -> 1; handler = -> 2; print(handler())` passes.
- `TestCheckRejectsFunctionReassignmentToDifferentKind`
  - `handler = -> 1; handler = 1` fails if concrete-kind checking is active.
- `TestParseDictionaryStringKeys`
  - String-literal dictionary keys parse in inline and block dictionary forms.
- `TestCheckRejectsDuplicateNormalizedDictionaryKeys`
  - `{ name: 1, "name": 2 }` fails.
- `TestCheckRejectsNonStringDictionaryLiteralKeys`
  - Numeric and expression keys fail in dictionary literals.

Eval tests:

- `TestRunUndefinedVariableIsRuntimeError`
  - Runtime-only undefined lookup fails and does not produce `nil`.
- `TestRunDictionaryStringLiteralKeys`
  - Reads JSON-style keys including `"Content-Type"`, `"$schema"`, `"1"`, and
    `""`.
- `TestRunCollectionMutationReturnValues`
  - Covers `push`, `pop`, `set`, and `delete` return values.
- `TestRunEmptyControlFlowValues`
  - Covers empty `for`, unmatched `if`, and unmatched `match` as `nil`.
- `TestRunBreakContinueLoopValues`
  - Covers break-before-body-value, break-after-body-value, and continue
    discarding the current iteration value.
- `TestRunCallArgumentEvaluationOrder`
  - Uses side effects to prove arguments evaluate left to right.
- `TestRunAssignmentEvaluationOrderAndSwap`
  - Proves all RHS expressions evaluate before LHS updates and `a, b = b, a`
    swaps.
- `TestRunStringIndexingUsesRunes`
  - Covers ASCII, multibyte rune, negative index, and out-of-range index.
- `TestRunBytesIndexingUsesBytes`
  - Confirms byte indexing remains byte-based.
- `TestRunIdentityEqualityForFunctionsAndResources`
  - Confirms non-collection heap-backed values are identity compared.
- `TestRunCatchCatchesAnyNonNilValue`
  - Covers string, error value, dictionary, and number raises.
- `TestRunRaiseNilFails`
  - Confirms `raise nil` is invalid.

Testscript coverage:

- Add a strict-semantics script or extend
  `tests/testdata/v65_strict/strict_semantics.txtar` with the same valid and
  invalid snippets.
- Add dictionary-string-key parser/checker diagnostics to the existing
  expression-level diagnostic fixtures if parser recovery changes.
- Add representative standard-library failure tests:
  - invalid argument kind is a runtime error;
  - missing lookup returns `nil` or `false`;
  - external failure raises a non-`nil` error.

## Verification

```sh
gofmt -w internal/parser internal/checker internal/eval internal/codegen
go test ./internal/parser ./internal/checker ./internal/eval ./internal/codegen -count=1
go test ./tests -run TestV65Scripts -count=1
go test ./tests -run 'TestExamplesGolden|TestV02Scripts|TestStdlibBinaryScript|TestSelfhostV01Scripts|TestSelfhostV02Scripts' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
