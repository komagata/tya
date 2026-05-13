# Tya v0.62 Specification

> **Status:** draft. v0.62 adds native package support for ordinary Tya
> packages.

## Native Packages

Packages may declare native C sources and link metadata in `tya.toml`:

```toml
[native]
sources = ["native/binding.c"]
headers = ["include/binding.h"]
include_dirs = ["include"]
pkg_config = []
cflags = []
ldflags = []

[native.functions]
binding_init = { symbol = "tya_binding_init", arity = 0 }
```

All paths are relative to the package root. `pkg_config` names are passed to
`pkg-config --cflags --libs`. Flags are de-duplicated while preserving first
occurrence.

Native wrapper functions use the runtime ABI:

```c
TyaValue tya_binding_init(TyaValue __this, TyaValue a0, TyaValue a1,
                          TyaValue a2, TyaValue a3);
```

Declared native functions are available as predeclared function names to package
Tya code loaded through the current project or locked dependencies. `tya build`,
`tya run`, and `tya test` compile declared native sources with the generated C
program and runtime.

`tya new --template lib --native <name>` creates a native library scaffold.
`tya doctor native` reports the detected C compiler, `pkg-config`, native
packages, sources, include directories, and effective flags.

## CLI Stdlib

The standard library includes a class-style `cli` package for predictable
command-line option parsing:

```tya
import cli

spec =
  options:
    verbose: { type: "bool", alias: "v" }
    output: { type: "string", alias: "o", required: true }

result = cli.Cli.parse(args(), spec)
```

`cli.Cli.parse(args, spec)` returns a dictionary with `options`,
`positionals`, `rest`, and `errors`.

Option specs live under `spec["options"]`. Each option can declare `type`,
`alias`, `default`, `required`, and `help`. Supported types are `bool`,
`string`, `int`, `float`, and `array`.

Supported forms:

- `--name value`
- `--name=value`
- `--flag` and `--no-flag` for boolean options
- `-v`, `-o value`, and grouped boolean aliases such as `-abc`
- `--` to stop option parsing and preserve the remaining arguments in `rest`

Unknown options produce structured parse errors unless `allow_unknown` is true.
Required options produce structured parse errors. `cli.Cli.usage(command, spec)`
returns deterministic usage text, and `cli.Cli.parse_or_exit(args, spec)` prints
usage/errors and exits non-zero on parse failure.

## Package Tools

Packages may declare Tya script tools in `tya.toml`:

```toml
[tools]
format_docs = "tools/format_docs.tya"
```

Tool paths are relative to the package root and must point to lowercase
entry-script `.tya` files, not PascalCase class files.

`tya tool <command> [args...]` discovers tools from the current project's
locked dependencies and runs the selected script with the same execution path as
`tya run`. The tool process receives forwarded stdin, stdout, stderr, arguments,
and exit status, and it runs with the invoking project root as its current
working directory.

`tya tool --list` prints available locked dependency tools in deterministic
order. If more than one locked package declares the same command name,
unqualified execution fails and reports the conflicting packages. Use
`tya tool package_name:command` to select one package explicitly.

`tya tool` requires a current `tya.lock`. Missing or stale lockfiles fail with a
diagnostic telling the user to run `tya install`.

One-shot execution runs tools from explicit sources without editing `tya.toml`
or `tya.lock`:

```sh
tya tool --path ../tools format_docs --check
tya tool --git https://github.com/example/tya-tools --tag v1.2.0 format_docs
tya tool --git https://github.com/example/tya-tools --rev <commit> format_docs
```

One-shot git tools are cached under `.tya/cache/exec/`. Branch execution is
rejected; use `--tag` or `--rev` so remote code execution is pinned.
`tya tool --offline` only uses already materialized project packages or cached
one-shot git packages.

## Color Stdlib

The standard library includes `color.Color`, a class-style RGBA value shared by
graphics, image, terminal, and web tooling.

```tya
import color as color

red = color.Color.rgb(255, 0, 0)
blue = color.Color.hex("#0066ff")
mixed = color.Color.blend(red, blue, 0.5)

print mixed.to_hex()
```

