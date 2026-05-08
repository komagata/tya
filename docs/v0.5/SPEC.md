# Tya v0.5 Specification

This document is the specification for Tya v0.5 after v0.4 testing and
script confidence.

## Theme

Tya v0.5 is about minimal classes and objects.

v0.4 makes user scripts easier to test. v0.5 adds the smallest class surface
that lets scripts group state and behavior without introducing the whole
object-oriented feature set at once.

## Goals

- Add class declarations.
- Add object construction.
- Add instance fields and methods.
- Keep object syntax explicit and easy to compile to C.
- Keep modules, dictionaries, and functions compatible with the existing language surface.
- Leave inheritance and richer object-oriented features for later versions.

## Included in v0.5

v0.5 adds:

- `class Name` declarations
- PascalCase class names
- constructor calls with `Name(args...)`
- an optional `init` method
- `self` inside instance methods
- public instance fields assigned with `self.field = value`
- public instance field reads with `object.field`
- public instance method calls with `object.method(args...)`
- classes as module members

## Not Included in v0.5

v0.5 does not include:

- inheritance
- `super`
- interfaces
- class methods
- class fields
- field defaults in class bodies
- visibility modifiers
- private fields
- method overloading
- operator overloading
- decorators
- metaclasses
- `@field` shorthand
- dictionary member access with `dict.key`
- package manager
- native-backed stdlib

## Class Declaration

A class declaration starts with `class` and a PascalCase class name.
The class body is indentation-based.

```tya
class User
  init = name ->
    self.name = name

  greet = ->
    "Hello, {self.name}"
```

A class body may contain instance method definitions. v0.5 does not allow
arbitrary statements or field defaults directly in the class body.

## Construction and `init`

Calling a class name constructs an object.

```tya
user = User("komagata")
```

If the class defines `init`, the constructor call passes its arguments to
`init`. The constructed object is returned from the constructor call.

```tya
class Point
  init = x, y ->
    self.x = x
    self.y = y

point = Point(10, 20)
```

`init` is an initializer, not a factory method. Its explicit return value, if
any, is ignored.

If a class does not define `init`, it can be constructed with no arguments.

```tya
class Marker
  label = ->
    "marker"

marker = Marker()
```

Passing arguments to a class without `init` is an error.

## Instance Fields

Instance fields are created by assigning through `self`.

```tya
class Counter
  init = ->
    self.value = 0

  increment = ->
    self.value = self.value + 1
```

Fields are public and can be read or assigned through dot access.

```tya
counter = Counter()
counter.increment()
print counter.value
counter.value = 10
```

Reading a missing object field is an error. This keeps field typos visible.

## Instance Methods

Methods are functions defined in the class body.

```tya
class User
  init = name ->
    self.name = name

  rename = name ->
    self.name = name

  greeting = ->
    "Hello, {self.name}"
```

Inside an instance method, `self` refers to the receiver object.

```tya
user = User("komagata")
user.rename("Tya")
print user.greeting()
```

v0.5 specifies method calls with `object.method(args...)`. Reading a method as
a first-class value without calling it is not part of v0.5.

## Modules and Classes

A class declared inside a module is a public module member when its class name
is public.

```tya
# user.tya
module user
  class User
    init = name ->
      self.name = name

    greeting = ->
      "Hello, {self.name}"
```

Use the class through the module namespace.

```tya
import user

komagata = user.User("komagata")
print komagata.greeting()
```

v0.5 does not import module classes directly into the local namespace.

## Dot Access Boundary

Dot access has three specified meanings in v0.5:

- module member access: `module_name.member`
- object field access: `object.field`
- object method calls: `object.method(args...)`

Dictionaries continue to use bracket access.

```tya
profile = {"name": "komagata"}
print profile["name"]
```

Dictionary member access with `profile.name` is not part of v0.5.

## Naming

Class names use PascalCase.

```tya
class User
class HttpClient
class CsvRow
```

Variables, functions, methods, fields, modules, files, and dictionary keys keep
using snake_case.

## Diagnostics

v0.5 implementations should report source-oriented errors for:

- non-PascalCase class names
- `self` outside an instance method
- duplicate methods in the same class
- constructor arity mismatch
- missing object fields
- missing object methods
- dictionary member access with dot syntax
