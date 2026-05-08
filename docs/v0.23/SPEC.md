# Tya v0.23 Specification

This document is the specification for Tya v0.23 after v0.22 unit testing.

## Theme

Tya v0.23 expands the standard library with data-format and small utility
modules, completes the deferred filesystem stdlib expansion, and adds three
character-level primitives that enable data-format work in Tya.

v0.23 introduces five pure-Tya modules — `toml`, `json`, `csv`, `base64`,
and `url` — covering human-edited configuration, machine data interchange,
tabular data, byte-to-text encoding, and URL handling. v0.23 also ships the
filesystem expansion that was deferred from v0.22 (`dir`, `file.remove` /
`file.rename` / `file.stat`, `path.expand_user`, `os.cwd` / `os.chdir`).

Three small global built-ins are added to enable pure-Tya data-format
implementations: `ord`, `chr`, and `kind`. These are character-level and
type-introspection primitives, not data-format functions; they unblock
parser and emitter work in Tya itself.

## Goals

- Add `toml`, `json`, `csv`, `base64`, and `url` standard modules.
- Keep all five implementations pure Tya.
- Use the same `parse` / `dump` (or `encode` / `decode`) shape across modules.
- Avoid schema validation, type coercion beyond each format's standard, and
  partial parsing.
- Add `ord`, `chr`, `kind` global built-ins as character-level primitives
  needed by pure-Tya data-format code.
- Complete the filesystem stdlib expansion deferred from v0.22.

## Included in v0.23

v0.23 includes all v0.22 behavior and adds:

- `toml` standard module (`toml.parse`, `toml.dump`)
- `json` standard module (`json.parse`, `json.dump`)
- `csv` standard module (`csv.parse`, `csv.dump`)
- `base64` standard module (`base64.encode`, `base64.decode`)
- `url` standard module (`url.encode`, `url.decode`, `url.parse`, `url.build`,
  `url.encode_query`, `url.decode_query`)
- `dir` standard module (`dir.list`, `dir.mkdir`, `dir.rmdir`)
- `file.remove`, `file.rename`, `file.stat`
- `path.expand_user`
- `os.cwd`, `os.chdir`
- `ord(char)`, `chr(byte)`, `kind(value)` global built-ins
- short-circuiting evaluation of `and` and `or`

## Not Included in v0.23

v0.23 does not include:

- schema validation
- streaming parsers
- preserving comments through round-trip
- preserving original formatting through round-trip
- TOML 1.1+ extensions
- JSON5, JSONC, or other JSON dialects
- CSV dialects beyond RFC 4180 (TSV is allowed via separator argument)
- url-safe base64 variants beyond what is described below
- IDN (internationalized domain name) handling
- HTTP, YAML, regex, random, time, or crypto modules
- TOML multi-line strings (basic and literal)
- TOML hexadecimal, octal, and binary integer literals
- TOML date and time scalar value types (returned as strings only)
- dotted-key TOML assignments (`a.b.c = 1`)

## Importing

All five modules are standard attached-library modules. They are not
available without import.

```tya
import toml
import json
import csv
import base64
import url
```

The standard module search behavior from v0.17 applies.

## `toml`

The `toml` module reads and writes TOML 1.0 documents.

### Functions

- `toml.parse(text)`
- `toml.dump(value)`

### Data model

| TOML kind | Tya kind |
|---|---|
| string | string |
| integer | int |
| float | float |
| boolean | bool |
| array | array |
| inline table | dict |
| table | dict |
| array of tables | array of dicts |
| offset date-time | string (as written, RFC 3339) |
| local date-time | string (as written) |
| local date | string (as written) |
| local time | string (as written) |

The top-level result of `toml.parse(text)` is always a dict.

### Example

```tya
import toml

text = "
title = \"Example\"
[server]
host = \"127.0.0.1\"
port = 8080
"

config = toml.parse(text)
println config["title"]
println config["server"]["port"]

println toml.dump(config)
```

### Subset

v0.23 supports the following TOML 1.0 features: comments, bare and quoted
keys, dotted keys, basic and multi-line basic strings, literal and multi-line
literal strings, integers (decimal/hex/oct/bin), floats including `inf` and
`nan`, booleans, homogeneous and heterogeneous arrays, inline tables,
`[table]` headers, `[[array.of.tables]]` headers, and date/time scalars
(returned as strings).

### Errors

`toml.parse(text)` raises a structured error on syntax errors with a line
number.

`toml.dump(value)` requires a dict at the top level. It raises a structured
error when the value contains `nil`, a function, class, object, module, or a
dict with a non-string key.

