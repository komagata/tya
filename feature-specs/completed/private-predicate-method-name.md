# Feature: Private Predicate Method Name

## Goal

`private?` should be accepted as a normal predicate callable name while preserving `private` as a class member visibility modifier.

## Context

Issue #12 reports that this class method cannot currently be defined or called:

```tya
class Address
  private? = ->
    true

addr = Address()
print(addr.private?())
```

The parser or checker treats the `private` prefix as a visibility modifier instead of recognizing `private?` as one predicate method name. This blocked the class-style stdlib migration for `net/ip/Address.tya`, where the API had to be renamed to `private_address?` as a workaround.

Predicate names with a trailing `?` are already part of Tya's callable surface, including names such as `empty?`, `has?`, `equal?`, `closed?`, and `cancelled?`. The bug is specific to the overlap between the visibility keyword `private` and the predicate suffix.

## Behavior

- `private? = -> ...` is a valid instance method definition inside a class.
- `static private? = -> ...` is a valid static method definition inside a class.
- `addr.private?()` and `Address.private?()` call the corresponding predicate methods.
- `private foo = -> ...` remains a visibility-modified method definition.
- `private static foo = -> ...` and any currently accepted visibility/static modifier ordering remain unchanged.
- Bare `private` without `?` remains reserved for the visibility modifier in class member declarations where it is currently valid.
- The same predicate callable-name validation rules apply to `private?` as to other `?` names: it must be a callable/method where predicate names are allowed, and non-callable predicate bindings remain invalid where the checker already rejects them.
- Formatter output preserves `private?` as a method name and does not rewrite it to `private ?` or treat it as a visibility modifier.

## Scope

- Lexer/parser handling for callable names with `?` suffix when the base token text is `private`.
- Class member parsing for visibility modifiers versus predicate method names.
- Checker validation for method names and predicate callable names.
- Codegen/eval paths only if method lookup or call emission mishandles `private?`.
- Formatter tests or golden tests that ensure `private?` remains canonical method-name output.
- Specification tests covering the issue's reproduction case.

## Out of Scope

- Renaming existing stdlib APIs such as `private_address?` back to `private?`.
- Changing the public/private visibility model.
- Allowing arbitrary reserved words as method names.
- Changing predicate naming rules for values, fields, class variables, or constants beyond the specific valid callable name `private?`.
- Changing method privacy enforcement semantics outside the existing `private` modifier behavior.

## Acceptance Criteria

- The issue #12 reproduction program parses, checks, runs, and prints `true`.
- A class can define and call an instance method named `private?`.
- A class can define and call a static method named `private?`.
- Existing visibility modifier syntax such as `private helper = -> ...` continues to parse and behave as before.
- Invalid predicate non-callable definitions remain rejected where they are rejected today.
- `tya format` preserves `private? = ->` as a predicate method definition.
- The self-host v01 invariant is not regressed by the parser/checker change.

## Verification

```sh
go test ./internal/parser ./internal/checker ./internal/formatter ./internal/eval ./internal/codegen -count=1
go test ./tests -run 'TestV19Scripts|TestSelfhostV01Scripts' -count=1
```
