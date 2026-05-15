# Tya Guide

Tya is an indentation-based language that compiles to C. You write `.tya`
files and use one command, `tya`, to run, build, format, test, document, and
manage packages.

For exact language rules, read `docs/SPEC.md`. This guide is the practical
starting point.

## First Program

Create `hello.tya`.

```tya
print("Hello, Tya")
```

Run it:

```sh
tya run hello.tya
```

Build an executable:

```sh
tya build hello.tya -o hello
./hello
```

Useful everyday commands:

```sh
tya check hello.tya
tya format -w hello.tya
tya test
tya lint
tya version
```

## Files

Lowercase files are scripts. They can be run and imported.

```text
main.tya
greeting.tya
```

PascalCase files are class files. They are library files, not entry scripts.

```text
User.tya
HttpClient.tya
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

Strings use interpolation:

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

Primitive values have methods:

```tya
print(" tya ".trim().upper())
print([1, 2, 3].len())
print(user.keys())
print(value.to_s())
print(value.class)
```

## Names

Use `snake_case` for values, functions, methods, files, imports, and dictionary
keys. Use `PascalCase` for classes and interfaces. Use
`SCREAMING_SNAKE_CASE` for constants.

```tya
user_name = "komagata"
MAX_COUNT = 10

class UserProfile
  initialize = name ->
    self.name = name
```

Leading `_` does not make a binding private. For class members, use
`private`.

## Control Flow

```tya
if age >= 20
  print("adult")
elseif age >= 13
  print("teen")
else
  print("young")
```

Use `and`, `or`, and `not`.

```tya
if ready and not disabled
  print("ok")
```

Loops:

```tya
count = 0
while count < 3
  print(count)
  count = count + 1

items = ["a", "b"]
for item in items
  print(item)

for item, index in items
  print("{index}: {item}")
```

Dictionaries can be iterated as entries:

```tya
user = { name: "komagata", age: 20 }

for entry in user
  print("{entry["key"]}: {entry["value"]}")
```

`break` exits the nearest loop. `continue` skips to the next iteration.

## Functions

Functions use `->`. Calls always use parentheses.

```tya
greet = name -> "Hello, {name}"

print(greet("Tya"))
```

Use an indented body for multiple statements. The final expression is returned.

```tya
double = value ->
  result = value * 2
  result
```

Use `return` for early return or multiple return values.

```tya
parse_user = text ->
  if text == ""
    return nil, error("empty user")
  return { name: text }, nil
```

Functions are values, so they can be passed to other functions.

```tya
items = [1, 2, 3]
print(items.map(item -> item * 2))
```

Closures can read values from the outer function.

```tya
make_adder = base ->
  value -> base + value

add_two = make_adder(2)
print(add_two(3))
```

## Errors

Use `raise`, `try`, and `catch`.

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

`try` can also be an expression inside a function.

```tya
load_user = path ->
  try
    text = read_file(path)
    { name: text.trim() }
  catch err
    { name: "guest" }
```

## Classes

Classes are runtime values. `initialize` is the constructor hook.

```tya
class User
  initialize = name ->
    self.name = name

  label = ->
    "user:{self.name}"

user = User("komagata")
print(user.label())
```

Use `private` for private class members.

```tya
class User
  private id = 0

  private normalize = ->
    self.id.to_s()
```

Inheritance uses `extends`; parent behavior is called with `super(...)`.

```tya
class Admin extends User
  initialize = name ->
    super(name)

  label = ->
    "admin:{self.name}"
```

## Interfaces

Interfaces are explicit contracts. Classes opt in with `implements`.

```tya
interface Named
  name = ->

  label = ->
    self.name()

class Account implements Named
  initialize = name ->
    self.name_value = name

  name = ->
    self.name_value
```

Interfaces can also provide default methods, fields, and zero-argument
initializer hooks. Standard interfaces include `Comparable`, `Equatable`,
`Stringable`, `Iterator`, `Iterable`, `Sequence`, `Readable`, `Writable`,
`Closable`, `Flushable`, and `Serializable`.

## Imports

Import a sibling file:

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

server = http.Server()
```

Directory packages expose public class and interface names directly when
imported without an alias:

```tya
import net/http

server = Server()
```

## Packages

Projects use `tya.toml` for dependencies and `tya.lock` for resolved versions.

```sh
tya install
tya add https://github.com/komagata/tya-sqlite --tag v0.1.0
tya update
tya outdated
```

Git and local path dependencies are supported. There is no central package
registry.

## Testing And Linting

Run tests:

```sh
tya test
tya test tests
tya test --cover
```

Show coverage:

```sh
tya cover
tya cover --format=json
```

Run lint:

```sh
tya lint
tya lint --fix src
```

Lint warnings are project-policy warnings. Compile-time validity is checked by
`tya check`.

## Formatting

`tya format` prints canonical Tya source. `-w` rewrites files.

```sh
tya format src/main.tya
tya format -w src tests
```

Formatting is part of the language: there is one canonical source
representation for each well-formed program.

## Concurrency

Use `spawn` to start a task and `await` to receive its result.

```tya
worker = value ->
  value * 2

task = spawn worker(21)
print(await task)
```

`scope` waits for tasks spawned inside it before leaving the block.

```tya
scope
  task = spawn worker(21)
  print(await task)
```

Channels and sync primitives are in the standard library.

## Cross Compilation

`tya build` can target native and WebAssembly outputs.

```sh
tya build --target native src/main.tya -o app
tya build --target wasm32-wasi examples/wasm/hello.tya -o hello.wasm
tya build --target wasm32-browser examples/wasm/hello.tya -o hello.wasm
tya doctor wasm
```

`tya run` is native-only.

## Standard Library

Common packages include `json`, `toml`, `csv`, `url`, `time`, `random`,
`math`, `file`, `path`, `unittest`, `template`, `markdown`, `compress`, `log`,
`io`, `net/ip`, `net/socket`, `net/http`, `channel`, `sync`, and `task`.

```tya
import net/http as http

resp = http.Client.get("http://example.test/")
print(resp["status"])
```

Built-in functions and standard-library packages are specified in
`docs/SPEC.md`.
