# Tya v0.44 Specification

This document is the specification for Tya v0.44 after v0.43 concurrency
known-gap close-out.

## Theme

Tya v0.44 is about a **class-oriented namespace and entry-file model**.

Through v0.43 Tya carried two coexisting library shapes: snake_case
`module name` files containing free functions, and PascalCase `class`
declarations placed inside modules. v0.44 picks one shape — class — and
reorganizes the surrounding namespace machinery so that every reusable unit
is a class, every file's role is decidable from its name alone, and every
program has exactly one source representation in the spirit of Canonical
Syntax.

## Goals

- Replace the snake_case `module name` namespace with directory-as-package.
- Make every importable file a class file: PascalCase filename, single
  top-level `class X` whose name matches the filename.
- Make every entry file a script file: lowercase filename, top-level
  statements only, internally desugared to an unnamed class with `main`.
- Make `tya run` accept only script files (compact form). Class files cannot
  be entries.
- Allow arbitrary-depth namespace hierarchy through directory nesting.
- Forbid `..` and absolute paths in `import`. Resolve only against the entry
  directory, `TYA_PATH`, and `stdlib`.
- Allow same-directory siblings to refer to each other without prefix.
- Allow private classes (only visible within the same file).
- Remove the `module` keyword.
- Keep current class machinery (`@field`, `@@method`, `init`, `extends`,
  `implements`, `abstract`, `final`, `override`, `interface`) unchanged.

## Non-Goals

v0.44 does not include:

- primitive-as-class sugar (`1` desugared to `Integer(1)`); deferred to a
  later Epic.
- module mixins / `include` (Ruby-style).
- a `static class` keyword. Java-style utility classes are written as
  ordinary classes with no `init` and only `@@method` members; this is a
  convention, not a language feature.
- a separate `module` (Ruby-style non-instantiable class) construct.
- changes to lambda or class member syntax.
- changes to the formatter except as needed for the new file kinds.
- changes to the C runtime or GC.
- automatic migration of third-party Tya code; the migration tool, if any,
  ships in a later minor.

## File Kinds

A `.tya` file's role is determined by the first character of its filename:

| Filename starts with | Kind          | Role                           |
| -------------------- | ------------- | ------------------------------ |
| Uppercase letter     | Class file    | Library; importable            |
| Lowercase letter     | Script file   | Entry; not importable          |

Filenames starting with `_`, digits, or other characters are language errors.

### Class file

A class file is a PascalCase `.tya` file containing **exactly one top-level
public class** whose name matches the filename without `.tya`.

```text
Request.tya          ->  class Request
HttpClient.tya       ->  class HttpClient
```

A class file may additionally declare any number of **private classes**
alongside the public one. A class is the file's public class only if its
name matches the filename without `.tya`. Every other class in the file is
private and visible only within that file. Private classes follow the
same `PascalCase` naming rule.

```tya
# Request.tya
class Header
  init = name, value ->
    @name = name
    @value = value

class Request
  init = url ->
    @url = url
    @headers = []
```

Here `Request` matches the filename and is the file's public class.
`Header` does not match the filename and is private; another file may
declare its own `Header` without conflict.

A class file must not contain top-level statements other than `import`,
class declarations, and interface declarations.

### Script file

A script file is a lowercase `.tya` file. Its body is **top-level statements
and bindings**, optionally preceded by `import` lines.

```tya
# hello.tya
print("hello")
```

```tya
# client.tya
import net/http

request = http.Request()
print(request.get("http://example.com"))
```

A script file may declare private classes; they are visible only within the
file. A script file must not declare a public class with a name matching the
filename, because a script file's public class is generated implicitly (see
*Entry execution*).

A script file is not importable. `import` paths never resolve to a script
file.

## Namespace and Packages

A **package** is a directory. The directory name is the package's
identifier. Package directory names must match the
variables-and-functions naming rule (`snake_case`). All `.tya` files in a
directory share that package's namespace.

Within a single package directory, sibling class files refer to each
other's public classes without any prefix.

```tya
# net/http/Response.tya
class Response
  ...

# net/http/Request.tya
class Request
  init = url ->
    @url = url

  send = ->
    Response()                # sibling, no prefix
```

Packages may nest arbitrarily deep through directory nesting:

```text
stdlib/
  net/
    http/
      Request.tya
      Response.tya
    tcp/
      Socket.tya
  os/
    Os.tya
```

## Import

`import path/to/package` loads the package directory at `path/to/package`
and binds the **last path segment** as the prefix used at call sites.

```tya
import net/http

req = http.Request()
res = req.send()
```

### Resolution

`import path` resolves `path` against the following roots, in order:

