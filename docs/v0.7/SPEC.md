# Tya v0.7 Specification

This document is the specification for Tya v0.7 after v0.6 class-level members
and instance field defaults.

## Theme

Tya v0.7 is about single inheritance for instance behavior.

v0.5 adds minimal classes and objects. v0.6 adds class variables, class methods,
and instance field defaults. v0.7 lets one class reuse and specialize another
class's instance fields, initializer, and instance methods.

## Goals

- Add single inheritance with `extends`.
- Add `super(args...)` for parent `init` and overridden instance methods.
- Allow instance method overriding.
- Inherit instance field defaults.
- Keep class-level member inheritance out of v0.7.
- Leave interfaces and richer object-oriented features for later versions.

## Included in v0.7

v0.7 includes all v0.6 class behavior and adds:

- `class Child extends Parent`
- single inheritance only
- inherited instance field defaults
- inherited instance methods
- instance method overriding
- `super(args...)` inside `init`
- `super(args...)` inside overridden instance methods
- constructor behavior for subclasses
- module class inheritance with `module_name.Parent`

## Not Included in v0.7

v0.7 does not include:

- multiple inheritance
- mixins
- interfaces
- `implements`
- abstract classes
- visibility modifiers
- private fields
- protected fields
- class variable inheritance
- class method inheritance
- `super` inside class methods
- `super.field`
- `super.method(args...)`
- method overloading
- operator overloading
- decorators
- metaclasses
- dictionary member access with `dict.key`
- package manager
- native-backed stdlib

## Extending a Class

A class may extend one parent class.

```tya
class User
  name = ""

  init = name ->
    @name = name

  greeting = ->
    "Hello, {@name}"

class Admin extends User
  role = "admin"
```

`Admin` is a subclass of `User`. It inherits `User`'s instance field defaults
and instance methods.

```tya
admin = Admin("komagata")
print admin.name
print admin.role
print admin.greeting()
```

Multiple inheritance is invalid.

```tya
class Bad extends User, Account
```

## Constructor Behavior

If a subclass does not define `init`, its constructor uses the nearest inherited
`init`.

```tya
class User
  init = name ->
    @name = name

class Admin extends User

admin = Admin("komagata")
print admin.name
```

If a subclass defines `init`, it is responsible for calling the parent `init`
with `super(args...)` when the parent has one.

```tya
class Admin extends User
  init = name, role ->
    super(name)
    @role = role
```

There is no implicit argument forwarding from a subclass `init` to a parent
`init`.

If a subclass defines `init` and the parent class defines `init`, omitting
`super(args...)` is an error.

```tya
class Admin extends User
  init = name ->
    @name = name
```

The `Admin` initializer is invalid because `User.init` is not called.

## Field Defaults and Inheritance

Instance field defaults are applied from parent to child before `init` runs.

```tya
class User
  active = true

class Admin extends User
  role = "admin"

admin = Admin()
print admin.active
print admin.role
```

A subclass may override a parent field default by declaring the same field name.

```tya
class User
  role = "user"

class Admin extends User
  role = "admin"
```

The subclass default wins for instances of the subclass.

## Method Inheritance

Instance methods are inherited by subclasses.

```tya
class User
  greeting = ->
    "Hello"

class Admin extends User

admin = Admin()
print admin.greeting()
```

A subclass may override an inherited instance method by declaring a method with
the same name.

```tya
class User
  greeting = ->
    "Hello"

class Admin extends User
  greeting = ->
    "Admin"
```

The subclass method is used for subclass instances.

## `super` in Instance Methods

Inside an overridden instance method, `super(args...)` calls the parent method
with the same name.

```tya
class User
  greeting = ->
    "Hello, {@name}"

class Admin extends User
  greeting = ->
    "{super()} (admin)"
```

`super(args...)` must be called explicitly with parentheses when there are no
arguments.

```tya
super()
```

There is no `super.greeting()` form in v0.7.

## Override Rules

An overriding instance method must use the same arity as the parent method.

```tya
class User
  rename = name ->
    @name = name

class Admin extends User
  rename = first_name, last_name ->
    @name = first_name + " " + last_name
```

The `Admin.rename` override is invalid because the arity differs from
`User.rename`.

Return values are not checked beyond existing runtime behavior because v0.7 has
no type annotations.

## Class-Level Members and Inheritance

v0.7 inheritance applies to instance behavior only.

Class variables and class methods are not inherited in v0.7.

```tya
class User
  @@count = 0

  @@count_users = ->
    @@count

class Admin extends User

print User.count
print User.count_users()
```

`Admin.count` and `Admin.count_users()` are not part of v0.7 unless `Admin`
declares its own class variable or class method.

This keeps `@@field` simple and avoids shared class-variable behavior across
inheritance chains.

## Modules and Inheritance

A class may extend a class from an imported module.

```tya
# user.tya
module user
  class User
    init = name ->
      @name = name

    greeting = ->
      "Hello, {@name}"
```

```tya
import user

class Admin extends user.User
  greeting = ->
    "{super()} (admin)"
```

Classes declared inside a module can also extend another class declared in the
same module.

```tya
module accounts
  class User
    name = ""

  class Admin extends User
    role = "admin"
```

v0.7 does not import module classes directly into the local namespace.

## Dot Access Boundary

Dot access keeps the v0.6 meanings:

- module member access: `module_name.member`
- object field access: `object.field`
- object method calls: `object.method(args...)`
- class variable access: `ClassName.field`
- class method calls: `ClassName.method(args...)`

v0.7 also allows a module class path in an `extends` clause:

```tya
class Admin extends user.User
```

Dictionaries continue to use bracket access.

```tya
profile = {"name": "komagata"}
print profile["name"]
```

Dictionary member access with `profile.name` is not part of v0.7.

## Naming

Class names use PascalCase.

```tya
class User
class AdminUser
class HttpClient
```

Variables, functions, methods, fields, class variables, modules, files, and
dictionary keys keep using snake_case.

## Diagnostics

v0.7 implementations should report source-oriented errors for:

- non-PascalCase class names
- unknown parent class
- multiple inheritance syntax
- inheritance cycles
- `super` outside `init` or an instance method
- `super` inside a class method
- `super` in a method that has no parent method
- subclass `init` missing a required parent `init` call
- overriding an instance method with different arity
- duplicate instance members in the same class
- duplicate class members in the same class
- missing object fields
- missing object methods
- missing class variables
- missing class methods
- dictionary member access with dot syntax
