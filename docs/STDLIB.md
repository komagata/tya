# Tya v0.3 Standard Attached Library

This document defines the standard attached library for Tya v0.3.

The standard attached library is a set of `.tya` modules shipped with Tya. It is
not a package manager and it does not download third-party code.

## Importing

Standard modules use the same import syntax as user modules.

```tya
import string
import array
```

The module search order is:

1. The importing file's directory.
1. Directories listed in `TYA_PATH`, searched left to right.
1. The `stdlib/` directory shipped with Tya.

## Initial Scope

v0.3 starts with lightweight modules that prove the attached-library
mechanism.

Included in the initial scope:

- `string`
- `array`

Deferred from v0.3:

- JSON parser
- CSV parser
- native-backed standard modules
- package manager
- remote module install
- versioned dependencies

## `string`

```tya
import string

print string.blank("  ")
print string.present("tya")
```

Functions:

```tya
blank text
present text
```

`blank(text)` returns `true` when `trim(text) == ""`.

`present(text)` returns `not blank(text)`.

## `array`

```tya
import array

print array.empty([])
print array.first(["tya"])
```

Functions:

```tya
empty items
first items
```

`empty(items)` returns `len(items) == 0`.

`first(items)` returns `items[0]`.

## `math`

```tya
import math

print math.abs(-3)
print math.min(2, 5)
print math.max(2, 5)
print math.clamp(12, 0, 10)
```

Functions:

```tya
abs value
min left, right
max left, right
clamp value, min, max
```

`abs(value)` returns the absolute value of an integer or float.

`min(left, right)` returns the smaller number.

`max(left, right)` returns the larger number.

`clamp(value, min, max)` returns `min` when `value < min`, `max` when
`value > max`, and `value` otherwise. It raises an error when `min > max`.

## `path`

```tya
import path

print path.join(["tmp", "tya", "memo.txt"])
print path.clean("tmp/./tya/../memo.txt")
print path.basename("/tmp/tya/memo.txt")
print path.dirname("/tmp/tya/memo.txt")
print path.extname("/tmp/tya/memo.txt")
```

Functions:

```tya
join parts
clean value
basename value
dirname value
extname value
```

`join(parts)` joins an array of path segments with `/` and cleans the result.

`clean(value)` normalizes `.` segments, `..` segments, and repeated `/`
separators lexically.

`basename(value)` returns the final path segment.

`dirname(value)` returns the path without the final segment, or `.` when no
directory segment exists.

`extname(value)` returns the final file extension including the leading `.`,
or `""` when the basename has no extension.

The `path` module uses `/` as the path separator and does not access the file
system.

## `file`

```tya
import file

if file.exists?("memo.txt")
  text = file.read("memo.txt")
  println text

file.write("out.txt", "hello")
```

Functions:

```tya
read path
write path, text
exists? path
```

`read(path)` reads the entire file and returns it as a string.

`write(path, text)` writes `text` to `path` and returns `nil` on success.

`exists?(path)` returns `true` when the path exists, `false` when it does not.

## `os`

```tya
import os

args = os.args()
home = os.env("HOME")
```

Functions:

```tya
args
env name
exit code
```

`args()` returns the command-line arguments as an array of strings.

`env(name)` returns the environment variable value, or `nil` when not present.

`exit(code)` exits the process with `code`.

`cwd()` returns the current working directory as a string.

`chdir(path)` changes the process working directory and raises an error on
failure.

## `dir`

```tya
import dir

names = dir.list(".")
dir.mkdir("tmp")
dir.rmdir("tmp")
```

Functions:

```tya
list path
mkdir path
rmdir path
```

`list(path)` returns an array of names directly under `path` in dictionary
order. `.` and `..` are excluded.

`mkdir(path)` creates one directory level. It raises an error when the
parent directory does not exist or when `path` already exists.

`rmdir(path)` removes an empty directory. It raises an error when the
directory is not empty.

## `file` (additional functions)

```tya
import file

file.remove("memo.txt")
file.rename("a.txt", "b.txt")
info = file.stat("b.txt")
println info["kind"]
println info["size"]
```

`remove(path)` removes a file. It raises an error when `path` is a
directory.

`rename(old_path, new_path)` renames a file or directory.

`stat(path)` returns a dictionary with `kind` (`"file"`, `"dir"`, or
`"other"`), `size` in bytes, and `readable`, `writable`, `executable`
booleans.

## `path` (additional functions)

`expand_user(value)` expands `~` and `~/...` to the current user's home
directory. Other strings are returned unchanged.

## `unittest`

```tya
import unittest
import string

module string_blank_test
  test_blank_for_whitespace = ->
    unittest.assert(string.blank(" "), "spaces")
  test_blank_returns_false_for_content = ->
    unittest.assert_equal(false, string.blank("tya"), "content")
```

A test case is an importable module containing `test_*` functions. Tests are
run by `tya test` (which synthesizes a suite) or by a user-written entry
program calling `unittest.run([cases...])`.

Functions:

```tya
assert cond, desc
assert_falsy cond, desc
assert_equal expected, actual, desc
assert_not_equal left, right, desc
assert_nil value, desc
assert_raises body
fail message
run cases
```

Each assertion raises a structured `{kind: "unittest_fail", message}` value
on failure; the test runner catches it so a single failed test does not stop
the rest of the suite.

`unittest.run(cases)` iterates each module's `test_*` members in dictionary
order, runs `setup` / `teardown` around each test, prints a summary line and
exits non-zero when at least one test failed.

See the v0.22 SPEC for the full surface.

