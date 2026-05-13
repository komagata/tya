# Tya v0.62 Release Notes

v0.62 lets ordinary packages ship native C wrapper code without changing Tya's
standard library or runtime.

## Highlights

- `tya.toml` supports an optional `[native]` table with C sources, headers,
  include directories, `pkg-config` dependencies, `cflags`, `ldflags`, and
  declared native functions.
- `tya build`, `tya run`, and `tya test` use the same native metadata
  collection path.
- Native wrappers are called through the existing `TyaValue` runtime ABI.
- Missing native files, missing `pkg-config`, missing host dependencies, and
  duplicate native function names produce clear diagnostics.
- `tya new --template lib --native <name>` creates a buildable native package
  scaffold.
- `tya doctor native` reports the current native build environment and
  effective flags.
- New `cli.Cli` stdlib helpers parse command-line options, positional
  arguments, defaults, required options, `--`, and deterministic usage text.
- New package tool declarations let packages expose Tya script commands through
  `[tools]`, and `tya tool` runs locked dependency tools or pinned one-shot
  git/path tools.
- String interpolation now balances nested braces and preserves quotes inside
  interpolation expressions, so forms such as `{user["name"]}` and dictionary
  literals inside interpolation compile correctly.

## Verification

The release includes package-manager unit coverage and script coverage for path
dependency native builds, native run/build/test behavior, diagnostics, the
generated native library scaffold, the `cli.Cli` stdlib parser, package tool
execution, and interpolation scanner edge cases.
