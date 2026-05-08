# Tya User Modules and Libraries

This document defines user-created modules and libraries outside the Tya
standard library.

The standard library is documented separately in `docs/STDLIB.md`. Shared terms
are defined in `docs/TERMINOLOGY.md`.

## Scope

This document covers:

- user modules
- user libraries
- library directory layout
- module path rules for libraries
- how applications import library modules
- the boundary between libraries and future packages

This document does not define:

- standard library modules
- built-in functions
- package management
- remote package installation
- version resolution
- lock files
- package registries

## User Module

A user module is a `.tya` file that declares one module and is loaded with
`import`.

Example:

```tya
# greeting.tya
module greeting
  hello = name -> "Hello, {name}"
```

```tya
# main.tya
import greeting

print greeting.hello("komagata")
```

Imported module files must contain only:

- top-level imports
- exactly one top-level module declaration

Entry files may contain imports and top-level statements.

## User Library

A user library is a directory tree of user modules intended to be reused by more
than one entry file or project.

A user library is not a package. It has no required manifest, version, registry,
download step, or dependency solver.

The current way to make a user library available is to put its library root on
`TYA_PATH`.

Example:

```text
libs/web/
  http/
    server.tya
    request.tya
    response.tya
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

## Library Root

A library root is a directory searched by the module loader.

Module paths are resolved relative to each library root.

For:

```tya
import http/server
```

and:

```sh
TYA_PATH=libs/web
```

the loader searches:

```text
libs/web/http/server.tya
```

The library root itself is not part of the import path.

## Module Path

A module path is the slash-separated name used after `import`.

Examples:

```tya
import greeting
import http/server
import data/json/parser
```

Each path segment must follow the module naming rule:

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

## Module File Rule

The file path must match the import path.

```text
import greeting          -> greeting.tya
import http/server       -> http/server.tya
import data/json/parser  -> data/json/parser.tya
```

The module declaration must match the final path segment.

```tya
# http/server.tya
module server
  listen = port -> print port
```

```tya
# data/json/parser.tya
module parser
  parse = text -> text
```

This keeps the visible namespace short:

```tya
import data/json/parser

parser.parse("{}")
```

Use an alias when the final segment is too generic or conflicts with another
import:

```tya
import data/json/parser as json_parser
import html/parser as html_parser

json_parser.parse("{}")
html_parser.parse("<p>hi</p>")
```

## Import Resolution

User library modules use the same import resolution order as other modules:

1. The importing file's directory.
2. Each directory listed in `TYA_PATH`, from left to right.
3. The bundled standard library directory.

The first matching file wins.

Local application modules can shadow modules from `TYA_PATH`, and both can
shadow standard library modules.

## Public API

The public API of a module is the set of non-private module members.

Public module members use normal names:

```tya
module path
  join = left, right -> left + "/" + right
```

Private module members start with `_`:

```tya
module path
  _normalize = value -> value
  join = left, right -> _normalize(left + "/" + right)
```

Classes exported from a module use `PascalCase`:

```tya
module client
  class Client
    init = url ->
      @url = url
```

Use:

```tya
import http/client

client.Client("https://example.com")
```

## Library Internal Imports

Modules inside a library should import each other with the same public module
paths that applications use.

Recommended:

```tya
# libs/web/http/server.tya
import http/request

module server
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

Avoid repeated names inside the final module segment when an alias can solve
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

## Relationship To Future Packages

A future package may contain one or more library roots, a manifest, version
metadata, dependencies, and package-manager behavior.

That future package layer should build on this module model instead of
replacing it.

Current user libraries are simple directory trees. They can be copied into a
project, vendored, or referenced through `TYA_PATH`.

## Relationship To Standard Library Modules

User libraries and standard library modules share the same import syntax and
module file rules.

The difference is ownership and distribution:

- user libraries are written and distributed by users
- standard library modules are shipped with Tya

The loader searches user-controlled locations before the standard library, so a
project can provide its own module with the same import path as a standard
library module. This shadowing should be used carefully because it can make code
harder to read.

## Examples

Single-file module:

```text
app.tya
greeting.tya
```

```tya
# greeting.tya
module greeting
  hello = name -> "Hello, {name}"
```

```tya
# app.tya
import greeting

print greeting.hello("komagata")
```

Reusable library:

```text
libs/text/
  text/
    case.tya
    slug.tya
```

```tya
# libs/text/text/slug.tya
module slug
  make = value -> value
```

```tya
# app.tya
import text/slug

print slug.make("Hello Tya")
```
