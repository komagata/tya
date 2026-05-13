# Tya Terminology

This document defines shared terms for Tya language, runtime, and tooling
documents.

## Purpose

Tya uses several kinds of names that can look similar from user code:

- language syntax
- built-in functions
- built-in classes
- user modules
- user libraries
- standard library modules
- bundled libraries
- packages

These terms should not be used interchangeably.

## Language Feature

A language feature is syntax or semantics built into the Tya language.

Examples:

- `if`
- `while`
- `for`
- `import`
- `class`
- `interface`
- `try`
- `return`
- assignment
- function calls
- indentation-based blocks

Language features are defined in `docs/SPEC.md` and versioned specification
documents under `docs/v*/SPEC.md`.

A language feature is not imported and cannot be shadowed by a user module.

## Built-In Function

A built-in function is a function available without `import`.

Examples:

- `print`
- `len`
- `push`
- `trim`
- `read_file`
- `args`
- `error`

Built-in functions are part of the standard Tya API and are defined in
`docs/API.md`.

Built-in functions are implemented by the compiler, runtime, or host
implementation. User code calls them like normal functions, but they are not
loaded from a `.tya` importable source file.

Built-in function names should use `snake_case`.

## Built-In Class

A built-in class is a class available without `import`.

Tya does not currently define standard built-in classes.

If Tya adds built-in classes later, they should be documented in `docs/API.md`
or a dedicated API reference, and this document should be updated with their
availability, construction rules, and relationship to user-defined classes.

Built-in classes should use `PascalCase`, like user-defined classes.

## User Module

User module: `.tya` source, written by a program or application author, that can
be loaded by `import`.

Example:

```tya
# greeting.tya
hello = name -> "Hello, {name}"
```

```tya
# main.tya
import greeting

print greeting.hello("komagata")
```

User modules are loaded with `import`.

The normal import-file rule is:

- the import path matches a `.tya` file or a package directory
- import path segments use `snake_case`
- public top-level bindings use normal public names
- package class files use `PascalCase`

User modules have higher import precedence than standard library modules.

## User Library

A user library is a directory tree of user modules intended to be reused by more
than one entry file or project.

User libraries are documented in `docs/LIBRARIES.md`.

A user library is not necessarily a package. A raw library can be made
available through `TYA_PATH` without a manifest, version, download step, or
dependency solver.

A reusable library can also be distributed as a package by adding `tya.toml` and
placing importable source under `src/`.

## Standard Library Module

Standard-library module: `.tya` source shipped with Tya and imported with the
same `import` syntax as user modules.

Standard library modules are defined in `docs/STDLIB.md`.

The standard library is not automatically imported. A program must use `import`
to access a standard library module.

Standard library modules are normal modules from the user's perspective. They
are not built-in functions and not language syntax.

## Attached Library

Attached library is the historical term used for the first Tya standard library
design: `.tya` sources shipped with Tya and resolved by the import loader after
user-local modules and `TYA_PATH`.

For current and future documentation, prefer the term standard-library module.

When older documents say "standard attached library" or "attached standard
library", they mean standard library modules shipped with Tya.

## Bundled Library

A bundled library is any library distributed together with the Tya toolchain.

This is a packaging term, not a language category.

Bundled libraries may include:

- standard library modules under `stdlib/`
- runtime support files needed by generated C programs
- future tool support files

Do not use bundled library when the intended meaning is specifically standard
library module. Prefer that terminology for importable `.tya`
sources in
the Tya standard library.

## Native-Backed Standard Library Module

Native-backed standard-library module: future standard-library source whose
public API is imported like normal importable source but whose implementation is
partly or entirely provided by the runtime or host implementation instead of
plain `.tya` source.

Tya does not currently define native-backed standard library modules.

If added later, they should still be documented as standard library modules
because user code accesses them through `import`.

## Package

A package is a versioned distribution unit for third-party or separately
versioned Tya code.

Packages are declared with `tya.toml`, resolved into `tya.lock`, and loaded by
the import resolver from manifest-declared dependencies. Tya currently supports
git and path dependency sources, optional `[native]` C wrapper metadata, and
optional `[tools]` Tya script command declarations. It does not currently define
a central package registry, `tya publish`, binary package distribution, or
workspaces.

## Package Tool

A package tool is a lowercase Tya entry script declared by a package under
`[tools]` in `tya.toml` and run by consumers with `tya tool`.

Package tools are not global installs and are not shell tasks. A package tool
belongs to the package that declares it; a project runs it from locked
dependencies or an explicit one-shot git/path source.

Do not use package to describe the current standard library.

## Recommended Terms

Use these terms in new documentation:

```text
language feature
built-in function
built-in class
user module
user library
standard library module
bundled library
native-backed standard library module
package
package tool
```

Avoid these terms in new documentation unless quoting older material:

```text
stdlib function
attached library
attached standard library
standard attached library
built-in module
bundled stdlib
```

## Term Boundaries

### Built-In Function vs Standard Library Module

`print` is a built-in function because it is available without `import`.

`json.Json.parse` is a standard library method inside the `json` standard
library module; user code must import `json` before using it.

```tya
print "hello"

import json
data = json.Json.parse("{\"ok\": true}")
```

### Standard Library Module vs Bundled Library

`stdlib/json/Json.tya` is part of the importable `json` standard library module
and is also bundled with Tya.

The import concept matters to language users:

```tya
import json
```

The bundled-library concept matters to installers and packagers:

```text
share/tya/stdlib/json/Json.tya
```

### Standard Library Module vs Package

The standard library ships with Tya.

A package is separately distributed or versioned and is declared in
`tya.toml`. The current package manager supports git and path dependencies
without a central registry.

### Built-In Class vs User Class

A user class is declared in Tya source:

```tya
class User
  init = name ->
    @name = name
```

A built-in class would be available without a source declaration or import.
Tya currently has no standard built-in classes.

### Built-In Module

Avoid the term built-in module.

If code is imported from `stdlib/`, call it a standard-library module.

If a future importable source is available without import, specify the feature directly
instead of calling it a built-in module, because it would need separate rules
for naming, shadowing, and member access.
