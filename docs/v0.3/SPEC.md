# Tya v0.3 Draft Language Spec

This document is the draft language specification for Tya v0.3.

v0.3 builds on v0.2. The theme is standard attached libraries: Tya should ship
small `.tya` modules that can be imported without a package manager.

v0.2 remains the baseline. Anything not changed here follows the v0.2
specification.

## Goals

- Make shared utility code available as ordinary Tya modules.
- Keep the language core small by avoiding format-specific builtins.
- Keep the language small and indentation-based.
- Preserve the compile-to-C execution path.
- Preserve the existing `import module_name` syntax.
- Avoid package management until local module behavior is mature.

## Included in v0.3

v0.3 adds:

- a `stdlib/` directory distributed with Tya
- standard attached library imports through existing `import module_name` syntax
- installed-tool lookup for the attached standard library
- initial lightweight standard modules such as `string` and `array`
- documentation for attached library APIs

## Not Included in v0.3

v0.3 does not include:

- object
- class
- interface
- inheritance
- `self`
- `super`
- `@property`
- object method
- class method
- class field
- import alias
- dictionary member access
- set literal
- package manager
- remote module install
- versioned dependencies
- native-backed standard modules
- JSON parser
- CSV parser
- async
- macro
- exception

## Standard Attached Libraries

Standard attached libraries are `.tya` modules shipped with the Tya
distribution. They are imported with the same syntax as user modules.

```tya
import string
import array

print string.blank("  ")
print array.empty([])
```

The standard attached library is not a package manager. It does not download
remote code, resolve dependency versions, or install third-party modules.

Standard modules should be written as normal Tya modules whenever practical.
v0.3 does not add native-backed standard modules. Format-heavy modules such as
JSON and CSV are intentionally deferred.

## Standard Library Search

v0.3 keeps v0.2 module syntax:

```tya
import module_name

print module_name.member()
```

Import aliases are still not included.

The v0.3 module search order is:

1. The importing file's directory.
1. Directories listed in `TYA_PATH`, searched left to right.
1. The `stdlib/` directory shipped with Tya.

`TYA_PATH` uses the host platform's path-list separator.

The module file name still matches the module name.

```text
string.tya -> module string
```

Installed tools must be able to find the shipped `stdlib/` directory even when
`tya` is run outside the source checkout. Packaged installs may place it under
the same shared data root as the C runtime, for example:

```text
share/tya/stdlib/string.tya
share/tya/stdlib/array.tya
```

## Initial Standard Modules

The initial v0.3 standard attached library should stay lightweight. It should
prove the attached-library mechanism before adding parser-heavy modules.

Initial module candidates:

- `string`
- `array`

Example `string` API:

```tya
import string

print string.blank("  ")
print string.present("tya")
```

Example `array` API:

```tya
import array

print array.empty([])
print array.first(["tya"])
```

The exact attached library API is documented in `docs/STDLIB.md`.

## Dictionary Direction

Dictionaries remain the main structured data value in v0.3.

Dictionary member access is still invalid.

```tya
print user["name"] # ok
print user.name    # invalid
```

v0.3 should improve library organization without changing dictionary syntax
into object syntax.

## v0.2 Baseline

v0.3 keeps the v0.2 builtins, commands, diagnostics, formatting behavior, and
module syntax unless this document explicitly changes them.

## Standard Builtins

v0.3 keeps every v0.2 standard builtin.

### Collections

```tya
map(items, item -> item * 2)
filter(items, item -> item > 10)
find(items, item -> item == "tya")
any(items, item -> item == "tya")
all(items, item -> len(item) > 0)
sum = total, item -> total + item
reduce(items, 0, sum)
```

`map(array, function)` returns a new array containing the mapped values.

`filter(array, function)` returns a new array containing values for which the
function returns truthy.

`find(array, function)` returns the first value for which the function returns
truthy. If no value matches, it returns `nil`.

`any(array, function)` returns `true` if the function returns truthy for at
least one value.

`all(array, function)` returns `true` if the function returns truthy for every
value.

`reduce(array, initial, function)` folds an array from left to right. The
function receives the current accumulator and the item.

### Equality

```tya
equal(left, right)
```

`equal(left, right)` performs deep equality for v0.2 runtime values.

The `==` operator keeps its existing v0.1 behavior. `equal` is the explicit
choice for deep equality in scripts and tests.

### Input

```tya
read_line()
```

`read_line()` reads one line from standard input and returns it without the
trailing newline. At end of input, it returns `nil`.

## Command Line

v0.2 keeps the v0.1 user-facing commands:

```sh
tya run file.tya [args...]
tya build file.tya -o output
tya version
```

v0.2 adds:

```sh
tya check file.tya
tya fmt file.tya
tya emit-c file.tya
```

`tya check` lexes, parses, and checks the file without writing an executable.
It exits with status `0` when the program is valid and non-zero when diagnostics
are emitted.

`tya fmt` writes formatted Tya source to standard output by default.

```sh
tya fmt file.tya
```

In-place formatting can be added with an explicit option:

```sh
tya fmt -w file.tya
```

`tya emit-c` emits generated C to standard output. This makes C emission a
public developer-facing command instead of a hidden inspection option.

## Diagnostics

v0.2 diagnostics should be friendly and source-oriented.

Diagnostics should include:

- file path
- line number
- column number
- concise message
- source line when available
- caret marker when available

Example format:

```text
main.tya:3:8: expected expression after +
  total = count +
                ^
```

Diagnostics should avoid Go implementation terms such as token struct names,
panic traces, or internal node names unless an explicit developer inspection
command asks for them.

## Formatting

`tya fmt` defines the canonical source layout.

Formatting rules:

- Use 2 spaces for each indentation level.
- Remove trailing whitespace.
- Keep one statement per line.
- Keep dictionary indentation readable.
- Keep array and dictionary inline literals inline when they already fit on one line.
- Do not rewrite names.
- Do not change semantics.

The formatter is allowed to be conservative in v0.2. It should prefer stable,
predictable output over aggressive rewriting.

## Reference Implementation

The v0.3 reference implementation remains the Go compile-to-C implementation:

```text
Go lexer
Go parser
Go AST
Go checker
Go C emitter
C runtime
v0.3 specification tests
```

The current execution path remains:

```text
Tya source -> lexer -> parser -> AST -> checker -> C emitter -> C compiler -> executable
```

The Tya-written compiler can continue separately, but v0.3 language authority
comes from the spec and the Go compile-to-C reference implementation.
