# Tya Standard Library

This document defines the current standard library shipped with Tya.

The standard library is a set of `.tya` modules shipped with Tya. Third-party
packages are managed separately through `tya.toml`, `tya.lock`, and
`tya install`.

## Importing

Standard modules use the same import syntax as user modules. Primitive string,
array, and dict helpers are not imported modules in v0.59; they are methods on
the values themselves.

```tya
import math as math
import json as json
```

Directory packages expose public class and interface names directly when they
are imported without an alias. Use an alias when a namespace binding is wanted:

```tya
import net/http

server = Server.new()

import net/http as http

server = http.Server.new()
```

The import search order is:

1. The importing file's directory.
1. Manifest-declared dependencies from `tya.lock`.
1. Directories listed in `TYA_PATH`, searched left to right.
1. The `stdlib/` directory shipped with Tya.

## Primitive Methods

Primitive literals expose their core operations through wrapper classes:

```
" tya ".trim()
"tya".present?()
["a", "b"].first()
["a", "b"].join(",")
{ name: "tya" }.keys()
value.class
```

See the v0.59 specification for the exhaustive primitive method surface.
Use `x.class` to inspect the runtime class wrapper for primitive and object
values.

## `color`

```tya
import color as color

red = color.Color.rgb(255, 0, 0)
blue = color.Color.hex("#0066ff")
mixed = color.Color.blend(red, blue, 0.5)

print mixed.to_hex()
```

`Color` instances expose numeric `r`, `g`, `b`, and `a` fields in the range
`0..255`. Constructors validate channels and raise `color.*` errors for invalid
input.

Class members on `color.Color` include `rgb`, `rgba`, `gray`, `hex`, `css`,
`from_array`, named color constructors, `equal?`, `nearly_equal?`, `luminance`,
`contrast_ratio`, `with_alpha`, `invert`, `grayscale`, `blend`, `over`,
`lighten`, and `darken`.

`hex(text)` accepts `#rgb`, `#rgba`, `#rrggbb`, `#rrggbbaa`, and the same forms
without `#`. `css(text)` accepts those hex forms, `rgb(...)`, `rgba(...)`, and
the documented lowercase named colors.

Instance methods:

```tya
color.to_hex()
color.to_hex(true)
color.to_array()
```

## `geometry`

```tya
import geometry as geo

p = geo.Point.new(10, 20)
v = geo.Vector2.normalize(geo.Vector2.new(3, 4))
r = geo.Rect.new(0, 0, 100, 50)

if geo.Rect.contains_point?(r, p)
  print "inside"
```

The `geometry` package provides class-style values for vectors and simple
shapes:

- `Vector2` with `x`, `y`
- `Vector3` with `x`, `y`, `z`
- `Point` with `x`, `y`
- `Size` with `width`, `height`
- `Rect` with `x`, `y`, `width`, `height`
- `Circle` with `x`, `y`, `radius`

Constructors validate numeric fields. `Size`, `Rect`, and `Circle` reject
negative dimensions where applicable. Geometry helpers return new instances and
do not mutate inputs.

Vector helpers include arithmetic, scale/divide, dot/cross products,
length/distance, normalization, linear interpolation, array conversion, and
exact/nearly-equal comparison. Shape helpers include point containment,
intersection, union, expansion, translation, bounding rectangles, and area
calculations.

## `transform2d`

```tya
import geometry as geo
import transform2d as transform2d

move = transform2d.Transform2D.translation(10, 20)
scale = transform2d.Transform2D.uniform_scale(2)
world = transform2d.Transform2D.compose(move, scale)

point = transform2d.Transform2D.apply_point(world, geo.Point.new(3, 4))
print geo.Vector2.to_array(geo.Point.to_vector(point))
```

`Transform2D` instances expose numeric `a`, `b`, `c`, `d`, `tx`, and `ty`
fields representing `[a c tx; b d ty; 0 0 1]`.

Constructors include `identity`, `translation`, `scale`, `uniform_scale`,
`rotation`, `rotation_around`, `skew`, `from_array`, and `new`.

Operations include `compose`, `translate`, `scale_by`, `rotate`,
`determinant`, `invertible?`, `inverse`, `equal?`, `nearly_equal?`,
`apply_point`, `apply_vector2`, `apply_rect`, `apply_size`, `to_matrix`, and
`from_matrix`.

## `compiler/*`

