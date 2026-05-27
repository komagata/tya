# Feature: tya doc member and parameter metadata

## Goal

Extend `tya doc` so public class variables, class constants, methods, static methods, and interface methods can expose structured type and parameter hint metadata from doc comments, without adding language-level type annotations or static checking.

## Context

`tya doc` currently extracts leading `#` comments for top-level bindings, public class constants, public methods, static methods, and interface methods. Public class instance variables are not emitted as documentation items, so comments such as `Csv.rows` do not appear in generated docs. Method signatures include parameter names and default values, but there is no structured way to document parameter types, return types, field types, option hints, or per-parameter descriptions.

Tya remains dynamically typed. `docs/static-typing-discussion.md` is not current language authority, so this feature must not introduce checker-enforced type annotations.

Relevant files:

- `internal/doc/extract.go`
- `internal/doc/html.go`
- `internal/doc/text.go`
- `internal/doc/json.go`
- `internal/doc/extract_test.go`
- `internal/doc/stdlib_test.go`
- `tests/testdata/v51_doc/`
- `docs/SPEC.md`
- stdlib files such as `lib/csv/Csv.tya`

## Behavior

- `tya doc` must document public class instance variables in addition to existing public class constants and methods.
- Private class variables, constants, methods, and helper members remain excluded from docs.
- Doc comment metadata is written with tag lines inside the leading `#` comment block.
- Supported tags:
  - `@type <type-hint>` for variables and constants.
  - `@param <name> <type-hint> <description>` for method/function/interface parameters.
  - `@return <type-hint> <description>` for return values.
  - `@option <param>.<key> <type-hint> <description>` for dictionary option keys.
- Tags are documentation metadata only. They must not affect parsing, checking, codegen, runtime behavior, or formatter output.
- Non-tag comment lines remain the main Markdown description.
- Tags must be parsed into structured metadata and removed from the rendered main description body.
- Unknown `@` tags in doc comments should produce a warning diagnostic but should not block output.
- `@param` tags whose parameter name is not present in the documented callable should produce a warning diagnostic.
- Duplicate tags for the same `@param`, `@option`, `@type`, or `@return` target should produce a warning diagnostic.
- Type hints are free-form display strings. `Dict?`, `Array<Dict>`, `String`, and similar strings must be accepted without validation.
- Output support:
  - text output must show type, params, options, and return metadata.
  - HTML output must render metadata near the signature.
  - JSON output must include structured metadata fields.

Example:

```tya
# Csv.options stores CSV parsing and stringifying options.
# @type Dict
# @option options.separator String field separator
# @option options.header Boolean whether the first row is a header
options: { separator: Self.SEPARATOR, header: false }

# Csv.initialize applies optional CSV options.
# @param options Dict? CSV parsing and stringifying options.
# @option options.separator String field separator
# @option options.header Boolean whether the first row is a header
initialize: options = nil ->
  self.apply_options(options)
```

Expected docs:

- `Csv.options` appears as a variable item.
- `Csv.options` displays type `Dict` and option metadata.
- `Csv.initialize(options = nil)` displays the `options` parameter with type `Dict?`, description, and option metadata.

## Scope

- Extend doc extraction data structures to carry metadata:
  - item type hint
  - parameters
  - return value
  - dictionary option entries
- Add doc items for public class instance variables.
- Update text, HTML, and JSON renderers.
- Add diagnostics for malformed, unknown, duplicate, or mismatched metadata tags.
- Add CLI/testscript coverage under `tests/testdata/v51_doc/`.
- Add unit tests in `internal/doc`.
- Update `docs/SPEC.md` toolchain documentation for `tya doc` metadata tags.
- Update representative stdlib docs comments, at minimum `lib/csv/Csv.tya`, to exercise variable and parameter metadata.

## Out of Scope

- Do not add language-level type annotations.
- Do not enforce type hints in checker, codegen, runtime, tests, or LSP.
- Do not add a full type grammar. Type hints are display strings.
- Do not document private members.
- Do not require every stdlib item to have metadata in this feature.
- Do not change Markdown parsing beyond recognizing metadata tag lines in leading doc comments.

## Acceptance Criteria

- `tya doc lib/csv/Csv.tya` includes `Csv.rows`, `Csv.headers`, and `Csv.options` as documented variable items.
- Public variable docs include normal descriptions and `@type` metadata.
- Method docs include `@param`, `@option`, and `@return` metadata where provided.
- `tya doc --html` renders the same metadata.
- `tya doc --json` exposes the same metadata in stable structured fields.
- Unknown metadata tags and mismatched parameter names produce stable warnings.
- Existing `tya doc` output without tags remains compatible except for newly documented public variables.
- Private members remain excluded.

## Verification

```sh
go test ./internal/doc -count=1
go test ./tests -run TestV51DocScripts -count=1
go test ./... -count=1
```
