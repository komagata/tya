---
status: completed
goal_ready: false
---

# Feature: WebAssembly Compilation Target

## Goal

Add WebAssembly build targets so Tya programs can compile to WASI modules for
CLI-style execution and browser modules for web embedding, while preserving the
existing compile-to-C pipeline and runtime model.

## Context

Tya currently emits C and builds native executables through the host C compiler.
`tya build` accepts an input file and `-o`, but it does not accept target
selection. The roadmap calls for Zig as the WASM toolchain and two public
targets:

```sh
tya build --target wasm32-wasi src/main.tya
tya build --target wasm32-browser src/main.tya
```

This feature should keep C as the compiler backend. The WASM work is a build
and runtime-portability layer, not a new Tya IR or direct wasm emitter.

## Behavior

- Extend `tya build` with:
  - `--target native`
  - `--target wasm32-wasi`
  - `--target wasm32-browser`
- `native` is the default and preserves current behavior.
- `--target wasm32-wasi` emits a `.wasm` module suitable for WASI runtimes.
- `--target wasm32-browser` emits:
  - a `.wasm` module;
  - a small JavaScript loader/shim when needed for browser imports.
- `-o` remains supported:
  - for `wasm32-wasi`, `-o app.wasm` writes the wasm module;
  - for `wasm32-browser`, `-o dist/app.wasm` writes wasm next to the generated
    loader;
  - if `-o` names a directory for browser output, write target files inside
    that directory.
- Use Zig as the default WASM C toolchain:
  - `zig cc -target wasm32-wasi` for WASI;
  - `zig cc -target wasm32-freestanding` or the closest supported Zig target
    for browser builds.
- Add clear diagnostics when Zig is required but unavailable.
- Keep `CC` behavior for native builds.
- Add target-aware runtime compilation:
  - include only runtime files supported by the selected target;
  - avoid POSIX-only runtime sources for WASM targets;
  - keep native package C sources out of WASM builds unless explicitly marked
    target-compatible later.
- Gate unsupported standard library modules per target with structured import
  errors.
- WASI target supports:
  - stdout/stderr;
  - args/env when WASI provides them;
  - basic file APIs where WASI preopens allow access;
  - deterministic process exit status.
- Browser target supports:
  - pure computation;
  - strings, arrays, dictionaries, classes, interfaces, errors;
  - bytes and embedded assets;
  - stdout/stderr routed through JavaScript callbacks or console output.
- Browser target rejects or gates:
  - filesystem APIs;
  - process APIs;
  - sockets;
  - `net/http` server;
  - native packages;
  - OS-specific time/process behavior that has no browser implementation.
- WASI target rejects or gates:
  - `net/http` server until a WASI-compatible implementation exists;
  - raw sockets unless WASI socket support is explicitly implemented;
  - native packages that require host libraries not available to WASM.
- Runtime behavior must be documented when it differs by target.
- `tya run` remains native-only in the first version.
- `tya emit-c` remains target-independent unless a future flag is needed.
- Add `tya doctor wasm` to report:
  - Zig availability and version;
  - supported Zig targets;
  - whether a minimal Tya WASI build can compile.

## Scope

- CLI argument parsing for `tya build --target`.
- Build planning for native, WASI, and browser outputs.
- Zig invocation and target-specific compiler/linker flags.
- WASM-compatible runtime source selection.
- Runtime portability fixes needed for WASM compilation.
- Target-aware stdlib availability checks and diagnostics.
- Target-aware native package gating.
- Browser loader/shim for stdout/stderr and exported entry invocation.
- Tests for successful WASI/browser builds and unsupported import diagnostics.
- Documentation for build targets, requirements, and target limitations.
- `tya doctor wasm`.
- `ROADMAP.md`.

## Out of Scope

- Direct WebAssembly code generation from Tya AST.
- Replacing the C emitter.
- Running WASM modules through `tya run`.
- A package registry for WASM artifacts.
- Browser DOM APIs as Tya stdlib.
- Async browser APIs or JavaScript Promise integration.
- WASI HTTP, WASI sockets, or component-model support.
- Native package cross-compilation beyond explicit rejection/gating.
- Full debugger/source-map support.
- Optimizing wasm size beyond reasonable compiler flags.

## Acceptance Criteria

- `tya build src/main.tya` still builds a native executable.
- `tya build --target native src/main.tya` matches native build behavior.
- `tya build --target wasm32-wasi src/main.tya -o app.wasm` produces a `.wasm`
  file.
- A simple WASI program prints expected stdout under a documented WASI runner.
- `tya build --target wasm32-browser src/main.tya -o dist/app.wasm` produces a
  wasm module and browser loader/shim.
- A simple browser-target program can be instantiated from the generated loader
  and produce observable output through the documented callback or console path.
- Unsupported target names produce a clear CLI error.
- Missing Zig produces a clear diagnostic for WASM builds.
- Unsupported stdlib imports produce structured target-gating diagnostics.
- Native packages are rejected for WASM builds unless explicitly declared
  compatible by a future mechanism.
- WASI builds do not link POSIX-only runtime files such as the current native
  HTTP server.
- Browser builds do not expose filesystem, process, socket, or native package
  APIs.
- Existing native build, run, test, emit-c, and self-host fixed-point behavior
  remains green.

## Verification

Focused CLI/build checks:

```sh
go test ./tests -run 'Test.*Wasm|Test.*Build' -count=1
```

Self-host invariant:

```sh
go test ./tests -run TestSelfhostV01Scripts -count=1
```

Full project check:

```sh
go test ./... -count=1
```

Manual or scripted toolchain smoke checks when Zig and a WASI runtime are
available:

```sh
tya doctor wasm
tya build --target wasm32-wasi examples/wasm/hello.tya -o /tmp/hello.wasm
tya build --target wasm32-browser examples/wasm/hello.tya -o /tmp/hello-browser/hello.wasm
```

## Dependencies

- Zig must be available for WASM build verification.
- A WASI runtime such as `wasmtime` or `wasmer` is recommended for smoke tests,
  but Go tests should skip runtime execution cleanly when no runner exists.
- Runtime portability work must preserve native runtime behavior.
- Target gating should reuse existing import/module resolution where possible.

## Open Questions

None.
