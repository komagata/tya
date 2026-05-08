# Tya v0.12 Specification

This document is the specification for Tya v0.12 after v0.11 explicit
interfaces and `implements`.

## Theme

Tya v0.12 is about interface inheritance and precise conflict diagnostics.

v0.11 adds explicit interfaces as standalone contracts. v0.12 lets interfaces
extend other interfaces so contracts can be composed without repeating method
requirements. Because Tya does not support method overloading, v0.12 also
defines how conflicting interface requirements are rejected and reported.

## Goals

- Add `interface Child extends Parent`.
- Allow an interface to extend multiple interfaces.
- Require classes that implement a child interface to satisfy inherited requirements.
- Reject interface inheritance cycles.
- Reject method requirement conflicts with different arity.
- Require conflict diagnostics to name the relevant interfaces, method, and arities.
- Keep interface inheritance separate from class inheritance.

## Included in v0.12

v0.12 includes all v0.11 interface behavior and adds:

- `interface Child extends Parent`
- `interface Combined extends First, Second`
- inherited interface method requirements
- transitive interface inheritance
- implementation checks through inherited interface requirements
- interface inheritance cycle checks
- conflict diagnostics for incompatible interface method requirements
- module-qualified interface inheritance

## Not Included in v0.12

v0.12 does not include:

- implicit interfaces generated from classes
- implementing a class as if it were an interface
- classes extending interfaces
- interfaces extending classes
- default interface method bodies
- interface fields
- interface class methods
- private interface methods
- interface introspection
- sealed classes
- base classes
- mixins
- traits
- final methods
- final fields
- type annotations
- generics
- method overloading
- operator overloading
- decorators
- metaclasses
- dictionary member access with `dict.key`
- package manager
- native-backed stdlib

## Interface Inheritance

An interface may extend another interface.

```tya
interface Reader
  read = ->

interface SeekableReader extends Reader
  seek = offset ->
```

`SeekableReader` requires both `read` and `seek`.

A class that implements `SeekableReader` must implement inherited requirements
from `Reader` as well as requirements declared directly in `SeekableReader`.

```tya
class File implements SeekableReader
  read = ->
    "data"

  seek = offset ->
    nil
```

This is valid because `File` implements both required methods.

```tya
class BrokenFile implements SeekableReader
  seek = offset ->
    nil
```

`BrokenFile` is invalid because the inherited `read` requirement is missing.

## Multiple Interface Inheritance

An interface may extend multiple interfaces.

```tya
interface Reader
  read = ->

interface Writer
  write = value ->

interface ReadWriter extends Reader, Writer
```

`ReadWriter` requires both `read` and `write`.

An interface may add its own requirements while extending other interfaces.

```tya
interface Closable
  close = ->

interface FileHandle extends Reader, Writer, Closable
  path = ->
```

`FileHandle` requires `read`, `write`, `close`, and `path`.

## Transitive Inheritance

Interface inheritance is transitive.

```tya
interface Named
  name = ->

interface Displayable extends Named
  display = ->

interface MenuItem extends Displayable
  select = ->
```

`MenuItem` requires `name`, `display`, and `select`.

## Compatible Duplicate Requirements

If inherited interfaces require the same method with the same arity, the
requirements are compatible.

```tya
interface Named
  name = ->

interface Labeled
  name = ->

interface MenuItem extends Named, Labeled
```

`MenuItem` has one `name` requirement.

```tya
class Item implements MenuItem
  name = ->
    "File"
```

One implementation satisfies both inherited requirements.

## Conflicting Requirements

If inherited interfaces require the same method name with different arity, the
interface declaration is an error.

```tya
interface LookupById
  find = id ->

interface LookupByName
  find = first, last ->

interface Searchable extends LookupById, LookupByName
```

`Searchable` is invalid because Tya does not support method overloading. A
single `find` method cannot satisfy both arities.

Conflict diagnostics should identify the interface being declared, the method
name, both source interfaces, and both arities.

For example:

```text
interface Searchable has conflicting method requirement find:
LookupById.find expects 1 argument, LookupByName.find expects 2 arguments
```

The exact wording may differ, but the diagnostic must include:

- the child interface name
- the conflicting method name
- the parent interface names
- the conflicting arities

## Direct and Inherited Conflicts

A method declared directly in a child interface must also be compatible with
inherited requirements.

```tya
interface Lookup
  find = id ->

interface BadLookup extends Lookup
  find = first, last ->
```

`BadLookup` is invalid because its direct `find` requirement conflicts with the
inherited `Lookup.find` requirement.

If the direct requirement uses the same arity, it is valid and represents the
same requirement.

```tya
interface Lookup
  find = id ->

interface CachedLookup extends Lookup
  find = key ->
```

`CachedLookup` is valid because both `find` requirements have arity 1.
Parameter names are local to each declaration and are not part of
compatibility.

## Inheritance Cycles

Interface inheritance cannot form cycles.

```tya
interface A extends B

interface B extends A
```

This is an error.

Longer cycles are also errors.

```tya
interface A extends B
interface B extends C
interface C extends A
```

Cycle diagnostics should mention the interfaces involved in the cycle when
available.

## Interface and Class Boundaries

Interfaces may extend only interfaces.

```tya
class Base
  name = ->
    "base"

interface Named extends Base
```

`Named` is invalid because `Base` is a class.

Classes may extend only classes.

```tya
interface Named
  name = ->

class User extends Named
```

`User` is invalid because `Named` is an interface.

Classes continue to use `implements` for interfaces.

```tya
class User implements Named
  name = ->
    "user"
```

## Abstract Classes and Inherited Interfaces

An abstract class may implement an interface that inherits requirements without
implementing every requirement.

```tya
interface Reader
  read = ->

interface SeekableReader extends Reader
  seek = offset ->

abstract class AbstractFile implements SeekableReader
  seek = offset ->
    nil
```

`AbstractFile` is valid because it is abstract. A concrete subclass must still
implement `read`.

```tya
class File extends AbstractFile
  read = ->
    "data"
```

## Modules

Interface inheritance works with module-qualified interface names.

```tya
module io
  interface Reader
    read = ->

interface FileReader extends io.Reader
  path = ->
```

A class implementing `FileReader` must satisfy both `io.Reader.read` and
`FileReader.path`.

```tya
class MemoryFile implements FileReader
  read = ->
    "data"

  path = ->
    "memory"
```

## Introspection

v0.12 keeps the v0.8 introspection surface:

- `object.class`
- `object.class_name`
- `ClassName.name`
- `ClassName.parent`

v0.12 does not add interface introspection.

## Diagnostics

v0.12 implementations should report source-oriented errors for:

- interfaces extending missing names
- interfaces extending classes
- classes extending interfaces
- interface inheritance cycles
- conflicting inherited interface method requirements
- direct interface method requirements that conflict with inherited requirements
- concrete classes missing inherited interface method requirements
- concrete class implementation arity mismatch against inherited interface requirements

Conflict diagnostics must include the relevant interface names, method name, and
arity values when available.
