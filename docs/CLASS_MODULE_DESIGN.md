# Class, Module, Dict, And Set Design

This document records planned language semantics that are not implemented yet.
It is the source note for future implementation work around classes, modules,
dictionaries, sets, imports, and entry files.

## Terms

Tya should use these terms consistently:

```text
dictionary / dict = key-value collection
set               = unique-value collection
object            = class instance
class             = object blueprint
module            = imported namespace-like public value
hash              = hash function or hash value only
map               = not used as a data type term
```

`dict` is an accepted abbreviation of `dictionary`. `hash` should not be used
as a synonym for dictionary.

## Dictionaries And Sets

Curly literals are split by whether entries contain `:`.

```tya
user = { name: "komagata", age: 20 } # dict
roles = { "admin", "owner" }         # set
empty = {}                           # empty dict
empty_roles = set()                  # empty set
```

Dictionary entries and set entries must not be mixed in one literal.

```tya
bad = { name: "komagata", "admin" } # invalid
bad = { "admin", name: "komagata" } # invalid
```

Suggested diagnostic:

```text
cannot mix dict entries and set entries in one literal
```

Dictionaries are not objects. They must use index access.

```tya
user = { name: "komagata" }

print user["name"] # ok
print user.name    # invalid
```

The existing indented object-literal form should become an indented dict
literal.

```tya
user =
  name: "komagata"
  age: 20
```

This is equivalent to:

```tya
user = { name: "komagata", age: 20 }
```

## Classes And Objects

Only class instances are objects.

```tya
# user.tya
class User
  init: name ->
    @name = name

  greet: ->
    "Hello, {@name}"
```

```tya
import user

user = User("komagata")
print user.name
print user.greet()
```

Class names are `PascalCase`. File names remain `snake_case`; the public class
name must match the PascalCase form of the file basename.

```text
user.tya             -> class User
http_client.tya      -> class HttpClient
xml_http_request.tya -> class XmlHttpRequest
```

The `.` operator is valid for class instance objects and modules, but not for
dictionaries, sets, or arrays.

```text
module.member
object.property
object.method()
```

Current implementation status: classes, constructors, instance fields, methods,
inheritance, `super`, explicit interfaces, module declarations, import aliases,
and entry-file/imported-file top-level rules are implemented for the
interpreter. Generated C supports module declarations, but class, inheritance,
`super`, and interface declarations are still interpreter/checker-only and are
rejected before C emission.

## Inheritance And Interfaces

Class inheritance is single inheritance only. A class may extend at most one
parent class.

```tya
class Admin extends User
```

Interfaces are behavior contracts. A class may implement multiple interfaces.

```tya
interface Reader
  read: -> string

interface Writer
  write: text -> nil

class File implements Reader, Writer
```

Like Dart, every class also defines an implicit interface for its public API.
`extends` reuses the parent implementation. `implements` only promises API
compatibility and does not inherit implementation.

```tya
class MockUser implements User
  greet: ->
    "Hello, test"
```

Method overloading is not supported. A class, interface, or module may not
define multiple methods with the same name.

Method overriding is supported. A subclass may define a method with the same
name as a parent method. Initial checking should require matching arity; if
type annotations are added later, override signature compatibility can be
checked then.

```tya
class User
  greet: ->
    "hello"

class Admin extends User
  greet: ->
    "admin"
```

`super` is supported inside class methods and `init`. It calls the parent
class's method with the same name, or the parent `init` when used in `init`.
`super` must be called explicitly with arguments; there is no implicit argument
forwarding.

```tya
class User
  init: name ->
    @name = name

  greet: ->
    "Hello, {@name}"

class Admin extends User
  init: name, role ->
    super name
    @role = role

  greet: ->
    "{super()} ({@role})"
```

Mixins are not part of the initial design. Implementation sharing should use
single inheritance. API contracts should use interfaces. Shared helper behavior
should use modules or composition.

Advanced Dart-style modifiers such as `interface class`, `abstract interface
class`, `final class`, `base class`, or `sealed class` are not part of the
initial design. They can be considered later if package API stability needs
more control.

## Modules

Imported non-class files define a `module`.

```tya
# util.tya
module util
  foo: "foo"

  bar: ->
    print "bar"
```

```tya
import util

print util.foo
util.bar()
```

Modules are not dictionaries. Module members use `.` access. Dictionary members
use `[]` access.

## One File, One Definition

Every imported file must define exactly one public top-level `class` or
`module`.

```text
user.tya -> class User
util.tya -> module util
```

The file basename must match the public declaration:

```text
snake_case(public class name) == file basename
module name == file basename
```

Invalid examples:

```tya
# user.tya
class Account # invalid: user.tya must define User
```

```tya
# user.tya
class User
class Profile # invalid: more than one top-level definition
```

```tya
# util.tya
module helper # invalid: util.tya must define module util
```

Top-level helper functions, variables, and private classes are also disallowed
in imported files. If a helper class or helper value is needed, place it in its
own file and import it.

```tya
# parser.tya
import parse_error

class Parser
  parse: source ->
    ParseError("bad syntax")
```

```tya
# parse_error.tya
class ParseError
  init: message ->
    @message = message
```

## Imports

Default import behavior is Ruby-like: the imported public name is loaded
directly into the current scope.

```tya
# user.tya
class User
```

```tya
import user

user = User("komagata")
```

For modules:

```tya
import util

util.bar()
```

`as` aliases the imported public value. It does not load the original public
name into the current scope.

```tya
import user as account

user = account("komagata")
```

```tya
import util as u

u.bar()
```

Import name conflicts should be errors unless an alias is used.

## Entry Files

An entry file is the file passed directly to the runner. Entry files are
executed as if all non-import top-level statements were inside an implicit
`main` function.

```tya
import user

user = User("komagata")
print user.name
```

is equivalent to:

```tya
import user

main = ->
  user = User("komagata")
  print user.name

main()
```

Entry files may contain multiple statements. Imported files may not; imported
files must follow the one-file-one-definition rule.

Entry files should not define `class` or `module` directly. Put definitions in
separate importable files.

## Import Tooling

Because public names map mechanically to file names, Tya can later provide an
import fixer.

```text
User       -> import user
HttpClient -> import http_client
util       -> import util
```

The fixer can inspect undefined names and add imports at the top of the entry
file. Ambiguous matches should be reported instead of fixed automatically.
