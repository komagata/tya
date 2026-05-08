# Tya v0.24 Specification

This document is the specification for Tya v0.24 after v0.23 data-format
and utility stdlib expansion.

## Theme

Tya v0.24 expands the standard library with scripting essentials:
time/date, random, expanded math, external process execution, hex encoding,
cryptographic digests, secure random, and a small linear-algebra matrix
module.

The v0.24 stdlib makes Tya practical as a shell-script replacement: scripts
can read the clock, sleep, run subprocesses, hash files for caching, and
generate cryptographically secure tokens.

## Goals

- Add `time`, `random`, and expanded `math` for everyday scripting.
- Add `process` to run external commands.
- Add `hex` byte-to-text encoding.
- Add `digest` for non-cryptographic-grade-ID-friendly hashing
  (MD5/SHA1/SHA256/SHA384/SHA512).
- Add `secure_random` for cryptographically secure tokens.
- Add `matrix` for basic 2-D linear algebra in pure Tya.
- Keep all native-backed APIs import-only and explicit.
- Keep failure handling consistent with v0.21+ (raise structured errors).

## Included in v0.24

v0.24 includes all v0.23 behavior and adds:

- `time` standard module
- `random` standard module
- `math` expansion (`sqrt`, `pow`, `floor`, `ceil`, `round`, `trunc`, `log`,
  `log2`, `log10`, `exp`, `sin`, `cos`, `tan`, `asin`, `acos`, `atan`,
  `atan2`, plus `pi` and `e` constants)
- `process` standard module
- `hex` standard module
- `digest` standard module
- `secure_random` standard module
- `matrix` standard module

## Not Included in v0.24

v0.24 does not include:

- a byte-array value type
- streaming digest (incremental `update`/`final`)
- HTTP, TCP, UDP, TLS modules
- regex
- yaml, xml, markdown
- timezone-aware date arithmetic beyond UTC and local
- async / threads / coroutines
- subprocess pipes between two children
- shell-string parsing (`process.run` requires an array, never a string)
- matrix inverse, eigenvalues, decomposition beyond determinant
- Mersenne Twister exposure beyond the seed/next-value abstraction

## Importing

All v0.24 modules are standard attached-library modules.

```tya
import time
import random
import math
import process
import hex
import digest
import secure_random
import matrix
```

## `time`

The `time` module reads the system clock and provides simple formatting.

### Functions

- `time.now()` → float (seconds since UNIX epoch, sub-second precision)
- `time.sleep(seconds)` → nil (accepts int or float)
- `time.format(t)` → string (RFC 3339 UTC, e.g. `"2026-05-09T12:34:56Z"`)
- `time.format(t, layout)` → string (layout `"iso"` or `"date"` or `"time"`
  or `"unix"`)
- `time.parse(text)` → float (parses `"YYYY-MM-DDTHH:MM:SSZ"` and `"YYYY-MM-DD"`)
- `time.since(t)` → float (seconds elapsed since `t`)

### Example

```tya
import time

start = time.now()
time.sleep(0.5)
println time.since(start)
println time.format(time.now())
```

### Errors

`time.parse` raises a structured error on invalid input.

`time.sleep` raises when the argument is negative or non-numeric.

`time` is UTC-only in v0.24 except where the system clock provides local time
(`time.now()` returns wall clock seconds without timezone information).

## `random`

The `random` module provides a seedable pseudo-random number generator
(PRNG). It is **not cryptographically secure**. Use `secure_random` for
tokens, IDs, salts, and similar.

### Functions

- `random.seed(value)` → nil (deterministic reseeding)
- `random.int(min, max)` → int (inclusive on both ends)
- `random.float()` → float (in [0.0, 1.0))
- `random.choice(items)` → element (raises on empty array)
- `random.shuffle(items)` → nil (mutates the array in place)

### Example

```tya
import random

random.seed(42)
println random.int(1, 6)
println random.float()
println random.choice(["red", "green", "blue"])
```

`random.seed(value)` accepts an int or a string. With no `seed` call, the
generator is seeded from the system clock at first use; results are not
reproducible.

## `math` Expansion

v0.24 expands `math` (already present from v0.20) with elementary numeric
functions and two constants.

### New functions

- `math.sqrt(x)`
- `math.pow(x, y)`
- `math.floor(x)`
- `math.ceil(x)`
- `math.round(x)` (banker-style rounding off; rounds half away from zero)
- `math.trunc(x)`
- `math.log(x)` (natural log)
- `math.log2(x)`
- `math.log10(x)`
- `math.exp(x)`
- `math.sin(x)`, `math.cos(x)`, `math.tan(x)`
- `math.asin(x)`, `math.acos(x)`, `math.atan(x)`, `math.atan2(y, x)`

### New constants

- `math.pi`
- `math.e`

Constants are values, not functions: `math.pi`, not `math.pi()`.

`math.abs`, `math.min`, `math.max`, `math.clamp` from v0.20 stay unchanged.

### Errors

`math.sqrt` of a negative number raises. `math.log` / `math.log2` /
`math.log10` of a non-positive number raises. Other domain errors follow C
math behavior (NaN propagates).

## `process`

The `process` module runs external commands.

### Functions

- `process.run(command)` → result dict
- `process.run(command, options)` → result dict

`command` is an array of strings: the program name (or path) followed by
arguments. Passing a single string is **not** supported; this prevents
shell-string injection.

### Result dict

```tya
{
  exit_code: 0,
  stdout: "the captured stdout text",
  stderr: "the captured stderr text",
}
```

`stdout` and `stderr` are strings. Output is buffered fully into memory.

### Options dict

All options are optional:

- `cwd`: working directory for the child process (string)
- `env`: environment variables (dict of string → string). When set, the child
  receives only this environment (replaces, does not merge).
