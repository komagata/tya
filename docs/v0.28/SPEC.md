# Tya v0.28 Specification

This document is the specification for Tya v0.28 after v0.27 hexadecimal
and binary integer literals.

## Theme

Tya v0.28 turns the project's strict-lint goals into compile errors by
default.

The Tya brief from the start lists "lint-like rules as compile errors" as a
core principle. v0.28 adopts the most useful subset and makes them
mandatory: shadowing, unused imports, unused function arguments, and
unused private top-level definitions.

The effect is fewer silent bugs at the cost of slightly more discipline at
write time. Where intent is genuinely "I do not need this name," the
discard form `_` makes it explicit.

## Goals

- Reject variable shadowing.
- Reject unused `import` bindings.
- Reject unused function arguments (with `_` opt-out).
- Reject unused private top-level definitions (names starting with `_`).
- Make these checks default compile errors, not opt-in.
- Keep migration straightforward: the discard underscore covers each case.

## Included in v0.28

v0.28 includes all v0.27 behavior and adds:

- **Shadowing**: a name introduced in a nested scope must not match a
  binding visible from the enclosing scope.
- **Unused imports**: `import foo` (with or without `as`) is an error if
  the binding is never read.
- **Unused function arguments**: a parameter that is never referenced in
  the function body is an error unless its name is `_` or starts with `_`.
- **Unused private top-level definitions**: a top-level binding whose
  name starts with `_` and which is never referenced is an error.

## Not Included in v0.28

v0.28 does not include:

- Unused **local** variables as a compile error (the existing
  `--check-unused` opt-in flag remains; it is not promoted to the default
  in v0.28 to avoid a wave of churn in third-party code).
- Tab-character-in-indentation as a compile error (already rejected at
  lexer level; this is unchanged).
- Trailing-whitespace lint.
- "Top-level executable code only in `main.tya`" enforcement (a separate
  breaking change to entry-program semantics, deferred).
- Naming-convention enforcement beyond what already exists (snake_case
  file names, predicate `?` rules from v0.19, private `_` prefix; v0.28
  does not add new naming bans).
- Duplicate-definition tightening beyond what already exists.

## Shadowing

A binding *shadows* another when a nested lexical scope introduces a name
that is visible from a strictly enclosing scope.

### Forbidden examples

```tya
x = 1
greet = ->
  x = 2          # error: shadows outer x
  print x

count = 0
for count in items   # error: shadows outer count
  print count

read_user = ->
  user = parse()
  if user
    user = repair(user)   # error: shadows outer user
    print user
```

### Allowed examples

```tya
greet = ->
  x = 2
  print x

x = 1
print x

# Sibling scopes do not see each other.
process = ->
  total = 0
  for item in items
    total = total + item
  total

count = ->
  total = 1
  total
```

A binding can be reassigned in the **same** scope without error:

```tya
x = 1
x = 2          # ok: same scope, not a new binding
```

### What counts as a binding

The following introduce a new binding for the purpose of shadowing checks:

- Top-level `name = value`
- Function parameters (`name = arg1, arg2 -> ...`)
- `for x in xs` and `for x, y in xs` loop variables
- `for k, v of obj` variables
- `try ... catch name ...` catch binding
- Assignment statements inside a nested scope where the name is not
  already bound in that same scope

### Underscore discards do not shadow

Names that are exactly `_` are not bindings. Multiple `_` in the same scope
are allowed, and `_` does not collide with any other name. Names that
begin with `_` (private discards) are bindings and **do** participate in
shadowing checks.

## Unused Imports

```tya
import string       # error: unused
import array
import string as s  # error: unused

print array.first(items)
```

A `import` is unused when no expression in the importing file references
the bound name (the alias if `as` is used, otherwise the module name).

### Opt-out

There is no opt-out. The fix is to remove the import. Importing for side
effects is not a Tya idiom; `import` only binds a module.

## Unused Function Arguments

```tya
greet = name -> "Hello, world"   # error: name is unused

handler = event ->
  log("hit")
  nil

# fix 1: rename to underscore prefix
handler = _event ->
  log("hit")
  nil

# fix 2: rename to bare underscore
handler = _ ->
  log("hit")
  nil

# fix 3: actually use the argument
greet = name -> "Hello, {name}"
```

### Method receivers

The implicit `@` receiver is not a parameter and is not subject to this
check.

### Class abstract method declarations

Abstract method declarations (`abstract method = a, b ->` without a body)
are signature-only. Their parameter names are not checked for use.

### Pattern parameters (destructuring)

Destructuring parameters are checked at the level of bound names. If you
destructure a value but never use one of the names, that name needs `_`:

```tya
process = [first, _second, third] ->
  use(first, third)
```

(Destructuring as a parameter form is not introduced in v0.28; this
clarification is forward-compatible.)

## Unused Private Top-Level Definitions

```tya
# private_helper is never used in this file
_private_helper = x -> x + 1

print "main"
```

Top-level bindings whose names start with `_` are private to their file.
v0.28 makes "defined but never used" a compile error for these names so
that dead helper functions are caught.

### Opt-out

Rename without the leading underscore (and accept it becomes part of the
file's public surface), or delete the definition. There is no inline
suppression.

## Migration

Existing code that does not satisfy the new rules has three migration
options for each violation:

| Violation | Fix |
|---|---|
| Unused import | Remove the import |
| Unused arg `name` | Rename to `_name` or `_` |
| Unused private `_helper` | Remove or actually call it |
| Shadowing inner binding | Rename inner binding |
| Shadowing loop variable | Rename loop variable |

The Tya v0.28 release migrates the standard library, examples, and tests
internally. Third-party code may need a one-time pass.

## CLI

The existing `--check-unused` flag retains its current behavior (warns on
unused **local** variables, which v0.28 does not promote). All checks
described above are emitted as compile errors by every command that does
type/scope checking (`tya check`, `tya run`, `tya build`, `tya emit-c`,
`tya test`, `tya fmt -w`).

`tya fmt` without `-w` is a syntax-only formatter and does not run the
new strict checks.

## Diagnostics

v0.28 implementations should report source-oriented errors with one of
these messages (or close equivalents):

- `shadows outer binding <name>` — points at the inner binding
- `unused import <name>` — points at the `import` statement
- `unused argument <name>` — points at the parameter
- `unused private top-level definition <name>` — points at the binding

Diagnostics should mention the line and column and the offending name.

## Implementation Notes (non-normative)

- Shadowing: the checker maintains the chain of enclosing scopes. When
  binding a new name, walk the chain (excluding the current scope) and
  reject if any frame holds the same name.
- Unused tracking: each scope records `(declared, read)` sets per name.
  At scope exit, names in `declared` minus `read` are diagnosed.
- Underscore: treat the bare name `_` as a fresh scope-anonymous binding
  that no read can target.
- Top-level private: the checker scans assigned names starting with `_`
  before the body and counts a use whenever any expression contains an
  identifier matching the name.
- Tests: each rule's positive / negative case in
  `tests/strict_lint_test.tya` plus targeted txtar.

These notes are guidance, not the spec.
