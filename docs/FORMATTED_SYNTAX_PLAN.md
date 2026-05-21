# Accepted Syntax and Formatted Syntax Plan

This document captures a proposed direction for Tya's formatter, language
server, and surface syntax. It is a planning document, not the current language
specification.

## Goal

Tya should provide an editor experience where a user can write code quickly,
save the file, and have the editor format it into the standard Tya shape.

The model is similar to `gofmt`: the formatter should not guess how to repair
arbitrary broken code, but it should normalize valid input into one stable
formatted representation. To make that experience work well, Tya can accept a
wider surface syntax than the formatter emits.

The design goal is:

```text
Accepted Syntax -> parser -> AST -> formatter -> Formatted Syntax
                         -> checker/run/build
                         -> LSP diagnostics
```

## Terminology

### Accepted Syntax

Accepted Syntax is the source surface accepted by the lexer and parser. It is
designed for editing convenience and may include non-formatted spellings such as
trailing commas or single-quoted strings.

`tya check`, `tya run`, `tya build`, `tya format`, and the language server
should all read the same Accepted Syntax.

### Formatted Syntax

Formatted Syntax is the deterministic source representation emitted by
`tya format`.

Tya source files in documentation, examples, the standard library, and the
self-hosted compiler should be kept in Formatted Syntax. `tya format --check`
should answer whether a file already matches Formatted Syntax.

### Replacing Canonical Syntax

Tya may stop describing its source model as "every program has exactly one
source representation." A better statement for this direction is:

```text
Every accepted program has exactly one standard formatted representation.
```

This keeps the value of a single standard output form while allowing a smoother
editing experience.

## Policy

- Accepted-but-unformatted source is valid Tya input.
- Accepted-but-unformatted source is not a syntax error.
- The formatter is responsible for rewriting accepted forms into Formatted
  Syntax.
- The compiler, checker, runner, builder, formatter, and language server should
  share the same lexer/parser path where practical.
- Formatter-only syntax should be avoided because it creates a shadow language
  that the CLI and LSP diagnostics do not agree on.
- LSP diagnostics should report real syntax, semantic, and lint issues. They
  should not report an error solely because `tya format` would rewrite the
  spelling.
- Completely broken or ambiguous source does not need speculative repair.
  Formatter failure on invalid source is acceptable.

## Initial Accepted Syntax Candidates

### Trailing Commas

Accept trailing commas in:

- array literals
- dictionary literals
- call arguments
- function parameter lists, if the parser and formatter can support this
  cleanly

The formatter should emit the chosen Formatted Syntax consistently. The first
implementation can remove trailing commas everywhere unless a later formatting
policy intentionally keeps them in multiline forms.

### Single-Quoted Strings

Accept single-quoted strings and format them as double-quoted strings.

Design decisions to make before implementation:

- whether single-quoted strings use exactly the same escape rules as
  double-quoted strings
- whether interpolation is allowed in single-quoted strings
- how the formatter escapes embedded double quotes when rewriting to
  double-quoted output

The final implementation should be in the lexer/parser, not a pre-format text
replacement pass. Text replacement is fragile around comments, raw strings,
escape handling, source positions, and diagnostics.

### More Flexible Delimited Layout

Within arrays, dictionaries, calls, and similar delimited constructs, accept a
slightly wider range of newlines and indentation where parsing remains
unambiguous.

This should be scoped to delimited constructs before changing general
indentation rules.

## Implementation Strategy

Prefer extending the existing lexer and parser over adding a formatter-specific
tolerant parser.

The recommended architecture is:

```text
lexer/parser: accepted syntax -> AST
formatter: AST -> formatted syntax
checker: AST -> semantic diagnostics
runner/build: AST -> execution or generated output
LSP diagnostics: parser/checker/lint diagnostics
LSP formatter: formatter output
```

If editor recovery for incomplete code is needed later, it should be treated as
parser recovery or LSP recovery, not as a second language accepted only by the
formatter.

## Large Task List

### Epic 1: Specification Redefinition

- [ ] Reconsider the current Canonical Syntax definition.
- [ ] Add Accepted Syntax and Formatted Syntax to `docs/SPEC.md`.
- [ ] State that `tya check`, `tya run`, `tya build`, `tya format`, and LSP read
      the same Accepted Syntax.
- [ ] Define Formatted Syntax as the deterministic output of `tya format`.
- [ ] State that documentation, examples, stdlib, and self-host sources should
      stay in Formatted Syntax.
- [ ] State that accepted-but-unformatted source is not a syntax error.
- [ ] Define `tya format --check` as a Formatted Syntax drift check.
- [ ] Update README wording that currently presents Canonical Syntax as exactly
      one accepted source representation.

### Epic 2: Lexer and Parser Accepted Syntax

