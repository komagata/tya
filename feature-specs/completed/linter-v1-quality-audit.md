# Feature: Linter v1 Quality Audit

## Goal

Lock down the existing `tya lint` behavior for v1.0.0 by making `TYAL0001` through `TYAL0008` fully specified, documented, tested, and consistent across the CLI, JSON/SARIF output, autofix, suppression comments, and LSP diagnostics.

## Context

- `ROADMAP.md` currently marks **Linter extensions** complete.
- The current linter surface includes:
  - `TYAL0001` unused local binding, autofixable
  - `TYAL0002` dead code after `return` or `raise`
  - `TYAL0003` redundant constant `if`, autofixable
  - `TYAL0004` deeply nested block
  - `TYAL0005` long function body
  - `TYAL0006` suspicious for-index binding order
  - `TYAL0007` unused function parameter
  - `TYAL0008` shadowed binding
- `cmd/tya/lint.go` implements `tya lint [--fix] [--format=text|json|sarif] [paths...]`.
- `cmd/tya/lint_metadata.go` assigns stable titles and doc URLs under `https://tya-lang.org/lint.html#tyal000N`.
- `internal/lsp/diagnostics.go` surfaces lint diagnostics in editors, but currently assembles LSP lint diagnostics separately from the CLI output enrichment path.
- `docs/SPEC.md` lists the current lint rules, while `docs/ja/spec.md` describes `tya lint` but does not enumerate the same rule table in Japanese.
- There is no active `docs/lint.md` or `docs/lint.html` page even though JSON/SARIF metadata points to `https://tya-lang.org/lint.html`.

## Behavior

- Treat `TYAL0001` through `TYAL0008` as the stable v1 lint rule set. This feature does not add new rule codes.
- For every rule, define and document:
  - rule code
  - title
  - plain-language description
  - trigger conditions
  - non-trigger examples
  - autofix availability
  - suppression examples
  - CLI text output shape
  - JSON fields
  - SARIF rule metadata
  - LSP diagnostic expectations
- Ensure all output surfaces agree on rule titles and documentation URLs:
  - text output remains `path:line:col: CODE message`
  - JSON output includes `code`, `title`, `message`, `path`, `line`, `col`, `autofixable`, and `doc_url`
  - SARIF output includes matching rule IDs, names, help URIs, and result locations
  - LSP diagnostics use the same code, warning severity, and human-readable message as the CLI rule contract
- Keep lint diagnostics as warnings, not language validity errors.
- Preserve `tya lint` exit codes:
  - `0` when no findings remain
  - `1` when findings remain
  - `2` for argument, parse, lex, or I/O errors
- Preserve existing suppression behavior:
  - `# tya-lint-ignore` suppresses all rules for the targeted line or next statement
  - `# tya-lint-ignore: CODE[, CODE...]` suppresses selected codes
  - `# tya-lint-ignore-file: CODE[, CODE...]` suppresses selected codes for the whole file
  - omitting codes in file-scope ignore suppresses all rules for the whole file
- Preserve autofix behavior:
  - `TYAL0001` removes the complete unused binding, including multi-line binding bodies where applicable
  - `TYAL0003` unwraps redundant constant `if` blocks
  - `--fix` does not apply non-declared fixes for `TYAL0002`, `TYAL0004`, `TYAL0005`, `TYAL0006`, `TYAL0007`, or `TYAL0008`
- Add or update a public lint documentation page at `docs/lint.md` or an equivalent generated-page source so the existing `https://tya-lang.org/lint.html#tyal000N` URLs resolve.
- Keep English and Japanese documentation synchronized for the rule list and user-facing lint behavior.

## Scope

- Audit and update lint implementation only where needed to make the existing behavior match the v1 contract:
  - `cmd/tya/lint*.go`
  - `internal/checker/lint_rules.go`
  - `internal/checker/unused.go`
  - `internal/checker/autofix.go`
  - `internal/lsp/diagnostics.go`
  - `internal/lsp/code_actions.go`
- Add focused tests for any rule/output/suppression/autofix case that is not already covered:
  - `tests/testdata/v55_lint/*.txtar`
  - LSP diagnostics/code-action tests under `tests/`
  - unit tests under `internal/checker` or `cmd/tya` if a behavior is easier to verify without CLI fixtures
- Update docs:
  - `docs/SPEC.md`
  - `docs/ja/spec.md`
  - `docs/lint.md` or equivalent
  - README only if its lint examples become inaccurate

## Out of Scope

- Adding new lint rules beyond `TYAL0001` through `TYAL0008`.
- Adding `tya.toml [lint]` configuration.
- Adding `--max-warnings`, baselines, diff-only lint, or GitHub Actions annotations.
- Changing lint diagnostics from warnings into compile/check errors.
- Changing the public code numbers or renaming existing rule codes.
- Publishing a release.

## Acceptance Criteria

- Each of `TYAL0001` through `TYAL0008` has at least one active test that demonstrates when it fires.
- Each rule has at least one active test or documented fixture demonstrating a non-trigger case when the boundary is non-obvious.
- `TYAL0001` and `TYAL0003` autofix behavior is tested, including that repeated `tya lint --fix` runs become clean.
- Suppression comments are tested for line, next-statement, wildcard, code-specific, and file-scope behavior.
- JSON output is tested for all required fields and stable rule metadata.
- SARIF output is tested for rule IDs, names, help URIs, and result locations.
- LSP diagnostics are tested for representative lint findings and use warning severity with matching codes.
- The public lint documentation page exists and contains anchors for `#tyal0001` through `#tyal0008`.
- `docs/SPEC.md` and `docs/ja/spec.md` contain matching current-rule lists and explain that lint failures do not make a Tya program invalid.
- Existing `tya lint` CLI usage remains backward compatible.

## Verification

```sh
go test ./internal/checker -count=1
go test ./internal/lsp -count=1
go test ./tests -run 'TestV49Scripts|TestV50Scripts|TestV55Scripts|TestLSP' -count=1
go test ./... -count=1
```
