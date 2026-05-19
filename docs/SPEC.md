---
layout: doc
title: Spec
permalink: /spec/
---

# Tya Language Specification

Status: current repository specification. This page describes the language
surface maintained on `main`, including the current package, tooling,
concurrency, interface, and standard-library integration rules.

## Overview

Tya is an indentation-based, dynamically typed language that compiles to C.
The implementation is intentionally small and explicit: source is tokenized,
parsed into an AST, checked, emitted as C, and linked with the Tya runtime.

Tya's user-facing commitments are:

- canonical source formatting through `tya format`;
- strict dynamic semantics with no implicit conversions;
- a compile-to-C runtime model;
- one all-in-one `tya` command for running, building, checking, formatting, testing, linting, documenting, packaging, and editor support;
- structured diagnostics with stable codes;
- a maintained self-hosting path.

This document specifies the language, built-in function surface,
standard-library surface, package rules, and tool surface.

## v1.0.0 Compatibility Boundary

Tya v1.x compatibility covers accepted syntax, documented runtime behavior,
documented public standard-library and package APIs, stable diagnostic codes,
the CLI JSON diagnostic schema, and release artifact semantics described in
this specification. Undocumented implementation details, generated C internals,
internal Go package layout, unsupported experimental flags, external package
internals, and post-v1 ecosystem packages are not compatibility guarantees.

The self-hosted compiler is the primary compiler direction for v1.0.0. The Go
implementation remains in the repository as a reference implementation and
bootstrap recovery path until the no-Go self-host bootstrap proof replaces that
recovery role.

Specification authority for v1.0.0 is, in order: this specification and the
frozen `docs/v1.0/SPEC.md`; the latest self-host compiler behavior when it
implements the documented v1 surface; then the Go implementation as reference
and bootstrap recovery path. If the Go implementation and latest self-host
compiler disagree, the behavior matching the v1 specification is authoritative.

The strict-semantics rule matrix in
[`docs/STRICT_SEMANTICS.md`](STRICT_SEMANTICS.md) is normative for v1.0.0
validity boundaries and records the active parser, checker, runtime, CLI, LSP,
and self-host coverage for each rule family.

Completed feature specs under `feature-specs/completed/` are design history.
They may explain why a rule was accepted, but users do not need them to know
the current v1.0.0 contract. Public v1 authority is this specification,
[`docs/STRICT_SEMANTICS.md`](STRICT_SEMANTICS.md), and the frozen documents
under `docs/v1.0/`.

## Source and Lexical Structure

Examples use ordinary Tya source. Grammar fragments are illustrative rather
than a complete parser grammar.

```text
snake_case            variable, function, method, import path segment
SCREAMING_SNAKE_CASE  constant
PascalCase            class and interface
```

The words "must", "must not", "may", and "should" are normative when they
describe program validity or implementation behavior.

### Naming

Tya names express naming category, not accessibility. Accessibility is
expressed by language constructs such as `private`.

Value names, function names, method names, file names, import path segments,
and dictionary keys use `snake_case`. Constants use `SCREAMING_SNAKE_CASE`.
Classes and interfaces use `PascalCase`.

Single-file imports use the source filename without `.tya` as the import path
segment. Import paths are slash-separated `snake_case` segments. Leading `_`
has no visibility meaning for ordinary bindings. Standard-library APIs use
`snake_case`; CamelCase builtin spellings are not part of the language surface.

### Source Code Representation

Tya source is UTF-8 text. The compiler normalizes CRLF line endings to LF before
lexing. Source files use `.tya`.

Indentation defines blocks. Spaces are the indentation unit. Tabs are forbidden
in source indentation and in heredoc body indentation.

```tya
if ready
  print("ready")
else
  print("not ready")
```

Each physical line is part of one logical line except when it is inside a
parenthesized call, an array literal, a string literal, or a canonical
continuation form accepted by the parser and formatter.

### Lexical Elements

### Comments

Line comments begin with `#` and continue to the end of the line.

```tya
# file header comment
name = "tya" # line-end comment
```

Comments may attach to declarations and statements for formatting, LSP hover,
and `tya doc`. Comment placement rules are part of canonical syntax.

Tya recognizes three source comment roles:

- file header comments at the beginning of a file;
- leading comments immediately attached to the following declaration or statement;
- line-end comments attached to the preceding statement.

Comments in positions with no definite attachment target are invalid. A block
whose body contains only comments is invalid because it has no executable or
declarative body item.

### Tokens

The token vocabulary includes identifiers, literals, indentation tokens,
operators, and punctuation.

```text
= == != < <= > >= : , . ? @ + - * / % ->
( ) [ ] { }
& | ^ ~ << >>
```

Whitespace separates tokens. Newlines are significant because they terminate
statements and define indentation blocks.

### Identifiers

Identifiers are ASCII-oriented by convention and by the current naming rules.
Public variable, function, method, file, and import path names use
`snake_case`. Class and interface names use `PascalCase`. Constants use
`SCREAMING_SNAKE_CASE`.

The following words are reserved in positions where ordinary names are parsed:

```text
abstract and as await break case catch class continue default else elseif embed
extends false final for if implements import in interface module nil not or
override private raise return scope select self Self spawn static super true try
while with
```

Some words are context-sensitive. For example, `as` is meaningful in imports,
`extends` and `implements` are meaningful in class and interface headers, and
`case`, `default`, `send`, `receive`, and `timeout` are meaningful inside
`select`.

### Literals

Tya has literals for `nil`, booleans, numbers, strings, bytes, arrays, and
dictionaries.

```tya
missing = nil
ready = true
count = 42
ratio = 3.14
name = "Tya"
data = b"abc"
items = [1, 2, 3]
user = { name: "komagata", age: 20 }
```

String literals use double quotes. Strings support interpolation with `{...}`.

```tya
print("Hello, {user["name"]}")
```

Triple-quoted strings and heredoc forms are available for multi-line text.
Raw and byte heredoc forms preserve their documented escaping behavior. The
formatter treats multi-line strings as atomic except where canonical syntax
defines a rewrite.

Byte literals use `b"..."` or byte heredoc forms and produce byte values rather
than strings.

Integer literals may be written in decimal, hexadecimal, or binary form.
Floating-point literals use decimal notation. `NaN`, `Infinity`, `nan`, and
`infinity` are ordinary identifiers when used as names; they are not numeric
literal spellings.

Tya `String` values are UTF-8 text. Text file APIs that return strings reject
invalid UTF-8. Binary file APIs return `Bytes` and do not validate UTF-8.

## Values And Kinds

Tya is dynamically typed. Values carry a runtime kind. The core runtime kinds
are:

```text
nil
bool
number
string
bytes
array
dict
function
class
object
error
task
channel
resource
```

Arrays and dictionaries are mutable. Strings and bytes are separate value
kinds. Classes are runtime values; object values are instances of classes.

Primitive values expose methods through runtime wrapper classes and standard
builtins.

```tya
print(" tya ".trim().upper())
print([1, 2, 3].len())
print({ name: "tya" }.keys())
print(value.class)
```

Tya does not perform implicit conversions. Operations that require a number,
string, array, dictionary, function, class, task, channel, or resource must
receive a value of the required kind or raise a runtime error. The documented
exceptions are formatting operations such as string interpolation and `to_s()`,
plus the exact operator cases listed below.

### Blocks

A block is a non-empty sequence of statements introduced by a header line and
an increased indentation level. Empty blocks are invalid; use an explicit
expression such as `nil` for an intentional no-op body.

```tya
while count < 3
  print(count)
  count = count + 1
```

Bindings created inside `if`, `while`, `for`, `catch`, `match case`, `scope`,
and `select` bodies are local to that body. Assigning to an existing outer
non-function binding from such a nested block updates the outer binding.

Blocks appear in control-flow statements, function bodies, class bodies,
interface bodies, `try` / `catch`, `scope`, `select`, and similar constructs.

Top-level source consists of imports, declarations, assignments, and statements
allowed by the file kind. Class files are more restrictive than script files.

### File Kinds

A `.tya` file's role is determined by its filename and context.

Lowercase `.tya` files are script files. They may be entry files for `tya run`
and may also be imported directly. When imported, their top-level names are
exposed through the import binding.

PascalCase `.tya` files are class files. They are library-only and cannot be
entry files. A class file must declare exactly one public class whose name
matches the filename without `.tya`.

Class files may be loaded explicitly as part of a directory package or
implicitly as same-directory siblings of an entry script. A script entry sees
PascalCase class files in its own directory without import.

