# Tya Guide

Tya is an indentation-based, dynamically typed language that compiles to C.
The toolchain is intentionally all-in-one: the same `tya` command checks,
formats, builds, runs, tests, documents, and manages packages.

This guide shows the current language surface used by the latest repository
state. Historical release snapshots live under `docs/vX.Y/`.

## Run A Program

Create `hello.tya`.

```tya
print("Hello, Tya")
```

Run it during development:

```sh
tya run hello.tya
```

Build a reusable executable:

```sh
tya build hello.tya -o hello
./hello
```

Useful commands:

```sh
tya check hello.tya
tya format -w hello.tya
tya emit-c hello.tya
tya test
tya version
```

## Values

```tya
name = "Tya"
age = 1
pi = 3.14
ready = true
missing = nil
data = b"abc"
```

Strings support interpolation:

```tya
print("Hello, {name}")
```

Arrays and dictionaries are mutable:

```tya
items = [1, 2]
items.push(3)
print(items[0])

user = { name: "komagata", age: 20 }
print(user["name"])
user["admin"] = true
```

Primitive values expose methods:

```tya
print(" tya ".trim().upper())
print([1, 2, 3].len())
print(user.keys())
print(value.to_s())
print(value.class)
```

## Names

Use `snake_case` for variables, functions, methods, imports, and dictionary
keys. Use `PascalCase` for classes and interfaces. Use
`SCREAMING_SNAKE_CASE` for constants.

```tya
user_name = "komagata"
MAX_COUNT = 10

class UserProfile
  initialize = name ->
    self.name = name
```

See `docs/NAMING.md` for the full naming rules.

## Conditions

```tya
if age >= 20
  print("adult")
elseif age >= 13
  print("teen")
else
  print("young")
```

Use `and`, `or`, and `not` for logic.

```tya
if ready and not missing
  print("ok")
```

## Loops

```tya
count = 0
while count < 3
  print(count)
  count = count + 1
```

```tya
items = ["a", "b"]

for item in items
  print(item)

for item, index in items
  print("{index}: {item}")
```

Use `of` to iterate dictionary keys and values.

```tya
user = { name: "komagata", age: 20 }

for key, value of user
  print("{key}: {value}")
```

`break` and `continue` are available inside loops.

## Functions

Functions use `->`. Calls use parentheses.

```tya
greet = name -> "Hello, {name}"

print(greet("Tya"))
```

Use an indented body for multiple statements. The last expression is returned
implicitly.

```tya
double = value ->
  result = value * 2
  result
```

Use `return` for early returns or multiple return values.

```tya
parse_user = text ->
  if text == ""
    return nil, error("empty user")
  return { name: text }, nil
```

Anonymous functions can be passed directly.

```tya
items = [1, 2, 3]
print(items.map(item -> item * 2))
```

## Errors And `try`

Tya has structured error handling with `raise`, `try`, and `catch`.

```tya
read_name = path ->
  text = read_file(path)
  if text == ""
    raise "empty file"
  text.trim()

try
  print(read_name("name.txt"))
catch err
  print("error: {err}")
```

`try` can also wrap an expression inside a function body:

```tya
load_user = path ->
  try
    text = read_file(path)
    { name: text.trim() }
  catch err
    { name: "guest" }
```

For APIs that return `value, err`, destructure both values explicitly.

## Classes

Classes are ordinary runtime values. Constructors use `initialize`.

```tya
class User
  initialize = name ->
    self.name = name

  label = ->
    "user:{self.name}"

user = User("komagata")
print(user.label())
```

Tya supports single class inheritance, `super(args...)`, private members,
abstract classes, final classes, class variables, class methods, and runtime
class inspection.

```tya
class Admin extends User
  initialize = name ->
    super(name)

  label = ->
    "admin:{self.name}"
```

## Interfaces

Interfaces are explicit. They can declare requirements and provide stackable
behavior through default methods, fields, and zero-argument initializer hooks.

```tya
import time as time

interface Named
  name = ->

  label = ->
    self.name()

interface Timestamped
  created_at = nil

  initialize = ->
    self.created_at = time.Time.now()

class Account implements Named, Timestamped
  initialize = name ->
    self.name_value = name
    super()

  name = ->
    self.name_value
```

Classes list implemented interfaces explicitly with `implements`.

## Imports And Packages

Import another `.tya` file from the same directory:

```tya
import greeting

print(greeting.hello("komagata"))
```

```tya
# greeting.tya
hello = name -> "Hello, {name}"
```

Use aliases for namespaces:

```tya
import net/http as http

app = http.Server()
```

Directory packages expose public class and interface names directly when they
are imported without an alias:

```tya
import net/http

app = Server()
```

Project dependencies live in `tya.toml`, are resolved into `tya.lock`, and are
installed with `tya install`. Git and local path dependencies are supported.

## Concurrency

Use `spawn` to start a lightweight task and `await` to receive its result.

```tya
worker = value ->
  value * 2

task = spawn worker(21)
print(await task)
```

`scope` waits for tasks spawned inside it. Channels and sync primitives are in
the standard library.

## Standard Library

The standard library is documented in `docs/STDLIB.md`. Common packages
include `json`, `toml`, `csv`, `url`, `time`, `random`, `math`, `file`, `dir`,
`path`, `process`, `unittest`, `template`, `markdown`, `compress`, `log`, `io`,
`net/ip`, `net/socket`, and `net/http`.

```tya
import net/http as http

resp = http.Client.get("http://example.test/")
print(resp["status"])
```

Built-in functions are documented in `docs/API.md`.
