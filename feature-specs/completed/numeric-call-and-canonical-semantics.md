# Feature: Numeric, Call, And Canonical Semantics

## Goal

Make numeric behavior, function parameter rules, trailing comma syntax, empty
blocks, comments, import duplication, and output streams deterministic and
testable for the current dynamically typed Tya language.

## Context

This spec records accepted behavior for the current dynamic language surface.
It is not a static typing plan. It is intended for a future implementation pass
and therefore includes the tests that should be added with that implementation.

The behavior here complements:

- `feature-specs/unambiguous-dynamic-semantics.md`
- `feature-specs/dynamic-edge-semantics.md`

Self-host compatibility is required. Any implementation must preserve the v01
and v02 self-host fixed point.

## Behavior

- User-facing numeric semantics expose one `Number` kind.
- The implementation may internally distinguish integer and floating-point
  storage, but language behavior treats both as numbers.
- `1 == 1.0` is `true`.
- Number display is natural: whole numbers display without a decimal suffix,
  and fractional numbers display with their fractional part.
- `/` always performs number division.
- `5 / 2` evaluates to `2.5`.
- Integer division is not expressed with `/`.
- Integer division must use an explicit API such as existing `div()` or a
  future named method.
- `%` is integer-only.
- `%` with a floating-point operand is an error.
- Negative indexes are invalid for arrays, strings, and bytes.
- Negative indexes do not mean "from the end" and do not return `nil`.
- Use explicit APIs such as `last()` when reading from the end.
- Slice syntax such as `items[1:3]` is not part of the language.
- Slicing uses explicit methods such as `items.slice(start, end)`.
- String equality uses `==` and `!=`.
- Ordering operators `<`, `<=`, `>`, and `>=` are invalid for strings.
- String ordering, collation, or locale-aware comparison can be added later
  only as explicit methods or standard-library APIs.
- `NaN` and `Infinity` are not language literals.
- `nan`, `NaN`, `infinity`, and `Infinity` are not reserved numeric literal
  spellings.
- If native or standard-library numeric code can produce NaN or infinity, that
  API must document how the value displays, compares, and serializes before it
  is exposed as public Tya behavior.
- Until explicit numeric-special-value APIs exist, ordinary Tya code should not
  rely on NaN or infinity values.
- Function parameters may have default values.
- Default values are written in the function definition, not at the call site.
- Parameters without defaults must precede parameters with defaults.
- A required parameter after an optional parameter is invalid.
- Calls may omit only a trailing run of defaulted parameters.
- Call arity remains deterministic: too few required arguments or too many
  arguments are errors.
- Default parameter expressions are evaluated at call time when the argument is
  omitted.
- Default parameter expressions are evaluated left to right in parameter order.
- Default parameter expressions may reference earlier parameters.
- Default parameter expressions may not reference later parameters.
- Mutable default values are freshly evaluated per call and are not shared
  between calls.
- Variadic arguments are prohibited.
- There is no `...args`, `*args`, `vararg`, or equivalent syntax.
- Pass an array explicitly when a function needs a variable number of values.
- `print` remains a normal one-argument builtin/special surface according to
  the active SPEC; it does not introduce user-defined variadic functions.
- Trailing commas are always prohibited.
- One-line and multi-line arrays, dictionaries, argument lists, parameter
  lists, and call lists use the same no-trailing-comma rule.
- The formatter removes or rejects trailing commas according to the formatter
  architecture chosen during implementation.
- Empty blocks are prohibited.
- A block that intentionally does nothing must contain an explicit expression,
  typically `nil`.
- Comments are allowed only as file header comments, statement-leading
  comments, and line-end comments.
- Comments inside expression structure are invalid and must not be preserved as
  alternate formatting choices.
- Top-level function references may point forward to functions defined later in
  the same file.
- Ordinary variable references before assignment are invalid.
- Duplicate imports are invalid.
- Importing the same path twice is invalid even if aliases differ.
- `print` writes only to stdout.
- stderr output is available only through explicit standard-library APIs.

## Scope

- Update `docs/SPEC.md` with these accepted semantics.
- Update `docs/STRICT_SEMANTICS.md` once active tests exist.
- Update parser/checker behavior for default parameters, variadic rejection,
  trailing comma rejection, empty block rejection, comment placement if needed,
  duplicate imports, negative indexes where statically known, string ordering,
  and `%` operands where statically known.
- Update eval interpreter behavior for runtime numeric and call semantics.
- Update C codegen/runtime behavior where the compiled path differs from the
  interpreter.
- Update formatter behavior for trailing commas and comments.
- Update examples only where they rely on now-invalid syntax.
- Add focused unit tests and testscript coverage listed below.

## Out of Scope

- Static types, generics, overloads, or typed function signatures.
- Variadic user-defined functions.
- Slice syntax.
- String ordering operators.
- NaN or infinity literals.
- Stable dictionary iteration order.
- Changing string `+` behavior.

## Acceptance Criteria

- `1 == 1.0` evaluates to `true`.
- `5 / 2` evaluates to `2.5`.
- `%` works for integers and errors for floating-point operands.
- `items[-1]`, `text[-1]`, and `bytes[-1]` are errors.
- `items.slice(1, 3)` remains the canonical slicing form.
- `items[1:3]` is rejected by the parser.
- `"a" == "a"` works.
- `"a" < "b"` is an error.
- `NaN` and `Infinity` are not accepted as numeric literals.
- A function with default parameters can omit trailing defaulted arguments.
- Required parameters after defaulted parameters are rejected.
- Too few required arguments and too many arguments are errors.
- Default expressions are evaluated at call time and are not shared between
  calls.
