---
layout: doc
title: Release Notes
permalink: /v0.57/release-notes/
---

# Tya v0.57 Release Notes

> **Status:** shipped. `tya version` reports `0.57.0` and
> `ROADMAP.md` carries the matching `Released` entry.

## TL;DR

v0.57 introduces **build-time asset embedding** so a tya program
can bake files into the compiled binary:

```tya
embed "assets/logo.png" as logo      # → bytes
embed "static/**" as assets          # → dict<string, bytes>
```

Single-binary distribution is a multi-Epic goal. v0.57 lays the
foundation; a HTTP stdlib (for static sites) and SDL / raylib
bindings (for games) follow in later Epics.

The language surface is unchanged from v0.56.

## What's new

### `embed "path" as name`

```tya
embed "data.txt" as payload
print(bytes_text(payload))
```

The file is read at codegen time, its bytes baked into the
generated C, and the binding is exposed as a top-level
`bytes` global.

### `embed "pattern/**" as name`

```tya
embed "static/**" as assets
for path, content of assets
  print(path)
```

Recursive globs produce a `dict<string, bytes>` keyed by the
path relative to the source file. Single-level globs
(`assets/*.png`) work the same way but only match the immediate
directory. Glob keys are normalized to `/`-separated form on all
hosts.

### Diagnostics

- `TYA-E0610 embed source not found` — single-file pattern
  points at a path that does not exist.
- `TYA-E0611 embed glob matched zero files` — glob pattern
  matched nothing.

Both fire at codegen, so `tya run`, `tya build`, and
`tya emit-c` surface them. `tya check` does not (it stops
before codegen).

## Examples

The `examples/embed_demo/` directory has working samples:

- `single.tya` — reads `data.txt` and prints its text content.
- `dir.tya` — recursively embeds `static/**` and enumerates the
  dict keys.

## Migration

`embed` is a new reserved name. Existing programs that used
`embed` as a variable or function name must rename it before
upgrading to v0.57.

## Tooling

- 5 new fixtures under `tests/testdata/v57_embed/` covering
  single-file, recursive glob, single-level glob, missing file,
  and empty glob.
- `Formula/tya.rb` → `0.57.0`,
  `editors/vscode/package.json` → `0.57.0`.

## Next

- HTTP server stdlib so embedded static sites can be served.
- SDL / raylib bindings so embedded game assets can be drawn /
  played.
- `as bytes` / `as text` modifiers (currently always bytes —
  call `bytes_text` for string conversion).
- `tya embed --list` CLI introspection.
- Build-time asset transforms (minify / gzip / hash).
