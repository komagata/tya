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

## `channel` (v0.42)

Bounded, FIFO buffered channels for inter-task communication.

```tya
import channel

c = channel.Channel.new(10)
spawn (() -> channel.Channel.send(c, "hello"))
print(channel.Channel.receive(c))
channel.Channel.close(c)
```

Class members on `channel.Channel`: `new`, `send`, `receive`,
`receive_timeout`, `close`, `closed?`, `select`.

`channel.Channel.send` blocks while the buffer is full and raises
if the channel has been closed. `channel.Channel.receive` blocks
while the buffer is empty; once closed, it drains the buffer and
then returns `nil` for every later call.
`channel.Channel.receive_timeout` returns `nil` when the deadline
elapses without a value. v0.42 treats `capacity = 0` as `1`; true
rendezvous channels arrive in a later minor.

## `sync` (v0.42)

Mutex, atomic integer, and wait group primitives.

```tya
import sync

m = sync.Sync.mutex()
sync.Sync.lock(m)
sync.Sync.unlock(m)

a = sync.Sync.atomic_integer(0)
sync.Sync.atomic_add(a, 1)
print(sync.Sync.atomic_load(a))

wg = sync.Sync.wait_group()
sync.Sync.wait_group_add(wg, 1)
spawn (() -> sync.Sync.wait_group_done(wg))
sync.Sync.wait_group_wait(wg)
```

Class members on `sync.Sync`: `mutex`, `lock`, `unlock`,
`with_lock`, `atomic_integer`, `atomic_add`, `atomic_load`,
`atomic_store`, `atomic_cas`, `wait_group`, `wait_group_add`,
`wait_group_done`, `wait_group_wait`.

`sync.Sync.with_lock(m, fn)` runs `fn()` with the mutex held and
releases the mutex even when `fn` raises. Tya closures cannot write
back to outer variables, so to share mutable state across tasks
pass a dict or array argument and mutate it through indexed
assignment.

## `task` (v0.43)

Cooperative cancellation for `spawn`-ed tasks.

```tya
import task

worker = () ->
  me = task.Task.current()
  while not task.Task.cancelled?(me)
    do_one_step()

t = spawn worker()
task.Task.cancel(t)
await t
```

Class members on `task.Task`: `cancel`, `cancelled?`, `current`.

`Task.cancel` sets the cancel flag on a task. `Task.cancelled?`
reads it. Cancellation is cooperative: setting the flag does not
stop a worker, only signals it. Long-running workers must poll
`Task.cancelled?(Task.current())` at safe points and return early.

`Task.current()` returns the currently-running task value, or `nil`
on the main thread. Workers use `task.Task.cancelled?(task.Task.current())`
to check whether they have been asked to stop.

A `scope` block also sets the cancel flag on every remaining
sibling once it observes the first task raise, and a synchronous
raise from the body sets the cancel flag on every spawned sibling
before the raise propagates.

## Updated for v0.43: `channel.select`

```tya
result = channel.Channel.select([
  [c1, "receive"],
  [c2, "send", value],
])
```

Returns a dict `{ index, kind, value }`. v0.43 polls each
operation in a tight loop and sleeps briefly when nothing is
ready; future minors will add a proper waiter-list mechanism.

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
IPv4-mapped IPv6 addresses. It returns a dictionary representation with stable
`version` and numeric parts. `Address.to_s(addr)` formats the normalized
address. Invalid input raises; `Address.valid?(text)` returns `false`.

`Network.parse(cidr)` accepts IPv4 and IPv6 CIDR prefixes. `Network.contains?`
checks whether an address is inside a parsed network.
