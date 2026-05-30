# Feature: Callable object call syntax

## Goal

Allow an object with an instance `call` method to be invoked with ordinary call
syntax, so `foo(args...)` behaves like `foo.call(args...)` when `foo` is an
instance value whose class provides `call`.

## Context

Tya already treats functions as values and requires parenthesized calls.
Classes are called to construct instances, and instance methods are called with
member syntax such as `foo.call()`. This feature adds a callable-object layer
for instance values only.

The feature must not change class constructor syntax. `Foo()` remains a
constructor call even when `Foo` defines `static call`. A static `call` method
is still callable only with explicit static member syntax such as `Foo.call()`.

Relevant implementation areas:

- `docs/SPEC.md`
- `internal/checker/checker.go`
- `internal/eval/eval.go`
- `internal/codegen/c.go`
- `internal/formatter/unparse.go`
- parser/formatter/checker/eval/codegen tests
- CLI testscript coverage under `tests/`
- self-host fixed-point tests

## Behavior

- If an expression used as a call callee evaluates to a class instance whose
  class, parent class, or implemented interface default provides an instance
  method named `call`, the call invokes that method on the receiver.

  ```tya
  class Greeter
    prefix: "Hello"

    call: name -> "{prefix}, {name}"

  greet = Greeter()
  print(greet("Tya"))
  ```

  This prints `Hello, Tya`.
- Callable-object syntax is equivalent to explicit member-call syntax for
  argument binding and method dispatch:

  ```tya
  foo(1, b: 2)
  foo.call(1, b: 2)
  ```

  These use the same positional arguments, keyword arguments, default
  arguments, duplicate-keyword checks, unknown-keyword checks, and arity
  diagnostics.
- Callable-object syntax evaluates the receiver expression exactly once before
  evaluating call arguments, matching method-call receiver evaluation rules.
- Callable-object syntax works when the callee expression is any expression
  that produces a callable object, including local bindings, fields, array
  indexes, dictionary indexes, and ordinary function return values.
- Plain function values keep their current call behavior. This feature does
  not route function values through a `call` member.
- Class calls keep their current constructor behavior. `Foo()` constructs
  `Foo`; it does not call `Foo.call()`, even when `Foo` defines `static call`.
- Explicit static calls remain valid where they are already valid:

  ```tya
  Foo.call()
  ```

- An object without an applicable instance `call` method is not callable.
  When the checker can prove the callee is a non-callable class instance, it
  should reject the program with an actionable diagnostic. When the callee is
  dynamically unknown until runtime, `tya run` and generated binaries must fail
  with a clear runtime error such as `object is not callable`.
- If a class provides a field named `call` but no instance method named `call`,
  the object is not callable. Callable-object syntax is based on methods, not
  arbitrary fields.
- Visibility rules are unchanged. A private or protected `call` method is only
  callable through `foo()` from contexts where `foo.call()` would already be
  allowed.
- Inheritance rules match ordinary method dispatch. A subclass inherits
  callable-object behavior from an inherited public `call` method, and
  overriding `call` changes the callable behavior.
- Formatter canonicalizes explicit public instance `call` member calls to
  callable-object syntax when the rewrite is semantically equivalent:

  ```tya
  foo.call(1, b: 2)
  ```

  formats to:

  ```tya
  foo(1, b: 2)
  ```

- Formatter must not rewrite class/static calls such as `Foo.call(...)`, module
  class calls such as `pkg.Foo.call(...)`, or any call where the receiver is
  known to be a class object rather than an instance value.
- Formatter must not rewrite `foo.call` member reads that are not calls.
- Formatting is idempotent: formatting the callable-object output again does
  not change it.

## Scope

- Add SPEC documentation for callable-object syntax and its relationship to
  functions, methods, constructors, keyword arguments, and formatted syntax.
- Update checker call validation for non-function callee expressions that may
  be callable objects.
- Update evaluator call dispatch so callable objects route to their instance
  `call` method.
- Update C code generation and runtime call dispatch so compiled programs
  behave the same as interpreted execution.
- Update formatter/unparser canonicalization from explicit `.call(...)` to
  callable-object syntax where the receiver is not a class/static receiver.
- Add focused parser/formatter/checker/eval/codegen/CLI tests.
- Preserve the self-host fixed point.

## Out of Scope

- Treating class objects as callable through `static call`.
- Changing constructor calls.
- Adding a new `Callable` interface or requiring classes to declare one.
- Treating fields named `call` as callable functions.
- Array splat or variadic call syntax.
- Bound method reference syntax such as `handler = foo.call`.
- Callable dictionary values through a `call` key.
- Automatically rewriting unrelated method names.
- Weakening visibility rules for private or protected methods.

## Acceptance Criteria

- A class with `call: -> ...` can be instantiated and invoked with `obj()`.
- `obj(args...)` produces the same result as `obj.call(args...)`.
- Positional, keyword, and default arguments bind exactly as they do for
  explicit `obj.call(...)`.
- Duplicate keyword, unknown keyword, too-few-argument, and too-many-argument
  diagnostics match the existing method-call behavior as closely as practical.
- A callable object returned from a function can be invoked immediately, such
  as `factory()("Tya")`, without evaluating `factory()` twice.
- Callable objects stored in arrays, dictionaries, and fields can be invoked.
- Inherited and overridden `call` methods dispatch the same way as ordinary
  methods.
- `Foo()` remains a constructor call, even if `Foo` defines `static call`.
- `Foo.call()` remains an explicit static method call when such a method
  exists.
- An object without an instance `call` method fails clearly when invoked.
- A class with a field named `call` but no `call` method is not callable.
- Formatter rewrites `foo.call(...)` to `foo(...)` when the receiver is an
  instance expression and the rewrite is semantically equivalent.
- Formatter does not rewrite `Foo.call(...)`, `pkg.Foo.call(...)`, or
  non-call member reads such as `handler = foo.call`.
- Evaluator and generated-C execution agree for all accepted callable-object
  cases.
- Existing function calls, constructor calls, method calls, and keyword
  argument behavior continue to pass.

## Verification

```sh
go test ./internal/parser ./internal/formatter ./internal/checker ./internal/eval ./internal/codegen -count=1
go test ./tests -run 'TestCLI|TestV.*Scripts|TestSelfhostV01Scripts|TestSelfhostV02Scripts' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
