# Feature: Initialize Field Method Shadowing

## Goal

Instance field assignment during `initialize` should not overwrite an instance method with the same name, while methods remain callable during construction.

## Context

Issue #13 reports that assigning to a field with the same name as an instance method inside `initialize` overwrites the method member on the object:

```tya
class Response
  initialize = ->
    self.status = 200

  status = ->
    self.status

response = Response()
print(response.status())
```

The expected output is:

```text
200
```

The current runtime object shape binds instance methods before `initialize` runs, then `self.status = 200` writes over the public `status` method member. This breaks class-style APIs such as `Response.status` and `Request.url`, where the field storage name and getter method name naturally match.

Construction still needs early method binding so `initialize` can call helper methods such as `self.helper()`. The fix should separate instance field storage from callable method members, or otherwise reestablish method bindings without losing initialized field values.

## Behavior

- `self.<name> = value` inside `initialize` stores an instance field value without replacing an existing instance method named `<name>`.
- Instance methods are callable during `initialize`.
- After construction, `object.<name>()` calls the method when `<name>` is both a field storage name and a method name.
- Inside that method, `self.<name>` reads the initialized field value when used as a value expression.
- Assigning to `self.<name>` from any instance method updates the field value and still does not destroy the method binding.
- Existing behavior for ordinary fields without same-name methods remains unchanged.
- Existing behavior for ordinary method calls without same-name fields remains unchanged.
- Explicit class-body duplicate instance member declarations remain governed by the current checker rules. This feature targets dynamic `self.<name> = ...` field assignment in methods and constructors, not duplicate declared class fields.

## Scope

- Runtime/evaluator object member representation and instance field assignment/read behavior.
- C codegen/runtime object member representation and generated `self.<name>` assignment/read behavior.
- Checker support only if current static checks reject or misclassify valid same-name method/field use through `self.<name> = ...`.
- Tests covering evaluator and generated-C execution of same-name field/method classes.
- CLI or testscript coverage for the issue reproduction case.

## Out of Scope

- Renaming existing stdlib workaround APIs back to names such as `status()` or `url()`.
- Changing class-body duplicate member validation for explicit field declarations.
- Adding new syntax for private fields, getters, properties, or accessors.
- Changing visibility rules for `private` methods or fields.
- Changing method lookup precedence for names that are not also assigned through `self.<name> = ...`.
- Changing interface method or field declaration semantics beyond preserving existing behavior.

## Acceptance Criteria

- The issue #13 reproduction program runs and prints `200`.
- A constructor can call another instance method before or after assigning a same-name field.
- A same-name method can read the field value through `self.<name>` and can still be called through `object.<name>()`.
- Reassigning `self.<name>` after construction updates the field value without making `object.<name>()` non-callable.
- Existing duplicate explicit class member tests continue to pass.
- The behavior is consistent between evaluator execution and generated-C execution.
- The self-host v01 invariant is not regressed.

## Verification

```sh
go test ./internal/eval ./internal/codegen ./internal/checker -count=1
go test ./tests -run 'TestV06Scripts|TestV44Scripts|TestSelfhostV01Scripts' -count=1
```
