# Feature: Primitive string and bytes sequences

## Goal

Make character and byte traversal reusable through Tya's existing `Iterable` / `Sequence` model so stdlib code can avoid hand-written `byte_len()` / index loops for common scans such as CSV field escaping.

## Context

`stdlib/sequence.tya` defines `Sequence implements Iterable` with `map`, `filter`, `take`, `drop`, `reduce`, and `to_a`. `docs/SPEC.md` already says arrays, dictionaries, and strings conform to `Iterable` as primitive values. Runtime support includes `tya_iter` and `tya_sequence`, and string iteration already exists in the runtime path. However:

- `Sequence` does not provide common predicate traversal methods such as `any?`, `all?`, `find`, or `each`.
- Bytes traversal is not documented as an `Iterable` primitive.
- `stdlib/csv/Csv.tya` still manually loops through string characters in `escape_field`.
- Adding ad hoc names such as `contains_any` is not desired; traversal should flow through `Sequence`.

Relevant files:

- `stdlib/sequence.tya`
- `stdlib/iterable.tya`
- `stdlib/iterator.tya`
- `stdlib/bytes/Bytes.tya`
- `stdlib/csv/Csv.tya`
- `docs/SPEC.md`
- `runtime/tya_runtime.c`
- `runtime/tya_runtime.h`
- `internal/eval/eval.go`
- `internal/codegen/c.go`
- `tests/testdata/v63_tool/lexical_closures.txtar`
- `tests/stdlib_csv_test.tya`

## Behavior

- `String` remains a primitive `Iterable`.
- String iteration yields characters as strings, not numeric bytes.
- `Bytes` becomes a primitive `Iterable`.
- Bytes iteration yields byte values as numbers in the range `0..255`.
- Primitive strings and bytes expose `.iter()` and `.sequence()` like arrays and dictionaries.
- `Sequence` gains these methods:
  - `each(fn)`: calls `fn(item)` for every item and returns `nil`.
  - `any?(fn)`: returns `true` when `fn(item)` is truthy for any item; short-circuits.
  - `all?(fn)`: returns `true` when `fn(item)` is truthy for every item; short-circuits.
  - `find(fn)`: returns the first item where `fn(item)` is truthy, otherwise `nil`; short-circuits.
- Existing `Sequence.map`, `filter`, `take`, `drop`, `reduce`, and `to_a` behavior must remain unchanged.
- `for item in "abc"` continues to yield `"a"`, `"b"`, `"c"`.
- `for byte in b"abc"` yields `97`, `98`, `99`.
- `Csv.escape_field` should use `field.sequence().any?(...)` for quote detection instead of a manual loop.
- CSV quote escaping should still use `field.replace("\"", "\"\"")` without first checking whether quotes exist.

Example target shape:

```tya
private escape_field: field ->
  needs_quote = field.sequence().any?(c ->
    c == self.options["separator"] or c == "\"" or c == "\n" or c == chr(13)
  )
  if not needs_quote
    return field
  "\"" + field.replace("\"", "\"\"") + "\""
```

Bytes example:

```tya
total = b"ABC".sequence().reduce(0, sum, byte -> sum + byte)
```

## Scope

- Extend `Sequence` interface/default methods in `stdlib/sequence.tya`.
- Ensure runtime primitive member dispatch exposes `iter` and `sequence` for `Bytes`.
- Ensure runtime iterator support handles bytes as numeric byte values.
- Ensure checker and codegen recognize bytes primitive `iter` / `sequence` methods where primitive method allowlists exist.
- Update `docs/SPEC.md` to state:
  - strings iterate by character strings;
  - bytes iterate by numeric byte values;
  - `Sequence` includes `each`, `any?`, `all?`, and `find`.
- Refactor `stdlib/csv/Csv.tya` to use sequence traversal in `escape_field`.
- Add focused tests for string sequence methods, bytes sequence methods, and CSV escaping.

## Out of Scope

- Do not add `String.contains_any`, `each_char`, `any_char?`, `each_byte`, or `any_byte?`.
- Do not change string indexing semantics.
- Do not change `String.bytes()` or `Bytes.to_array()` semantics.
- Do not add static type checking for sequence item types.
- Do not require every stdlib manual loop to be migrated in this feature.
- Do not change dictionary iteration behavior; dictionaries continue to yield entry dictionaries.

## Acceptance Criteria

- `"abc".sequence().to_a()` returns `["a", "b", "c"]`.
- `"abc".sequence().any?(c -> c == "b")` returns `true`.
- `"abc".sequence().all?(c -> c != "z")` returns `true`.
- `"abc".sequence().find(c -> c == "b")` returns `"b"`.
- `b"ABC".sequence().to_a()` returns `[65, 66, 67]`.
- `b"ABC".sequence().reduce(0, sum, byte -> sum + byte)` returns `198`.
- `Sequence.any?`, `all?`, and `find` short-circuit and do not evaluate later items once the result is known.
- `Sequence.each(fn)` visits each item and returns `nil`.
- `Csv.escape_field` no longer contains a manual `while` loop for quote detection.
- Existing sequence, array, dictionary, and CSV tests continue to pass.

## Verification

```sh
go test ./internal/eval ./internal/codegen -run 'Sequence|Bytes|String|Dict|Csv' -count=1
go test ./tests -run 'TestV63Scripts|TestStdlib' -count=1
go run ./cmd/tya test tests/stdlib_csv_test.tya
go test ./... -count=1
```
