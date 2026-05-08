# Naming

Tya uses names to express both naming category and accessibility.
Names outside these rules are language errors.

## Rules

```text
variables/functions: snake_case
private binding:     _snake_case
modules/files:       snake_case
dictionary keys:     snake_case
module members:      snake_case or PascalCase classes
constants:           SCREAMING_SNAKE_CASE
classes:             PascalCase
```

## Module Rule

A module file defines exactly one top-level `module`, and its name must match
the file name without `.tya`.

```text
file_system.tya -> file_system
http_client.tya -> http_client
json_parser.tya -> json_parser
```

Allowed:

```tya
module file_system
  read = path -> read_file path
  exists = path -> file_exists path
```

Forbidden:

```tya
module file_system
  read = path -> read_file path

module path
  join = left, right -> left + "/" + right
```

The second example is invalid because `file_system.tya` would define two
modules: `file_system` and `path`.

Use a module from another file with `import`:

```tya
import file_system

print file_system.exists("memo.txt")
```

`import file_system` loads `file_system.tya` from the same directory as the
importing file.

## Accessibility

Module members beginning with `_` are private to the module.

```tya
module path
  _normalize_path = path -> path
```

Dictionary keys beginning with `_` are not module privacy. They may be reserved
for a future visibility rule, but v0.1 privacy is enforced only for module
members.

## Builtins

Standard library APIs use snake_case names. CamelCase builtin spellings are not
part of the language surface.
