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
