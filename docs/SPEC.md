# Tya Language Specification

Status: current repository specification. Historical release snapshots live
under `docs/vX.Y/`. This page describes the language surface maintained on
`main`, including the current package, tooling, concurrency, interface, and
standard-library integration rules.

## Introduction

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

This document specifies the language. Built-in functions are listed in
`docs/API.md`. Standard-library modules are listed in `docs/STDLIB.md`.
Reusable user libraries and packages are described in `docs/LIBRARIES.md`.
Canonical formatting details are described in `docs/CANONICAL_SYNTAX.md`.

## Notation

Examples use ordinary Tya source. Grammar fragments are illustrative rather
than a complete parser grammar. Names in examples follow `docs/NAMING.md`.

```text
snake_case            variable, function, method, import path segment
_snake_case           private source binding
SCREAMING_SNAKE_CASE  constant
PascalCase            class and interface
```

The words "must", "must not", "may", and "should" are normative when they
describe program validity or implementation behavior.

## Source Code Representation

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

## Lexical Elements

### Comments

Line comments begin with `#` and continue to the end of the line.

```tya
# file header comment
name = "tya" # line-end comment
```

Comments may attach to declarations and statements for formatting, LSP hover,
and `tya doc`. Comment placement rules are part of canonical syntax.

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
extends false final for if implements import in interface module nil not of or
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
receive a value of the required kind or raise a runtime error.

## Blocks

A block is a non-empty or empty sequence of statements introduced by a header
line and an increased indentation level.

```tya
while count < 3
  print(count)
  count = count + 1
```

Blocks appear in control-flow statements, function bodies, class bodies,
interface bodies, `try` / `catch`, `scope`, `select`, and similar constructs.

Top-level source consists of imports, declarations, assignments, and statements
allowed by the file kind. Class files are more restrictive than script files.

## Declarations And Scope

### Bindings

Assignment creates or updates bindings.

```tya
name = "Tya"
count = count + 1
```

Multiple assignment is supported.

```tya
value, err = parse_user(text)
```

Names beginning with `_` are private when they are top-level bindings in an
importable source file. Private top-level bindings are not exported through a
single-file module namespace.

Constants use `SCREAMING_SNAKE_CASE` and are checked as constants by naming
and assignment rules.

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

The final expression in a function body is returned implicitly. Use `return`
for early return or multiple return values.

```tya
parse_user = text ->
  if text == ""
    return nil, error("empty user")
  return { name: text }, nil
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
- runtime class inspection through `.class`.

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
operator case says otherwise. `nil` arithmetic is invalid.

### Collections

Arrays use bracket literals and integer indexing.

```tya
items = ["a", "b"]
items.push("c")
print(items[0])
```

Dictionaries use brace literals. Identifier keys in dictionary literals are
stored as string keys.

```tya
user = { name: "komagata", age: 20 }
print(user["name"])
user["admin"] = true
```

Dictionary block forms and empty collection forms are canonicalized by the
formatter.

### Error Expressions

`try` may be used as an expression inside a function body. A `catch` branch
receives the raised value.

```tya
load_name = path ->
  try
    read_file(path).trim()
  catch err
    "guest"
```

### Concurrency Expressions

`spawn` starts a task and returns a task value. `await` waits for a task and
returns or re-raises its result.

```tya
task = spawn work(21)
print(await task)
```

Channels and sync resources are standard-library-backed runtime values with
documented methods in `docs/STDLIB.md`.

## Statements

### Expression Statements

Calls and other useful expressions may appear as statements.

```tya
print("hello")
logger.info("started")
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
left-hand names.

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

### While Statements

`while` repeats while its condition is truthy.

```tya
while count < 3
  print(count)
  count = count + 1
```

### For Statements

`for ... in` iterates arrays and other iterable values. For arrays, the second
binding receives the index when present.

```tya
for item in items
  print(item)

for item, index in items
  print("{index}: {item}")
```

`for ... of` iterates dictionary keys and values.

```tya
for key, value of user
  print("{key}: {value}")
```

`break` exits the nearest loop. `continue` skips to the next iteration.

### Match Statements

`match` selects one `case` block by comparing an expression to case patterns.
`case _` is the wildcard case and is canonical only as the final case.

### Return Statements

`return` exits the current function or method. It may return zero, one, or
multiple values.

```tya
return
return value
return value, err
```

### Raise, Try, And Catch Statements

