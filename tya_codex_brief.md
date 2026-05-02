# Tya Language: Project Brief for Codex

## Goal

Build Tya, pronounced "cha".

Tya is a small dynamically typed programming language inspired by CoffeeScript.

It should:

- use `.tya` files
- be indentation-based
- prioritize human writeability and short syntax
- compile to native binaries eventually
- start with a Go implementation
- later become self-hosted
- enforce lint-like rules as compile errors

Repository name:

```text
tya-language
```

## Implementation Plan

Start with a Go implementation.

Initial implementation should be:

- hand-written lexer
- hand-written parser
- AST
- interpreter
- tests
- examples

Do not start with LLVM.
Do not start with C code generation.
Do not start with Tree-sitter.
Do not use ANTLR.

The first goal is to run simple Tya programs with a Go interpreter.

Later phases:

1. Go interpreter
2. Go compiler that emits C
3. C runtime
4. Tya standard library
5. Tya compiler written in Tya
6. self-hosting

## Suggested Repository Structure

```text
tya-language/
  cmd/
    tya/
      main.go

  internal/
    lexer/
    parser/
    ast/
    eval/
    checker/
    codegen/

  examples/
    hello.tya
    object.tya
    function.tya
    method.tya
    error.tya

  tests/
```

Do not add `runtime/` yet unless C code generation is being implemented.

## Milestone 1

Implement only enough to run these examples.

### examples/hello.tya

```tya
print "Hello, Tya"
```

### examples/object.tya

```tya
user =
  name: "komagata"
  age: 20

print "Hello, {user.name}"
```

### examples/function.tya

```tya
greet = user -> "Hello, {user.name}"

user =
  name: "komagata"
  age: 20

print greet user
```

### examples/method.tya

```tya
counter =
  count: 0

  inc: ->
    @count = @count + 1
    @count

print counter.inc()
print counter.inc()
```

For Milestone 1, implement only the minimum needed for these examples.

## Language Basics

Tya is indentation-based.

Indentation rules:

- exactly 2 spaces
- tabs are forbidden
- inconsistent indentation is a compile error

Comments:

```tya
# this is a comment
print "hello" # inline comment
```

Only `#` line comments exist.
No block comments in v1.

## Files and Modules

File extension:

```text
.tya
```

File names must be `snake_case.tya`.

Valid:

```text
main.tya
user_utils.tya
string_tools.tya
```

Invalid:

```text
UserUtils.tya
user-utils.tya
userUtils.tya
```

Only `main.tya` may contain top-level executable code.

Other `.tya` files are library files and may only contain top-level definitions.

Modules are not required in Milestone 1.

## Naming Rules

Variables use `snake_case`.

```tya
user_name = "komagata"
retry_count = 3
```

Functions use `snake_case`.

```tya
parse_user = text ->
  text
```

Object properties use `snake_case`.

```tya
user =
  user_name: "komagata"
  age: 20
```

Constants use `SCREAMING_SNAKE_CASE`.

```tya
MAX_RETRY = 3
```

Constants cannot be reassigned.

```tya
MAX_RETRY = 3
MAX_RETRY = 5 # compile error
```

Private top-level definitions start with `_`.

```tya
_parse_user = text ->
  text
```

Naming rules should eventually be compile errors, but Milestone 1 may implement them gradually.

## Dynamic Values

Tya is dynamically typed.

Runtime value kinds:

```text
nil
bool
int
float
string
array
object
function
error
```

For Milestone 1, implement at least:

```text
nil
bool
int
float
string
object
function
```

Arrays can come later unless needed.

## nil and Truthiness

Tya has `nil`.

Falsy values:

```text
nil
false
```

Everything else is truthy.

Therefore these are truthy:

```text
0
""
[]
{}
```

## Undefined Variables and Missing Properties

Reading an undefined variable is a compile/runtime error.

```tya
user_name = "komagata"
print user_nmae # error
```

Reading a missing object property returns `nil`.

