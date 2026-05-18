# Feature: v1 Language Syntax Boundaries

## Goal

Freeze the high-impact syntax and language-surface boundaries that Tya v1.0.0
will intentionally not expand, so the compiler, formatter, diagnostics, and
documentation can reject those forms consistently.

## Context

Tya v1.0.0 is a dynamically typed, indentation-based, compile-to-C language
with canonical syntax. `feature-specs/v1-language-semantics-freeze.md` records
accepted runtime and semantic decisions. This spec records the accepted syntax
boundaries for features that are common in other languages but intentionally
excluded from the v1.0.0 surface.

The direction is to keep one clear spelling for each capability, prefer
explicit method calls and class-style APIs over special syntax, and avoid
partially introducing static-type or metaprogramming expectations.

## Behavior

- Slice syntax is not part of v1.0.0.
  - `items[1:3]`, `items[:3]`, `items[1:]`, and step forms are invalid.
  - Use explicit methods such as `items.slice(1, 3)`, `text.slice(1, 3)`, and
    `bytes.slice(1, 3)`.
- Named arguments and keyword arguments are not part of v1.0.0.
  - `request(url, timeout: 10)` is invalid.
  - Configuration values use dictionary options such as
    `request(url, { timeout: 10 })`.
- Variadic parameter and splat call syntax is not part of v1.0.0.
  - `fn = *args -> ...`, `fn(a, *rest)`, and `fn(*items)` are invalid.
  - Variable-size inputs are passed as explicit arrays.
- Destructuring assignment is limited to multi-return assignment.
  - `a, b = pair()` is valid.
  - `[a, b] = items`, `{ name } = user`, and nested destructuring assignments
    are invalid.
- `match` patterns stay narrow for v1.0.0.
  - Literal patterns, `nil`, `true`, `false`, `case _`, and array/dictionary
    structure patterns are valid where already specified.
  - Guard patterns such as `case value if condition` are invalid.
  - Binding patterns that introduce new names are invalid.
- Operator overloading is not part of v1.0.0.
  - Operators keep primitive semantics only.
  - User-domain behavior uses explicit methods.
- Function and method overloading are not part of v1.0.0.
  - Reusing a binding or method name to define another overload is invalid.
  - Use distinct names, default parameters, or dictionary options.
- Generic type-parameter syntax is not part of v1.0.0.
  - `Array<Int>`, `Box<T>`, `fn<T>`, and type-argument calls are invalid.
  - Static typing discussion notes remain non-authoritative.
- Public `module` declarations are not part of v1.0.0.
  - Source organization uses script files, class files, directory packages, and
    import aliases.
- Dedicated `enum`, `record`, or `struct` syntax is not part of v1.0.0.
  - Domain values use classes, dictionaries, constants, or standard-library
    value classes.
- Macro and compile-time metaprogramming syntax is not part of v1.0.0.
  - `embed` remains a dedicated declaration, not a general macro system.
  - Compiler, formatter, checker, and doc tooling remain deterministic.
- Async function coloring is not part of v1.0.0.
  - `async fn`, async-only function kinds, and async method modifiers are
    invalid.
  - Concurrency remains expressed through `spawn`, `await`, `scope`, `select`,
    tasks, and channels.
- Visibility remains two-level.
  - Public and `private` are the only visibility categories.
  - `protected`, package-private, friend, and similar visibility modifiers are
    invalid.
- Top-level public APIs stay split by file kind.
  - Lowercase script files may expose imported top-level function and value
    bindings.
  - Directory package public API is limited to PascalCase public classes and
    interfaces from class files.
  - Directory packages do not expose lowercase top-level function modules as
    public package API.

## Scope

- `docs/SPEC.md`
- `docs/STRICT_SEMANTICS.md` only where a rejected syntax has a validity rule
- parser and checker diagnostics for rejected syntax
- formatter tests that ensure rejected syntax is not canonicalized
- CLI testscript fixtures for `tya check` and representative `tya run` failures
- self-host sources only where needed to preserve fixed-point gates

