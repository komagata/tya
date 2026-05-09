# Tya v0.29 Specification — Diagnostics Foundation

Tya v0.29 lays the foundation for Elm-grade diagnostics across the
toolchain. The language itself does not change in this release: no new
syntax, no new builtins, no new stdlib behavior. v0.29 introduces the
common diagnostic model, the human and JSON renderers, the color and
format CLI flags, the stable error-code namespace, and migrates the
**checker's strict diagnostics (v0.28 checks)** to the new pipeline as
the first concrete user of it.

The remaining stages (lexer, parser, codegen, runner, fmt) keep their
current error strings in v0.29 and will be migrated to the new model in
later minor releases. v0.29 is intentionally a foundation release.

## Goals

- A single in-process diagnostic model (`internal/diag`) that every
  stage can target over time.
- A human renderer that produces an Elm-style banner with source
  snippet, caret underline, hints, notes, and an error code.
- A JSON (NDJSON) renderer behind `--format=json`, with a stable shape
  that future tooling and an eventual LSP can consume.
- `--color=auto|always|never` and `NO_COLOR` honored consistently.
- A stable error-code namespace (`TYA-Exxxx` / `TYA-Wxxxx`) with a
  reference document under `docs/v0.29/CODES.md`.
- The v0.28 strict-check diagnostics (shadowing, unused import, unused
  argument, unused private top-level) are emitted through the new
  pipeline with banners, snippets, hints, and codes.
- `tya check` runs the strict pass to completion and reports every
  strict diagnostic it finds in one invocation, instead of stopping at
  the first.

## Non-Goals (v0.29)

- Migrating lexer, parser, codegen, runner, or fmt errors to the new
  model. They keep their current stderr strings in v0.29.
- Multi-error parsing.
- Did-you-mean suggestions (deferred — needs scope/type info plumbing
  not yet in `internal/diag`).
- LSP server, watch mode, persistent build cache.
- Source maps from `.tya` to emitted C.
- Localization or hyperlink rendering (OSC 8).
- Auto-applied fix-its.

## Diagnostic Model

A new package `internal/diag` provides the core types:

```go
package diag

type Severity int
const (
    Error Severity = iota
    Warning
)

type Pos struct{ Line, Col int }

type Region struct {
    File       string
    Start, End Pos
}

type Diagnostic struct {
    Severity Severity
    Code     string   // e.g. "TYA-E0301"
    Title    string   // short noun phrase
    Message  string   // one-sentence problem statement
    Primary  Region
    Hints    []string
    Notes    []string
    Source   string   // stage: "checker", etc.
}
```

Lines and columns are 1-based. End is exclusive. v0.29 ships only
`Primary`; secondary regions are reserved for a later release.

`internal/diag` also exposes:

- `SourceMap` — caches file bytes and line offsets so renderers can
  draw snippets without each stage re-reading files.
- `Render(diags []Diagnostic, sm *SourceMap, opts RenderOptions) string`
  — human renderer.
- `RenderJSON(diags []Diagnostic) string` — NDJSON renderer.
- `RenderOptions{ Color ColorMode, TermWidth int }` and
  `ColorMode = Auto | Always | Never`.

## Human Format

Each diagnostic renders as:

```
-- TITLE -------------------------------------- file:line:col

<one-sentence problem statement>

  12 |   foo = bar + baz
                ^^^

Hint: <actionable suggestion>

Note: <optional context>

(TYA-E0301)
```

Rules:

1. Banner: `-- ` + uppercased TITLE + space + dashes padded to col 70 +
   space + `file:line:col`. If terminal width is narrower than 70,
   dashes are truncated; the location is never truncated.
2. Blank line separates banner from message.
3. Snippet: the line containing the primary region's start is shown
   with a right-aligned line-number gutter and ` | ` separator. The
   underline row uses the same gutter width filled with spaces, then
   ` | ` replaced with `   `, then `^` for every column in the primary
   region (single-character regions get one `^`).
