# Feature: Iterable Sequence Protocol

## Goal

Make iteration in Tya unambiguous by standardizing `for item in value` as the
single primary way to consume multiple values, backed by explicit `Iterator`,
`Iterable`, and lazy `Sequence` protocols.

## Context

Tya already has language-level interfaces with default methods, primitive
collection methods such as `Array.map`, `Array.filter`, `Array.reduce`,
`String.chars`, and `Dict.entries`, and `for ... in` / `for ... of` syntax.
Current `for ... in` behavior is array-oriented, while dictionaries use
`for ... of` for key/value traversal. Tya's design goal is a language without
hesitation, so collection traversal should have one preferred spelling.

Lexical closures are a dependency for lazy sequence combinators because
`map`, `filter`, `take`, and related wrappers store functions for later
iterator consumption.

Primitive classes in Tya are wrapper class singletons over tagged `TyaValue`
runtime kinds, not heap boxes around every primitive value. This feature must
preserve that representation. Treating `Array`, `Dict`, and `String` as
`Iterable` is a language-level conformance rule, not permission to allocate a
wrapper instance for each primitive value.

## Behavior

- Add standard interfaces:

  ```tya
  interface Iterator
    has_next? = ->
    next = ->

  interface Iterable
    iter = ->

    sequence = ->
      IterableSequence.new(self)

  interface Sequence implements Iterable
    iter = ->

    map = fn ->
      MapSequence.new(self, fn)

    filter = fn ->
      FilterSequence.new(self, fn)

    take = n ->
      TakeSequence.new(self, n)

    drop = n ->
      DropSequence.new(self, n)

    reduce = initial, fn ->
      acc = initial
      for item in self
        acc = fn(acc, item)
      acc

    to_a = ->
      out = []
      for item in self
        out.push(item)
      out
  ```

- `Iterator.has_next?()` returns a boolean and does not advance the iterator.
- Repeated `has_next?()` calls without `next()` return the same result.
- `Iterator.next()` returns the next value and advances the iterator.
- Calling `next()` after `has_next?()` returns `false` raises a runtime error.
- `nil` is a valid iterated value and must not be used as an end sentinel.
- Iterators are consuming cursors and are not thread-safe.
- `Iterable.iter()` returns a new independent iterator each time.
- `for item in value` evaluates `value` once, calls `value.iter()` once, then
  consumes the returned iterator with `has_next?()` and `next()`.
- `for item, index in value` remains valid for Iterable values. `index` starts
  at `0` and increments by one per yielded item.
- `for ... of` is deprecated for dictionary traversal. The canonical spelling
  is `for entry in dict`.
- Primitive classes formally implement `Iterable`:
  - `Array implements Iterable`
  - `Dict implements Iterable`
  - `String implements Iterable`
- Primitive Iterable conformance must preserve the existing primitive runtime
  representation. Arrays, dictionaries, and strings remain tagged `TyaValue`
  values with process-global wrapper class objects.
- `Sequence implements Iterable`; all sequence wrappers must be usable directly
  in `for`.
- Array iteration yields elements in index order.
- String iteration yields characters using the same unit and order as
  `String.chars()`.
- Dict iteration yields entry dictionaries in dictionary storage order. Each
  yielded entry has this shape:

  ```tya
  { key: key, value: value }
  ```

- Code that wants only dictionary keys or values must say so explicitly:

  ```tya
  for key in dict.keys()
    print(key)

  for value in dict.values()
    print(value)
  ```

- `sequence()` returns a lazy, re-iterable `Sequence`.
- `Sequence.map`, `filter`, `take`, and `drop` do not consume the source when
  called. They return wrapper sequences.
- `for`, `to_a`, and `reduce` consume a sequence by requesting an iterator.
- Re-iterating a sequence must produce the same values when the underlying
  iterable has not changed:

  ```tya
  seq = items.sequence().map(double)
  print(seq.to_a())
  print(seq.to_a())
  ```

- Iterating a collection while mutating that same collection is unspecified for
  this feature. The implementation must not promise fail-fast behavior yet.

## Performance Model

- `Array`, `Dict`, and `String` must not be boxed into ordinary object
  instances merely to satisfy `Iterable`.
