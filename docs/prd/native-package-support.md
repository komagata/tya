---
status: approved
goal_ready: true
---

# Feature: Native Package Support

## Goal

Allow ordinary Tya packages to include native C code and native link metadata so
users can publish bindings to host libraries such as graphics, audio,
compression, database, or system APIs without adding those bindings to the Tya
standard library.

## Context

Tya currently compiles programs to C and links them with the bundled runtime.
The compiler already supports native-backed standard library internals through
hardcoded builtins and runtime C files. That mechanism is not suitable for
third-party packages because every new binding would require changing Tya
itself.

The package manager already supports `tya.toml`, `tya.lock`, path
dependencies, git dependencies, and package source resolution under `src/`.
`tya new --template lib` can scaffold a pure Tya library package. It does not
yet define a way for a package to declare native sources, include directories,
link flags, `pkg-config` dependencies, or callable native symbols.

This feature defines a general native package contract. SDL2 and raylib are
example users of the contract, not special cases.

## Behavior

- Extend `tya.toml` with an optional `[native]` table.

```toml
[native]
sources = ["native/binding.c"]
headers = ["include/binding.h"]
include_dirs = ["include"]
pkg_config = ["sdl2"]
cflags = []
ldflags = []

[native.functions]
binding_init = { symbol = "tya_binding_init", arity = 0 }
binding_poll = { symbol = "tya_binding_poll", arity = 1 }
```

- `sources`, `headers`, and `include_dirs` are relative to the package root.
- `pkg_config` entries are package names passed to `pkg-config --cflags --libs`.
- `cflags` and `ldflags` are explicit additional compiler and linker flags.
- `[native.functions]` declares Tya-callable native functions exposed by the
  package.
- Each native function declaration has:
  - `symbol`: C function name.
  - `arity`: number of Tya arguments accepted by the wrapper.
- Native wrapper functions use the Tya runtime ABI:

```c
TyaValue tya_binding_init(TyaValue __this, TyaValue a0, TyaValue a1,
                          TyaValue a2, TyaValue a3);
```

- The first version supports the same maximum positional arity as existing Tya
  runtime dynamic calls.
- Native function names must be snake_case and enter the checker as available
  predeclared function names only when the declaring package is imported or
  loaded through a dependency used by the entry program.
- Tya wrapper code remains normal package code:

```tya
# src/binding/Binding.tya
class Binding
  static init = ->
    binding_init()
```

- `tya build`, `tya run`, and `tya test` collect native metadata from:
  - the current project manifest
  - locked direct and transitive dependencies that are resolved for the program
  - path dependencies
  - git dependencies materialized under `.tya/packages/`
- The build command compiles declared native C sources alongside the generated
  C file and Tya runtime files.
- The build command adds declared include directories, `pkg-config` flags,
  `cflags`, and `ldflags` to the host C compiler invocation.
- Native metadata is deterministic:
  - package traversal order is lockfile order
  - flags are de-duplicated while preserving first occurrence
  - duplicate native function names are rejected unless they come from the same
    package declaration
- If `pkg-config` is required but missing, or a requested pkg-config package is
  missing, build/test/run fails with a clear diagnostic that names the package
  and native dependency.
- If a native source, header, or include directory is missing, build/test/run
  fails with a clear diagnostic.
- `tya install` records enough source information in `tya.lock` for native
  package builds to be reproducible with path and git dependencies.
- Add `tya new --template lib --native`. The scaffold creates:

```text
my_binding/
  tya.toml
  src/my_binding/MyBinding.tya
  native/my_binding.c
  include/my_binding.h
  tests/my_binding_test.tya
  README.md
```

- Add a native dependency check command:

```sh
tya doctor native
```

  It reports the detected C compiler, `pkg-config`, each native dependency, and
  the effective flags for the current project.

## Scope

- `internal/pkg/manifest.go`
- `internal/pkg/lockfile.go`
- package manager resolution paths if native metadata must be copied or
  normalized
- build pipeline in `cmd/tya/main.go`
- checker builtin/predeclared-name handling for package-declared native
  functions
- C codegen call lowering for package-declared native functions
- `tya run`, `tya build`, and `tya test`
- `tya new` native library scaffold
- `tya doctor native` or an equivalent native environment check command
- `docs/LIBRARIES.md`
- `docs/TERMINOLOGY.md`
- next release docs
- CLI and testscript coverage under `tests/testdata/`
- `ROADMAP.md`

## Out of Scope

- SDL2, raylib, or any specific binding implementation.
- Adding SDL2, raylib, or other host libraries to the Tya standard library.
- A central package registry.
- Binary package distribution.
- Cross-compilation toolchain management.
- C++ support in the first version.
- Dynamic library loading through `dlopen`.
- Arbitrary inline C inside `.tya` source files.
- Unsafe pointer types in the Tya language surface.
- Sandboxing native package code.

## Acceptance Criteria

- A path dependency package with `[native] sources = [...]` is compiled and
  linked when an app imports and uses that package.
- A git dependency package with native metadata works after `tya install`.
- A native wrapper function declared under `[native.functions]` can be called
  from package Tya code and returns a `TyaValue`.
- Missing native source files fail with a diagnostic naming the package and
  missing path.
- Missing `pkg-config` binary fails with a diagnostic naming `pkg-config`.
- Missing `pkg_config` package fails with a diagnostic naming the native
  package and the missing host dependency.
- Duplicate native function names across different packages fail with a clear
  conflict diagnostic.
- `tya build`, `tya run`, and `tya test` all use the same native metadata
  collection path.
- The generated native-lib scaffold builds and its test passes without external
  host libraries.
- Documentation explains how users should write C wrappers against the Tya
  runtime ABI.
- Existing pure Tya packages continue to build without requiring a C compiler
  beyond the compiler already needed by Tya execution.
- The self-host fixed point remains green.

## Verification

```sh
go test ./internal/pkg -count=1
go test ./internal/checker ./internal/codegen -count=1
go test ./tests -run TestV.*Script -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```

## Open Questions

None.
