# Feature: Remove Legacy Primitive Helper APIs

## Goal

Keep ordinary instance-method syntax on primitive values, while removing legacy
helper API surfaces that expose primitive operations as receiverless builtins or
lowercase pseudo-module calls. Internal primitive fast paths are encouraged for
performance-sensitive primitive receiver methods when they implement the same
public instance-method behavior.

## Context

Current code still contains multiple API shapes for operations on strings,
arrays, dictionaries, numbers, and generic values:

- ordinary instance-method syntax such as `" tya ".trim()`, `[1, 2].len()`,
  `{ name: "Tya" }.has("name")`, `42.to_s()`, and `value.class`;
- receiverless legacy builtins such as `len(value)`, `trim(text)`,
  `contains(text, part)`, `keys(dict)`, `push(array, value)`, and
  `to_number(value)`;
- lowercase pseudo-module calls such as `string.trim(text)`,
  `array.len(items)`, and `dict.has(obj, key)`;
- standard-library class APIs such as `math/Math`.

The language specification no longer lists a primitive method surface. The
desired public rule is simpler: if a primitive operation is exposed as
`receiver.method(...)`, it behaves like an ordinary instance method. The
implementation may still lower that method through runtime primitive dispatch
or C helper functions for performance. Methods that are hot, allocation-heavy,
or part of core collection/string iteration should preferentially use primitive
fast paths instead of slower generic dispatch when that does not change public
semantics.

## Behavior

- Keep primitive receiver instance methods as valid public syntax when they look
  like ordinary instance methods:
  - strings: `"x".trim()`, `"x".upper()`, `"x".lower()`,
    `"x".contains("x")`, `"x".starts_with("x")`, `"x".ends_with("x")`,
    `"x".split(",")`, `"x".lines()`, `"x".chars()`, `"x".bytes()`,
    `"x".byte_len()`, `"x".blank?()`, `"x".present?()`
  - arrays: `items.len()`, `items.empty?()`, `items.first()`,
    `items.last()`, `items.push(value)`, `items.pop()`,
    `items.slice(start, end)`, `items.reverse()`, `items.sort()`,
    `items.join(separator)`, `items.map(fn)`, `items.filter(fn)`,
    `items.find(fn)`, `items.any(fn)`, `items.all(fn)`,
    `items.each(fn)`, `items.reduce(initial, fn)`
  - dictionaries: `dict.len()`, `dict.has(key)`, `dict.has?(key)`,
    `dict.get(key)`, `dict.get(key, fallback)`, `dict.set(key, value)`,
    `dict.delete(key)`, `dict.keys()`, `dict.values()`, `dict.entries()`,
    `dict.merge(other)`
  - values: `value.to_s()`, `value.class`, and `value.class.name`
- Keep user-defined class instance methods unchanged.
- Keep `Iterable` and `Sequence` instance-method syntax unchanged when it uses
  ordinary receiver syntax such as `items.iter()` or `items.sequence()`.
- Internal primitive method dispatch and runtime fast paths should be used for
  performance-sensitive public receiver methods when they only implement the
  public receiver-method surface.
- Performance-sensitive examples include string length and slicing helpers,
  array and dictionary length/access/mutation helpers, collection iteration,
  `map`/`filter`/`reduce`-style loops, value stringification, and class
  introspection.
- Remove receiverless legacy helper builtins for primitive operations from the
  active language surface, including at least:
  `kind`, `len`, `byte_len`, `char_len`, `trim`, `contains`, `starts_with`,
  `ends_with`, `replace`, `split`, `join`, `keys`, `values`, `has`, `push`,
  `to_number`.
- Remove lowercase pseudo-module primitive helper calls from the active
  language surface:
  `string.*`, `array.*`, `dict.*`, and `value.nil?`.
- Numeric math helper usage should go through `Math` from the standard library,
  not lowercase primitive pseudo-modules or receiverless math helpers when a
  class API exists.
- Diagnostics for removed helper APIs should point to the canonical
  instance-method or `Math` spelling.
- Standard library, examples, selfhost sources, and current fixtures should use
  canonical receiver-method syntax or standard-library class APIs.

## Scope

- Checker rules for removed receiverless primitive helper builtins and
  lowercase pseudo-module calls.
- Evaluator support for removing `string.*`, `array.*`, `dict.*`, `value.nil?`,
  and receiverless primitive helper calls where they remain accepted.
- C codegen support for the same removed helper APIs.
- Runtime support only where obsolete public entry points can be removed
  without breaking the retained receiver-method implementation.
- Standard-library code under `lib/`.
- Selfhost sources under `selfhost/v01/` and `selfhost/v02/`.
- Examples and current test fixtures under `tests/testdata/`.
- `docs/SPEC.md`, `docs/ja/spec.md`, and user-facing guide text if it mentions
  removed helper APIs.

## Out of Scope

- Removing ordinary primitive receiver instance methods.
- Removing `value.class`, `value.class.name`, or `value.to_s()`.
- Removing user-defined class methods.
- Removing primitive runtime fast paths used to implement valid
  instance-method syntax.
- Requiring valid primitive receiver methods to use slow generic class dispatch
  when a primitive implementation is simpler or faster.
- Redesigning `Iterable` or `Sequence`.
- Removing archived historical documentation under `docs/archive/`.
- Rewriting unrelated standard-library APIs.

## Acceptance Criteria

- `" tya ".trim()`, `[1, 2].len()`, `{ name: "Tya" }.has("name")`,
  `42.to_s()`, `value.class`, and `value.class.name` remain valid.
- A user-defined instance method call such as `user.name()` remains valid.
- `len(items)`, `trim(text)`, `keys(dict)`, `push(items, value)`, and
  `to_number(value)` are rejected or removed from current active paths.
- `string.trim(text)`, `array.len(items)`, `dict.has(obj, key)`, and
  `value.nil?(value)` are rejected or removed from current active paths.
- Removed helper API diagnostics include the canonical replacement where one
  exists.
- Hot primitive receiver methods keep or gain direct evaluator/codegen/runtime
  fast paths where the implementation can do so without changing public
  behavior.
- Numeric helper examples and stdlib usage use `Math` when a standard-library
  class API exists.
- Standard library, examples, selfhost sources, and current fixtures do not
  depend on removed receiverless or lowercase pseudo-module primitive helpers.
- Existing selfhost fixed-point invariants remain green.
- Full repository tests pass.

## Verification

```sh
go test ./internal/checker -count=1
go test ./internal/eval -count=1
go test ./internal/codegen -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./tests -run TestSelfhostV02Scripts -count=1
go test ./tests -run 'TestV01Scripts|TestV02Scripts|TestV03Scripts|TestV18Scripts|TestV59Scripts|TestV61Scripts|TestV63Scripts' -count=1
go test ./... -count=1 -timeout=20m
```
