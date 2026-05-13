# Naming

Tya uses names to express both naming category and accessibility.
Names outside these rules are language errors.

## Rules

```text
variables/functions: snake_case
private binding:     _snake_case
import paths/files:  snake_case
dictionary keys:     snake_case
public members:      snake_case or PascalCase classes
constants:           SCREAMING_SNAKE_CASE
classes:             PascalCase
```

## Import Path Rule

Single-file imports use the source file name without `.tya` as their import
path segment.

```text
file_system.tya -> file_system
http_client.tya -> http_client
json_parser.tya -> json_parser
```

Allowed:

```tya
# file_system.tya
read = path -> read_file path
exists = path -> file_exists path
```

Forbidden:

```tya
# file_system.tya
read-file = path -> read_file path
```

The second example is invalid because public binding names must follow the
public member naming rule.

Use source from another file with `import`:

```tya
import file_system

print file_system.exists("memo.txt")
```

`import file_system` loads `file_system.tya` from the same directory as the
importing file.

## Accessibility

Top-level bindings beginning with `_` are private to the source file.

```tya
# path.tya
_normalize_path = path -> path
```

Dictionary keys beginning with `_` are not source-file privacy. They may be
reserved for a future visibility rule, but privacy is enforced only for
bindings.

## Builtins

Standard library APIs use snake_case names. CamelCase builtin spellings are not
part of the language surface.
