---
layout: doc
title: Spec
permalink: /v0.9/spec/
---

# Tya v0.9 Specification

This document is the specification for Tya v0.9 after v0.8 class-level
inheritance and class introspection.

## Theme

Tya v0.9 is about class visibility and encapsulation.

v0.5 through v0.8 make classes useful for state, behavior, inheritance, and
basic introspection. v0.9 adds private class members so a class can keep helper
state and helper behavior out of its public API.

## Goals

- Add private instance fields.
- Add private instance methods.
- Add private class variables.
- Add private class methods.
- Add private constructors with `_init`.
- Add abstract classes.
- Keep privacy based on the existing leading `_` naming convention.
- Keep public class behavior unchanged.
- Leave protected visibility and richer access-control features for later versions.

## Included in v0.9

v0.9 includes all v0.8 class behavior and adds:

- private instance fields with `@_field`
- private instance methods with `_method = args ->`
- private class variables with `@@_field`
- private class methods with `@@_method = args ->`
- private member access from methods declared in the same class
- private class member access from class methods declared in the same class
- private constructors with `_init = args ->`
- private construction from methods declared in the same class
- `abstract class Name`
- direct construction checks for abstract classes
- privacy checks for external object access
- privacy checks for external class access
- privacy checks across inheritance boundaries
- privacy checks across module boundaries

## Not Included in v0.9

v0.9 does not include:

- protected fields
- protected methods
- `public` / `private` keywords
- per-member annotations
- friend classes
- package-private visibility
- interface declarations
- abstract methods
- mixins
- method overloading
- operator overloading
- decorators
- metaclasses
- listing methods or fields
- dynamic method calls
- monkey patching
- dictionary member access with `dict.key`
- package manager
- native-backed stdlib

## Private Instance Fields

An instance field whose name starts with `_` is private.

```tya
class User
  init = name ->
    @_name = name

  name = ->
    @_name
```

Private instance fields can be read and written only by methods declared in the
same class.

```tya
user = User("komagata")

print user.name()
print user._name
```

The `user._name` access is an error.

Instance field defaults can also be private.

```tya
class User
  _active = true

  active = ->
    @_active
```

The field is still private even though the default is declared in the class
body without `@`.

## Private Instance Methods

An instance method whose name starts with `_` is private.

```tya
class User
  init = name ->
    @name = _normalize_name(name)

  _normalize_name = name ->
    name
```

Private instance methods can be called only by methods declared in the same
class.

```tya
user = User("komagata")
print user._normalize_name("tya")
```

The external method call is an error.

## Private Class Variables

A class variable whose name starts with `_` is private.

```tya
class User
  @@_count = 0

  @@count = ->
    @@_count
```

Private class variables can be read and written only by methods declared in the
same class.

```tya
print User.count()
print User._count
```

The `User._count` access is an error.

## Private Class Methods

A class method whose name starts with `_` is private.

```tya
class User
  @@build = name ->
    @@_new_user(name)

  @@_new_user = name ->
    User(name)
```

Private class methods can be called only by methods declared in the same class.

```tya
user = User.build("komagata")
user = User._new_user("komagata")
```

The external class method call is an error.

## Private Constructors

`_init` is a private constructor.

```tya
class User
  _init = name ->
    @_name = name

  @@build = name ->
    User(name)
```

A class with `_init` cannot be constructed from outside the class.

```tya
user = User("komagata")
```

The external constructor call is an error.

Methods declared in the same class may construct the class.

```tya
user = User.build("komagata")
```

`init` and `_init` may not both be declared in the same class.

```tya
class User
  init = name ->
    @name = name

  _init = name ->
    @_name = name
```

The second constructor declaration is an error.

Subclasses cannot call a parent `_init` with `super(args...)`.

```tya
class User
  _init = name ->
    @_name = name

class Admin extends User
  init = name ->
    super(name)
```

The `super(name)` call is an error because the parent constructor is private.

## Abstract Classes

An abstract class cannot be constructed directly.

```tya
abstract class Repository
  init = name ->
    @name = name
```

Direct construction is an error.

```tya
repo = Repository("users")
```

Subclasses of an abstract class can be constructed when they are not abstract.

```tya
class UserRepository extends Repository

repo = UserRepository("users")
```

Abstract classes may define field defaults, instance methods, class variables,
class methods, private members, and constructors.

v0.9 does not include abstract methods. An abstract class is only a class that
cannot be directly constructed.

## Inheritance and Private Members

Private members are private to the class that declares them.

A subclass does not access a parent's private instance fields or private
instance methods directly.

```tya
class User
  init = name ->
    @_name = name

class Admin extends User
  name = ->
    @_name
```

The `@_name` access in `Admin` is an error because `_name` is private to
`User`.

A subclass does not access a parent's private class variables or private class
methods directly.

```tya
class User
  @@_count = 0

class Admin extends User
  @@count = ->
    @@_count
```

The `@@_count` access in `Admin` is an error because `_count` is private to
`User`.

Private members are not inherited as callable public API.

```tya
class User
  _normalize_name = name ->
    name

class Admin extends User

admin = Admin()
print admin._normalize_name("tya")
```

The method call is an error.

## Overriding and Private Names

A subclass may declare a private member with the same name as a parent private
member. It is a separate member, not an override.

```tya
class User
  _label = ->
    "user"

class Admin extends User
  _label = ->
    "admin"
```

Public methods may still override public parent methods.

```tya
class User
  label = ->
    "user"

class Admin extends User
  label = ->
    "admin"
```

## Private Members and `super`

`super(args...)` can call only parent public methods.

```tya
class User
  _label = ->
    "user"

class Admin extends User
  label = ->
    super()
```

The `super()` call is an error because there is no public parent `label`
method.

There is no syntax for calling a parent private method.

`super(args...)` also cannot call a parent `_init`.

## Module Boundaries

Private class members stay private even when the class is exported by a module.

```tya
# user.tya
module user
  class User
    _normalize_name = name ->
      name

    @@_count = 0
```

```tya
import user

u = user.User("komagata")
print u._normalize_name("tya")
print user.User._count
```

Both accesses are errors.

## Introspection

v0.9 keeps the small v0.8 introspection surface:

- `object.class`
- `object.class_name`
- `ClassName.name`
- `ClassName.parent`

These APIs do not expose private member lists because v0.9 does not include
method or field listing.

## Naming

Private class member names use a leading `_` after the class-member prefix.

```tya
@_name
_normalize_name = name ->
@@_count = 0
@@_build = name ->
```

Private constructors use `_init`.

```tya
_init = name ->
```

Public class member names keep using snake_case without a leading `_`.

```tya
@name
normalize_name = name ->
@@count = 0
@@build = name ->
```

## Diagnostics

v0.9 implementations should report source-oriented errors for:

- external access to private instance fields
- external calls to private instance methods
- external access to private class variables
- external calls to private class methods
- subclass direct access to parent private instance fields
- subclass direct calls to parent private instance methods
- subclass direct access to parent private class variables
- subclass direct calls to parent private class methods
- `super` targeting a private parent method
- `super` targeting a parent `_init`
- external construction of a class with `_init`
- duplicate `init` and `_init` constructors
- direct construction of an abstract class
- duplicate private members in the same class namespace
- duplicate public members in the same class namespace
- dictionary member access with dot syntax
