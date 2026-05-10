# Tya v0.44 Migration Guide

This guide explains how to migrate Tya code from the v0.43 module
shape to the v0.44 class-oriented namespace model.

The full specification lives in [SPEC.md](SPEC.md). This document is
practical: it walks the common migration cases with before/after
examples.

## Why migrate

v0.44 replaces the `module name + free functions` shape with a
**class-oriented namespace model**. The wins:

- One way to organize a library, not two. Removes the `module`
  vs. `class` choice the language used to leave to authors.
- Directory structure becomes the namespace structure. `import
  net/http` resolves to a directory with class files inside.
- Within a package, sibling class files refer to each other without
  prefix; cross-package access uses `<pkg>.<Class>`.
- Entry script files keep their script feel — no boilerplate added.

## Two file kinds

| Filename starts with | Kind        | Role                          | Importable |
| -------------------- | ----------- | ----------------------------- | ---------- |
| Uppercase letter     | Class file  | Library member                | Yes        |
| Lowercase letter     | Script file | Entry; runs via `tya run`     | No         |

### Class file: `Request.tya`

```tya
class Request
  init = url ->
    @url = url

  get = ->
    # ...
```

The PascalCase filename matches the public class name. Additional
class declarations in the same file are private to that file.

### Script file: `client.tya`

```tya
import net/http

request = http.Request("http://example.com")
print(request.get())
```

Lowercase filename, top-level statements run from the top.

## Migrating a free-function module

### Before (v0.43)

`stdlib/path.tya`:

```tya
module path
  basename = value -> ...
  dirname = value -> ...
  join = parts -> ...
```

Caller:

```tya
import path
print(path.join(["tmp", "x.txt"]))
```

### After (v0.44)

`stdlib/path/Path.tya`:

```tya
class Path
  @@basename = value -> ...
  @@dirname = value -> ...
  @@join = parts -> ...
```

Caller:

```tya
import path
print(path.Path.join(["tmp", "x.txt"]))
```

Mechanical recipe:

1. Move `stdlib/<name>.tya` to `stdlib/<name>/<Name>.tya` (capitalize
   the leaf as the class name).
2. Replace `module <name>` with `class <Name>`.
3. Prefix every member function with `@@`. Leave constants without
   `@@` prefix only if they are class variables; in v0.44 the
   convention is `@@CONSTANT_NAME = ...`.
4. Update all callers from `<name>.X(...)` to `<name>.<Name>.X(...)`.
5. Internal cross-method calls (`path.basename(x)` from inside another
   path member) become `Path.basename(x)`.

## Migrating between class methods within a package

Inside a single package, sibling classes in different files call each
other by **bare** PascalCase name:

`api/v1/Client.tya`:

```tya
class Client
  fetch = url ->
    request = Request(url)        # bare; resolves to v1.Request
    response = Response(200)      # bare; resolves to v1.Response
    "{request.method()} -> {response.status()}"
```

`api/v1/Request.tya`:

```tya
class Request
  init = url ->
    @url = url

  method = ->
    "GET"
```

The bare name resolution is automatic for sibling classes within the
same package directory. From outside the package, you still write
`v1.Request(url)`, `v1.Response(200)`, etc.

## Same-directory siblings for entry scripts

If your project has a script entry plus a few PascalCase class files
in the same directory, the entry sees the classes without an explicit
import:

```text
Greeter.tya
main.tya
```

`Greeter.tya`:

```tya
class Greeter
  init = name ->
    @name = name

  greet = ->
    "Hello, {@name}"
```

`main.tya`:

```tya
greeter = Greeter("komagata")     # no import needed
print(greeter.greet())
```

`tya run main.tya` prints `Hello, komagata`.

## Package directory rules

- The directory name is the package binding (`net/http` binds to
  `http`).
- A package directory may contain only PascalCase `.tya` files
  (class files). A lowercase script file inside a package is rejected
  at import time.
- Packages may nest arbitrarily: `lib/api/v1/auth/` is a valid
  package path; `import api/v1/auth` binds the prefix `auth`.