- Default expressions can reference earlier parameters but not later
  parameters.
- Variadic syntax is rejected.
- Passing an explicit array remains the supported variable-length argument
  pattern.
- Trailing commas are rejected or canonicalized away everywhere.
- Empty blocks are rejected.
- `nil` is accepted as an explicit no-op block body.
- Comments in file-header, statement-leading, and line-end positions are
  accepted.
- Comments inside expressions are rejected or reformatted away according to the
  formatter/checker design, with no second canonical representation.
- Top-level function forward references work.
- Ordinary variable forward references fail.
- Duplicate imports fail, including alias-different duplicate imports.
- `print` writes to stdout and not stderr.
- stderr output requires the documented standard-library API.

## Tests To Add

Parser/checker tests:

- `TestCheckNumberKindAllowsIntFloatEquality`
  - `print(1 == 1.0)` is valid.
- `TestCheckRejectsFloatModulo`
  - `print(5.5 % 2)` and `print(5 % 2.0)` fail.
- `TestCheckRejectsNegativeLiteralIndexes`
  - `items[-1]`, `"abc"[-1]`, and `b"abc"[-1]` fail when statically visible.
- `TestParseRejectsSliceSyntax`
  - `items[1:3]` fails.
- `TestCheckRejectsStringOrdering`
  - `"a" < "b"` fails.
- `TestParseRejectsNaNAndInfinityLiterals`
  - `NaN`, `Infinity`, `nan`, and `infinity` are not numeric literals.
- `TestParseFunctionDefaultParameters`
  - `greet = name, suffix = "!" -> ...` parses.
- `TestCheckRejectsRequiredParameterAfterDefault`
  - `f = a = 1, b -> b` fails.
- `TestCheckDefaultParameterReferences`
  - defaults may reference earlier params and may not reference later params.
- `TestCheckRejectsVariadicParameters`
  - `f = values... -> values` and similar spellings fail.
- `TestParseRejectsTrailingCommas`
  - arrays, dictionaries, calls, and parameter lists with trailing commas fail.
- `TestParseRejectsEmptyBlocks`
  - `if true` with no body and equivalent empty blocks fail.
- `TestParseAllowsExplicitNilNoOpBlock`
  - `if true; nil` passes.
- `TestParseCommentPlacement`
  - file header, statement-leading, and line-end comments pass; expression
    interior comments fail or are canonicalized by formatter tests.
- `TestCheckAllowsTopLevelFunctionForwardReference`
  - `print(greet("Tya")); greet = name -> name` passes when `greet` is a
    top-level function binding.
- `TestCheckRejectsVariableForwardReference`
  - `print(name); name = "Tya"` fails.
- `TestCheckRejectsDuplicateImports`
  - repeated import paths fail, including alias-different repeats.

Eval tests:

- `TestRunNumberDivision`
  - `5 / 2` prints `2.5`.
- `TestRunModuloRequiresIntegers`
  - runtime float modulo errors when not caught earlier.
- `TestRunNegativeIndexesError`
  - runtime negative index errors for arrays, strings, and bytes.
- `TestRunStringOrderingErrors`
  - runtime string ordering errors when not caught earlier.
- `TestRunDefaultParameters`
  - omitted trailing defaulted parameters use defaults.
- `TestRunDefaultParametersEvaluateAtCallTime`
  - mutable defaults are fresh per call.
- `TestRunDefaultParametersEvaluateLeftToRight`
  - side effects prove call-time left-to-right evaluation.
- `TestRunDefaultParameterCanReferenceEarlierParameter`
  - `label = name, text = name -> text` works.
- `TestRunDefaultParameterCannotReferenceLaterParameter`
  - later-param reference fails.
- `TestRunVariadicFunctionSyntaxUnavailable`
  - parser/checker rejection is covered through eval fixture if needed.
- `TestRunTopLevelFunctionForwardReference`
  - calling a later top-level function works.
- `TestRunVariableForwardReferenceFails`
  - ordinary later variable lookup fails.
- `TestRunPrintWritesStdoutOnly`
  - stdout contains print output and stderr remains empty.

Formatter tests:

- `TestFormatRemovesOrRejectsTrailingCommas`
  - formatter output has no trailing commas.
- `TestFormatCommentPlacementCanonical`
  - only accepted comment positions survive as canonical output.

Testscript coverage:

- Add or extend a strict semantics script with valid and invalid snippets for
  numeric behavior, defaults, no variadics, no trailing commas, no empty blocks,
  duplicate imports, and stdout/stderr behavior.
- Include compiled C execution coverage for number division, default
  parameters, stdout behavior, and top-level function forward references.
- Keep self-host gates in the verification set because parser and call
  semantics affect self-host.

## Verification

```sh
gofmt -w internal/parser internal/checker internal/eval internal/codegen internal/format
go test ./internal/parser ./internal/checker ./internal/eval ./internal/codegen ./internal/format -count=1
go test ./tests -run TestV65Scripts -count=1
go test ./tests -run 'TestExamplesGolden|TestV02Scripts|TestStdlibBinaryScript|TestSelfhostV01Scripts|TestSelfhostV02Scripts' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
