# Feature: Protected Methods

## Goal
Add `protected` visibility for class methods that are part of a class inheritance contract, so subclasses can reuse parent helper methods without exposing those helpers as public API.

## Context
Tya currently supports public class members and `private` class members. `private` members are accessible only from the defining class; subclasses cannot call parent private methods. `protected` is currently rejected as syntax outside the v1.0.0 surface.

This feature adds a Java-style inheritance visibility in a simplified Tya form:

```text
public    -> accessible from any valid call site
protected -> accessible from the defining class and descendant classes
private   -> accessible only from the defining class
```

Unlike Java, Tya `protected` does not grant same-package access. Unlike Ruby, it does not allow arbitrary same-hierarchy instance-to-instance protected calls.

## Behavior
- `protected` may be used on instance methods.

```tya
class Parent
  protected normalize: value ->
    value.to_s()

class Child extends Parent
  label: value ->
    normalize(value)
```

- `protected` may be used on static methods.

```tya
class Parent
  protected static build_label: value ->
    "label:{value}"

class Child extends Parent
  label: value ->
    build_label(value)
```

- A protected method is accessible inside the class that declares it.
- A protected method is accessible inside descendant classes.
- A protected instance method may be called from descendant instance methods using the same receiver forms as inherited public instance methods:
  - implicit receiver calls such as `helper(args)`;
  - explicit self receiver calls such as `self.helper(args)`.
- A protected static method may be called from descendant class or instance contexts using the same receiver forms as inherited public static methods where those forms are already valid:
  - implicit calls such as `build_label(args)`;
  - `Self.build_label(args)` or current-class receiver forms where the existing class-call rules allow them.
- A protected method is not accessible from unrelated external code through an instance or class receiver.

```tya
parent = Parent()
parent.normalize("x") # invalid

Parent.build_label("x") # invalid
```

- A protected parent method remains inaccessible from classes that are not descendants, even if they are in the same directory package.
- A protected method may be overridden by a subclass.
- A protected override must not reduce visibility below protected. Overriding a protected method with `private` is invalid.
- Overriding a protected method with public visibility is valid.
- Overriding a public method with `protected` is invalid because it reduces public API visibility.
- `override protected name: ...` and `protected override name: ...` both parse if the current modifier parser supports flexible modifier order; otherwise the formatter should canonicalize to the repository's existing modifier order with `protected` in the same position class as `private`.
- Formatted Syntax treats `protected` like `private` for class member ordering:
  - public members before protected members before private members within the same static/instance category;
  - `initialize` remains first among public instance methods;
  - otherwise existing alphabetical ordering rules apply.
- Diagnostics should distinguish protected access failures from private access failures where practical, for example `protected instance method normalize is not accessible from Outside`.
- LSP diagnostics, hover, completion, document symbols, and go-to-definition recognize protected methods.
- Documentation generation does not include protected methods as public API items unless the existing private-member documentation mode is explicitly requested.

## Scope
- Update `docs/SPEC.md` and `docs/GUIDE.md` to describe `protected` method visibility and distinguish it from Java package access and Ruby protected calls.
- Remove `protected` from the v1 syntax exclusion docs and parser rejection path.
- Add lexer/parser/AST support for `protected` method modifiers on instance and static methods.
- Extend checker class metadata to track protected instance methods and protected static methods.
- Update checker access rules for implicit receiver calls, `self.` calls, `Self.`/class receiver calls, external member access, and inherited method lookup.
- Update override checks so subclasses cannot reduce inherited public/protected visibility.
- Update formatter support and class member ordering for protected methods.
- Update LSP support for diagnostics, symbols, completion, hover, and go-to-definition where existing private/public method paths need extension.
- Update documentation generator behavior so protected methods are not emitted as public API by default.
- Add focused parser, formatter, checker, interpreter, codegen, LSP/doc generator, and testscript coverage where appropriate.

## Out of Scope
- Protected fields.
- Protected class variables.
- Protected constants.
- Protected constructors / `initialize`.
- Same-package protected access.
- Ruby-style protected calls on arbitrary peer instances.
- Visibility modifiers on top-level functions, modules, structs, records, or interfaces.
- New `public` keyword.
- Changing existing `private` semantics.
- Changing public method dispatch semantics outside protected-access checks.

## Acceptance Criteria
- A class can declare `protected normalize: value -> ...`.
- A class can declare `protected static build_label: value -> ...`.
- The declaring class can call its own protected instance method with implicit receiver and `self.` receiver.
- A subclass can call an inherited protected instance method with implicit receiver and `self.` receiver.
- A subclass can call an inherited protected static method using the existing valid static call forms.
- External code cannot call a protected instance method through `obj.method(...)`.
- External code cannot call a protected static method through `Class.method(...)`.
- A non-descendant class in the same directory package cannot call another class's protected method.
- A subclass may override a protected method as protected.
- A subclass may override a protected method as public.
- A subclass may not override a protected method as private.
- A subclass may not override a public method as protected.
- Existing private behavior remains unchanged: parent private methods are still inaccessible from subclasses.
- Existing public method behavior remains unchanged.
- `protected` is no longer rejected as excluded v1 syntax when used as a class method modifier.
- `protected` remains invalid where the feature does not allow it, including fields, constants, constructors, top-level bindings, interfaces, structs, and records.
- Formatter output for protected methods is stable and idempotent.
- LSP and documentation generator behavior remains consistent with the checker and public API rules.

## Verification
```sh
gofmt -w internal/**/*.go cmd/**/*.go tests/**/*.go
go test ./internal/lexer ./internal/parser ./internal/checker ./internal/eval ./internal/codegen ./internal/formatter -count=1
go test ./tests -run 'protected|private|override|TestSelfhostV01Scripts' -count=1
go test ./... -count=1
```