4. Long source lines are not wrapped.
5. Hints render as separate paragraphs prefixed with `Hint: `.
6. Notes render as separate paragraphs prefixed with `Note: `.
7. Error code on its own line, parenthesized, last.
8. Diagnostics are separated by a single blank line.
9. After all diagnostics, the renderer emits a final summary line:
   `Found N error(s), M warning(s).` Skipped when there are no
   diagnostics.

### Color

When color is enabled, ANSI codes are applied:

- Banner dashes and TITLE: bold, severity color (red error, yellow
  warning).
- File:line:col: cyan.
- Underline carets: severity color.
- `Hint:` prefix: bold blue. `Note:` prefix: bold dim.
- Error code: dim.

`--color` precedence:

| Value    | Behavior                                              |
|----------|-------------------------------------------------------|
| `auto`   | Color when stderr is a TTY and `NO_COLOR` is unset.   |
| `always` | Color regardless.                                     |
| `never`  | No color.                                             |

`auto` is the default. `NO_COLOR=1` in the environment forces `never`
unless `--color=always` is passed (the explicit flag wins).

## JSON Format

`--format=json` switches stderr to NDJSON. One diagnostic per line:

```json
{"severity":"error","code":"TYA-E0301","title":"Shadowed binding","message":"This binding shadows an outer name.","primary":{"file":"main.tya","start":{"line":12,"col":3},"end":{"line":12,"col":7}},"hints":["Rename the inner binding, or prefix it with `_`."],"notes":[],"source":"checker"}
```

After all diagnostics, a single summary object:

```json
{"summary":{"errors":1,"warnings":0}}
```

Stdout is unchanged: `tya run` still streams program stdout, `tya
emit-c` still writes C, etc. Only diagnostic stderr switches format.

`--format=json` implies `--color=never`.

The JSON shape is stable from v0.29 onward. New optional fields may be
added; existing fields will not be removed or change type without a
major version bump.

## CLI Surface

Two new flags are accepted by every subcommand that may emit
diagnostics (`run`, `build`, `check`, `test`, `fmt`, `emit-c`):

```
--format=human|json     Output format for diagnostics. Default: human.
--color=auto|always|never  Color in human format. Default: auto.
```

In v0.29 these flags only affect *checker strict* diagnostics
(the diagnostics migrated in this release). Lexer / parser / codegen /
runner errors continue to be printed as plain strings; they will be
migrated in a later release. The flags are accepted everywhere now so
that users do not have to change scripts when later migrations land.

`tya check` behavior change for the strict pass: it collects every
strict diagnostic in the file and reports them all in one run, instead
of stopping at the first. The exit code is non-zero if any strict
diagnostic is an error.

