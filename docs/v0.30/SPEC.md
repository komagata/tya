---
layout: doc
title: Spec
permalink: /v0.30/spec/
---

# Tya v0.30 Specification — Test Coverage (Foundation)

Tya v0.30 ships the **measurement foundation** for line coverage:
per-statement counters in generated C, an on-disk profile written at
process exit, and a minimal `tya cover` subcommand that reports the
profile as text or JSON. Richer reporting (HTML), fine-grained filters
(`--include` / `--exclude`), `Tyafile` configuration, and coverage of
control-flow tail statements (`break` / `continue`) are deferred to
later v0.30.x patches and v0.31.

The language itself does not change. Coverage is opt-in: when
`--cover` is not passed, generated C and binary behavior are
bit-identical to v0.29.

## Goals

- `tya test --cover` runs the test suite with instrumentation and
  writes a coverage profile to `.tya/coverage/profile`.
- `tya cover report` prints a human-readable per-file summary table
  for that profile.
- `tya cover --format=json` emits machine-readable output.
- Coverage maps back to original Tya source lines.
- The stdlib, generated code, and `.tya/packages/` are excluded by
  default.
- The `selfhost/v01/compiler.tya` fixed point continues to pass.

## Non-Goals (v0.30.0)

- Branch / condition / MC/DC coverage. Statement-level only.
- Mutation testing.
- Cross-process / cross-binary aggregation across multiple
  `tya test --cover` invocations.
