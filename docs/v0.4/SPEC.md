# Tya v0.4 Specification

This document is the specification for Tya v0.4 after v0.3 standard
attached libraries.

## Theme

Tya v0.4 is about testing and script confidence.

v0.3 makes shared `.tya` modules easier to ship and import. v0.4 makes
those modules, and user scripts built on them, easier to verify.

## Goals

- Add a simple built-in test runner for Tya projects.
- Keep tests as ordinary `.tya` programs.
- Provide minimal assertions without adding a large test framework.
- Make stdlib tests the first real user of the test runner.
- Keep the language core small and compile-to-C.

## Included in v0.4

v0.4 adds:

- `tya test`
- `*_test.tya` discovery
- `assert value`
- `assert_equal expected, actual`
- deep equality in `assert_equal`
- specified test output and exit status behavior
- stdlib tests written as ordinary Tya test files

## Not Included in v0.4

v0.4 does not include:

- `describe` / `it` DSL
- mocking
- coverage
- snapshot testing
- benchmark
- watch mode
- parallel test execution
- package manager
- native-backed stdlib
- JSON parser
- CSV parser

## Test Command

`tya test` runs Tya test files.

```sh
tya test
tya test tests
tya test tests/string_test.tya
```

Discovery rules:

1. With no argument, search the current directory recursively for `*_test.tya`.
1. With a directory argument, search that directory recursively for
   `*_test.tya`.
1. With a file argument, run that file only.

Each test file is a normal Tya program. If any test file fails, `tya test`
exits with a non-zero status.

## Assertions

`assert value` fails when `value` is falsey.

```tya
assert true
assert 1 + 1 == 2
```

`assert_equal expected, actual` compares with deep equality.

```tya
assert_equal 4, add(2, 2)
assert_equal ["a", "b"], names
```

Failure output should be source-oriented and concise.

```text
tests/math_test.tya:3:1: assertion failed
```

For `assert_equal`, include expected and actual values.

```text
tests/math_test.tya:4:1: assert_equal failed
expected: 4
actual: 5
```

## Stdlib Tests

The v0.3 standard attached library is tested through the same runner
that user projects use.

Preferred layout:

```text
tests/
  stdlib_string_test.tya
  stdlib_array_test.tya
```

Example:

```tya
import string

assert string.blank(" ")
assert_equal false, string.blank("tya")
```
