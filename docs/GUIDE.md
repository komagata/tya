---
layout: doc
title: Guide
permalink: /guide/
language_url: /ja/guide/
---

# Tya Guide

This guide is for people trying Tya for the first time. It starts with
installation, then walks through writing a small program, running it, building
an executable, and using the basic language features.

For the exact language contract, read the [specification](/spec/). This guide
is practical: copy the examples, run the commands, and change the code as you
go.

## Install

Download the latest release for your platform from GitHub:

```sh
curl -fsSL https://tya-lang.org/install.sh | sh
```

Or install manually from the release page:

```text
https://github.com/komagata/tya/releases/latest
```

Check that the command is available:

```sh
tya version
```

Tya builds native executables through its bundled toolchain support. If native
builds fail on your machine, run:

```sh
tya doctor
```

## Create a Program

Create a new directory and a file named `hello.tya`.

```sh
mkdir hello-tya
cd hello-tya
```

```tya
name = "Tya"
print("Hello, {name}")
```

Run it:

```sh
tya run hello.tya
```

Expected output:

```text
Hello, Tya
```

Tya source files use `.tya`. Lowercase files such as `hello.tya` and
`main.tya` can be used as script entry points.

## Check and Format

Use `tya check` before running or building when you want a fast validity check.

```sh
tya check hello.tya
```

Use `tya format` to print canonical source:

```sh
tya format hello.tya
```

Use `-w` to rewrite the file:

```sh
tya format -w hello.tya
```

Formatting is part of Tya's language design. A valid program has one canonical
source representation.

## Build an Executable

Build a native executable:

```sh
tya build hello.tya -o hello
```

Run the built program:

```sh
./hello
```

On Windows, use an `.exe` output name:

```sh
tya build hello.tya -o hello.exe
hello.exe
```

`tya run` is for quick local execution. `tya build` creates a reusable
executable.

## A Small Script

Replace `hello.tya` with a slightly larger program.

```tya
greet = name ->
  "Hello, {name}"

names = ["Ada", "Matz", "Tya"]

for name in names
  print(greet(name))
```

Run it:

```sh
tya run hello.tya
```

This example shows three core ideas:

- functions are values and use `->`;
- arrays use `[...]`;
- indentation defines blocks.

## Values

Tya is dynamically typed. Values carry runtime kinds.

```tya
name = "Tya"
count = 3
price = 12.5
ready = true
missing = nil
data = b"abc"
```

Strings use interpolation:

```tya
print("count = {count}")
```

Arrays are mutable:

```tya
items = [1, 2]
items.push(3)
print(items[0])
print(items.len())
```

Dictionaries use string keys:

```tya
user = { name: "Ada", admin: true }
print(user["name"])
user["city"] = "London"
```

Primitive values have methods:

```tya
print(" tya ".trim().upper())
print(42.to_s())
print([1, 2, 3].len())
```

## Names

Use `snake_case` for values, functions, methods, files, import paths, and
dictionary keys.

Use `PascalCase` for classes and interfaces.

Use `SCREAMING_SNAKE_CASE` for constants.

```tya
user_name = "Ada"
MAX_RETRIES = 3

class UserProfile
  initialize = name ->
    self.name = name
```

Leading `_` does not make a binding private. For class members, use `private`.

## Control Flow

Use `if`, `elseif`, and `else`.

```tya
age = 20

if age >= 20
  print("adult")
elseif age >= 13
  print("teen")
else
  print("young")
```

Use `and`, `or`, and `not` for boolean logic.

```tya
if ready and not disabled
  print("ready")
```

Use `while` for repeated work:

```tya
count = 0
while count < 3
  print(count)
  count = count + 1
```

Use `for` to iterate arrays:

```tya
items = ["a", "b", "c"]

for item in items
  print(item)

for item, index in items
  print("{index}: {item}")
```

Use `break` to leave the nearest loop and `continue` to skip to the next
iteration.

