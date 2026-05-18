# Selfhost Coverage Manifest

Status: v1.0.0 release-gate manifest. This file maps documented language
features from `docs/SPEC.md` to self-host compiler coverage. A row marked
`unsupported` in any required column blocks the v1.0.0 release until resolved.

| SPEC feature | Lexer | Parser | AST | Checker | C emitter | Runtime | Fixtures | v1 release gate |
|---|---|---|---|---|---|---|---|---|
| Indentation, comments, identifiers, and literals | supported | supported | supported | supported | supported | supported | `tests/testdata/v01_selfhost/minimal_compile.txtar`; `tests/testdata/v02_selfhost/parser_current_surface.txtar` | pass |
| Bindings, assignment, constants, and block scope | supported | supported | supported | supported | supported | supported | `tests/testdata/v01_selfhost/current_v04.txtar`; `tests/testdata/v02_selfhost/checker_current_surface.txtar` | pass |
| Functions, calls, returns, defaults, and multiple returns | supported | supported | supported | supported | supported | supported | `tests/testdata/v01_selfhost/current_v05.txtar`; `tests/testdata/v02_selfhost/emitter_latest_surface.txtar` | pass |
| If, while, for, match, break, and continue | supported | supported | supported | supported | supported | supported | `tests/testdata/v01_selfhost/current_v06.txtar`; `tests/testdata/v02_selfhost/parser_current_surface.txtar` | pass |
| Arrays, dictionaries, strings, bytes, and indexing | supported | supported | supported | supported | supported | supported | `tests/testdata/v01_selfhost/current_v07.txtar`; `tests/testdata/v02_selfhost/checker_current_surface.txtar` | pass |
| Classes, objects, interfaces, inheritance, `self`, and `super` | supported | supported | supported | supported | supported | supported | `tests/testdata/v02_selfhost/current_v05.txtar`; `tests/testdata/v02_selfhost/current_v07.txtar` | pass |
| Imports, packages, class files, and script entries | supported | supported | supported | supported | supported | supported | `tests/testdata/v01_selfhost/module_rules.txtar`; `tests/testdata/v02_selfhost/parser_current_surface.txtar` | pass |
| `raise`, `try`, `catch`, `finally`, and structured errors | supported | supported | supported | supported | supported | supported | `tests/testdata/v01_selfhost/minimal_compile.txtar`; `tests/testdata/v65_strict/v1_unified_error_handling.txtar` | pass |
| `spawn`, `await`, `scope`, channels, and `select` | supported | supported | supported | partial | partial | supported | `tests/testdata/v02_selfhost/parser_current_surface.txtar`; `tests/testdata/v02_selfhost/emitter_latest_surface.txtar` | gap |
| Package manager, external native packages, WASM, LSP, formatter, linter, docs, and release tooling | not applicable | not applicable | not applicable | not applicable | not applicable | supported by Go toolchain | `tests/testdata/v65_strict/strict_semantics.txtar`; `tests/release_package_test.go`; `tests/bootstrap_no_go_test.go` | gap |

The v1.0.0 release gate fails while any documented core language feature has a
`gap` release-gate value. Tooling that is intentionally outside the self-hosted
compiler is tracked separately from the language compiler surface.
