---
status: completed
goal_ready: false
---

# Feature: Coverage Tooling

## Goal

Finish the coverage tooling deferred from v0.30 so users can generate a
self-contained HTML report, filter measured source paths, enforce a minimum
coverage threshold in CI, and trust that reported lines map to their real Tya
source files.

## Context

Tya v0.30 already ships the measurement foundation:

- `tya test --cover [--profile FILE]` instruments tests and writes
  `.tya/coverage/profile`.
- `tya cover [report]` renders a human-readable table.
- `tya cover --format=json` renders the stable v0.30 JSON shape.
- `internal/cover` owns profile parsing, merging, summarizing, and report
  rendering.
- The C emitter skips stdlib, generated code, `.tya/packages/`, synthesized
  test-suite source, and empty source paths by default.

The v0.30 spec explicitly deferred richer reporting, include/exclude filters,
project-config coverage settings, `break` / `continue` source-position
coverage, and import-source path mapping. This feature implements those
deferred items without changing the profile format header or the default
coverage behavior for existing users.

## Behavior

- `tya cover html [--profile FILE] [-o FILE]` renders a self-contained HTML
  report.
- The default HTML output path is `.tya/coverage/index.html`.
- The HTML report must not depend on external CSS, JavaScript, fonts, images,
  CDNs, or network access.
- The HTML report includes:
  - total statement coverage,
  - per-file coverage rows sorted by source path,
  - source snippets for covered and missed coverable lines,
  - clear visual distinction between covered, missed, and non-coverable lines.
- `tya cover html` exits non-zero when the input profile cannot be read, when a
  referenced source file cannot be read, or when `-o` / `--output` is missing
  its value.
- `tya test --cover` accepts repeatable `--include GLOB` and `--exclude GLOB`
  options.
- `tya cover`, `tya cover report`, and `tya cover html` accept the same
  repeatable `--include GLOB` and `--exclude GLOB` options and apply them while
  reporting an existing profile.
- Filter patterns match normalized slash-separated project-relative paths.
- Built-in exclusions from v0.30 continue to apply before user filters.
- Include semantics:
  - when no include patterns are configured, every non-built-in-excluded source
    path is included unless excluded;
  - when one or more include patterns are configured, a source path must match at
    least one include pattern to be measured or reported.
- Exclude semantics:
  - a path matching any exclude pattern is omitted, even when it also matches an
    include pattern.
- Coverage filters can be read from `tya.toml`:

  ```toml
  [coverage]
  include = ["src/**", "tests/**"]
  exclude = ["tests/fixtures/**"]
  minimum = 80.0
  ```

- CLI filter flags are additive with `tya.toml` filters. CLI `--min` overrides
  `coverage.minimum`.
- `tya cover [report|html|--format=json] --min PERCENT` exits non-zero when the
  total statement coverage is below `PERCENT`.
- Minimum coverage checks use the same filtered profile view as the selected
  report.
- `tya test --cover --min PERCENT` runs tests, writes the merged profile, then
  applies the threshold to that profile before exiting.
- A failed minimum coverage check does not change individual test pass/fail
  output; it only changes the final command exit status and prints a concise
  coverage-threshold failure.
- Imported modules and classes are recorded under the real imported source path,
  not the synthesized test-suite entry path.
- Duplicate imports of the same source path merge into one per-file coverage
  row.
- `break` and `continue` statements become coverable once their AST nodes carry
  source positions. A line containing only an executed `break` or `continue`
  should count as covered; an unexecuted one should count as missed.
- Existing `tya cover --format=json` output remains backward compatible. New
  fields may be added, but existing top-level keys and meanings must not change.

## Scope

- CLI parsing in `cmd/tya` for coverage output, filters, output path, and
  minimum threshold flags.
- Project configuration loading for `[coverage]` settings in `tya.toml`.
- `internal/cover` support for filtered summaries, total coverage threshold
  checks, and HTML rendering.
- Coverage registry path normalization and import-source path mapping in the
  compiler/test runner path.
- AST, parser, and codegen updates needed for `break` / `continue` source
  positions and instrumentation.
- Focused tests for profile filtering, HTML rendering, threshold behavior, path
  mapping, and `break` / `continue` coverage.
- Documentation updates for the coverage CLI and CI usage.

## Out of Scope

- Branch, condition, MC/DC, or expression coverage.
- Differential or per-PR coverage reports.
- Hosted reports, upload APIs, or browser server mode.
- Changing the `# tya-cover 1` profile header unless required by an
  unavoidable compatibility bug.
- Measuring stdlib or C runtime coverage by default.
- Replacing the current statement-counter model.

## Acceptance Criteria

- `tya cover html` generates a readable self-contained HTML file at
  `.tya/coverage/index.html` by default.
- `tya cover html -o path/to/report.html` and
  `tya cover html --output path/to/report.html` write to the requested file.
- `tya test --cover --include 'src/**' --exclude 'tests/fixtures/**'` measures
  only matching user source paths after built-in exclusions.
- `tya cover --include 'src/**' --exclude 'tests/fixtures/**'` reports the same
  filtered totals from an existing profile.
- `[coverage]` settings in `tya.toml` are honored by both `tya test --cover` and
  `tya cover`.
- CLI filter flags combine with config filters deterministically, with excludes
  taking precedence.
- `tya cover --min 80` exits 0 at or above 80.0% total coverage and exits
  non-zero below 80.0%.
- `tya test --cover --min 80` preserves normal test execution, writes the
  profile, and then fails only when the filtered merged profile is below the
  threshold.
- Imported source files appear in coverage reports under their original source
  paths, not under synthesized entry paths.
- `break` and `continue` lines participate in statement coverage when they have
  source positions.
- Existing `tya cover`, `tya cover report`, and `tya cover --format=json`
  behavior remains compatible when no new flags or config are used.
- `go test ./... -count=1` passes, including the self-host fixed-point tests.

## Verification

```sh
go test ./internal/cover -count=1
go test ./internal/codegen -count=1
go test ./tests -run 'Test.*Cover|Test.*Coverage' -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
go run ./cmd/tya test --cover --include 'src/**' --exclude 'tests/fixtures/**'
go run ./cmd/tya cover html --profile .tya/coverage/profile -o .tya/coverage/index.html
go run ./cmd/tya cover --min 80
```

## Dependencies

- Builds on the v0.30 coverage profile and instrumentation foundation.
- Requires parser/AST source positions for `break` and `continue` before those
  statements can be instrumented.
- Should be implemented before relying on coverage as a release gate in CI.

## Open Questions

None.