`tya run`, `tya build`, `tya test`, `tya emit-c`, and `tya fmt` keep
fail-fast semantics for non-strict errors. When the strict pass runs
in those subcommands, it still surfaces only the first strict
diagnostic (preserving today's behavior outside of `tya check`); this
is to keep error output minimal in the build path. `tya check` is the
designated multi-error reporter.

## Error Codes

Every diagnostic carries a stable code of the form `TYA-Xnnnn`:

- `X` is `E` for errors and `W` for warnings.
- `nnnn` is a zero-padded four-digit number.

Ranges are pre-allocated by stage so future migrations have room:

| Range           | Stage    |
|-----------------|----------|
| `E0001`–`E0099` | lexer    |
| `E0100`–`E0299` | parser   |
| `E0300`–`E0599` | checker  |
| `E0600`–`E0799` | codegen  |
| `E0800`–`E0899` | runner   |
| `E0900`–`E0999` | fmt      |
| `W1000`+        | warnings |

v0.29 assigns:

| Code         | Title                               | Source  |
|--------------|-------------------------------------|---------|
| `TYA-E0301`  | Shadowed binding                    | checker |
| `TYA-E0302`  | Unused import                       | checker |
| `TYA-E0303`  | Unused argument                     | checker |
| `TYA-E0304`  | Unused private top-level definition | checker |
| `TYA-E0305`  | Duplicate parameter                 | checker |
| `TYA-E0306`  | Duplicate binding name in pattern   | checker |

Codes, once assigned, never change meaning. Retired codes stay listed
in `docs/v0.29/CODES.md` marked *retired*. The reference doc lists
every code with a one-line description and a minimal example.

## Diagnostics Migrated in v0.29

The strict checks introduced in v0.28 are emitted through the new
pipeline. Each gains a banner, snippet, code, and at least one hint
where applicable.

### TYA-E0301 — Shadowed binding

```
-- SHADOWED BINDING --------------------------- main.tya:5:3

This binding shadows the outer name `count`.

   5 |   count = 0
         ^^^^^

Hint: Rename the inner binding, or prefix it with `_` to mark it as
intentional.

(TYA-E0301)
```

### TYA-E0302 — Unused import

```
-- UNUSED IMPORT ------------------------------ main.tya:1:8

The module `string` is imported but never used.

   1 | import string
              ^^^^^^

Hint: Remove the import, or reference the module somewhere in this
file.

(TYA-E0302)
```

### TYA-E0303 — Unused argument

```
-- UNUSED ARGUMENT ---------------------------- main.tya:3:9

The argument `value` is never used in the body of this function.

   3 | greet = value -> "hello"
               ^^^^^

Hint: Rename it to `_` or prefix it with `_` (e.g. `_value`) to mark
it as intentional.

(TYA-E0303)
```

### TYA-E0304 — Unused private top-level definition

```
-- UNUSED PRIVATE DEFINITION ------------------ main.tya:1:1

The private top-level definition `_helper` is never referenced in
this file.

   1 | _helper = 42
       ^^^^^^^

Hint: Remove the definition, or reference it elsewhere in this file.

(TYA-E0304)
```

### TYA-E0305 — Duplicate parameter

```
-- DUPLICATE PARAMETER ------------------------ main.tya:1:13

The parameter `x` appears more than once in this function.

   1 | f = x, x -> x
                  ^

Hint: Rename one of the parameters.

(TYA-E0305)
```

### TYA-E0306 — Duplicate binding name in pattern

```
-- DUPLICATE BINDING IN PATTERN ---------------- main.tya:2:11

The name `name` is bound more than once in this pattern.

   2 |   case [name, name]
                      ^^^^

Hint: Rename one of the bindings, or compare with an equality check
in a guard.

(TYA-E0306)
```

## Source Snippet Loading

`internal/diag.SourceMap` exposes:

```go
type SourceMap struct{ /* ... */ }

func (sm *SourceMap) Add(file string, src []byte)
func (sm *SourceMap) Line(file string, n int) (string, bool)
```

The CLI populates the source map from the entry program and any
imported modules before invoking the renderer. For files missing from
the map, the renderer omits the snippet and prints `(snippet
unavailable)` on a single line in place of the snippet block. Banner,
message, hints, notes, and code still render.

## Testing Strategy

1. **Unit tests** for `internal/diag` cover human rendering (with and
   without color), JSON rendering (line-by-line + summary), color mode
   resolution including `NO_COLOR`, and source-map snippet lookup.
2. **Golden tests** under `tests/diagnostics/` exercise each migrated
   checker code with a small `.tya` source and `.golden.txt` /
   `.golden.json` expected outputs.
3. **Code inventory test** asserts that every code registered in code
   matches an entry in `docs/v0.29/CODES.md` and vice versa.
4. **Self-host invariant** (`TestSelfhostV01Scripts`) continues to
   pass.
5. **Existing CLI script tests** keep their current expectations
   because `tya run` / `tya build` / `tya test` still surface only the
   first strict diagnostic. Only `tya check` golden tests assert the
   new banner format.

## Acceptance Criteria

A v0.29 build is acceptable when:

1. `internal/diag` exists and exports the documented types and
   renderers.
2. `--format=human|json` and `--color=auto|always|never` are accepted
   by every subcommand and behave as specified.
3. `NO_COLOR` is honored.
4. The six checker codes (`TYA-E0301`–`TYA-E0306`) render through the
   new pipeline with banners, snippets, hints, and codes.
5. `tya check` reports multiple strict diagnostics in one run.
6. `docs/v0.29/CODES.md` is present and matches the registered codes.
7. `go test ./... -count=1` passes, including the self-host invariant.
