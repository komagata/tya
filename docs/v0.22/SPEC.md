# Tya v0.22 Specification

This document is the specification for Tya v0.22 after v0.21 native-backed
`file` and `os` standard modules.

## Theme

Tya v0.22 adds a standard unit testing facility.

v0.22 introduces the `unittest` standard module, makes module values
reflectable, and turns `tya test` into a test runner that synthesizes a
suite from its argument.

## Goals

- Add a `unittest` standard module.
- Make module values reflectable through `keys` and string-key indexing.
- Turn `tya test` into a test runner that synthesizes a suite from a file
  or directory argument.
- Keep the testing surface small and explicit.

## Included in v0.22

v0.22 includes all v0.21 behavior and adds:

- `unittest` standard module
- module value reflection through `keys(module)` and `module[name]`
- `tya test` test-runner behavior described below

## Not Included in v0.22

v0.22 does not include:

- source line numbers in `unittest` failure output
- focus, skip, or pending test markers
- parameterized tests
- parallel test execution
- color output for `unittest`
- test name filters in `tya test`
- filesystem stdlib expansion (deferred)

## Module Reflection

v0.22 makes module values reflectable so the `unittest` standard module can
discover test functions at runtime.

`keys(module)` returns the names of all members declared in the module as an
array of strings.

`module[name]` returns the member named `name`, or `nil` when no such member
exists.

```tya
import string

names = keys(string)
fn = string["len"]
println fn("tya")
```

Both forms have already been valid for dictionaries. v0.22 extends them to
module values without changing existing dictionary behavior.

`module[name]` reads only. Assignment such as `module[name] = value` is not
supported.

## `unittest`

The `unittest` standard module provides a lightweight unit testing facility.
It is a pure Tya module and does not introduce any new global built-in.

### Concepts

- A **test** is a top-level function in a test case module whose name begins
  with `test_`.
- A **test case** is one importable Tya file containing a single module
  declaration whose members are tests, plus optional `setup` and `teardown`
  functions.
- A **test suite** is a set of test cases run together. A suite is normally
  produced by `tya test`, but a user may also write a suite as a regular entry
  program that imports test cases and calls `unittest.run`.

### Test case shape

A test case file is a normal importable Tya module: it consists of imports
and exactly one module declaration.

```tya
import unittest
import string

module string_blank_test
  setup = ->
    nil

  teardown = ->
    nil

  test_returns_true_for_whitespace = ->
    unittest.assert(string.blank("   "))

  test_returns_false_for_content = ->
    unittest.assert_equal(false, string.blank("tya"))
```

A test case file declares no top-level statements other than the imports and
the module declaration.

`setup` and `teardown` are optional. When present, `setup` runs before each
test in the case and `teardown` runs after each test in the case.

Member names that begin with `test_` are tests. Other member names are not
treated as tests by the runner.

### Test suite shape (manual)

A test suite is an entry program that imports test case modules and calls
`unittest.run` with the modules.

```tya
import unittest
import string_blank_test
import string_present_test

unittest.run([string_blank_test, string_present_test])
```

This form is always available. It is useful when a project needs a custom
suite, for example to run only a subset of cases.

### Functions

- `unittest.assert(cond, desc)`
- `unittest.assert_falsy(cond, desc)`
- `unittest.assert_equal(expected, actual, desc)`
- `unittest.assert_not_equal(left, right, desc)`
- `unittest.assert_nil(value, desc)`
- `unittest.assert_raises(body)`
- `unittest.fail(message)`
- `unittest.run(cases)`

`desc` is a short string describing the assertion. It appears in failure
output. An empty string is allowed.

`unittest.assert(cond, desc)` raises a unittest failure when `cond` is falsy.

`unittest.assert_falsy(cond, desc)` raises a unittest failure when `cond` is
truthy.

`unittest.assert_equal(expected, actual, desc)` raises a unittest failure
when `expected` and `actual` are not deeply equal. Deep equality compares
arrays and dictionaries by structure.

`unittest.assert_not_equal(left, right, desc)` raises a unittest failure when
`left` and `right` are deeply equal.

`unittest.assert_nil(value, desc)` raises a unittest failure when `value` is
not `nil`.

`unittest.assert_raises(body)` calls `body()` and raises a unittest failure
when the call did not raise. Any value raised by `body()` is treated as a
success.

`unittest.fail(message)` raises a unittest failure with `message`.

`unittest.run(cases)` runs the given test case modules and prints a summary.
For each module:

1. Names of `test_` members are collected.
2. Test names are sorted in dictionary order.
3. For each test, `setup` runs, the test runs, then `teardown` runs.
4. Test failures are caught and counted.

After all cases have run, `unittest.run` prints a summary line of the form
`<n> tests, <p> passed, <f> failed` and calls `exit(1)` when at least one
test failed.

`cases` is an array of module values. Passing values that are not modules is
an error.

### Failure semantics

`unittest` assertion failures are raised as structured errors. The runner
catches these errors so that one failed test does not stop the rest of the
case or suite.

Errors raised by the test body that are not unittest failures are also
caught. They are reported as test failures with their value formatted as the
failure message.

Errors raised outside any test (for example during `setup`) propagate
normally and abort the suite.

### Output

```
  PASS  test_returns_true_for_whitespace
  FAIL  test_returns_false_for_content
        : expected false, got true
  PASS  test_handles_empty_string
  PASS  test_returns_true_for_content
  PASS  test_returns_false_for_whitespace
5 tests, 4 passed, 1 failed
```

The output format is plain text. Color output is not part of v0.22.

## `tya test` Command

v0.22 changes `tya test` to act as a test runner that synthesizes a suite
from its argument.

### Forms

```sh
tya test                       # all *_test.tya under cwd
tya test path/to/dir            # all *_test.tya under dir, recursively
tya test path/to/foo_test.tya   # one test case file
```

When the argument is a directory or omitted, `tya test` discovers files whose
basename ends in `_test.tya` recursively under the chosen root.

When the argument is a file, `tya test` runs that one file as a single-case
suite.

### Behavior

For the discovered set of test case files, `tya test` synthesizes a small
entry program of the form:

```tya
import unittest
import case_a_test
import case_b_test

unittest.run([case_a_test, case_b_test])
```

The synthesized program is compiled and executed in the same way as
`tya run`. Module imports are resolved using each case's directory, so test
case files import their dependencies as usual.

The `tya test` exit code is the exit code of the synthesized program. A
non-zero exit code indicates that at least one test failed.

`tya test` does not change the working directory of the user shell.

### Test case file requirements

A file processed by `tya test` must be an importable Tya module. The file
must contain exactly one module declaration and no other top-level statements
other than imports.

Files that do not satisfy this shape are reported as test discovery errors.

### Suite-only execution

`tya test` always runs through `unittest.run`. The legacy bare-assertion
form, where a `*_test.tya` file contained top-level `assert` statements, is
no longer supported by `tya test` in v0.22. Bare-assertion test programs may
still be executed individually with `tya run`, but they are not part of the
standard test runner surface.

## Diagnostics

v0.22 implementations should report source-oriented errors for:

- non-importable test case files passed to `tya test`
- duplicate test case module names within a suite synthesized by `tya test`
- invalid arguments to `unittest.run` (non-array, or array containing a
  non-module value)

Diagnostics should mention the offending file path, module name, or value
kind when available.
