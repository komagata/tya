# Tya v0.61 Specification

> **Status:** released. v0.61 grows `interface` into Tya's stackable-trait
> mechanism. This is intentionally a large `/goal` scope: default method bodies,
> interface-contributed state, deterministic initialization, conflict
> resolution, and `super` across stacked interfaces are all part of the target.
> The keyword remains `interface`; Tya does not support both `interface` and
> `trait`.

## Theme

Tya currently has explicit interfaces:

```tya
interface Reader
  read = ->

class File implements Reader
  read = ->
    "data"
```

That is useful for contracts, but not enough for stackable behavior. v0.61
turns `interface` into the single composition mechanism for reusable behavior
that does not require class inheritance.

The target is:

```tya
interface Named
  name = ->

  label = ->
    self.name()

interface Timestamped
  created_at = nil

  initialize = ->
    self.created_at = Time.now()

  age = ->
    Time.since(self.created_at)

class User implements Named, Timestamped
  initialize = name ->
    self.name_value = name
    super()

  name = ->
    self.name_value
```

`User` receives the `label`, `created_at`, and `age` behavior while still using
ordinary single class inheritance for `extends`.

## Goals

- Keep `interface` as the single user-facing concept.
- Add interface default instance methods.
- Add interface fields as state requirements / contributions.
- Add interface initialization hooks with deterministic ordering.
- Define `super` across class inheritance and stacked interfaces.
- Define conflict rules for methods, fields, and initialization.
- Preserve explicit `implements`; no implicit conformance.
- Preserve single class inheritance.
- Keep method overloading out of Tya.
- Keep the implementation plan large enough for `/goal` long-running work.

## Non-goals

- No separate `trait` keyword in v0.61.
- No support for both `interface` and `trait` as synonyms.
- No implicit interfaces generated from classes.
- No implementing a class as if it were an interface.
- No method overloading.
- No generics.
- No operator-overload traits.
- No sealed / base / open class hierarchy system.

## Naming Decision

v0.61 keeps `interface`.

The concept becomes trait-like, but Tya should not have two spellings for the
same abstraction. Canonical Syntax requires one source representation. If Tya
renames the concept to `trait` later, that must be a clean migration, not a
long-term alias.

Documentation should use:

```text
interface
interface default method
interface field
interface initializer
stacked interface
```

Avoid using `trait` as syntax in examples.

## Interface Members

An interface may contain:

- body-free instance method requirements;
- default instance methods;
- field declarations;
- `initialize` hooks.

Example:

```tya
interface Audited
  created_at = nil

  initialize = ->
    self.created_at = Time.now()

  touch = ->
    self.created_at = Time.now()

  label = ->
```

An interface may not contain:

- static fields;
- static methods;
- private members;
- class constructors other than the interface `initialize` hook;
- nested class declarations;
- nested interface declarations.

## Default Methods

A body-free method remains a requirement:

```tya
interface Reader
  read = ->
```

A method with a body is a default:

```tya
interface Reader
  read_twice = ->
    self.read() + self.read()
```

Default methods are inherited by implementing classes when no class method with
the same name exists.

```tya
interface Named
  name = ->

  label = ->
    self.name()

class User implements Named
  name = ->
    "user"

print(User().label())  # "user"
```

A class method wins over an interface default:

```tya
class Admin implements Named
  name = ->
    "admin"

  label = ->
    "admin:{self.name()}"
```

`Admin().label()` uses the class body.

## Default Methods Satisfy Requirements

A default method can satisfy a compatible body-free requirement from another
interface.

```tya
interface RequiresLabel
  label = ->

interface DefaultLabel
  label = ->
    "default"

class Item implements RequiresLabel, DefaultLabel
```

`Item` is concrete because `DefaultLabel.label` supplies the implementation.

If the arity differs, it is a conflict.

## Interface Fields

An interface field contributes instance state to implementing classes.

```tya
interface Timestamped
  created_at = nil

  age = ->
    Time.since(self.created_at)
```

A class implementing `Timestamped` has a `created_at` instance field unless it
declares that field itself.

```tya
class Post implements Timestamped
```