### Canonical Syntax {#canonical-syntax}

Tya has a canonical syntax: every well-formed program has one source
representation. `tya format` is therefore part of the language surface, not an
optional style tool.

Canonical syntax covers indentation, blank lines, comment attachment, line
wrapping, import grouping, operator spacing, string literal forms, empty
collection forms, and other source-shape decisions. The formatter is the
canonical serializer and has no style configuration.

The core canonical rules are:

- indentation is two spaces; tabs are invalid in source;
- the column limit is 80, except for one unbreakable atomic token;
- comments must be file header comments, leading comments, or line-end
  comments with a definite attachment target;
- blank lines are determined by AST shape, not user preference;
- multi-line calls, arrays, dictionaries, parameter lists, operator chains, and
  long conditions use the formatter-defined continuation forms;
- trailing commas are prohibited in arrays, dictionaries, calls, and parameter
  lists;
- imports are atomic and not line-wrapped;
- `elseif` is the canonical spelling, and `else if` is not canonical;
- `case _` in `match` is the wildcard case and must be final;
- empty collection forms and empty `else` branches follow formatter-defined
  shapes.

Implementations must preserve semantic behavior when formatting. Formatting
must be idempotent and stable across platforms.

### V1 Language Boundaries

Tya v1.0.0 intentionally keeps the syntax surface small. The following forms
are not part of v1.0.0 and must fail before code generation:

- slice syntax such as `items[1:3]`, `items[:3]`, `items[1:]`, and stepped
  slices; use explicit `.slice(...)` methods;
- named or keyword arguments such as `request(url, timeout: 10)`; pass
  dictionary options such as `request(url, { timeout: 10 })`;
- variadic parameter syntax and splat calls such as `fn = *args -> args` and
  `fn(*items)`; pass arrays explicitly;
- destructuring assignment with array or dictionary patterns such as
  `[a, b] = items` and `{ name } = user`; use multi-return assignment such as
  `a, b = pair()`;
- match guards and binding patterns such as `case value if ready` and
  `case [head, tail]`; match patterns remain literals, `nil`, booleans,
  `case _`, and array/dictionary structure patterns that do not introduce
  names;
- operator overloading, function overloading, and method overloading; use
  explicit method names, default parameters, or dictionary options;
- generic type-parameter syntax such as `Array<Int>`, `Box<T>`, `fn<T>`, and
  type-argument calls;
- public `module` declarations; use script files, class files, directory
  packages, and import aliases;
- dedicated `enum`, `record`, or `struct` declarations; use classes,
  dictionaries, constants, or standard-library value classes;
- macro and general compile-time metaprogramming syntax; `embed` remains a
  dedicated declaration;
- async function coloring such as `async fn`; use `spawn`, `await`, `scope`,
  `select`, tasks, and channels;
- visibility modifiers beyond public and `private`, including `protected` and
  `friend`.

## Declarations And Scope

### Bindings

Assignment creates or updates bindings.

```tya
name = "Tya"
count = count + 1
```

Reassignment must preserve the binding's runtime kind, except that `nil` may
move to or from a concrete kind because it represents absence. Assigning `nil`
does not erase the last known concrete kind. A name first assigned a number may
later receive another number or `nil`, but not a string, array, dictionary,
function, class, object, error, or resource value. This keeps Tya dynamically
typed while making rebinding strict and predictable.

```tya
count = 1
count = 2      # valid
count = "two"  # invalid

err = nil
err = error("failed") # valid
```

Multiple assignment is supported.

```tya
min, max = bounds(items)
```

Leading `_` has no visibility meaning for ordinary bindings. Top-level
privacy is not expressed by name spelling.

Constants use `SCREAMING_SNAKE_CASE` and are checked as constants by naming
and assignment rules. Constants cannot be reassigned. Heap-backed values stored
in constants are also immutable through that constant binding.

Class member privacy uses the `private` keyword for private class fields,
methods, class variables, class methods, and constructors.

```tya
class User
  private id = 0

  private normalize = ->
    self.id.to_s()
```

### Embedded Assets

`embed` declares a top-level binding whose value is loaded from a file at build
time. Embed declarations are resolved relative to the source file.

```tya
embed "templates/index.html" as index_html
```

Embed transforms are implementation-defined by the compiler surface and must
produce ordinary Tya values.

### Functions

Functions are values. Function literals use `->`.

```tya
greet = name -> "Hello, {name}"

double = value ->
  result = value * 2
  result
```

Calls always use parentheses.

```tya
print(greet("Tya"))
```

The final evaluated statement or expression in a function body is returned
implicitly when no explicit `return` exits first. Use `return` for early return
or multiple return values.

```tya
parse_user = text ->
  if text == ""
    raise error("empty user")
  { name: text }
```

Parameters are local bindings. `_` may be used for intentionally ignored
parameters.

### Classes

A class declares a runtime class value.

```tya
class User
  initialize = name ->
    self.name = name

  label = ->
    "user:{self.name}"
```

Instances are constructed by calling the class.

```tya
user = User("komagata")
print(user.label())
```

`initialize` is the constructor hook. Instance methods receive `self`.
Instance fields are created by assignment to `self.<name>`.

Tya supports:

- single class inheritance with `extends`;
- constructor and method delegation with `super(...)`;
- `private` members;
- `static` class methods and class variables;
- `abstract class` and abstract methods;
- `final class`;
- `override` for explicit method override checks;
- runtime class inspection through `.class`;
- read-only class metadata members such as `class`, `class_name`, `name`, and `parent` where documented by the runtime.

```tya
class Admin extends User
  initialize = name ->
    super(name)

  override label = ->
    "admin:{self.name}"
```

A class file is a PascalCase `.tya` file. It must declare exactly one public
class whose name matches the filename. It may also declare private helper
classes and interfaces. Class files are library files and cannot be run as
entry scripts.

Additional classes in a class file are private to that file. They are not
visible from other files, even inside the same directory package.

### Interfaces

Interfaces are explicit contracts and stackable behavior units.

```tya
interface Named
  name = ->

  label = ->
    self.name()
```

An interface may contain:

- body-free instance method requirements;
- default instance methods;
- field declarations;
- a zero-argument `initialize` hook.

An interface may not contain static members, private members, nested classes,
or nested interfaces. `Self` is invalid inside interface methods.

Classes list implemented interfaces with `implements`.

```tya
interface Timestamped
  created_at = nil

  initialize = ->
    self.created_at = Time.now()

class Account implements Named, Timestamped
  initialize = name ->
    self.name_value = name
    super()

  name = ->
    self.name_value
```

Default methods are inherited when the class does not define a method with the
same name. A class method wins over an interface default. Interface defaults
stack in the declared `implements` order and may call `super()`.

Interface fields contribute instance state. A class that implements multiple
interfaces must not receive conflicting field definitions. If a class
constructor implements interfaces with initializer hooks, it must call
`super()` exactly where it wants the interface initialization chain to run.

Interface conflict rules are strict:

- duplicate requirements collapse to one requirement;
- a default method can satisfy a requirement;
- unrelated defaults for the same method are ambiguous unless the class overrides the method;
- arity conflicts are errors;
- initializer order is deterministic and follows class inheritance before newly implemented interfaces.

Interfaces declared in class files are exported as package public names unless
their names begin with `_`.

`Comparable` is the standard ordering protocol. A class implements it by
providing `compare(other)`, which returns a negative number, zero, or a
positive number when the receiver sorts before, equal to, or after `other`.
Numbers conform to `Comparable` as primitive values. Strings do not define
ordering operators; string ordering, collation, or locale-aware comparison must
use explicit methods or standard-library APIs. The ordering operators `<`,
`<=`, `>`, and `>=` keep their primitive numeric behavior and do not dispatch
to user-defined `compare`.

`Equatable` is the standard domain equality protocol. A class implements it by
providing `equal?(other)`, which must return a boolean. Primitive values expose
`equal?`; scalar primitives follow `==`, while arrays and dictionaries use
deep equality. The `==` operator and top-level `equal(left, right)` keep their
existing behavior and do not dispatch to user-defined `equal?`.

`Stringable` is the standard human-readable formatting protocol. A class
implements it by providing `to_s()`, which returns a string and should be
side-effect free for ordinary formatting use. Number, String, Array, Dict,
Boolean, and Nil conform to `Stringable` as primitive values without changing
their tagged runtime representation or `value.class` behavior. `Stringable` is
not a structured serialization protocol; use `Serializable.to_data()` for data
trees.

