# Tya v0.8 Specification

This document is the specification for Tya v0.8 after v0.7 single inheritance
for instance behavior.

## Theme

Tya v0.8 is about class-level member inheritance.

v0.7 adds inheritance for instance field defaults, instance methods, and
`super(args...)`. v0.8 extends inheritance to class variables and class methods,
following the common CoffeeScript, JavaScript, and Python-style behavior where
class-level members can be read through subclasses and subclass writes create
subclass-owned members.

## Goals

- Inherit class methods through subclasses.
- Inherit class variables through subclasses.
- Keep subclass writes to class variables local to the subclass.
- Make class methods construct the receiving class when they call `self`.
- Add small class introspection for class names and parent classes.
- Keep class-level inheritance predictable without adding metaclasses.
- Leave interfaces and richer object-oriented features for later versions.

## Included in v0.8

v0.8 includes all v0.7 class behavior and adds:

- inherited class variables
- inherited class methods
- class variable shadowing by subclass assignment
- class method overriding
- `self` inside class methods as the receiving class
- `super(args...)` inside overridden class methods
- inherited class-level members through module class paths
- `object.class`
- `object.class_name`
- `ClassName.name`
- `ClassName.parent`

## Not Included in v0.8

v0.8 does not include:

- multiple inheritance
- mixins
- interfaces
- `implements`
- abstract classes
- visibility modifiers
- private fields
- protected fields
- private class variables
- private class methods
- metaclasses
- listing methods or fields
- dynamic method calls
- monkey patching
- `self` inside instance methods
- `super.field`
- `super.method(args...)`
- method overloading
- operator overloading
- decorators
- dictionary member access with `dict.key`
- package manager
- native-backed stdlib

## Class Variable Inheritance

Class variables declared on a parent class can be read through a subclass.

```tya
class User
  @@count = 0

class Admin extends User

print Admin.count
```

If a subclass writes to an inherited class variable, the subclass gets its own
class variable with that name. The parent class variable is not changed.

```tya
class User
  @@count = 0

class Admin extends User

Admin.count = 10

print User.count  # 0
print Admin.count # 10
```

This matches the practical behavior of CoffeeScript, JavaScript, and Python
class properties: inherited reads walk up the class chain, while assignment
creates or updates the receiver class's own member.

## Class Introspection

v0.8 adds small read-only introspection for classes and objects.

`object.class` returns the object's actual class.

```tya
class User
class Admin extends User

admin = Admin()

print admin.class
```

The printed representation of a class is its class name.

`object.class_name` returns the object's actual class name as a string.

```tya
print admin.class_name # "Admin"
```

`ClassName.name` returns the class name as a string.

```tya
print User.name  # "User"
print Admin.name # "Admin"
```

`ClassName.parent` returns the parent class, or `nil` when the class has no
parent.

```tya
print User.parent  # nil
print Admin.parent # User
```

Module class paths work the same way.

```tya
import user

class Admin extends user.User

admin = Admin()

print admin.class
print admin.class_name
print Admin.parent
print user.User.name
```

These introspection members are read-only. Assigning to `object.class`,
`object.class_name`, `ClassName.name`, or `ClassName.parent` is an error.

v0.8 does not add reflection APIs for listing methods, listing fields, dynamic
method calls, or modifying classes at runtime.

## Class Method Inheritance

Class methods declared on a parent class can be called through a subclass.

```tya
class User
  @@build = name ->
    self(name)

class Admin extends User

admin = Admin.build("komagata")
```

When a class method is called through a subclass, `self` inside that class
method is the receiving class. In the example above, `self(name)` constructs an
`Admin`, not a `User`.

Class methods may also read inherited class variables.

```tya
class User
  @@label = "user"

  @@label_name = ->
    @@label

class Admin extends User

print Admin.label_name()
```

If the subclass shadows the class variable, the inherited class method sees the
subclass value when called through the subclass.

```tya
class Admin extends User
  @@label = "admin"

print User.label_name()  # user
print Admin.label_name() # admin
```

