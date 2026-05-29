---
layout: doc
title: Static Typing Discussion Note
permalink: /static-typing-discussion/
---

# Static Typing Discussion Note

Status: discussion note only. This document records a static-typing design that
was considered during language discussion. It is not accepted, not proposed for
a release, not on the roadmap, and not an implementation plan. `docs/SPEC.md`
remains the authority for the current dynamically typed language.

The notes below preserve the shape of the discussion in case the tradeoffs need
to be revisited. They should not be read as a direction Tya will take.

## Non-Goals For Current Tya

- Do not change the current dynamic language.
- Do not add static typing to the active compiler, checker, emitter, runtime,
  formatter, LSP, tests, or self-host compiler yet.
- Do not weaken the self-host fixed point.
- Do not treat this note as v1.0.0 scope.

## Core Type Rules

Types are explicit. There is no type inference.

```tya
name: String = "Tya"
count: Int = 3
active: Bool = true
```

Initial declaration uses `name: Type = value`. Reassignment uses `name = value`
and must preserve the declared type.

```tya
count: Int = 1
count = 2
count = "two" # invalid
```

Shadowing is forbidden. A name cannot be redeclared in an inner scope.

`SCREAMING_SNAKE_CASE` names are constants. Constants cannot be reassigned, and
their contained values cannot be mutated.

```tya
ITEMS: Array<Int> = [1, 2]
ITEMS = []    # invalid
ITEMS.push(3) # invalid
```

## Function Values

There is no separate named-function declaration syntax. A named function is a
typed variable holding a function literal.

```tya
add: (Int, Int): Int =
  (a: Int, b: Int): Int ->
    return a + b
```

Function type syntax:

```tya
(ArgType, ArgType): ReturnType
```

Function literal syntax:

```tya
(arg: ArgType, arg: ArgType): ReturnType -> body
```

`Void` is the return type for procedures.

```tya
log: (String): Void =
  (message: String): Void ->
    print(message)
    return
```

Implicit return is forbidden. Non-`Void` functions must return a value on every
control-flow path. `Void` functions may use bare `return`; `return nil` is
invalid.

## Generics

Generic functions put type parameters on the function name.

```tya
identity<T>: (T): T =
  (value: T): T ->
    return value
```

Generic calls always spell type arguments explicitly.

```tya
value: Int = identity<Int>(1)
```

Generic classes and interfaces use `Name<T>`.

```tya
class Box<T>
  value: T

  initialize: (T): Void =
    (value: T): Void ->
      self.value = value
      return
```

```tya
interface Iterator<T>
  next: (): T?
```

Generic constraints are allowed when a type parameter must implement an
interface.

```tya
show<T: Stringable>: (T): String =
  (value: T): String ->
    return value.to_string()
```

## Nil And Optional Types

`nil` is only valid for optional types. `T` and `T?` are distinct.

```tya
name: String = nil   # invalid
name: String? = nil  # valid
```

The nil-coalescing operator `??` converts an optional value to a non-optional
value by providing a default.

```tya
name: String? = nil
display_name: String = name ?? "anonymous"
```

`??` checks only `nil`; `false`, `0`, and `""` are ordinary values.

Forced unwrap syntax is not part of this note. Optional values must be handled
with `??` or explicit nil checks. Comparing non-optional values to `nil` is
invalid.

## Collections

Arrays and dictionaries are generic and homogeneous.

```tya
items: Array<Int> = [1, 2, 3]
scores: Dict<String, Int> = {"alice": 10}
```

Empty literals are valid when the declared type supplies the element types.

```tya
items: Array<Int> = []
scores: Dict<String, Int> = {}
```

Array and dictionary indexing returns the element value, not an optional.
Missing array indexes or dictionary keys are runtime errors.

```tya
item: Int = items[0]
score: Int = scores["alice"]
```

Use `Dict<K, V>.get(key)` when absence is expected.

```tya
maybe_score: Int? = scores.get("bob")
score: Int = scores.get("bob") ?? 0
```

Dictionary keys are expressions. String keys must be quoted.

```tya
labels: Dict<String, String> = {"name": "Tya"}
```

Mutable variables may update collection elements if the value type matches.
Constants may not.

```tya
items[0] = 9
scores["bob"] = 20
```

## Classes And Interfaces

Class fields are declared explicitly. `private` fields are allowed. Non-optional
fields must be initialized by every `initialize` overload, unless they have a
field initializer.

```tya
class User
  private name: String
  age: Int = 0
  nickname: String?

  initialize: (String): Void =
    (name: String): Void ->
      self.name = name
      self.nickname = nil
      return
```

Object fields may be updated through mutable variables. Fields reached through
constants may not be changed.

Interfaces are implemented explicitly. Structural, implicit implementation is
not part of this note.

```tya
interface Stringable
  to_string: (): String

class User implements Stringable
  to_string: (): String =
    (): String ->
      return self.name
```

Generic interfaces are allowed.

## Overload

Top-level functions, methods, and `initialize` may be overloaded.

```tya
parse: (String): Int =
  (text: String): Int ->
    return text.to_i()

parse: (Bytes): Int =
  (data: Bytes): Int ->
    return data.to_string().to_i()
```

Overload identity is `name + argument type list`. Return-type-only overloads
are invalid. A call must resolve to exactly one candidate. There is no overload
resolution through implicit conversion.

`nil` passed directly to an overloaded call is invalid when the target overload
is ambiguous.

Children may add overloads with new signatures. Replacing a parent method with
the same signature requires `override`.

## Operators And Control Flow

There are no implicit numeric conversions. Mixed numeric operations are invalid.

```tya
1 + 2      # Int
1.0 + 2.0  # Float
1 + 2.0    # invalid
```

`Int / Int` returns `Int` and truncates toward zero. `Float / Float` returns
`Float`.

`if` and `while` conditions must be `Bool`.

```tya
if count > 0
  print(count)
```

`for` loop variables spell their type explicitly.

```tya
for item: String in names
  print(item)
```

Equality is valid for matching types only. Optional values may be compared to
`nil`.

`match` is treated as a statement in this note, not as a typed expression.
`try` and `catch` are statement-oriented; raised errors are not part of function
types.

## Builtins And Display

`print` remains a special builtin that accepts any type and returns `Void`.
This does not introduce general implicit conversion.

String interpolation is also display-special and may embed any type.

```tya
count: Int = 3
print(count)
message: String = "count: {count}"
```

Other string contexts require explicit conversion methods.

## Casts

Runtime casts were included in the discussed design.

```tya
dog: Dog = animal as Dog
maybe_dog: Dog? = animal as? Dog
```

`as` fails with a runtime error. `as?` returns `nil` on failure.

Upcasts from a child class to a parent class are assignment-compatible.
Conversions from a class to an explicitly implemented interface are also
assignment-compatible. Downcasts require `as` or `as?`.