- `value.class` for primitive values must keep returning the existing
  process-global primitive class singleton.
- `for ... in` over primitive arrays, dictionaries, and strings should lower to
  direct runtime loops or equivalent fast paths. It must not require allocating
  a user-visible iterator object for the common `for` case.
- Explicit calls to `array.iter()`, `dict.iter()`, or `string.iter()` may return
  iterator values. Those iterators may be runtime-backed objects or resources,
  but their allocation cost is paid only when the iterator is explicitly
  requested or when a generic Iterable path needs it.
- User-defined `Iterable` values and `Sequence` wrappers may use normal method
  dispatch through `iter()`, `has_next?()`, and `next()`.
- Codegen may specialize primitive receiver calls such as `array.sequence()` or
  `array.iter()` when the receiver kind is statically or dynamically known, as
  long as observable behavior matches the protocol.
- Existing eager primitive helpers such as `Array.map`, `Array.filter`, and
  `Array.reduce` may keep their current direct runtime implementations.

## Scope

- Add standard library interface files for `Iterator`, `Iterable`, and
  `Sequence`.
- Add standard library sequence wrapper classes:
  - `IterableSequence`
  - `MapSequence`
  - `FilterSequence`
  - `TakeSequence`
  - `DropSequence`
- Add iterator implementations for primitive values:
  - Array iterator
  - Dict entry iterator
  - String character iterator
- Add primitive methods:
  - `Array.iter()`
  - `Array.sequence()`
  - `Dict.iter()`
  - `Dict.sequence()`
  - `String.iter()`
  - `String.sequence()`
- Teach `for ... in` codegen/runtime lowering to use the Iterable protocol for
  arrays, dictionaries, strings, sequences, and user objects.
- Preserve primitive fast paths for `for ... in` over arrays, dictionaries, and
  strings so canonical iteration does not regress primitive loop performance.
- Teach the checker that primitive classes conform to `Iterable` where interface
  conformance is checked or documented.
- Update `docs/SPEC.md`, `docs/API.md`, and `docs/STDLIB.md`.
- Add black-box testscript coverage for primitive iteration, user-defined
  Iterable classes, lazy Sequence behavior, dictionary entry iteration,
  exhausted iterator errors, and deprecated `for ... of` behavior.

## Out of Scope

- Generic type parameters such as `Iterator<T>`.
- Parallel sequence execution.
- Infinite range syntax or a new `Range` class.
- Fail-fast mutation detection while iterating.
- Removing existing eager `Array.map`, `Array.filter`, or `Array.reduce` in this
  feature.
- Removing `for ... of` immediately. It may remain as a compatibility alias or
  produce a deprecation diagnostic, but `for ... in` is the canonical form.
- Making `each(fn)` a primary standard-library API. Side-effect iteration is
  written with `for`.

## Acceptance Criteria

- `for item in [1, 2, 3]` prints `1`, `2`, `3` in order.
- `for item, index in [10, 20]` binds `item/index` as `10/0`, then `20/1`.
- `for char in "tya"` yields the same values as `"tya".chars()`.
- `for entry in { name: "Tya" }` yields a dictionary with keys `key` and
  `value`, where `entry["key"] == "name"` and `entry["value"] == "Tya"`.
- `dict.keys()` and `dict.values()` remain the explicit way to iterate only
  keys or values.
- A user-defined class that implements `Iterable.iter()` works with `for`.
- A user-defined iterator can yield `nil` as a normal value.
- Calling `next()` on an exhausted iterator raises a runtime error.
- `items.sequence().filter(fn).map(fn).take(2).to_a()` is lazy and produces the
  expected array without creating intermediate arrays for `filter` or `map`.
- `Sequence` values can be used directly in `for`.
- Re-iterating the same sequence works when the source collection is unchanged.
- `for ... of` is no longer documented as the preferred dictionary traversal
  spelling.
- Existing eager array helpers still pass their current tests.
- `TestSelfhostV01Scripts` and `TestSelfhostV02Scripts` continue to pass.

## Verification

```sh
go test ./... -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./tests -run TestSelfhostV02Scripts -count=1
```
