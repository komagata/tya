---
status: completed
goal_ready: false
---

# Feature: Asset-Embedding Extensions

## Goal

Extend v0.57 asset embedding so Tya programs can build deployable static asset
bundles: transform assets at build time, inspect embedded manifests from the
CLI, and serve embedded assets through `net/http.Server.static`.

## Context

Tya v0.57 shipped:

```tya
embed "assets/logo.png" as logo
embed "static/**" as assets
```

Without modifiers, single-file embeds produce `bytes` and glob embeds produce
`dict<string, bytes>` keyed by normalized relative path. That behavior must stay
compatible.

The roadmap tracks three follow-up extensions:

- build-time asset transforms: minify, gzip, hash;
- `tya embed --list` CLI introspection;
- `app.static("/assets", embedded_dict)` for the HTTP server.

## Behavior

- Preserve existing v0.57 `embed "pattern" as name` behavior exactly.
- Add an optional transform clause:

  ```tya
  embed "static/**" as assets with { gzip: true, hash: true, minify: true }
  ```

- The `with` value is a dictionary literal with supported boolean keys:
  - `gzip`
  - `hash`
  - `minify`
- Unknown transform keys are compile-time errors.
- Transform option values must be boolean literals in the first version.
- `with` on a single-file embed returns one metadata dictionary.
- `with` on a glob embed returns `dict<string, dict>` keyed by the original
  normalized source path.
- The original source path remains the stable lookup key.
- Each metadata dictionary has:
  - `path`: original normalized path;
  - `content`: transformed or original bytes for normal serving;
  - `size`: original byte size;
  - `hash`: lowercase hex SHA-256 when `hash: true`, otherwise `nil`;
  - `hashed_path`: fingerprinted path when `hash: true`, otherwise the original
    path;
  - `gzip`: gzip-compressed bytes when `gzip: true`, otherwise `nil`;
  - `gzip_size`: gzip byte size when `gzip: true`, otherwise `nil`;
  - `content_type`: best-effort MIME type from extension;
  - `encoding`: `nil` for `content`; gzip is represented separately.
- `minify: true` applies deterministic safe minification to known text assets:
  - `.html`
  - `.css`
  - `.js`
  - `.json`
  - `.svg`
- Unknown file types are not minified; their bytes pass through unchanged.
- Minification must not require a large external parser dependency.
- Hashes are computed after minification, before gzip.
- `hashed_path` preserves the extension:

  ```text
  app.css -> app.<sha256-prefix>.css
  app.min.css -> app.min.<sha256-prefix>.css
  ```

- Use a 16-character lowercase hex SHA-256 prefix for `hashed_path`.
- Detect hashed-path collisions within one embed binding and raise a structured
  codegen diagnostic.
- Glob embed ordering remains deterministic.
- `tya embed --list <file.tya>` prints embedded assets declared by that source
  file without running the program.
- `tya embed --list` supports:
  - human-readable table output by default;
  - `--format=json` for machine-readable output.
- The list output includes:
  - binding name;
  - source path;
  - output path;
  - original size;
  - transformed size;
  - gzip size when available;
  - hash when available;
  - content type;
  - enabled transforms.
- Add `net/http.Server.static(prefix, assets)`.
- `static` registers GET routes below `prefix`.
- `assets` may be a v0.57 `dict<string, bytes>` or the new metadata dict shape.
- For metadata assets:
  - requests for original paths serve `content`;
  - requests for `hashed_path` also serve the same asset;
  - `Content-Type` is set from `content_type`;
  - `ETag` is set from `hash` when present;
  - `Cache-Control` is long-lived for `hashed_path` and conservative for
    original paths;
  - if the request `Accept-Encoding` includes `gzip` and `gzip` bytes exist,
    serve the gzip bytes with `Content-Encoding: gzip`.
- Prevent path traversal in `static`:
  - normalize request paths;
  - reject `..` traversal;
  - never serve paths outside the embedded dictionary.
- Missing static assets return the existing HTTP 404 behavior.

## Scope

- Parser and AST support for the optional `embed ... with { ... }` clause.
- Formatter round-trip for transformed embeds.
- Checker validation for transform dictionaries.
- C codegen asset transform pipeline in or near `internal/codegen/embed.go`.
- CLI support for `tya embed --list`.
- `net/http.Server.static`.
- HTTP runtime support if route dictionaries need static route metadata.
- MIME type helper for common web assets.
- Tests under v57 embed coverage and v58 HTTP coverage, or new focused
  fixtures.
- Examples under `examples/embed_demo/`.
- Documentation updates for embed, HTTP static serving, and release notes.
- `ROADMAP.md`.

## Out of Scope

- Lazy or mmap-backed loading for very large assets.
- User-defined transform plugins.
- Source maps.
- Brotli compression.
- Image optimization.
- CSS/JS semantic rewriting.
- HTML templating.
- Directory index generation.
- Range requests.
- Conditional request handling beyond basic `ETag` response headers.
- CDN integration.
- SDL/raylib bindings.

## Acceptance Criteria

- Existing v0.57 embed tests remain compatible.
- `embed "file" as value with { hash: true }` returns metadata for one file.
- `embed "static/**" as assets with { gzip: true, hash: true, minify: true }`
  returns deterministic metadata for every matched file.
- Unsupported transform keys produce a structured diagnostic.
- Non-boolean transform values produce a structured diagnostic.
- Empty globs still produce the existing empty-glob diagnostic.
- Missing files still produce the existing missing-file diagnostic.
- Minification is deterministic and preserves valid content for supported text
  asset types.
- Hashes are stable across repeated builds with unchanged inputs.
- `hashed_path` preserves extensions and is collision-checked.
- `tya embed --list <file.tya>` reports embedded assets without running user
  code.
- `tya embed --list --format=json <file.tya>` emits parseable JSON.
- `http.Server.static(prefix, assets)` serves v0.57 byte dictionaries.
- `http.Server.static(prefix, assets)` serves metadata dictionaries.
- Static responses set `Content-Type`.
- Static responses set `ETag` when a hash is available.
- Static responses serve gzip bytes when `Accept-Encoding: gzip` is present.
- Static requests cannot traverse outside the embedded asset dictionary.
- The self-host fixed point remains green.

## Verification

Focused embed and HTTP checks:

```sh
go test ./tests -run 'TestV(57|58).*Script' -count=1
```

Self-host invariant:

```sh
go test ./tests -run TestSelfhostV01Scripts -count=1
```

Full project check:

```sh
go test ./... -count=1
```

## Dependencies

- Preserve v0.57 `embed` behavior for code without a `with` clause.
- Reuse existing compression support where practical for gzip output.
- Keep `net/http.Server.static` compatible with the current route registration
  model.

## Open Questions

None.
