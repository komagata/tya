# Feature: Constructor Default Arguments

## Goal

Class construction should use `initialize` default arguments the same way ordinary function and method calls do, so constructors with trailing defaults can be called with omitted arguments.

## Context

GitHub issue #18 reports that a class whose `initialize` method defines default arguments still requires the full constructor arity at check time or runtime.

`stdlib/net/ip/Address.tya` currently defines:

```tya
initialize = version = nil, bytes = nil, groups = nil, mapped = false ->
  self.version = version
  self.bytes = bytes
  self.groups = groups
  self.mapped = mapped
```

The intended call shape is:

```tya
import net/ip as ip

parser = ip.Address()
addr = parser.parse("127.0.0.1")
print(addr.to_s())
```

The current workaround in `tests/stdlib_net_ip_test.tya` is to pass every default explicitly:

```tya
ip.Address(nil, nil, nil, false)
```

Normal functions and methods already support omitted trailing default arguments. Constructors should follow the same rule because class construction delegates to `initialize`.

## Behavior

- A class call invokes `initialize` using the same default-argument rules as ordinary function and method calls.
- If every `initialize` parameter has a default value, callers may pass no arguments.
- If only trailing `initialize` parameters have default values, callers may omit those trailing arguments.
- Constructor arguments below the required non-default parameter count remain invalid.
- Constructor arguments above the total `initialize` parameter count remain invalid.
- Default expressions are evaluated when the constructor call omits that argument, using the same timing and scope rules as method default arguments.
- Constructor arity diagnostics should describe the accepted arity range when defaults create a range, matching the existing function/method style where practical.
- Constructor privacy, abstract-class checks, interface initializer checks, `super(...)`, and inheritance behavior remain unchanged except where they need the same default-argument arity/range semantics.

Examples:

```tya
class Address
  initialize = version = nil, bytes = nil, groups = nil, mapped = false ->
    self.version = version

Address()
Address(4)
Address(4, [127, 0, 0, 1])
```

```tya
class User
  initialize = name, role = "member" ->
    self.name = name
    self.role = role

User("komagata")
User("komagata", "admin")
```

Invalid examples:

```tya
User()
User("komagata", "admin", "extra")
```

## Scope

- Checker constructor arity logic in `internal/checker/checker.go`.
- Generated C constructor invocation/default handling in `internal/codegen/c.go`.
- Interpreter/runtime class instantiation path in `internal/eval/eval.go`, if needed to preserve parity.
- Tests for constructor default arguments in checker, codegen, testscript, or stdlib tests as appropriate.
- Update `tests/stdlib_net_ip_test.tya` to use `ip.Address()` instead of explicit default placeholders.
- Update `docs/SPEC.md` to state that constructor calls use `initialize` default arguments like ordinary method calls.

## Out of Scope

- Adding named arguments.
- Allowing omitted middle arguments.
- Changing default-argument syntax.
- Changing constructor visibility rules.
- Changing abstract class construction rules.
- Changing interface `initialize` arity rules.
- Reworking inheritance or `super(...)` semantics beyond default-argument arity/range support where already applicable.
- Changing public `net/ip` APIs except removing the explicit-default workaround in tests.

## Acceptance Criteria

- `ip.Address()` is valid and can parse/format `127.0.0.1` through the existing `net/ip` API.
- `tests/stdlib_net_ip_test.tya` no longer needs `ip.Address(nil, nil, nil, false)` as a workaround.
- A constructor with all-default parameters can be called with zero arguments.
- A constructor with required parameters followed by default parameters can be called with only the required arguments.
- Constructor calls with too few required arguments still fail.
- Constructor calls with more arguments than declared parameters still fail.
- Checker behavior and generated-C/runtime behavior agree for valid and invalid constructor calls.
- Existing constructor privacy, abstract-class, inheritance, interface initializer, and `super(...)` tests continue to pass.
- `docs/SPEC.md` documents constructor default-argument behavior without contradicting existing function/method default-argument rules.
- The self-host invariant remains valid.

## Verification

```sh
gofmt -w <changed-go-files>
go test ./... -count=1
```

Focused checks likely useful during implementation:

```sh
go test ./internal/checker ./internal/codegen ./internal/eval -count=1
go test ./tests -run 'TestStdlib|TestSelfhostV01Scripts' -count=1
```
