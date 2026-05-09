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

## `base64`

```tya
import base64

println base64.encode("hello")
println base64.decode("aGVsbG8=")
```

`encode(text)` returns the standard-alphabet Base64 representation with `=`
padding. `decode(text)` decodes standard Base64 (padding optional, whitespace
ignored).

## `url`

```tya
import url

println url.encode("hello world")
parts = url.parse("https://example.com:8080/path?x=1")
```

Functions: `encode`, `decode`, `encode_query`, `decode_query`, `parse`,
`build`. See the v0.23 spec for full details.

## `json`

```tya
import json

println json.dump({ name: "tya", version: 23 })
data = json.parse("[1, 2, 3]")
```

`parse(text)` parses RFC 8259 JSON; `dump(value)` emits compact JSON.
Numbers without fractional parts decode to ints; with fractional parts to
floats.

## `csv`

```tya
import csv

rows = csv.parse("name,age\ntya,1\n", { header: true })
println csv.dump([["a", "b"], ["1", "2"]], nil)
```

`parse(text, options)` accepts `{ separator, header }` options. `dump(rows,
options)` quotes fields containing the separator, quote, CR, or LF.

## `toml`

```tya
import toml

config = toml.parse("[server]\nhost = \"localhost\"\nport = 80\n")
println toml.dump(config)
```

`parse(text)` parses TOML 1.0 documents. `dump(value)` emits TOML for a dict
of primitives, arrays, and nested tables (including arrays of tables).

## `time`

```tya
import time

now = time.now()
println time.format(now, "iso")
time.sleep(0.1)
println time.since(now)
```

Functions: `now`, `sleep`, `format`, `parse`, `since`. Format layouts:
`"iso"`, `"date"`, `"time"`, `"unix"`.

## `random`

```tya
import random

random.seed(42)
println random.int(1, 100)
println random.float()
println random.choice(["a", "b", "c"])
```

Seedable PRNG. **Not** cryptographically secure — use `secure_random` for
tokens.

## `math` (additional functions)

`sqrt`, `pow`, `floor`, `ceil`, `round`, `trunc`, `log`, `log2`, `log10`,
`exp`, `sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `atan2`, plus the
constants `pi` and `e`.

## `process`

```tya
import process

result = process.run(["echo", "hello"], nil)
println result["stdout"]
println result["exit_code"]

process.run(["sh", "-c", "echo $X"], { env: { X: "tya" } })
```

`run(command, options)` returns `{exit_code, stdout, stderr}`. `command` is
an array of strings — never a shell string. Options: `cwd`, `env`, `input`.

## `hex`

```tya
import hex

println hex.encode("Tya")     # 547961
println hex.decode("547961")  # Tya
```

## `digest`

```tya
import digest

println digest.md5("hello")
println digest.sha256("hello")
```

Functions: `md5`, `sha1`, `sha256`, `sha384`, `sha512`. Each takes a text
and returns a lowercase hex digest string.

## `secure_random`

```tya
import secure_random

println secure_random.hex(16)     # 32 hex chars
println secure_random.uuid()      # RFC 4122 v4
println secure_random.int(0, 99)
```

Cryptographically secure. Functions: `bytes`, `hex`, `base64`, `uuid`,
`int`.

## `matrix`

```tya
import matrix

a = matrix.new([[1, 2], [3, 4]])
b = matrix.identity(2)

println matrix.add(a, b)["data"]
println matrix.mul(a, a)["data"]
println matrix.det(a)
```

Functions: `new`, `zero`, `identity`, `at`, `put`, `add`, `sub`, `scale`,
`mul`, `transpose`, `det` (≤ 4×4), `equal?`.

## `bytes` (v0.25)

`bytes` is a built-in value type for raw byte sequences (each element is an
int 0..255). Construct with the `b"..."` literal or builtins:

```tya
b1 = b"hello"
b2 = b"\x00\x01\xff"
b3 = bytes([72, 101, 108, 108, 111])

println len(b1)              # 5
println b1[0]                # 104
println bytes_text(b1)       # hello
println bytes_array(b1)      # [104, 101, 108, 108, 111]
println bytes_slice(b1, 0, 3)
println b1 + b2              # concat
```

Builtins: `bytes(int_array)`, `bytes_of(text)`, `bytes_text(b)`,
`bytes_array(b)`, `bytes_concat(a, b)`, `bytes_slice(b, start, end)`.

## `file` (v0.25 binary I/O)

```tya
import file

raw = file.read_bytes("/path/to/file")
file.write_bytes("/tmp/copy", raw)
```

`read_bytes(path)` returns a `bytes` value. `write_bytes(path, b)` writes
raw bytes.

## Updated for bytes (v0.25)

- `digest.md5/sha1/sha256/sha384/sha512` accept either `string` or `bytes`.
- `secure_random.bytes(n)` returns `bytes`.
- `hex.encode(value)` accepts `string` or `bytes`. `hex.decode(text)` returns
  `bytes` (was `string` in v0.24 — breaking change).
- `base64.encode(value)` accepts `string` or `bytes`. `base64.decode(text)`
  returns `bytes` (was `string` in v0.24 — breaking change).

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


## `runtime` (v0.41)

GC introspection and explicit collection.

```tya
import runtime

stats = runtime.gc_stats()
runtime.gc()
```

Functions:

```tya
gc_stats ()
gc       ()
```

`runtime.gc_stats()` returns a dict snapshot of the GC counters with
keys `alloc_count`, `alloc_bytes`, `freed_count`, `freed_bytes`,
`live_count`, `live_bytes`, `collect_count`, `threshold`.

`runtime.gc()` runs a full mark-and-sweep collection. The collector
treats module-level globals as roots (plus the active raise-frame
chain). Locals inside user functions are not roots in v0.41, so
`runtime.gc()` is safe to call only at points where every live local
TyaValue is also reachable from a registered root — in practice, at
the top level of the program. See the v0.41 SPEC for the full safety
contract and known limitations.
