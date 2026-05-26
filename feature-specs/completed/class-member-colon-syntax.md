# Feature: Class Member Colon Syntax

## Goal

Change class and interface member declarations from assignment-like `=` syntax
to member-declaration `:` syntax, while keeping ordinary top-level, local, and
field assignment syntax unchanged.

## Context

Tya currently declares class and interface members with `=`:

```tya
class User
  ROLE = "user"

  initialize = name = "guest" ->
    self.name = name

  label = ->
    "user:{self.name}"
```

This makes class body declarations look like ordinary assignment even though
they define members. The new canonical class and interface syntax should use
`:` for all member declarations:

```tya
class User
  ROLE: "user"

  initialize: name = "guest" ->
    self.name = name

  label: ->
    "user:{self.name}"
```

Ordinary assignments such as `name = "Tya"`, `self.name = name`, destructuring
assignment, and function declarations outside class/interface bodies remain
unchanged.

## Behavior

- Class body member declarations use `:` instead of `=`.
- Interface body member declarations use `:` instead of `=`.
- The following member categories use `:`:
  - instance fields;
  - class constants;
  - static class variables;
  - instance methods;
  - static methods;
  - constructors through `initialize`;
  - abstract methods;
  - override methods;
  - interface method requirements;
  - interface default methods;
  - interface fields;
  - interface `initialize` hooks.
- Modifiers keep their existing positions:

```tya
class User
  VALUE: "public"
  private SECRET: "secret"

  private id: 0
  static count: 0

  private static normalize: value ->
    value.to_s()

  initialize: name = "guest" ->
    self.name = name

  override label: ->
    self.name
```

- Method declarations may parse with or without parentheses around the
  parameter list:

```tya
class User
  one: name = "guest" ->
    name

  two: (name = "guest") ->
    name
```

- `tya format` emits method declarations without parameter-list parentheses:

```tya
class User
  one: name = "guest" ->
    name

  two: name = "guest" ->
    name
```

- Zero-argument methods are written as `name: ->`.
- Multi-argument methods are written as `name: a, b = 1 ->`.
- Body-free interface requirements are written as `name: ->`.
- `abstract` body-free methods are written as `abstract name: ->`.
- Old class/interface member declaration forms using `=` are invalid in class
  and interface bodies.
- Error messages for old member declaration syntax should be explicit enough
  to point users at `:` syntax.
- Top-level and local function syntax remains assignment-based:

```tya
greet = name -> "Hello, {name}"
handler = (value = nil) -> value
```

- Assignments inside methods remain assignment-based:

```tya
self.name = name
count = count + 1
```

## Scope

- Lexer/parser:
  - parse class and interface member declarations with `:`;
  - accept both parenthesized and unparenthesized method parameter lists in
    class/interface member declarations;
  - reject old `=` class/interface member declarations with targeted
    diagnostics.
- AST/checker/codegen/eval:
  - preserve existing class and interface semantics after the syntax change;
  - update any AST fields or parser paths needed by formatter and self-host
    compiler support.
- Formatter:
  - emit `:` for class and interface members;
  - remove optional method parameter-list parentheses in class/interface member
    declarations;
  - keep formatting idempotent.
- Documentation:
  - update `docs/SPEC.md`;
  - update current versioned specs where this repo mirrors current language
    behavior;
  - update strict semantics wording if needed.
- Self-host compiler:
  - update `selfhost/v01/compiler.tya` and related self-host fixtures as
    needed so the fixed-point invariant remains green.
- Standard library, tests, examples, and docs:
  - migrate class/interface member declarations to `:`;
  - keep ordinary assignment and top-level function syntax unchanged;
  - regenerate stdlib docs if public API rendering changes.
- Editor assets:
  - update syntax grammars, tree-sitter grammar, or VS Code highlighting tests
    if class/interface member parsing or highlighting depends on the old `=`
    syntax.
- Release preparation:
  - because this changes accepted source syntax, bump the compiler release from
    the current patch line to the next minor version;
  - for the current `0.67.x` line, the implementation release target is
    `0.68.0`;
  - update compiler version constants, version command tests, README install
    snippets, and versioned release docs according to the repo's normal minor
    release process.

## Out of Scope

- Changing top-level or local binding syntax.
- Changing ordinary assignment syntax.
- Changing dictionary literal syntax such as `{ name: "Tya" }`.
- Changing method call syntax.
- Changing function literal syntax outside class/interface member declarations.
- Adding named arguments.
- Adding new visibility modifiers beyond existing `private`.
- Changing class member ordering rules beyond what is required for the new
  syntax.
- Preserving old `=` class/interface member syntax as a compatibility alias.

## Acceptance Criteria

- `class User; name: "aaa"` declares the same instance field that
  `name = "aaa"` declared before.
- `class User; VALUE: "aaa"` declares the same class constant that
  `VALUE = "aaa"` declared before.
- `private`, `static`, `abstract`, and `override` members parse and behave with
  `:` syntax.
- `interface Named; name: ->` declares the same method requirement that
  `name = ->` declared before.
- `interface Timestamped; created_at: nil` declares the same interface field
  that `created_at = nil` declared before.
- Method declarations with parenthesized parameter lists parse:
  `method: (aaa = "aaa") ->`.
- `tya format` rewrites parenthesized class/interface method declarations to
  unparenthesized form: `method: aaa = "aaa" ->`.
- `tya format` emits `:` for every class and interface member declaration.
- Old `=` class/interface member declarations fail with targeted diagnostics.
- Top-level assignments and top-level function declarations using `=` still
  parse and run.
- Assignments inside method bodies using `=` still parse and run.
- Existing class, inheritance, interface, private member, class constant,
  abstract, override, formatter, and documentation tests are migrated and pass.
- The self-host v0.1 compiler fixed-point invariant remains green.
- The release-ready implementation reports the new minor version, expected as
  `0.68.0` when implemented from the current `0.67.x` line.

## Verification

```sh
go test ./internal/lexer ./internal/parser ./internal/checker ./internal/formatter -count=1
go test ./internal/eval ./internal/codegen ./internal/doc -count=1
go test ./tests -run 'TestV05Scripts|TestV06Scripts|TestV08Scripts|TestV11Scripts|TestV44Scripts|TestV45Scripts|TestV46Scripts|TestV51Scripts|TestV61Scripts|TestV65Scripts|TestSelfhostV01Scripts|TestFormat' -count=1 -timeout=20m
go run ./cmd/tya version
go test ./... -count=1 -timeout=20m
```
