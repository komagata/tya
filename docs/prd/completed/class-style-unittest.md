---
status: completed
goal_ready: false
---

# Feature: Class-Style Unittest Framework

## Goal

Make the `unittest` standard library feel like a conventional JUnit/xUnit
framework by supporting class-based test cases, explicit test suites, a
dedicated test runner, per-test fixture methods, and instance assertion helpers.

## Context

The current `unittest` surface is function/module based:

- `stdlib/unittest/Unittest.tya` exposes static assertion helpers and
  `Unittest.run(cases)`.
- `tya test` synthesizes a program that imports test modules and calls
  `unittest.Unittest.run([module_a, module_b])`.
- Existing stdlib tests define top-level `test_*` functions and call
  `unittest.Unittest.assert_*`.
- `docs/STDLIB.md` documents module cases with optional `setup` / `teardown`.

Tya is not publicly released to external users yet, so this feature can replace
the old API instead of preserving user-facing backward compatibility. Existing
repository tests and templates should migrate to the new API in the same change.

## Behavior

### Public Classes

- Add `TestCase` as the base class for user-authored test cases exported by the
  `unittest` package.
- Add `TestSuite` as an ordered collection of tests and/or nested
  suites.
- Add `TestRunner` as the object responsible for executing suites, printing
  results, and returning/exiting with the final status.
- Add `TestResult` if needed to keep runner state explicit instead of passing
  loose dictionaries around.
- Remove the old `unittest.Unittest` public class/API.

### `TestCase`

- Test authors define subclasses of `TestCase` after `import unittest`.
- Instance methods whose names start with `test_` are test methods.
- Optional fixture hooks run around each test method:
  - `setup` before each test
  - `teardown` after each test
- Assertion helpers are available as instance methods on `TestCase`:
  - `assert`
  - `assert_falsy`
  - `assert_equal`
  - `assert_not_equal`
  - `assert_nil`
  - `assert_raises`
  - `fail`
- A class-style test file should look like:

```tya
import unittest

class StringBlankTest < TestCase
  setup = ->
    self.subject = " "

  test_blank_for_whitespace = ->
    self.assert(self.subject.blank?(), "spaces")

  test_blank_returns_false_for_content = ->
    self.assert_equal(false, "tya".blank?(), "content")
```

### `TestSuite`

- `TestSuite` preserves insertion order.
- `TestSuite.add(test)` accepts:
  - a `TestCase` subclass
  - a `TestCase` instance
  - another `TestSuite`
- `TestSuite.add_all(tests)` appends every item from an array.
- `TestSuite.count()` returns the number of concrete test methods that will run.
- `TestSuite.discover(cases)` discovers `TestCase` subclasses from imported test
  modules.

### `TestRunner`

- `TestRunner` owns output formatting and process status.
- `TestRunner.run(suite)` executes a `TestSuite` and returns a result object or
  dictionary containing at least:
  - `tests`
  - `passes`
  - `failures`
  - `errors`
- `TestRunner.run_and_exit(suite)` runs the suite and exits non-zero when any
  failure or error occurred.
- `TestRunner.default()` returns a runner using the current text output format.
- Test output remains recognizable:
  - each test reports `PASS`, `FAIL`, or `ERROR`
  - failures include the test case class and method name
  - summary remains `<n> tests, <p> passed, <f> failed` unless errors require
    the extended `<n> tests, <p> passed, <f> failed, <e> errors` form

### Discovery and Compatibility

- `tya test` synthesizes a suite and runs it through `TestRunner`.
- User-written suites should call `TestRunner.default().run_and_exit` after
  `import unittest`.
- The old `unittest.Unittest.run([cases])` entry point is removed.
- Module/function-style test discovery is removed from the standard runner.
- Test output remains recognizable:
  - class-style names should use `ClassName.test_method`
- A failing `setup`, test method, or `teardown` is reported as a failed test and
  does not stop the rest of the suite.
- `teardown` runs after a test when `setup` succeeds, even if the test method
  fails.

## Scope

- `stdlib/unittest/Unittest.tya`
- `docs/prd/completed/bare-package-imports.md` landed first in the same
  long-running implementation batch.
- `cmd/tya/main.go` test-suite synthesis
- `cmd/tya/new.go` test templates
- `docs/STDLIB.md`
- release spec/docs for the next version
- script tests under `tests/testdata/`
- existing stdlib test files only if needed to prove compatibility or migrate
  examples

## Out of Scope

- Parallel test execution.
- Parameterized tests.
- Test filtering by name.
- XML/JUnit report output.
- Colorized output changes.
- Assertion aliases such as camelCase `assertEqual`.
- Preserving source compatibility with old `unittest.Unittest.assert_*` or
  `unittest.Unittest.run`.

## Acceptance Criteria

- A `tya test` fixture with a `unittest.TestCase` subclass runs all `test_*`
  instance methods.
- A user-written program can manually build a `unittest.TestSuite`, add two
  `TestCase` classes, and run it with `unittest.TestRunner`.
- Nested suites execute in deterministic insertion order.
- `TestRunner.run` returns counts for tests, passes, failures, and errors
  without exiting.
- `TestRunner.run_and_exit` exits non-zero for failures or errors.
- `setup` and `teardown` run once per test method, not once per class.
- `self.assert_equal` failure reports the class and method that failed.
- A failure in one class-style test method does not stop later methods or later
  test files.
- Existing repository tests and scaffold templates are migrated away from
  `unittest.Unittest.assert_*` and `unittest.Unittest.run`.
- The old `unittest.Unittest` public API is no longer documented.
- `tya new` templates use the class-style unittest pattern.
- `docs/STDLIB.md` documents the class-style API as the only supported unittest
  API.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run TestV.*Script -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```

## Open Questions

None.