```tya
import compiler/lexer
import compiler/parser
import compiler/ast
import compiler/checker
import compiler/format

tokens = Lexer.lex("x = 1\n")["tokens"]
program = Parser.parse_tokens(tokens)["program"]
print Ast.kind(program)
print Checker.check("print missing\n")["ok"]
print Format.unparse(program)
```

The public compiler introspection packages expose stable dictionaries and
arrays instead of private Go handles:

- `compiler/lexer.Lexer.lex(source)`
- `compiler/lexer.Lexer.lex_with_comments(source)`
- `compiler/parser.Parser.parse(source)`
- `compiler/parser.Parser.parse_tokens(tokens)`
- `compiler/ast.Ast.walk(node, visitor)`
- `compiler/ast.Ast.children(node)`
- `compiler/ast.Ast.kind(node)`
- `compiler/ast.Ast.span(node)`
- `compiler/checker.Checker.check(source)`
- `compiler/checker.Checker.check_ast(program)`
- `compiler/format.Format.format(source)`
- `compiler/format.Format.unparse(program)`

Lexer tokens contain `kind`, `lexeme`, `line`, `col`, `end_line`, and
`end_col`. Parser results contain a `program` node or `nil` plus
`diagnostics`. AST nodes contain `kind`, `span`, `ast_version` on the program
node, and node-specific fields such as `body`, `targets`, `values`, and
`expr`. Diagnostics use dictionaries with `severity`, `code`, `title`,
`message`, `primary`, `hints`, and `url`.

## `binary`

```tya
import binary as binary

reader = binary.Reader.new(b"\x34\x12", { endian: "little" })
print(reader.read_u16())

writer = binary.Writer.new({ endian: "big" })
writer.write_u16(0x1234).write_i16_le(-2)
print(bytes_array(writer.bytes()))
```

`binary.Reader` reads structured values from an existing `bytes` value and
tracks a cursor. It supports `position`, `seek`, `skip`, `remaining`, `eof?`,
`read_bytes`, unsigned/signed 8/16/32-bit integers, and IEEE 754 `f32`/`f64`
methods. Multi-byte methods use the instance default endian unless the method
name ends in `_le` or `_be`.

`binary.Writer` appends structured values to an internal byte buffer. It
supports `position`, `bytes`, `write_bytes`, unsigned/signed 8/16/32-bit
integers, and IEEE 754 `f32`/`f64` methods. Write methods return the writer so
calls can be chained. Invalid endian names, invalid cursor movement, reads past
EOF, negative byte counts, and out-of-range integer writes raise `binary`
errors.

## `math`

```tya
import math as math

print math.Math.abs(-3)
print math.Math.min(2, 5)
print math.Math.max(2, 5)
print math.Math.clamp(12, 0, 10)
```

Class members on `math.Math`:

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
import path as path

print path.Path.join(["tmp", "tya", "memo.txt"])
print path.Path.clean("tmp/./tya/../memo.txt")
print path.Path.basename("/tmp/tya/memo.txt")
print path.Path.dirname("/tmp/tya/memo.txt")
print path.Path.extname("/tmp/tya/memo.txt")
```

Class members on `path.Path`:

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

The `path` standard-library API uses `/` as the path separator and does not access the file
system.

## `file`

```tya
import file as file

if file.File.exists?("memo.txt")
  text = file.File.read("memo.txt")
  println text

file.File.write("out.txt", "hello")
```

Class members on `file.File`:

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
import os as os

args = os.Os.args()
home = os.Os.env("HOME")
```

Class members on `os.Os`:

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
import dir as dir

names = dir.Dir.list(".")
dir.Dir.mkdir("tmp")
dir.Dir.rmdir("tmp")
```

Class members on `dir.Dir`:

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
import file as file

file.File.remove("memo.txt")
file.File.rename("a.txt", "b.txt")
info = file.File.stat("b.txt")
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
import base64 as base64

println base64.Base64.encode("hello")
println base64.Base64.decode("aGVsbG8=")
```

`encode(text)` returns the standard-alphabet Base64 representation with `=`
padding. `decode(text)` decodes standard Base64 (padding optional, whitespace
ignored).

## `url` (v0.62)

```tya
import url as url

println url.Url.encode("hello world")
parts = url.Url.parse("https://example.com:8080/path?x=1")
full = url.Url.resolve("https://example.com/a/b", "../c?x=1")
```

Class members on `url.Url`: `encode`, `decode`, `encode_query`,
`decode_query`, `query_dict`, `parse`, `build`, `resolve`, and `normalize`.

