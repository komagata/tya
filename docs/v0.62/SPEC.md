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
