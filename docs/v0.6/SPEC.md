---
layout: doc
title: Spec
permalink: /v0.6/spec/
---

# Tya v0.6 Specification

This document is the specification for Tya v0.6 after v0.5 minimal classes and
objects.

## Theme

Tya v0.6 is about class-level state and behavior.

v0.5 adds instance fields and instance methods. v0.6 extends the same class
model with class variables and class methods, using the `@@field` syntax that
v0.5 reserved for future class-level members.

## Goals

- Add class variables.
- Add class methods.
- Add instance field defaults in class bodies.
- Keep `@field` and `@@field` visually distinct.
- Keep class-level members explicit and easy to compile to C.
- Leave inheritance and richer object-oriented features for later versions.

## Included in v0.6

v0.6 includes all v0.5 class behavior and adds:

- `@@field` class variable syntax
- instance field defaults declared with `field = value`
- class variables declared in class bodies
- class variable read/write inside instance methods
- class variable read/write inside class methods
- public class variable read/write with `ClassName.field`
- class methods declared with `@@method = args ->`
- public class method calls with `ClassName.method(args...)`
- class methods as module class members with `module_name.ClassName.method(...)`

## Not Included in v0.6

v0.6 does not include:

- inheritance
- `super`
- interfaces
- visibility modifiers
- private fields
- private class variables
- private class methods
- method overloading
- operator overloading
- decorators
- metaclasses
- dictionary member access with `dict.key`
- package manager
- native-backed stdlib

## Instance and Class Fields

`@field` is an instance field.

```tya
class User
  init = name ->
    @name = name
```

`@@field` is a class variable.

```tya
class User
  @@count = 0
```

The distinction is lexical:

- `@name` belongs to the current object.
- `@@count` belongs to the current class.

## Class Variables

Class variables are declared directly in the class body with `@@field = value`.

```tya
class User
  @@count = 0

  init = name ->
    @name = name
    @@count = @@count + 1
```

The initializer expression is evaluated once when the class is defined.

Class variables are shared by all instances of the class.

```tya
User("komagata")
User("tya")

print User.count
```

Class variables are public in v0.6. They can be read or assigned through the
class name.

```tya
User.count = 0
print User.count
```

Reading a missing class variable is an error.

## Instance Field Defaults

Instance field defaults are declared directly in the class body with
`field = value`.

```tya
class Counter
  value = 0

  increment = ->
    @value = @value + 1
```

The default value is copied into each new instance before `init` runs.

```tya
counter = Counter()
print counter.value
```

`init` may overwrite a default field.

```tya
class User
  name = ""
  active = true

  init = name ->
    @name = name
```

Field defaults define instance fields, not class variables. Use `@@field` for
class-level state.

```tya
class User
  name = ""   # instance field default
  @@count = 0 # class variable
```

Field default names use snake_case. A field default and an instance method may
not share the same name in the same class.

## Class Methods

Class methods are declared in the class body with `@@method = args ->`.

```tya
class User
  @@count = 0

  @@build = name ->
    User(name)

  @@count_users = ->
    @@count
```

Call class methods through the class name.

```tya
user = User.build("komagata")
print User.count_users()
```

Class methods do not have an instance receiver. `@field` is invalid inside a
class method. Use `@@field` for class-level state.

```tya
class User
  @@count = 0

  @@reset = ->
    @@count = 0
```

Reading a class method as a first-class value without calling it is not part of
v0.6.

## Instance Methods and Class Variables

Instance methods may read and write class variables with `@@field`.

```tya
class User
  @@count = 0

  init = name ->
    @name = name
    @@count = @@count + 1

  count = ->
    @@count
```

`@field` and `@@field` may appear in the same instance method.

```tya
class User
  @@prefix = "user"

  init = name ->
    @name = name

  label = ->
    "{@@prefix}:{@name}"
```

## Module Classes

A class declared inside a module exposes class variables and class methods
through the module namespace.

```tya
# user.tya
module user
  class User
    @@count = 0

    init = name ->
      @name = name
      @@count = @@count + 1

    @@count_users = ->
      @@count
```

Use the class-level members through the module class.

```tya
import user

user.User("komagata")
print user.User.count
print user.User.count_users()
```

v0.6 does not import module classes directly into the local namespace.

## Member Namespaces

A class has two member namespaces:

- instance members, used through objects
- class members, used through the class name

An instance member and a class member may use the same name because they are
called through different receivers.

```tya
class User
  init = name ->
    @name = name

  name = ->
    @name

  @@name = ->
    "User"

user = User("komagata")
print user.name()
print User.name()
```

Within the class member namespace, a class variable and a class method may not
share the same name.

```tya
class User
  @@name = "User"
  @@name = ->
    "User"
```

The second `@@name` is an error.

Within the instance member namespace, a field default and an instance method
may not share the same name.

```tya
class User
  name = ""
  name = ->
    @name
```

The second `name` is an error.

## Construction and `init`

v0.6 keeps the v0.5 construction rules.

```tya
class User
  init = name ->
    @name = name

user = User("komagata")
```

`init` is still an instance initializer. v0.6 does not add class constructors
or factory constructors. Use a class method when factory-like construction is
needed.

```tya
class User
  @@build = name ->
    User(name)
```

## Dot Access Boundary

Dot access has these specified meanings in v0.6:

- module member access: `module_name.member`
- object field access: `object.field`
- object method calls: `object.method(args...)`
- class variable access: `ClassName.field`
- class method calls: `ClassName.method(args...)`

Dictionaries continue to use bracket access.

```tya
profile = {"name": "komagata"}
print profile["name"]
```

Dictionary member access with `profile.name` is not part of v0.6.

## Naming

Class names use PascalCase.

```tya
class User
class HttpClient
class CsvRow
```

Variables, functions, methods, fields, class variables, modules, files, and
dictionary keys keep using snake_case.

## Diagnostics

v0.6 implementations should report source-oriented errors for:

- non-PascalCase class names
- `@field` outside an instance method
- `@field` inside a class method
- `@@field` outside a class body, instance method, or class method
- duplicate class members in the same class
- duplicate instance members in the same class
- constructor arity mismatch
- missing object fields
- missing object methods
- missing class variables
- missing class methods
- dictionary member access with dot syntax