is equivalent, observably, to a class with a `created_at = nil` instance field
and the default `age` method, after conflict resolution.

Field initializers are evaluated for each instance during construction. They
may reference imported modules and constants. They should not reference `self`
unless the implementation can guarantee a deterministic point where `self`
exists and earlier fields are initialized. The v0.61 rule is:

- interface field initializers may not reference `self`;
- interface initializer hooks may reference `self`.

Invalid:

```tya
interface Bad
  name = self.default_name()
```

Valid:

```tya
interface Good
  name = nil

  initialize = ->
    self.name = self.default_name()
```

## Field Conflict Rules

Tya has one field per name on an instance. Interfaces must not silently define
two different fields with the same name.

Conflict:

```tya
interface A
  enabled = false

interface B
  enabled = false

class User implements A, B
```

Even though the initializer source is the same, v0.61 requires the class or a
child interface to resolve same-name interface fields explicitly. This keeps
state composition visible and avoids subtle changes when two interfaces evolve
independently.

Also conflicting:

```tya
interface A
  enabled = false

interface B
  enabled = true

class User implements A, B
```

The class must resolve the conflict explicitly:

```tya
class User implements A, B
  enabled = false
```

A class field declaration wins over interface fields with the same name.

Any same-name interface field conflict is resolved by a class field declaration
or by a child interface field declaration. Initializer equality does not make
two interface fields compatible in v0.61.

## Interface Initialization

An interface may declare an `initialize` hook:

```tya
interface Timestamped
  created_at = nil

  initialize = ->
    self.created_at = Time.now()
```

Interface initializers are not constructors. They do not receive the class
constructor arguments unless the class passes state through fields or calls
ordinary methods. The signature must be zero-arity in v0.61:

```tya
initialize = ->
```

This keeps construction deterministic and avoids argument routing across
multiple stacked interfaces.

When constructing an instance:

1. parent class construction runs according to existing `super(...)` rules;
2. class and interface fields are initialized in deterministic order;
3. interface initializer hooks run in deterministic order;
4. the class constructor body continues after its `super()` point, or starts
   after implicit parent/interface initialization when no parent constructor is
   required.

## Initialization Order

Interface order is source order.

```tya
class User implements A, B, C
```

The effective interface order is:

1. parents of `A`, then `A`;
2. parents of `B`, then `B`;
3. parents of `C`, then `C`;
4. duplicates removed by interface identity, keeping the first occurrence.

This is depth-first, left-to-right, postorder, with de-duplication.

Example:

```tya
interface Root
  initialize = -> log("Root")

interface A extends Root
  initialize = -> log("A")

interface B extends Root
  initialize = -> log("B")

class User implements A, B
```

Initialization order:

```text
Root
A
B
```

`Root` runs once.

## Class Constructor Interaction

If a class has no `initialize`, interface fields and initializers still run.

```tya
interface Timestamped
  created_at = nil
  initialize = -> self.created_at = Time.now()

class Post implements Timestamped

post = Post()
```

If a class has `initialize`, interface initialization happens at the class's
`super()` point. This is true even for root classes that do not extend another
class.

Recommended canonical form when implementing interfaces with initialization:

```tya
class Post implements Timestamped
  initialize = title ->
    super()
    self.title = title
```

In a root class, `super()` means "run interface initialization chain". This is a
v0.61 extension. In a subclass, `super(args...)` first calls the parent
constructor; the parent constructor is responsible for its own interface chain.
After parent construction, the subclass's newly implemented interfaces run.

If a class implements interfaces with initializer hooks and declares
`initialize` without calling `super()`, the checker rejects it. This mirrors the
existing parent-constructor rule and prevents skipped interface initialization.

This keeps the class-constructor spelling `initialize` but avoids treating an
interface initializer as a second constructor. Interface `initialize` hooks are
zero-arity lifecycle hooks; class `initialize` methods remain the only
constructors that receive construction arguments.

## `super` in Interface Methods

`super()` participates in stacked interface default methods.

Within an interface default method, `super()` calls the next implementation of
the same method in the stack.

Order for method lookup inside a class:

1. class method;
2. parent class method chain;
3. effective interface defaults in stack order;
4. missing method.

Stack order is Scala-like: in `implements A, B`, the rightmost interface wraps
the interfaces to its left. A class method is always before interface defaults.

When an interface default calls `super()`, lookup resumes after that interface
in the effective interface method chain.

Example:

```tya
interface BaseLabel
  label = ->
    "base"

interface BracketLabel extends BaseLabel
  label = ->
    "[" + super() + "]"

interface StarLabel
  label = ->
    "*" + super() + "*"

class User implements BracketLabel, StarLabel
```

Effective method stack for `label`:

```text
StarLabel.label
BracketLabel.label
BaseLabel.label
```

`User().label()` returns `"*[base]*"` if `StarLabel` wraps
`BracketLabel`. The exact order must follow the effective interface order rule
above and be covered by tests.

If there is no next implementation, `super()` is an error. The checker should
detect this when possible.

## Class Overrides and `super`

When a class overrides a method provided by interfaces, `super()` in the class
method calls the next implementation after the class method.

```tya
interface Label
  label = ->
    "interface"

class User implements Label
  label = ->
    "class:" + super()
```

`User().label()` returns `"class:interface"`.

If a parent class provides the method, class inheritance remains first:

```tya
class Base
  label = ->
    "base"

interface Label
  label = ->
    "interface"

class User extends Base implements Label
  label = ->
    "class:" + super()
```

`super()` calls `Base.label`, not `Label.label`. Class inheritance wins over
interface defaults. Interface defaults only fill gaps after the class chain.

## Conflict Rules

Tya does not support overloading. A method name has one effective arity.

### Duplicate Requirements

Two body-free requirements with the same arity are compatible.

### Default vs Requirement

A default method with the same arity satisfies a body-free requirement.

### Class Method Wins

A class method with the same name overrides all interface defaults for that
method.

### Stackable Defaults

Two unrelated defaults with the same name and arity are not silently accepted.
The implementing class, or the child interface that combines them, must declare
the method explicitly. That explicit method resolves the conflict and may call
`super()` to enter the ordered stack.

```tya
interface A
  label = -> "a"

interface B
  label = -> "b:" + super()

class User implements A, B
  label = ->
    super()
```

This is valid because `User.label` explicitly resolves the conflict. Since
`implements A, B` uses rightmost-wraps-leftmost order, `super()` calls
`B.label`, and `B.label` may call `A.label`.

Without the explicit `User.label`, this is an error:

```tya
class User implements A, B
```

The diagnostic should tell the author to declare `label` and choose the desired
composition.

### Ambiguous Diamond

Duplicate inheritance of the same source default is compatible and runs once.

```tya
interface Root
  label = -> "root"

interface A extends Root
interface B extends Root

class User implements A, B
```

There is one `Root.label`.

If two unrelated parent interfaces contribute same-name defaults and a child
interface extends both without overriding, the child interface is invalid:

```tya
interface C extends A, B
```

It must resolve the conflict locally:

```tya
interface C extends A, B
  label = ->
    super()
```

Inside that explicit method, `super()` follows rightmost-wraps-leftmost order:
`B.label`, then `A.label`.

### Arity Conflict

Same method name with different arity is always an error unless the class or
child interface declares a method that makes the intended arity explicit and all
other requirements are compatible with it. Since Tya has no overloading,
incompatible arities usually require renaming one method.

## Static and Private Members

Interface static members remain invalid in v0.61. Stackable behavior is
instance behavior.

Private interface members are invalid. Interface methods and fields are public
because they are composed into implementing classes.

## Runtime / Codegen Model

The implementation should lower effective interface contributions into class
metadata before C emission.

Observable requirements:

- interface fields become instance fields;
- interface field initialization runs per instance;
- interface initializer hooks run once per interface identity;
- default methods are callable through ordinary method dispatch;
- class methods override interface defaults;
- `super()` traverses class chain first, then interface method stack;
- conflicts are diagnosed before C is emitted.

Preferred lowering:

