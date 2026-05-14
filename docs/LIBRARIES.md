# Tya User Libraries and Packages

This document defines reusable Tya code outside the Tya standard library.

The standard library is documented separately in `docs/STDLIB.md`. Shared terms
are defined in `docs/TERMINOLOGY.md`.

## Scope

This document covers:

- user modules
- user libraries
- package directory layout
- import path rules for libraries and packages
- how applications import library and package code
- the boundary between `TYA_PATH` libraries and manifest-managed packages

This document does not define:

- standard library modules
- built-in functions
- package registries
- workspaces
- publishing workflows

## User Module

User module: importable Tya source outside the standard library. It is loaded
with `import` and exposed through the import binding.

Single-file modules export public top-level bindings:

```tya
# greeting.tya
hello = name -> "Hello, {name}"
```

```tya
# main.tya
import greeting

print greeting.hello("komagata")
```

A single-file source may contain top-level imports, assignments, functions,
classes, and embeds. Public top-level names become members of the imported
namespace. Names starting with `_` are not exported.

Directory packages export public class files:

```text
pkg/
  User.tya
  Helper.tya
```

```tya
# pkg/User.tya
class User
  initialize = name ->
    self.name = name
```

```tya
import pkg

user = pkg.User("komagata")
```

A directory package may contain PascalCase `.tya` class files. Lowercase script
files are rejected inside package directories.

## User Library

A user library is a directory tree of modules intended to be reused by more than
one entry file or project.

A user library does not require a manifest. The direct way to make one available
is to put its library root on `TYA_PATH`.

Example:

```text
libs/web/
  http/
    server.tya
    request.tya
```

Use it with:

```sh
TYA_PATH=libs/web tya run app.tya
```

```tya
import http/server
import http/request

server.listen(8080)
request.parse(raw_request)
```

## Package

A package is a versioned distribution unit for reusable Tya code. A package has a
`tya.toml` manifest and normally exposes importable code under `src/`.

Minimal package layout:

```text
my_lib/
  tya.toml
  src/
    my_lib/
      MyLib.tya
```

```toml
name = "my_lib"
version = "0.1.0"
license = "MIT"
```

```tya
# src/my_lib/MyLib.tya
class MyLib
  initialize = ->
    self.name = "my_lib"
```

Applications import the package directory and use its public classes:

```tya
import my_lib

lib = MyLib()
```

A package can be used through a local path dependency:

```toml
[dependencies]
my_lib = { path = "../my_lib" }
```

or through a git dependency:

```toml
[dependencies]
my_lib = { git = "https://github.com/example/my_lib", tag = "v0.1.0" }
```

After editing dependencies, run:

```sh
tya install
```

`tya install` resolves dependencies, writes `tya.lock`, and materializes git
packages under `.tya/packages/`. Path dependencies are read from their source
path. `tya.lock` should be committed by applications.

There is currently no central package registry or `tya publish` command. Public
packages are distributed by git URL plus tag, branch, or revision.

The first public external packages and tools are maintained in separate
repositories:

- `https://github.com/komagata/tya-sqlite`
- `https://github.com/komagata/tya-sdl2`
- `https://github.com/komagata/tya-gtk4`
- `https://github.com/komagata/tya-raylib`
- `https://github.com/komagata/tya-slim`
- `https://github.com/komagata/flakewatch`
- `https://github.com/komagata/magvideo`

### Native Package Support

A package may declare native C wrappers in `tya.toml` when plain Tya code needs
to call a host library or a small C shim:

```toml
[native]
sources = ["native/my_binding.c"]
headers = ["include/my_binding.h"]
include_dirs = ["include"]
pkg_config = []
cflags = []
ldflags = []

[native.functions]
my_binding_version = { symbol = "tya_my_binding_version", arity = 0 }
```

Native paths are relative to the package root. `tya build`, `tya run`, and
`tya test` compile declared C sources with the generated C program, the Tya
runtime, include directories, `pkg-config` flags, `cflags`, and `ldflags`.

Native wrapper functions use the Tya runtime ABI:

```c
TyaValue tya_my_binding_version(TyaValue __this, TyaValue a0, TyaValue a1,
                                TyaValue a2, TyaValue a3);
```

Tya package code calls the declared native function like any other predeclared
function inside that package:

```tya
class MyBinding
  static version = ->
    return my_binding_version()
```

Use `tya doctor native` from a project root to inspect the detected C compiler,
`pkg-config`, native packages, sources, include directories, and effective
flags. `tya new --template lib --native my_binding` creates a minimal native
library scaffold.

## Import Path

An import path is the slash-separated name used after `import`.

Examples:

```tya
import greeting
import http/server
import data/json/parser
```

Each path segment must follow the import naming rule:

```text
^[a-z][a-z0-9_]*$
```

Module paths must not be relative filesystem paths.

Invalid:

```tya
import ./greeting
import ../shared/greeting
import /system/greeting
import http//server
```

## File Rule

For single-file modules, the file path matches the import path:

```text
import greeting          -> greeting.tya
import http/server       -> http/server.tya
import data/json/parser  -> data/json/parser.tya
```

For directory packages, the import path can resolve to a directory containing
PascalCase class files:

```text
import lib -> lib/User.tya, lib/Client.tya, ...
```

The visible namespace is the import binding. Use an alias when the final segment
is too generic or conflicts with another import:

```tya
import data/json/parser as json_parser
import html/parser as html_parser

json_parser.parse("{}")
html_parser.parse("<p>hi</p>")
```

## Import Resolution

Imports are resolved in this order:

1. The importing file's directory.
2. Manifest-declared dependencies from `tya.lock`.
3. Each directory listed in `TYA_PATH`, from left to right.
4. The bundled standard library directory.

The first matching file or package directory wins.

Local application modules can shadow package, `TYA_PATH`, and standard library
modules. Package dependencies can shadow `TYA_PATH` and standard library
modules.

## Public API

The public API of a single-file source is the set of public top-level bindings.

```tya
# path.tya
join = left, right -> left + "/" + right
_normalize = value -> value
```

```tya
import path

print path.join("a", "b")
```

The public API of a directory package is the set of public class files. A class
is public when its class name matches its file name:

```text
User.tya   -> public User
Helper.tya -> public Helper
```

Additional classes in a class file are private to that file.

## Internal Imports

Modules inside a library or package should import each other with the same
public import paths that applications use.

Recommended:

```tya
# libs/web/http/server.tya
import http/request

handle = raw ->
  request.parse(raw)
```

Avoid depending on relative filesystem imports. They are not part of the module
path model.

## Naming Guidance

Use directory names to group related modules.

Good:

```text
http/server.tya
http/request.tya
http/response.tya
json/parser.tya
json/encoder.tya
```

Avoid repeated names inside the final import segment when an alias can solve
call-site clarity.

Less useful:

```text
http/http_server.tya
json/json_parser.tya
```

Prefer:

```text
http/server.tya
json/parser.tya
```

with aliases at use sites when needed:

```tya
import http/server as http_server
import json/parser as json_parser
```

## Relationship To Standard Library Modules

User libraries, packages, and standard library modules share the same import
syntax.

The difference is ownership and distribution:

- user libraries are directory trees provided by the application or by
  `TYA_PATH`
- packages are versioned dependency units declared in `tya.toml`
- standard library modules are shipped with Tya

The loader searches user-controlled and manifest-controlled locations before the
standard library, so a project can provide its own source with the same import
path as a standard library module. This shadowing should be used carefully
because it can make code harder to read.

## Examples

Single-file module:

```text
app.tya
greeting.tya
```

```tya
# greeting.tya
hello = name -> "Hello, {name}"
```

```tya
# app.tya
import greeting

print greeting.hello("komagata")
```

Reusable package:

```text
text_tools/
  tya.toml
  src/
    text/
      slug.tya
```

```tya
# text_tools/src/text/slug.tya
make = value -> value
```

```toml
[dependencies]
text_tools = { path = "../text_tools" }
```

```tya
# app.tya
import text/slug

print slug.make("Hello Tya")
```