1. The directory containing the entry script file.
2. Each entry of `TYA_PATH`, in order.
3. The bundled `stdlib/` directory.

The first directory found that matches `path` exactly is the resolved
package.

### Restrictions

- The path must use forward slashes (`/`) as separators.
- `..` segments are forbidden.
- Absolute paths (paths beginning with `/`) are forbidden.
- The path must end on a directory, not a file. Importing a class file
  directly (`import net/http/Request`) is forbidden.
- The terminal segment must be a valid `snake_case` identifier (i.e., it is a
  package directory, not a class file).
- Importing a script file is impossible because script-file directories
  cannot be the terminal segment.

### Same-Package Reference

A `.tya` file does not need to `import` its own package. All public classes
in the same directory are in scope without prefix.

### Cross-Package Reference

Public classes in another package are only reachable through `import` and
the package prefix. The full reference form is
`<last-segment>.<ClassName>`, even when the package contains a single
class whose name matches the directory:

```text
stdlib/math/Math.tya       contains class Math

import math
Math.sin(0.5)              # ERROR: Math is not in scope
math.Math.sin(0.5)         # OK
```

The `math.Math` repetition is intentional and consistent with Java's
`java.lang.Math.sin()` style.

## Entry Execution

`tya run path/to/file.tya` accepts **only script files** (lowercase
filename). Running a class file is a runner error.

The runner desugars the script file into a class file with an unnamed class
that has a `main` method whose body is the script's top-level statements:

```tya
# hello.tya  (what the user writes)
import os

print("hello")
for arg in os.Os.args()
  print(arg)
```

is internally equivalent to:

```text
# implicit form  (not user-writable)
import os

class _Anonymous
  main = ->
    print("hello")
    for arg in os.Os.args()
      print(arg)
```

The implicit class is unnamed; it has no source-level identifier and
cannot be referred to from any file. Bindings introduced at script
top-level (e.g. `request = http.Request()`) become locals of the
implicit `main`.

This desugaring is internal. The strict form is **not user-writable**:
PascalCase class files with a `main` method exist as ordinary library
classes and are not entry points.

## Class Files Are Library-Only

A class file's `main` method, if defined, has no special meaning. It is an
ordinary instance method. `tya run Hello.tya` is a runner error regardless
of whether `Hello.tya` defines `main`.

Rationale: every Tya program has exactly one source representation per
Canonical Syntax. Allowing both `tya run Hello.tya` (with `class Hello +
main`) and `tya run hello.tya` (compact) to launch a program would create
two ways to write the same thing.

## Public and Private Classes

Every `.tya` file contains exactly one **public class**:

- In a class file the public class is the user-declared class whose name
  matches the filename.
- In a script file the public class is the runner-generated unnamed class.

Any additional class declared in the same file is **private**: it is
visible only to code inside that file. Private classes have no externally
visible name and cannot be referenced from another file under any
mechanism, including reflection.

Private classes follow the same naming rules as public classes
(PascalCase). They may use `extends`, `implements`, `abstract`, `final`,
and `override` exactly like public classes.

## Class Member Conventions (Unchanged)

The class member surface from v0.10–v0.13 is preserved without change:

- `@field` — instance field.
- `@@method = args -> body` — class method, called as `Class.method(args)`.
- `init = args -> ...` — constructor.
- `extends`, `implements`, `abstract class`, `final class`,
  `abstract @@method`, `override`, `override @@method` — unchanged.

A class with no `init` and only `@@method` members is a Java-style
utility class. This is a documented convention; the language does not add
a keyword for it.

## CLI Arguments

CLI arguments are not delivered as a `main` parameter. They are read
through the standard library:

```tya
import os

for arg in os.Os.args()
  print(arg)
```

`os.Os.args()` returns an array of strings (empty when no arguments).
`main`'s signature is fixed at no parameters.

## Removed Constructs

- The `module` keyword is removed. Files may not declare `module name`.
- The naming rule "module file: snake_case, must match filename" no longer
  applies; that rule is replaced by file-kind rules above.
- Module-private `_name` members no longer exist as a category. Privacy is
  expressed through class private members (existing `_name` rule on class
  members is unchanged) and the file-private nature of private classes.

## Naming Rules (v0.44)

```text
script file (entry):  lowercase ASCII (e.g. hello.tya, client.tya)
class file:           PascalCase (e.g. Request.tya), filename = class name
package directory:    snake_case (e.g. net, http, file_system)
variables/functions:  snake_case
private binding:      _snake_case
classes:              PascalCase
class methods:        @@snake_case
instance fields:      @snake_case
dictionary keys:      snake_case
constants:            SCREAMING_SNAKE_CASE
```

## Errors

The new rules introduce the following diagnostics. Code blocks are
reserved per stage; individual codes are finalized when each STEP lands.

