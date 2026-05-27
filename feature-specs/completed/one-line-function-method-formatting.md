# Feature: One-Line Function and Method Formatting

## Goal
Make `tya format` prefer the one-line function form whenever a function or method body can be represented as a single expression within the canonical column limit.

## Context
Tya already accepts expression-bodied functions such as `greet = name -> "Hello, {name}"` and class methods such as `foo: -> "aaa"`. Current formatter behavior can preserve or produce block-bodied forms even when the body is only one expression. Formatted Syntax should choose the shorter one-line representation when it is semantically equivalent.

## Behavior
- Format function assignments with a single expression body as one line when the rendered line fits within the 80-column limit.
- Format class instance methods, class static methods, private methods, and constructors with a single expression body as one line when the rendered line fits.
- Format interface default methods with a single expression body as one line when the rendered line fits.
- Keep body-free interface method requirements as `name: ->`.
- A block body containing only `return expr` is formatted as `-> expr`, following the existing final-return omission rule.
- Do not one-line format when the body has multiple statements.
- Do not one-line format when the one-line rendering would exceed the 80-column limit.
- Do not one-line format when preserving attached comments requires the block shape.
- Do not one-line format bodies containing block-shaped constructs such as `if`, `match`, `try`, `while`, `for`, or other statements that cannot be represented as a single expression.
- Formatting must remain idempotent.

Examples:

```tya
foo = ->
  "aaa"
```

formats to:

```tya
foo = -> "aaa"
```

```tya
class Foo
  foo: ->
    return "aaa"
```

formats to:

```tya
class Foo
  foo: -> "aaa"
```

```tya
interface Named
  name: ->
```

stays body-free, while:

```tya
interface Named
  name: ->
    "default"
```

formats to:

```tya
interface Named
  name: -> "default"
```

## Scope
- Update formatter/unparser behavior for function assignments.
- Update formatter/unparser behavior for class members and interface default methods.
- Update Formatted Syntax docs in `docs/SPEC.md` and `docs/ja/spec.md`.
- Add or update formatter tests and rewrite catalog cases for top-level functions, class methods, static/private methods, constructors, final return, interface requirements, interface defaults, comments, and long-line fallback.
- Update affected test fixtures if formatter output changes.

## Out of Scope
- No change to parser accepted syntax.
- No change to function or method runtime semantics.
- No change to the 80-column canonical limit.
- No one-line formatting for multi-statement bodies or statement-only block constructs.
- No configuration option for formatter style.

## Acceptance Criteria
- `foo = ->\n  "aaa"` formats to `foo = -> "aaa"`.
- `foo = ->\n  return "aaa"` formats to `foo = -> "aaa"`.
- `class Foo\n  foo: ->\n    "aaa"` formats to `class Foo\n  foo: -> "aaa"`.
- `class Foo\n  static foo: ->\n    "aaa"` formats to `static foo: -> "aaa"` inside the class body.
- `class Foo\n  initialize: ->\n    "aaa"` formats to `initialize: -> "aaa"` when it fits.
- Body-free interface requirements remain `name: ->`.
- Interface default methods with one expression format to one line.
- Multi-statement bodies remain block-bodied.
- Bodies whose one-line rendering would exceed 80 columns remain block-bodied.
- Bodies with attached comments remain formatted without losing or misplacing comments.
- Running the formatter twice produces identical output.

## Verification
```sh
go test ./internal/formatter ./tests -count=1
go test ./... -count=1
```
