# Feature: v1 Language Semantics Freeze

## Goal

Lock the remaining high-impact v1.0.0 language-semantics decisions so future
implementation work can update `docs/SPEC.md`, tests, and diagnostics without
reopening core behavior questions.

## Context

`docs/SPEC.md` and `docs/STRICT_SEMANTICS.md` already define most of Tya's
current dynamic language surface. This spec records the accepted v1.0.0
decisions that are easy to accidentally weaken while polishing the compiler,
checker, formatter, self-host compiler, and standard-library integration.

The accepted direction favors canonical syntax, explicit errors, deterministic
runtime behavior, and no silent fallback. Tya remains dynamically typed.

## Behavior

- Truthiness remains strict and small:
  - only `nil` and `false` are falsey;
  - `0`, `""`, empty arrays, empty dictionaries, empty bytes, functions,
    classes, objects, tasks, channels, and resources are truthy.
- Collection read/write behavior remains asymmetric:
  - out-of-range array, string, and bytes reads return `nil`;
  - out-of-range array and bytes writes are errors;
  - dictionary missing-key reads return `nil`;
  - dictionary writes may create keys.
- Dictionary key access uses string indexes only.
  - `user["name"]` is valid;
  - dictionary key member access such as `user.name` is invalid;
  - dot access on dictionaries is reserved for documented dictionary receiver
    methods.
- Ordering operators remain numeric only.
  - `<`, `<=`, `>`, and `>=` require numbers;
  - string ordering, collation, and locale-aware comparison use explicit
    methods or standard-library APIs.
- `==` and `!=` do not dispatch to user-defined `equal?`.
  - scalar primitives use built-in equality;
  - arrays and dictionaries use deep equality;
  - functions, classes, objects, resources, tasks, and channels use identity;
  - user-defined domain equality remains explicit through `equal?`.
- Interface default-method conflicts require explicit class overrides.
  - unrelated defaults for the same method are ambiguous;
  - implemented-interface order must not silently choose a winner.
- Interface initializer hooks run only through an explicit constructor
  `super()` call.
  - the call position controls class and interface initialization order;
  - initializer hooks are not implicitly auto-run when a class lists
    `implements`.
- Closure capture remains value-snapshot based.
  - closures may read captured function-local values;
  - closures cannot reassign outer function bindings;
  - closures cannot mutate through captured outer mutable bindings;
  - mutation intended for a closure must be passed as an explicit parameter or
    held in an object designed for that mutation.
- Function implicit return remains part of Tya.
  - the final evaluated statement or expression in a function body is returned
    when no explicit `return` exits first;
  - explicit `return` remains available for early return and multi-return.
- Lowercase `.tya` files remain script/import compatible.
  - importing a lowercase `.tya` file evaluates its top-level module
    initialization deterministically according to the import graph;
  - Tya does not switch to declaration-only imports for v1.0.0.
- PascalCase class files keep one public class.
  - the public class name must match the filename;
  - additional classes and interfaces in that file are file-private;
  - multiple public classes in one class file remain invalid.
- Directory package imports keep the current bare-name/alias split.
  - `import net/http` imports public package classes and interfaces as reserved
    bare names;
  - `import net/http as http` creates a namespace binding and does not import
    public names bare.
- `raise` remains restricted to structured error values.
  - `raise error("failed")` is valid;
  - `raise nil` and other non-error values are invalid;
  - `nil` continues to represent absence rather than an error payload.
- `and` and `or` return booleans.
  - they use Tya truthiness;
  - they short-circuit;
  - they do not return either operand as a value.
- The v1.0.0 language surface includes the currently specified concurrency
  `select`, interface default/initializer stacking, native package metadata,
  and WebAssembly target rules.
  - these features may still need implementation, diagnostics, and test
    hardening;
  - they are not removed from the v1.0.0 specification surface.

## Scope

- `docs/SPEC.md`
- `docs/STRICT_SEMANTICS.md`
- parser, checker, evaluator, codegen, runtime, CLI, LSP, and formatter tests
  needed to enforce these decisions
- self-host sources only where required to preserve existing fixed-point gates
- focused fixtures under `tests/testdata/`

## Out of Scope

