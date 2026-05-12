# Tya v0.57 Specification

> **Status:** shipped. The `tya version` constant is `0.57.0`.
> v0.57 introduces the `embed` statement so files can be baked
> into a compiled tya binary at build time. The rest of the
> language surface is unchanged from v0.56.

## Theme

Tya programs are compiled to a self-contained C executable. Until
v0.57 every asset (HTML / CSS / images / config) had to ship as a
separate file beside the binary. v0.57 adds a build-time
`embed` form that turns one or more files into in-binary
constants:

- Single file → a `bytes` global.
- Glob → a `dict<string, bytes>` global keyed by the path
  relative to the source file.

This is the foundation for "single-binary distribution": a
static site or small game can ultimately ship as a single tya
binary, once a matching HTTP / graphics stdlib lands in a
follow-up Epic.

## Syntax

```tya
embed "assets/logo.png" as logo            # single file → bytes
embed "static/**" as assets                # recursive glob → dict
embed "assets/*.png" as sprites            # single-level glob → dict
```

- `embed` is a reserved name. It cannot be rebound or shadowed.
- The form is `embed STRING_LIT as IDENT NEWLINE`. Statements may
  only appear at the top level (not inside functions, classes,
  or `if` / `while` / `for` blocks).
- The path is interpreted relative to the directory of the
  source file that contains the `embed` statement, just like
  `import`.
- Path separators are always `/`. Glob metacharacters: `*`
  (single-level) and `**` (recursive). A pattern containing `*`
  or `?` is treated as a glob; everything else is a literal
  single-file path.

## Semantics

### Single-file form

```tya
embed "data.txt" as payload
print(bytes_text(payload))
```

The bytes are read at codegen time and emitted into the
generated C as a `tya_bytes_lit((const char*)(unsigned char[]){…}, N)`
initializer. The binding is registered as a top-level value so
the runtime GC can root it like any other global.

### Glob form

```tya
embed "static/**" as assets
for path, _ of assets
  print(path)
```

The codegen walks the filesystem (recursively when `**` is
present, single-level for `*`) and builds a
`tya_dict((TyaDictEntry[]){…}, N)` initializer. Dictionary keys
are the matching files' paths relative to the source file's
directory, normalized to `/`-separated form even on Windows
hosts. Insertion order is deterministic (sorted alphabetically).

A glob that matches zero files raises `TYA-E0611` at codegen.

### Type

`embed` always produces `bytes` (single) or
`dict<string, bytes>` (glob). There is no `as bytes` /
`as text` modifier and no extension-based auto-detection. When
a string value is needed, call `bytes_text` (or
`tya_bytes_text` in the runtime) explicitly:

```tya
embed "page.html" as raw
html = bytes_text(raw)
```

## Path resolution

- Patterns starting with `/` are interpreted as absolute paths.
- Relative patterns are joined to the source file's directory
  (the same rule as `import`).
- Patterns may include `..` segments and walk outside the
  source tree. tya source is a trusted boundary so v0.57 does
  not restrict it.

## Diagnostics

| Code | Trigger |
|------|---------|
| `TYA-E0610` | `embed` source file not found at codegen time |
| `TYA-E0611` | `embed` glob matched zero files |

Both codes are codegen-band diagnostics, so the failure surfaces
through `tya run`, `tya build`, and `tya emit-c`, not through
`tya check` (which stops before codegen).

## Implementation notes

- New `ast.EmbedStmt {Path, PathTok, Name, NameTok}`.
- Parser recognizes `embed` at top-level statement position
  using the same IDENT-pattern as `import`.
- Checker registers the binding under `kindUnknown` in the
  top-level scope so subsequent references resolve.
- Codegen reads the file (or walks the glob) in
  `internal/codegen/embed.go` and emits a `tya_bytes_lit` (or
  `tya_dict`) initializer assigned to the global.
- `internal/codegen/c.go::assignedNames` collects the binding so
  it gets a `TyaValue` declaration and a GC root registration.
- Formatter prints `embed %q as %s` for round-tripping.

## Scope-out (v0.58+)

- `as bytes` / `as text` modifiers and extension-based auto-
  detection.
- Build-time asset transforms (minify / gzip / hash).
- A CLI `tya embed --list` introspection command.
- Lazy / `mmap`-backed loading for very large assets.
- HTTP server stdlib (the natural consumer of glob embeds).
- SDL / raylib bindings (the natural consumer of game-asset
  embeds).
