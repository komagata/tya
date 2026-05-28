# Feature: Struct and Record Definitions

## Goal
Add compile-time `struct` and `record` declarations for lightweight named-field data types, so Tya programs can define simple mutable structures and immutable value records without runtime metaprogramming.

## Context
Tya currently uses `class` for all user-defined object shapes. Classes support fields, constructors, methods, inheritance, interfaces, static members, privacy, and runtime class values. That is expressive, but verbose for plain data carriers.

This feature intentionally does not add Ruby-style runtime class generation. `struct` and `record` are declarations parsed, checked, formatted, documented, and emitted at compile time like `class` and `interface`. They must preserve Tya's current deterministic implementation model: no `Struct.new`, `define_method`, `method_missing`, runtime field synthesis, or dynamic type creation API.

The field syntax should mirror class fields:

```tya
struct User
  name
  age: 0
```

Existing keyword-argument support already allows constructor calls to use parameter names, so this feature should not add a separate keyword-only initialization mode.

## Behavior
- `struct Name` declares a mutable lightweight data type.
- `record Name` declares an immutable lightweight value type.
- `struct` and `record` names use the same PascalCase naming rule as classes and interfaces.
- A `struct` or `record` body may contain only field declarations.
- A field without a default value is a required constructor parameter.
- A field with `: value` has a default value and may be omitted when constructing.
- Field declaration order defines positional constructor parameter order.
- Required fields must precede defaulted fields, matching the existing function/default-parameter rule.
- Constructor calls use the existing call binding rules:

```tya
struct User
  name
  age: 0

user1 = User("komagata", 45)
user2 = User(name: "komagata", age: 45)
user3 = User("komagata")
```

- Unknown constructor keywords, duplicate keywords, positional arguments after keywords, too few required arguments, and too many arguments are invalid using the existing call diagnostics where possible.
- Struct fields are assignable after construction:

```tya
user = User("komagata")
user.age = 46
```

- Record fields are read-only after construction:

```tya
record Point
  x
  y

point = Point(1, 2)
point.x = 3 # invalid
```

- Both structs and records reject assignment to undeclared fields.
- Both structs and records provide structural equality. Two values are equal when they have the same declared type and all declared field values compare equal in declaration order.
- Values of different struct/record types are not structurally equal, even when they have the same field names and values.
- Records provide `with(...)`, which returns a new record value of the same type with selected fields replaced:

```tya
record Money
  amount
  currency: "JPY"

base = Money(1200)
next = base.with(amount: 1500)
```

- `with(...)` accepts only keyword arguments or `**dictionary` keyword expansion under the existing keyword call rules.
- `with(...)` rejects unknown fields and duplicate field updates.
- `with(...)` preserves all fields not explicitly replaced.
- `with(...)` on a record with no changes returns a value equal to the receiver. It does not need to preserve object identity.
- `struct` does not provide `with(...)` in this feature.
- Formatter output is canonical:
  - one blank line between adjacent top-level declarations;
  - field declarations use `name` for required fields and `name: value` for defaults;
  - no method, static, private, inheritance, or interface syntax appears inside `struct` or `record` bodies.
- Class/interface files may use `struct` and `record` file rules equivalent to classes:
  - snake_case filename;
  - exactly one public PascalCase declaration whose name maps to the filename;
  - additional declarations remain private to the file if current class-file companion rules can support them without new visibility semantics.
- Imports expose public structs and records under the same public-name and package rules as public classes/interfaces.
- LSP completion, hover, go-to-definition, document symbols, and diagnostics treat structs and records as first-class compile-time declarations.
- The documentation generator includes structs and records as public API items.

## Scope
- Add `struct` and `record` tokens/keywords and parser support.
- Extend AST representation for struct and record declarations and fields.
- Update checker structure validation, naming validation, file-shape validation, import/export visibility, constructor arity checks, field access checks, assignment checks, and structural equality behavior.
- Update interpreter behavior for struct/record construction, field access, struct field assignment, record assignment rejection, equality, and record `with(...)`.
- Update C code generation and runtime support for struct/record construction, field access, struct field assignment, record assignment rejection, equality, and record `with(...)`.
- Update formatter support for canonical struct/record declarations.
- Update docs in `docs/SPEC.md`, `docs/GUIDE.md`, and `docs/ja/spec.md`.
- Update LSP support for diagnostics, symbols, completion, hover, and go-to-definition where the existing class/interface paths need extension.
- Update documentation generator output for public struct/record declarations.
- Add parser, formatter, checker, interpreter, codegen, testscript, LSP/doc generator tests where appropriate.
- Migrate small stdlib or test helper data carriers only when they are useful focused examples; broad stdlib conversion is not required.

## Out of Scope
- Runtime type generation APIs such as `Struct.new`, `Record.new`, `define_method`, `method_missing`, or dynamic field creation.
- Validation hooks or type annotations for fields.
- Methods inside `struct` or `record` bodies.
- Inheritance, `extends`, `implements`, abstract/final modifiers, static members, class variables, class constants, and `private` members on structs or records.
- Keyword-only constructors.
- Reordering or omitting middle positional constructor arguments.
- `with(...)` on mutable structs.
- User-defined equality overrides for structs or records.
- Destructuring or pattern matching syntax.
- Tuple structs or header-style declarations such as `record User(name, age)`.
- Changing existing class semantics.

## Acceptance Criteria
- `struct User; name; age: 0` parses, formats, checks, runs interpreted, and compiles through C.
- `User("komagata", 45)`, `User("komagata")`, and `User(name: "komagata", age: 45)` construct equivalent `User` values.
- `User(age: 45)` is rejected because `name` is required.
- `User("komagata", age: 45)` is accepted.
- `User(age: 45, "komagata")` is rejected under the existing keyword-call rule.
- `User(name: "a", name: "b")` is rejected.
- `User(unknown: 1)` is rejected.
- `user.age = 46` works for a `struct User`.
- Assigning `user.unknown = 1` is rejected.
- `record Point; x; y` constructs immutable values.
- `point.x = 3` is rejected for a record.
- `Point(1, 2) == Point(x: 1, y: 2)` evaluates true.
- Two different struct/record declaration types with the same field values compare unequal.
- Record `with(...)` returns a new record with selected fields changed and untouched fields preserved.
- `point.with(x: 3)` works, while `point.with(z: 3)` is rejected.
- `point.with(**{ "x": 3 })` works under existing keyword expansion rules.
- `point.with(**{ "z": 3 })` is rejected.
- `struct` values do not have `with(...)`.
- Struct/record declarations cannot contain methods, static members, private members, inheritance, or interface clauses.
- `struct` and `record` cannot be created dynamically at runtime.
- Imports, class-file/public-file rules, LSP features, formatter, and documentation generator all recognize public structs and records consistently.
- Existing class, interface, keyword argument, constructor default, equality, self-host, and stdlib tests continue to pass.

## Verification
```sh
gofmt -w internal/**/*.go cmd/**/*.go tests/**/*.go
go test ./internal/lexer ./internal/parser ./internal/checker ./internal/eval ./internal/codegen ./internal/formatter -count=1
go test ./tests -run 'struct|record|TestSelfhostV01Scripts' -count=1
go test ./... -count=1
```
