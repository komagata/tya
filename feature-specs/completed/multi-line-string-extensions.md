---
status: completed
goal_ready: false
---

# Feature: Multi-Line String Extensions

## Goal

Extend Tya's existing multi-line string support with explicit heredoc markers
and language tags so large SQL, HTML, JSON, shell, and template snippets can be
written clearly without changing their runtime value semantics.

## Context

Tya already supports:

- `"""..."""` interpolating multi-line strings;
- `r"""..."""` raw multi-line strings;
- `b"""..."""` bytes multi-line literals;
- indentation normalization based on the closing triple quote.

The remaining roadmap item is narrower: add heredoc-style markers and
language-tagged interpolation specifiers. These should build on the existing
string pipeline, not introduce a new runtime string type.

## Behavior

- Preserve all existing string literal behavior.
- Add heredoc string literals:

  ```tya
  query = <<<SQL
    select *
    from users
    where name = {name}
    SQL
  ```

- The opening delimiter is `<<<MARKER`.
- `MARKER` must match `[A-Z][A-Z0-9_]*`.
- The closing delimiter is a line containing optional spaces, the same marker,
  and nothing else.
- The closing delimiter's indentation defines the baseline strip, matching the
  existing `"""..."""` indentation rule.
- Heredoc body newlines are part of the value.
- Heredoc strings interpolate by default, using the same `{expr}`, `{{`, `}}`,
  and escape behavior as `"""..."""`.
- Add raw heredoc strings:

  ```tya
  regex = r<<<REGEX
    \d+ files in {dir}
    REGEX
  ```

- Raw heredocs do not process escapes or interpolation.
- Add bytes heredoc strings:

  ```tya
  payload = b<<<BYTES
    hello\x0a
    BYTES
  ```

- Bytes heredocs use the existing bytes escape rules after indentation
  normalization.
- Add language-tagged triple strings:

  ```tya
  query = sql"""
    select *
    from users
    where name = {name}
    """
  ```

- Add language-tagged heredocs:

  ```tya
  page = html<<<HTML
    <h1>{title}</h1>
    HTML
  ```

- A language tag must match `[a-z][a-z0-9_]*`.
- Language tags do not change runtime value semantics.
- Language tags are preserved in tokens and AST nodes for tooling, syntax
  highlighting, formatter output, docs, and future lint rules.
- Supported combinations:
  - `tag"""..."""`
  - `tag<<<MARKER ... MARKER`
  - `rtag"""..."""` is not supported; use raw heredoc/triple strings without a
    language tag in the first version.
  - `btag"""..."""` is not supported.
- Heredoc markers solve the main case where the body needs to contain literal
  `"""` without escaping or string splitting.
- Formatter preserves the chosen literal form:
  - it does not rewrite heredocs into triple strings;
  - it does not invent language tags;
  - it normalizes indentation according to canonical string rules.
- `tya format` may continue rewriting long ordinary strings to `"""..."""`;
  it must not rewrite them to heredocs automatically in the first version.
- Unterminated heredocs produce a structured lexer diagnostic.
- Closing marker mismatch produces a structured lexer diagnostic.
- Invalid marker or tag names produce structured diagnostics.

## Scope

- Lexer support for heredoc delimiters and language-tag prefixes.
- Token metadata for string kind and optional language tag.
- AST support for preserving string literal language tags and literal form where
  needed by the formatter.
- Parser updates only where token metadata must be carried into AST nodes.
- Formatter/unparser support for heredoc and tagged forms.
- C codegen and interpreter/eval support if string AST/token metadata affects
  existing value construction.
- Syntax highlighting samples and editor grammar updates where practical.
- Tests for lexer, parser, formatter, runtime, and self-host fixed point.
- Documentation updates in current docs and release docs.
- `ROADMAP.md`.

## Out of Scope

- Changing runtime string values based on language tags.
- SQL/HTML/JS parsing or validation.
- Automatic SQL parameterization.
- Automatic HTML escaping.
- Raw language-tagged forms such as `rsql"""..."""`.
- Bytes language-tagged forms.
- Formatter auto-selection of heredoc markers.
- Custom user-defined string processors.
- Macro-like compile-time interpretation of tagged strings.

## Acceptance Criteria

- Existing v0.31 and v0.40 string tests remain compatible.
- `<<<MARKER ... MARKER` produces the same runtime string value as an
  equivalent `"""..."""` literal.
- Heredoc indentation normalization matches triple-quoted string normalization.
- Interpolating heredocs support `{expr}`, `{{`, `}}`, and escapes.
- Raw heredocs preserve braces and backslashes literally.
- Bytes heredocs produce bytes and apply bytes escapes correctly.
- Heredoc bodies can contain literal `"""` without ending the string.
- `sql"""..."""` and `html<<<HTML ... HTML` preserve the language tag in
  token/AST metadata.
- Language tags do not change runtime value.
- Formatter round-trips tagged strings and heredocs without changing their
  literal form.
- Invalid marker names, invalid tag names, unterminated heredocs, and closing
  marker mismatches produce structured diagnostics.
- Editor syntax samples include tagged and heredoc string examples.
- The self-host fixed point remains green.

## Verification

Focused string tests:

```sh
go test ./tests -run 'TestV(31|40).*Script|Test.*String' -count=1
```

Lexer/parser/formatter package checks:

```sh
go test ./internal/lexer ./internal/parser ./internal/formatter -count=1
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

- Preserve existing v0.31 and v0.40 string semantics.
- Keep Canonical Syntax deterministic.
- Coordinate token/AST metadata changes with the public self-introspection
  library spec if that feature has already landed.

## Open Questions

None.
