# Feature: Bare Array Block Assignments

## Goal

Allow array values assigned with `=` to be written as an indented element block,
so long arrays can use a clean multi-line style without bracket noise while
short arrays still format to canonical bracket style.

## Context

Tya already supports bracket array literals:

```tya
items = ["aaa", "bbb", "ccc"]
```

The formatter can already wrap bracketed array literals when they are too long,
but assignment dictionary blocks have a cleaner block form:

```tya
user =
  name: "komagata"
  age: 20
```

This feature adds the corresponding assignment form for arrays:

```tya
array =
  "aaa"
  "bbb"
  "ccc"
```

The syntax is deliberately scoped to assignment right-hand sides. It does not
make arbitrary indented expression sequences into arrays.

## Behavior

- When an assignment uses `=` followed by `NEWLINE INDENT`, the parser accepts
  an array block body if the indented lines are expressions rather than
  dictionary entries.
- Each non-empty line in the array block is parsed as one array element.
- Array block elements are ordinary expressions using existing expression
  semantics.
- The parsed value is an ordinary `ast.ArrayLit`, so checker, interpreter, C
  emitter, equality, indexing, mutation, and iteration behavior match existing
  bracket array literals.
- Empty array blocks are invalid because empty blocks are invalid. Use `[]` for
  an empty array.
- Existing assignment dictionary blocks continue to parse as dictionaries when
  each entry begins with a valid dictionary key followed by `:`.
- Mixed array and dictionary block bodies are invalid:

  ```tya
  value =
    "aaa"
    name: "bbb"
  ```

- Array block assignment may contain inline dictionary or array literals as
  elements:

  ```tya
  values =
    { name: "aaa" }
    ["bbb", "ccc"]
  ```

- Array block assignment does not introduce brace-less dictionary literals as
  array elements. Use braces for dictionary elements.
- Array block assignment does not introduce array blocks in call arguments,
  array elements, function returns, method returns, or arbitrary expression
  positions.
- Formatter output is deterministic:
  - when an assigned array fits within 80 characters, formatter emits bracket
    style;
  - when bracket style would exceed 80 characters, formatter emits array block
    assignment style.
- Formatting is idempotent for array block assignments.
- `tya check`, `tya run`, `tya build`, `tya format`, and LSP all use the same
  parser behavior for this syntax.

## Scope

- Parser:
  - Extend `valuesAfterAssign()` or nearby assignment RHS parsing so `=`
    followed by an indented block can parse either a dictionary block or an
    array block.
  - Reject mixed dictionary-entry and array-element lines in the same block.
  - Preserve current assignment dictionary block behavior.
- Formatter:
  - Apply the 80-character bracket-style versus block-style rule to assigned
    array literals.
  - Keep output idempotent.
- Checker/runtime/codegen:
  - Treat parsed array blocks exactly like existing `ast.ArrayLit` values.
- Docs:
  - Update `docs/SPEC.md` to document assignment array block syntax and the
    formatter rule.
- Tests:
  - Add parser tests for array block assignments.
  - Add negative parser tests for empty and mixed array/dictionary block bodies.
  - Add formatter tests for under-80 bracket output and over-80 block output.
  - Add run/build coverage proving array blocks behave as ordinary arrays.
  - Keep default self-host verification green.

## Out of Scope

- Array block literals in arbitrary expression positions.
- Array block literals as function or method return syntax.
- Array block literals in call arguments.
- Array block literals inside array literals.
- Brace-less dictionary literals as array elements.
- Changing array indexing, mutation, equality, iteration, or display behavior.
- Changing formatter width globally beyond applying the existing 80-character
  rule to assignment array literals.

## Acceptance Criteria

- This source parses, checks, runs, and builds:

  ```tya
  array =
    "aaa"
    "bbb"
    "ccc"

  print(array[0])
  print(array[2])
  ```

- The assigned value is equivalent to:

  ```tya
  array = ["aaa", "bbb", "ccc"]
  ```

- Empty array block assignments are rejected; `array = []` remains valid.
- Mixed array/dictionary assignment blocks are rejected.
- Existing dictionary block assignments continue to parse as dictionaries.
- Inline dictionary literals can be array block elements when written with
  braces.
- Formatter emits bracket style when the assigned array fits within 80
  characters.
- Formatter emits array block assignment style when bracket style would exceed
  80 characters.
- Formatting is idempotent.

## Verification

```sh
gofmt -w internal/parser/parser.go internal/formatter/unparse.go
go test ./... -count=1
```
