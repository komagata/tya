# Feature: Lexical Closures

## Goal

Implement lexical closures in the active compile-to-C path so function literals
can read bindings from enclosing function scopes while preserving Tya's rule
that function bodies cannot write back to outer bindings.

## Context

Tya already has function literals such as `x -> x + 1`. Historical concurrency
documentation also records the intended closure rule: closures cannot write
back to outer variables, and shared mutable state across tasks should be passed
explicitly and synchronized.

The current Go interpreter has an environment-bearing function representation,
but the active `tya run` path compiles to C. The C runtime currently represents
a function as a C function pointer plus receiver/member/class metadata. It does
not store a lexical environment, so nested functions that read outer locals can
emit C references to variables that are out of scope.

This should work after the feature:

```tya
make_adder = x ->
  y -> x + y

add2 = make_adder(2)
print(add2(3))
```

Expected output:

```text
5
```

## Behavior

- A function literal may read bindings from lexically enclosing function
  scopes.
- Capturable bindings are enclosing function parameters and local bindings.
- Top-level bindings are not closure-captured; they keep the existing
  module/global binding behavior.
- Captures use the value visible at function-literal creation time.

  ```tya
  make = ->
    x = 1
    f = -> x
    x = 2
    f

  print(make()())
  ```

  This prints `1`.
- Captured values are stored as `TyaValue` values. The implementation must not
  deep-copy arrays, dicts, objects, strings, functions, resources, tasks, or
  other heap-backed values.
- A closure may read captured mutable values.

  ```tya
  make = ->
    items = [1, 2]
    ->
      items[0]
  ```

- A closure must not mutate through a captured binding. This includes indexed
  assignment and member assignment through captured arrays, dicts, objects, and
  class instances.

  ```tya
  make = ->
    state = { count: 0 }
    ->
      state["count"] = state["count"] + 1
  ```

  ```tya
  make = ->
    items = [1]
    ->
      items[0] = 2
  ```

  ```tya
  make = ->
    user = User.new("komagata")
    ->
      user.name = "new"
  ```

  These must fail during checking with an actionable diagnostic.
- Direct reassignment to an outer binding remains invalid.

  ```tya
  make = ->
    x = 1
    ->
      x = 2
  ```

- Mutating an explicit parameter remains allowed, because the shared mutable
  value is visible at the function boundary.

  ```tya
  inc = state ->
    state["count"] = state["count"] + 1
  ```

- Shared mutable state used concurrently should be passed explicitly and
  protected with `sync` primitives or channels.
- `self`, `Self`, class fields, static fields, and method dispatch keep their
  existing semantics. This feature does not add new class-context capture
  rules.
- Closure values can be returned from functions and passed to functions.
- Closure values can be passed to stdlib higher-order APIs such as `map`,
  `filter`, `reduce`, and `with_lock`.
- Closure values can be passed to `spawn`; the captured environment must remain
  alive until the closure and any task using it are no longer reachable.
- Captured values participate in GC marking.
- Function members and bound receivers keep working for closure-backed
  functions.
- Existing non-capturing function literals keep their current behavior.

## Scope

- `runtime/tya_runtime.h`
- `runtime/tya_runtime.c`
- `internal/codegen/c.go`
- `internal/checker/checker.go`
- `internal/checker/strict.go`
- `internal/ast/ast.go` only if capture metadata is added to AST nodes
- focused codegen, checker, runtime, and CLI black-box tests
- `docs/SPEC.md`
- release documentation for the version that ships this feature
- self-host verification fixtures as needed to preserve the v01 fixed point

## Out of Scope

- Deep-copy capture semantics.
- Capture-list syntax.
- Allowing closures to reassign outer bindings.
- Allowing closures to mutate captured arrays, dicts, objects, or class
  instances through captured bindings.
- Changing top-level binding lookup.
- New concurrency primitives.
- A race detector.
- Rewriting the interpreter or making it the active execution path.
- Removing or weakening `TestSelfhostV01Scripts`.

## Implementation Notes

- Add runtime support for closure environments associated with `TyaFunction` or
  an equivalent function value representation.
- Keep non-capturing functions on the current simple path when practical.
- Preserve the distinction between method receivers and lexical captures.
- Ensure GC marks closure environments and every captured `TyaValue`.
- Teach codegen to discover free variables for each function literal relative
  to enclosing function scopes.
- Emit closure creation code that stores the current `TyaValue` for each
  captured binding at function-literal creation time.
- Emit closure function bodies so captured-name reads load from the closure
  environment instead of unavailable C locals.
- Reject direct assignment, indexed assignment, and member assignment that
  writes through a captured outer binding.
- The exact environment representation is an implementation detail as long as
  generated C remains deterministic and GC/task lifetime requirements hold.

## Acceptance Criteria

- Returning a closure that reads an enclosing function parameter works.
- Returning a closure that reads an enclosing function local works.
- Multiple closures created from different calls receive independent captured
  values.
- Reassignment in the creator after closure creation does not change the
  closure's captured value.
- Direct reassignment to an outer binding from inside a closure is rejected.
- Indexed assignment through a captured dict is rejected.
- Indexed assignment through a captured array is rejected.
- Member assignment through a captured object or class instance is rejected.
- Mutating an explicit parameter remains allowed.
- Passing a closure to `Array.map`, `Array.filter`, and `Array.reduce` works.
- Passing a closure to `spawn` and awaiting it works.
- Captured heap values remain alive across explicit `runtime.gc()` while the
  closure is reachable.
- Existing non-capturing function-literal behavior remains unchanged.
- `go test ./tests -run TestSelfhostV01Scripts -count=1` remains green.

## Verification

```sh
go test ./... -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
```

Add focused testscript fixtures for:

- returned parameter capture
- returned local capture
- independent closure captures
- post-creation reassignment snapshot behavior
- direct outer reassignment rejection
- captured dict indexed assignment rejection
- captured array indexed assignment rejection
- captured object/member assignment rejection
- explicit parameter mutation remaining allowed
- closure passed to array higher-order methods
- closure passed to `spawn` / `await`
- closure capture surviving `runtime.gc()`
