# Tya v0.25 Specification

This document is the specification for Tya v0.25 after v0.24 scripting
toolkit and lightweight numerics.

## Theme

Tya v0.25 adds bit-level integer operations, a byte-sequence value type,
and binary file I/O. Together they unblock workloads that need to read,
manipulate, and write raw bytes — for example, NES emulators, simple image
codecs, byte-stream protocols, and binary parsing.

## Goals

- Add bitwise operators on integers: `&`, `|`, `^`, `~`, `<<`, `>>`.
- Add a `bytes` value type for raw byte sequences (0..255 each).
- Add binary file I/O: `file.read_bytes`, `file.write_bytes`.
- Extend `digest`, `secure_random`, `hex`, and `base64` to accept and return
  byte sequences while remaining compatible with strings.
- Avoid disturbing the existing string type (strings stay UTF-8 text).

## Included in v0.25

v0.25 includes all v0.24 behavior and adds:

- `&`, `|`, `^`, `<<`, `>>` binary operators on integers
- `~` unary bitwise NOT operator on integers
- `bytes` value type
- Byte-sequence literal: `b"..."` (basic-string with `\xHH` byte escapes)
- `bytes(values)` constructor from an array of integers
- `bytes_of(text)` constructor from a string (UTF-8 bytes)
- `bytes_text(b)` decoder back to a string (raises on invalid UTF-8)
- `bytes_array(b)` to get an array of ints
- `bytes_concat(a, b)` and `+` overload for `bytes + bytes`
- `bytes_slice(b, start, end)` for sub-sequences
- `len(b)` for byte length
- `b[i]` indexing returning an int (0..255)
- `file.read_bytes(path)` returning `bytes`
- `file.write_bytes(path, b)` accepting `bytes`
- `digest.*(text_or_bytes)` accepts both
- `secure_random.bytes(n)` returns `bytes` (no longer a string)
- `hex.encode(bytes)`, `hex.decode(text) -> bytes`
- `base64.encode(bytes)`, `base64.decode(text) -> bytes`

## Not Included in v0.25

v0.25 does not include:

- arbitrary-precision integers
- 16-bit / 32-bit / 64-bit fixed-width integer types
- bitwise ops on `bytes` values directly (must index to int first)
- mutable byte buffers (each `bytes` value is immutable; build with
  concatenation)
- bytes pattern matching beyond basic equality
- character-set-aware string ↔ bytes conversion (always UTF-8)
- streaming readers or writers
- `mmap`, file handles, sockets

## Bitwise Operators

The new operators apply to integer operands. Float operands raise an error.

| Operator | Form | Meaning |
|---|---|---|
| `a & b` | binary | bitwise AND |
| `a \| b` | binary | bitwise OR |
| `a ^ b` | binary | bitwise XOR |
| `a << n` | binary | left shift (n ≥ 0) |
| `a >> n` | binary | arithmetic right shift (n ≥ 0) |
| `~a` | unary | bitwise NOT |

### Semantics

- Operands and results are 64-bit signed integers (the same kind as Tya's
  existing `int`).