```tya
user =
  name: "komagata"

print user.age # nil
```

Out-of-range array access should return `nil`, but arrays are not required in Milestone 1.

## Assignment and Scope

Variables may be reassigned.

```tya
name = "komagata"
name = "alice"
```

Because Tya is dynamically typed, the value kind may change.

```tya
value = 1
value = "hello"
```

Variable shadowing is forbidden.

A nested scope must not define a new variable with the same name as one in an outer scope.

For Milestone 1, implement basic reassignment first. Strict shadowing checks can be added after the interpreter works.

## Functions

Function syntax uses `->`.

Multiline function:

```tya
greet = name ->
  "Hello, {name}"
```

One-line function:

```tya
add = a, b -> a + b
```

No-argument function:

```tya
say_hello = ->
  print "hello"
```

The last expression is returned implicitly.

Explicit `return` is allowed eventually, but not required for Milestone 1.

Functions are first-class values.

## Function Calls

Parentheses are optional.

```tya
print "hello"
greet "komagata"
move player, 10, 20
```

Parentheses are also allowed eventually:

```tya
print("hello")
greet("komagata")
```

For Milestone 1, support the no-parentheses style used in the examples.

## Objects

Tya uses CoffeeScript-style object literals.

```tya
user =
  name: "komagata"
  age: 20
```

One-line objects can be added later:

```tya
user = { name: "komagata", age: 20 }
```

Objects are mutable.

```tya
user.name = "alice"
```

## Methods and @

Tya does not have a `this` keyword.

Use `@` inside object functions.

```tya
counter =
  count: 0

  inc: ->
    @count = @count + 1
    @count
```

`@count` means the receiver object's `count` property.

This should work when called as:

```tya
counter.inc()
```

For Milestone 1, implement enough method binding to support `counter.inc()`.

Do not support CoffeeScript constructor shorthand like `(@name)`.

## Strings

Strings are UTF-8.

Only double-quoted string literals are valid.

```tya
name = "komagata"
```

Single quotes are invalid.

```tya
name = 'komagata'
```

String interpolation uses `{...}`.

```tya
name = "komagata"
print "Hello, {name}"
print "next year: {age + 1}"
```

Literal braces use doubled braces eventually:

```tya
print "literal braces: {{ and }}"
```

For Milestone 1, implement interpolation for simple identifiers and property access if possible.

Required:

```tya
"Hello, {user.name}"
```

## Numbers

Tya internally distinguishes Int and Float.

Int:

```text
64-bit signed integer
```

Float:

```text
64-bit double
```

Integer literals are Int.

```tya
x = 1
```

Float literals are Float.

```tya
x = 1.5
```

Supported numeric literals eventually:

```text
10
0xFF
0b1010
1.5
```

For Milestone 1, decimal int and float are enough.

Arithmetic:

```tya
1 + 2
1 + 2.5
```

Rules:

```text
Int + Int -> Int
Float + Float -> Float
Int + Float -> Float
Float + Int -> Float
```

`/` is always Float division.

```tya
5 / 2 # 2.5
```

Integer division uses `div` later.

Overflow should eventually panic.

## Operators

Arithmetic:

```text
+
-
*
/
%
```

Bit operators eventually:

```text
&
|
^
~
<<
>>
```

Bit operators are not required in Milestone 1.

Equality:

```text
==
!=
```

Object and array equality is reference equality.

Deep equality is provided later by standard function:

```tya
equal a, b
```

Comparison:

```text
<
<=
>
>=
```

For v1:

- number comparisons are allowed
- string equality is allowed
- string ordering is forbidden
- object/array ordering is runtime error

Logical operators:

```text
and
or
not
```

Rules:

- `and` and `or` use truthiness
- `and` and `or` short-circuit
- `and` and `or` return values, not necessarily Bool
- `not` returns Bool

Example:

```tya
name = user.name or "anonymous"
```

## Control Flow

### if

```tya
if user
  print user.name
else
  print "no user"
```

### postfix if

Eventually support postfix `if`.

