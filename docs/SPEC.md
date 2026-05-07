# Tya v0.1 Language Spec

This document is the language specification for Tya v0.1.

The canonical v0.1 execution path lexes, parses, and checks Tya source, emits C,
and uses a C compiler to produce an executable.

```text
Tya source -> lexer -> parser -> AST -> checker -> C emitter -> C compiler -> executable
```

The Go interpreter, current `selfhost/*`, ASTMODE, legacy node strings, and
self-host bootstrap gates are not v0.1 specification authority. They do not
need to be deleted, but they are outside the maintained v0.1 path until the
v0.1 implementation is complete.

## Scope

Included in v0.1:

- `.tya` files
- 2-space indentation block syntax
- comments
- assignment
- multiple assignment
- arrays
- dictionaries
- index access
- function literals
- function calls
- `if` / `elseif` / `else`
- `while`
- `for value in array`
- `for value, index in array`
- `for key, value of dictionary`
- `break` / `continue`
- `return`
- multiple return values
- `try`
- `error`
- `module`
- `import module_name`
- `module.member`
- string interpolation
- v0.1 standard built-in functions
- compilation to C and execution

Not included in v0.1:

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
- using the Go interpreter as the canonical execution path
- including legacy node string compatibility in the specification

## Source Files

Tya source files use the `.tya` extension.

An entry file can contain imports and statements. An imported module file has
exactly one `module` declaration with the same name as the file.

```tya
# main.tya
import greeting

print greeting.hello("komagata")
```

```tya
# greeting.tya
module greeting
  hello = name -> "Hello, {name}"
```

## Lexical Structure

Comments start with `#` and continue to the end of the line.

```tya
# comment
name = "Tya"
```

Blocks are represented by indentation. One indentation level is 2 spaces. The
lexer emits `INDENT` and `DEDENT` tokens when indentation increases or
decreases.

Tokens include identifiers, integer literals, float literals, string literals,
newlines, indentation tokens, and the following operators and delimiters.

```text
= == != < <= > >= : , . + - * / % -> ( ) [ ] { }
```

The following identifiers have reserved meanings.

```text
if elseif else while for in of break continue return
import module
true false nil and or not try
```

`print`, `panic`, `exit`, `error`, and similar names are standard built-in
functions, not keywords.

## Names

Names follow `docs/NAMING.md`.

```text
variable / function:    snake_case
private binding:        _snake_case
module / file:          snake_case
dictionary key:         snake_case
module member:          snake_case
constant:               SCREAMING_SNAKE_CASE
type / class name:      PascalCase  # reserved in v0.1
```

## Values

v0.1 has the following runtime values.

```text
nil
boolean
integer
float
string
array
dictionary
function
error
module
```

v0.1 has no objects. If classes are introduced in the future, objects will be
defined as class instances.

## Strings

Strings support the following escapes.

```text
\" \\ \n \t
```

`{expression}` inside a string is interpolated.

```tya
name = "Tya"
print "Hello, {name}"
```

## Dictionaries

Dictionaries are key-value collections.

```tya
user = { name: "komagata", age: 20 }
empty = {}
```

Dictionary literal keys are `snake_case` identifiers in v0.1. Dictionary values
are read with index access.

```tya
print user["name"] # ok
print user.name    # invalid
```

In v0.1, `.` is reserved for module member access. Member access on dictionaries
is invalid.

## Statements

assignment:

```tya
name = "Tya"
left, right = pair
items[0] = 10
user["name"] = "komagata"
```

expression statement:

```tya
print "hello"
```

conditional branch:

```tya
if score >= 90
  print "A"
elseif score >= 80
  print "B"
elseif score >= 70
  print "C"
else
  print "D"
```

An `if` can have zero or more `elseif` branches. It can have zero or one final
`else` branch. `elseif` and `else` are placed at the same indentation as the
matching `if`. `elseif` is a single reserved identifier, not `else if`.

while loop:

```tya
count = 0
while count < 3
  print count
  count = count + 1
```