## `json`

The `json` module reads and writes JSON documents (RFC 8259).

### Functions

- `json.parse(text)`
- `json.dump(value)`
- `json.dump(value, indent)`

### Data model

| JSON kind | Tya kind |
|---|---|
| object | dict |
| array | array |
| string | string |
| number (integer) | int |
| number (fractional or exponent) | float |
| `true` / `false` | bool |
| `null` | nil |

### Example

```tya
import json

text = "{\"name\": \"tya\", \"versions\": [0.22, 0.23]}"
data = json.parse(text)
println data["name"]
println data["versions"][1]

println json.dump(data)
println json.dump(data, 2)
```

### Behavior

`json.parse(text)` parses a complete JSON document. The top-level value may
be any JSON value (object, array, string, number, boolean, or null).

`json.dump(value)` emits compact JSON (no insignificant whitespace).

`json.dump(value, indent)` emits pretty-printed JSON with `indent` spaces of
indentation per nesting level. `indent` must be a non-negative integer.

### Errors

`json.parse(text)` raises a structured error on syntax errors with a line
number and a short message.

`json.dump(value)` raises a structured error when the value contains a
function, class, object, module, a dict with a non-string key, or a float
that is `nan`, `inf`, or `-inf` (JSON does not represent these).

## `csv`

The `csv` module reads and writes CSV documents (RFC 4180).

### Functions

- `csv.parse(text)`
- `csv.parse(text, options)`
- `csv.dump(rows)`
- `csv.dump(rows, options)`

### Data model

CSV is row-oriented. By default `csv.parse(text)` returns an array of arrays
of strings, one inner array per row. With the `header: true` option, it
returns an array of dicts using the first row as keys.

`csv.dump(rows)` accepts:

- an array of arrays of strings, or
- an array of dicts (in which case all dicts must share the same set of keys
  and their union determines the header row)

Every CSV value is emitted and parsed as a string. Numeric or boolean
inference is not performed in v0.23.

### Options

`options` is a dict with the following keys; all are optional:

- `separator`: the field separator character (default `","`). `"\t"` enables
  TSV.
- `header`: when `true`, treat the first row as a header on parse; when
  `true` on dump, emit the dict keys as the first row.

### Example

```tya
import csv

text = "name,age\ntya,1\nzig,12\n"
rows = csv.parse(text, { header: true })
for row in rows
  println "{row[\"name\"]}: {row[\"age\"]}"

raw = [["a", "b"], ["1", "2"]]
println csv.dump(raw)
```

### Behavior

`csv.parse(text)` accepts CRLF or LF line endings.

`csv.dump(rows)` emits LF line endings and quotes fields that contain the
separator, double-quote, CR, or LF. Double-quotes inside quoted fields are
doubled per RFC 4180.

### Errors

`csv.parse(text)` raises a structured error on unterminated quoted fields or
malformed escapes.

`csv.dump(rows)` raises a structured error when rows are not a uniform
structure (mixed array-of-arrays and array-of-dicts), when a row contains
non-string values, or when dict rows have inconsistent keys.

## `base64`

The `base64` module encodes and decodes Base64 (RFC 4648).

### Functions

- `base64.encode(text)`
- `base64.decode(text)`

### Behavior

`base64.encode(text)` encodes a Tya string (UTF-8 bytes) and returns the
Base64 representation as a string. Output uses the standard alphabet with
`=` padding.

`base64.decode(text)` decodes a Base64 string and returns a Tya string. It
accepts standard alphabet input with or without padding. Whitespace inside
the input is ignored.

### Example

```tya
import base64

encoded = base64.encode("hello")
println encoded                # aGVsbG8=

println base64.decode(encoded) # hello
```

### Errors

`base64.encode(text)` requires a string argument.

`base64.decode(text)` raises a structured error on characters outside the
standard alphabet, or on input that does not decode cleanly. The decoded
result is interpreted as a UTF-8 string; non-UTF-8 byte sequences raise an
error.

### Out of scope

URL-safe Base64 (`-` and `_` instead of `+` and `/`) is not part of v0.23.
Binary blob handling is not part of v0.23 because Tya has no byte-array type
yet.

## `url`

The `url` module performs percent-encoding, percent-decoding, and basic URL
parsing and construction.

### Functions

- `url.encode(text)`
- `url.decode(text)`
- `url.encode_query(pairs)`
- `url.decode_query(text)`
- `url.parse(text)`
- `url.build(parts)`

### Encoding helpers

