# Tya Reference

This document is the compact language reference for Tya. It favors exactness
over tutorial flow.

## Source Files

Tya source files use the `.tya` extension. A program is a list of statements.
Newlines separate statements.

## Lexical Structure

Comments start with `#` and continue to the end of the line.

```tya
# comment
name = "Tya"
```

Blocks are indentation-based. One indentation level is two spaces. The lexer
emits `INDENT` and `DEDENT` tokens when indentation increases or decreases.

Tokens include identifiers, integer literals, float literals, string literals,
newlines, indentation tokens, and these operators or delimiters:

```text
= == != < <= > >= : , . @ + - * / % -> ( ) [ ] { }
```

Keywords are parsed as identifiers with reserved meanings:

```text
if else while for in of break continue return
true false nil and or not try
```

## Names

Names must follow `docs/NAMING.md`.

```text
variables/functions: snake_case
private binding:     _snake_case
modules/files:       snake_case
object properties:   snake_case
constants:           SCREAMING_SNAKE_CASE
types/classes:       PascalCase  # reserved
```

## Values

Runtime values are:

```text
nil
boolean
integer
float
string
array
object
function
error
```

Strings support escapes for `\"`, `\\`, `\n`, and `\t`. Interpolation uses
`{expression}` inside a string.

## Statements

Assignment:

```tya
name = "Tya"
left, right = pair
items[0] = 10
user["name"] = "komagata"
```

Expression statement:

```tya
print "hello"
```

Condition:

```tya
if condition
  then_statement
else
  else_statement
```

Loop:

```tya
while condition
  statement
```

Array iteration:

```tya
for value in array
  statement

for value, index in array
  statement
```

Dictionary iteration:

```tya
for key, value of dictionary
  statement
```

Function return:

```tya
return
return value
return value, err
```

## Expressions

Primary expressions:

```text
identifier
integer
float
string
true
false
nil
[items]
{ name: value }
(expression)
@property
```

Function literals:

```tya
name -> expression
left, right -> expression

name ->
  statement
```

Calls:

```tya
fn(arg)
fn arg
fn arg1, arg2
object.method()
```

Member and index access:

```tya
module.name
items[0]
dictionary["name"]
```

Unary operators:

```text
not expression
-expression
try expression
```

Binary operator precedence, from low to high:

```text
or
and
== !=
< <= > >=
+ -
* / %
```

## Truthiness

`nil` and `false` are falsey. Other values are truthy.

## Functions

A function creates a child scope. A block function returns the last expression
unless `return` is used. A function can return multiple values.

When an object method is called through `object.method()`, `@property` reads or
writes the receiver object.

## Errors

`error "message"` creates an error value with a `message` property.

`try expression` is valid inside a function. It expects `expression` to produce
a `value, err` pair. If `err` is truthy, the current function returns
`nil, err`; otherwise the value is used.

## Modules

`import module_name` loads `module_name.tya` from the same directory as the
importing file.

Each module file exposes exactly one public top-level binding. The public
binding name must match the file name without `.tya`. Top-level bindings
starting with `_` are private to the module.

## Execution

The CLI can lex, check, interpret, emit C, or compile and run.

```sh
go run ./cmd/tya --tokens file.tya
go run ./cmd/tya --check-unused file.tya
go run ./cmd/tya --emit-c file.tya
go run ./cmd/tya run file.tya
```

The runner and C emitter load `stdlib/prelude.tya`.
