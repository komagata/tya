# Feature: Validation Validators

## Goal

Add chainable number and string validators under `validation`, so Tya standard-library code and user code can express common runtime preconditions without repeating type, range, and length checks.

## Context

Tya is dynamically typed. Runtime classes include `Number`, `String`, `Array`, `Dict`, `Boolean`, and `Nil`; there is no separate runtime `Integer` class. Integer expectations are currently checked by verifying `value.class == Number` and `value == value.to_i()`.

Several stdlib files repeat small validation blocks for numbers, integer ranges, string presence, and length-like constraints. `lib/color.tya` is the immediate target: color channels should be validated through a reusable number validator instead of a `Color`-specific helper.

This feature depends on `feature-specs/completed/bang-callable-names.md`, because the primary raising API is named `validate!`.

## Behavior

- Add `lib/validation/number_validator.tya` exporting `NumberValidator`.
- Add `lib/validation/string_validator.tya` exporting `StringValidator`.
- Add a validation package surface file so `import validation/*` is the preferred import path and exposes both validators directly.
- The validation package surface file imports the validator implementations with `import validation/*`, so users can write:

```tya
import validation/*

r = NumberValidator(r, "r", "color.channel")
  .integer()
  .between(0, 255)
  .validate!()

name_validator = StringValidator(name, "name", "user")
  .present()
  .max(40)
  .validate()
```

- Individual imports are also valid:

```tya
import validation/number_validator
import validation/string_validator
```

- `NumberValidator` constructor:

```tya
NumberValidator(value, name = "value", context = "")
```

- `StringValidator` constructor:

```tya
StringValidator(value, name = "value", context = "")
```

- `name` is used in error messages.
- `context` is an optional message prefix. When present, errors use `"context: message"`. When empty, errors omit the prefix.
- `NumberValidator` always requires the value to be a `Number` during validation.
- `StringValidator` always requires the value to be a `String` during validation.
- `NumberValidator` exposes these chainable constraint methods, each returning `self`:
  - `integer()`
  - `gt(n)`
  - `gte(n)`
  - `lt(n)`
  - `lte(n)`
  - `between(min, max)`
  - `positive()`
  - `non_negative()`
- `positive()` means greater than `0`.
- `non_negative()` means greater than or equal to `0`.
- `between(min, max)` is inclusive for both bounds.
- `StringValidator` exposes these chainable constraint methods, each returning `self`:
  - `present()`
  - `min(n)`
  - `max(n)`
  - `between(min, max)`
- For `StringValidator`, `min`, `max`, and `between` apply to string length.
- Both validators expose:
  - `validate()`
  - `validate!()`
  - `valid?()`
  - `errors`
  - `value`
- `validate()` runs validation, stores validation state on the instance, and returns `self`.
- `validate()` does not raise on invalid input.
- `validate!()` runs validation and raises the first validation error when invalid.
- `validate!()` returns the validated value when valid.
- `validate!()` returns `value.to_i()` when `NumberValidator.integer()` is present.
- `validate!()` returns the original number when no integer constraint is present.
- `validate!()` returns the original string for `StringValidator`.
- `valid?()` reports the result of the most recent validation run.
- `errors` stores validation error messages from the most recent validation run.
- `value` stores the input value and remains available after validation. `validate()` returns `self`; callers inspect `validator.value`, `validator.valid?()`, and `validator.errors`.

Example number errors:

```text
color.channel: r must be a number
color.channel: r must be an integer
color.channel: r must be between 0 and 255
r must be an integer
```

Example string errors:

```text
user: name must be a string
user: name must be present
user: name must be at least 3 characters
user: name must be at most 40 characters
```

- Update `lib/color.tya` to use `NumberValidator(value, name, "color.channel").integer().between(0, 255).validate!()` for channel validation.
- Keep `Color` named color factories and `rgb`/`rgba` as static methods if they are already static in the implementation branch.

## Scope

- New validator stdlib files under `lib/validation/`.
- New validation package surface file for `import validation/*`.
- `lib/color.tya` channel validation migrated to `NumberValidator`.
- Standard-library surface documentation in `docs/SPEC.md`.
- Tests for `NumberValidator`, `StringValidator`, validation package imports, individual imports, `validate()`, `validate!()`, errors, and `Color` integration.

## Out of Scope

- Adding a runtime `Integer` class.
- Adding static typing or compile-time validation.
- Replacing all stdlib numeric and string validation sites in the initial implementation.
- Regex or pattern validation for `StringValidator`.
- Collection validators, dictionary validators, schema validators, or Rails-style model validation.
- A facade object such as `Validation.number(...)`; the package surface is for import/export convenience, not a separate builder API.
- Changing `Number` primitive semantics.

## Acceptance Criteria

- `import validation/*` makes `NumberValidator` and `StringValidator` directly available.
- `import validation/number_validator` and `import validation/string_validator` continue to work for individual imports.
- `NumberValidator(12, "r", "color.channel").integer().between(0, 255).validate!()` returns `12`.
- `NumberValidator(12.5, "r", "color.channel").integer().validate!()` raises `color.channel: r must be an integer`.
- `NumberValidator(256, "r", "color.channel").integer().between(0, 255).validate!()` raises `color.channel: r must be between 0 and 255`.
- `NumberValidator(1).positive().lt(25).validate!()` returns `1`.
- `NumberValidator(0).positive().validate!()` raises an error.
- `NumberValidator(0).non_negative().validate!()` returns `0`.
- `StringValidator("tya", "name", "user").present().min(2).max(10).validate!()` returns `"tya"`.
- `StringValidator("", "name", "user").present().validate!()` raises `user: name must be present`.
- `StringValidator("toolong", "name", "user").max(3).validate!()` raises `user: name must be at most 3 characters`.
- `validate()` returns the validator instance, does not raise, updates `valid?()`, and populates `errors`.
- `validate!()` raises the first stored validation error when invalid and returns the validated value when valid.
- `Color` channel validation uses `NumberValidator` and preserves the tested `Color` behavior.

## Verification

```sh
go run ./cmd/tya test tests/stdlib_validation_test.tya
go run ./cmd/tya test tests/stdlib_color_test.tya
go test ./... -count=1
```
