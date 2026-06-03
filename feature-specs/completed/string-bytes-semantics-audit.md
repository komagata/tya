# Feature: String and Bytes Semantics Audit

## Goal
Make Tya's text and binary data model internally consistent before v1.0.0 by treating `String` as Unicode text and `Bytes` as raw bytes across the specification, runtime, compiler, standard library, and tests.

## Context
Tya is still before its first stable release, so broad semantic fixes are acceptable when they make the language easier to reason about. Current docs already state that `String` values are UTF-8 text, text APIs reject invalid UTF-8, binary APIs return `Bytes`, string indexing uses Unicode runes, and bytes indexing returns integer bytes. The implementation mostly follows this direction, but the standard library still contains code paths that mix `byte_len()` with `String#[]`, which caused `xml.Xml.parse` to fail on non-ASCII XML attribute values in `komagata/tya#33`.

Relevant files include:

- `docs/SPEC.md`
- `docs/STRICT_SEMANTICS.md`
- `docs/ja/spec.md`
- `internal/eval/eval.go`
- `internal/checker/`
- `internal/codegen/`
- `tests/stdlib_string_test.tya`
- `tests/bytes_type_test.tya`
- `tests/testdata/v65_strict/`

## Behavior
- `String` is Unicode text. It is not a byte array.
- `Bytes` is raw binary data. It is not text unless explicitly decoded.
- `String#len()` returns the number of Unicode characters/runes.
- `String#char_len()` either remains as an alias for `String#len()` or is removed only if all docs/tests/call sites are updated in the same change.
- `String#byte_len()` returns the UTF-8 byte length.
- `String#[]` uses zero-based Unicode character/rune indexes and returns a one-character `String`.
- `String#slice(start, end)` uses Unicode character/rune indexes.
- String iteration yields one-character `String` values and reports the loop index in Unicode character/rune indexes.
- `String#index_of(needle, start)` accepts and returns Unicode character/rune indexes.
- Regex match dictionaries continue to report `start` and `end` as Unicode rune indexes.
- `Bytes#len()` returns byte length.
- `Bytes#[]` uses zero-based byte indexes and returns an integer byte.
- `Bytes#slice` and `bytes_slice` use byte indexes.
- `bytes_of(text)` encodes a `String` as UTF-8 bytes.
- `bytes_text(bytes)` decodes UTF-8 and raises `invalid UTF-8` for invalid byte sequences.
- Text file APIs return `String` and reject invalid UTF-8.
- Binary file APIs return `Bytes` and do not validate UTF-8.
- Operators remain strict: `+` concatenates only `String + String` or `Bytes + Bytes`; mixed `String`/`Bytes` concatenation remains invalid.
- Any standard library code that needs byte-level scanning must explicitly convert to or operate on `Bytes`; it must not use `String#[]` as a byte accessor.
- Any standard library code that scans text with `String#[]` must use character/rune indexes consistently and must not compare those indexes against `byte_len()`.

## Scope
- Update the current editable specification and strict-semantics documentation where the String/Bytes model is incomplete or inconsistent.
- Add focused tests for non-ASCII strings to `tests/stdlib_string_test.tya`.
- Extend `tests/bytes_type_test.tya` where needed to prove byte-index behavior remains separate from string behavior.
- Add or update Go runtime tests around string indexing, string length, string slicing, string iteration, bytes indexing, UTF-8 encode/decode, and invalid UTF-8 rejection.
- Update checker/codegen tests if any current generated-runtime behavior disagrees with the finalized semantics.
- Audit Tya standard library call sites that mix `byte_len()` with `String#[]`, including parser-like modules such as XML, TOML, JSON, URL, template, color, and base64.
- The audit should produce direct fixes for small inconsistencies and identify parser rewrites that belong to the follow-up parser-scanner spec.

## Out of Scope
- Rewriting XML/TOML/JSON/URL/template parser internals in this spec, except for small changes required to keep the String/Bytes semantics tests passing.
- Adding grapheme-cluster semantics. Tya string indexing is by Unicode rune/code point, not by user-perceived grapheme cluster.
- Adding implicit conversion between `String` and `Bytes`.
- Adding lossy UTF-8 decoding or replacement-character behavior.
- Adding slice syntax such as `text[1:3]`.
- Changing numeric, array, dictionary, or error-value indexing semantics except where documentation cross-references string/bytes indexing.
- Changing Flakewatch. Flakewatch receives a separate spec after Tya's Unicode XML behavior is fixed.

## Acceptance Criteria
- `docs/SPEC.md`, `docs/STRICT_SEMANTICS.md`, and `docs/ja/spec.md` consistently describe `String` as Unicode text and `Bytes` as raw bytes.
- Tests prove that `"日報".len()` returns `2`.
- Tests prove that `"日報".byte_len()` returns `6`.
- Tests prove that `"日報"[0]` returns `"日"` and `"日報".slice(0, 1)` returns `"日"`.
- Tests prove that string iteration over `"日報"` yields `"日"` then `"報"` with indexes `0` and `1`.
- Tests prove that `bytes_of("日報").len()` returns `6` and byte indexing returns integer byte values.
- Tests prove that `bytes_text(...)` rejects invalid UTF-8.
- Tests prove that `String + Bytes` and `Bytes + String` remain invalid.
- A repository-wide audit finds no parser/scanner code path that treats `String#[]` as a byte accessor without either converting to `Bytes` or using character/rune indexes consistently.
- Any remaining parser rewrite work is captured in the follow-up parser-scanner spec, not left as an unresolved blocker in this spec.

## Verification
```sh
go test ./... -count=1
tya test tests/stdlib_string_test.tya
tya test tests/bytes_type_test.tya
tya test tests
go test ./tests -run TestSelfhostV01Scripts -count=1
```
