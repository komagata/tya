# Feature: Documentation Generator Extensions

## Goal

Polish `tya doc` so it can produce machine-readable API output, document stdlib-style re-exports consistently, and report documentation quality problems with stable diagnostics while preserving the existing text and HTML generator behavior.

## Context

- `ROADMAP.md` tracks **Documentation generator extensions** as polish, not a v1.0.0 blocker unless docs publication requires it.
- Current `tya doc` supports:
  - `tya doc [paths...]` text output
  - `tya doc --html <out> [paths...]` static HTML output
  - default path `src/`
- Current extraction lives under `internal/doc` and extracts documented top-level functions, classes, modules, and interfaces from parsed `.tya` files.
- Current HTML generation warns about duplicate item filenames, but duplicate public definitions, orphan doc comments, and invalid Markdown bodies are not first-class diagnostics.
- ROADMAP asks for:
  - stdlib re-exports by following imports
  - `tya doc --json`
  - reuse of the public Tya self-introspection library when it exists
  - diagnostics for orphan doc comments, duplicate definitions, and unparseable Markdown bodies

## Behavior

- Add `tya doc --json [paths...]`.
  - Writes one JSON object to stdout.
  - Uses the same default path and file discovery rules as text/HTML output.
  - Exit status matches text/HTML behavior:
    - `0` when extraction succeeds and no documentation diagnostics are errors
    - `1` when documentation diagnostics with error severity are emitted but JSON can still be produced
    - `2` for CLI argument, path, I/O, lex, or parse failures that prevent reliable extraction
  - JSON includes:
    - `version`
    - `items`
    - `diagnostics`
  - Each item includes:
    - `name`
    - `kind`
    - `signature`
    - `raw_doc`
    - rendered text doc
    - source `path`
    - source `line`
    - `reexported_from` when applicable
  - Each diagnostic includes:
    - stable `code`
    - `severity`
    - `message`
    - `path`
    - `line`
    - `col`
- Extend extraction to follow public re-exports by imports.
  - When a module/package imports another module and re-exports public items through the package surface, `tya doc` includes those items in the documented package output.
  - Re-exported items preserve their original source location and set `reexported_from`.
  - Cyclic imports are detected and diagnosed without infinite recursion.
  - Duplicate re-exported names are diagnosed deterministically.
- Add documentation diagnostics:
  - orphan doc comments attached to no public item
  - duplicate public documentation names within the same output namespace
  - Markdown bodies that cannot be rendered safely
  - import cycles encountered while following re-exports
- Diagnostics are emitted consistently:
  - text output writes diagnostics to stderr
  - HTML output writes diagnostics to stderr and still writes pages when possible
  - JSON output includes diagnostics in the JSON payload and writes fatal CLI/I/O errors to stderr
- Existing text output remains backward compatible for successful inputs.
- Existing HTML output remains backward compatible for successful inputs, except duplicate pages must no longer silently use last-write-wins without a stable diagnostic.
- Reuse the public Tya self-introspection library if it exists at implementation time. If it does not exist yet, keep extraction in `internal/doc` but shape the data model so it can be moved later without changing CLI output.

## Scope

- Update CLI parsing and command handling in `cmd/tya/doc.go`.
- Update documentation extraction and model code in `internal/doc`.
- Add JSON output support under `internal/doc` or `cmd/tya`.
- Add stable documentation diagnostic types/codes.
- Add tests for:
  - `tya doc --json`
  - JSON shape and deterministic ordering
  - re-export following
  - import cycle diagnostics
  - orphan doc comment diagnostics
  - duplicate definition diagnostics
  - Markdown body diagnostics
  - existing text/HTML compatibility
- Update docs:
  - `docs/SPEC.md`
  - `docs/ja/spec.md`
  - `docs/GUIDE.md` if command examples change

## Out of Scope

- Redesigning the generated HTML site.
- Adding search, themes, client-side JavaScript, or versioned API hosting.
- Publishing generated docs to GitHub Pages.
- Making `tya doc` a type-aware semantic documentation engine beyond currently available parser/import information.
- Blocking v1.0.0 on this work unless docs publication later depends on it.
- Replacing current extraction with the self-introspection library before that library exists.

## Acceptance Criteria

- `tya doc --json src` emits valid JSON with stable ordering and the documented fields.
- `tya doc --json` uses the same default `src/` behavior as existing `tya doc`.
- Text and HTML outputs for existing clean fixtures remain unchanged.
- Re-exported public items are included once with `reexported_from` metadata.
- Import cycles while following re-exports produce a stable diagnostic and do not hang.
- Orphan doc comments produce stable diagnostics.
- Duplicate public names produce stable diagnostics in text, HTML, and JSON modes.
- Invalid or unsupported Markdown constructs produce stable diagnostics without unsafe HTML output.
- English and Japanese specs document `--json`, re-export behavior, and diagnostics.

## Verification

```sh
go test ./internal/doc -count=1
go test ./tests -run TestV51Scripts -count=1
go test ./... -count=1
```