The standard library also defines protocol interfaces for iteration,
sequences, I/O, and structured data:

- `Iterator` requires `has_next()` and `next()`;
- `Iterable` requires `iter()` and provides `sequence()`;
- `Sequence implements Iterable` and provides lazy-style `map(fn)`,
  `filter(fn)`, `take(n)`, `drop(n)`, `reduce(initial, fn)`, and `to_a()`;
- `Readable` requires `read(size)`;
- `Writable` requires `write(data)`;
- `Closable` requires `close()`;
- `Flushable` requires `flush()`;
- `Serializable` requires `to_data()`.

Arrays, dictionaries, and strings conform to `Iterable` as primitive values.
`for ... in` consumes primitive iterables directly and consumes user-defined
iterables through `iter()`. I/O protocol interfaces are defined in the relevant
standard-library packages such as `io` and `net/socket`; they document shared
stream behavior and are implemented by concrete reader, writer, socket, and
server classes where their methods match.

## Expressions

Expressions compute values.

### Primary Expressions

Primary expressions include identifiers, literals, parenthesized expressions,
function literals, indexing, member access, calls, `self`, `Self`, and
`super`.

```tya
user["name"]
items[0]
user.label()
User("komagata")
self.name
super(name)
```

`self` is available inside instance methods and constructors. `Self` refers to
the current class in class contexts where it is valid. `super(...)` delegates
to the parent constructor, parent method, or next stacked interface method
depending on context.

Function literals are lexical closures. A function literal may read parameters
and local bindings from enclosing function bodies. Captures are value snapshots
created when the function literal is evaluated; heap-backed values such as
arrays, dictionaries, objects, functions, resources, and tasks are captured as
values, not deep-copied. Top-level names are not captured and continue to use
module/global lookup.

Function bodies cannot write back to outer function bindings. Direct
reassignment of an outer function binding is invalid, and indexed or member
assignment through a captured outer function binding is invalid. Pass mutable
state as an explicit parameter when a function is intended to mutate that
value.

Each evaluation of a function literal creates an independent closure
environment.

Function literals themselves have no declaration name. When a function value
is assigned to a binding, the binding may be used as a debugging/display name;
that name does not affect equality, identity, or call behavior.

Function parameters may have default values. Required parameters must precede
defaulted parameters, calls may omit only a trailing run of defaulted
parameters, and too few required arguments or too many arguments are invalid.
Default expressions are evaluated at call time, left to right, and may
reference earlier parameters but not later parameters. Mutable defaults are
fresh because the default expression is re-evaluated for each omitted argument.
Variadic parameter syntax is not part of Tya; pass an array explicitly when a
function needs a variable number of values.

```tya
make_adder = base ->
  value -> base + value

add_two = make_adder(2)
add_ten = make_adder(10)

print(add_two(3))
print(add_ten(3))
```

Reassigning the original local after a closure is created does not change the
captured value.

```tya
make_label = name ->
  label = -> name
  name = "changed"
  label

print(make_label("first")())
```

Mutating through a captured binding is invalid. The closure must receive the
mutable value as a parameter if mutation is intended.

```tya
make_bad = items ->
  ->
    items[0] = "changed" # invalid: cannot mutate captured binding
```

### Operators

Tya supports arithmetic, comparison, logical, and bitwise operators.

```text
or
and
not
== != < <= > >=
| ^ &
<< >>
+ -
* / %
```

Logical operators use words: `and`, `or`, and `not`.

```tya
if ready and not disabled
  print("ok")
```

Arithmetic operations require numbers unless a documented primitive method or
operator case says otherwise. `+` adds two numbers, concatenates two strings,
and concatenates two bytes values. `+` does not format mixed operands through
implicit string conversion. String interpolation formats embedded values with
the display surface. `/` always performs number division, so `5 / 2` evaluates
to `2.5`; integer division uses an explicit API such as `div()`. `%` is
integer-only and is invalid for floating-point operands. `nil` arithmetic is
invalid.

`and` and `or` return booleans. They test operands with Tya truthiness, do not
return either operand as a value, and short-circuit: `and` skips the right
operand when the left operand is falsey, while `or` skips the right operand
when the left operand is truthy.

Method-call receivers are evaluated exactly once before method lookup and
argument evaluation continues.

Bitwise operators require integer-compatible number values.

Equality operators may compare any two runtime values without coercion. Values
with different runtime kinds compare unequal. Arrays and dictionaries compare
by contents; functions, classes, objects, resources, tasks, and channels
compare by identity unless their documented primitive surface says otherwise.
Numeric int and float values compare as one number kind, so `1 == 1.0` is
true. Ordering operators `<`, `<=`, `>`, and `>=` require numbers; string
ordering is not defined by these operators.

Deep equality on cyclic arrays or dictionaries is a runtime error. Display of
cyclic arrays or dictionaries must terminate with a stable cycle marker.

### Collections

Arrays use bracket literals and integer indexing.

```tya
items = ["a", "b"]
items.push("c")
print(items[0])
```

Dictionaries use brace literals. Identifier keys and string-literal keys in
dictionary literals are stored as string keys. String-literal keys support
JSON-style names such as `"Content-Type"`, `"$schema"`, `"1"`, and `""`.
Duplicate keys are invalid after normalizing identifier and string-literal
forms to strings.

```tya
user = { name: "komagata", age: 20 }
print(user["name"])
user["admin"] = true
```

Dictionary block forms and empty collection forms are canonicalized by the
formatter.

Dictionary keys are read and written with string indexes. Dot access on
dictionaries is reserved for documented dictionary receiver methods such as
`keys()`, `has?()`, `get()`, `set()`, and `delete()`; dictionary key member
access such as `user.name` is invalid.

Array, string, and bytes indexes must be integers. Dictionary and error-value
indexes must be strings. Missing dictionary keys and out-of-range array,
string, or bytes indexes return `nil`; indexing a non-collection target is
invalid. Negative indexes are invalid for arrays, strings, and bytes. Slice
syntax such as `items[1:3]` is not part of the language; use explicit methods
such as `items.slice(1, 3)`. String indexing is by Unicode rune. Bytes indexing
is byte-based.

Array index assignment requires an existing array index and fails on
out-of-range writes. Dictionary index assignment may create a new key.
Arrays and dictionaries are mutable. Strings and bytes are immutable.
`Array.push`, `Dict.set`, and `Dict.delete` return `nil`. `Array.pop` returns
the removed value, or `nil` when the array is empty.

### Error Values

`error(message, options = {})` creates an error value. `message`, `kind`, and
`code` are strings; `data` is a dictionary; and `cause` is another error value
or `nil`. Unknown option keys are invalid. Error display uses `message`.

```tya
err = error("not found", {
  kind: "io",
  code: "file_not_found",
  data: { path: "missing.txt" }
})
print(err["message"])
print(err["kind"])
print(err["code"])
```

### Concurrency Expressions

`spawn` starts a task and returns a task value. `await` waits for a task and
returns or re-raises its result.

```tya
task = spawn work(21)
print(await task)
```

Channels and sync resources are standard-library-backed runtime values with
the methods specified in the Standard Library section.

### Parallelism And Concurrency

Tya exposes structured concurrency through tasks, scopes, channels, sync
resources, and `select`.

Tasks are lightweight runtime values created by `spawn`. `await` joins a task.
Awaiting a completed task returns the cached result or re-raises the cached
error.

`scope` defines a structured lifetime for tasks spawned inside it. A scope
waits for its child tasks before leaving the region.

Channels and sync resources are implemented by the runtime and surfaced through
standard-library classes and methods. `select` waits across channel send,
receive, timeout, and default branches.

The runtime may run tasks in parallel where the target platform and runtime
support it. Program correctness must not depend on a specific scheduling order
except where the language or standard library documents an ordering guarantee.

## Statements

### Expression Statements

Calls and other useful expressions may appear as statements.

```tya
print("hello")
save_user(user)
```

### Assignment Statements

Assignment updates a binding, field, or indexed collection entry.

```tya
name = "Tya"
self.name = name
items[0] = "first"
user["admin"] = true
```

Multiple assignment evaluates the right-hand side and binds the corresponding
left-hand targets. Right-hand expressions are evaluated first, left to right;
after that, assignment targets are evaluated and assigned left to right.

### If Statements

`if`, `elseif`, and `else` select among blocks.

