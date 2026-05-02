# Naming

Tya uses names to express both naming category and accessibility.
Names outside these rules are language errors.

## Rules

```text
variables/functions: snake_case
private binding:     _snake_case
modules/files:       snake_case
object properties:   snake_case
constants:           SCREAMING_SNAKE_CASE
types/classes:       PascalCase  # reserved for future use
```

## Module Rule

A module file exposes exactly one public top-level binding, and its name must
match the file name without `.tya`.

```text
file_system.tya -> file_system
http_client.tya -> http_client
json_parser.tya -> json_parser
```

Allowed:

```tya
file_system =
  read: path -> _read_file path
  exists: path -> _exists path

_read_file = path ->
  read_file path

_exists = path ->
  file_exists path
```

Forbidden:

```tya
file_system =
  read: path -> _read_file path

path =
  join: left, right -> left + "/" + right
```

The second example is invalid because `file_system.tya` would expose two public
top-level bindings: `file_system` and `path`.

## Accessibility

Top-level bindings beginning with `_` are private to the module.

```tya
_normalize_path = path -> path
```

Object properties beginning with `_` are not module privacy. They may be
reserved for a future object visibility rule, but initially privacy is enforced
only for top-level module bindings.

## Builtins and Migration

Existing camelCase builtins are compatibility names. New standard library APIs
should use snake_case canonical names.

```text
readFile   -> read_file
writeFile  -> write_file
fileExists -> file_exists
startsWith -> starts_with
endsWith   -> ends_with
toString   -> to_string
toInt      -> to_int
```
