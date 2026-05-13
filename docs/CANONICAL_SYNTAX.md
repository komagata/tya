# Canonical Syntax

This document defines Tya's **Canonical Syntax** — the property that every
valid Tya program has exactly one source representation, and the rules that
make this true.

This is a specification document intended for implementation. It captures all
design decisions on Canonical Syntax made to date. It is self-contained; an
implementer should be able to act on this document without additional
conversational context.

## 1. Principle

**Tya has a Canonical Syntax.** Every well-formed Tya program has exactly one
source byte sequence per AST. Formally:

```
unparse(parse(source)) == source     (byte-for-byte)
parse(unparse(ast))    == ast        (structural)
```

The formatter (`tya format`) is **the canonical serializer**. It is part of
the language, not a separate tool with its own opinions. There is no
configuration, no per-project override, and no style flag. Running `tya
format` twice on the same program must produce identical output (idempotency).

This property is a **language-level invariant**, second only to the self-host
fixed point (`selfhost/v01/compiler.tya`). Any change that breaks
`source ↔ AST` bijection is a regression.

## 2. The atomic-token exception

A line may exceed the column limit if and only if the overflow is caused by a
single **atomic token** that cannot be broken without changing meaning. The
formatter never *chooses* to exceed; the exception only reflects an
unbreakable token in user code.

Atomic tokens are:

- identifiers
- numeric literals
- string literals (after multi-line normalization, see §6)
- import paths

This is the only allowed deviation from the column limit. It is documented as
a rare, honest exception, not a license to ignore the limit.

## 3. Comments

Tya recognizes exactly **three** comment kinds. Every other position where a
`#` could appear is a parse error.

### 3.1 Leading comment

One or more `#` lines that appear immediately before a node, at the same
indentation as that node. They are attached to that node as the
`leading_comments` AST attribute (a list of strings, in source order).

```tya
# greet a user by name
greet = name -> "Hello, " + name
```

The two lines `# greet a user by name` are this function's `leading_comments`.

There must be no blank line between the comment block and the node it
attaches to. A blank line breaks attachment.

### 3.2 Line-end comment

A single `#` comment placed on the same line as a statement, after the
statement. Attached to that statement as the `line_end_comment` AST
attribute (a single string, or null).

```tya
x = 1  # initial value
```

There is exactly one space between the statement and `#`. The comment runs
to end-of-line. No more than one line-end comment per statement.

### 3.3 File header comment

`#` lines at the start of a file, separated from the file body by exactly
one blank line. Attached to the file AST node as the
`file_header_comments` attribute (a list of strings).

```tya
# This file is the dog entry point.
# It coordinates dog-related top-level items.

import json
import file

# ...
```

The blank line between the header and the body is mandatory and is what
distinguishes a file header from a leading comment on the first node:

```tya
# This is a leading comment on the import below.
import json
```

vs

```tya
# This is a file header comment.

import json
```

If the file has only a header (no body), the header is still well-formed:
the `file_header_comments` attribute is set, and the body is empty.

### 3.4 Forbidden comment positions

All of the following are parse errors:

- A `#` at the end of a block with no following node (block-trailing
  comment).
- A `#` at the end of a file with no following node (file-trailing
  comment).
- A `#` inside an expression, argument list, array literal, dict literal,
  or any other bracketed context.
- A block whose body consists only of comments (no statements).

Every comment must have a definite attachment target. Comments without one
are not legal.

### 3.5 Blank line rules

Blank lines are determined by AST shape. The formatter inserts them; users
do not choose. The rules:

1. Exactly **one** blank line between top-level definitions.
2. Exactly **one** blank line before any in-block statement that has a
   leading comment block, **except** when that statement is the first
   statement in its block.
3. Otherwise, no blank lines.

Example:

```tya
bark = ->
  # initial voice
  voice = "bow!"

  # add second voice
  # it is cute!
  voice = voice + " wow!"
  voice = voice + " wan!"
```

