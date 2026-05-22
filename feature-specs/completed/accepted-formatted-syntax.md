# Feature: Accepted and Formatted Syntax

## Goal

Introduce a two-layer source model where Tya accepts a wider editing-friendly
syntax surface, while `tya format` emits one deterministic Formatted Syntax.
Users may write accepted-but-unformatted source in editors and CLI commands
without receiving syntax errors solely because the spelling would be rewritten.

## Context

`docs/FORMATTED_SYNTAX_PLAN.md` defines the direction: Accepted Syntax is the
source surface accepted by the lexer/parser, and Formatted Syntax is the stable
output emitted by `tya format`.

The current public wording still emphasizes Canonical Syntax as if every valid
program has exactly one accepted source representation. This feature replaces
that wording with the stricter and more practical rule that every accepted
program has exactly one standard formatted representation.

The first implementation must use the normal lexer/parser path shared by
`tya check`, `tya run`, `tya build`, `tya format`, and LSP. Do not add a
formatter-only tolerant parser or pre-format text replacement pass.

## Behavior

- `docs/SPEC.md` defines:
  - Accepted Syntax as the source accepted by the lexer/parser.
  - Formatted Syntax as the deterministic output of `tya format`.
  - Every accepted program has exactly one standard formatted representation.
  - Accepted-but-unformatted source is valid Tya input, not a syntax error.
  - `tya check`, `tya run`, `tya build`, `tya format`, and LSP read the same
    Accepted Syntax.
  - Documentation, examples, stdlib, and self-host sources should stay in
    Formatted Syntax.
  - `tya format --check` reports Formatted Syntax drift.
- README wording that currently describes Canonical Syntax as exactly one
  accepted source representation is updated to the new Accepted Syntax /
  Formatted Syntax model.
- Lexer/parser accept trailing commas in:
  - array literals;
  - dictionary literals;
  - call arguments;
  - function parameter lists.
- Formatter removes trailing commas in all formatted output.
- Function definitions may be accepted with parenthesized parameter lists, with
  or without a trailing comma.
  - Accepted Syntax:

    ```tya
    add = (a, b,) ->
      a + b
    ```

  - Formatted Syntax:

    ```tya
    add = a, b ->
      a + b
    ```

- Formatter normalizes function parameter lists to the existing no-parentheses
  style when parentheses are not otherwise required by the language.
- Lexer/parser accept single-quoted strings.
- Single-quoted strings are literal strings and do not perform interpolation.
  `{...}` inside a single-quoted string is literal text.
- Single-quoted strings use the same escape vocabulary as double-quoted
  strings, plus `\'` for a literal single quote.
  - `\\` is a backslash.
  - `\'` is a single quote.
  - `\n`, `\t`, `\r`, and existing double-quoted string escapes keep the same
    meaning.
- Formatter rewrites single-quoted strings to double-quoted strings without
  changing runtime value.
  - Embedded double quotes are escaped as needed.
  - Backslashes and control escapes are preserved or re-escaped as needed.
  - Literal `{` and `}` from a single-quoted string must be escaped or otherwise
    emitted so that the formatted double-quoted string does not introduce
    interpolation.
- LSP diagnostics do not report syntax errors for accepted-but-unformatted
  trailing commas, parenthesized function parameter lists, or single-quoted
  strings.
- LSP formatting returns edits that rewrite Accepted Syntax to Formatted
  Syntax.
- Invalid or ambiguous broken source may still fail parsing and formatting. The
  formatter does not need speculative repair for incomplete programs.

## Scope

- Documentation:
  - `docs/SPEC.md`
  - `README.md`
  - release notes or current-version docs only if they directly describe the
    source model changed by this feature
- Lexer/parser:
  - string tokenization and parsed string values
  - trailing comma handling for arrays, dictionaries, calls, and function
    parameter lists
  - parenthesized function parameter lists
- Formatter:
  - single-quoted string output normalization
  - trailing comma removal
  - function parameter list normalization to no-parentheses style
  - idempotency for the newly accepted forms
- CLI behavior:
  - `tya check`
  - `tya run`
  - `tya build`
  - `tya format`
  - `tya format --check`
  - `tya lint` where it reuses parser/checker input
- LSP:
  - parser/checker diagnostics
  - document formatting
- Tests:
  - lexer tests
  - parser tests
  - formatter golden/idempotency tests
  - checker/runner/build CLI tests
  - LSP diagnostics and formatting tests
  - strict syntax tests currently rejecting the newly accepted forms
  - self-host fixed-point tests remain green

## Out of Scope

- Semicolon statement separators.
- Optional parentheses for ordinary function or method calls.
- JSON-like unquoted dictionary keys beyond the existing accepted dictionary
  syntax.
- Broader comment placement changes.
- General indentation changes outside delimited constructs and function
  parameter parsing needed by this feature.
- Speculative broken-code recovery for incomplete editor buffers.
- VS Code manual or browser-level validation. Go LSP tests are sufficient for
  this feature.
- Changing stdlib, examples, docs examples, or self-host sources into
  accepted-but-unformatted style. They should remain in Formatted Syntax.

## Acceptance Criteria

- `docs/SPEC.md` and README describe Accepted Syntax and Formatted Syntax
  without claiming that every accepted program has exactly one source spelling.
- `tya format --check` is documented as a Formatted Syntax drift check.
- `tya check` accepts:
  - `[1, 2,]`
  - `{ name: "Ada", age: 20, }`
  - `print(add(1, 2,))`
  - `add = (a, b,) -> a + b`
  - `name = 'Tya'`
- `tya format` rewrites those forms to Formatted Syntax:
  - trailing commas removed;
  - function parameter list parentheses removed;
  - single-quoted strings rewritten to double-quoted strings.
- Single-quoted strings do not interpolate before formatting and preserve the
  same runtime string value after formatting.
- Formatting single-quoted strings containing `{`, `}`, `"`, `\\`, `\'`, and
  newline/tab/carriage-return escapes does not change runtime string values.
- `format(format(src)) == format(src)` holds for fixtures covering arrays,
  dictionaries, calls, function parameters, single-quoted strings, and nested
  combinations.
- `tya run` executes accepted-but-unformatted source that uses the new accepted
  forms.
- `tya build` builds accepted-but-unformatted source that uses the new accepted
  forms.
- `tya lint` can parse accepted-but-unformatted source and does not reject it
  solely because formatting would rewrite it.
- LSP diagnostics do not mark the new accepted forms as syntax errors.
- LSP formatting rewrites the new accepted forms to Formatted Syntax.
- Existing strict syntax tests are updated so they no longer assert these newly
  accepted forms are invalid.
- Stdlib, examples, docs examples, and self-host sources remain formatted.
- The self-host fixed-point invariant remains green.

## Verification

```sh
go test ./internal/lexer ./internal/parser ./internal/formatter ./internal/checker ./internal/lsp -count=1
go test ./tests -run 'TestFormat|TestCLI|TestLsp|TestSelfhostV01Scripts|TestSelfhostV02Scripts' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
