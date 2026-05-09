# Tya v0.31 Specification — Multi-line String Literals

Tya v0.31 adds **triple-quoted string literals** (`"""..."""`) so that
multi-line strings can be written without `\n` escapes. The new form
parses to the same `StringLit` AST node as today's `"..."` literal,
runs through the same interpolation pipeline, and produces the same
runtime string value. Codegen, eval, the runtime, and the C ABI do
not change.

## Goals

- Triple-quoted strings can span any number of lines.
- Newlines inside `"""..."""` are part of the string value.
- `{expr}` interpolation works exactly as in `"..."` strings.
- Indentation normalization makes nested multi-line strings readable
  without leaking enclosing indent into the value.
- Single-line `"""..."""` is also valid (opens and closes on the same
  line) — the body between the quotes is the value.

## Non-Goals (v0.31)

- Heredoc-style markers (`<<<TAG ... TAG`).
- Raw-string prefixes (`r"..."`, `r"""..."""`).
- Language-tagged interpolation specifiers (e.g. `sql"""..."""`).
- Formatter rewrite rule — turning long single-line `"..."` literals
  into `"""..."""` is intentionally deferred to a follow-up release.
- Bytes equivalent (`b"""..."""`) — bytes literals stay single-line in
  v0.31.

## Surface Syntax

```
TripleString = '"""' { Char | NEWLINE } '"""'
```

`Char` is the same alphabet as today's `"..."` literals: regular UTF-8
text plus the escape sequences `\n`, `\t`, `\\`, `\"`, `\{`, the
double-brace literals `{{` / `}}`, and `{expr}` interpolation.

### Single-line form

```tya
greet = """Hello, {name}!"""
```

is equivalent to `"Hello, {name}!"`.

### Multi-line form

```tya
poem = """
Roses are red,
Violets are blue.
"""
```

Newlines in the body are part of the value. The value of `poem` is:

```
\nRoses are red,\nViolets are blue.\n
```

The opening newline immediately after the opening `"""` is preserved
unless it sits on the line containing the opening `"""` (see
*Indentation Normalization* below).

### Interpolation

```tya
table = """
| name | role     |
| ---- | -------- |
| {name} | {role} |
"""
```

`{expr}` is parsed and evaluated identically to its `"..."` cousin.
`{{` and `}}` produce literal braces. `\{` is also accepted.

## Indentation Normalization

The indentation of the **closing** `"""` defines a *baseline*. The
baseline is removed from every body line before the value is emitted.
Concretely:

1. Let `b` be the column (number of leading ASCII spaces) of the
   closing `"""`. Tabs are forbidden by the project's existing
   indentation rule and remain forbidden here.
2. For each body line (every line strictly between the opening line
   and the closing line):
   - If the line is empty (zero characters before the trailing
     newline), it remains empty.
   - Otherwise, the first `b` characters must all be ASCII spaces;
     they are stripped.
   - If a non-empty body line has fewer than `b` leading spaces, it is
     a compile error: `mixed indentation in triple-quoted string`.
3. The opening line's content (anything between the opening `"""` and
   the end of that line) is preserved verbatim. If that content is
   empty, the leading newline that follows the opening `"""` is
   *omitted* from the value (so `"""\nfoo\n  """` renders as `foo\n`,
   not `\nfoo\n`).
4. The closing line's content (anything before the closing `"""` on
   the closing line) is preserved verbatim and is **not** subject to
   the leading-baseline strip — but must equal `<b spaces>` followed
   immediately by `"""` for indentation normalization to apply.

### Examples

```tya
msg = """
  Hello
  Tya
  """
```

Closing `"""` is at column 3, baseline `b = 2`. The two body lines
have ≥ 2 leading spaces; both lose 2. Value: `Hello\nTya\n`.

```tya
nested = """
    {
      "a": 1
    }
    """
```

Baseline `b = 4`. All body lines lose 4 leading spaces. Value:
`{\n  "a": 1\n}\n`.

```tya
flat = """one line"""
```

Single-line form. Value: `one line`. No newline normalization.

```tya
inline_open = """first
  more
  """
```