- `# initial voice` is the first statement in the function body → no
  preceding blank line.
- `# add second voice` is preceded by a blank line because it has a leading
  comment block and is not first.
- `voice = voice + " wan!"` has no leading comment → no preceding blank
  line.

## 4. Indentation

- Indentation is **2 spaces** per level.
- Tabs are forbidden anywhere in source. Tab characters in source are a
  parse error.
- Continuation lines (multi-line forms in §5) are indented by `+2` from the
  parent.

## 5. Long-line wrapping

### 5.1 Column limit

The column limit is **80** columns.

This is fixed by the language and not configurable.

### 5.2 Algorithm

For each "wrappable" construct, the formatter:

1. Renders the construct in its **single-line form** at the current indent.
2. If the rendered length plus the current indent ≤ 80, emits the
   single-line form.
3. Otherwise, emits the **multi-line form** for that construct.
4. Recurses into nested constructs **only as needed** — if an outer
   construct wraps but an inner construct fits inline, the inner stays
   inline.

In other words: minimum-necessary wrapping. The formatter does not
opportunistically wrap things that fit.

### 5.3 Per-construct multi-line forms

Each wrappable construct has exactly one canonical multi-line form. The
formatter does not choose between alternatives.

#### 5.3.1 Function call

Single-line:
```tya
foo(a, b, c)
```

Multi-line:
```tya
foo(
  a,
  b,
  c,
)
```

Rules:
- Each argument on its own line.
- **Trailing comma required** in multi-line form.
- Closing `)` on its own line at the same indent as the call's start.
- Continuation indent: `+2` from the call's start.

#### 5.3.2 Array literal

Single-line:
```tya
[1, 2, 3]
```

Multi-line:
```tya
[
  1,
  2,
  3,
]
```

