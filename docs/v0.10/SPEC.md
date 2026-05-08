# Tya v0.10 Specification

This document is the specification for Tya v0.10 after v0.9 class visibility,
private members, private constructors, and abstract classes.

## Theme

Tya v0.10 is about abstract behavior contracts and final classes.

v0.9 adds abstract classes that cannot be directly constructed. v0.10 adds
abstract methods so abstract classes can require subclasses to implement
specific behavior. It also adds `final class` so a class can explicitly opt out
of subclassing.

## Goals

- Add abstract instance methods.
- Add abstract class methods.
- Require concrete subclasses to implement inherited abstract methods.
- Keep abstract methods body-free.
- Add final classes.
- Keep interfaces and implicit interface conformance for later versions.

## Included in v0.10

v0.10 includes all v0.9 class behavior and adds:

- `abstract method = args ->`
- `abstract @@method = args ->`
- abstract methods only inside abstract classes
- concrete subclass implementation checks
- abstract subclass partial implementation
- abstract method overriding with matching arity
- `final class Name`
- final class inheritance checks

## Not Included in v0.10

v0.10 does not include:

- interface declarations
- `implements`
- implicit interface conformance checks
- abstract fields
- abstract constructors
- final methods
- final fields
- sealed classes
- base classes
- mixins
- method overloading
- operator overloading
- decorators
- metaclasses
- type annotations
- generics
- dictionary member access with `dict.key`
- package manager
- native-backed stdlib

## Abstract Instance Methods

An abstract instance method declares a required instance method without a body.

```tya
abstract class Repository
  abstract find = id ->
  abstract save = record ->
```

Only abstract classes may declare abstract methods.

```tya
class Repository
  abstract find = id ->
```

The `abstract find` declaration is an error because `Repository` is not
abstract.

An abstract method has a name and parameter list. It does not have a method
body.

## Implementing Abstract Instance Methods

A concrete subclass must implement every inherited abstract instance method.

```tya
abstract class Repository
  abstract find = id ->
  abstract save = record ->

class UserRepository extends Repository
  find = id ->
    "user"

  save = record ->
    nil
```

If a concrete subclass omits an abstract method, it is an error.

```tya
class BrokenRepository extends Repository
  find = id ->
    "user"
```

`BrokenRepository` is invalid because `save` is not implemented.

The implementation must use the same arity as the abstract method.

```tya
class BadRepository extends Repository
  find = first, second ->
    "user"

  save = record ->
    nil
```

The `find` implementation is invalid because its arity differs from the
abstract method.

## Abstract Subclasses

An abstract subclass may leave inherited abstract methods unimplemented.

```tya
abstract class ReadOnlyRepository extends Repository
  find = id ->
    "record"
```

`ReadOnlyRepository` is valid because it is abstract. A concrete subclass of
`ReadOnlyRepository` must still implement `save`.

An abstract subclass may also introduce new abstract methods.

```tya
abstract class SearchRepository extends Repository
  abstract search = query ->
```

## Abstract Class Methods

An abstract class method declares a required class method without a body.

```tya
abstract class Model
  abstract @@table_name = ->
```

A concrete subclass must implement inherited abstract class methods.

```tya
class User extends Model
  @@table_name = ->
    "users"
```

The implementation must use the same arity as the abstract class method.

Abstract class methods participate in class-level inheritance like normal class
methods once they are implemented.

## Abstract Methods and `super`

`super(args...)` cannot call an abstract parent method because there is no
method body to execute.

```tya
abstract class User
  abstract label = ->

class Admin extends User
  label = ->
    super()
```

The `super()` call is an error.

## Final Classes

A final class can be constructed normally but cannot be extended.

```tya
final class User
  name = ""
```

Direct construction is valid.

```tya
user = User()
```

Subclassing is invalid.

```tya
class Admin extends User
```

The `Admin` declaration is an error because `User` is final.

## Final and Abstract Modifiers

`abstract` and `final` cannot be used together on the same class.

```tya
abstract final class User
```

The declaration is an error.

`final class` may still contain private members, class variables, class
methods, instance field defaults, and constructors.

```tya
final class User
  _name = ""

  init = name ->
    @_name = name
```

## Modules

Abstract methods and final classes work inside modules.

```tya
module repositories
  abstract class Repository
    abstract find = id ->

  final class UserRepository extends Repository
    find = id ->
      "user"
```

Module users follow the same construction and inheritance rules.

```tya
import repositories

repo = repositories.UserRepository()
```

## Introspection

v0.10 keeps the v0.8 introspection surface:

- `object.class`
- `object.class_name`
- `ClassName.name`
- `ClassName.parent`

v0.10 does not add introspection for abstract methods or final classes.

## Diagnostics

v0.10 implementations should report source-oriented errors for:

- abstract methods declared outside abstract classes
- abstract methods with bodies
- concrete classes missing inherited abstract instance methods
- concrete classes missing inherited abstract class methods
- abstract method implementation arity mismatch
- abstract class method implementation arity mismatch
- `super` targeting an abstract parent method
- extending a final class
- declaring a class as both abstract and final
- dictionary member access with dot syntax
