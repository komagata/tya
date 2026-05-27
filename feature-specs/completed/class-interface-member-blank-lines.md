# Feature: Class and Interface Member Blank Lines

## Goal
Make `tya format` place exactly one blank line between every class or interface member, including when the next member has leading comments.

## Context
Formatted Syntax already orders class body members by category, static/instance, public/private, and member name. The formatter currently can omit the blank line before a member's leading comment, which visually attaches the comment block too tightly to the previous variable or method. Class and interface bodies should use the same member separation rule regardless of whether the next member starts with a comment.

This spec depends on `feature-specs/one-line-function-method-formatting.md` and should be implemented after it, because one-line method formatting changes the rendered shape of method members that this spacing rule separates.

## Behavior
- In a class body, format exactly one blank line between every adjacent member.
- In an interface body, format exactly one blank line between every adjacent member.
- The blank line appears after the previous member and before any leading comments attached to the next member.
- The rule applies to all class members: class constants, class variables, instance variables, static methods, instance methods, private members, and constructors.
- The rule applies to all interface members: body-free requirements and default methods.
- Do not add an extra blank line after the final member before the class or interface ends.
- Do not add more than one blank line between adjacent members.
- Formatting must preserve comment attachment and remain idempotent.

Examples:

```tya
class Foo
  a: 1
  # b docs.
  b: 2
```

formats to:

```tya
class Foo
  a: 1

  # b docs.
  b: 2
```

```tya
interface Iterator
  # has_next docs.
  has_next: ->
  # next docs.
  next: ->
```

formats to:

```tya
interface Iterator
  # has_next docs.
  has_next: ->

  # next docs.
  next: ->
```

## Scope
- Update formatter/unparser class body spacing.
- Update formatter/unparser interface body spacing.
- Update Formatted Syntax docs in `docs/SPEC.md` and `docs/ja/spec.md`.
- Add formatter tests for class members with comments, interface members with comments, mixed member categories, final-member behavior, and idempotence.
- Update affected formatter fixtures and stdlib formatting output if needed.

## Out of Scope
- No change to top-level blank-line rules.
- No change to member ordering rules.
- No change to comment attachment semantics except preserving the next member's leading comment with a blank line before it.
- No formatter configuration option.

## Acceptance Criteria
- A class member followed by a commented member formats with one blank line before the comment.
- An interface member followed by a commented member formats with one blank line before the comment.
- Adjacent uncommented class members still format with exactly one blank line between them.
- Adjacent uncommented interface members format with exactly one blank line between them.
- The final class or interface member is not followed by an extra blank line solely because it is final.
- Formatting preserves leading comments on the following member.
- Running the formatter twice produces identical output.

## Verification
```sh
go test ./internal/formatter ./tests -count=1
go test ./... -count=1
```
