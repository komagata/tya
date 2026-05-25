# Feature: Class-Level Constant Names

## Goal

Class bodies should accept `SCREAMING_SNAKE_CASE` class-level constants such as `ALPHABET`, so immutable class-owned values can be defined directly instead of using mutable static fields or method-local constants.

## Context

Issue #16 reports that Tya constants use `SCREAMING_SNAKE_CASE`, but class bodies can reject class-level constant members:

```tya
class Foo
  ALPHABET = "abc"
  value = -> Self.ALPHABET
```

Using this from another file can fail with:

```text
invalid field name ALPHABET
```

This blocks natural stdlib APIs such as `base64.Base64` alphabet tables and protocol constants. A completed historical spec exists at `feature-specs/completed/class-level-constants.md`; this spec narrows the active implementation queue item to the remaining issue #16 bug surface.

Current language docs already describe constants as `SCREAMING_SNAKE_CASE` and class constants as class-owned immutable members accessed through `Self.NAME` or public external class access.

## Behavior

- A class body may define a public class-level constant with `SCREAMING_SNAKE_CASE = value`.
- A class body may define a private class-level constant with `private SCREAMING_SNAKE_CASE = value` if private class constants are otherwise supported.
- Class-level constants are stored separately from mutable static fields.
- Inside the defining class, canonical access is `Self.NAME`.
- Public class constants can be read externally through the class value, including imported package classes, for example `pkg.Foo.ALPHABET`.
- Reassigning a class constant is rejected.
- Mutating a heap-backed value through a class constant is rejected where existing constant mutation rules apply.
- `static ALPHABET = ...` remains invalid or non-canonical according to the current checker/formatter behavior; it should not become the required spelling for constants.
- Lowercase class-body assignments remain fields or static fields according to existing syntax and naming rules.

## Scope

- Parser and AST handling for class-body `SCREAMING_SNAKE_CASE = value` members if they are currently misclassified as fields.
- Checker validation for class constant names, duplicate class members, constant reassignment, mutation-through-constant, and private/public visibility.
- Evaluator and generated-C support for reading class constants through `Self.NAME` and public external class access.
- Formatter support for preserving class constant declarations and canonical class member ordering.
- Tests covering the issue reproduction case and imported-file access.

## Out of Scope

- Changing top-level constant syntax or top-level constant semantics.
- Introducing a `const` keyword.
- Making `static ALPHABET = ...` the canonical spelling.
- Allowing lowercase names as class constants.
- Reworking broader class member visibility or inheritance semantics beyond class constant access.
- Migrating `stdlib/base64/Base64.tya` to use a class constant unless the implementation change naturally needs one focused fixture.

## Acceptance Criteria

- The issue #16 reproduction class parses and checks without `invalid field name ALPHABET`.
- `Self.ALPHABET` works inside an instance method.
- A public class constant is readable through an imported package class.
- Reassigning a class constant is rejected.
- Mutating through a class constant is rejected where the current constant rules reject mutation.
- `static ALPHABET = ...` remains covered by an explicit test for the current intended diagnostic or formatting behavior.
- Existing class constant tests remain green.
- The self-host v01 invariant is not regressed.

## Verification

```sh
go test ./internal/parser ./internal/checker ./internal/eval ./internal/codegen ./internal/formatter -count=1
go test ./tests -run 'TestV65Scripts/class_constants|TestSelfhostV01Scripts' -count=1
```
