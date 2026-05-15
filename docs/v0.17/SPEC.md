---
layout: doc
title: Spec
permalink: /v0.17/spec/
---

# Tya v0.17 Specification

This document is the specification for Tya v0.17 after v0.16 pattern matching
and string interpolation polish.

## Theme

Tya v0.17 is about import aliases and module loading rules.

Tya already supports top-level `import module_name` and `module.member` access.
v0.17 keeps that model and makes imports more practical by adding aliases,
adding slash-separated module paths for structured libraries, fixing module
resolution order, documenting imported file shape, and requiring cycle
detection.

The feature intentionally avoids package management and selective imports.
Modules remain explicit namespaces.

## Goals

- Add `import module_name as alias`.
- Keep `import module_name` unchanged.
- Add slash-separated module paths such as `import http/server`.
- Bind an unaliased module path to the imported module's declared module name.
- Bind only the alias name when an alias is used.
- Detect import binding conflicts.
- Keep imports top-level only.
- Specify imported file shape.
- Specify module resolution order.
- Load each resolved module once.
- Detect import cycles with source-oriented diagnostics.

## Included in v0.17

v0.17 includes all v0.16 behavior and adds:

- import aliases
- alias-only binding
- slash-separated module paths
- module path to file path resolution
- import name conflict checks
- imported file shape rules
- same-directory, `TYA_PATH`, and bundled stdlib resolution order
- module load-once behavior
- import cycle detection
- cycle diagnostics

## Not Included in v0.17

v0.17 does not include:

- selective import
- wildcard import
- relative filesystem import syntax such as `./foo` or `../foo`
- dotted package import
- remote import
- package manager
- import inside functions or blocks
- conditional import
- re-export
- public export lists
- private export lists
- module initialization hooks
- dynamic import
- type annotations
- generics
- native-backed stdlib

## Import Without Alias

The existing import form stays valid.

```tya
import string

print string.blank("")
```

`import string` binds the module to the name `string`.

## Module Paths

An import name may be a slash-separated module path.

```tya
import http/server

server.listen(8080)
```

`import http/server` resolves `http/server.tya`.

The slash-separated import name is a module path, not a relative filesystem
path. It must not start with `/`, `./`, or `../`, and it must not contain empty
path segments.

Each module path segment uses the same naming rule as a module name:

```text
^[a-z][a-z0-9_]*$
```

The imported file declares a normal module using only the final path segment.

```tya
# http/server.tya
module server
  listen = port ->
    print "listening on {port}"
```

The default binding for an unaliased module path is the declared module name.
For a valid imported file, this is the final path segment.

```text
import path:  http/server
file path:    http/server.tya
module name:  server
binding name: server
```

This keeps module access consistent with existing `module.member` syntax while
allowing the standard library and future package layouts to be structured.

## Import Alias

An import may bind the module to an alias.

```tya
import string as str

print str.blank("")
```

The alias changes only the local binding name. It does not change the module's
declared name or the source file name.

When an alias is used, only the alias is bound.

```tya
import string as str

print str.blank("")
print string.blank("")
```

The final `print string.blank("")` is invalid because `string` is not bound by
the aliased import.

Aliases use the same identifier rules as variables and modules.

```tya
import very_long_module_name as short

print short.run()
```

Aliases also work with module paths.

```tya
import http/server as http_server

http_server.listen(8080)
```

## Import Binding Conflicts

An import binding must not conflict with another top-level binding.

```tya
import string as text
text = "hello"
```

This is an error because `text` is already used as an import binding.

Two imports must not bind the same name.

```tya
import string as util
import array as util
```

This is an error because both imports bind `util`.

Importing the same resolved module with different binding names is valid.

```tya
import string
import string as str
```

Both `string` and `str` refer to the same loaded module.

The same rule applies to module paths.

```tya
import http/server
import http/server as http_server
```

Both `server` and `http_server` refer to the same loaded module.

## Top-Level Imports Only

Imports remain top-level only.

```tya
load = ->
  import string
```

This is invalid.

Dynamic or conditional imports are not part of v0.17.

## Imported File Shape

An imported file must contain top-level imports and exactly one module
declaration.

```tya
# string.tya
module string
  blank = value ->
    value == ""
```

For a single-segment import, the module name must match the imported module
name and file name.

```tya
# greeting.tya
module message
  text = "hello"
```

`import greeting` is invalid for this file because it defines `module message`
instead of `module greeting`.

For a slash-separated module path, the file path must match the import path and
the module declaration must match the final path segment.

```tya
# http/server.tya
module server
  listen = port ->
    print port
```

`import http/server` is valid for this file.

```tya
# http/server.tya
module http_server
  listen = port ->
    print port
```

`import http/server` is invalid for this file because the final path segment is
`server`, but the file declares `module http_server`.

Imported files may import other modules before the module declaration.

```tya
# greeting.tya
import string

module greeting
  hello = name ->
    "Hello, {string.trim(name)}"
```

Imported files must not contain arbitrary top-level statements outside imports
and the single module declaration.

Entry files keep the existing behavior: an entry file may contain imports and
top-level statements.

## Resolution Order

`import foo` resolves `foo.tya` in this order:

1. The importing file's directory.
2. Each directory listed in `TYA_PATH`, from left to right.
3. The bundled standard library directory.

`import http/server` uses the same root search order, but resolves
`http/server.tya` under each root:

1. `http/server.tya` under the importing file's directory.
2. `http/server.tya` under each directory listed in `TYA_PATH`, from left to
   right.
3. `http/server.tya` under the bundled standard library directory.

The first matching file wins.

If no file is found, the import is an error.

Diagnostics should mention the module name and searched locations when
available.

## Module Load Once

Each resolved module file is loaded once and shared.

```tya
import counter
import counter as c
```

Both `counter` and `c` refer to the same module instance.

Load-once behavior is based on the resolved file path, not only the import
spelling.

## Import Cycles

Import cycles are errors.

```tya
# a.tya
import b

module a
  name = "a"
```

```tya
# b.tya
import a

module b
  name = "b"
```

This is invalid because `a` imports `b` and `b` imports `a`.

Cycle diagnostics should include the cycle path when available.

```text
import cycle: a -> b -> a
```

Longer cycles are also errors.

## Standard Library Imports

Bundled standard libraries use the same import syntax and alias rules.

```tya
import string as str
import array
import http/server

print str.blank("  ")
print array.first(["tya"])
server.listen(8080)
```

Local files and `TYA_PATH` entries can shadow bundled standard library modules
because they are earlier in the resolution order.

## Diagnostics

v0.17 implementations should report source-oriented errors for:

- invalid import alias syntax
- invalid alias names
- invalid module paths
- import binding conflicts
- imports outside the top level
- imported files with no module declaration
- imported files with multiple module declarations
- imported files with top-level statements other than imports and one module
- imported module name or final path segment mismatch
- missing imported module files
- import cycles

Diagnostics should mention the import path, alias name, resolved file path,
declared module name, and cycle path when available.