```tya
if age >= 20
  print("adult")
elseif age >= 13
  print("teen")
else
  print("young")
```

`elseif` is the canonical spelling. `else if` is not canonical Tya.
Only `nil` and `false` are falsey. All other values, including `0`, `""`,
empty arrays, and empty dictionaries, are truthy.

### While Statements

`while` repeats while its condition is truthy.

```tya
while count < 3
  print(count)
  count = count + 1
```

### For Statements

`for ... in` is the canonical way to consume iterable values. Arrays yield
elements, strings yield characters, dictionaries yield `{ key: key, value:
value }` entry dictionaries, and user values participate by exposing `iter()`.
The second binding receives a zero-based index when present.

```tya
for item in items
  print(item)

for item, index in items
  print("{index}: {item}")

for entry in user
  key = entry["key"]
  value = entry["value"]
  print("{key}: {value}")
```

`for` evaluates to the last completed body value. An empty loop evaluates to
`nil`. `break` exits the nearest loop and leaves the loop value as the last
completed body value before the break, or `nil` if there is none. `continue`
skips to the next iteration and discards the current iteration's partial value.

### Match Statements

`match` selects one `case` block by comparing an expression to case patterns.
`case _` is the wildcard case and is canonical only as the final case.

```tya
match value
case "ok"
  print("ok")
case _
  print("other")
```

### Return Statements

`return` exits the current function or method. It may return zero, one, or
multiple values.

```tya
return
return value
return min, max
```

### Raise, Try, And Catch Statements

`raise` raises an error value. `raise nil` and other non-error values are
invalid. `try` is a statement only; try expressions are not part of v1.0.0.
`catch err` is the only catch syntax and catches raised error values. Typed
catch, pattern catch, catch filters, and multiple catch clauses are invalid.
Branch by error details inside the `catch` body with `if` or `match`.
`finally` may follow `try/catch` or a bare `try` block. A bare `try` with
neither `catch` nor `finally` is invalid.

`finally` always runs as control leaves the `try` or `catch` body, including
normal completion, `return`, `raise`, `break`, and `continue`. The value of a
`finally` body is ignored for ordinary completion. If the `finally` body
performs its own control flow, that control flow replaces any pending control
flow from the `try` or `catch` body.

```tya
try
  save_user(user)
catch err
  print("save failed: {err}")
finally
  cleanup()
```

### Scope Statements

`scope` defines a structured concurrency region. Tasks spawned inside the
scope are joined according to the runtime scope rules before the scope exits.

```tya
scope
  task = spawn work()
  print(await task)
```

### Runtime Boundaries

`defer`, language-level `assert`, and language-level cancellation syntax are
not part of v1.0.0. Cleanup is written with `try/finally` and explicit resource
methods such as `close()`. Programs must not rely on GC finalizers for
correctness.

Channels have fixed closed-state behavior: receiving from a closed channel
returns `nil`, and sending to a closed channel raises an error. Task lifetime
is structured by `scope`, which waits for child tasks before leaving. v1.0.0
does not define a cancellation statement or a language-level cancellation
token; cancellation helpers are ordinary library APIs where documented.

### Select Statements

`select` waits on channel operations, timeouts, and default branches.

```tya
select
case value = receive ch
  print(value)
case send ch, next
  print("sent")
timeout 1
  print("timeout")
default
  print("none")
```

The exact channel methods and sync primitives are defined in the Standard
Library section.

### Built-In Functions

Tya keeps the public builtin surface intentionally small. File, directory,
path, process, stream, bytes, random, compression, digest, socket, compiler,
and collection helper operations are exposed through class-style standard
library APIs. Low-level runtime intrinsics may exist internally to implement
those classes, but they are not public standalone builtins.

Public builtins:

```text
print(value)
println(value)
error(message)
exit(status)
args()
env(name)
```

`print` and `println` write only to stdout. stderr output is available through
explicit standard-library APIs such as `Io.stderr()`.

Use standard-library APIs such as `File.read(path)`, `File.append(path, text)`,
`Dir.list(path)`, `Path.expand_user(path)`, `Process.cwd()`,
`Process.chdir(path)`, `Io.open(path, mode)`, `Reader#read(size)`,
`Writer#write(value)`, `Random.int(min, max)`, `Compress.gzip(value)`,
`Digest.sha256(value)`, `Socket.connect(host, port, options)`,
`Lexer.lex(source)`, `Parser.parse(source)`, `Checker.check(source)`, and
`Format.format(source)` instead of low-level intrinsic names. Use receiver
methods for conversions and collections, for example `value.to_s()`,
`value.to_i()`, `dict.delete(key)`, `dict.keys()`, and `items.pop()`.

Standard library APIs are imported with the same `import` syntax as user code.

## Imports and Packages

Current Tya documentation uses these terms normatively:

```text
language feature             syntax or semantics built into Tya
built-in function            function available without import
built-in class               class available without import; none currently
user package                 importable directory of PascalCase class files
user library                 reusable directory tree of user packages
standard-library package     .tya source shipped with Tya and imported normally
bundled library              library or support file shipped with the toolchain
native-backed stdlib module  importable stdlib API backed by runtime or host code
package                      versioned dependency unit declared by tya.toml
package tool                 [tools] entry run by tya tool
```

Language features are not imported and cannot be shadowed. Standard-library
packages are specified in the Standard Library section; they are imported
packages, not builtins.

### Import Syntax

Imports appear at top level before other declarations and statements.

```tya
import greeting
import net/http
import net/http as http
```

Import paths are slash-separated `snake_case` segments. Relative filesystem
paths, absolute paths, empty segments, and PascalCase terminal segments are
invalid.

### Directory Packages

A directory package is a directory resolved by import path containing
PascalCase class files. It must contain at least one class file and must not
contain lowercase script files at the package leaf.

Unaliased directory imports expose public class and interface names directly.
They do not create a lowercase namespace binding for the import path or terminal
directory segment.

```tya
import net/http

server = Server()
http = "local label"
```

In the example above, `http` is an ordinary local binding. `http.Server()` is
invalid because unaliased directory imports do not expose a namespace object.
If a namespace is desired, use an alias.

Aliased directory imports expose a namespace binding and do not import public
names bare.

```tya
import net/http as http

server = http.Server()
```

With an alias, package public names are only available through the alias
namespace. `Server()` is invalid in the example above; use `http.Server()`.

Imported public names are reserved in the importing scope. Reassigning or
redeclaring an imported public class or interface name is invalid, and importing
two packages that expose the same public name is invalid. Importing the same
path twice is invalid even when aliases differ.

```tya
import net/http

Request = -> "local" # invalid: Request is imported from net/http
```

Within the same directory package, sibling public classes are visible by bare
PascalCase name without import.

The public API of a directory package is the set of public classes and
interfaces in its PascalCase class files. A class is public when its class name
matches its filename. Additional classes in a class file are private to that
file.

### User Libraries

A user library is a directory tree of importable source intended for reuse. It
does not require a manifest. A library root may be made available through
`TYA_PATH`.

```sh
TYA_PATH=libs/web tya run app.tya
```

`TYA_PATH` entries are import roots, not relative import syntax. Source inside
a user library should use the same import paths that applications use.

### Resolution Order

Imports are resolved in this order:

1. the importing file's directory;
2. manifest-declared dependencies from `tya.lock`;
3. directories listed in `TYA_PATH`, from left to right;
4. the bundled `stdlib/` directory.

The first matching file or package directory wins. Local application imports
may shadow package dependencies, `TYA_PATH`, and standard-library imports.
Package dependencies may shadow `TYA_PATH` and standard-library imports.

### Package Manifests

`tya.toml` declares package metadata, dependencies, native wrappers, and
package-provided tools. Unknown top-level keys are errors so typos are not
silently ignored. Package manifests require `name` and `version`; `license` is
recommended package metadata and is preserved when present.
`tya install` resolves dependencies and writes `tya.lock`. Git and explicit
local path dependencies are supported. Registry-style implicit package source
discovery is not supported. There is currently no central package registry and
no `tya publish` command.

A package is a versioned distribution unit for reusable Tya code. Package code
normally exposes importable source under `src/`. Applications consume packages
through manifest dependencies:

```toml
[dependencies]
my_lib = { git = "https://github.com/example/my_lib", tag = "v0.1.0" }
local_lib = { path = "../local_lib" }
```

