# Feature: Managed Zig Native Toolchain

## Goal

Make one-line Tya installation prepare the native build toolchain needed for
ordinary `tya run` and `tya build`. Tya should use a Tya-managed Zig
distribution and invoke `zig cc` for native builds, so users do not need to
install a separate host C compiler for the core language workflow.

## Context

Current native builds use `CC` when set, otherwise `cc`. WebAssembly builds
already require `zig` and invoke `zig cc`. The public install scripts currently
install the `tya` binary and report missing native C compiler or Zig
requirements, but they do not install toolchain dependencies.

The intended direction is to make Zig the single managed compiler dependency
for Tya's core build targets. The installer should manage Zig together with
Tya, and Tya should use that managed Zig for both native and WebAssembly
compilation.

The current native runtime also links platform libraries such as `libuv`,
`zlib`, pthreads, and libm. Managing Zig alone solves the compiler dependency,
but not necessarily every native link dependency. Core one-line installation
must therefore either remove, vendor, statically link, or otherwise provide the
runtime libraries required for ordinary `tya run` and `tya build`.

## Behavior

- Official one-line installers install or update both `tya` and a pinned,
  supported Zig distribution for macOS, Linux, and Windows.
- Release packages may either bundle the pinned Zig distribution or the
  installer may download it as part of installation. In both cases the
  resulting installation owns the Zig copy under the Tya installation prefix.
- Tya resolves Zig from its managed installation before looking at user PATH.
- Native `tya run` and `tya build` invoke the resolved Zig as `zig cc`.
- WebAssembly builds use the same Zig resolver instead of independently
  requiring `zig` on PATH.
- Core native runtime link dependencies are handled as part of the same
  supported installation story. The implementation may choose vendored sources,
  static artifacts, release-bundled libraries, dependency reduction, or another
  repository-approved approach, but a fresh supported install must not require
  users to manually install `libuv`, `zlib`, or an equivalent runtime link
  prerequisite for ordinary programs.
- `CC` and host `cc` are not part of the default user-facing native build path.
  They may remain only as internal escape hatches if an implementation already
  needs them for development or tests, but the supported installed behavior is
  managed `zig cc`.
- `tya doctor native` reports the selected native compiler as managed `zig cc`,
  including the Zig path and version.
- `tya doctor wasm` reports the same managed Zig path and version.
- If managed Zig is missing or unusable, native and WebAssembly build commands
  fail with an actionable diagnostic that tells the user to reinstall or repair
  Tya, not to install a system C compiler.
- The installer is idempotent: rerunning it keeps an existing matching pinned
  Zig version and replaces or upgrades only when the pinned version changes.
- The pinned Zig version is defined in one place used by installers and docs.
  An environment override such as `TYA_ZIG_VERSION` may be supported for
  release testing, but the documented path uses the pinned version.
- The install page states that the one-line installer prepares Tya plus the
  managed Zig toolchain for core native and WebAssembly builds.
- Homebrew installation should either install the same Zig dependency through
  the formula or clearly report that Homebrew supplies Zig as a dependency.

## Scope

- Native compiler selection in `cmd/tya/main.go`.
- WebAssembly Zig selection in `cmd/tya/wasm.go`.
- Doctor output in `cmd/tya/doctor.go`.
- Shared Zig resolution helper code and unit tests.
- Release packaging and install metadata needed to place managed Zig under the
  Tya installation prefix.
- Native runtime link dependency handling for core Tya execution.
- `docs/install.sh` and `docs/install.ps1`.
- Homebrew formula or release instructions if this repository owns them.
- Homepage install sections in `docs/index.html` and `docs/ja/index.html`.
- Toolchain wording in `docs/SPEC.md` and `docs/ja/spec.md`.

## Out of Scope

- Installing OS libraries for native packages such as SQLite, GTK, raylib, or
  other third-party C dependencies.
- Replacing `pkg-config` for packages that explicitly require it.
- Making external native package builds dependency-free.
- Adding a central package registry.
- Changing Tya code generation semantics.
- Making WebAssembly execution part of `tya run`.
- Supporting unsupported OS or CPU combinations beyond the release matrix.

## Acceptance Criteria

- A fresh one-line install on supported macOS, Linux, and Windows platforms
  leaves both `tya` and the pinned managed Zig available to Tya.
- On a supported system with no host `cc` on PATH, core `tya run` and native
  `tya build` work by invoking managed `zig cc`.
- On a supported system without manually installed `libuv` or `zlib`, ordinary
  core programs still compile and link after one-line installation.
- `tya build --target wasm32-wasi` and `tya build --target wasm32-browser`
  use the same managed Zig resolver.
- Selfhost stage-2/stage-3 fixed point verification remains green when native
  compilation uses managed `zig cc`.
- `tya doctor native` reports managed `zig cc`, the Zig path, and the Zig
  version, and reports any remaining runtime link dependency status separately
  from third-party native package dependencies.
- `tya doctor wasm` reports the same managed Zig path and version.
- If the managed Zig files are removed or corrupted, native and WebAssembly
  build failures clearly point to repairing or reinstalling Tya.
- Installer reruns are safe and do not redownload an already-installed matching
  Zig version.
- The install page no longer presents a separate system C compiler as required
  for the core one-line install path.
- Documentation still states that native packages may require their own system
  libraries or `pkg-config`.

## Verification

```sh
go test ./cmd/tya -count=1
go test ./tests -count=1
go test ./... -count=1 -timeout=20m
bundle exec jekyll build --source docs --destination _site
sh -n docs/install.sh
pwsh -NoProfile -Command { $null = [scriptblock]::Create((Get-Content docs/install.ps1 -Raw)) }
```