`Url.parse(text)` handles absolute URLs, relative references, host/port,
username/password, bracketed IPv6 hosts, path, query, and fragment. It raises
on malformed percent escapes and invalid ports. `Url.build(parts)` rebuilds a
URL from parsed parts and brackets IPv6 hosts.

`Url.resolve(base, ref)` resolves relative references against a base URL.
`Url.normalize(text)` lowercases scheme/host and removes `.` / `..` path
segments.

`Url.decode_query(text)` returns ordered `[key, value]` pairs so duplicate keys
are preserved. `Url.query_dict(query)` collapses decoded pairs into a
dictionary, storing duplicate values as arrays.

## `json`

```tya
import json as json

println json.Json.dump({ name: "tya", version: 23 })
data = json.Json.parse("[1, 2, 3]")
```

`Json.parse(text)` parses RFC 8259 JSON; `Json.dump(value)` emits compact JSON.
Numbers without fractional parts decode to ints; with fractional parts to
floats. Parsed JSON values are data-exchange dictionaries and arrays, not
stdlib domain instances.

## `csv`

```tya
import csv as csv

rows = csv.Csv.parse("name,age\ntya,1\n", { header: true })
println csv.Csv.dump([["a", "b"], ["1", "2"]], nil)
```

`Csv.parse(text, options)` accepts `{ separator, header }` options. `Csv.dump(rows,
options)` quotes fields containing the separator, quote, CR, or LF.

## `cli`

```tya
import cli

spec =
  options:
    verbose: { type: "bool", alias: "v" }
    output: { type: "string", alias: "o", required: true }

result = cli.Cli.parse(args(), spec)
```

`cli.Cli.parse(args, spec)` returns `{options, positionals, rest, errors}`.
Option specs support `bool`, `string`, `int`, `float`, and `array` types, long
flags, short aliases, grouped boolean aliases, defaults, required markers,
unknown-option handling, and `--` rest handling.

`cli.Cli.usage(command, spec)` returns deterministic usage text.
`cli.Cli.parse_or_exit(args, spec)` prints usage and errors, then exits non-zero
when parsing fails.

## `toml`

```tya
import toml

config = toml.Toml.parse("[server]\nhost = \"localhost\"\nport = 80\n")
println toml.Toml.dump(config)
```

`toml.Toml.parse(text)` parses TOML 1.0 documents.
`toml.Toml.dump(value)` emits TOML for a dict of primitives, arrays, and nested
tables (including arrays of tables).

The public `toml` stdlib package is separate from the toolchain's private TOML
parser for `tya.toml` manifests and `tya.lock` lockfiles.

## `time`

```tya
import time

now = time.Time.now()
println(time.Time.format(now, "iso"))
time.Time.sleep(0.1)
println(time.Time.since(now))
```

Class members on `time.Time`: `now`, `sleep`, `format`, `parse`,
`since`. Format layouts: `"iso"`, `"date"`, `"time"`, `"unix"`.

## `random`

```tya
import random as random

random.Random.seed(42)
println random.Random.int(1, 100)
println random.Random.float()
println random.Random.choice(["a", "b", "c"])
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
import digest as digest

println digest.Digest.md5("hello")
println digest.Digest.sha256("hello")
```

Class members on `digest.Digest`: `md5`, `sha1`, `sha256`, `sha384`,
`sha512`. Each takes a text and returns a lowercase hex digest string.

## `secure_random`

```tya
import secure_random as secure_random

println secure_random.SecureRandom.hex(16)     # 32 hex chars
println secure_random.SecureRandom.uuid()      # RFC 4122 v4
println secure_random.SecureRandom.int(0, 99)
```

Cryptographically secure. Class members on `secure_random.SecureRandom`:
`bytes`, `hex`, `base64`, `uuid`, `int`.

## `matrix`

```tya
import matrix as matrix

a = matrix.Matrix.new([[1, 2], [3, 4]])
b = matrix.Matrix.identity(2)

println matrix.Matrix.add(a, b).data
println matrix.Matrix.mul(a, a).data
println matrix.Matrix.det(a)
```

Class members on `matrix.Matrix`: `new`, `zero`, `identity`, `at`, `put`,
`add`, `sub`, `scale`, `mul`, `transpose`, `det` (<= 4x4), `equal?`.

`Matrix.new`, `Matrix.zero`, `Matrix.identity`, `Matrix.add`, `Matrix.sub`,
`Matrix.scale`, `Matrix.mul`, and `Matrix.transpose` return a Matrix instance
with public `data`, `rows`, and `cols` fields.

## `bytes` (v0.25)