`tya.lock` records resolved dependency sources, revisions, and content hashes
where applicable, and should be committed by applications. When `tya.lock`
exists, it is authoritative for dependency versions and sources. If `tya.toml`
and `tya.lock` disagree, commands that resolve package imports fail with a
stale-lock diagnostic and instruct the user to run `tya install` before the
changed dependency graph is used. If cached package content does not match the
locked hash, import resolution fails and instructs the user to run
`tya install`.

Native package metadata lives under `[native]`. Native paths are relative to
the package root. `tya build`, `tya run`, and `tya test` compile declared C
sources with generated C, the Tya runtime, include directories, `pkg-config`
flags, `cflags`, and `ldflags`. Native wrapper functions use the Tya runtime
ABI and are called from package code like predeclared functions inside that
package. Native packages are declarative; arbitrary shell build scripts in
manifest metadata are not supported. Native packages compile and link trusted
C/native code and are outside the safe-analysis trust boundary.

Package-provided tools live under `[tools]` and run through `tya tool`.
Package tools are not global installs and are not shell tasks; they run from
locked dependencies or an explicit one-shot git/path source.

## Runtime and Concurrency

A script file is a lowercase `.tya` file and may be used as an entry file for
`tya run`, `tya build`, and `tya emit-c`.

```sh
tya run hello.tya
tya build hello.tya -o hello
tya emit-c hello.tya
```

Class files are library-only and cannot be entry files.

Tya uses a compile-to-C pipeline for native execution. `tya run` compiles a
temporary native executable, runs it, and removes the temporary executable.
`tya build -o PATH` writes a reusable executable to `PATH`; without `-o`,
`tya build` writes an executable in the current directory using the source
basename. Intermediate native build artifacts live under `.tya/build/`.
`tya emit-c` prints or writes the C program generated from Tya source. The
generated C links against the Tya runtime.

The default native target uses the Tya-managed Zig toolchain as `zig cc`.
Native package metadata from `[native]` contributes C sources, headers,
include directories, `pkg-config` flags, compiler flags, and linker flags to
the build.

WASM build targets are available where supported. Native packages are rejected
for unsupported WASM targets. `tya run` remains native-only.

### Cross Compilation {#cross-compilation}

Cross-compilation is selected with `--target` on `tya build`. The native target
is the default and uses the Tya-managed Zig toolchain as `zig cc`.
WebAssembly targets produce artifacts for a different execution environment
without running the program.

Current targets include:

- `native`, the host native executable target;
- `wasm32-wasi`, a WASI `.wasm` artifact for WASI runtimes;
- `wasm32-browser`, a browser-oriented `.wasm` artifact and JavaScript loader.

Typical commands:

```sh
tya build --target native src/main.tya -o app
tya build --target wasm32-wasi examples/wasm/hello.tya -o hello.wasm
tya build --target wasm32-browser examples/wasm/hello.tya -o hello.wasm
```

`tya doctor wasm` reports the WebAssembly build environment and selected Zig
path/version. `tya doctor native` reports the native build environment and
selected managed `zig cc` path/version. Native package metadata may contribute
C sources and linker flags for native builds, but packages with unsupported
native requirements are rejected for unsupported WebAssembly targets.

WebAssembly builds preserve the compile-to-C backend and use the same Zig
resolver as native builds. The first WebAssembly target layer supports
stdout-oriented smoke programs. Browser builds also reject filesystem and
process-oriented imports. `tya run` is native-only and does not execute
WebAssembly artifacts. WebAssembly is documented for v1.0.0 but is not a
release-blocking target; WASM-specific gaps are tracked separately from the
core language release gates.

## Errors and Diagnostics

Tya has two related error systems:

- language-level `raise`, `try`, and `catch` for program errors;
- compiler and tool diagnostics for invalid source and tooling failures.

Compiler diagnostics use stable codes such as `TYA-E0015` and linter
diagnostics use stable codes such as `TYAL0001`. Diagnostics should include
an actionable message and, where practical, a hint and documentation URL.

Runtime kind errors, invalid operations, failed assertions, failed I/O, and
native wrapper errors are represented as Tya error values or raised runtime
errors according to the API being used.

Standard-library failure behavior is part of each public API contract. Invalid
argument kinds or arity are runtime errors. Absence and lookup APIs return
`nil` only where documented. Operations that can fail because of the outside
world, such as file, process, network, parse, compression, digest, time,
random, native-wrapper, and serialization APIs, raise structured error values.
Public v1.0.0 APIs do not use `value, err` pairs as the failure convention.
Multiple return values remain valid for ordinary successful values.

`raise` may raise only error values. Raising `nil`, strings, numbers,
dictionaries, or other non-error values is invalid because they do not carry the
language error kind. A `catch` binding receives the exact raised error value.

## Built-In Tools {#builtin-tools}

The `tya` binary contains the compiler, formatter, language server, test
runner, linter, documentation generator, package manager, project scaffolder,
task runner, doctor commands, and package tool runner. Tool commands share the
parser, checker, formatter, package resolver, and diagnostic conventions where
applicable.

```text
run       compile and execute a lowercase script entry as a temporary native executable
build     compile a reusable executable or target artifact; accepts --target and -o
emit-c    emit generated C
check     parse, load imports, and validate source without executing or invoking C
format    emit canonical source; -w rewrites files in place
test      discover and run unittest tests; --cover records coverage
cover     render coverage profiles as text, JSON, or HTML
lint      report project-policy diagnostics for valid programs
lsp       run the JSON-RPC language server on stdio
doc       extract source-comment documentation; may emit JSON or static HTML
new       scaffold projects and libraries
task      list or run tya.toml tasks
install   resolve dependencies and write tya.lock
update    refresh locked dependency versions
add       add manifest dependencies
remove    remove manifest dependencies
outdated  report dependencies with newer versions
tool      list or run package-provided tools
doctor    report native or WebAssembly build environment details
embed     inspect embedded asset declarations
version   print the installed Tya version
```

`tya clean` removes `.tya/build/` and keeps `.tya/packages/`. `tya clean
--packages` also removes the project-local dependency cache. Project-local
`.tya/packages/` defines dependency meaning; global caches may optimize fetches
but must not change resolution.

`tya version --json` reports compiler, runtime, SPEC, and self-host version
metadata.

### V1 Public Contract

All user-facing failures use the single stable `TYA-E....` diagnostic-code
namespace across lexer, parser, checker, codegen, runtime, CLI, LSP,
formatter, package manager, stdlib, release, and bootstrap tooling. Runtime
structured error values may additionally carry domain-specific `kind` and
machine-readable `code` fields, but CLI human output, CLI JSON output, LSP
diagnostics, and runtime failure reporting surface the same `TYA-E....`
diagnostic code for the same user-facing failure.

The v1.0.0 stdlib blocker set is part of the public release gate:
`regex/Regex`, filesystem utilities in `file/File` and `dir/Dir`,
`time/Time`, environment and process APIs in `os/Os` and
`process/Process`, and `hmac/Hmac` must be implemented, documented, and
covered before v1.0.0.

Compiler introspection compatibility is intentionally narrow. Stable v1 APIs
are the documented entry points such as `Lexer.lex`, `Parser.parse`,
`Checker.check`, `Format.format`, and explicitly documented AST helper
methods. Full AST dictionary shapes, checker internals, and implementation
helper fields are not v1 compatibility guarantees unless this specification
documents them. Tooling may expose more data, but undocumented fields may
change across v1.x releases.

Platform-dependent stdlib packages are importable on every supported release
platform. Unsupported operations fail only when called, and those failures are
structured errors with stable `TYA-E....` diagnostics. The v1 release gate is
repo-internal: Go tests, testscript fixtures, shell scripts, and release
packaging checks are used for the release, but v1.0.0 does not add or promise
a stable public `tya conformance` command.

Legacy compatibility aliases that remain only for `selfhost/v01` or bootstrap
recovery are legacy compatibility only. Public docs prefer canonical
class-style stdlib APIs and canonical language spellings; legacy aliases are
not v1.x compatibility guarantees.

Only `tya install` and explicit update/add package operations perform
dependency network access by default. `tya check`, `tya run`, `tya build`, and
`tya test` do not fetch missing dependencies automatically; if a required
locked dependency is unavailable locally, they fail with an explicit error.

`tya check` and `tya doc` are safe-analysis commands: they parse, inspect, and
report without executing user code or compiling native code. `tya run`,
`tya build`, and `tya test` execute or build user code and may compile/link
native package code.