`raise` raises a value. `try` executes a block and handles raised values with
`catch`.

```tya
try
  save_user(user)
catch err
  print("save failed: {err}")
```

### Scope Statements

`scope` defines a structured concurrency region. Tasks spawned inside the
scope are joined according to the runtime scope rules before the scope exits.

```tya
scope
  task = spawn work()
  print(await task)
```

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

The exact channel methods and sync primitives are defined in `docs/STDLIB.md`.

## Built-In Functions

Tya has predeclared builtins for core runtime operations, I/O, conversion,
errors, process access, files, collections, and compiler introspection.
The normative list is `docs/API.md`.

Common examples:

```tya
print("hello")
args()
type(value)
error("message")
read_file("memo.txt")
write_file("memo.txt", "text")
```

Standard library APIs are imported as ordinary modules.

## Imports And Packages

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

### Single-File Modules

A single-file module is a lowercase `.tya` file resolved by import path.

```text
import greeting          -> greeting.tya
import http/server       -> http/server.tya
```

It exports public top-level bindings through the import binding.

```tya
import greeting

print(greeting.hello("komagata"))
```

### Directory Packages

A directory package is a directory resolved by import path containing
PascalCase class files. It must contain at least one class file and must not
contain lowercase script files at the package leaf.

Unaliased directory imports expose public class and interface names directly.

```tya
import net/http

server = Server()
```

Aliased directory imports expose a namespace binding and do not import public
names bare.

```tya
import net/http as http

server = http.Server()
```

Within the same directory package, sibling public classes are visible by bare
PascalCase name without import.

### Resolution Order

Imports are resolved in this order:

1. the importing file's directory;
2. manifest-declared dependencies from `tya.lock`;
3. directories listed in `TYA_PATH`, from left to right;
4. the bundled `stdlib/` directory.

The first matching file or package directory wins. Local application modules
may shadow package, `TYA_PATH`, and standard-library modules. Package
dependencies may shadow `TYA_PATH` and standard-library modules.

### Package Manifests

`tya.toml` declares package metadata, dependencies, native wrappers, and
package-provided tools. `tya install` resolves dependencies and writes
`tya.lock`. Git and local path dependencies are supported. There is currently
no central package registry and no `tya publish` command.

Native package metadata lives under `[native]`. Package-provided tools live
under `[tools]` and run through `tya tool`.

## Program Execution

A script file is a lowercase `.tya` file and may be used as an entry file for
`tya run`, `tya build`, and `tya emit-c`.

```sh
tya run hello.tya
tya build hello.tya -o hello
tya emit-c hello.tya
```

Class files are library-only and cannot be entry files.

`tya check` validates source without running it. `tya format` rewrites source
to canonical syntax. `tya test` discovers and runs tests using the standard
`unittest` surface. `tya lint` reports style and safety diagnostics.
`tya doc` extracts source documentation and may generate static HTML.

WASM build targets are available where supported. Native packages are rejected
for unsupported WASM targets.

## Errors And Diagnostics

Tya has two related error systems:

- language-level `raise`, `try`, and `catch` for program errors;
- compiler and tool diagnostics for invalid source and tooling failures.

Compiler diagnostics use stable codes such as `TYA-E0015` and linter
diagnostics use stable codes such as `TYAL0001`. Diagnostics should include
an actionable message and, where practical, a hint and documentation URL.

Runtime kind errors, invalid operations, failed assertions, failed I/O, and
native wrapper errors are represented as Tya error values or raised runtime
errors according to the API being used.

## Standard Library

The standard library is shipped with Tya under `stdlib/` and is imported using
the same import syntax as user modules and packages.

Examples include:

```text
math
path
file
json
toml
csv
url
time
random
unittest
template
markdown
compress
log
io
net/ip
net/socket
net/http
channel
sync
task
```

The normative standard-library API reference is `docs/STDLIB.md`.

## External Packages

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

## System Considerations

Tya programs compile to C and link against the Tya runtime. The runtime
provides value representation, garbage collection, primitive methods, class
dispatch, task and channel support, resources, and native wrapper integration.

The implementation must preserve the self-host fixed-point invariant documented
in `ROADMAP.md`: the maintained Tya-written compiler under `selfhost/v01/`
must continue to compile itself to stable stage-2 and stage-3 output.

The compiler front end is hand-written. Parser generators and large grammar
frameworks are not language authority for the active compiler path.