`bytes` is a built-in value type for raw byte sequences (each element is an
int 0..255). Construct with the `b"..."` literal or builtins:

```tya
b1 = b"hello"
b2 = b"\x00\x01\xff"
b3 = bytes([72, 101, 108, 108, 111])

println b1.len()             # 5
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
import file as file

raw = file.File.read_bytes("/path/to/file")
file.File.write_bytes("/tmp/copy", raw)
```

`read_bytes(path)` returns a `bytes` value. `write_bytes(path, b)` writes
raw bytes.

## Updated for bytes (v0.25)

- `digest.Digest.md5/sha1/sha256/sha384/sha512` accept either `string` or `bytes`.
- `secure_random.SecureRandom.bytes(n)` returns `bytes`.
- `hex.Hex.encode(value)` accepts `string` or `bytes`. `hex.Hex.decode(text)` returns
  `bytes` (was `string` in v0.24 — breaking change).
- `base64.Base64.encode(value)` accepts `string` or `bytes`. `base64.Base64.decode(text)`
  returns `bytes` (was `string` in v0.24 — breaking change).

## `unittest`

```tya
import unittest

class StringBlankTest extends TestCase
  setup = ->
    self.subject = " "

  test_blank_for_whitespace = ->
    self.assert(self.subject.blank?(), "spaces")

  test_blank_returns_false_for_content = ->
    self.assert_equal(false, "tya".blank?(), "content")
```

Test cases are classes that extend `TestCase`. Instance methods whose names
start with `test_` are run by `tya test`; optional `setup` and `teardown`
methods run around each test method.

Assertion methods:

```tya
self.assert(cond, desc)
self.assert_falsy(cond, desc)
self.assert_equal(expected, actual, desc)
self.assert_not_equal(left, right, desc)
self.assert_nil(value, desc)
self.assert_raises(body)
self.fail(message)
```

Each assertion raises a structured `{kind: "unittest_fail", message}` value
on failure; the test runner catches it so a single failed test does not stop
the rest of the suite.

`TestSuite` is an ordered collection of `TestCase` classes, instances, or
nested suites. `TestRunner.default().run(suite)` returns a result dictionary
with `tests`, `passes`, `failures`, and `errors`; `run_and_exit` exits non-zero
when any failure or error occurred.

```tya
import unittest

suite = TestSuite()
suite.add(StringBlankTest)
TestRunner.default().run_and_exit(suite)
```


## `runtime` (v0.41)

GC introspection and explicit collection.

```tya
import runtime

stats = runtime.Runtime.gc_stats()
runtime.Runtime.gc()
```

Class members on `runtime.Runtime`: `gc_stats`, `gc`.

`runtime.Runtime.gc_stats()` returns a dict snapshot of the GC
counters with keys `alloc_count`, `alloc_bytes`, `freed_count`,
`freed_bytes`, `live_count`, `live_bytes`, `collect_count`,
`threshold`.

`runtime.Runtime.gc()` runs a full mark-and-sweep collection. The
collector treats module-level globals as roots (plus the active
raise-frame chain). Locals inside user functions are not roots in
v0.41, so `runtime.Runtime.gc()` is safe to call only at points
where every live local TyaValue is also reachable from a registered
root — in practice, at the top level of the program. See the v0.41
SPEC for the full safety contract and known limitations.

## `channel` (v0.60)

Bounded, FIFO buffered channels for inter-task communication.

```tya
import channel

c = channel.Channel(10)
spawn (() -> c.send("hello"))
print(c.receive())
c.close()
```

Constructor: `channel.Channel(capacity)`.

Instance methods: `send`, `receive`, `receive_timeout`, `close`,
`closed?`.

`c.send(value)` blocks while the buffer is full and raises if the
channel has been closed. `c.receive()` blocks while the buffer is
empty; once closed, it drains the buffer and then returns `nil` for
every later call. `c.receive_timeout(seconds)` returns `nil` when
the deadline elapses without a value. v0.60 still treats
`capacity = 0` as `1`; true
rendezvous channels arrive in a later minor.

## `sync` (v0.60)

Mutex, atomic integer, and wait group primitives.

```tya
import sync

m = sync.Mutex()
m.with_lock(() -> nil)

a = sync.AtomicInteger(0)
a.add(1)
print(a.load())

wg = sync.WaitGroup()
wg.add(1)
spawn (() -> wg.done())
wg.wait()
```

Classes: `sync.Mutex`, `sync.AtomicInteger`, `sync.WaitGroup`.

