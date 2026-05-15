# Tya API

This document defines the current built-in functions and primitive method
surface. Standard library packages are documented separately in
`docs/STDLIB.md`.

Tya calls use parentheses:

```tya
print("hello")
write_file("out.txt", "hello")
```

## Core Builtins

```tya
print(value)
println(value)
assert(value)
assert_equal(expected, actual)
panic("bad state")
exit(1)
error("message")
```

`print` writes a value followed by a newline. `println` is an alias for newline
output. `assert` and `assert_equal` stop execution with source-oriented
diagnostics when the assertion fails. `panic` aborts execution. `exit` exits
with the specified process status. `error(message)` returns an error value with
a `message` entry.

## Process And Environment

```tya
args()
env("HOME")
cwd()
chdir("/tmp")
```

`args()` returns user arguments. The script filename is not included, so the
first user argument is `args()[0]`. `env(name)` returns the environment value or
`nil`. `cwd()` returns the current working directory, and `chdir(path)` changes
it.

## Files And Directories

```tya
read_file("input.txt")
write_file("out.txt", "hello")
file_append("out.txt", "\n")
file_exists("out.txt")
file_stat("out.txt")
file_remove("out.txt")
file_rename("old.txt", "new.txt")

dir_list(".")
dir_mkdir("tmp")
dir_rmdir("tmp")
path_expand_user("~/src")
```

Text file functions operate on strings. Binary file functions use bytes:

```tya
data = file_read_bytes("image.bin")
file_write_bytes("copy.bin", data)
```

## Input And Streams

```tya
read_line()
stderr_write("message\n")

stdin = io_stdin()
stdout = io_stdout()
stream = io_open("out.txt", { mode: "w" })
io_stream_write(stream, "hello")
io_stream_flush(stream)
io_stream_close(stream)
```

`read_line()` reads one line from standard input without the trailing newline
and returns `nil` at EOF. The `io_*` builtins back the class-style `io`
standard package.

## Text, Bytes, And Conversion

```tya
chr(65)
ord("A")
kind(value)
to_string(value)
to_int("42")
to_float("3.14")
to_number("3.14")

bytes("abc")
bytes_of("abc")
bytes_text(b"abc")
bytes_array(b"abc")
bytes_concat(b"a", b"b")
bytes_slice(b"abc", 0, 2)
```

Prefer method syntax when a primitive method exists, such as `value.to_s()` or
`"A".ord()`.

## Collections

```tya
equal(left, right)
delete(dict, "name")
has(dict, "name")
keys(dict)
values(dict)
pop(array)
```

`equal` performs deep equality for arrays and dictionaries. `==` checks normal
runtime equality. `value.equal?(other)` is the `Equatable` protocol method:
primitive arrays and dictionaries use deep equality there, while scalar
primitive values follow normal runtime equality. Dictionaries and arrays also
expose method forms such as `dict.has("name")`, `dict.keys()`, and
`array.pop()`.

## Primitive Methods

Every runtime value exposes common methods:

```tya
value.to_s()
value.class
value.equal?(other)
```

Strings expose text helpers:

```tya
" tya ".trim()
"tya".upper()
"TYA".lower()
"tya".starts_with("t")
"tya".ends_with("a")
"tya".contains("y")
"a,b".split(",")
"a\nb".lines()
"abc".chars()
"abc".bytes()
"abc".byte_len()
"abc".blank?()
"abc".present?()
"abc".iter()
"abc".sequence()
"abc".compare(other)
"abc".lt?(other)
"abc".lte?(other)
"abc".gt?(other)
"abc".gte?(other)
"abc".between?(min, max)
```

Arrays expose sequence helpers:

```tya
items = [1, 2, 3]
items.len()
items.empty?()
items.first()
items.last()
items.push(4)
items.pop()
items.slice(0, 2)
items.reverse()
items.sort()
items.join(",")
items.map(item -> item * 2)
items.filter(item -> item > 1)
items.find(item -> item == 2)
items.any(item -> item == 3)
items.all(item -> item > 0)
sum = total, item -> total + item
items.reduce(0, sum)
items.iter()
items.sequence()
```

`iter()` returns a consuming iterator with `has_next?()` and `next()`.
`sequence()` returns a lazy, re-iterable sequence with `map`, `filter`, `take`,
`drop`, `reduce`, and `to_a`.

Dictionaries expose key/value helpers:

```tya
dict.has("name")
dict.get("name", "fallback")
dict.set("name", "Tya")
dict.delete("name")
dict.keys()
dict.values()
dict.entries()
dict.merge(other)
dict.iter()
dict.sequence()
```

`for entry in dict` is the canonical dictionary traversal spelling. It yields
entry dictionaries with `key` and `value` fields. Use `dict.keys()` or
`dict.values()` when only one side is needed.

Numbers expose numeric helpers:

```tya
n.abs()
n.floor()
n.ceil()
n.round()
n.trunc()
n.sqrt()
n.pow(2)
n.integer?()
n.finite?()
n.nan?()
n.compare(other)
n.lt?(other)
n.lte?(other)
n.gt?(other)
n.gte?(other)
n.between?(min, max)
```

## Errors

Use `raise`, `try`, and `catch` for structured error handling.

```tya
parse = text ->
  if text == ""
    raise "empty input"
  text

try
  print(parse(""))
catch err
  print("caught: {err}")
```

`panic` is for unrecoverable runtime failure. `error(message)` is still useful
for APIs that intentionally return `value, err`.

## Time, Random, Math, Digest, Compression

These low-level builtins back class-style standard packages and are usually
used through `docs/STDLIB.md`:

```tya
time_now()
time_sleep(0.1)
random_seed(1)
random_int(1, 6)
random_float()
compress_gzip("text")
compress_gunzip(data)
digest_sha256("text")
secure_random_bytes(16)
```

Prefer the packages `time`, `random`, `math`, `compress`, `digest`, and
`secure_random` for public code.

## Networking

Socket builtins back `net/socket`:

```tya
sock = socket_connect("127.0.0.1", 8080, {})
socket_write(sock, "ping")
line = socket_read_line(sock)
socket_close(sock)
```

HTTP server support is exposed through `net/http.Server`, not direct builtin
calls. Use `http.Client` for HTTP client requests.

The stdlib stream packages expose protocol interfaces for shared I/O shapes:
`io.Readable`, `io.Writable`, `io.Closable`, `io.Flushable`, and the socket
package's `Readable`, `Writable`, and `Closable` interfaces. Concrete
`io.Reader`, `io.Writer`, `net/socket.Socket`, and `net/socket.Server` classes
declare these contracts where their existing methods match.

`serialization.Serializable` is the structured data protocol. Its `to_data()`
method is for machine-readable JSON/TOML/XML-style data trees and is preferred
by `serialization.Serializer.to_data`; use `Stringable.to_s()` for
human-readable text.

## Compiler Introspection

The compiler builtins back `compiler/*` packages:

```tya
compiler_lexer_lex(source)
compiler_parser_parse(source)
compiler_checker_check(source)
compiler_format_format(source)
```

Prefer importing `compiler/lexer`, `compiler/parser`, `compiler/checker`,
`compiler/ast`, or `compiler/format` instead of calling these builtins
directly.

## Naming

Builtins use `snake_case`. CamelCase builtin spellings are not part of the
language specification.