`Color` instances expose integer `r`, `g`, `b`, and `a` fields in `0..255`.
Constructors include `rgb`, `rgba`, `gray`, `hex`, `css`, `from_array`, and
named color helpers such as `red`, `blue`, `white`, and `transparent`.

`Color.hex` supports short and long RGB/RGBA forms with or without `#`.
`Color.css` supports those hex forms, `rgb(...)`, `rgba(...)`, and the common
lowercase names documented in `docs/STDLIB.md`.

Operations include `to_hex`, `to_array`, `equal?`, `nearly_equal?`,
`luminance`, `contrast_ratio`, `with_alpha`, `invert`, `grayscale`, `blend`,
`over`, `lighten`, and `darken`.

## Geometry Stdlib

The standard library includes a pure Tya `geometry` package for deterministic
vector and shape calculations.

```tya
import geometry as geo

p = geo.Point.new(10, 20)
v = geo.Vector2.normalize(geo.Vector2.new(3, 4))
r = geo.Rect.new(0, 0, 100, 50)
```

Public classes are `Vector2`, `Vector3`, `Point`, `Size`, `Rect`, and
`Circle`. Values are class instances exposing numeric public fields and helpers
return new instances rather than mutating inputs.

`Vector2` and `Vector3` provide arithmetic, scaling, division, dot products,
length/distance helpers, normalization, linear interpolation, array conversion,
and exact/nearly-equal comparison. `Vector3` also provides `cross`.

`Point`, `Size`, `Rect`, and `Circle` provide common layout and collision
helpers, including rectangle containment/intersection/union and circle
containment/intersection/bounding-rect operations.

## Transform2D Stdlib

The standard library includes `transform2d.Transform2D`, a class-style affine
2D transform value for translation, rotation, scaling, skewing, composition,
and coordinate conversion.

```tya
import geometry as geo
import transform2d as transform2d

move = transform2d.Transform2D.translation(10, 20)
scale = transform2d.Transform2D.uniform_scale(2)
world = transform2d.Transform2D.compose(move, scale)
point = transform2d.Transform2D.apply_point(world, geo.Point.new(3, 4))
```

`Transform2D` exposes `a`, `b`, `c`, `d`, `tx`, and `ty`, representing the
affine matrix `[a c tx; b d ty; 0 0 1]`. `compose(a, b)` applies `b` first and
then `a`.

The API includes constructors for identity, translation, scale, uniform scale,
rotation, rotation around a point, skew, and array conversion. It can apply
transforms to `geometry.Point`, `geometry.Vector2`, `geometry.Rect`, and
`geometry.Size`, and it converts to/from class-style `matrix.Matrix` values.

## Compiler Introspection Stdlib

The standard library exposes the reference compiler through public class-style
packages:

- `compiler/lexer.Lexer`
- `compiler/parser.Parser`
- `compiler/ast.Ast`
- `compiler/checker.Checker`
- `compiler/format.Format`

These APIs return public Tya dictionaries and arrays, not Go handles. Tokens
include stable `kind` strings, lexemes, and `line` / `col` / `end_line` /
`end_col` spans. Parser results return `{ program: ..., diagnostics: [...] }`,
where the top-level AST node has `kind: "program"`, `ast_version: 1`, `body`,
and `file_header_comments`.

Diagnostics use structured dictionaries with `severity`, `code`, `title`,
`message`, `primary`, `hints`, and `url`. Invalid source returns diagnostics
instead of panicking. `Format.unparse(Parser.parse(source)["program"])`
provides deterministic canonical output for supported ASTs.

## Binary Stdlib

The standard library includes `binary.Reader` and `binary.Writer` for
endian-aware structured access to `bytes` values.

```tya
import binary as binary

reader = binary.Reader.new(b"\x34\x12", { endian: "little" })
value = reader.read_u16()

writer = binary.Writer.new(nil)
writer.write_u16(value)
out = writer.bytes()
```

