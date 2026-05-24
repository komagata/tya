# Feature: Class-Level Constants

## Goal
Allow classes to define immutable constants using `SCREAMING_SNAKE_CASE` member assignments, so stdlib and user code can express values such as `Base64` alphabets as class-owned constants instead of mutable `static` fields or method-local bindings.

## Context
Tya already treats `SCREAMING_SNAKE_CASE` ordinary bindings as constants: they cannot be reassigned, and heap-backed values stored through constant bindings cannot be mutated through that binding. Class bodies currently accept private fields, methods, class variables, class methods, and constructors, but reject class member constants such as:

```tya
class Base64
  private ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
```

This blocks the most natural design for immutable class-owned protocol tables and similar implementation details. The current workaround in `stdlib/base64/Base64.tya` is a method-local `ALPHABET` binding inside `encode`.

Related bug: https://github.com/komagata/tya/issues/16

## Behavior
- A class body may contain a member assignment whose name is `SCREAMING_SNAKE_CASE`; that member is a class-level constant.
- Class-level constants use existing constant semantics:
  - They cannot be reassigned.
  - Mutable values reachable through the constant binding cannot be mutated through that binding.
- Visibility uses existing `private` syntax:
  - `private ALPHABET = "..."` defines a private class constant.
  - `ALPHABET = "..."` defines a public class constant.
- Class-level constants are accessed like class members:
  - From inside the defining class, canonical access is `Self.ALPHABET`.
  - Public constants may be accessed externally as `pkg.Class.ALPHABET`.
  - Private constants may not be accessed from outside the defining class.
- `static ALPHABET = ...` is not the canonical spelling for a constant. The canonical syntax is `ALPHABET = ...` or `private ALPHABET = ...`.
- Formatter and checker diagnostics should preserve the existing canonical member-access rule: inside the defining class, `ClassName.ALPHABET` should be reported or rewritten as `Self.ALPHABET` consistently with current class member access behavior.

## Scope
- Parser and AST support for class-level constant members.
- Checker support for class constant definitions, lookup, visibility, reassignment rejection, and mutation-through-constant rejection.
- C emitter/runtime support for reading class constants wherever class fields can currently be read.
- Formatter support preserving `private ALPHABET = ...` and public `ALPHABET = ...` in class bodies.
- LSP/documentation support equivalent to other class members where applicable.
- Specification documentation in `docs/SPEC.md`, `docs/ja/spec.md`, and frozen v1 docs if required by existing documentation policy.
- Tests covering parser, checker, formatter, codegen/runtime, strict diagnostics, and stdlib usage.
- Update `stdlib/base64/Base64.tya` to use:

```tya
class Base64
  private ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
```

## Out of Scope
- Adding a new `const` keyword.
- Changing top-level constant syntax or semantics.
- Making ordinary `snake_case` class fields immutable.
- Changing the meaning of existing mutable `static snake_case = ...` class fields.
- Broad stdlib refactoring beyond replacing Base64's method-local alphabet workaround.

## Acceptance Criteria
- `class Foo; VALUE = 1` style class constants parse and run.
- `private VALUE = 1` is accessible through `Self.VALUE` inside the class and rejected from outside the class.
- Public class constants are accessible as `pkg.Foo.VALUE`.
- Reassigning a class constant is rejected.
- Mutating a heap-backed value through a class constant is rejected.
- `static ALPHABET = ...` is not used as the canonical constant form; tests document the chosen diagnostic or formatting behavior.
- `Base64` uses a private class constant for its alphabet and all existing Base64, HMAC, and serialization behavior remains unchanged.
- Self-host invariant remains green.

## Verification
```sh
go test ./... -count=1
```

Focused checks should include:

```sh
go run ./cmd/tya test tests/stdlib_base64_test.tya
go run ./cmd/tya test tests/stdlib_serialization_test.tya
go test ./tests -run 'TestV65Scripts/v1_stdlib_hmac|TestSelfhostV01Scripts' -count=1 -timeout=10m
```