- The `<pkg>.<Pkg>` redundancy when a single-class package matches
  its directory name (e.g. `solo/Solo.tya` referenced as
  `solo.Solo.greet()`) is intentional.

## Imports

```tya
# Cross-package import (always required)
import net/http

# Path restrictions: forbidden
import /etc/passwd       # absolute paths
import ./local           # leading dot
import ../parent         # leading dotdot
import foo/../bar        # embedded dotdot
import foo//bar          # empty segment
import foo/Bar           # PascalCase terminal segment
```

Resolution order (unchanged from v0.43): the importing file's
directory, each `TYA_PATH` entry, then `stdlib/`.

## Entry execution (`tya run`)

`tya run` accepts only script files (lowercase filename). Running a
class file gives a structured diagnostic:

```text
$ tya run Hello.tya
Hello.tya is a class file; tya run accepts only script files (lowercase filename)
```

Class files are library-only. To make a class file runnable, write a
script file next to it that instantiates and uses it:

```text
Hello.tya          # class Hello with whatever methods
main.tya           # script: h = Hello(); h.run()
$ tya run main.tya
```

## Public and private classes per file

Every `.tya` file has exactly one **public class**:

- A class file's public class is the user-declared class whose name
  matches the filename.
- A script file's public class is the runner-generated unnamed class.

Other classes in the same file are **private** to that file. They
follow the same `PascalCase` naming rule and may use `extends`,
`implements`, `abstract`, `final`, and `override` like public
classes.

```tya
# Server.tya
class Connection                 # private to Server.tya
  init = socket ->
    @socket = socket

class Server                     # public; matches filename
  init = port ->
    @port = port

  accept = socket ->
    Connection(socket)           # bare reference inside the same file
```

> **Limitation note:** v0.44 does not yet enforce cross-file privacy
> at compile time. Another file in the same package can still
> reference `Connection` if it knows the name. Full enforcement is
> tracked under M5 in `ROADMAP.md` and pairs with M8 self-host
> migration.

## Class member conventions (unchanged)

- `@field` — instance field
- `@@method = args -> body` — class method, called as
  `<pkg>.<Class>.method(args)`
- `init = args -> body` — constructor
- `extends`, `implements`, `abstract class`, `final class`,
  `abstract @@method`, `override`, `override @@method` — unchanged
- A class with no `init` and only `@@method` members is a
  Java-style utility class. This is a documented convention; the
  language does not add a keyword for it.

## CLI arguments

CLI arguments are not delivered as a `main` parameter. They are read
through the standard library:

```tya
import os

for arg in os.Os.args()
  print(arg)
```

## What the migration removes

- The `module` keyword. After M9, declaring `module name` in a file
  is a parser error.
- The naming rule "module file: snake_case, must match filename".
  It is replaced by file-kind rules above.
- The "module member can be snake_case function or PascalCase class"
  ambiguity. Class files have one public class; nothing else.

## What stays the same

- Lambda syntax (`name = args -> body`).
- All class member surface from v0.10–v0.13 (`@field`, `@@method`,
  `init`, `extends`, `implements`, `abstract`, `final`, `override`,
  `interface`, `extends interface`).
- Builtin functions are still global (`print`, `len`, `push`, etc.).
- The `import` keyword and most of its surface; only the
  restrictions tighten.

## Migration status (M6)

Stdlib packages migrated to the new shape:

- `path`, `random`, `hex`, `base64`, `digest`, `secure_random`
- `process`, `dir`, `file`, `os`, `math`, `csv`, `matrix`, `json`,
  `toml`, `url`
- `unittest`, `value`, `markdown`

Stdlib packages held back:

- `string`, `array`, `dict` — referenced by the v0.1 self-host
  compiler tests, which only resolve `import X` as single-file
  modules. These migrate alongside M8 self-host migration.
- `runtime`, `time`, `channel`, `sync`, `task` — held back while
  their callers in `examples/` and `tests/testdata/v4{1,2,3}/` are
  cleaned up in a separate working-tree pass.

See `ROADMAP.md` for the full milestone breakdown.
