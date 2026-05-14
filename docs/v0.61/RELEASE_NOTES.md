# Tya v0.61 Release Notes

v0.61 is the current released implementation. It includes stackable interface
behavior, native packages, package tools, new class-style standard-library
packages, WASM build targets, and the first external package ecosystem split
into separate repositories.

## Language

- Interfaces may define default instance methods.
- Interface defaults can satisfy body-free requirements from other interfaces.
- Interfaces may contribute instance fields.
- Interfaces may define zero-argument `initialize` hooks.
- Interface initializer hooks run at a class constructor's `super()` point.
- Class overrides can call `super()` into the interface default stack.
- Interface default methods can call `super()` through stacked defaults.
- Same-name interface fields and unrelated default methods require explicit
  class resolution.
- String interpolation now balances nested braces and preserves quotes inside
  interpolation expressions, so forms such as `{user["name"]}` and dictionary
  literals inside interpolation compile correctly.

## Toolchain and Packages

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
- New package tool declarations let packages expose Tya script commands through
  `[tools]`, and `tya tool` runs locked dependency tools or pinned one-shot
  git/path tools.
- WebAssembly build targets were added, with unsupported native package usage
  rejected for WASM builds.

## Standard Library

- `net/http.Server` now supports route groups, middleware, custom 404/500
  handlers, named paths, redirects, PATCH/HEAD/OPTIONS/ANY routing, wildcard
  tail captures, and optional trailing-slash matching.
- New `cli.Cli` helpers parse command-line options, positional arguments,
  defaults, required options, `--`, and deterministic usage text.
- New `template.Template` renderer handles generic text templates with
  variables, dotted/indexed paths, conditionals, loops, explicit partials,
  file rendering, strict missing-value errors, and optional HTML escaping.
- `markdown.Markdown` now has class-style `parse`, `to_html_ast`, and `render`
  APIs, plus tables, task lists, strikethrough, fenced-code info strings,
  reference links, images, nested lists, setext headings, and opt-in raw HTML.
- New `compress.Compress` helpers gzip/gunzip and zlib/unzlib strings, bytes,
  and files.
- New `log.Logger` writes level-filtered text or JSON records to stderr by
  default or appends them to a configured file path.
- New `io.Io` streams wrap stdin/stdout/stderr and file readers/writers with
  line iteration, chunk reads, binary modes, and stream copying.
- New `net/ip` package parses and classifies IPv4, IPv6, IPv4-mapped IPv6, and
  CIDR networks.
- New `net/socket` package exposes TCP client/server sockets with text and
  binary reads, line helpers, address inspection, timeouts, and structured
  connection failures.
- `url.Url` now parses bracketed IPv6 hosts, validates percent escapes and
  ports, resolves relative references, normalizes paths, and preserves
  duplicate query keys.
- New `color.Color` values provide RGBA channels, hex/CSS parsing, named
  colors, conversion helpers, and deterministic color operations.
- New `geometry` values provide `Vector2`, `Vector3`, `Point`, `Size`, `Rect`,
  and `Circle` helpers for reusable spatial calculations.
- New `transform2d.Transform2D` values provide 2D affine transforms, geometry
  application helpers, inversion, composition, and matrix interop.
- New `compiler/*` introspection packages expose lexer, parser, AST, checker,
  and formatter results as stable public Tya dictionaries.
- New `binary.Reader` and `binary.Writer` values provide endian-aware integer
  and IEEE 754 float reads/writes over `bytes`.
- New `collections` values provide `Stack`, `Queue`, `Deque`, `Set`, and stable
  min-priority queue containers.
- `random` now includes independent `Rng` instances plus `bool`,
  `shuffle_copy`, `sample`, `weighted_choice`, and `weighted_index` helpers.
- New `serialization.Serializer` helpers convert primitives, dictionaries,
  bytes, and class instances to JSON-compatible data, JSON, and TOML.
- New `xml` DOM classes parse, inspect, and dump practical XML, including
  common JUnit XML output.
- New `image.Image` and `image.Codec` classes provide deterministic RGBA image
  metadata, pixel access, transforms, and byte/file round trips.

## External Packages and Tools

The following projects are released as separate repositories and are not part of
Tya's standard library:

- `https://github.com/komagata/tya-sqlite`
- `https://github.com/komagata/tya-sdl2`
- `https://github.com/komagata/tya-gtk4`
- `https://github.com/komagata/tya-raylib`
- `https://github.com/komagata/tya-slim`
- `https://github.com/komagata/flakewatch`
- `https://github.com/komagata/magvideo`

Each package/tool is distributed by git URL plus tag. Tya still has no central
package registry and no `tya publish` command.

## Verification

The release gate is:

```sh
go test ./... -count=1
```

The published v0.61.0 tag passed the full suite, including the maintained
self-host fixed-point tests.