Rules: same as function call (trailing comma required, closing bracket on
its own line at the literal's start indent).

#### 5.3.3 Dict literal

Single-line is the **inline form**:
```tya
{ name: "x", age: 1 }
```

Multi-line is the **block form** (no braces, no commas):
```tya
user =
  name: "x"
  age: 1
```

The column limit determines which form is used; the user does not pick.
Each key-value pair on its own line in the block form. The block form
attaches to its assignment target on the previous line via `=`.

#### 5.3.4 Function expression with multiple parameters

Tya introduces `(a, b) -> body` syntax for multi-parameter function
expressions. (Currently Tya supports `name -> body` for single-parameter
functions; this extends it.)

Single-line:
```tya
add = (a, b) -> a + b
```

Multi-line wrap of the parameter list:
```tya
add = (
  a,
  b,
) -> a + b
```

If the body itself is too long, switch to block body form (§5.3.7).

#### 5.3.5 Binary operator chains — leading-operator style

Single-line:
```tya
total = a + b + c + d
```

Multi-line:
```tya
total = a
  + b
  + c
  + d
```

Rules:
- The first operand stays on the line with the assignment.
- Each subsequent operand is preceded by its operator at the start of the
  continuation line.
- Continuation indent: `+2` from the start of the right-hand-side
  expression.

This applies to all binary operators (`+`, `-`, `*`, `/`, `%`, `==`, `!=`,
`<`, `>`, `<=`, `>=`, `and`, `or`, `&`, `|`, `^`, `<<`, `>>`).

#### 5.3.6 Long conditions in `if` / `while`

When the condition expression exceeds 80, the formatter inserts
parentheses around the condition and wraps the inside.

Single-line:
```tya
if some_condition + another_part > threshold and not exceptional_case
  process()
```

Multi-line:
```tya
if (
  some_condition
    + another_part
    > threshold
    and not exceptional_case
)
  process()
```

Rules:
- The opening `(` follows `if` (or `while`) with one space.
- The condition's first operand is on the next line, indented `+2` from
  the keyword.
- Continuation lines follow leading-operator style (§5.3.5).
- The closing `)` is on its own line, at the keyword's indent.
- The body is indented `+2` from the keyword as usual.

The parentheses are formatter-inserted. The user did not write them; the
formatter adds them when wrapping. This is the only case where the
formatter adds tokens that the user did not write.

#### 5.3.7 Long iterable / value in `for` / `match`

`for x in iterable` and `match value` do not get extra outer parentheses.
The iterable / value is wrapped using the normal rule for whatever
construct it is.

```tya
for item in compute_filtered_items(
  source_a,
  source_b,
  source_c,
)
  process(item)
```

Here the function call wrap rule (§5.3.1) handles the wrapping. The
closing `)` on its own line visually distinguishes the iterable from the
body that follows.

#### 5.3.8 Long function body after `->`

Single-line lambda:
```tya
greet = name -> "Hello, " + name
```

When the single-line form exceeds 80, switch to block body form:
```tya
greet = name ->
  "Hello, " + name
```

If the body is still too long, wrap recursively per §5.3.5:
```tya
greet = name ->
  "Hello, "
    + name
    + "! Welcome to "
    + service_name
```

### 5.4 Trailing commas

- Single-line forms: trailing comma is **forbidden**.
  - `[1, 2, 3]` — correct
  - `[1, 2, 3,]` — parse error
- Multi-line forms: trailing comma is **required**.
  - The multi-line form examples above always include `,` after the last
    element before the closing bracket.

### 5.5 Imports are atomic

Import statements are not wrapped. If an import path is unusually long, the
line exceeds 80 — the atomic-token exception (§2) applies.

```tya
import some_very_long_module_path_that_exceeds_eighty_columns_in_total_length
```

The formatter does not split imports.

### 5.6 String literals are atomic

A regular `"..."` string literal is never split mid-string by the
formatter.

If a single-line `"..."` literal exceeds 80 columns and its content has
natural breakpoints (i.e. logical line breaks expressible without changing
meaning), the formatter rewrites it to the multi-line `"""..."""` form
(§6).

If the content cannot be naturally split (e.g. a long URL with no
whitespace), the literal is emitted as-is and the line exceeds 80 under
the atomic-token exception.

## 6. Multi-line string literals

Tya introduces a triple-quote multi-line string literal: `"""..."""`.

### 6.1 Syntax

```tya
message = """
  User {user.name} performed {action.type}
  on resource {resource.id}
  at {timestamp}
  """
```

- Opens with `"""` and closes with `"""`.
- Newlines inside `"""..."""` are part of the string value.
- `{expr}` interpolation works the same as in regular `"..."` strings.
- Standard escapes (`\n`, `\t`, `\\`, `\"`, `\{`, `{{`, `}}`) work as in
  regular strings.
- Literal `"""` inside the body is not allowed.

### 6.2 Indentation normalization

The closing `"""` defines a baseline indentation. That baseline is stripped
from every line of the literal.

In the example above, the closing `"""` is indented 2 spaces, so the
string value is:

```
User {user.name} performed {action.type}
on resource {resource.id}
at {timestamp}
```

Each content line had 2 extra spaces beyond the baseline; those spaces are
**preserved** as content. The 2-space baseline itself is stripped.

This makes nested multi-line strings readable without leaking enclosing
indent into the value.

### 6.3 Formatter rewrite rule

When `tya format` encounters a single-line `"..."` literal that:

1. exceeds 80 columns at its position, **and**
2. has content where a multi-line form would be readable (i.e. content
   contains literal `\n` that the formatter can convert to actual newlines
   without changing semantics),

then the formatter rewrites it to the multi-line `"""..."""` form. The
rewrite rule is part of the canonical-form specification — given the same
AST, the formatter always produces the same multi-line layout.

When the literal cannot be naturally split, the formatter leaves it as-is
under the atomic-token exception.

## 7. Operator spacing

Whitespace around operators is canonical and not user-configurable.

- Binary operators: exactly one space on each side.
  - `a + b`, `a == b`, `a and b`
- Unary operators: no space between operator and operand.
  - `-x`, `not x`
- `,` in argument lists, array literals, dict inline form: no space before,
  one space after.
  - `foo(a, b)`, `[1, 2, 3]`, `{ a: 1, b: 2 }`
- `:` in dict key-value pairs: no space before, one space after.
  - `name: "x"`
- `->` in function expressions: one space on each side.
  - `name -> body`, `(a, b) -> body`
- `=` in assignments: one space on each side.
  - `x = 1`
- No space inside `(`, `[`, `{` in single-line forms (with one specific
  exception: dict inline form uses `{ k: v }` with one space inside, see
  §5.3.3).

## 8. Other canonical-form decisions

### 8.1 String concatenation vs. interpolation

The formatter does **not** rewrite `"a" + b + "c"` into `"a{b}c"`, nor the
reverse. The two are distinct AST shapes; users choose one when writing,
and the formatter preserves the choice. This is a deliberate exception to
"one way only" — automatic rewrite risks changing semantics in subtle
cases (precedence, nil handling), and the cost is low.

### 8.2 String quote normalization

`"..."` is the only canonical form for regular string literals. The
formatter normalizes any non-canonical spelling to `"..."`. (Currently
Tya only allows `"..."`, so this is a confirmation, not a change.)

### 8.3 `elseif` vs `else if`

`elseif` is canonical. `else if` is rejected by the parser as a syntax
error.

### 8.4 Import ordering and grouping

- Imports are sorted alphabetically by import path.
- Stdlib imports and user imports form **separate groups**, separated by
  exactly one blank line. Stdlib imports come first.
- Within a group, imports are in alphabetical order with no blank lines.
- Imports with leading comments form an implicit subgroup boundary: a
  blank line precedes any import that has a leading comment, except when
  it is the first import in its group.

### 8.5 `case _` position in `match`

The `case _` (wildcard) branch, if present, must appear **last** in a
`match` statement. The formatter does not reorder cases; the parser
rejects a `case _` followed by another `case`.

### 8.6 Empty collections

- Empty array: `[]` (no spaces).
- Empty dict: `{}` (no spaces).

These are the only canonical empty forms. Alternative spellings (e.g.
`[ ]`, `{ }`) are normalized to the canonical form.

### 8.7 Empty `else` branches

An `if` with an `else` block whose body is empty (or contains only
no-op constructs) is rewritten by the formatter to remove the `else`.

```tya
if cond
  do_thing()
else
  # (empty)
```

becomes

```tya
if cond
  do_thing()
```

(Note: under §3.4, a block consisting only of comments is itself a parse
error, so the body is never literally just comments.)

## 9. Multiple-return value style

Until a static type system is introduced, multiple-return functions may
return either the full tuple including `nil` for the error slot or the
non-error value alone:

```tya
return user["name"], nil    # explicit nil
return user["name"]         # implicit nil for the second slot
```

The canonical form is **explicit nil**. The formatter rewrites the
implicit form to the explicit form. Rationale: explicit form is
self-documenting and matches the call-site shape.

## 10. Project-policy boundary

Per-project rules — such as maximum identifier length, banned APIs, or
naming conventions specific to a team — are **not** part of Canonical
Syntax. They belong in `tya lint`, which operates as project policy on
top of the canonical form.

The formatter and the language do not enforce or accept any
project-specific stylistic rule. If a property is universal across all
Tya programs, it is in this document. If it is project-specific, it is
in `tya lint`.

## 11. Implementation notes

### 11.1 Parser changes

- Reject tabs as indentation.
- Reject `else if` (require `elseif`).
- Reject trailing commas in single-line list / dict / argument forms.
- Require trailing commas in multi-line list / dict / argument forms.
- Reject comments in forbidden positions (§3.4) with a structured
  diagnostic.
- Recognize `(a, b) -> body` syntax (§5.3.4).
- Recognize `"""..."""` literal with indentation normalization (§6).

### 11.2 AST shape

Each AST node gains:

- `leading_comments: list[string]`
- `line_end_comment: string | null`

The file AST node gains:

- `file_header_comments: list[string]`

The AST does **not** carry blank-line information, indentation
information, or wrap-form information. These are derived deterministically
from the AST shape by the formatter.

### 11.3 Formatter (canonical serializer)

The formatter is **`unparse(ast)`** — a deterministic function from AST to
source bytes. Implementation outline:

```
unparse(node):
  for each child of node, recursively decide single-line or multi-line
  emit canonical bytes per §3, §5, §6, §7, §8
  emit blank lines per §3.5
```

Required properties:

- **Idempotent**: `unparse(parse(unparse(parse(s))))) == unparse(parse(s))`
- **Stable**: same input produces same output across platforms (LF only,
  no locale dependencies)
- **Exhaustive**: every AST shape is handled; never falls through to a
  default

### 11.4 Round-trip tests

Add tests asserting:

- For a representative corpus, `unparse(parse(s)) == s` byte-for-byte
  after the corpus is normalized once.
- For a representative AST corpus, `parse(unparse(ast)) == ast`
  structurally.
- Idempotency over the full corpus.

### 11.5 Migration from current Tya

Current Tya allows multiple non-canonical forms (e.g. dict inline vs.
block, with different element counts, no fixed wrap rule). The migration:

1. Implement the formatter per this spec.
2. Run the formatter over `examples/`, `stdlib/`, `selfhost/v01/`, and
   `tests/testdata/` to normalize.
3. Verify the self-host fixed point still holds after normalization
   (`go test ./tests -run TestSelfhostV01Scripts -count=1`).
4. Reject non-canonical forms at parse time once the codebase is fully
   normalized.

This is a large change. It should ride a minor version (e.g. v0.30) and
the version's `docs/vX.Y/SPEC.md` should reference this document.

## 12. Decisions deferred

These items are not yet decided and are out of scope for the initial
Canonical Syntax implementation. They will be added in follow-up work:

- Pipe / method-chain syntax (e.g. `a |> b |> c`). When introduced, its
  multi-line form will follow leading-operator style (§5.3.5).
- Static type annotations. If introduced, their syntax and canonical form
  must be specified before they ship.
- Macros, quasi-quoting, custom operator definitions — none of these are
  on the roadmap; if added later, their canonical forms must be defined
  here.

## 13. Relation to other Tya invariants

| Invariant | Relation to Canonical Syntax |
|---|---|
| Self-host fixed point | Canonical Syntax must not regress `selfhost/v01/compiler.tya` compilation. |
| Omakase Declaration | Canonical Syntax is the operational form of "one canonical way" (Core 1) and "no customization" (Core 3). |
| Specification over Configuration | Canonical Syntax is the most direct expression of this principle in Tya. |
| Kind diagnostics | Comment-position errors, indentation errors, and trailing-comma errors must follow the diagnostics philosophy: stable code, expected/found, hint, doc URL. |

## 14. Summary checklist for the implementer

- [ ] Parser rejects tabs, `else if`, forbidden comment positions,
      mismatched trailing-comma usage.
- [ ] Parser accepts `(a, b) -> body` and `"""..."""` literals.
- [ ] AST nodes carry `leading_comments` and `line_end_comment`.
- [ ] File AST node carries `file_header_comments`.
- [ ] Formatter implements per-construct single-line / multi-line forms
      per §5.
- [ ] Formatter inserts blank lines per §3.5.
- [ ] Formatter normalizes operator spacing per §7.
- [ ] Formatter rewrites long single-line strings to `"""..."""` per
      §6.3.
- [ ] Formatter is idempotent and platform-stable.
- [ ] Round-trip tests pass on examples, stdlib, selfhost, tests/testdata.
- [ ] Self-host fixed point verified after migration.
- [ ] Diagnostics for new parse-error cases follow the kind-diagnostics
      philosophy.
