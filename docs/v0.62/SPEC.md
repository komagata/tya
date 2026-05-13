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