## Out of Scope

- Implementing any excluded syntax.
- Removing already accepted v1.0.0 features such as class files, interfaces,
  `spawn`, `await`, `scope`, `select`, `embed`, native package metadata, or
  WebAssembly targets.
- Adding static types.
- Changing runtime semantics already frozen by
  `feature-specs/v1-language-semantics-freeze.md`.
- Changing standard-library method names beyond documenting the method-based
  alternatives for excluded syntax.

## Acceptance Criteria

- `docs/SPEC.md` explicitly lists these v1.0.0 exclusions in the relevant
  sections or in a compact language-boundaries section.
- Every excluded syntax form has a parser or checker diagnostic that fails
  clearly before code generation.
- `tya format` never rewrites excluded syntax into a different accepted form.
- `tya check`, `tya run`, and `tya build` agree that the excluded forms are
  invalid.
- Existing accepted alternatives remain valid:
  - `.slice(...)` methods for slicing behavior;
  - dictionary options for named configuration;
  - arrays for variable-size arguments;
  - multi-return assignment;
  - explicit methods for domain behavior;
  - class/interface package APIs;
  - `spawn` / `await` / `scope` / `select` concurrency.
- Existing self-host fixed-point gates remain valid.

## Tests To Add

Parser/checker tests:

- `TestParseRejectsSliceSyntax`
  - `items[1:3]`
  - Expected: parser diagnostic that slice syntax is not part of Tya.

- `TestParseRejectsNamedArguments`
  - `request(url, timeout: 10)`
  - Expected: parser diagnostic for named arguments.

- `TestParseRejectsVariadicAndSplatSyntax`
  - `fn = *args -> args`
  - `fn(*items)`
  - Expected: parser diagnostics for variadic/splat syntax.

- `TestParseRejectsDestructuringAssignment`
  - `[a, b] = items`
  - `{ name } = user`
  - Expected: parser diagnostics while `a, b = pair()` remains valid.

- `TestParseRejectsMatchGuardsAndBindingPatterns`
  - `case value if ready`
  - `case [head, tail]`
  - Expected: diagnostics for guard or binding patterns.

- `TestCheckRejectsOperatorOverloadDeclarations`
  - A class attempts to define `+`, `<`, or another operator as a method.
  - Expected: checker diagnostic.

- `TestCheckRejectsFunctionAndMethodOverloads`
  - Two functions or methods with the same name but different arity.
  - Expected: checker diagnostic for duplicate definition.

- `TestParseRejectsGenericSyntax`
  - `items: Array<Int> = []`
  - `Box<T>`
  - Expected: parser diagnostics for type-parameter syntax.

- `TestParseRejectsModuleEnumRecordStructMacroAsyncSyntax`
  - Representative snippets using `module`, `enum`, `record`, `struct`,
    macro syntax, and `async`.
  - Expected: parser diagnostics.

- `TestCheckRejectsUnsupportedVisibilityModifiers`
  - `protected name = 1`
  - `friend class Helper`
  - Expected: parser or checker diagnostics.

Testscript coverage:

- `v1_syntax_boundaries.txtar`
  - Covers CLI-level rejection for slice syntax, named arguments, splat,
    destructuring assignment, match guards, generics, public `module`, enum,
    record, struct, macro, async function coloring, and unsupported visibility.

- `v1_language_alternatives.txtar`
  - Covers accepted alternatives:
    `.slice(...)`, dictionary options, explicit arrays, multi-return assignment,
    explicit domain methods, class/interface package APIs, and structured
    concurrency constructs.

Formatter tests:

- `TestFormatRejectsExcludedV1Syntax`
  - Ensures excluded syntax returns an error and leaves input unchanged under
    `tya format -w`.

## Verification

```sh
go test ./internal/parser -count=1
go test ./internal/checker -count=1
go test ./internal/format -count=1
go test ./tests -run 'TestV.*Scripts|TestSelfhostV01Scripts|TestSelfhostV02Scripts' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