## `self` in Class Methods

`self` is valid only inside class methods in v0.8.

Inside a class method, `self` refers to the class that received the method call.

```tya
class User
  @@build = name ->
    self(name)

class Admin extends User

admin = Admin.build("komagata")
```

`self` is not valid inside instance methods. Instance fields continue to use
`@field`.

```tya
class User
  greeting = ->
    self.name
```

The `self.name` expression is invalid in v0.8.

## Class Method Overriding

A subclass may override an inherited class method by declaring a class method
with the same name.

```tya
class User
  @@role = ->
    "user"

class Admin extends User
  @@role = ->
    "admin"
```

The subclass method is used when the method is called through the subclass.

```tya
print User.role()
print Admin.role()
```

An overriding class method must use the same arity as the parent class method.

## `super` in Class Methods

Inside an overridden class method, `super(args...)` calls the parent class
method with the same name.

```tya
class User
  @@role = ->
    "user"

class Admin extends User
  @@role = ->
    "{super()} admin"
```

`super(args...)` must be called explicitly with parentheses when there are no
arguments.

```tya
super()
```

There is no `super.role()` form in v0.8.

## Class Variable Lookup from Methods

`@@field` lookup inside a class method starts at the receiving class.

```tya
class User
  @@count = 0

  @@increment = ->
    @@count = @@count + 1

class Admin extends User

Admin.increment()

print User.count
print Admin.count
```

The `Admin.increment()` call updates `Admin.count`, not `User.count`. If
`Admin` did not already have its own `@@count`, the assignment creates one.

Inside an instance method, `@@field` lookup starts at the instance's class.

```tya
class User
  @@label = "user"

  label = ->
    @@label

class Admin extends User
  @@label = "admin"

admin = Admin()
print admin.label()
```

The result is `admin`.

## Modules and Class-Level Inheritance

Inherited class-level members work through module class paths.

```tya
# user.tya
module user
  class User
    @@build = name ->
      self(name)
```

```tya
import user

class Admin extends user.User

admin = Admin.build("komagata")
```

Classes declared inside a module also inherit class-level members from other
classes in the same module.

```tya
module accounts
  class User
    @@label = "user"

  class Admin extends User

print accounts.Admin.label
```

## Member Namespaces

v0.8 keeps the v0.6 and v0.7 member namespaces:

- instance members, used through objects
- class members, used through the class name

Inheritance walks these namespaces separately.

An instance method and a class method may share the same name. Overriding is
checked only inside the same namespace.

```tya
class User
  name = ->
    @name

  @@name = ->
    "User"

class Admin extends User
  name = ->
    "admin:{@name}"

  @@name = ->
    "Admin"
```

## Dot Access Boundary

Dot access keeps the v0.7 meanings:

- module member access: `module_name.member`
- object field access: `object.field`
- object method calls: `object.method(args...)`
- class variable access: `ClassName.field`
- class method calls: `ClassName.method(args...)`
- object class access: `object.class`
- object class-name access: `object.class_name`
- class name access: `ClassName.name`
- class parent access: `ClassName.parent`

Inherited class variables and class methods use the same `ClassName.member`
syntax.

Dictionaries continue to use bracket access.

```tya
profile = {"name": "komagata"}
print profile["name"]
```

Dictionary member access with `profile.name` is not part of v0.8.

## Diagnostics

v0.8 implementations should report source-oriented errors for:

- non-PascalCase class names
- unknown parent class
- multiple inheritance syntax
- inheritance cycles
- `self` outside a class method
- `self` inside an instance method
- `super` outside `init`, instance methods, or class methods
- `super` in a class method that has no parent class method
- overriding a class method with different arity
- overriding an instance method with different arity
- duplicate instance members in the same class
- duplicate class members in the same class
- missing object fields
- missing object methods
- missing class variables
- missing class methods
- assigning to read-only introspection members
- dictionary member access with dot syntax
