---
layout: doc
title: Spec
permalink: /v0.40/spec/
---

# Tya v0.40 Specification — Raw and Bytes String Extensions

Tya v0.40 extends the v0.31 multi-line string foundation with two
prefix forms:

- `r"..."` and `r"""..."""` — **raw** strings: no escape processing,
  no interpolation.
- `b"""..."""` — **bytes** triple-quoted literal, complementing the
  existing single-line `b"..."` from v0.25.

The language otherwise is unchanged. AST, parser, codegen, and the
runtime do not change; both forms decode into the same `STRING` /
`BYTES` token kinds and `ast.StringLit` / `ast.BytesLit` AST nodes
the rest of the pipeline already handles.

## Goals

- A single-line `r"..."` literal preserves every byte of its body
  verbatim. `\n` is two characters, `{x}` is the literal four
  characters `{`, `x`, `}`. Closing `"` ends the literal.
- A multi-line `r"""..."""` literal preserves every byte of its
  body verbatim. Indentation normalization (closing-`"""` baseline
  strip) still applies, mirroring v0.31. Interpolation does not run.
- A multi-line `b"""..."""` literal carries `\n` / `\t` / `\xHH`
  escapes the same way `b"..."` from v0.25 does, with the same
  indentation-normalization rule as `"""..."""`. The result is a
  `bytes` value.
- `r"""..."""` and `b"""..."""` cooperate with the v0.38 lexer
  bracket-newline post-process: opening `"""` may sit at end of
  line, body lines are read verbatim, closing `"""` defines the
  baseline indent.
- Self-host fixed point preserved.

## Non-Goals (v0.40)

- Heredoc-style markers (`<<<TAG ... TAG`).
- Language-tagged interpolation specifiers (e.g.
  `sql"""..."""`).
- A combined `rb"""..."""` form. Each prefix is single-character.
- Changing how the existing `"..."`, `"""..."""`, or `b"..."`
  literals work.

## Surface Syntax

```
RawString          = 'r' '"'   { Char without " }      '"'
RawTripleString    = 'r' '"""' { Char | NEWLINE }      '"""'
BytesTripleString  = 'b' '"""' { Char | NEWLINE }      '"""'
```

Existing forms (unchanged):

```
String             = '"'     { Char | Escape }         '"'
TripleString       = '"""'   { Char | NEWLINE }        '"""'
BytesString        = 'b' '"' { Char | Escape | \xHH }  '"'
```

`Char` excludes the closing delimiter sequence. `NEWLINE` is
literally the source-line break. `Escape` is the v0.5 / v0.31 set
(`\n`, `\t`, `\\`, `\"`, `\{`).

### Examples

```tya
# Raw string: backslash and brace stay literal, no interpolation.
re = r"\d+ files in {dir}"
# Value: \d+ files in {dir}    (verbatim, 19 chars)

# Raw triple: spans lines, no escapes, no interpolation.
template = r"""
  Use {name} with care:
    \n is two characters here.
  """
# Value: Use {name} with care:\n  \n is two characters here.\n
# (closing-""" baseline of 2 spaces stripped per line)

# Bytes triple: \xHH and \n carry escape semantics.
header = b"""
  HTTP/1.1 200 OK\r\n
  Content-Type: text/plain\r\n
  \r\n
  """
```

### Behavior

- **Raw** (`r"..."` and `r"""..."""`):
  - No escape processing. `\n` is two characters; `\\` is two
    characters.
  - No interpolation. `{name}` is the literal four characters.
    The lexer encodes `{` as `{{` and `}` as `}}` in the emitted
    `STRING.Lexeme` so the existing interpolation runtime decodes
    them back to literal braces — semantically the body is what
    the user typed.
  - Single-line: closing `"` ends the literal. The body cannot
    contain `"`. Use the triple form when `"` is needed.
  - Multi-line: closing `"""` ends the literal. Body cannot contain
    `"""`. Indentation normalization per v0.31 §6.2.

- **Bytes triple** (`b"""..."""`):
  - Same escape set as v0.25 single-line `b"..."`: `\n`, `\t`,
    `\r`, `\\`, `\"`, `\xHH`.
  - Indentation normalization per v0.31 §6.2 applies before escape
    interpretation: each body line has the closing-`"""` baseline
    stripped, then escapes are interpreted, then the bytes are
    concatenated with literal `\n` line separators.

## Lexer

The lexer's existing single-quote string path picks up the new
`r"` prefix, mirroring the existing `b"` path. The triple-quote
path picks up `r"""` and `b"""` prefixes; it tracks the prefix
kind through body collection, applies indentation normalization,
then dispatches to:

- raw escape interpretation (no-op, plus `{` / `}` doubling) for
  `r"..."` and `r"""..."""`,
- standard string-escape interpretation for `"""..."""`,
- bytes-escape interpretation for `b"""..."""`.

The token kind emitted is `STRING` for raw forms (raw strings are
strings) and `BYTES` for `b"""..."""`.

## Acceptance Criteria

A v0.40 build is acceptable when:

1. `r"\d+"` lexes to a `STRING` token whose lexeme is `\d+`
   (4 characters, backslash literal).
2. `r"{name}"` lexes to a `STRING` whose value at runtime prints
   as the literal `{name}`.
3. `r"""..."""` spans lines and applies indentation normalization
   without escape or interpolation processing.
4. `b"""..."""` spans lines, applies indentation normalization,
   then interprets escapes per v0.25 bytes rules.
5. Existing `"..."`, `"""..."""`, and `b"..."` literals are
   unchanged.
6. `go test ./... -count=1` passes, including the self-host
   invariant.

## Self-Host Invariant

The Tya-written self-host compiler (`selfhost/v01/compiler.tya`)
does not use `r"..."` or `b"""..."""` and parses neither today.
The Go-side lexer adds these forms; the Tya-side lexer is left
alone. The fixed point holds because no normalized stdlib /
examples / selfhost source uses the new prefixes.
