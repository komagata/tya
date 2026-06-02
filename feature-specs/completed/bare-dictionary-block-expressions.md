# Feature: Bare Dictionary Block Expressions

## Goal

Make dictionary blocks usable in the value-producing contexts where they read
naturally, so dictionary-returning code does not need temporary variables while
Tya keeps call argument blocks reserved for keyword arguments.

## Context

Tya already supports inline brace dictionary literals:

```tya
user = { name: "komagata", age: 20 }
```

Tya also already supports assignment to a dictionary block:

```tya
user =
  name: "komagata"
  age: 20
```

The awkward case is a function or method that wants to return a multi-property
dictionary. Because function and method bodies implicitly return their final
value, the current source often needs a temporary variable just to make the
dictionary block fit the final-expression rule:

```tya
method = ->
  out =
    aaa: "aaa"
    bbb: "bbb"
    ccc: "ccc"
  out
```

The desired shape is:

```tya
method = ->
  aaa: "aaa"
  bbb: "bbb"
  ccc: "ccc"
```

Nested dictionary values should also be writable without returning to brace
style:

```tya
user =
  name: "komagata"
  profile:
    github: "komagata"
```

The feature deliberately does not adopt every CoffeeScript object-literal
position. Brace-less dictionaries inside arrays are hard to scan. Indented
`key: value` lines after a call are not dictionary literals; they are the
multi-line spelling of keyword arguments.

## Behavior

- Existing assignment dictionary blocks remain valid:

  ```tya
  user =
    name: "komagata"
    age: 20
  ```

- A function or method body may consist of dictionary entry lines as its final
  value. That final entry sequence is parsed as one dictionary literal returned
  by the function or method:

  ```tya
  method = ->
    aaa: "aaa"
    bbb: "bbb"
    ccc: "ccc"
  ```

- The function/method form applies to function literals and class method bodies
  that use the same body semantics, including instance and static methods.
- A dictionary block entry may use an indented dictionary block as its value:

  ```tya
  user =
    name: "komagata"
    profile:
      github: "komagata"
      location: "Tokyo"
  ```

- Nested dictionary block values may be nested more than one level.
- Dictionary block keys use the existing dictionary key rules:
  - identifier keys are stored as strings;
  - string-literal keys are stored as strings;
  - duplicate normalized keys are invalid;
  - non-string keys remain invalid.
- Dictionary block values are ordinary expressions unless the value is supplied
  by an indented nested dictionary block.
- Empty indented dictionary blocks are invalid. Use `{}` for an empty
  dictionary value.
- Bare dictionary blocks do not become valid in array elements:

  ```tya
  users = [
    name: "Alice"
    age: 20
  ]
  ```

  This remains invalid or must be written with braces.

- Bare dictionary blocks do not become valid as function-call arguments. An
  indented `key: value` block after a call is parsed as multi-line keyword
  arguments instead:

  ```tya
  render
    title: "Home"
    user: "aaa"
  ```

  This call is equivalent to:

  ```tya
  render(title: "Home", user: "aaa")
  ```

  Use braces when the argument value is meant to be one dictionary.

- Formatter output is deterministic:
  - when a dictionary expression fits within 80 characters, formatter emits
    brace style;
  - when brace style would exceed 80 characters in a supported block context,
    formatter emits block style.
- Formatting is idempotent for assignment dictionary blocks, function/method
  dictionary returns, and nested dictionary block values.
- `tya check`, `tya run`, `tya build`, `tya format`, and LSP all use the same
  parser behavior for this syntax.

## Scope

- Parser:
  - Preserve existing `valuesAfterAssign()` support for assignment dictionary
    blocks.
  - Extend function/method body parsing so a final dictionary-entry sequence can
    become an `ast.DictLit` final value.
  - Extend dictionary block parsing so `key:` followed by `NEWLINE INDENT`
    parses a nested `ast.DictLit` value.
  - Keep brace-less dictionaries rejected in array literals and call argument
    lists.
  - Parse indented call argument blocks as keyword arguments, not dictionaries.
- Formatter:
  - Apply the 80-character brace-style versus block-style rule to assignment
    dictionary blocks, function/method dictionary returns, and nested
    dictionary block values.
  - Format indented call argument blocks consistently with keyword argument
    call formatting.
  - Keep output idempotent.
- Checker/runtime/codegen:
  - Treat parsed block dictionaries exactly like existing `ast.DictLit` values.
  - Preserve duplicate-key checks and runtime dictionary behavior.
- Docs:
  - Update `docs/SPEC.md` to document the accepted dictionary block expression
    contexts and the out-of-scope positions.
- Tests:
  - Add parser tests for function/method dictionary returns and nested
    dictionary block values.
  - Add negative parser tests for brace-less dictionaries in arrays.
  - Add parser and call-binding tests for multi-line keyword argument blocks.
  - Add formatter tests for under-80 brace output and over-80 block output.
  - Add run/build coverage proving returned and nested values are ordinary
    dictionaries.
  - Keep default self-host verification green.

## Out of Scope

- Brace-less dictionary literals as array elements.
- Brace-less dictionary literals as function or method call arguments. Indented
  call argument blocks are keyword arguments, not dictionaries.
- General brace-less dictionary literals in arbitrary expression positions.
- Changing dictionary key access syntax. Dictionary keys still use string
  indexes such as `user["profile"]["github"]`.
- Changing dictionary key rules, duplicate-key checks, equality, iteration
  order, or mutation behavior.
- Treating dictionaries as implicit keywords.
- Changing formatter width globally beyond applying the existing 80-character
  rule to these dictionary block contexts.

## Acceptance Criteria

- Existing assignment dictionary blocks still parse, format, check, run, and
  build.
- This source parses, checks, runs, and builds:

  ```tya
  method = ->
    aaa: "aaa"
    bbb: "bbb"
    ccc: "ccc"

  result = method()
  print(result["aaa"])
  ```

- Class instance methods and static methods can return a bare dictionary block
  as their final value.
- This source parses, checks, runs, and builds:

  ```tya
  user =
    name: "komagata"
    profile:
      github: "komagata"
      location: "Tokyo"

  print(user["profile"]["github"])
  ```

- Multiple nested dictionary block levels parse successfully.
- String-literal keys work in function/method dictionary returns and nested
  dictionary block values.
- Duplicate normalized keys in any supported dictionary block form are rejected.
- Empty nested dictionary blocks are rejected; `{}` remains valid for empty
  dictionaries.
- Brace-less dictionaries in array literals remain invalid.
- Indented `key: value` blocks after calls parse as keyword arguments:

  ```tya
  render
    title: "Home"
    user: "aaa"
  ```

  This is equivalent to `render(title: "Home", user: "aaa")`.
- Brace-less dictionaries in call argument lists remain unavailable as
  dictionary literals.
- Formatter emits brace style when the dictionary expression fits within 80
  characters.
- Formatter emits block style when brace style would exceed 80 characters in a
  supported block context.
- Formatting is idempotent.

## Verification

```sh
gofmt -w internal/parser/parser.go internal/formatter/unparse.go
go test ./... -count=1
```
