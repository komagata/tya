# Feature: Bang Callable Names

## Goal

Allow function and method names to end with `!`, so Tya code can mark destructive or dangerous callable variants such as `parse!` without adding a new runtime feature.

## Context

Tya already allows predicate callable names with a trailing `?`, such as `empty?`, `has?`, and `private?`. There is no equivalent suffix for callables that mutate their receiver or represent a more dangerous variant of a safer API.

Ruby commonly uses a trailing `!` for methods that are destructive or otherwise more dangerous than a paired non-bang method. Tya should support the same naming convention for callables while keeping `!` as a suffix marker, not a standalone logical operator.

Current identifier docs and parser/checker rules describe snake_case names plus existing callable suffixes. This feature extends the callable-name surface only; it does not change variable, constant, field, import, or file naming.

## Behavior

- A trailing `!` is valid only for callable names:
  - top-level function declarations;
  - class instance methods;
  - static methods;
  - private methods;
  - abstract methods;
  - override methods;
  - interface method requirements;
  - interface default methods;
  - function calls, method calls, and member calls.
- `!` is allowed only as exactly one trailing suffix.

```tya
parse! = text -> Url.parse(text)

class Url
  parse!: text ->
    self.scheme = "https"
    self
```

- Calls preserve `!` as part of the callable name:

```tya
url.parse!("https://example.com")
parse!("https://example.com")
```

- `!` is not valid for non-callable names:

```tya
value! = 1        # invalid

class Url
  value!: ""      # invalid field or variable name
```

- `!` may not be repeated or appear in the middle of a name:

```tya
parse!      # valid
parse!!     # invalid
parse!_now  # invalid
```

- `?` and `!` suffixes are mutually exclusive:

```tya
empty?   # valid predicate callable
parse!   # valid bang callable
empty?!  # invalid
parse!?  # invalid
```

- `!=` remains the not-equal operator. Lexing and parsing must continue to distinguish `name!` from `!=`.
- `!` by itself remains invalid syntax; Tya continues to use `not` for logical negation.
- `!` is a naming convention, not a semantic checker rule. The compiler does not verify that a bang callable mutates state or raises an exception.
- Standard-library naming guidance:
  - Use `!` for destructive methods or methods that are intentionally more dangerous than a paired safe/non-destructive API.
  - Do not mechanically add `!` to every method that can raise. Many ordinary APIs can fail; `!` should carry useful contrast.
  - Prefer a non-bang safe or non-destructive form when exposing a bang variant.

Example convention:

```tya
parsed = Url.parse(text) # returns a new Url
url.parse!(text)         # mutates url and returns self
```

- `tya format` preserves bang callable names and emits canonical member syntax normally:

```tya
class Url
  parse!: text -> self

Url.parse!(text)
```

## Scope

- Lexer support for recognizing `!` as a callable-name suffix while preserving `!=`.
- Parser support for bang names anywhere callable names are currently accepted.
- Checker validation so bang names are accepted for callables and rejected for non-callable bindings.
- Formatter/unparser support for preserving bang names.
- Evaluator and C codegen method/function lookup support if they currently assume callable names cannot end with `!`.
- Documentation updates in `docs/SPEC.md` and `docs/v1.0/SPEC.md`.
- Tests covering declarations, calls, invalid names, formatter output, eval, codegen, and interface requirements/defaults.

## Out of Scope

- Static enforcement that bang callables mutate state, raise, or have a paired safe method.
- Adding `!` as logical negation.
- Allowing `!` in variable names, constants, class variables, instance variables, import aliases, file names, or import path segments.
- Allowing multiple suffixes or combined `?`/`!` names.
- Renaming existing stdlib APIs to bang names as part of this feature.
- Changing `!=` operator behavior.

## Acceptance Criteria

- A top-level function named `parse!` can be declared and called.
- A class can define and call an instance method named `parse!`.
- A class can define and call a static method named `parse!`.
- A class can define and call a private method named `normalize!` internally.
- Abstract methods, override methods, interface requirements, and interface default methods may use bang names.
- Non-callable names ending in `!` are rejected with clear diagnostics.
- `parse!!`, `parse!_now`, `parse!?`, and `empty?!` are rejected.
- `!=` continues to parse and behave as not-equal.
- `!value` remains invalid syntax; `not value` remains the logical negation spelling.
- `tya format` preserves bang callable names in declarations and calls.
- Evaluator and generated C execution both dispatch bang function and method calls correctly.
- The self-host v01 invariant is not regressed.

## Verification

```sh
go test ./internal/lexer ./internal/parser ./internal/checker ./internal/formatter ./internal/eval ./internal/codegen -count=1
go test ./tests -run 'TestV19Scripts|TestSelfhostV01Scripts' -count=1
go test ./... -count=1
```