Reserved ranges:

| Range           | Stage    | Purpose                                  |
| --------------- | -------- | ---------------------------------------- |
| `E0200`–`E0219` | parser   | `module` keyword removed (M9)            |
| `E0400`–`E0429` | checker  | class file structure (M2, M5)            |
| `E0850`–`E0879` | runner   | import resolution and entry kind (M3, M4) |

Diagnostics in scope:

| Code (TBD)  | Stage   | Condition                                                                      |
| ----------- | ------- | ------------------------------------------------------------------------------ |
| `E0400`     | checker | Class file does not contain a class declaration.                               |
| `E0401`     | checker | Class file's public class name does not match the filename.                    |
| `E0402`     | checker | Class file contains a non-import / non-class / non-interface top-level statement. |
| `E0403`     | checker | Script file declares a public class with a filename-matching name.             |
| `E0404`     | checker | Filename starts with an unsupported character (not ASCII letter).              |
| `E0405`     | checker | Same-directory class name collision.                                           |
| `E0406`     | checker | Cross-file reference to a private class.                                       |
| `E0850`     | runner  | `import` path contains `..` or starts with `/`.                                |
| `E0851`     | runner  | `import` path's terminal segment is not a valid package directory name.        |
| `E0852`     | runner  | `import` path resolves to a script file (lowercase leaf).                      |
| `E0853`     | runner  | `import` path cannot be resolved against the configured roots.                 |
| `E0854`     | runner  | `tya run` invoked on a class file (only script files are runnable).            |
| `E0200`     | parser  | `module` keyword used (removed in M9).                                         |

Codes in `E04xx` and `E08xx` are additive within the checker and runner
ranges already reserved by `docs/v0.29/CODES.md`. The parser block
`E0200`–`E0219` is reserved out of the parser range `E0100`–`E0299`
allocated for the Toolchain "Migrate remaining stages to the
diagnostics pipeline" Epic; the `module`-removal code lands when the
parser has been migrated.

## Examples

### Single-file script

```text
hello.tya
```

```tya
# hello.tya
print("hello")
```

```sh
$ tya run hello.tya
hello
```

### Two-file program

```text
Greeter.tya
main.tya
```

```tya
# Greeter.tya
class Greeter
  init = name ->
    @name = name

  greet = ->
    "Hello, {@name}"
```

```tya
# main.tya
greeter = Greeter()
print(greeter.greet("komagata"))
```

```sh
$ tya run main.tya
Hello, komagata
```

### Imported package with multiple classes

```text
lib/
  net/
    http/
      Request.tya
      Response.tya
client.tya
```

```tya
# lib/net/http/Request.tya
class Request
  init = url ->
    @url = url

  send = ->
    Response()        # same package, no prefix
```

```tya
# lib/net/http/Response.tya
class Response
  status = ->
    200
```

```tya
# client.tya
import net/http

request = http.Request("http://example.com")
response = request.send()
print(response.status())
```

### Utility class

```text
stdlib/
  math/
    Math.tya
```

```tya
# stdlib/math/Math.tya
class Math
  pi = 3.14159265358979

  @@sin = x ->
    ...

  @@cos = x ->
    ...
```

```tya
# user code
import math

print(math.Math.sin(0.5))
print(math.Math.pi)
```

### Private class inside a class file

```tya
# Server.tya
class Connection
  init = socket ->
    @socket = socket

  close = ->
    ...

class Server
  init = port ->
    @port = port
    @connections = []

  accept = ->
    conn = Connection(socket)    # Connection is file-private
    push(@connections, conn)
```

`Server` matches the filename and is public. `Connection` does not
match and is private; nothing outside `Server.tya` can reference it.

## Migration Sketch (informative)

The implementation order, captured for cross-reference with `ROADMAP.md`:

1. Parser/checker accepts the new model alongside the existing `module`
   keyword. Both shapes coexist temporarily.
2. Resolver gains directory-as-package support. Existing module imports
   keep working.
3. Compact entry-file desugaring becomes the runner's default for
   lowercase files; existing top-level execution semantics are preserved.
4. Private-class semantics land.
5. `stdlib/` is migrated package by package from `module + functions`
   to class-file form. Each package landing keeps tests green.
6. `examples/` is migrated.
7. `selfhost/v01/compiler.tya` is migrated, preserving the self-host
   fixed point at every STEP.
8. The `module` keyword is removed; remaining `module` files are deleted
   or moved.
9. `docs/SPEC.md`, `docs/NAMING.md`, `docs/STDLIB.md`, and
   `docs/CANONICAL_SYNTAX.md` are updated to reflect v0.44.

The detailed STEP breakdown lives in `ROADMAP.md`.