The default endian is big-endian. `{ endian: "little" }` switches an instance
default, and `_le` / `_be` method suffixes override the default for a single
operation. Readers support cursor movement, remaining/eof checks, byte slices,
u8/i8/u16/i16/u32/i32, and f32/f64. Writers support the same numeric widths and
return `self` from write methods.

## Collections Stdlib

The standard library includes a pure Tya `collections` package with mutable
class-style container instances.

```tya
import collections as collections

queue = collections.Queue.new()
queue.push("job")
next = queue.pop()

seen = collections.Set.from_array(["asset.png", "asset.png"])
print(seen.has?("asset.png"))
```

Public classes are `Stack`, `Queue`, `Deque`, `Set`, and `PriorityQueue`.
Every collection supports `new`, `from_array`, `len`, `empty?`, `clear`, and
`to_array`. Empty pop and peek methods raise collection-specific errors because
`nil` is a valid stored value.

`Set` uses Tya value equality, preserves first-insertion order, and provides
`union`, `intersection`, `difference`, and `subset?`. `PriorityQueue` is a
stable min-priority queue whose `to_array()` returns pop order without mutating
the queue.

## Interpolation Expression Scanning

Interpolated strings now balance nested braces while scanning `{expression}`
bodies. Quotes and braces inside string literals that appear in the expression
do not terminate the interpolation body, so dictionary indexing and dictionary
literals work without escaping inner quotes:

```tya
user = {"name": "komagata"}
print("Hello, {user["name"]}!")
print("kind: {{"kind": "ok"}["kind"]}")
```

Triple-quoted interpolating strings use the same scanner. Raw strings and bytes
literals remain non-interpolating.

## Template Stdlib

`import template` exposes `template.Template`, a generic text template renderer
for application output, HTML, configuration files, generated code, emails, and
documentation.

`Template.render(source, data)` renders a template string. Tags use
`{{ name }}` for value insertion and support dotted/indexed paths such as
`{{ user.name }}` and `{{ items[0].name }}`. Missing values render as an empty
string by default; `{ strict: true }` reports missing values as template
errors.

`Template.render(source, data, options)` accepts options. `escape: "html"`
escapes `&`, `<`, `>`, `"`, and `'`; `escape` defaults to `"none"`.
`Template.render_html(source, data)` is equivalent to HTML escaping mode.
Triple-brace tags such as `{{{ trusted_html }}}` explicitly bypass escaping.

Conditionals use `{{ if path }}` / `{{ else }}` / `{{ end }}`. Loops use
`{{ for item in items }}` / `{{ end }}` and render the body once per item.
Explicit partials use `{{ partial "name" context }}` with a `partials`
dictionary supplied through options.

`Template.render_file(path, data)` and `Template.render_file(path, data,
options)` read a template file and render it with the same semantics as
`Template.render`.

## Markdown Stdlib

`import markdown` exposes `markdown.Markdown` with class-style parsing and
rendering APIs: `Markdown.parse(text)`, `Markdown.to_html_ast(ast)`,
`Markdown.render(ast_or_html_ast)`, and `Markdown.to_html(text)`.

`Markdown.parse` returns a document dictionary with stable block `kind` fields.
`Markdown.to_html(text)` remains the simple one-step API and is equivalent to
`Markdown.render(Markdown.to_html_ast(Markdown.parse(text)))` for supported
syntax.

The supported subset includes tables, task lists, strikethrough, fenced-code
info strings, reference links, images, nested unordered lists, setext headings,
and selected HTML blocks. HTML remains escaped by default; raw HTML block
pass-through is opt-in with `{ raw_html: true }` when calling `render`.

## Compress Stdlib

`import compress` exposes `compress.Compress` for gzip and zlib compression.

`Compress.gzip(value)` and `Compress.zlib(value)` accept strings or bytes and
return compressed bytes. `Compress.gunzip(bytes)` and
`Compress.unzlib(bytes)` return decompressed bytes and raise on invalid
compressed input.

`Compress.gzip_file(src, dst)` and `Compress.gunzip_file(src, dst)` provide
file helpers built on the `io` stream package.

## Log Stdlib

