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
- future third-party packages

These terms should not be used interchangeably.

## Language Feature

A language feature is syntax or semantics built into the Tya language.

Examples:

- `if`
- `while`
- `for`
- `import`
- `module`
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
loaded from a `.tya` module file.

Built-in function names should use `snake_case`.

## Built-In Class

A built-in class is a class available without `import`.

Tya does not currently define standard built-in classes.

If Tya adds built-in classes later, they should be documented in `docs/API.md`
or a dedicated API reference, and this document should be updated with their
availability, construction rules, and relationship to user-defined classes.

Built-in classes should use `PascalCase`, like user-defined classes.

## User Module

A user module is a `.tya` module written by a program or application author.

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

User modules are loaded with `import`.

The normal module file rule is:

- one module per `.tya` file
- the module name matches the file name without `.tya`
- module names use `snake_case`
- module members use `snake_case`, except exported classes use `PascalCase`

User modules have higher import precedence than standard library modules.

## User Library

A user library is a directory tree of user modules intended to be reused by more
than one entry file or project.

User libraries are documented in `docs/LIBRARIES.md`.

A user library is not a package. It has no required manifest, version,
registry, download step, or dependency solver.

The current way to make a user library available is to put its library root on
`TYA_PATH`.

## Standard Library Module

A standard library module is a `.tya` module shipped with Tya and imported with
the same `import` syntax as user modules.

Examples:

```tya
import string
import array

print string.blank("  ")
print array.first(["tya"])
```

Standard library modules are defined in `docs/STDLIB.md`.

The standard library is not automatically imported. A program must use `import`
to access a standard library module.

Standard library modules are normal modules from the user's perspective. They
are not built-in functions and not language syntax.

## Attached Library

Attached library is the historical term used for the first Tya standard library
design: `.tya` modules shipped with Tya and resolved by the module loader after
user-local modules and `TYA_PATH`.

For current and future documentation, prefer the term standard library module.

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
library module. Use standard library module for importable `.tya` modules in
the Tya standard library.

## Native-Backed Standard Library Module

A native-backed standard library module is a future standard library module
whose public API is imported like a normal module but whose implementation is
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

`string.blank` is a standard library function inside the `string` standard
library module because user code must import `string` before using it.

```tya
print "hello"

import string
print string.blank("")
```

### Standard Library Module vs Bundled Library

`stdlib/string.tya` is both a standard library module and a bundled library
file.

The module concept matters to language users:

```tya
import string
```

The bundled-library concept matters to installers and packagers:

```text
share/tya/stdlib/string.tya
```

### Standard Library Module vs Package

The standard library ships with Tya.

A package, if added later, would be separately distributed or versioned. A
package manager is out of current scope.

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

If a module is imported from `stdlib/`, call it a standard library module.

If a future module is available without import, specify the feature directly
instead of calling it a built-in module, because it would need separate rules
for naming, shadowing, and member access.
