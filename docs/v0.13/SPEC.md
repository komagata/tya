---
layout: doc
title: Spec
permalink: /v0.13/spec/
---

# Tya v0.13 Specification

This document is the specification for Tya v0.13 after v0.12 interface
inheritance and conflict diagnostics.

## Theme

Tya v0.13 is about safer method overrides and constructor chains.

Earlier class versions add inheritance, abstract methods, interfaces, and
interface inheritance. v0.13 makes class inheritance safer by letting a method
definition explicitly say that it is intended to override inherited behavior,
and by checking that subclass constructors initialize their parent class
properly.

## Goals

- Add explicit `override` method declarations.
- Add `override` for instance methods and class methods.
- Allow `override` to implement inherited abstract methods.
- Keep override annotations optional in v0.13.
- Detect `override` declarations that do not actually override a parent method.
- Check override arity against the overridden method.
- Require subclass `init` methods to call parent `init` when a parent `init` exists.
- Check constructor `super(...)` placement, count, and arity.

## Included in v0.13

v0.13 includes all v0.12 class and interface behavior and adds:

- `override method = args ->`
- `override @@method = args ->`
- override validation against inherited concrete methods
- override validation against inherited abstract methods
- override arity checks
- optional override annotations
- required parent `init` chaining when a subclass defines `init`
- constructor `super(...)` count checks
- constructor `super(...)` placement checks
- constructor `super(...)` arity checks

## Not Included in v0.13

v0.13 does not include:

- mandatory `override` on every overriding method
- `override` for interface-only requirements
- `final method`
- `final field`
- duplicate method definition errors
- default interface method bodies
- sealed classes
- base classes
- mixins
- traits
- type annotations
- generics
- method overloading
- operator overloading
- decorators
- metaclasses
- dictionary member access with `dict.key`
- package manager
- native-backed stdlib

## Instance Method Override

An instance method may use `override` when it intentionally overrides an
inherited instance method.

```tya
class User
  label = ->
    "user"

class Admin extends User
  override label = ->
    "admin"
```

`Admin.label` is valid because `User` defines an inherited instance method named
`label`.

The overriding method must use the same arity as the overridden method.

```tya
class User
  label = prefix ->
    prefix + " user"

class Admin extends User
  override label = ->
    "admin"
```

`Admin.label` is invalid because the parent method expects 1 argument.

## Missing Override Target

An `override` declaration is an error when no inherited method with that name
exists.

```tya
class User
  label = ->
    "user"

class Admin extends User
  override lable = ->
    "admin"
```

`Admin.lable` is invalid because there is no inherited `lable` method. This
helps catch method-name typos.

`override` checks only parent classes. A method that only satisfies an interface
requirement is not an override.

```tya
interface Named
  name = ->

class User implements Named
  override name = ->
    "user"
```

`User.name` is invalid because `Named.name` is an interface requirement, not an
inherited class method.

## Optional Override Annotations

v0.13 does not require `override` for every overriding method.

```tya
class User
  label = ->
    "user"

class Admin extends User
  label = ->
    "admin"
```

This remains valid. `override` is an opt-in safety annotation in v0.13.

## Abstract Method Override

`override` may implement an inherited abstract instance method.

```tya
abstract class Repository
  abstract find = id ->

class UserRepository extends Repository
  override find = id ->
    "user"
```

The same arity rule applies. The implementation must match the abstract method
arity.

## Class Method Override

Class methods may also use `override`.

```tya
class Model
  @@table_name = ->
    "models"

class User extends Model
  override @@table_name = ->
    "users"
```

`override @@table_name` is valid because `Model` defines an inherited class
method named `@@table_name`.

`override @@method` must override an inherited class method, not an instance
method.

```tya
class Model
  table_name = ->
    "models"

class User extends Model
  override @@table_name = ->
    "users"
```

This is invalid because the parent method is an instance method.

`override method` must override an inherited instance method, not a class method.

## Constructor Chaining

If a subclass defines `init` and its parent class has a public `init`, the
subclass `init` must call `super(...)`.

