# Feature: Control, Display, And Platform Semantics

## Goal

Make receiver evaluation, assignment target evaluation, unary logic, `finally`,
`try/catch` values, display strings, cyclic data, identity equality, API
canonicalization, platform differences, time/random tests, paths, and text
encoding deterministic and testable.

## Context

This spec records accepted behavior for the current dynamically typed Tya
surface. It is not a static typing plan. It is intended for a future
implementation pass and therefore includes the tests that should be added with
that implementation.

The behavior here complements:

- `feature-specs/unambiguous-dynamic-semantics.md`
- `feature-specs/dynamic-edge-semantics.md`
- `feature-specs/numeric-call-and-canonical-semantics.md`

Self-host compatibility is required. Any implementation must preserve the v01
and v02 self-host fixed point.

Tya's current public model uses imported packages/classes and aliases rather
than a user-facing `module` concept. Any internal module representation must
not leak into user-facing terminology or display.

## Behavior

- A method-call receiver is evaluated exactly once.
- `get_user().name()` must call `get_user()` once.
- Assignment right-hand sides are evaluated first, left to right.
- After all right-hand sides are evaluated, assignment targets are evaluated
  left to right and assigned left to right.
- For `items[i()], items[j()] = a(), b()`, the observable order is `a`, `b`,
  `i`, `j`.
- `and` and `or` return booleans.
- `and` and `or` short-circuit. If the right-hand side is skipped, its side
  effects do not happen.
- `not` always returns a boolean.
- `not` applies Tya truthiness and returns the opposite boolean.
- `not nil` is `true`.
- `not "x"` is `false`.
- `finally` is part of the language.
- `try/catch/finally` is valid.
- `try/finally` without `catch` is valid.
- A bare `try` statement with neither `catch` nor `finally` is invalid.
- `finally` always runs when control leaves the `try` or `catch` body.
- `finally` runs before outward `return`, `raise`, `break`, or `continue`
  completes.
- The value of a `finally` body is not the value of the `try` statement.
- If `try` completes normally, the `try` statement value is the final value of
  the `try` body.
- If `try` raises and `catch` handles it, the `try` statement value is the
  final value of the `catch` body.
- If `try` raises and there is no `catch`, the original raised value is
  re-raised after `finally`.
- If `finally` itself performs `return`, `raise`, `break`, or `continue`, that
  control flow replaces any prior pending control flow.
- `try` without `finally` but with `catch` keeps the existing `try/catch`
  behavior.
- `try/catch` succeeds with the `try` block's final value when no raise occurs.
- `try/catch` evaluates to the `catch` block's final value when a raise is
  caught.
- Empty `try`, `catch`, or `finally` blocks are invalid under the empty-block
  rule. Use explicit `nil` for an intentional no-op.
- `catch` without a matching `try` is invalid.
- Function literals do not have declaration names.
- A function assigned to a binding may use the binding name as a debugging and
  diagnostic name.
- Reassigning a function to another binding may update the debugging name to
  the new binding.
- Debugging names do not affect equality, identity, or call behavior.
- Printing functions, classes, objects, packages/import aliases, resources,
  tasks, and channels uses stable short display strings.
- Function display is `<function name>` when a debug name exists and
  `<function>` otherwise.
- Class display is `<class User>`.
- Object display is `<User>`, or another stable class-based object display
  chosen by the implementation and documented in `docs/SPEC.md`.
- Imported package or alias display must use package terminology, not
  user-facing `module` terminology, for example `<package http>`.
- Resource, task, channel, and other runtime handle displays use stable kind
  names and must not include nondeterministic memory addresses.
- Array display preserves array order.
- Dictionary display order is unspecified because dictionary iteration order is
  unspecified.
- Tests and snapshots must not depend on dictionary display order.
- `print` and `to_s` detect cyclic arrays and dictionaries.
- Cyclic display must terminate and show a stable cycle marker such as
  `<cycle>`.
- Deep equality for arrays and dictionaries containing cycles is a runtime
  error.
- Deep equality must not recurse forever.
- User-defined objects compare by identity with `==` and `!=`.
- `==` does not call user-defined `equal?`.
- `equal?` may exist as an explicit method, but using it is a direct method
  call, not operator dispatch.
- Unknown member or method errors include the receiver kind/class and the
  member name.
- Example diagnostics: `unknown method len on number`,
  `unknown member name on User`.
