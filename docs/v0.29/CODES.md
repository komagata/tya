# Tya Diagnostic Codes

Every Tya diagnostic carries a stable code of the form `TYA-Xnnnn`:

- `X` is `E` for errors and `W` for warnings.
- `nnnn` is a zero-padded four-digit number.

Ranges are pre-allocated by stage so future migrations have room:

| Range           | Stage    |
|-----------------|----------|
| `E0001`–`E0099` | lexer    |
| `E0100`–`E0299` | parser   |
| `E0300`–`E0599` | checker  |
| `E0600`–`E0799` | codegen  |
| `E0800`–`E0899` | runner   |
| `E0900`–`E0999` | fmt      |
| `W1000`+        | warnings |

v0.29 ships the checker codes below. Lexer, parser, codegen, runner, and
fmt errors will be migrated in later releases.

---

## TYA-E0301 — Shadowed binding

A new binding (for-loop variable, catch binding, match-pattern binding,
or function-local introduction) reuses a name that is visible in a
strictly enclosing scope.

```tya
count = 0
for count in [1, 2, 3]   # TYA-E0301: count shadows outer count
  print count
```

Fix: rename the inner binding, or prefix it with `_` to mark it as
intentional (`_count`).

## TYA-E0302 — Unused import

A module is imported but no expression in the file references it.

```tya
import string            # TYA-E0302: string is imported but never used
print "hi"
```

Fix: remove the import, or reference the module somewhere in the file.

## TYA-E0303 — Unused argument

A function parameter is never read in the body.

```tya
greet = value -> "hello"  # TYA-E0303: value is unused
print greet(1)
```

Fix: rename the parameter to `_`, or prefix it with `_` (e.g. `_value`)
to mark it intentional.

## TYA-E0304 — Unused private definition

A top-level binding whose name starts with `_` is never referenced
elsewhere in the file. Names starting with `_` are private to the file
by convention; declaring one and never using it is treated as dead code.

```tya
_helper = 42              # TYA-E0304: _helper is never referenced
print "hi"
```

Fix: remove the definition, or reference it elsewhere in the file.

## TYA-E0305 — Duplicate parameter

A function declares the same parameter name more than once.

```tya
f = x, x -> x             # TYA-E0305: duplicate parameter x
```

Fix: rename one of the parameters.

## TYA-E0306 — Duplicate binding in pattern

A `match` pattern binds the same name in more than one position.

```tya
match ["a", "b"]
  case [name, name]       # TYA-E0306: name is bound twice
    print name
```

Fix: rename one of the bindings, or compare with an equality check in a
guard once guards land.