- `<<` and `>>` raise when the shift count is negative.
- `<<` shift counts ≥ 64 produce 0; `>>` produces 0 or -1 depending on the
  sign of `a` (consistent with two's-complement arithmetic).
- `~a` is equivalent to `-1 ^ a`.

### Precedence

Higher precedence to lower:

```
~  (unary)
* / % << >>
+ -
& 
^
|
< <= > >= == !=
and
or
```

This matches C's relative ordering and resolves expressions like
`a & 0xFF == 0xFF` as `(a & 0xFF) == 0xFF`. (Note that v0.25 introduces no
hexadecimal integer literals — see Out of scope.)

### Examples

```tya
a = 0b11110000      # not part of v0.25; use 240 or to_int("0xF0")
b = 0b00001111      # not part of v0.25; use 15

# 6502 emulator-style bit work
status = 0
status = status | 1                # set carry flag
if (status & 1) != 0
  println "carry set"

byte = 0xFF                        # not part of v0.25 (use 255)
high = (byte >> 4) & 0xF
low  = byte & 0xF
```

(Hexadecimal and binary literals are not added in v0.25; use decimal or
`to_int(string, base)`-style helpers in stdlib if needed in user code.)

## `bytes` Value Type

`bytes` is a new top-level value type alongside `string`, `int`, `float`,
`bool`, `nil`, `array`, `dict`, etc.

### Properties

- Immutable.
- Each element is an integer in `0..255`.
- `len(b)` returns the byte count.
- `b[i]` returns the integer byte at index `i` (raises on out-of-range).
- `b1 == b2` is true when the byte sequences match.
- `b1 + b2` concatenates.
- `kind(b)` returns `"bytes"`.

### Literal

```tya
b1 = b""
b2 = b"hello"
b3 = b"\x01\x02\xff"
b4 = b"a\nb\tc"            # \n, \t, \r, \\, \", \xHH supported
```

The `b"..."` literal accepts the same backslash escapes as a basic string,
plus `\xHH` two-digit hexadecimal byte escapes. The literal contents are
parsed byte-by-byte; non-ASCII bytes from source UTF-8 are stored verbatim.

### Conversions

```tya
bytes_of("hello")            # bytes from a string (UTF-8 bytes)
bytes_text(b3)               # string from bytes (UTF-8); raises on invalid
bytes_array(b3)              # array of ints
bytes([72, 101, 108, 108])   # bytes from an array of 0..255 ints
```

### Slicing

```tya
b = b"hello world"
bytes_slice(b, 0, 5)         # b"hello"
bytes_slice(b, 6, len(b))    # b"world"
```

`bytes_slice(b, start, end)` returns the sub-sequence `[start, end)` in
half-open form. Out-of-range indices raise.

### Concatenation

```tya
b"foo" + b"bar"              # b"foobar"
bytes_concat(b"a", b"b")     # equivalent function form
```

`+` between `string` and `bytes` is an error in v0.25; convert explicitly.

### Iteration

```tya
for byte in bytes_array(b)
  println byte
```

The runtime does not yet support `for x in bytes_value` directly; iterate
via `bytes_array(b)` or via index in v0.25.

## Binary File I/O

`file.read_bytes(path)` reads a file and returns a `bytes` value.

`file.write_bytes(path, b)` writes a `bytes` value to a file.

```tya
import file
import digest

raw = file.read_bytes("/usr/local/bin/something")
println len(raw)
println digest.sha256(raw)
file.write_bytes("/tmp/copy", raw)
```

The existing `file.read(path)` (string) and `file.write(path, text)` are
unchanged. They remain UTF-8 text-oriented.

## Updated stdlib API

Affected modules accept and return `bytes` where it makes sense, while
preserving string compatibility.

### `digest`

`digest.md5`, `digest.sha1`, `digest.sha256`, `digest.sha384`,
`digest.sha512` accept either `string` or `bytes` input. Output remains a
lowercase hex string.

### `secure_random`

`secure_random.bytes(n)` now returns a `bytes` value (was a NUL-bearing
string in v0.24). `secure_random.hex(n)` and `secure_random.base64(n)` are
unchanged.

### `hex`

`hex.encode(value)` accepts `bytes` (preferred) or `string` (treated as
UTF-8 bytes). `hex.decode(text)` returns `bytes`.

### `base64`

`base64.encode(value)` accepts `bytes` (preferred) or `string`.
`base64.decode(text)` returns `bytes`.

### Backward compatibility

- Existing user code that passes `string` to `digest.*`, `hex.encode`,
  `base64.encode` continues to work.
- `hex.decode("...")` and `base64.decode("...")` previously returned
  `string`; in v0.25 they return `bytes`. **This is a breaking change.**
  Migrate by wrapping with `bytes_text(...)` to recover a string when text
  data was assumed.

## Diagnostics

v0.25 implementations should report source-oriented errors for:

- bitwise ops applied to non-integer operands
- negative shift counts
- `b"\xHH"` with malformed hex escape
- out-of-range `b[i]` indexing
- out-of-range `bytes_slice` arguments
- `bytes(...)` with non-integer or out-of-range elements (must be 0..255)
- `bytes_text(...)` on invalid UTF-8 input
- `+` between `string` and `bytes` (mixed-type concatenation)

Diagnostics should mention the operator, function name, expected operand
type, and actual value kind when available.

## Implementation Notes (non-normative)

- C runtime adds a new `TYA_BYTES` value kind with `data: const uint8_t *`
  and `len: int` fields (separate from `string`'s NUL-terminated `char *`).
- `tya_index` returns an int when the target is `bytes`.
- `tya_add` is overloaded to handle `bytes + bytes`.
- C codegen for `&`, `|`, `^`, `<<`, `>>`, `~` emits direct C bitwise
  operators on `(long)x.number`, then wraps the result with `tya_number`.
- Lexer adds `&`, `|`, `^`, `~`, `<<`, `>>` tokens (already mostly absent;
  ensure they don't conflict with existing tokens).
- Parser adds operator precedence levels.

These notes are guidance, not the spec; conforming implementations may
differ as long as the user-visible behavior matches.