array iteration:

```tya
for value in items
  print value

for value, index in items
  print "{index}: {value}"
```

dictionary iteration:

```tya
for key, value of user
  print "{key}: {value}"
```

loop control:

```tya
while true
  break

while true
  continue
```

return:

```tya
return
return value
return value, err
```

## Expressions

primary expression:

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
```

function literal:

```tya
name -> expression
left, right -> expression

name ->
  statement
```

call:

```tya
fn()
fn(arg)
fn(arg1, arg2)
module.member()
```

In v0.1, ordinary function calls must use `()`. No-paren call chains such as
`fn arg`, `fn arg1, arg2`, `fn arg1 arg2`, and `print len keys user` are not
part of the syntax.

As an exception, `print` is allowed as statement-level output syntax:
`print expression`. `print` accepts only the single expression that follows it.

```tya
print "hello"
print len(items)
print len(keys(user))
print add(2, 3)
```

Multiple arguments and nested calls are written explicitly with `()`.

```tya
add(2, 3)
len(keys(user))
has(user, "name")
```

index access:

```tya
items[0]
dictionary["name"]
```

module member access:

```tya
module_name.member_name
```

unary expression:

```text
not expression
-expression
try expression
```

Binary operator precedence, from lowest to highest, is as follows.

```text
or
and
== !=
< <= > >=
+ -
* / %
unary: not, -, try
call / member / index
primary
```

## Truthiness

`nil` and `false` are falsey. All other values are truthy.

## Functions

Functions create a child scope.

```tya
double = value -> value * 2

sum = values ->
  total = 0
  for value in values
    total = total + value
  total
```

If a block function has no explicit `return`, it returns the final expression.
Functions can return multiple values.

```tya
parse_user = text ->
  if text == ""
    return nil, error("empty user")
  return { name: text }, nil
```

## Errors

`error("message")` creates an error value with a `message`. In v0.1, `.` is
reserved for module member access, so read the message with `err["message"]`.

```tya
err = error("file not found")
print err["message"]
```

`try expression` can only be used inside a function. The target expression is
expected to return two values: `value, err`. If `err` is truthy, the current
function returns `nil, err`. If `err` is falsey, `value` becomes the expression
value.

```tya
load_user = text ->
  user = try parse_user(text)
  user["name"]
```

## Modules

`import module_name` reads `module_name.tya` from the same directory as the
importing file.

An imported module file has exactly one `module` declaration with the same name
as the file.

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

Module members are read with `module_name.member_name`. In v0.1, `.` is
reserved for module member access.

Import aliases are not included in v0.1.

## Standard Builtins

Standard built-in functions for v0.1 are defined in `docs/API.md`. Convenience
functions that are not required in v0.1 are not included as standard builtins.

## Execution

The canonical v0.1 execution path builds and runs an executable from Tya source.
The user-facing CLI does not require Go commands.

## Command Line

v0.1 has the following user-facing commands.

```sh
tya run file.tya [args...]
tya build file.tya -o output
tya version
```

`tya run` builds a temporary executable, runs it, and removes the temporary file
after execution. It has the same role as Go's `go run`; it is not interpreter
execution.

`tya build` builds an executable and leaves it at the specified output path. If
`-o` is not specified, it removes `.tya` from the input file basename and writes
the executable in the current directory.

```sh
tya build hello.tya
# writes ./hello

tya build examples/hello.tya
# writes ./hello

tya build examples/hello.tya -o bin/hello
# writes ./bin/hello
```

`tya version` prints the Tya version to stdout.

The Go interpreter and Go development commands are not the user-facing canonical
execution path for v0.1.

## v0.1 Reference Implementation

The following components are maintained as the v0.1 reference implementation.

```text
Go lexer
Go parser
Go AST
Go checker
Go C emitter
C runtime
v0.1 specification tests
```

The Tya-written compiler should be newly implemented on an AST basis after the
v0.1 specification works fully through the Go compile-to-C path.
