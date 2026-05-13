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
- New `template.Template` stdlib renderer handles generic text templates with
  variables, dotted/indexed paths, conditionals, loops, explicit partials,
  file rendering, strict missing-value errors, and optional HTML escaping.
- `markdown.Markdown` now has class-style `parse`, `to_html_ast`, and `render`
  APIs, plus tables, task lists, strikethrough, fenced-code info strings,
  reference links, images, nested lists, setext headings, and opt-in raw HTML.
- New `compress.Compress` stdlib helpers gzip/gunzip and zlib/unzlib strings,
  bytes, and files.
- New `log.Logger` stdlib logger writes level-filtered text or JSON records to
  stderr by default or appends them to a configured file path.
- New `io.Io` stdlib streams wrap stdin/stdout/stderr and file readers/writers
  with line iteration, chunk reads, binary modes, and stream copying.
- New `net/ip` stdlib package parses and classifies IPv4, IPv6, IPv4-mapped
  IPv6, and CIDR networks.
- New `net/socket` stdlib package exposes TCP client/server sockets with text
  and binary reads, line helpers, address inspection, timeouts, and structured
  connection failures.
- `url.Url` now parses bracketed IPv6 hosts, validates percent escapes and
  ports, resolves relative references, normalizes paths, and preserves
  duplicate query keys.
- New `color.Color` stdlib values provide RGBA channels, hex/CSS parsing, named
  colors, conversion helpers, and deterministic color operations.
- New `geometry` stdlib values provide `Vector2`, `Vector3`, `Point`, `Size`,
  `Rect`, and `Circle` helpers for reusable spatial calculations.
- New `transform2d.Transform2D` stdlib values provide 2D affine transforms,
  geometry application helpers, inversion, composition, and matrix interop.
- New `compiler/*` introspection packages expose lexer, parser, AST, checker,
  and formatter results as stable public Tya dictionaries.
- `tya lsp` now advertises and handles prepare-rename, inlay-hint, call
  hierarchy, selection range, code lens, folding range, and document-link
  provider requests, and semantic tokens include a stable modifier legend.
- New `binary.Reader` and `binary.Writer` stdlib values provide endian-aware
  integer and IEEE 754 float reads/writes over `bytes`.
- New `collections` stdlib values provide `Stack`, `Queue`, `Deque`, `Set`,
  and stable min-priority queue containers.
- `random` now includes independent `Rng` instances plus `bool`,
  `shuffle_copy`, `sample`, `weighted_choice`, and `weighted_index` helpers.
- New `serialization.Serializer` helpers convert primitives, dictionaries,
  bytes, and class instances to JSON-compatible data, JSON, and TOML.

## Verification

The release includes package-manager unit coverage and script coverage for path
dependency native builds, native run/build/test behavior, diagnostics, the
generated native library scaffold, the `cli.Cli` stdlib parser, package tool
execution, interpolation scanner edge cases, the `template.Template` stdlib
renderer, Markdown parser/rendering extensions, compression round trips and
file helpers, the `log.Logger` stdlib logger, `io.Io` stream tests, and
`net/ip` address/network tests, plus localhost `net/socket` client/server,
binary I/O, collections behavior, timeout, and connection-refused tests, and
random helper coverage, and URL parser tests for IPv6, relative resolution,
duplicate queries, serialization round trips, and malformed input.