- Differential / per-PR coverage diff.
- HTML report (`tya cover html`) — deferred.
- `--include` / `--exclude` flags — deferred.
- `Tyafile` `coverage:` section — deferred.
- Coverage of `break` / `continue` statements (their AST nodes carry
  no source position in v0.29 and instrumenting them would require AST
  surgery beyond v0.30.0's scope). Other `ast.Stmt` shapes
  (`AssignStmt`, `ImportStmt`, `IfStmt`, `WhileStmt`, `ForInStmt`,
  `ReturnStmt`, `RaiseStmt`, `MatchStmt`, `TryCatchStmt`, `ModuleDecl`,
  `ClassDecl`, `InterfaceDecl`, and `ExprStmt` whose inner expression
  carries a position) are instrumented.
- Coverage of stdlib code or the C runtime.

## Granularity

v0.30 measures **statement coverage**, projected to **line coverage**
in reports. Concretely:

- Every instrumented `ast.Stmt` whose source position falls in a user
  file (not stdlib, not generated synthesis) is associated with a
  counter.
- Each counter is keyed by `(file, line, col)` of the statement's
  start position. Multiple statements on the same line each get their
  own counter; reports aggregate by line.
- A line is *covered* when any counter that overlaps it has been
  incremented at least once.
- A line is *coverable* when at least one counter overlaps it.
- A line is *uncovered* when it is coverable but no overlapping
  counter fired.
- Comment-only lines, blank lines, lines containing only a `break` or
  `continue`, and lines that contain only block delimiters are *not
  coverable* and are excluded from the denominator.

For block statements (`if`, `while`, `for`, `match`, `try`), the
counter records entry of the *header* line. Each nested body
statement has its own counter so that empty branches do not inflate
"covered" counts.

## CLI Surface

```
tya test --cover [path]
tya test --cover --profile FILE [path]
tya cover [report]               # default subcommand
tya cover --format=json
tya cover --profile FILE
```

### tya test --cover

Behaves exactly like `tya test` (v0.4) except:

1. The C emitter runs in instrumented mode.
2. Each compiled test binary writes a profile fragment to
   `${TYA_COVERAGE_DIR:-.tya/coverage}/fragments/<binary>.cov` on
   normal or abnormal exit. The path is communicated to the binary
   via the `TYA_COVERAGE_FRAGMENT` environment variable, which `tya
   test --cover` sets per binary.
3. After all test binaries finish, `tya test --cover` merges fragments
   into a single `.tya/coverage/profile` and removes the
   `fragments/` directory.
4. Exit code follows `tya test` semantics. Coverage instrumentation
   does not affect pass/fail.

The profile path can be overridden:

```
tya test --cover --profile path/to/profile.cov
```

`TYA_COVERAGE_DIR` is honored and overrides the default
`.tya/coverage` directory. `--profile` overrides both.

### tya cover

Reads a profile (default `.tya/coverage/profile`) and renders it.

`tya cover` and `tya cover report` are equivalent and produce the
human-readable table:

```
File                          Stmts   Hit  Missed  Coverage
src/string.tya                   42    40       2    95.2%
tests/string_test.tya            12    12       0   100.0%
------------------------------------------------------------
Total                            54    52       2    96.3%
```

Sorted by file path. The `Total` row sums numerators and
denominators; it is not an average of percentages.

`tya cover --format=json` emits a single JSON document on stdout:

```json
{
  "tool": "tya",
  "version": "0.30.0",
  "format": 1,
  "profile": ".tya/coverage/profile",
  "files": [
    {
      "path": "src/string.tya",
      "statements": 42,
      "hits": 40,
      "lines": [
        {"line": 3, "hits": 1, "coverable": true},
        {"line": 4, "hits": 0, "coverable": true}
      ]
    }
  ],
  "totals": {"statements": 54, "hits": 52, "files": 2}
}
```

The shape is stable from v0.30 onward.

`--profile FILE` overrides the default profile path on either form.

## Profile File Format

The on-disk profile is a single text file with one record per line:

```
# tya-cover 1
F <id> <path>
S <id> <file_id> <line> <col>
H <stmt_id> <count>
```

Rules:

1. The first line is the literal header `# tya-cover 1`. The trailing
   `1` is the format version. Future format-incompatible changes bump
   it.
2. `F` records register a file. `<id>` is a small integer assigned in
   first-seen order. `<path>` is the file path with spaces (`%20`)
   and percent (`%25`) percent-encoded; no other characters are
   encoded.
3. `S` records register a statement counter. `<id>` is a small
   integer unique within the profile. `<file_id>` references an `F`
   record. Lines and columns are 1-based.
4. `H` records carry the hit count for a statement. A statement
   absent from `H` records counts as zero.
5. Records may appear in any order after the header. `tya cover`
   normalizes and deduplicates on load.

Fragment files use the same format. Merging unions records and sums
`H` counts by statement id. Within a single `tya test --cover` run,
all fragments share file and statement ids (because all binaries are
emitted by the same instrumented build); the merger therefore does
not need to deduplicate on `(file, line, col)`. Re-running `tya test
--cover` overwrites the merged profile.

## Instrumentation

The C emitter gains an internal coverage option plumbed via a new
exported entry point:

```go
// EmitCWithCoverage is like EmitCWithPath but emits per-statement
// counter increments and exposes the registry via the returned
// *CoverageRegistry. When opt is nil, this is identical to
// EmitCWithPath.
func EmitCWithCoverage(prog *ast.Program, sourcePath string, opt *CoverageOptions) (string, *CoverageRegistry, error)
```

When coverage is enabled:

1. A static array `tya_cov_counts[N]` is emitted at the top of the
   compiled program, where `N` is the number of registered
   statements.
2. Before each instrumented statement, the emitter prepends
   `tya_cov_inc(<id>);` (a runtime function defined in
   `runtime/tya_cover.c`). The increment is unconditional.
3. `main` calls `tya_cov_init(<N>, "<binary_path>")` first. At
   process exit, an `atexit`-registered writer reads
   `TYA_COVERAGE_FRAGMENT` from the environment and writes the
   fragment file. When the variable is unset, the writer is a no-op.

Programs built without `--cover` do not reference `tya_cov_*`
symbols; the runtime file is linked but generates no I/O.

`CoverageRegistry` exposes the registered file/statement table so
`tya test --cover` can write the matching `F` and `S` records into
fragments alongside the runtime-emitted `H` records.

### Runtime fragment shape

The runtime writes only `H` records to the fragment file:

```
# tya-cover 1
H 0 12
H 1 0
H 2 7
```

The runner combines this with the registry's `F` and `S` records
when merging fragments into the final profile. This keeps the
runtime small and avoids passing source paths through the C side.

### Excluded files

The emitter skips counters for statements whose file path is:

- under `stdlib/` of the running tya binary,
- under `.tya/packages/` of the project root,
- the synthesized test-suite source produced by `tya test`,
- empty (not associated with a real file).

Excluded statements still emit their normal C; they simply do not get
a `tya_cov_inc` prefix and do not consume a counter id.

## Self-Host Invariant

`selfhost/v01/compiler.tya` is compiled and run without `--cover` by
the existing `TestSelfhostV01Scripts`. v0.30 does not instrument the
self-host pipeline by default and does not change codegen output for
non-coverage builds. The self-host test continues to pass unchanged.

## Testing Strategy

1. **Unit tests** under `internal/cover` exercise round-trip of the
   on-disk profile format, including fragment merge.
2. **Codegen unit tests** verify that instrumented programs emit
   counter increments before each instrumented stmt kind, and that
   non-instrumented builds emit no `tya_cov_*` references.
3. **CLI golden test** (`tests/testdata/v30/coverage.txtar`):
   - `tya test --cover tests/probe_test.tya` exits 0 and creates
     `.tya/coverage/profile`.
   - `tya cover` prints a table containing the file path and a
     percentage.
   - `tya cover --format=json` produces parseable JSON with the
     documented top-level keys.
4. **Self-host smoke** test compiles the self-host compiler under
   `--cover` (asserts only that compilation succeeds, not specific
   coverage numbers).
5. **Default test suite** (`go test ./...`) remains green.

## Acceptance Criteria

A v0.30.0 build is acceptable when:

1. `tya test --cover` writes `.tya/coverage/profile` after a
   successful run.
2. `tya cover` and `tya cover --format=json` consume that profile and
   produce the documented outputs.
3. The profile file matches the documented format (`# tya-cover 1`
   header + `F` / `S` / `H` records).
4. Default exclusions (stdlib, `.tya/packages/`, synthesized test
   suite source, empty file) hold.
5. Non-coverage builds produce identical C and identical binary
   behavior to v0.29.
6. `go test ./... -count=1` passes, including the self-host
   invariant.

## Deferred to v0.30.x / v0.31

- `tya cover html` self-contained HTML report.
- `--include` / `--exclude` glob filters.
- `Tyafile` `coverage:` section.
- AST surgery to give `BreakStmt` and `ContinueStmt` source positions
  so they participate in coverage.
- Coverage diff between profiles.
- Branch / condition / MC/DC coverage.
- Editor / LSP gutter integration.