## Functions

Functions use `->`. Calls always use parentheses.

```tya
add = a, b -> a + b
print(add(2, 3))
```

Use an indented body for multiple statements. The final expression is returned.

```tya
double = value ->
  result = value * 2
  result

print(double(21))
```

Use `return` for early return:

```tya
label = name ->
  if name == ""
    return "anonymous"
  name
```

Functions can be passed to other functions:

```tya
items = [1, 2, 3]
print(items.map(item -> item * 2))
```

## Errors

Use `error(...)` to create an error value. Use `raise`, `try`, and `catch` to
handle failures.

```tya
require_name = name ->
  if name == ""
    raise error("name is required")
  name

try
  print(require_name(""))
catch err
  message = err["message"]
  print("error: {message}")
```

Use `try/finally` when cleanup must run.

```tya
try
  print("work")
finally
  print("cleanup")
```

## Split Code into Files

Create `greeting.tya`:

```tya
hello = name ->
  "Hello, {name}"
```

Create `main.tya`:

```tya
import greeting

print(greeting.hello("Tya"))
```

Run the entry file:

```sh
tya run main.tya
```

Use an alias when a package name is long:

```tya
import net/http as http

resp = http.Client().get("https://example.com/")
print(resp["status"])
```

## Classes

Classes are runtime values. `initialize` is the constructor hook.

```tya
class User
  initialize = name ->
    self.name = name

  label = ->
    "user:{self.name}"

user = User("Ada")
print(user.label())
```

Use `private` for private class members.

```tya
class Counter
  private count = 0

  increment = ->
    self.count = self.count + 1
    self.count
```

Class files use snake_case filenames such as `user.tya` while declaring PascalCase classes such as `class User`. They are library files, not script entry points.

## Standard Library

Standard-library packages are imported like user files.

```tya
import json as json
import time as time

json_text = json.Json().dump({ name: "Tya" })
print(json_text)

now = time.Time().now()
print(now.format("rfc3339"))
```

Common packages include `json`, `toml`, `csv`, `url`, `time`, `random`,
`math`, `file`, `path`, `unittest`, `template`, `markdown`, `compress`, `log`,
`io`, `net/ip`, `net/socket`, `net/http`, `channel`, and `sync`.

Generated API documentation can be produced from source comments:

```sh
tya doc --json stdlib
```

## Tests

Tya test files end with `_test.tya`.

Create `hello_test.tya`:

```tya
import unittest

class AdditionTest extends TestCase
  test_addition_works = ->
    self.assert_equal(4, 2 + 2, "addition")
```

Run tests:

```sh
tya test
```

Show coverage when your project uses coverage-supported tests:

```sh
tya test --cover
tya cover
```

## Lint and Docs

Run lint:

```sh
tya lint
tya lint --fix .
```

Generate docs from source comments:

```sh
tya doc
tya doc --html ./site src
```

## Packages

Projects use `tya.toml` for metadata and dependencies. Resolved dependencies
are written to `tya.lock`.

```sh
tya install
tya add https://github.com/komagata/tya-sqlite --tag v0.1.0
tya update
tya outdated
```

Git and local path dependencies are supported. Tya does not currently use a
central package registry.

## Cross Compilation

Build the native target explicitly:

```sh
tya build --target native main.tya -o app
```

Build WebAssembly targets:

```sh
tya build --target wasm32-wasi examples/wasm/hello.tya -o hello.wasm
tya build --target wasm32-browser examples/wasm/hello.tya -o hello.wasm
tya doctor wasm
```

`tya run` is native-only.

## Everyday Command List

```sh
tya version
tya run main.tya
tya check main.tya
tya format -w .
tya build main.tya -o app
tya test
tya lint
tya doc
tya doctor
```

After this guide, read the [specification](/spec/) when you need exact syntax,
runtime behavior, package rules, or standard-library boundaries.