`tya lint` diagnostics are warnings, not language validity errors. Suppression
comments are `# tya-lint-ignore: CODE` for one line or next statement and
`# tya-lint-ignore-file: CODE, CODE` for a file. Omitting codes suppresses all
rules for that target. `--fix` may rewrite only rules with declared autofix.
JSON findings include `code`, `title`, `message`, `path`, `line`, `col`,
`autofixable`, and `doc_url`. SARIF and LSP diagnostics use the same stable
rule codes, titles, help URLs, and warning severity. Public rule documentation
lives at `docs/lint.md` and `https://tya-lang.org/lint.html#tyal000N`.

Current lint rules are:

```text
TYAL0001 unused local binding              autofix
TYAL0002 dead code after return or raise
TYAL0003 redundant constant if             autofix
TYAL0004 deeply nested block
TYAL0005 long function body
TYAL0006 suspicious for-index binding order
TYAL0007 unused function parameter
TYAL0008 shadowed binding
```

`tya test` discovers only files whose basename ends in `_test.tya`. Ordinary
`.tya` files are not auto-discovered as test files. Directory test discovery is
deterministic: files run in ascending path order, and tests inside one file run
in definition order. Tests are not run in parallel unless a future explicit
option adds that behavior.

`tya check` reports every recoverable checker diagnostic it can collect and
exits with status 1 when validation fails. If parser recovery is not possible,
`tya check` may stop at that parser error and still exits with status 1.
`tya check -` reads source from standard input and uses `<stdin>` when
reporting diagnostics for stdin source.

`tya run -` reads source from standard input. Relative imports for stdin source
resolve from the current working directory. Program arguments for `tya run` may
follow `--`: `tya run file.tya -- arg1 arg2` makes `args()` return
`["arg1", "arg2"]`. The legacy form `tya run file.tya arg1 arg2` remains
accepted for compatibility. `tya build` does not accept program arguments.

`tya format` never rewrites a file that cannot be lexed, parsed, and
serialized by the canonical formatter. Invalid source reports an error and
exits without modifying the input file. `tya format` formats only `.tya`
source files and never rewrites `tya.toml`.

LSP diagnostics use the same stable diagnostic codes and messages as the
parser, checker, and linter diagnostics used by CLI tools. LSP may transport
diagnostics in LSP shape, but it must not invent alternate wording for the same
source validity issue.

`--json` is accepted as a global alias for JSON diagnostic output. JSON
diagnostics use the existing stable NDJSON schema: diagnostic objects followed
by a summary object. `--format=json` remains accepted for compatibility.
Human-readable diagnostic color follows the existing color controls:
`NO_COLOR=1` and `--no-color` disable color, and
`--color=auto|always|never` remains accepted for compatibility.

Ordinary `emit-c` output is stable for the same source and toolchain version.
It does not include absolute paths, timestamps, random identifiers, or other
nondeterministic metadata unless a future debug option explicitly requests it.

v1.0 has no experimental language feature gates. New syntax is not added in
ordinary v1.x releases; v1.x may add standard-library APIs, package APIs,
tooling improvements, diagnostics, and compatible runtime behavior. New syntax
requires v2 planning or an explicitly accepted experimental feature path
outside the stable v1 public contract. Experimental syntax or `--experimental-*`
tool options are invalid unless a later accepted SPEC adds that behavior
explicitly.

Environment variables are read by user programs only through explicit calls
such as `env(name)` or documented standard-library APIs. `.env` files are not
loaded automatically; users or task-runner commands may load them before
invoking Tya. Toolchain behavior may be affected by the documented variables
`TYA_PATH`, `TYA_STDLIB_DIR`, `TYA_RUNTIME_DIR`, `TYA_PROJECT_ROOT`, `TYA_ZIG`,
`TYA_ZIG_HOME`, `TYA_ZIG_VERSION`, `TYA_ZIG_SHA256`, `TYA_DISABLE_ZLIB`,
`TYA_ENABLE_ZLIB`, `TYA_DISABLE_OPENSSL`, `TYA_ENABLE_OPENSSL`, `CC`,
`NO_COLOR`, and `HOME` in isolated test/toolchain contexts.

`tya doc` extracts leading source comments attached to public top-level
functions, classes, modules, and interfaces. A doc comment attaches only to the
immediately following public item; a blank line breaks attachment and leaves the
comment orphaned. With no path it scans `src/`. It can render terminal text,
generate static HTML, or emit a stable JSON report:

```sh
tya doc
tya doc src
tya doc --json src
tya doc --html ./out src
```

The JSON report contains `version`, `items`, and `diagnostics`. Each item
includes its name, kind, signature, raw Markdown comment, rendered text,
source path and line, and `reexported_from` when the item is included through
an imported package surface. Documentation extraction follows public
re-exports through imports, reports import cycles without hanging, and keeps
ordering deterministic.

Documentation diagnostics use stable `TYADOC` codes. Orphan doc comments,
duplicate public documentation names, unsupported Markdown bodies, and import
cycles are reported to stderr for text and HTML output and embedded in JSON
output. Error diagnostics exit with status 1 after output is produced;
argument, path, I/O, lex, and parse failures exit with status 2.

`tya task` lists and runs project-local shell tasks declared under `[tasks]`
in `tya.toml`. It discovers the project manifest by walking up from the
current directory and runs commands with the project root as the working
directory.

Task entries may be strings, arrays of strings, or table forms:

```toml
[tasks]
run = "tya run src/main.tya"
check = ["tya format --check src", "tya check src/main.tya"]

[tasks.dev]
depends_on = ["check"]
env = { TYA_ENV = "dev" }
cmds = ["tya run src/main.tya"]

[tasks.watch]
cmds = ["tya check src/main.tya", "tya lint src"]
parallel = true
watch = ["src/**/*.tya", "tests/**/*.tya"]
ignore = ["tmp/**", "dist/**"]
```

String tasks run as one `/bin/sh -c` command. Array tasks run sequentially and
stop on the first failing command. Table tasks use `cmds = [...]`; when
`parallel = true`, commands run concurrently and line output is prefixed with
the command index. `depends_on` runs dependency tasks before the selected task,
once per invocation, in the order written. Dependency cycles and unknown
dependencies are reported before any task command runs.

`env` values override the inherited process environment for that task only.
Dependencies use their own environment and do not inherit the selected task's
overrides.

`tya task <name> --watch` runs the task once and reruns it after watched project
files change. By default, watch mode observes `.tya` files, `tya.toml`, and
files under existing `src/`, `tests/`, `stdlib/`, and `examples/` directories,
while ignoring `.git/`, `node_modules/`, `_site/`, common build output
directories, and hidden cache directories. Table-form `watch` overrides the
default watch set and `ignore` adds ignored globs. `--watch` is consumed by the
task runner before `--`; arguments after `--` are passed to the task command.

### Verification Commands

Verification commands inspect source and report whether it satisfies a specific
contract. They do not define language syntax or standard-library behavior.

Verification commands include `tya format --check`, `tya check`, `tya lint`,
`tya test`, and future `tya verify`. `tya run` and `tya build` may share
diagnostics and exit-code conventions with verification commands, but they are
execution and build commands.

`tya format --check` checks whether source files already match canonical Tya
formatting. It answers whether `tya format` would change the file. It must not
rewrite files.

`tya check` checks whether source files are valid Tya programs before C
emission or execution. It includes lexical analysis, parsing, semantic
checking, and import loading needed for the requested program. It excludes C
emission, C compiler invocation, executable creation, program execution, unit
test execution, and lint rules.

`tya lint` checks rules that are not required for language validity. A program
that fails only lint rules is still a valid Tya program. Lint rules may be
built in, configured by a project, or added later by tooling.

`tya test` is the execution entry point for unittest-based tests. It reports
passed tests, failed assertions, skipped tests when supported, runtime errors,
and test discovery errors.

`tya verify` is reserved for the standard verification pipeline. Its order is:

```text
format --check -> check -> lint -> test
```

Initial implementations may run only the commands that exist at the time.
Before `tya verify` exists, CI may run:

```sh
tya format --check .
tya check .
```

Verification commands use stable exit-code meanings:

```text
0  verification passed
1  verification failed
2  command usage error
3  internal tool error
```

Verification commands accept explicit file and directory targets. Directory
targets recursively select `.tya` source files meaningful for the command. With
no target, verification commands use the current directory unless the command
has a stronger existing convention.

Human-readable verification output is concise by default. Failures should
include the command name, file path, line and column when available, a short
rule or diagnostic name when available, and an actionable message. Multi-file
commands should continue after ordinary verification failures where practical
and then report a summary.