`Mutex` methods: `lock`, `unlock`, `with_lock`.
`AtomicInteger` methods: `add`, `load`, `store`, `cas`.
`WaitGroup` methods: `add`, `done`, `wait`.

## `task` (v0.60)

Cooperative cancellation for `spawn`-ed tasks.

```tya
import task

worker = () ->
  me = task.Task.current()
  while not me.cancelled?()
    do_one_step()

t = spawn worker()
t.cancel()
await t
```

Class member on `task.Task`: `current`.

Task instance methods: `cancel`, `cancelled?`.

`t.cancel()` sets the cancel flag on a task. `t.cancelled?()` reads
it. Cancellation is cooperative: setting the flag does not stop a
worker, only signals it. Long-running workers must poll
`task.Task.current().cancelled?()` at safe points and return early.

`task.Task.current()` returns the currently-running task value, or
`nil` on the main thread.

A `scope` block also sets the cancel flag on every remaining
sibling once it observes the first task raise, and a synchronous
raise from the body sets the cancel flag on every spawned sibling
before the raise propagates.

## `select` (v0.60)

```tya
select
  value = receive c1
    print(value)
  send c2, value
    print("sent")
  timeout 1.0
    raise "timeout"
  default
    print("idle")
```

`select` waits for the first ready channel operation, timeout, or
default arm and runs that arm's body. Receive bindings are scoped to
their arm body.

## `markdown` (v0.62)

Markdown parsing and HTML rendering.

```tya
import markdown as markdown

ast = markdown.Markdown.parse("# Hello")
html = markdown.Markdown.render(markdown.Markdown.to_html_ast(ast))
```

Class members on `markdown.Markdown`: `parse`, `to_html_ast`, `render`,
`to_html`.

`Markdown.parse(text)` returns a document dictionary with stable block `kind`
fields. `Markdown.to_html_ast(ast)` prepares that document for rendering.
`Markdown.render(ast)` renders a Markdown document or HTML-oriented document.
`Markdown.to_html(text)` is equivalent to
`Markdown.render(Markdown.to_html_ast(Markdown.parse(text)))`.

Supported block syntax includes paragraphs, ATX and setext headings, thematic
breaks, block quotes, ordered and unordered lists, one-level nested unordered
lists, task lists, pipe tables, fenced code blocks with info strings, and a
selected raw HTML block subset. Inline syntax includes emphasis, strong,
strikethrough, code spans, links, reference links, images, and autolinks.

HTML is escaped by default. Raw HTML blocks pass through only when rendering
with `{ raw_html: true }`.

## `compress` (v0.62)

Gzip and zlib compression helpers.

```tya
import compress as compress

packed = compress.Compress.gzip("hello")
text = bytes_text(compress.Compress.gunzip(packed))
```

Class members on `compress.Compress`: `gzip`, `gunzip`, `zlib`, `unzlib`,
`gzip_file`, `gunzip_file`.

`Compress.gzip(value)` and `Compress.zlib(value)` accept strings or bytes and
return compressed bytes. `Compress.gunzip(bytes)` and
`Compress.unzlib(bytes)` return decompressed bytes; callers convert to text
explicitly with `bytes_text` when needed. Invalid compressed data raises a
clear error.

`Compress.gzip_file(src, dst)` and `Compress.gunzip_file(src, dst)` read and
write files through the `io` stream package.

## `template` (v0.62)

Generic text template rendering.

```tya
import template as template

html = template.Template.render_html(
  "Hello, {{ name }}",
  { name: "Tya" }
)
```

Class members on `template.Template`: `render`, `render_file`, `render_html`.

`Template.render(source, data)` renders a template string using values from
`data`. `Template.render(source, data, options)` accepts options.
`Template.render_file(path, data)` reads a template file and renders it with
the same semantics.

Template tags use `{{ name }}` for escaped or plain insertion depending on
`options["escape"]`. Dotted and indexed paths such as `{{ user.name }}` and
`{{ items[0].name }}` are supported. Missing values render as an empty string
unless `{ strict: true }` is passed.

Blocks use `{{ if path }}` / `{{ else }}` / `{{ end }}` and
`{{ for item in items }}` / `{{ end }}`. Explicit partials use
`{{ partial "name" context }}` with `partials` supplied in options.

`Template.render_html(source, data)` is equivalent to
`Template.render(source, data, { escape: "html" })` and escapes `&`, `<`, `>`,
`"`, and `'`. Triple-brace tags such as `{{{ trusted_html }}}` insert trusted
content without escaping.