Opening line carries content (`first`). The leading newline after
`"""` is not omitted because the opening line is non-empty in this
form; the value begins with `first\n  more\n` — wait, that body line
does not start with the closing `"""`'s baseline.

To clarify, the opening-line content rule only suppresses the
**opening-newline** insertion; baseline stripping still applies to all
*body* lines (lines after the opening line and before the closing
line). For the example above, baseline is `b = 2` (closing `"""` is at
column 3). The body line `  more` has the 2-space baseline stripped,
producing `more`. The value is `first\nmore\n`.

A single-line form — opening and closing `"""` on the same line — is a
special case where there are no body lines at all and no
normalization happens.

### Error cases

| Condition                                                        | Error                                                        |
|------------------------------------------------------------------|--------------------------------------------------------------|
| Unterminated `"""..."""` (EOF before close)                      | `unterminated triple-quoted string`                          |
| A body line that is non-empty and has fewer leading spaces than the baseline | `mixed indentation in triple-quoted string`           |
| The closing line has non-space content before `"""`              | `closing """ must be on a line of its own (whitespace + """)` when normalization is requested. The single-line form (`"""x"""`) is not subject to this. |
| Tab inside a `"""..."""` body                                    | The existing project-wide *tabs are forbidden* error fires.  |

## Escape rules

Inside `"""..."""`:

- `\n`, `\t`, `\\`, `\"`, `\{` behave exactly as in `"..."`.
- `{{` produces `{`. `}}` produces `}`.
- A literal three quote sequence `"""` inside the body terminates the
  string. To embed `"""` inside the value, use `\"\"\"` or split the
  string and concatenate.

## Lexer changes

The lexer continues to drive line-by-line, but when `lexLine`
encounters an unmatched `"""`:

1. The remainder of the current line (after the opening `"""`) is
   buffered into the literal value.
2. The lexer enters a "triple-string" mode that consumes whole
   subsequent lines verbatim (raw, not via `lexLine`'s normal
   tokenization) until a line whose trimmed body starts with `"""` is
   found.
3. The lexer collects the closing-line indent for normalization,
   strips the baseline from each buffered body line, applies escape
   handling, and emits a single `STRING` token whose value is the
   normalized body.
4. Indent / dedent processing for the lines inside the literal is
   suppressed; the lexer resumes its normal indent state on the line
   *after* the closing `"""`.

The token kind is the existing `STRING`. Downstream stages don't need
to change.

## AST and runtime

No changes. `"""..."""` produces an `ast.StringLit` whose `Value` is
the normalized body string with `{expr}` interpolations preserved
verbatim, exactly as the existing `"..."` form does.

The string-interpolation pipeline (existing) sees the same body shape
it sees today, so codegen, the C runtime, and the Go interpreter all
work unchanged.

## Self-Host Invariant

The self-host pipeline does not use triple-quoted strings. Its source
remains valid v0.31 source. `TestSelfhostV01Scripts` continues to
pass.

## Testing Strategy

1. **Lexer unit tests** under `internal/lexer/lexer_test.go`:
   - Single-line `"""x"""` → STRING with value `x`.
   - Multi-line with indentation normalization.
   - Multi-line with interpolation.
   - Mixed-indent → error.
   - Unterminated → error.
2. **Script test** under `tests/testdata/v31/multiline.txtar`:
   - End-to-end `tya run` of a program using `"""..."""` with
     interpolation and verifies stdout.
3. **Self-host invariant** continues to pass.
4. **Default test suite** (`go test ./...`) remains green.

## Acceptance Criteria

A v0.31 build is acceptable when:

1. `"""..."""` lexes into the same `STRING` token kind today's
   `"..."` form produces.
2. Multi-line literals preserve internal newlines.
3. `{expr}` interpolation works inside `"""..."""`.
4. Indentation normalization follows the closing-`"""` baseline rule.
5. Error cases produce structured errors (line + column).
6. `go test ./... -count=1` passes, including the self-host
   invariant.

## Deferred to v0.31.x / v0.32

- Formatter rewrite of long single-line `"..."` to `"""..."""`.
- Raw-string prefix `r"""..."""`.
- Bytes-equivalent `b"""..."""`.
- Heredoc-style markers.
- Language-tagged interpolation specifiers.