`--quiet`, `--verbose`, and `--json` are reserved for consistent verification
behavior. `--json` preserves the same pass/fail meaning and exit codes as
human-readable output.

Verification commands distinguish checking from rewriting. `tya format` may
rewrite files. `tya format --check`, `tya check`, `tya lint`, `tya test`, and
`tya verify` do not rewrite files by default. Automatic lint fixes require an
explicit option such as `--fix`.

### Single Binary Distribution

Tya is distributed as one primary `tya` binary. The binary contains the
toolchain entry points and uses the bundled standard library and C runtime
files that ship with the release.

The one-binary model is part of the language's operational design: users should
not need separate formatter, test runner, LSP server, doc generator, package
manager, or build driver executables for normal Tya work.

Releases may include support files such as the standard library, C runtime
sources, editor assets, examples, or installation metadata, but the command
surface is centered on the single `tya` executable.

## Standard Library

The standard library is shipped with Tya under `stdlib/` and is imported using
the same import syntax as user files and packages.

The standard library is part of the language distribution. Its public surface
is the set of importable PascalCase package classes and interfaces under
`stdlib/`. Standard-library imports are resolved after local packages, locked
package dependencies, and `TYA_PATH` entries.

Public standard-library classes, interfaces, and user-facing methods carry
source doc comments. Generated stdlib API documentation is produced from those
comments with `tya doc`, for example `tya doc --json stdlib` for a
machine-readable reference that includes package paths, signatures, rendered
comments, source paths, and source lines.

Current standard-library surface:

```text
base64/Base64              Base64 encode/decode helpers
binary/Reader              binary input reader
binary/Writer              binary output writer
channel/Channel            native channel value
cli/Cli                    command-line option parser and usage formatter
collections/Deque          double-ended queue
collections/PriorityQueue  priority queue
collections/Queue          FIFO queue
collections/Set            set collection
collections/Stack          LIFO stack
color/Color                RGBA color value and conversions
compiler/ast/Ast           compiler AST helpers
compiler/checker/Checker   compiler checker helpers
compiler/format/Format     compiler formatter helpers
compiler/lexer/Lexer       compiler lexer helpers
compiler/parser/Parser     compiler parser helpers
compress/Compress          compression helpers
csv/Csv                    CSV parse/generate helpers
digest/Digest              digest/hash helpers
dir/Dir                    directory helpers
file/File                  file helpers
geometry/Circle            circle value
geometry/Point             point value
geometry/Rect              rectangle value
geometry/Size              size value
geometry/Vector2           2D vector value
geometry/Vector3           3D vector value
hex/Hex                    hexadecimal encode/decode helpers
hmac/Hmac                  keyed message authentication helpers
image/Codec                image codec helpers
image/Image                image value
io/Io                      stream helpers
io/Reader                  readable stream wrapper
io/Writer                  writable stream wrapper
json/Json                  JSON parse/generate helpers
log/Logger                 logger
markdown/Markdown          Markdown renderer
math/Math                  numeric helpers
regex/Regex                regular expression helpers
matrix/Matrix              matrix value
net/http/Client            HTTP client
net/http/Next              HTTP middleware continuation
net/http/Server            HTTP router/server
net/ip/Address             IP address value
net/ip/Network             IP network value
net/socket/Server          socket listener
net/socket/Socket          socket connection
os/Os                      operating-system helpers
path/Path                  path manipulation helpers
process/Process            process helpers
random/Random              random helpers
random/Rng                 seeded random generator
runtime/Runtime            runtime introspection helpers
secure_random/SecureRandom secure random helpers
serialization/Serializer   data serialization helpers
sync/AtomicInteger         native atomic integer
sync/Mutex                 native mutex
sync/WaitGroup             native wait group
task/Task                  task helpers
template/Template          template renderer
time/Time                  time value and time helpers
toml/Toml                  TOML parse/generate helpers
transform2d/Transform2D    2D affine transform value
unittest/TestCase          test case base class
unittest/TestRunner        test runner
unittest/TestSuite         test suite
url/Url                    URL parse/build helpers
value/Value                value introspection helpers
xml/Xml                    XML parse/generate helpers
xml/Document               XML document node
xml/Element                XML element node
xml/Text                   XML text node
xml/CData                  XML CDATA node
xml/Comment                XML comment node
```

Current standard-library protocol and sequence helper files:

```text
comparable                 Comparable interface
equatable                  Equatable interface
stringable                 Stringable interface
iterator                   Iterator interface; requires has_next() and next()
iterable                   Iterable interface; requires iter()
sequence                   Sequence class and chainable sequence protocol
iterable_sequence          sequence wrapper for Iterable values
map_sequence               lazy map sequence
filter_sequence            lazy filter sequence
take_sequence              lazy take sequence
drop_sequence              lazy drop sequence
```

`Comparable` requires `compare(other)` and provides `lt?`, `lte?`, `gt?`,
`gte?`, and `between?`. `Equatable` requires `equal?(other)`. `Stringable`
requires `to_s()`. `Iterable` requires `iter()` and provides `sequence()`.
`Sequence` provides `iter()`, `map(fn)`, `filter(fn)`, `take(n)`, `drop(n)`,
`reduce(initial, fn)`, and `to_a()`.

### Environment And Process

`os/Os` exposes process environment and working-directory helpers:
`args()`, `env(name)`, `environ()`, `setenv(name, value)`,
`unsetenv(name)`, `cwd()`, `chdir(path)`, and `exit(code)`. Environment names
and values are strings. A missing environment variable returns `nil`, and
environment mutation affects the current process and child processes started
after the mutation.

`process/Process.run(command, options = {})` runs a child process and returns a
result dictionary. `command` may be an array of strings for direct execution or
a string when `options["shell"] == true`. String commands without explicit
shell opt-in are invalid. Supported options are `cwd`, `env`, `clear_env`,
`stdin`, `capture_stdout`, `capture_stderr`, `timeout`, and `shell`. Unknown
option keys are invalid. `env` values override the inherited environment
unless `clear_env` is true. `stdin` may be a string or bytes value.

The result dictionary contains `status`, `success`, `stdout`, `stderr`, and
`timed_out`; `exit_code` remains as a compatibility alias for `status`.
Non-zero child exit status is reported in the result dictionary and is not a
raised error. Spawn/setup failures, invalid options, invalid environment
values, timeout setup failures, and unsupported `Process.exec(command,
options = {})` raise structured process errors.

### Filesystem Utilities

`file/File.copy(src, dst, options = {})` copies file contents as bytes.
Supported options are `overwrite` and `preserve_mode`, both defaulting to
`true`. When `overwrite` is false and `dst` exists, the operation raises a
filesystem error.

`file/File.chmod(path, mode)` changes POSIX-like permissions where the platform
supports them. Windows permissions are best-effort and unsupported permission
changes raise filesystem errors rather than silently promising POSIX behavior.
`file/File.temp(prefix = "tya", suffix = "")` creates an empty temporary file
under the operating-system temporary directory and returns its path.

`dir/Dir.mkdir_all(path)` creates a directory and missing parents.
`dir/Dir.remove_all(path)` removes a file or directory tree recursively;
missing paths are a no-op, while dangerous roots such as `""`, `"."`, `/`, and
platform roots are invalid. `dir/Dir.temp_dir(prefix = "tya")` creates a
temporary directory and returns its path.

`dir/Dir.walk(path, fn, options = {})` visits a directory tree in ascending
path order. `fn` receives an entry dictionary with `path`, `name`, `kind`, and
`stat`. Supported options are `follow_symlinks`, `include_dirs`, and
`include_files`; symlink following is not required where the host platform
cannot safely detect loops.

### HMAC

`hmac/Hmac` provides keyed message authentication without adding a broader
cryptography suite to v1.0.0. `Hmac.digest(algorithm, key, message)` returns
raw bytes. `Hmac.hexdigest(algorithm, key, message)` returns lowercase
hexadecimal text, and `Hmac.base64digest(algorithm, key, message)` returns
Base64 text. Supported algorithms are `sha256`, `sha384`, and `sha512`.

`key` and `message` must be strings or bytes; strings are encoded as UTF-8
bytes. `Hmac.verify(algorithm, key, message, expected, options = {})` compares
the computed digest to `expected` using constant-time comparison for equal
length byte sequences. `expected` may be raw bytes, hex text, or Base64 text
when `options["encoding"]` is `"raw"`, `"hex"`, or `"base64"`. Unsupported
algorithms, malformed encodings, unknown options, and wrong kinds raise
structured crypto errors. General encryption, public-key cryptography,
password hashing, certificates, and streaming HMAC contexts are outside
v1.0.0.