## `log` (v0.62)

Structured application logging.

```tya
import log as log

logger = log.Logger.default()
logger.info("started", { port: 8080 })
logger.with({ request: "42" }).warn("slow", { ms: 120 })
```

Class members on `log.Logger`: `default`, `new`.

Instance methods: `debug`, `info`, `warn`, `error`, `with`, `level`.

`Logger.default()` writes text logs to stderr at level `info`. The supported
levels are `debug`, `info`, `warn`, and `error`; messages below the current
level are suppressed. `logger.level("debug")` changes the minimum level and
returns the logger.

`Logger.new(options)` accepts `level`, `format`, `file`, and `fields`.
`format: "json"` emits deterministic JSON records. `file: "path"` appends log
records to a file instead of stderr. `fields` are included on every record, and
`logger.with(fields)` returns a child logger with merged base fields.

Text logs use a deterministic field order:

```text
2026-05-13T12:00:00Z info started app=api port=8080
```

## `io` (v0.62)

Line-oriented and chunk-oriented streams.

```tya
import io as io

reader = io.Io.open("input.txt", "r")
writer = io.Io.open("output.txt", "w")
count = io.Io.copy(reader, writer)
reader.close()
writer.close()
```

Class members on `io.Io`: `stdin`, `stdout`, `stderr`, `open`, `copy`.

Reader methods: `read`, `read_line`, `each_line`, `eof?`, `close`.
Writer methods: `write`, `write_line`, `flush`, `close`.

`Io.stdin()`, `Io.stdout()`, and `Io.stderr()` wrap the process streams.
Closing those borrowed streams is a no-op for the host process. `Io.open(path,
mode)` accepts text modes such as `"r"` and `"w"` and binary modes `"rb"` and
`"wb"`.

`reader.read(size)` returns a string in text mode and bytes in binary mode. It
returns an empty value at EOF for chunk reads; `reader.read_line()` returns
`nil` after the last line. `reader.each_line(fn)` calls `fn(line)` for each
line, preserving trailing newlines.

`writer.write(value)` writes strings, bytes, or stringified values and returns
the byte count. `writer.write_line(value)` appends a newline. `Io.copy(reader,
writer)` copies chunks until EOF, flushes the writer, and returns the copied
byte count.

## `net/ip` (v0.62)

IP address and CIDR parsing.

```tya
import net/ip as ip

addr = ip.Address.parse("2001:db8::1")
print(ip.Address.to_s(addr))

network = ip.Network.parse("2001:db8::/32")
print(ip.Network.contains?(network, addr))
```

Class members on `net/ip.Address`: `parse`, `valid?`, `version`, `to_s`,
`loopback?`, `private?`, and `unspecified?`.

Class members on `net/ip.Network`: `parse` and `contains?`.

`Address.parse(text)` accepts IPv4 dotted decimal, full or compressed IPv6, and
IPv4-mapped IPv6 addresses. It returns an Address instance with stable public
`version` and numeric-part fields. `Address.to_s(addr)` formats the normalized
address. Invalid input raises; `Address.valid?(text)` returns `false`.

`Network.parse(cidr)` accepts IPv4 and IPv6 CIDR prefixes. `Network.contains?`
checks whether an address is inside a parsed network. `Network.parse` returns a
Network instance.

## `net/socket` (v0.62)

TCP client and server sockets.

```tya
import net/socket as socket

server = socket.Server.listen("127.0.0.1", 9000, {})
client = socket.Socket.connect("127.0.0.1", 9000, {})
client.write_line("hello")
client.close()
server.close()
```

Class members on `net/socket.Socket`: `connect`.

Class members on `net/socket.Server`: `listen`.

Socket methods: `read`, `read_line`, `write`, `write_line`, `close`,
`closed?`, `local_address`, and `remote_address`.

Server methods: `accept`, `close`, and `local_address`.

`Socket.connect(host, port, options)` connects to a TCP server. `Server.listen`
binds a TCP listener; pass `0` for the port to request an ephemeral port and
read it back from `server.local_address()["port"]`.

`read(size)` returns a string in text mode and bytes in binary mode.
`read_line()` returns one line including the newline, or `nil` after EOF.
`write(value)` accepts strings, bytes, or stringified values and returns the
byte count. `write_line(value)` appends a newline.

Options are dictionaries. `{ mode: "binary" }` makes reads return bytes.
`{ timeout: seconds }` sets blocking read, write, and accept timeouts. DNS,
connection-refused, timeout, and closed-socket failures raise.