1. compute effective interfaces for each class;
2. compute effective fields and detect field conflicts;
3. compute effective initializer order;
4. compute method stacks per method name;
5. emit wrapper methods for interface defaults where needed;
6. make `super()` in generated methods carry an explicit next-method target.

This may require extending the current `super` representation in codegen.
Runtime dynamic search alone is not enough unless it can resume lookup from a
specific point in the interface stack.

## Diagnostics

New diagnostics should be stable and actionable:

| Code | Meaning |
|---|---|
| `TYA-E0830` | conflicting interface method arity |
| `TYA-E0831` | conflicting interface field initializer |
| `TYA-E0832` | invalid member in interface body |
| `TYA-E0833` | interface initializer must be zero-arity |
| `TYA-E0834` | class constructor must call `super()` to run interface initialization |
| `TYA-E0835` | `super()` has no next method in interface stack |
| `TYA-E0836` | interface static members are not supported |
| `TYA-E0837` | private interface members are not supported |

Existing diagnostics for unknown interfaces, duplicate implements entries,
inheritance cycles, classes extending interfaces, and interfaces extending
classes still apply.

## Implementation Plan

This is intentionally a long-running `/goal` implementation.

1. **Parser / AST**
   - Allow interface method bodies.
   - Allow interface field declarations.
   - Allow interface `initialize = ->` hook.
   - Reject static/private interface members.

2. **Formatter**
   - Format body-free requirements, default methods, fields, and initializers.
   - Preserve canonical member ordering rules once decided.

3. **Checker interface model**
   - Replace simple `method -> arity` interface info with records for
     requirements, defaults, fields, and initializers.
   - Track source interface identity for diamonds.

4. **Effective interface graph**
   - Compute depth-first left-to-right postorder.
   - Remove duplicate interface identities.
   - Preserve enough source order for method stacks and initialization.

5. **Conflict resolver**
   - Resolve method arity conflicts.
   - Resolve field initializer conflicts.
   - Build method stacks.
   - Verify `super()` has a next target when required.

6. **Class integration**
   - Merge interface fields into classes.
   - Require `super()` in constructors when interface initialization must run.
   - Let class declarations override fields and methods explicitly.

7. **Codegen**
   - Emit interface default methods as class-callable methods.
   - Emit interface field initialization.
   - Emit interface initializer calls in deterministic order.
   - Extend `super()` lowering for interface stacks.

8. **Runtime**
   - Prefer no new value kind.
   - Add helper support only if codegen cannot encode next-method targets
     statically.

9. **Self-host**
   - Teach the self-host parser/checker/codegen the new interface body shapes
     once the Go implementation is stable.
   - Preserve the fixed point.

10. **Docs and examples**
   - Add examples for default methods, stateful interfaces, initializer order,
     and stackable `super`.
   - Document when abstract classes are still the right tool.

11. **Verification**
   - Existing v0.11 / v0.12 interface tests pass.
   - New tests cover defaults, fields, initialization order, class override,
     interface `super`, class `super` into interface stack, diamonds, and
     conflicts.
   - `go test ./... -count=1` passes, including self-host.

## Migration Guidance

Use abstract classes when shared behavior needs one implementation base:

```tya
abstract class Repository
  abstract find = id ->

  first = ->
    self.find(1)
```

Use interfaces when behavior should be stacked with other behavior:

```tya
interface FindFirst
  find = id ->

  first = ->
    self.find(1)
```

Use interface fields and initializers when the behavior owns a small,
self-contained state slot:

```tya
interface Counted
  count = 0

  increment = ->
    self.count = self.count + 1
```

If the behavior requires complex construction arguments or strong invariants,
prefer an ordinary class until the interface model proves itself in practice.

## Success Criteria

v0.61 is complete when:

- interface methods may be body-free requirements or default methods;
- interface fields are composed into implementing classes;
- interface initializers run in deterministic order;
- `super()` works through class inheritance and stacked interface defaults;
- class members can explicitly override interface contributions;
- interface inheritance diamonds are deterministic and de-duplicated;
- method and field conflicts produce structured diagnostics;
- old v0.11 and v0.12 interface behavior is preserved;
- `go test ./... -count=1` passes, including the self-host fixed point.
