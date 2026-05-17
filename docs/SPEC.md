---
layout: doc
title: Spec
permalink: /spec/
---

# Tya Language Specification

Status: current repository specification. This page describes the language
surface maintained on `main`, including the current package, tooling,
concurrency, interface, and standard-library integration rules.

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

This document specifies the language, built-in function surface,
standard-library surface, package rules, and tool surface.

The strict-semantics rule matrix in
[`docs/STRICT_SEMANTICS.md`](STRICT_SEMANTICS.md) is normative for v1.0.0
validity boundaries and records the active parser, checker, runtime, CLI, LSP,
and self-host coverage for each rule family.

## Notation

Examples use ordinary Tya source. Grammar fragments are illustrative rather
than a complete parser grammar.

```text
snake_case            variable, function, method, import path segment
SCREAMING_SNAKE_CASE  constant
PascalCase            class and interface
```

The words "must", "must not", "may", and "should" are normative when they
describe program validity or implementation behavior.

## Naming

Tya names express naming category, not accessibility. Accessibility is
expressed by language constructs such as `private`.

Value names, function names, method names, file names, import path segments,
and dictionary keys use `snake_case`. Constants use `SCREAMING_SNAKE_CASE`.
Classes and interfaces use `PascalCase`.

Single-file imports use the source filename without `.tya` as the import path
segment. Import paths are slash-separated `snake_case` segments. Leading `_`
has no visibility meaning for ordinary bindings. Standard-library APIs use
`snake_case`; CamelCase builtin spellings are not part of the language surface.

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
Floating-point literals use decimal notation.

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

## File Kinds

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

## Canonical Syntax {#canonical-syntax}

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
- imports are atomic and not line-wrapped;
- `elseif` is the canonical spelling, and `else if` is not canonical;
- `case _` in `match` is the wildcard case and must be final;
- empty collection forms and empty `else` branches follow formatter-defined
  shapes.

Implementations must preserve semantic behavior when formatting. Formatting
must be idempotent and stable across platforms.

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

Leading `_` has no visibility meaning for ordinary bindings. Top-level
privacy is not expressed by name spelling.

Constants use `SCREAMING_SNAKE_CASE` and are checked as constants by naming
and assignment rules.

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
Numbers and strings conform to `Comparable` as primitive values. The ordering
operators `<`, `<=`, `>`, and `>=` keep their existing primitive behavior and
do not dispatch to user-defined `compare`.

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

Function bodies cannot write back to outer bindings. Direct reassignment of an
outer binding is invalid, and indexed or member assignment through a captured
outer binding is invalid. Pass mutable state as an explicit parameter when a
function is intended to mutate that value.

Each evaluation of a function literal creates an independent closure
environment.

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
operator case says otherwise. `+` formats through string conversion when either
operand is a string, and concatenates two bytes values. String interpolation
formats embedded values with the `Stringable` surface. `nil` arithmetic is
invalid.

Bitwise operators require integer-compatible number values.

Equality operators may compare any two runtime values without coercion.
Ordering operators `<`, `<=`, `>`, and `>=` require numbers.

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

Array, string, and bytes indexes must be integers. Dictionary and error-value
indexes must be strings. Missing dictionary keys and out-of-range array or
string indexes return `nil`; indexing a non-collection target is invalid.

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
the methods specified in the Standard Library section.

## Parallelism And Concurrency

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

`break` exits the nearest loop. `continue` skips to the next iteration.

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

The exact channel methods and sync primitives are defined in the Standard
Library section.

## Built-In Functions

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

## Terminology

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
package-provided tools. `tya install` resolves dependencies and writes
`tya.lock`. Git and local path dependencies are supported. There is currently
no central package registry and no `tya publish` command.

A package is a versioned distribution unit for reusable Tya code. Package code
normally exposes importable source under `src/`. Applications consume packages
through manifest dependencies:

```toml
[dependencies]
my_lib = { git = "https://github.com/example/my_lib", tag = "v0.1.0" }
local_lib = { path = "../local_lib" }
```

`tya.lock` records resolved dependency sources and should be committed by
applications.

Native package metadata lives under `[native]`. Native paths are relative to
the package root. `tya build`, `tya run`, and `tya test` compile declared C
sources with generated C, the Tya runtime, include directories, `pkg-config`
flags, `cflags`, and `ldflags`. Native wrapper functions use the Tya runtime
ABI and are called from package code like predeclared functions inside that
package.

Package-provided tools live under `[tools]` and run through `tya tool`.
Package tools are not global installs and are not shell tasks; they run from
locked dependencies or an explicit one-shot git/path source.

## Program Execution

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
`tya build` writes a reusable executable. `tya emit-c` prints or writes the C
program generated from Tya source. The generated C links against the Tya
runtime.

The default native target uses the Tya-managed Zig toolchain as `zig cc`.
Native package metadata from `[native]` contributes C sources, headers,
include directories, `pkg-config` flags, compiler flags, and linker flags to
the build.

WASM build targets are available where supported. Native packages are rejected
for unsupported WASM targets. `tya run` remains native-only.

## Cross Compilation {#cross-compilation}

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
WebAssembly artifacts.

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

`tya doc` extracts leading source comments attached to public top-level
functions, classes, modules, and interfaces. With no path it scans `src/`.
It can render terminal text, generate static HTML, or emit a stable JSON
report:

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

## Verification Commands

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

## Single Binary Distribution

Tya is distributed as one primary `tya` binary. The binary contains the
toolchain entry points and uses the bundled standard library and C runtime
files that ship with the release.

The one-binary model is part of the language's operational design: users should
not need separate formatter, test runner, LSP server, doc generator, package
manager, or build driver executables for normal Tya work.

Releases may include support files such as the standard library, C runtime
sources, editor assets, examples, or installation metadata, but the command
surface is centered on the single `tya` executable.

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
image/Codec                image codec helpers
image/Image                image value
io/Io                      stream helpers
io/Reader                  readable stream wrapper
io/Writer                  writable stream wrapper
json/Json                  JSON parse/generate helpers
log/Logger                 logger
markdown/Markdown          Markdown renderer
math/Math                  numeric helpers
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