- [ ] Accept single-quoted strings in the lexer.
- [ ] Define single-quoted string escape rules.
- [ ] Decide whether single-quoted strings allow interpolation.
- [ ] Accept trailing commas in array literals.
- [ ] Accept trailing commas in dictionary literals.
- [ ] Accept trailing commas in call arguments.
- [ ] Decide whether function parameter lists accept trailing commas.
- [ ] Confirm method calls and chained calls follow the same argument rules.
- [ ] Broaden newline handling inside delimited constructs where unambiguous.
- [ ] Broaden indentation handling inside delimited constructs where
      unambiguous.
- [ ] Improve parse recovery only where it benefits both diagnostics and editor
      behavior.

### Epic 3: Formatter Output

- [ ] Rewrite single-quoted strings to double-quoted strings.
- [ ] Fix the double-quoted string escape policy for formatter output.
- [ ] Decide whether formatted multiline constructs keep or remove trailing
      commas.
- [ ] Stabilize array literal formatting.
- [ ] Stabilize dictionary literal formatting.
- [ ] Stabilize call argument formatting.
- [ ] Add snapshot tests for nested arrays, dictionaries, and calls.
- [ ] Add formatter idempotency tests.
- [ ] Assert `format(format(src)) == format(src)` for accepted syntax fixtures.

### Epic 4: CLI Semantics

- [ ] Test that `tya check` accepts Accepted Syntax.
- [ ] Test that `tya run` accepts Accepted Syntax.
- [ ] Test that `tya build` accepts Accepted Syntax.
- [ ] Test that `tya format` rewrites Accepted Syntax to Formatted Syntax.
- [ ] Add or clarify `tya format --check`.
- [ ] Confirm `tya lint` works on Accepted Syntax.
- [ ] Ensure accepted-but-unformatted source is never rejected only because it is
      not formatted.

### Epic 5: LSP and VS Code Experience

- [ ] Confirm LSP formatting accepts Accepted Syntax.
- [ ] Ensure accepted-but-unformatted source does not produce error
      diagnostics.
- [ ] Return no formatting edit for invalid source that cannot be parsed.
- [ ] Keep `.tya` `editor.defaultFormatter` set to `komagata.tya`.
- [ ] Keep `.tya` `editor.formatOnSave` enabled by default.
- [ ] Keep `tya.executable` auto-detection documented.
- [ ] Keep diagnostics documented as LSP-based check/lint-style diagnostics.
- [ ] Verify save-time formatting and diagnostics do not conflict in VS Code.

### Epic 6: Compatibility and Migration

- [ ] Update strict syntax tests that currently reject newly accepted forms.
- [ ] Replace tests that assert trailing commas are invalid.
- [ ] Keep self-host compiler sources in Formatted Syntax.
- [ ] Keep stdlib sources in Formatted Syntax.
- [ ] Keep examples in Formatted Syntax.
- [ ] Keep documentation code examples in Formatted Syntax.
- [ ] Add release notes explaining that the accepted input surface has widened.

### Epic 7: Regression Test Matrix

- [ ] Lexer tests for accepted single-quoted strings.
- [ ] Parser tests for trailing comma variants.
- [ ] Parser tests for multiline arrays, dictionaries, and calls.
- [ ] Formatter tests for Accepted Syntax to Formatted Syntax rewrites.
- [ ] Checker tests for accepted syntax.
- [ ] Runner tests for accepted syntax.
- [ ] LSP diagnostics tests that accepted syntax is not rejected.
- [ ] LSP formatting tests that accepted syntax produces formatted edits.
- [ ] CLI golden tests for `tya format`.
- [ ] Self-host fixed-point test remains green.

### Epic 8: Early Policy Decisions

- [ ] Do single-quoted strings allow interpolation?
- [ ] Do single-quoted strings use the same escape rules as double-quoted
      strings?
- [ ] Does Formatted Syntax remove all trailing commas or keep multiline
      trailing commas?
- [ ] Should semicolon separators remain unsupported?
- [ ] Should optional parentheses for calls remain unsupported?
- [ ] Should JSON-like unquoted dictionary keys remain out of scope for now?
- [ ] Should broader comment positions be handled separately?
- [ ] Should partial broken-code recovery be LSP-only rather than language
      syntax?

## Suggested First Large Slice

Start with trailing commas before single-quoted strings.

1. Add the Accepted Syntax and Formatted Syntax policy to the spec.
2. State that CLI and LSP share the same accepted input syntax.
3. Accept trailing commas in arrays, dictionaries, and calls.
4. Format those inputs back to the selected Formatted Syntax.
5. Add CLI formatter and LSP formatter tests.
6. Add diagnostics tests showing accepted-but-unformatted source is not an
   error.
7. Update README wording that currently overstates Canonical Syntax.

Single-quoted strings should be the second slice because escape and
interpolation policy need explicit decisions before implementation.
