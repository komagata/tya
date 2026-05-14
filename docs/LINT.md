# Tya Lint

`tya lint` reports project-policy warnings for valid Tya programs. It does not
define language validity; `tya check` owns compile-time errors.

```sh
tya lint [--fix] [--format=text|json|sarif] [paths...]
```

With no paths, lint scans the current directory recursively for `.tya` files.
Text output is the default. JSON output includes stable rule metadata, and
SARIF output can be uploaded to code-scanning tools.

## Opt-Out Comments

Suppress one finding on the same line:

```tya
unused = 1  # tya-lint-ignore: TYAL0001
```

Suppress findings for the next statement:

```tya
# tya-lint-ignore: TYAL0007
handler = used, unused ->
  print(used)
```

Suppress an entire file:

```tya
# tya-lint-ignore-file: TYAL0001, TYAL0007
```

Omit the code list to suppress every lint rule for the target line or file.

## Autofix

`tya lint --fix` applies conservative rewrites:

- `TYAL0001`: remove unused local bindings. Multi-line bindings remove their
  indented continuation block as well as the binding line.
- `TYAL0003`: unwrap `if true` and `if false` blocks using the same autofix
  hints consumed by LSP code actions.

Rules without a listed autofix remain warnings.

## Rules

### TYAL0001 Unused Local

Reports a local binding that is never read.

```tya
unused = 1
```

Autofix: remove the binding line, or the full multi-line binding block.

### TYAL0002 Dead Code After Return Or Raise

Reports statements that cannot run after `return` or `raise` in the same block.

```tya
load = ->
  return "done"
  print("unreachable")
```

### TYAL0003 Redundant Constant If

Reports `if true` and `if false`.

```tya
if true
  print("always")
```

Autofix: replace the whole `if` statement with the reachable body.

### TYAL0004 Deeply Nested Block

Reports blocks nested at depth 5 or deeper.

### TYAL0005 Long Function Body

Reports function literals with more than 50 statements in the body.

### TYAL0006 Suspicious For Index Pattern

Reports loops where the first binding looks like an index name and the second
binding does not. Tya's array loop order is `value, index`, so this usually
means the bindings were accidentally reversed.

```tya
for i, item in items
  print(item)
```

Use:

```tya
for item, i in items
  print(item)
```

### TYAL0007 Unused Function Parameter

Reports function parameters that are never read. Use `_` for intentionally
ignored parameters.

```tya
handler = req, unused ->
  print(req)
```

### TYAL0008 Shadowed Binding

Reports a binding that reuses a name from the same or an outer lexical scope.

```tya
value = 1
if ready
  value = 2
```
