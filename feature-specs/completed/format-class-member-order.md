# Feature: Format Class Member Order

## Goal

Make `tya format` emit a deterministic class member order so class bodies have
one canonical layout across constants, variables, fields, and methods.

## Context

Tya treats the formatter as part of the language's Formatted Syntax. Class
members are currently parsed into separate AST collections for constants,
class variables, instance fields, and methods, but formatted output does not yet
define the complete canonical order within a class body.

The requested order is:

1. constants
2. variables and fields
3. methods

Within each category, static members come before instance members. Within each
static/instance group, public members come before private members. Members in
the same final group are sorted alphabetically by member name, except
`initialize`, which is fixed as the first public instance method.

## Behavior

- `tya format` reorders members inside each class body.
- Class member order is a Formatted Syntax rule, not a parser/checker validity
  rule.
- Unordered class members remain accepted by `tya check`, `tya run`, and
  `tya build` as long as the program is otherwise valid.
- Class member categories are emitted in this order:
  - constants
  - variables and fields
  - methods
- Constants are class-owned immutable members written with `SCREAMING_SNAKE_CASE`.
- Variables and fields include:
  - static class variables
  - instance fields
- Methods include instance methods and static class methods, including
  abstract and override methods where accepted by the parser.
- Within each category, emit static members before instance members.
- Within each static/instance group, emit public members before private
  members.
- Within each final group, sort alphabetically by member name.
- `initialize` is a special case:
  - it remains an instance method;
  - it is treated as public;
  - it is emitted before all other public instance methods regardless of
    alphabetical order.
- Private `initialize` is not expected to be valid style. If accepted by the
  parser today, formatting should not invent new semantics; it may keep the
  same parser/checker validity rules while still placing `initialize` at the
  top of the instance-method area.
- Member doc comments and immediately preceding comments move with the member
  they describe.
- Blank lines between reordered member groups should follow existing formatter
  conventions for class bodies.
- Formatting must remain idempotent: formatting already formatted class bodies
  must produce no further changes.
- Reordering must preserve runtime behavior for valid programs.

## Scope

- Formatter:
  - class-body member ordering in `internal/formatter`;
  - comment attachment and movement for class members;
  - idempotency tests for reordered classes.
- AST/parser integration only as needed to let the formatter retain enough
  source-adjacent comments to move them with members.
- Documentation:
  - `docs/SPEC.md` Formatted Syntax wording for class member order.
  - Versioned/current docs only if the repo convention requires mirroring the
    current spec update.
- Tests:
  - formatter unit tests for class ordering;
  - CLI `tya format --check` drift test for unordered class members if useful;
  - corpus/idempotency coverage for classes with constants, variables, fields,
    methods, static members, private members, comments, `initialize`,
    `abstract`, and `override`.

## Out of Scope

- Interface member ordering.
- Top-level declaration ordering outside class bodies.
- Reordering methods or fields across different classes.
- Renaming members or changing visibility/static modifiers.
- Alphabetical sorting of imports, modules, functions, or non-class
  declarations beyond existing formatter behavior.
- Changing parser/checker language validity rules for private constructors,
  abstract methods, override methods, or duplicate members.
- Inferring intent for standalone section comments that are not immediately
  attached to a member.
- Rewriting stdlib files solely to apply the new order, except where tests or
  fixtures need updates.

## Acceptance Criteria

- `tya format` rewrites an unordered class into this order:
  - public static constants;
  - private static constants;
  - public static variables;
  - private static variables;
  - public instance fields;
  - private instance fields;
  - public static methods;
  - private static methods;
  - `initialize`;
  - other public instance methods;
  - private instance methods.
- Members in each final group are sorted alphabetically by name.
- `initialize` appears before other public instance methods even when another
  method would sort earlier alphabetically.
- Member comments immediately preceding a member move with that member after
  reordering.
- Formatting class bodies with constants, variables, fields, methods, static,
  private, `abstract`, `override`, and comments is idempotent.
- `tya format --check` reports drift for a file whose class members are not in
  canonical order.
- `tya check`, `tya run`, and `tya build` do not fail solely because class
  members are outside canonical formatted order.
- The implementation does not change parser/checker/runtime semantics.
- The self-host fixed-point invariant remains green.

## Verification

```sh
go test ./internal/formatter -count=1
go test ./tests -run 'TestCLIFormat|TestFormat|TestSelfhostV01Scripts' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