- `input`: text to write to the child's stdin (string)

### Example

```tya
import process

result = process.run(["git", "rev-parse", "HEAD"])
println result["stdout"]

process.run(["sh", "-c", "echo $X"], { env: { X: "tya" } })
```

### Errors

`process.run` raises when the program cannot be launched (e.g. file not
found, permission denied). A non-zero exit code does **not** raise; it is
reported in `exit_code`.

## `hex`

The `hex` module encodes and decodes hexadecimal text.

### Functions

- `hex.encode(text)` → string (lowercase hex)
- `hex.decode(text)` → string

### Example

```tya
import hex

println hex.encode("Tya")     # 547961
println hex.decode("547961")  # Tya
```

`hex.decode` accepts uppercase or lowercase. Whitespace is not allowed in
the input. Odd-length input raises a structured error.

## `digest`

The `digest` module computes cryptographic message digests.

### Functions

- `digest.md5(text)` → string (lowercase hex)
- `digest.sha1(text)` → string
- `digest.sha256(text)` → string
- `digest.sha384(text)` → string
- `digest.sha512(text)` → string

### Example

```tya
import digest

println digest.sha256("hello")
# 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
```

For files, combine with `file.read`:

```tya
import digest, file

println digest.sha256(file.read("config.toml"))
```

### Notes

- Each function takes a single text (UTF-8 bytes are hashed) and returns the
  hex-encoded digest.
- Output is always lowercase hex.
- Incremental (`update` / `final`) digesting is not in v0.24.
- MD5 and SHA1 are included for compatibility but should not be used for
  new security work.

## `secure_random`

The `secure_random` module provides cryptographically secure random data.
It uses the operating system's CSPRNG (`/dev/urandom`, `getrandom`,
`getentropy`).

### Functions

- `secure_random.bytes(n)` → string (n raw bytes; may include NUL)
- `secure_random.hex(n)` → string (2n lowercase hex characters)
- `secure_random.base64(n)` → string (Base64 of n random bytes)
- `secure_random.uuid()` → string (RFC 4122 v4 UUID, lowercase)
- `secure_random.int(min, max)` → int (inclusive, uniform)

### Example

```tya
import secure_random

token = secure_random.hex(16)           # 32 chars
session = secure_random.uuid()          # 550e8400-e29b-41d4-a716-446655440000
salt = secure_random.bytes(16)
println secure_random.int(0, 100)
```

### Notes

- `secure_random` is not seedable. The OS entropy source is the only input.
- `secure_random.bytes(n)` returns a Tya string. Because Tya strings are NUL-
  terminated at the C level, callers should generally consume the bytes via
  `secure_random.hex` or `secure_random.base64` instead of `bytes` if NUL
  bytes might appear.
- `secure_random.int` is uniform; rejection sampling is used internally to
  avoid modulo bias.

## `matrix`

The `matrix` module provides basic 2-D matrix operations on numeric values.
It is implemented in pure Tya.

### Construction

- `matrix.new(rows)` — `rows` is an array of equal-length arrays of numbers.
- `matrix.zero(rows, cols)` — returns a `rows × cols` zero matrix.
- `matrix.identity(n)` — returns the `n × n` identity matrix.

A matrix is represented as a dict:

```tya
{ rows: 2, cols: 3, data: [[1, 2, 3], [4, 5, 6]] }
```

### Element access

- `matrix.at(m, r, c)` → element at row `r`, column `c`.
- `matrix.set(m, r, c, value)` → sets the element in place; returns nil.

### Operations

- `matrix.add(a, b)` — element-wise; both matrices must share dimensions.
- `matrix.sub(a, b)` — element-wise.
- `matrix.scale(a, k)` — scalar multiplication.
- `matrix.mul(a, b)` — matrix multiplication; `a.cols` must equal `b.rows`.
- `matrix.transpose(a)` — returns the transpose.
- `matrix.det(a)` — determinant for square matrices up to size `4 × 4`.
  Larger matrices raise an error in v0.24.
- `matrix.equal?(a, b)` — element-wise equality.

### Example

```tya
import matrix

a = matrix.new([[1, 2], [3, 4]])
b = matrix.identity(2)

println matrix.add(a, b)["data"]      # [[2, 2], [3, 5]]
println matrix.mul(a, a)["data"]      # [[7, 10], [15, 22]]
println matrix.transpose(a)["data"]   # [[1, 3], [2, 4]]
println matrix.det(a)                 # -2
println matrix.at(a, 0, 1)            # 2
```

### Errors

- `matrix.new` raises when rows have inconsistent lengths or non-numeric
  elements.
- `matrix.add`, `matrix.sub`, `matrix.equal?` raise on dimension mismatch.
- `matrix.mul` raises when inner dimensions disagree.
- `matrix.det` raises on non-square matrices and on sizes > 4.
- `matrix.at` and `matrix.set` raise on out-of-range indices.

### Out of scope

Inverse, eigenvalue decomposition, LU/QR decomposition, sparse storage,
broadcasting, vector convenience functions, complex numbers, GPU offload,
all out of v0.24.

## Diagnostics

v0.24 implementations should report source-oriented errors for:

- missing imports for any of the v0.24 modules
- unknown module functions
- wrong argument counts
- wrong argument kinds
- domain errors in `math` (negative `sqrt`, non-positive `log`)
- failed `process.run` (program launch failure)
- malformed input to `hex.decode`
- wrong array shape to `matrix.new`
- dimension mismatches in `matrix` ops
- invalid range in `random.int` / `secure_random.int`
- empty array passed to `random.choice`

Diagnostics should mention the module name, function name, expected
argument shape, and actual value kind when available.
