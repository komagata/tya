# Tya Lint Rules

`tya lint` reports project-policy warnings for valid Tya programs. Lint
findings do not make a program invalid. Text output is:

```text
path:line:col: CODE message
```

JSON output contains `version` and `findings`. Each finding has `path`, `line`,
`col`, `code`, `title`, `doc_url`, `message`, and `autofixable`. SARIF output
uses the same rule IDs, names, help URIs, warning level, messages, and result
locations. LSP diagnostics use warning severity, source `tya`, the same rule
code, and the same human-readable message.

Suppressions:

```tya
# tya-lint-ignore
# tya-lint-ignore: TYAL0001, TYAL0007
# tya-lint-ignore-file
# tya-lint-ignore-file: TYAL0001, TYAL0007
```

Line comments target findings on that line or the next statement. File comments
target the whole file. Omitting codes suppresses all lint rules.

## TYAL0001

Title: Unused local

Reports a local binding that is never read. It triggers for ordinary local
assignments inside reachable code. It does not trigger for names that are read,
for `_`-prefixed intentionally ignored names, or for bindings suppressed by a
lint comment.

Autofix: yes. `tya lint --fix` removes the complete unused binding, including
indented multi-line binding bodies.

## TYAL0002

Title: Dead code after return or raise

Reports a statement that follows `return` or `raise` in the same block. It does
not trigger for statements in another branch or after an `if` whose branch may
not execute.

Autofix: no.

## TYAL0003

Title: Redundant constant if

Reports `if true` and `if false` blocks whose condition is a literal constant.
It does not trigger for variable conditions or expressions whose value is not a
literal boolean in the source.

Autofix: yes. `tya lint --fix` unwraps the reachable constant branch.

## TYAL0004

Title: Deeply nested block

Reports blocks nested at or above the linter's depth threshold. It does not
trigger for shallower block structures.

Autofix: no.

## TYAL0005

Title: Long function body

Reports function literals whose body has more statements than the linter's
length threshold. It does not trigger for functions at or below the threshold.

Autofix: no.

## TYAL0006

Title: Suspicious for index pattern

Reports `for` bindings that look like the index and item variables were written
in the wrong order. It does not trigger when the index-like name is in the
expected position or when names do not look like an index/item pair.

Autofix: no.

## TYAL0007

Title: Unused function parameter

Reports a function parameter that is never read. It does not trigger for used
parameters or `_`-prefixed intentionally ignored parameters.

Autofix: no.

## TYAL0008

Title: Shadowed binding

Reports a binding that shadows a binding from the same or an outer lexical
scope. It does not trigger when the name is unique in the visible scopes.

Autofix: no.