- Standard-library API aliases are not added.
- Where compatibility aliases already exist, `docs/SPEC.md` must choose one
  canonical spelling.
- Non-canonical compatibility aliases are documented as legacy compatibility
  only and may be removed by a later spec.
- Deprecated APIs are not kept indefinitely behind warnings.
- A removed API is documented as removed or legacy compatibility only.
- Long-lived warning-only deprecations are avoided.
- Environment-dependent standard-library behavior must fail explicitly when
  unsupported.
- Unsupported OS features, missing platform capabilities, or unavailable
  native backends return or raise documented errors.
- Platform differences must not silently return `nil`.
- Time and random APIs are tested deterministically only where deterministic
  seams exist, such as seedable random generators or injectable clocks.
- Current time and secure random APIs are tested by properties rather than
  exact values.
- Language-level and package import paths use `/` as the canonical separator.
- Standard-library `Path` APIs absorb host OS path differences, including
  Windows path conventions.
- Tya `String` is UTF-8 text.
- Invalid UTF-8 byte sequences are represented as `Bytes`, not `String`.
- Text file APIs that return `String` must error on invalid UTF-8.
- Binary file APIs return `Bytes` and do not validate UTF-8.

## Scope

- Update `docs/SPEC.md` with these accepted semantics.
- Update `docs/STRICT_SEMANTICS.md` once active tests exist.
- Update parser support for `finally` and invalid bare `try`.
- Update checker rules for `try/catch/finally`, `catch` placement, unknown
  member diagnostics where statically known, deprecated/removed API diagnostics,
  and platform/path/text API contracts where applicable.
- Update eval interpreter behavior for evaluation order, `finally`, display,
  cycles, equality, unknown member errors, and UTF-8 text APIs.
- Update C codegen/runtime behavior where the compiled path differs from the
  interpreter.
- Update formatter behavior for `finally` blocks.
- Update standard-library docs/comments for canonical aliases, platform
  failures, time/random testing contracts, path separators, and UTF-8 behavior.
- Add focused unit tests and testscript coverage listed below.

## Out of Scope

- Static typing, generics, overloads, or typed catch clauses.
- Stable dictionary iteration or dictionary display order.
- Operator dispatch to user-defined `equal?`.
- Warning-based deprecation lifecycle.
- Introducing a public `module` concept.
- Making current time or secure random deterministic.

## Acceptance Criteria

- Method receivers are evaluated once.
- Assignment side effects prove right-hand sides run before assignment targets,
  and targets run left to right.
- `and`, `or`, and `not` always return booleans with the specified short-circuit
  behavior.
- `try/catch/finally` and `try/finally` parse, format, run, and compile.
- Bare `try` without `catch` or `finally` is rejected.
- `finally` runs for normal completion, `return`, `raise`, `break`, and
  `continue`.
- `finally` values do not replace ordinary `try` or `catch` values.
- Control flow inside `finally` replaces pending control flow.
- Function debug names affect display/diagnostics only.
- Function, class, object, package/import alias, resource, task, and channel
  displays are stable and do not expose memory addresses.
- Package/import alias display uses package terminology rather than `module`.
- Array display is ordered.
- Dictionary display tests avoid order assumptions.
- Cyclic `print` and `to_s` terminate with a stable cycle marker.
- Deep equality on cyclic arrays/dictionaries raises a runtime error.
- Object equality is identity-based and does not call `equal?`.
- Unknown member/method errors include receiver kind/class and member name.
- Canonical standard-library API names are documented.
- Existing compatibility aliases, if kept, are documented as legacy
  compatibility only.
- Unsupported platform features fail with explicit documented errors.
- Time/random tests do not assert exact current time or secure random values.
- `/` is the canonical import/path separator in language-level paths.
- `Path` standard-library APIs handle host separators.
- Text APIs reject invalid UTF-8; binary APIs return bytes.

## Tests To Add

Parser/checker tests:

- `TestParseTryCatchFinally`
  - Parses `try/catch/finally`.
- `TestParseTryFinally`
  - Parses `try/finally`.
- `TestParseRejectsBareTry`
  - Rejects `try` with no `catch` and no `finally`.
- `TestParseRejectsCatchWithoutTry`
  - Rejects standalone `catch`.
- `TestFormatTryFinally`
  - Formatter emits canonical `try`, `catch`, and `finally` indentation.