`import log` exposes `log.Logger`, a small structured logger for CLI tools,
servers, package tools, and long-running tasks.

`Logger.default()` creates a text logger that writes to stderr at level `info`.
`Logger.new(options)` accepts `level`, `format`, `file`, and `fields`.
Supported levels are `debug`, `info`, `warn`, and `error`; records below the
current level are suppressed.

Logger instances provide `debug(message, fields)`, `info(message, fields)`,
`warn(message, fields)`, and `error(message, fields)`. `fields` may be omitted
and defaults to `{}`. Text output uses deterministic sorted field order.
`format: "json"` emits JSON records with stable keys.

`logger.with(fields)` returns a child logger that includes merged base fields on
every record. `logger.level(value)` changes the minimum level and returns the
logger. `file: "path"` appends records to a file; stderr remains the default
destination.

## IO Stdlib

`import io` exposes `io.Io`, `io.Reader`, and `io.Writer` for process streams,
file streams, line iteration, and chunk copying.

`Io.stdin()`, `Io.stdout()`, and `Io.stderr()` return borrowed wrappers around
the process streams. Closing those wrappers does not close the host process
stream. `Io.open(path, mode)` opens text modes such as `"r"` and `"w"` and
binary modes `"rb"` and `"wb"`.

Reader instances provide `read(size)`, `read_line()`, `each_line(fn)`,
`eof?()`, and `close()`. Text readers return strings; binary readers return
bytes. `read_line()` returns `nil` after the last line.

Writer instances provide `write(value)`, `write_line(value)`, `flush()`, and
`close()`. `write` returns the number of bytes written.

`Io.copy(reader, writer)` copies chunks until EOF, flushes the writer, and
returns the copied byte count.

## Net IP Stdlib

`import net/ip` exposes `Address` and `Network` classes for shared address
handling across networking libraries.

`Address.parse(text)` accepts IPv4 dotted decimal, full or compressed IPv6, and
IPv4-mapped IPv6 addresses. `Address.valid?(text)` returns `false` for invalid
input instead of raising. `Address.version(addr)` returns `4` or `6`, and
`Address.to_s(addr)` returns a normalized string representation.

`Address.loopback?(addr)`, `Address.private?(addr)`, and
`Address.unspecified?(addr)` classify conventional IPv4 and IPv6 ranges.

`Network.parse(cidr)` accepts IPv4 and IPv6 CIDR prefixes.
`Network.contains?(network, addr)` reports whether a parsed address is inside a
parsed network.

## Net Socket Stdlib

`import net/socket` exposes `Socket` and `Server` classes for TCP sockets.

`Socket.connect(host, port, options)` connects to a TCP server. `Server.listen`
binds a TCP listener; `server.accept()` returns a connected socket and
`server.close()` closes the listener.

Socket instances provide `read(size)`, `read_line()`, `write(value)`,
`write_line(value)`, `close()`, `closed?()`, `local_address()`, and
`remote_address()`. Address values are dictionaries with `host` and `port`.

`read(size)` returns a string in text mode and bytes when opened with
`{ mode: "binary" }`. `write(value)` accepts strings and bytes. `{ timeout:
seconds }` sets blocking read, write, and accept timeouts. DNS,
connection-refused, timeout, and closed-socket failures raise.

## URL Stdlib Extensions

`import url` exposes `url.Url` for URL encoding, parsing, building,
normalization, and relative resolution.

`Url.parse(text)` handles absolute URLs, relative references, host/port,
username/password, bracketed IPv6 hosts, path, query, and fragment. Malformed
percent escapes and invalid ports raise.

`Url.build(parts)` rebuilds a URL from parsed parts and brackets IPv6 hosts.
`Url.resolve(base, ref)` resolves relative references against a base URL.
`Url.normalize(text)` lowercases scheme/host and removes dot segments from the
path.

`Url.decode_query(text)` returns ordered `[key, value]` pairs so duplicate keys
are preserved. `Url.query_dict(query)` collapses decoded pairs into a
dictionary, storing duplicate values as arrays.