```tya
print user.name if user
return nil if not user
```

Postfix `if` is not required in Milestone 1.

Do not implement `unless` in v1.

### for

Eventually:

```tya
for user in users
  print user.name
```

With index:

```tya
for user, index in users
  print "{index}: {user.name}"
```

Object iteration:

```tya
for key, value of object
  print "{key}: {value}"
```

Loops are not required in Milestone 1 unless needed.

### while

Eventually:

```tya
while running
  tick()
```

`break` and `continue` should exist eventually.

## Error Handling

Tya does not use exceptions in v1.

Use `value, err`.

Success:

```text
value, nil
```

Failure:

```text
nil, err
```

Example:

```tya
text, err = read_file "memo.txt"

if err
  print err.message
else
  print text
```

Built-in `error` eventually creates an error object.

```tya
err = error "file not found"
print err.message
```

`try` eventually propagates `value, err`.

```tya
read_user = path ->
  text = try read_file path
  user = try parse_user text
  user, nil
```

Equivalent to:

```tya
read_user = path ->
  text, err = read_file path
  return nil, err if err

  user, err = parse_user text
  return nil, err if err

  user, nil
```

Error handling is not required in Milestone 1 except for internal interpreter errors.

## Standard Library

Standard library functions are implicitly available.

Names use `snake_case`.

Initial target list:

```text
print
read_line
read_file
write_file
file_exists

split
join
trim
replace
contains
starts_with
ends_with
byte_len
char_len
to_string
to_number
to_int
to_float

len
push
pop
map
filter
find
any
all
each
reduce

keys
values
has
delete
equal

error
panic

args
env
exit

div
```

For Milestone 1, only `print` is required.

## Strict Compile-Time Checks

Tya should eventually enforce lint-like rules as compile errors.

Examples:

```text
tabs in indentation
indentation not equal to 2 spaces
trailing whitespace
undefined variable reference
unused local variable
unused function argument
unused import
unused private top-level definition
variable shadowing
duplicate definition in same scope
reassignment of constants
invalid file name
invalid naming convention
top-level executable code outside main.tya
library file containing executable top-level code
```

Explicitly discarded values use `_`.

```tya
_ = do_something()
```

Unused arguments can be `_`.

```tya
handler = _ ->
  print "called"
```

Do not try to implement all strict checks in Milestone 1.

Build them gradually after the interpreter runs examples.

## Memory Management

Users do not manage memory.

Tya uses GC eventually.

Initial GC strategy for the C runtime:

```text
mark and sweep
```

No manual free/delete.
No ownership model.
No finalizers in v1.
No weak references in v1.

The Go interpreter can rely on Go's GC.

## Self-Hosting Roadmap

Phase 1:

```text
Go lexer/parser/interpreter
```

Phase 2:

```text
Go compiler that emits C
```

Phase 3:

```text
C runtime
```

Phase 4:

```text
standard library
```

Phase 5:

```text
Tya lexer written in Tya
```

Phase 6:

```text
Tya parser written in Tya
```

Phase 7:

```text
Tya code generator written in Tya
```

Phase 8:

```text
Tya compiler compiles itself
```

## Development Rules for Codex

When implementing:

1. Keep the implementation small.
2. Prefer explicit Go code.
3. Do not add clever abstractions early.
4. Write tests for lexer/parser behavior.
5. Add one language feature at a time.
6. Preserve source locations in AST nodes.
7. Produce clear error messages.
8. Do not add LLVM.
9. Do not add ANTLR.
10. Do not add package management.
11. Do not add async, classes, macros, or exceptions.
12. Do not implement C codegen until the interpreter runs the milestone examples.

## First Task

Create the Go project skeleton.

Then implement enough lexer, parser, AST, and interpreter to run:

```text
examples/hello.tya
examples/object.tya
examples/function.tya
examples/method.tya
```

Do not implement more than necessary.

The first milestone is:

```text
Can Tya run small pleasant CoffeeScript-like programs?
```
