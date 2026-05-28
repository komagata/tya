# Feature: Keyword Arguments

## Goal
Allow Tya calls to pass arguments by parameter name, so call sites can be clearer without adding a separate keyword-parameter declaration syntax.

## Context
`docs/SPEC.md` currently excludes named or keyword arguments and recommends dictionary options instead. The parser also rejects `name: value` inside call parentheses with `named arguments are not part of Tya v1.0.0`, and rejects `*items` splat call syntax. Function parameters already have names and may have default values. Required parameters must precede defaulted parameters, calls may omit only trailing defaulted parameters, defaults evaluate left to right at call time, and variadic parameters are not part of Tya.

The feature should preserve the existing function/method/constructor model: a keyword name is matched against the existing parameter name. There is no new declaration syntax for keyword-only parameters.

## Behavior
- Calls may include keyword arguments using `name: value`.

```tya
request(url: "https://example.com", timeout: 10)
```

- Keyword arguments bind to parameters with the same name.

```tya
request = url, timeout = 30, method = "GET" ->
  [url, timeout, method]

request(url: "https://example.com", timeout: 10)
```

- Positional arguments remain supported and are bound from left to right.
- Positional arguments may appear before keyword arguments.

```tya
request("https://example.com", timeout: 10)
```

- Positional arguments after any keyword argument are invalid.

```tya
request(timeout: 10, "https://example.com") # invalid
```

- Required parameters may be supplied by keyword.

```tya
greet = name -> "Hello {name}"

greet(name: "Tya")
```

- Default parameters continue to work when omitted.

```tya
request(url: "https://example.com")
```

- A parameter supplied both positionally and by keyword is invalid.

```tya
request("https://example.com", url: "https://example.org") # invalid
```

- Unknown keyword names are invalid.

```tya
request(url: "https://example.com", retries: 3) # invalid when no retries parameter exists
```

- Keyword argument order does not affect binding.

```tya
request(method: "POST", url: "https://example.com")
```

- Duplicate keyword names are invalid.

```tya
request(url: "a", url: "b") # invalid
```

- Keyword arguments apply to all call forms that use the normal call machinery:
  - function values;
  - instance methods;
  - class/static methods;
  - constructors such as `Color(r: 255, g: 0, b: 0, a: 1)`;
  - `super(...)` calls.
- Dictionary expansion is included with `**expr`.

```tya
options = { timeout: 10, method: "GET" }
request("https://example.com", **options)
```

- `**expr` expands a dictionary into keyword arguments.
- Expanded dictionary keys must be strings matching parameter names.
- Unknown expanded keys, duplicate expanded keys, and keys that target a positionally supplied parameter are invalid using the same rules as explicit keyword arguments.
- If multiple `**expr` expansions are used, later duplicate keys are invalid rather than overriding earlier keys.
- `**expr` may appear after positional arguments and among explicit keyword arguments.
- Positional arguments after `**expr` are invalid because `**expr` starts the keyword portion of the call.
- Existing array splat calls such as `fn(*items)` remain out of scope and invalid.
- Formatting preserves keyword calls in canonical form:
  - no space before `:`;
  - one space after `:`;
  - existing call wrapping rules still apply;
  - `**options` is preserved as a call argument.
- Interpreter execution and C emitted execution must behave the same.

## Scope
- Update `docs/SPEC.md` to remove keyword arguments from the excluded v1 forms and document the accepted call syntax.
- Update token/parser handling for call arguments so call parentheses can contain positional arguments, keyword arguments, and dictionary expansions.
- Update AST representation for call arguments so keyword names and `**` expansions are not represented as dictionary literals.
- Update formatter support for keyword arguments and `**` dictionary expansion in calls.
- Update checker arity validation for functions, methods, constructors, class/static methods, and `super(...)` calls.
- Update runtime call binding in the interpreter.
- Update C code generation and runtime support so compiled programs bind keyword calls the same way as interpreted programs.
- Update diagnostics for:
  - positional argument after keyword argument;
  - unknown keyword;
  - duplicate keyword;
  - argument supplied both positionally and by keyword;
  - non-dictionary `**expr`;
  - non-string keys in expanded dictionaries;
  - too few required arguments after keyword/default binding;
  - too many positional arguments.
- Add focused tests for parser, formatter, checker, interpreter, codegen, methods, constructors, `super(...)`, defaults, duplicate detection, unknown keyword detection, and `**` dictionary expansion.

## Out of Scope
- New keyword-only parameter declaration syntax.
- Required keyword-only parameters.
- Variadic parameters.
- Array splat calls such as `fn(*items)`.
- Accepting arbitrary unknown keywords.
- Keyword forwarding syntax beyond explicit `**dictionary`.
- Function, method, or constructor overloading.
- Changing dictionary literal syntax or dictionary option-passing behavior outside call argument parsing.
- Ruby-compatible treatment of a final dictionary argument as implicit keywords.

## Acceptance Criteria
- `f = a, b -> [a, b]` accepts `f(a: 1, b: 2)`, `f(1, b: 2)`, and `f(b: 2, a: 1)`.
- Required parameters can be supplied only by keyword, such as `greet(name: "Tya")`.
- Default parameters are filled when omitted after keyword binding.
- `f(a: 1, 2)` is rejected.
- `f(1, a: 2)` is rejected as the same parameter supplied twice.
- `f(c: 1)` is rejected when `c` is not a parameter.
- `f(a: 1, a: 2)` is rejected.
- `options = { b: 2 }` followed by `f(1, **options)` behaves like `f(1, b: 2)`.
- `f(**{ a: 1 }, a: 2)` is rejected as duplicate.
- `f(**non_dict)` is rejected at check time when statically known and at runtime otherwise.
- `f(**{ 1: "x" })` is rejected because expanded keyword names must be strings.
- Instance methods, class/static methods, constructors, and `super(...)` calls accept keyword arguments with the same binding rules.
- Existing positional-only call behavior continues to pass.
- Existing default argument behavior continues to pass.
- `fn(*items)` remains invalid.
- Formatted keyword calls are idempotent.
- `tya run` and compiled executables produce the same results.

## Verification
```sh
go test ./internal/parser ./internal/checker ./internal/eval ./internal/codegen ./internal/formatter -count=1
go test ./tests -run 'TestV65Scripts|keyword_arguments' -count=1
go test ./... -count=1
```