```tya
class User
  init = name ->
    @name = name

class Admin extends User
  init = name, role ->
    super(name)
    @role = role
```

This is valid because `Admin.init` calls `User.init`.

If the call is missing, the subclass is invalid.

```tya
class Admin extends User
  init = name, role ->
    @name = name
    @role = role
```

`Admin.init` is invalid because it does not call the parent `init`.

If the parent class has no `init`, a subclass `init` does not need `super(...)`.

```tya
class User
  label = ->
    "user"

class Admin extends User
  init = role ->
    @role = role
```

This is valid.

## Constructor `super(...)` Count

A subclass `init` may call parent `init` at most once.

```tya
class Admin extends User
  init = name, role ->
    super(name)
    super(name)
    @role = role
```

This is invalid because `super(...)` is called twice.

If the parent class has no public `init`, calling `super(...)` from `init` is an
error.

```tya
class User
  label = ->
    "user"

class Admin extends User
  init = role ->
    super()
    @role = role
```

This is invalid because `User` has no `init`.

## Constructor `super(...)` Placement

Inside a subclass `init`, `super(...)` must run before the subclass assigns
instance fields.

```tya
class Admin extends User
  init = name, role ->
    @role = role
    super(name)
```

This is invalid because `@role` is assigned before the parent `init` runs.

An explicit `return` before constructor `super(...)` is also invalid.

```tya
class Admin extends User
  init = name, role ->
    return nil
    super(name)
```

This is invalid because the parent `init` would never run.

Local variables may be prepared before `super(...)` if they do not access or
assign instance fields.

```tya
class Admin extends User
  init = name, role ->
    normalized = string.strip(name)
    super(normalized)
    @role = role
```

This is valid.

## Constructor `super(...)` Arity

The constructor `super(...)` call must pass the same number of arguments as the
parent `init` expects.

```tya
class User
  init = first, last ->
    @name = first + " " + last

class Admin extends User
  init = name, role ->
    super(name)
    @role = role
```

This is invalid because `User.init` expects 2 arguments.

## Private Constructors

Private constructors keep the v0.9 rule: a subclass cannot call parent `_init`
with `super(...)`.

```tya
class Token
  _init = value ->
    @value = value

class ApiToken extends Token
  init = value ->
    super(value)
```

`ApiToken.init` is invalid because parent `_init` is private.

If a parent class has `_init` but no public `init`, the subclass is not required
to call `super(...)`, and `super(...)` remains invalid.

## `super(...)` Outside Constructors

v0.13 keeps the existing method `super(...)` behavior for normal methods. The
constructor chaining rules in this document apply specifically to `super(...)`
inside `init`.

```tya
class User
  label = ->
    "user"

class Admin extends User
  override label = ->
    super() + " admin"
```

This remains valid when the normal method `super()` call matches the overridden
method.

## Modules

`override` and constructor chaining checks work inside modules.

```tya
module accounts
  class User
    init = name ->
      @name = name

    label = ->
      @name

  class Admin extends User
    init = name, role ->
      super(name)
      @role = role

    override label = ->
      @name + " admin"
```

## Introspection

v0.13 keeps the v0.8 introspection surface:

- `object.class`
- `object.class_name`
- `ClassName.name`
- `ClassName.parent`

v0.13 does not add introspection for `override` annotations or constructor
chain state.

## Diagnostics

v0.13 implementations should report source-oriented errors for:

- `override` with no inherited class method target
- `override` arity mismatch
- `override @@method` targeting an inherited instance method
- `override method` targeting an inherited class method
- `override` used only to satisfy an interface requirement
- subclass `init` missing `super(...)` when parent public `init` exists
- constructor `super(...)` called more than once
- constructor `super(...)` used when parent public `init` does not exist
- instance field assignment before constructor `super(...)`
- explicit `return` before constructor `super(...)`
- constructor `super(...)` arity mismatch
- constructor `super(...)` targeting parent `_init`

Diagnostics should mention the relevant class, parent class, method, and
constructor names when available.
