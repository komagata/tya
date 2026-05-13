---
status: completed
goal_ready: false
---

# Feature: Public Tya Self-Introspection Library

## Goal

Expose Tya's lexer, parser, AST, checker, and formatter through stable public
standard-library modules so Tya programs and external tools can inspect,
validate, transform, and re-emit Tya source without depending on private Go
packages.

## Context

The current implementation has internal Go packages for lexer, parser, AST,
checker, and formatter, plus self-host compiler sources under `selfhost/`.
Those internals are not a public Tya API. Tooling such as docs, lint, LSP, code
mods, package tools, and future self-host work needs a documented interface
that can be used from Tya code.

The roadmap names these public modules:

```text
compiler.lexer
compiler.parser
compiler.ast
compiler.checker
compiler.format
```

In the class-file stdlib layout, these should live under `stdlib/compiler/` as
PascalCase class files and be imported with `import compiler/lexer`,
`import compiler/parser`, etc.

## Behavior

- Add public stdlib packages:
  - `compiler/lexer`
  - `compiler/parser`
  - `compiler/ast`
  - `compiler/checker`
  - `compiler/format`
- Use class-style public APIs:
  - `lexer.Lexer.lex(source)`
  - `lexer.Lexer.lex_with_comments(source)`
  - `parser.Parser.parse(source)`
  - `parser.Parser.parse_tokens(tokens)`
  - `ast.Ast.walk(node, visitor)`
  - `ast.Ast.children(node)`
  - `ast.Ast.kind(node)`
  - `ast.Ast.span(node)`
  - `checker.Checker.check(source)`
  - `checker.Checker.check_ast(program)`
  - `format.Format.format(source)`
  - `format.Format.unparse(program)`
- Return plain Tya dictionaries and arrays, not Go object handles.
- Use stable `kind` strings for every token, diagnostic, and AST node.
- Include source spans on every token and AST node that has source location:
  - `line`
  - `col`
  - `end_line`
  - `end_col`
- Preserve comments through parse and unparse where the current parser supports
  comment attachment.
- `Lexer.lex(source)` returns:

  ```tya
  {
    tokens: [...],
    diagnostics: [...]
  }
  ```

- Token dictionaries include:
  - `kind`
  - `lexeme`
  - `line`
  - `col`
  - `end_line`
  - `end_col`
- `Parser.parse(source)` returns:

  ```tya
  {
    program: <ast node or nil>,
    diagnostics: [...]
  }
  ```

- `Parser.parse_tokens(tokens)` accepts token dictionaries from
  `Lexer.lex(source)`.
- AST node dictionaries include:
  - `kind`
  - `span`
  - node-specific fields
  - `leading_comments` when present
  - `line_end_comment` when present
- The top-level program node includes:
  - `kind: "program"`
  - `body`
  - `file_header_comments`
- `Checker.check(source)` returns:

  ```tya
  {
    ok: true_or_false,
    diagnostics: [...]
  }
  ```

- `Checker.check_ast(program)` accepts AST dictionaries produced by
  `Parser.parse`.
- Diagnostics use the existing structured diagnostic shape where possible:
  - `severity`
  - `code`
  - `title`
  - `message`
  - `primary`
  - `hints`
  - `url`
- `Format.format(source)` parses and returns canonical source plus diagnostics:

  ```tya
  {
    source: "...",
    diagnostics: [...]
  }
  ```

- `Format.unparse(program)` accepts AST dictionaries and returns source.
- AST dictionaries must be versioned:
  - include `ast_version`;
  - document breaking changes as spec changes.
- Public AST shape must not expose private Go struct names or pointer identity.
- Round-tripping should be deterministic:

  ```text
  source -> parse -> unparse -> parse
  ```

  should preserve the same public AST for the supported corpus, modulo
  documented canonical-format normalization.
- Invalid source must return diagnostics, not panic.
- Invalid AST dictionaries passed to `check_ast` or `unparse` must return
  structured diagnostics or raise a documented structured error.

## Scope

- `stdlib/compiler/lexer/`
- `stdlib/compiler/parser/`
- `stdlib/compiler/ast/`
- `stdlib/compiler/checker/`
- `stdlib/compiler/format/`
- Builtin/runtime bridge functions as needed to call existing internal compiler
  functionality from Tya stdlib wrappers.
- Conversion between internal Go tokens/AST/diagnostics and public Tya
  dictionaries.
- Public AST schema documentation.
- Tests for lexer, parser, AST walking, checker diagnostics, formatting, and
  round-trip behavior.
- Documentation in `docs/STDLIB.md` and release docs.
- `ROADMAP.md`.

## Out of Scope

- Exposing unstable Go package APIs directly.
- A macro system.
- Runtime evaluation of arbitrary AST nodes.
- Source-to-source mutation helpers beyond generic AST dictionaries and
  `Format.unparse`.
- A full typed semantic model.
- Public incremental parser APIs.
- LSP protocol APIs.
- Replacing existing internal Go compiler packages.
- Requiring the whole self-host compiler migration to be complete first.

## Acceptance Criteria

- Tya code can lex source and inspect stable token dictionaries.
- Tya code can parse source into a stable public AST dictionary.
- Tya code can walk AST nodes and inspect child relationships.
- Tya code can run checker diagnostics on source.
- Tya code can format source through the public formatter API.
- Tya code can unparse a parsed AST back to Tya source.
- Diagnostics from lexer, parser, and checker are exposed as structured
  dictionaries with stable codes.
- Public AST node kinds, fields, and spans are documented.
- Comments are preserved through parse/unparse for supported comment positions.
- A representative corpus round-trips through parse/unparse/parse.
- Invalid source returns diagnostics without panicking.
- Invalid public AST input returns a documented error shape.
- Existing formatter, parser, checker, LSP, and doc-generator tests remain
  green.
- The self-host fixed point remains green.

## Verification

Focused stdlib/compiler API tests:

```sh
go test ./tests -run 'Test.*Compiler|Test.*Introspection|Test.*Format|Test.*Parser|Test.*Lexer' -count=1
```

Formatter and parser package checks:

```sh
go test ./internal/lexer ./internal/parser ./internal/formatter ./internal/checker -count=1
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

- Preserve the existing compiler package boundaries while adding public wrapper
  conversion layers.
- Keep AST schema changes explicit and documented.
- Reuse existing structured diagnostic types where possible.
- Coordinate with doc-generator extensions so `tya doc` can use this API later.

## Open Questions

None.