`url.encode(text)` percent-encodes a string for safe inclusion in a URL
component. Characters that are unreserved per RFC 3986 (letters, digits,
`-`, `.`, `_`, `~`) are passed through; all other bytes are encoded as
`%XX`.

`url.decode(text)` reverses percent-encoding. It decodes `%XX` to bytes and
returns the result interpreted as a UTF-8 string. Plus signs are not treated
as spaces; for that, use `url.decode_query`.

### Query helpers

`url.encode_query(pairs)` accepts either:

- a dict of `{ key: value }` pairs, or
- an array of `[key, value]` two-element arrays (for ordered or duplicate
  keys).

It emits `key=value&key=value` form, percent-encoding each key and value
using the same rules as `url.encode`. Spaces are encoded as `%20` (not `+`).

`url.decode_query(text)` parses a `key=value&key=value` string. It returns
an array of `[key, value]` two-element arrays so that order and duplicate
keys are preserved. Both `+` and `%20` decode to a space, matching common
practice.

### Parsing and building

`url.parse(text)` returns a dict with the following string members:

- `scheme`
- `user`
- `password`
- `host`
- `port`
- `path`
- `query`
- `fragment`

Members that are absent in the input are returned as the empty string. The
`query` is returned as the raw string; pass it to `url.decode_query` to get
key/value pairs.

`url.build(parts)` is the inverse of `url.parse`. It accepts a dict with the
same keys (any may be omitted) and returns a URL string.

### Example

```tya
import url

println url.encode("hello world")            # hello%20world
println url.decode("hello%20world")          # hello world

q = url.encode_query({ q: "tya lang", page: "2" })
println q                                    # q=tya%20lang&page=2

parts = url.parse("https://example.com:8080/path?x=1#frag")
println parts["host"]                        # example.com
println parts["port"]                        # 8080

rebuilt = url.build({
  scheme: "https",
  host: "example.com",
  path: "/search",
  query: "q=tya",
})
println rebuilt                              # https://example.com/search?q=tya
```

### Errors

All `url.*` functions raise a structured error on wrong argument kinds.
`url.decode` and `url.decode_query` raise a structured error on malformed
percent-escapes or non-UTF-8 byte sequences.

## New Built-ins

v0.23 adds three small global built-ins that data-format code in Tya needs:

- `ord(char)` returns the byte value (0..255) of the first byte of the
  argument string. Empty input is an error.
- `chr(byte)` returns a one-byte string for the given byte value (0..255).
  Out-of-range values are an error.
- `kind(value)` returns one of `"nil"`, `"bool"`, `"int"`, `"float"`,
  `"string"`, `"array"`, `"dict"`, `"object"`, `"function"`, `"error"`.

`ord` and `chr` are byte-level. They are not full Unicode primitives.
Multi-byte sequences must be handled by the caller (for example by reading
raw UTF-8 bytes from a string).

`kind` distinguishes integer and float numbers based on whether the numeric
value is exactly representable as an integer.

## Short-circuit Evaluation

v0.23 changes `and` and `or` to short-circuit. The right operand is
evaluated only when the left operand does not already determine the result.
This matches the conventional behavior of Boolean operators in mainstream
languages and is required to write parser-style Tya code that guards index
accesses with `i < n and is_valid(s[i])`.

## Filesystem Modules

v0.23 ships the filesystem stdlib expansion deferred from v0.22. See
`docs/STDLIB.md` for usage details. Native operation failures raise
structured errors so they can be caught with block `try ... catch`.

- `dir.list(path)` returns an array of names directly under `path` in
  dictionary order, excluding `.` and `..`.
- `dir.mkdir(path)` creates one directory level.
- `dir.rmdir(path)` removes an empty directory.
- `file.remove(path)` removes a regular file.
- `file.rename(old_path, new_path)` renames a file or directory.
- `file.stat(path)` returns `{ kind, size, readable, writable, executable }`.
- `path.expand_user(value)` expands a leading `~` or `~/...` to the user's
  home directory.
- `os.cwd()` returns the current working directory.
- `os.chdir(path)` changes it.

## Diagnostics

v0.23 implementations should report source-oriented errors for:

- missing imports for `toml`, `json`, `csv`, `base64`, or `url`
- unknown module functions
- wrong argument counts
- wrong argument kinds
- syntax errors during `toml.parse`, `json.parse`, `csv.parse`, `url.parse`
  (with line number where applicable)
- unsupported value kinds during `toml.dump`, `json.dump`, `csv.dump`,
  `url.build`

Diagnostics should mention the module name, function name, expected argument
shape, and actual value kind when available.
