---
layout: doc
title: Spec
permalink: /v0.2.0/spec/
---

# Tya v0.2 Draft Language Spec

This document is the draft language specification for Tya v0.2.

v0.2 builds on v0.1. The theme is friendly scripting: practical builtins,
better diagnostics, formatting, and small module ergonomics without introducing
objects or classes.

v0.1 remains the baseline. Anything not changed here follows the v0.1
specification.

## Goals

- Make everyday scripts easier to write and read.
- Improve the command-line experience for users and language implementers.
- Keep the language small and indentation-based.
- Preserve the compile-to-C execution path.
- Avoid object and class design until the data model is more mature.

## Included in v0.2

v0.2 adds:

- practical collection builtins
- practical equality and input builtins
- better diagnostics
- formatting command
- check command
- public C emission command
- small module import ergonomics

## Not Included in v0.2

v0.2 still does not include:

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
- async
- macro
- exception

## Dictionary Direction

Dictionaries remain the main structured data value in v0.2.

Dictionary member access is still invalid.

```tya
print user["name"] # ok
print user.name    # invalid
```

v0.2 should improve dictionary usefulness without changing dictionary syntax
into object syntax.

## Standard Builtins

v0.2 keeps every v0.1 standard builtin and adds the following standard builtins.

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
- Keep array and dictionary inline literals inline when they already fit on one
  line.
- Do not rewrite names.
- Do not change semantics.

The formatter is allowed to be conservative in v0.2. It should prefer stable,
predictable output over aggressive rewriting.

## Module Ergonomics

v0.2 keeps v0.1 module syntax:

```tya
import module_name

print module_name.member()
```

Import aliases are still not included.

v0.2 may improve module loading rules without adding a package manager.

The initial v0.2 module search order is:

1. The importing file's directory.
1. Directories listed in `TYA_PATH`, searched left to right.

`TYA_PATH` uses the host platform's path-list separator.

The module file name still matches the module name.

```text
json_parser.tya -> module json_parser
```

## Reference Implementation

The v0.2 reference implementation remains the Go compile-to-C implementation:

```text
Go lexer
Go parser
Go AST
Go checker
Go C emitter
C runtime
v0.2 specification tests
```

The current execution path remains:

```text
Tya source -> lexer -> parser -> AST -> checker -> C emitter -> C compiler -> executable
```

The Tya-written compiler can continue separately, but v0.2 language authority
comes from the spec and the Go compile-to-C reference implementation.