- `TestCheckUnknownMemberDiagnosticIncludesReceiver`
  - Statically known unknown members include receiver kind/class and member
    name when checker can know them.
- `TestCheckRemovedOrLegacyApiDiagnostics`
  - Removed APIs are errors and legacy aliases are not presented as canonical.

Eval tests:

- `TestRunMethodReceiverEvaluatedOnce`
  - A receiver-producing function increments a counter once for one method
    call.
- `TestRunAssignmentTargetEvaluationOrder`
  - Records side effects for `items[i()], items[j()] = a(), b()` and expects
    `a`, `b`, `i`, `j`.
- `TestRunLogicalOperatorsReturnBoolAndShortCircuit`
  - Covers `and`, `or`, skipped side effects, and bool return values.
- `TestRunNotReturnsBool`
  - Covers `not nil`, `not false`, `not true`, `not 0`, and `not "x"`.
- `TestRunTryCatchFinallyValue`
  - Success returns try value, handled raise returns catch value, finally value
    is ignored.
- `TestRunTryFinallyReraises`
  - `try/finally` without `catch` runs `finally` and re-raises the original
    value.
- `TestRunFinallyRunsBeforeReturn`
  - Function return still runs cleanup.
- `TestRunFinallyRunsBeforeBreakAndContinue`
  - Loop control flow still runs cleanup.
- `TestRunFinallyControlFlowOverridesPendingFlow`
  - `raise` or `return` inside `finally` replaces pending return/raise.
- `TestRunFunctionDebugNameDisplay`
  - Assigned function display uses the binding debug name.
- `TestRunStableRuntimeDisplays`
  - Function/class/object/package/resource/task/channel displays are stable and
    do not contain memory addresses.
- `TestRunArrayAndDictionaryDisplay`
  - Array display order is stable; dictionary display test checks contents
    without assuming order.
- `TestRunCyclicDisplayTerminates`
  - Cyclic arrays and dictionaries print with `<cycle>` or the documented cycle
    marker.
- `TestRunCyclicDeepEqualityErrors`
  - Cyclic array/dict equality raises a runtime error.
- `TestRunObjectEqualityIsIdentity`
  - Distinct objects with equal fields are not `==`; identical object reference
    is `==`.
- `TestRunEqualityDoesNotCallEqualMethod`
  - `==` does not invoke user-defined `equal?`.
- `TestRunUnknownMemberErrorIncludesReceiver`
  - Runtime unknown member/method errors include kind/class and member name.
- `TestRunInvalidUtf8TextReadErrors`
  - Text file API rejects invalid UTF-8.
- `TestRunBinaryReadReturnsBytesForInvalidUtf8`
  - Binary file API returns bytes for the same data.

Standard-library tests:

- `TestStdlibPlatformUnsupportedErrors`
  - Platform-unsupported APIs return or raise explicit errors, not `nil`.
- `TestStdlibRandomSeedDeterministic`
  - Seeded random generator is deterministic.
- `TestStdlibSecureRandomPropertiesOnly`
  - Secure random tests assert length/type/properties, not exact bytes.
- `TestStdlibCurrentTimePropertiesOnly`
  - Current time tests assert shape/range, not exact instant.
- `TestStdlibPathUsesSlashCanonicalForms`
  - Language/package paths use `/`; `Path` handles host separators.
- `TestStdlibCanonicalApiNames`
  - Canonical names are documented and aliases are marked legacy compatibility
    only where they exist.

Testscript coverage:

- Add or extend a strict semantics script with valid and invalid snippets for
  `try/catch/finally`, receiver evaluation, assignment target evaluation order,
  logical operator bool returns, `not`, stable displays, cyclic display,
  cyclic equality errors, and UTF-8 text/binary behavior.
- Include compiled C execution coverage for `finally`, display, cycles, and
  UTF-8 behavior where the compiled path differs from eval.
- Keep self-host gates in verification because parser, formatter, runtime
  display, and control-flow changes can affect the compiler.

## Verification

```sh
gofmt -w internal/parser internal/checker internal/eval internal/codegen internal/format runtime
go test ./internal/parser ./internal/checker ./internal/eval ./internal/codegen ./internal/format -count=1
go test ./tests -run TestV65Scripts -count=1
go test ./tests -run 'TestExamplesGolden|TestV02Scripts|TestStdlibBinaryScript|TestSelfhostV01Scripts|TestSelfhostV02Scripts' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
