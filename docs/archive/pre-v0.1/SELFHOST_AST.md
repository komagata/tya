# Self-Host AST Migration

This document defines the migration target for the Tya-written compiler.
Self-host work should move parser, checker, and codegen behavior toward this
shape before removing more generated-tool fallbacks.

## Direction

The current self-host parser emits colon-delimited node strings such as
`ASSIGN:name:INT_ADD:left:right`. Those strings are useful as a bootstrap
compatibility format, but they are not the semantic representation. New
self-hosting work should introduce structured AST values and render legacy node
strings only through an adapter.

Do not add a new source-specific or example-specific generated-C fallback to
remove an old fallback. If a feature cannot be represented by the AST schema
below, extend the schema first.

## Program Shape

The AST is an object tree:

```tya
{
  kind: "program",
  body: [statement]
}
```

Every statement has:

```tya
{
  kind: "assign",
  line: 1,
  indent: 0
}
```

Every expression has:

```tya
{
  kind: "ident",
  line: 1
}
```

`line` is the source line. `indent` is statement indentation. Meaning must live
in `kind` and typed fields, not in a split string position.

## Statement Nodes

Initial statement kinds:

- `indent`: `level`
- `assign`: `targets`, `values`
- `expr_stmt`: `expr`
- `print`: `expr`
- `return`: `values`
- `panic`: `expr`
- `exit`: `expr`
- `if`: `cond`, `then`, `else`
- `while`: `cond`, `body`
- `for`: `value_name`, `index_name`, `iterable`, `body`
- `break`
- `continue`

Later statement kinds should cover function, object, class, module, import,
constant, `try` statement forms, and method definitions.

## Expression Nodes

Initial expression kinds:

- `ident`: `name`
- `int`: `value`
- `float`: `value`
- `string`: `value`
- `bool`: `value`
- `nil`
- `array`: `elems`
- `object`: `props`
- `call`: `callee`, `args`
- `member`: `object`, `name`
- `index`: `object`, `index`
- `unary`: `op`, `expr`
- `binary`: `op`, `left`, `right`
- `try`: `expr`

This is intentionally broader than the current self-host subset so expression
parser work has a stable target.

## Adapter Rule

During migration, the parser may build AST values and then render the existing
node-string output for checker/codegen compatibility. The adapter is the only
place that should know legacy node names such as `PRINT_CALL1` or
`IF_COMPARE_GT`.

Checker and codegen migrations should then move one statement or expression
family at a time from legacy nodes to AST nodes.

## Expression Parser Migration

The final parser should use one expression entry point that returns both the
expression AST and the next token index. That is required before the parser can
handle nested expressions, postfix chains, and arbitrary call arguments without
line-specific branches.

Do not switch `parse_expr_at` to a consumed-token result while generated stage
parsers still depend on the current legacy adapter shape. The safe sequence is:

1. Keep `parse_expr_at(tokens, start)` returning only an expression AST for
   legacy callers.
2. Add a separate internal result object such as `{ expr: expr, next: next }`
   only after the generated checker/codegen path can parse and lower that
   helper shape generically.
3. Move primary parsing first: literals, identifiers, parenthesized
   expressions, calls, member access, and index access.
4. Add precedence parsing for unary, multiplicative, additive, comparison,
   `and`, and `or`.
5. Change statement parsers to consume `next` only after their legacy
   node-string adapter output is covered by fixtures and repeated-stage gates.

Until step 5, expression parser work must preserve the shallow legacy outputs
that generated checker/codegen stages already understand. For example,
`name[0] == "_"` may still need to adapt as the existing index node until the
checker and codegen can consume a real binary expression whose left side is an
index expression.

Bare call arguments are another current limit. The self-host lexer and parser
sources rely on forms such as `emit line, "INDENT", to_string(line_spaces), 1`.
The current legacy adapter only has stable generated-stage behavior for the
existing shallow `CALL1`, `CALL2`, and `CALL3` node strings, and bare calls with
more surface arguments are intentionally truncated to the compatible shape in
the node stream. Do not widen bare-call parsing in the parser alone. First move
call rendering, checking, and lowering behind a shared expression-call adapter,
then widen the parser and update generated-stage gates.

The checker has initial helper boundaries for legacy `CALL2`, `CALL3`,
`PRINT_CALL2`, and `PRINT_CALL3` validation. Codegen has an initial array-based
environment adapter for `PRINT_CALL2`, `PRINT_CALL3`, and assignment `CALL3`
lowering so helper signatures stay within the current self-host `FUNC4` limit.
Assignment `CALL2` lowering has started moving behind the same adapter for the
`Admin`, `split`, and collection-helper branches, but the fallback branch is
still inline. Do not extract that by adding wider helper signatures that the
self-host parser cannot represent. Extend the environment adapter first, then
move the remaining assignment `CALL2` fallback behind it.

## Current Migration Hazard

The repeated-stage gate still depends on coarse generated parser/checker paths
for the self-host compiler sources. In particular, small line-moving edits in
`selfhost/parser.tya` can change the generated stage-4 parser-source node stream
around the `legacy_program` loop and surface checker errors such as an undefined
`body` binding.

Do not treat those failures as a reason to add another source-specific fallback.
The next durable fix is to make the generated parser/checker path understand
the relevant AST/helper forms generically, or to remove the coarse parser-source
stream shortcut, before expanding statement parsers to consume expression
`next` values broadly.

## Verification Rule

A self-hosting slice is valid only when it proves at least one generic AST path.
Prefer fixtures that cover two different surface programs with the same AST
logic. A change that only recognizes one source line, one example name, or one
literal value is a fallback patch, not AST migration.

For volatile generated source streams, prefer semantic or inclusion-style
shape checks over exact full-stream fixtures. Exact fixtures are useful when
the ordering itself is the behavior under test. They are harmful when helper
extraction merely moves unrelated source lines and forces repeated fixture
updates without changing parser/checker/codegen meaning.

## Strategy Review Rule

After every few self-hosting slices, pause and explicitly check for loop or
dead-end signals before continuing. The review should answer:

1. Are we adding another fixed-shape legacy node such as `CALL1_*`,
   `PRINT_CALL1_*`, or `IF_CALL_*` instead of removing fixed-shape handling?
2. Are most changes line-number fixture updates caused by movement in
   `selfhost/parser.tya`, `selfhost/checker.tya`, or generated stage streams?
3. Did the slice reduce a generic gap in parser, checker, or codegen, or did it
   only make one surface spelling pass?
4. Is the next slice still aligned with the medium-term goal: generic
   expression adapters first, then broader call arguments, then statement and
   generated-code migration?

If the answer shows special-case growth or brittle fixture churn, stop the
current slice and pivot. The preferred pivot is to add or improve a generic
adapter/helper before accepting more surface syntax.
