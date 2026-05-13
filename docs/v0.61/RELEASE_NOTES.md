# Tya v0.61 Release Notes

v0.61 grows `interface` from a requirement-only contract into Tya's
stackable behavior mechanism.

## Highlights

- Interfaces may define default instance methods.
- Interface defaults can satisfy body-free requirements from other interfaces.
- Interfaces may contribute instance fields.
- Interfaces may define zero-argument `initialize` hooks.
- Interface initializer hooks run at a class constructor's `super()` point.
- Class overrides can call `super()` into the interface default stack.
- Interface default methods can call `super()` through stacked defaults.
- Same-name interface fields and unrelated default methods require explicit
  class resolution.

## Non-Goals Kept

- No `trait` keyword was added.
- Static interface members remain invalid.
- Private interface members remain invalid.
- `Self` remains invalid inside interface methods.

## Verification

The release includes script coverage for:

- default methods and default-method requirements;
- interface-contributed fields;
- interface initializer hooks and constructor `super()` placement;
- class `super()` into interface defaults;
- stacked interface `super()` with rightmost-wraps-leftmost order;
- default-method and field conflict diagnostics.
