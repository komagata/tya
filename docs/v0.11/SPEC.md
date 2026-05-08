# Tya v0.11 Specification

This document is the specification for Tya v0.11 after v0.10 abstract methods
and final classes.

## Theme

Tya v0.11 is about explicit interface contracts.

v0.10 lets abstract classes describe required behavior while still belonging to
the class inheritance tree. v0.11 adds `interface` and `implements` so a class
can declare a behavior contract separately from implementation inheritance.

Tya does not adopt Dart-style implicit interfaces. A class never becomes an
interface automatically. A class implements an interface only when it explicitly
uses `implements InterfaceName`.

## Goals

- Add explicit `interface Name` declarations.
- Add `class Name implements InterfaceName`.
- Allow a class to implement multiple interfaces.
- Allow `extends` and `implements` to be used together.
- Check implemented method names and arity.
- Keep interface bodies small and method-only.
- Avoid implicit interface conformance.

## Included in v0.11

v0.11 includes all v0.10 class behavior and adds:

- `interface Name`
- interface instance method requirements
- `class Name implements InterfaceName`
- multiple implemented interfaces
- `class Name extends Parent implements InterfaceName`
- implementation checks for concrete classes
- partial implementation by abstract classes
- interface method arity checks
- interface declarations inside modules

## Not Included in v0.11

v0.11 does not include:

- implicit interfaces generated from classes
- implementing a class as if it were an interface
- class methods in interfaces
- fields in interfaces
- class variables in interfaces
- constructors in interfaces
- interface inheritance
- private interface methods
- default interface method bodies
- interface introspection
- type annotations
- generics
- mixins
- traits
- sealed classes
- base classes
- final methods
- final fields
- method overloading
- operator overloading
- decorators
- metaclasses
- dictionary member access with `dict.key`
- package manager
- native-backed stdlib

## Interface Declarations

An interface declares required instance methods without method bodies.

```tya
interface Reader
  read = ->

interface Writer
  write = value ->
```

Interface method declarations use the same parameter-list syntax as methods.
They do not use `abstract` because every interface method is abstract.

```tya
interface Repository
  find = id ->
  save = record ->
```

Interface declarations do not create constructible values.

```tya
reader = Reader()
```

The `Reader()` construction is an error because `Reader` is an interface, not a
class.

## Interface Bodies

Only instance method requirements are valid inside an interface.

```tya
interface Named
  name = ->
```

Fields are not valid inside an interface.

```tya
interface Named
  name = ""
```

Class variables and class methods are not valid inside an interface.

```tya
interface Model
  @@table_name = ->
```

Private methods are not valid inside an interface.

```tya
interface Reader
  _read = ->
```

Interface methods cannot have bodies.

```tya
interface Reader
  read = ->
    "value"
```

The `read` declaration is an error because interface method declarations are
body-free.

## Implementing an Interface

A class implements an interface with `implements`.

```tya
interface Reader
  read = ->

class FileReader implements Reader
  read = ->
    "data"
```

A concrete class must implement every method required by every interface it
implements.

```tya
class BrokenReader implements Reader
```

`BrokenReader` is invalid because `read` is not implemented.

The implementation must use the same arity as the interface method.

```tya
class BadReader implements Reader
  read = path ->
    "data"
```

The `read` implementation is invalid because `Reader.read` has arity 0.

## Multiple Interfaces

A class may implement multiple interfaces.

```tya
interface Reader
  read = ->

interface Writer
  write = value ->

class File implements Reader, Writer
  read = ->
    "data"

  write = value ->
    nil
```

If two implemented interfaces require the same method with the same arity, one
implementation satisfies both requirements.

```tya
interface Named
  name = ->

interface Labeled
  name = ->

class User implements Named, Labeled
  name = ->
    "komagata"
```

If two implemented interfaces require the same method name with different
arity, the class declaration is an error.

```tya
interface LookupById
  find = id ->

interface LookupByName
  find = first, last ->

class UserRepository implements LookupById, LookupByName
  find = id ->
    "user"
```

`UserRepository` is invalid because one method implementation cannot satisfy
both required `find` arities.

## `extends` with `implements`

`extends` may be combined with `implements`. `extends` comes before
`implements`.

```tya
interface Named
  name = ->

class User
  name = ->
    "user"

class Admin extends User implements Named
```

Inherited instance methods can satisfy interface requirements.

`implements` must name interfaces, not classes.

```tya
class User
  name = ->
    "user"

class Admin implements User
```

The `Admin` declaration is an error because `User` is a class.

## Abstract Classes and Interfaces

An abstract class may implement an interface without implementing all required
methods.

```tya
interface Repository
  find = id ->
  save = record ->

abstract class ReadOnlyRepository implements Repository
  find = id ->
    "record"
```

`ReadOnlyRepository` is valid because it is abstract. A concrete subclass must
implement the remaining requirements.

```tya
class UserRepository extends ReadOnlyRepository
  save = record ->
    nil
```

An abstract class may also restate an interface requirement as an abstract
method.

```tya
abstract class BaseRepository implements Repository
  abstract find = id ->
  abstract save = record ->
```

The abstract method arity must match the interface method arity.

## Final Classes and Interfaces

A final class may implement interfaces.

```tya
interface Named
  name = ->

final class User implements Named
  name = ->
    "user"
```

The class is constructible and satisfies `Named`, but it still cannot be
extended.

## Modules

Interfaces work inside modules.

```tya
module io
  interface Reader
    read = ->

  class FileReader implements Reader
    read = ->
      "data"
```

Module users refer to exported interfaces with normal module member syntax.

```tya
import io

class MemoryReader implements io.Reader
  read = ->
    "memory"
```

## Introspection

v0.11 keeps the v0.8 introspection surface:

- `object.class`
- `object.class_name`
- `ClassName.name`
- `ClassName.parent`

v0.11 does not add interface introspection.

## Diagnostics

v0.11 implementations should report source-oriented errors for:

- constructing an interface
- invalid members inside an interface body
- interface method declarations with bodies
- `implements` targets that are not interfaces
- missing interface method implementations on concrete classes
- interface method implementation arity mismatch
- conflicting interface method arity requirements
- abstract method arity mismatch against an implemented interface
- duplicate interface names in a single `implements` list
- duplicate interface declarations in the same scope

Diagnostics should mention the relevant class, interface, and method names when
available.
