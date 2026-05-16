---
status: completed
goal_ready: false
---

# Feature: Raw Quotes Inside Interpolation Expressions

## Goal

Allow string interpolation expressions to contain ordinary string literals with
double quotes, so code such as `"Hello, {user["name"]}!"` works without escaping
the inner quotes or falling out of the interpolation body early.

## Context

`docs/SPEC.md` already documents interpolation expressions with this example:

```tya
print "name: {user["name"]}"
```

The implementation currently treats the first `}` after `{` as the end of the
interpolation expression. That is too shallow for real expressions: dictionary
indexing, dictionary literals, nested calls, arrays, and strings inside the
expression can all contain braces or quotes that should be parsed as part of the
expression.

The relevant runtime paths are duplicated today:

- `internal/eval/eval.go` scans interpolated strings for the interpreter path.
- `internal/codegen/c.go` scans interpolated strings for generated C.
- `internal/checker/checker.go` validates interpolation expressions during
  checking.
- `internal/lexer/lexer.go` handles single-line, triple-quoted, raw, and bytes
  string lexing before interpolation evaluation.

Raw strings (`r"..."` and `r"""..."""`) are explicitly non-interpolating in
`docs/v0.40/SPEC.md`; this feature does not change that. Here "raw quotes"
means unescaped `"` characters inside a `{expr}` body in interpolating string
forms.

## Behavior

- The interpolation scanner finds the matching `}` by balancing `{` and `}`
  inside the expression body.
- A `{` or `}` inside a nested string literal in the expression does not affect
  brace depth.
- A `"` inside the expression body starts or ends a nested string literal for
  scanner purposes and does not terminate the outer interpolated string.
- Escaped quotes inside the nested expression string, such as
  `{user["\"quoted\""]}`, are handled by the existing lexer after the full
  expression body is extracted.
- Dictionary indexing with string keys works in interpolation:

```tya
user = {"name": "komagata"}
print("Hello, {user["name"]}!")
```

- Dictionary literals work in interpolation when they are valid Tya expressions:

```tya
print("kind: {{"kind": "ok"}["kind"]}")
```

- Nested braces in calls, arrays, dictionaries, and parenthesized expressions
  are allowed as long as the final interpolation expression parses to exactly
  one expression.
- `{{` and `}}` outside interpolation keep their current literal-brace meaning.
- Empty interpolation (`{}`), unclosed interpolation, unmatched `}` outside
  interpolation, and invalid interpolation expressions keep producing
  diagnostics.
- Triple-quoted strings (`"""..."""`) use the same interpolation body scanner as
  single-line strings.
- Raw strings (`r"..."`, `r"""..."""`) remain non-interpolating and continue to
  treat braces literally.
- Bytes literals remain non-interpolating.

## Scope

- `internal/eval/eval.go`
- `internal/codegen/c.go`
- `internal/checker/checker.go`
- shared helper code if introduced to avoid divergent interpolation scanning
- `internal/lexer/lexer.go` only if lexer string handling blocks the new cases
- `docs/SPEC.md`
- next release `docs/vX.Y/SPEC.md` and `docs/vX.Y/RELEASE_NOTES.md`
- script tests under `tests/testdata/`
- focused lexer/interpolation unit tests if useful
- `ROADMAP.md`

## Out of Scope

- String interpolation inside raw strings.
- Interpolation format specifiers.
- Statement blocks or assignments inside interpolation.
- Multi-expression interpolation.
- Changing string escape semantics.
- Changing `{{` / `}}` literal brace behavior outside interpolation.

## Acceptance Criteria

- `"Hello, {user["name"]}!"` compiles and runs, printing the indexed value.
- `"""{user["name"]}"""` compiles and runs through the same behavior after
  multi-line indentation normalization.
- Interpolation with a dictionary literal and nested braces extracts the full
  expression before parsing.
- Braces inside nested expression string literals do not affect interpolation
  depth.
- Existing empty interpolation, unclosed interpolation, unmatched brace, and
  invalid expression diagnostics still fire.
- Raw strings continue to print `{user["name"]}` literally without evaluating
  interpolation.
- Interpreter, checker, and generated-C execution paths agree on accepted and
  rejected interpolation cases.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run TestV16Script -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```

## Open Questions

None.