- Adding a static type system.
- Removing dynamic typing.
- Removing Go implementation files.
- Redesigning package manifests or dependency resolution.
- Adding new syntax beyond what is needed to enforce these accepted decisions.
- Changing standard-library APIs unrelated to these language-boundary rules.

## Acceptance Criteria

- `docs/SPEC.md` states every accepted decision in this spec without
  contradictory wording.
- `docs/STRICT_SEMANTICS.md` maps each runtime-validity boundary to an active
  test, diagnostic, or runtime error.
- Invalid programs for dictionary member key access, string ordering,
  invalid `raise` operands, interface default conflicts, implicit interface
  initializer assumptions, captured mutation, and multiple public class files produce
  clear parser/checker/runtime failures at the earliest feasible layer.
- Valid programs for truthiness, out-of-range reads returning `nil`, implicit
  returns, imported lowercase module initialization, bare directory imports,
  alias directory imports, and boolean-returning `and`/`or` remain accepted.
- Interpreter and generated C behavior agree for covered valid and invalid
  runtime cases, except for explicitly documented self-host compatibility
  exemptions.
- Existing self-host fixed-point gates remain valid.

## Tests To Add

Parser/checker tests:

- `TestCheckRejectsDictionaryKeyMemberAccess`
  - `user = { name: "tya" }; user.name`
  - Expected: checker diagnostic for dictionary key member access.

- `TestCheckRejectsStringOrdering`
  - `"a" < "b"`
  - Expected: checker or runtime diagnostic stating ordering requires numbers.

- `TestParseOrCheckRejectsInvalidRaiseOperands`
  - `raise nil`
  - `raise "failed"`
  - Expected: invalid source or checker diagnostic requiring an error value.

- `TestCheckRejectsInterfaceDefaultConflictWithoutOverride`
  - Two unrelated interfaces provide the same default method and a class
    implements both without overriding it.
  - Expected: checker diagnostic requiring an override.

- `TestCheckRejectsImplicitInterfaceInitializerAssumption`
  - A class implements an interface initializer hook but its constructor omits
    `super()`.
  - Expected: no automatic initializer execution; tests should assert either
    the documented explicit-call requirement or the runtime result that proves
    no implicit hook ran.

- `TestCheckRejectsCapturedMutableMutation`
  - A closure mutates an array or dictionary captured from an outer function.
  - Expected: checker diagnostic for captured mutation.

- `TestCheckRejectsMultiplePublicClassesInClassFile`
  - A PascalCase class file defines two public classes.
  - Expected: checker diagnostic.

Eval tests:

- `TestRunV1TruthinessFreeze`
  - Checks `nil` and `false` are falsey while `0`, `""`, `[]`, `{}`, and
    `b""` are truthy.

- `TestRunOutOfRangeReadsReturnNilAndWritesFail`
  - Reads out-of-range array/string/bytes values and writes out-of-range array
    or bytes indexes.
  - Expected: reads return `nil`; writes fail.

- `TestRunEqualityDoesNotDispatchToEqualQuestion`
  - A class defines `equal?(other)` with observable behavior.
  - Expected: `==` uses identity and does not call `equal?`.

- `TestRunInterfaceInitializerRequiresExplicitSuper`
  - A class implements an interface initializer hook.
  - Expected: hook effects appear only when the constructor calls `super()`.

- `TestRunImplicitReturnRemainsValid`
  - A function returns the final expression without explicit `return`.
  - Expected: returned value matches the final expression.

- `TestRunAndOrReturnBooleans`
  - Exercises truthy/falsy operands whose operand values are not booleans.
  - Expected: result values are `true` or `false`, not operand values.

Testscript coverage:

- `v1_language_semantics_freeze.txtar`
  - Covers CLI-level `tya check`, `tya run`, and `tya build` agreement for the
    accepted valid and invalid snippets above.

- `v1_package_import_surface.txtar`
  - Covers lowercase module import initialization order, bare directory imports,
    alias directory imports, and public-name collision diagnostics.

- `v1_class_file_surface.txtar`
  - Covers one-public-class class files and file-private helper
    class/interface visibility.

## Verification

```sh
go test ./internal/parser -count=1
go test ./internal/checker -count=1
go test ./internal/eval -count=1
go test ./internal/codegen -count=1
go test ./tests -run 'TestV.*Scripts|TestSelfhostV01Scripts|TestSelfhostV02Scripts|TestLSP' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