### Regex

`regex/Regex` provides regular expression helpers without adding regex
literals or operators to the language. `Regex.compile(pattern, options = {})`
returns a reusable regex value. `Regex.match?(pattern, text, options = {})`
returns whether the pattern appears anywhere in `text`, and
`Regex.search(pattern, text, options = {})` returns the first match dictionary
or `nil`.

Compiled regex values support `match?(text)`, `find(text)`,
`find_all(text)`, `split(text, limit = nil)`, and
`replace(text, replacement, limit = nil)`. Match dictionaries contain
`text`, `start`, `end`, and `groups`; `start` and `end` are Unicode rune
indexes and `groups` is an array of captured strings or `nil` for unmatched
optional groups. Replacement supports explicit numeric capture references with
the `${1}` spelling; `$$` emits a literal dollar sign, and unknown capture
references are invalid.

Supported options are `ignore_case`, `multi_line`, and `dot_all`, all bools
defaulting to `false`. The v1 portable regex syntax subset is common extended
regular expression syntax: literals, `.`, character classes, grouping,
alternation, `*`, `+`, `?`, bounded repeats, and `^`/`$` anchors. Lookbehind,
backtracking-control verbs, locale-dependent matching, regex literals, and
engine-specific extensions are outside v1.0.0. Invalid patterns, unknown
options, wrong option kinds, wrong argument kinds, and invalid replacement
captures raise structured regex errors.

### Time

`time/Time` provides wall-clock times, monotonic timestamps, durations,
formatting, parsing, arithmetic, and sleeping without adding date/time syntax.
`Time.now()` returns the current wall-clock time. `Time.monotonic()` returns a
monotonic timestamp for elapsed-time measurement; monotonic values may be
subtracted or compared but cannot be formatted as wall-clock dates.

`Time.unix(seconds, nanos = 0)` constructs a UTC wall-clock time. Wall-clock
time values support `unix()`, `unix_nanos()`, `utc()`, `local()`,
`format(layout)`, `add(duration)`, and `sub(other)`. `sub` returns a duration.
`Time.parse(text, layout = "rfc3339")` parses documented layouts. Supported
layout names are `rfc3339`, `date`, `time`, and `unix` for formatting, and
`rfc3339`, `date`, and `unix` for parsing.

`Time.duration(seconds = 0, options = {})` constructs a duration. Supported
duration options are `minutes`, `hours`, `milliseconds`, `microseconds`, and
`nanoseconds`. Duration values support `seconds()`, `milliseconds()`,
`microseconds()`, `nanoseconds()`, `add(other)`, and `sub(other)`.
`Time.sleep(duration_or_seconds)` accepts either a duration value or a number
of seconds.

Timezone support for v1.0.0 is limited to UTC and the process local timezone.
Named timezone database lookup, locale-aware month/day names, date/time
literals, recurrence APIs, and leap-second guarantees beyond host runtime
behavior are outside v1.0.0. Invalid layouts, invalid parse text, wrong
argument kinds, and unknown duration options raise structured time errors.

`io/Reader`, `io/Writer`, and `net/socket` define stream capability
interfaces for readable, writable, closable, and flushable values. `Reader`
supports `read`, `read_line`, `each_line`, `eof?`, and `close`. `Writer`
supports `write`, `write_line`, `flush`, and `close`. `Socket` supports
`connect`, `read`, `read_line`, `write`, `write_line`, `close`, `closed?`,
`local_address`, and `remote_address`; `net/socket/Server` supports `listen`,
`accept`, `close`, and `local_address`. The compiled runtime supports
`net/socket` on POSIX socket platforms and Windows through WinSock2.

`net/http/Server` defines route registration by HTTP method (`get`, `post`,
`put`, `delete`, `patch`, `options`, `head`, `any`), middleware (`use`,
`group`), error and not-found handlers, static-file serving, redirects, route
dispatch, and server execution. `net/http/Client` defines `get`, `post`, and
generic `request`.

`net/http/Client` accepts both `http://` and `https://` URLs. HTTPS uses the
compiled runtime's OpenSSL backend. Certificate verification is enabled by
default; request options may set `ca_file` to a PEM CA bundle or
`insecure_skip_verify: true` to disable verification explicitly. TLS failures
raise `http.tls:` or `http.request:` errors. `net/http/Server.run_tls(port,
cert_file, key_file, options)` serves HTTPS using PEM certificate and private
key files; options may include `host` and `timeout`. Building TLS-enabled
programs requires OpenSSL headers and libraries.

Compiled `net/http/Server` handlers receive request dictionaries with
`cookies`, a dictionary parsed from the incoming `Cookie` header. Missing
cookies produce `{}`. Malformed pairs without `=` are ignored, whitespace
around names and values is trimmed, and repeated names keep the last value.
Handlers also receive `form` and `files` dictionaries. For non-multipart
requests both are empty. For `multipart/form-data` requests, `form` maps field
names to the last string value and `files` maps field names to the last uploaded
file metadata dictionary. File metadata contains `filename`, `content_type`,
`body` as bytes, and `size`. The original raw request body remains available in
`body`. Malformed multipart bodies return `400 Bad Request` before the handler
runs.

`Server.cookie(name, value, options)` formats a `Set-Cookie` header value.
Options may include `path`, `domain`, `max_age`, `expires`, `secure`,
`http_only`, and `same_site` (`Lax`, `Strict`, or `None`). `SameSite=None`
requires `secure: true`. `Server.with_cookie(response, name, value, options)`
appends a cookie to `response["header_values"]["Set-Cookie"]`. Response
dictionaries may use `header_values` for repeated response headers; each array
entry is emitted as a separate header line while ordinary `headers` behavior is
unchanged.

`Server.render(template, data, options)` and
`Server.render_html(template, data, options)` return response dictionaries with
rendered HTML bodies. `options` may be `nil`; the default response status is
`200` and the default `Content-Type` is `text/html; charset=utf-8`. Supported
options are `status`, `headers`, `content_type`, and `template_options`.
String templates that name an existing file are rendered with
`template.Template.render_file`; other strings are rendered as template source.
Embedded bytes are decoded as UTF-8 text before rendering. `render_html` forces
HTML escaping even when `template_options` is present. Extra headers are merged
after defaults, so callers may override `Content-Type`.

Response dictionaries may set `chunked: true` to send an HTTP/1.1 chunked
response. In that mode the runtime writes `Transfer-Encoding: chunked`, omits
`Content-Length`, and writes each string or bytes item from an array body as one
chunk. A channel body may yield string or bytes chunks and closes the stream
when the channel closes. Empty chunks are skipped except for the final
terminating chunk. Non-chunked responses keep the normal `Content-Length`
behavior.

HTTP/1.1 server connections default to keep-alive unless the request contains
`Connection: close`. HTTP/1.0 connections default to close unless the request
contains `Connection: keep-alive`. Request dictionaries expose
`keep_alive` as the boolean decision for that request. Responses include
`Connection: keep-alive` or `Connection: close`, and each accepted connection is
limited to a conservative maximum number of requests.

`serialization/Serializer` converts Tya values to and from data values, JSON,
and TOML. Classes that implement `Serializable` expose `to_data()`.

### External Packages

External packages and tools are distributed outside this repository and are
consumed by git URL plus tag, branch, or revision through `tya.toml`.

Known public packages and tools include:

- `https://github.com/komagata/tya-sqlite`
- `https://github.com/komagata/tya-sdl2`
- `https://github.com/komagata/tya-gtk4`
- `https://github.com/komagata/tya-raylib`
- `https://github.com/komagata/tya-slim`
- `https://github.com/komagata/flakewatch`
- `https://github.com/komagata/magvideo`

## Distribution and System Considerations

Tya programs compile to C and link against the Tya runtime. The runtime
provides value representation, garbage collection, primitive methods, class
dispatch, task and channel support, resources, and native wrapper integration.

The implementation must preserve the self-host fixed-point invariant documented
in `ROADMAP.md`: the maintained Tya-written compiler under `selfhost/v01/`
must continue to compile itself to stable stage-2 and stage-3 output.

The compiler front end is hand-written. Parser generators and large grammar
frameworks are not language authority for the active compiler path.
