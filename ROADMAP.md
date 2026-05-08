# Tya Roadmap

`ROADMAP.md` is the single source of truth for current TODO, TASK, and roadmap
planning.

Pre-v0.1 planning documents and self-host migration notes are archived under
[`docs/archive/pre-v0.1/`](docs/archive/pre-v0.1/). They are historical
references, not current language or implementation authority.

## Self-Host Invariant

The Tya-written compiler fixed point is a maintained invariant. Later language,
runtime, CLI, stdlib, and documentation work must not regress
`selfhost/v01/compiler.tya`.

Required evidence:

```sh
go test ./tests -run TestSelfhostV01Scripts -count=1
```

This gate proves that the Tya-written compiler can compile itself to stable
stage-2/stage-3 C output, and that the self-hosted stage-2 compiler can compile
and run representative programs through the maintained v0.5 surface.

## Current Direction

Tya v0.5 is implemented as a small compile-to-C language. The frozen release
documents are:

1. [`docs/v0.1.0/SPEC.md`](docs/v0.1.0/SPEC.md)
1. [`docs/v0.1.0/API.md`](docs/v0.1.0/API.md)
1. [`docs/v0.2.0/SPEC.md`](docs/v0.2.0/SPEC.md)
1. [`docs/v0.2.0/API.md`](docs/v0.2.0/API.md)

Tya uses semantic versioning. Specification changes happen at the minor version
level, such as `v0.3` and `v0.5`. Patch releases such as `v0.3.1` must not
change language or standard-library semantics. In other words, the `x` in
`0.0.x` is never a specification-change unit. Therefore, specification
documents use minor-version labels such as `v0.3`.

Latest editable documentation is:

1. [`docs/SPEC.md`](docs/SPEC.md)
1. [`docs/API.md`](docs/API.md)
1. [`docs/STDLIB.md`](docs/STDLIB.md)
1. [`docs/NAMING.md`](docs/NAMING.md)

Current planned minor-version documents are:

1. [`docs/v0.3/SPEC.md`](docs/v0.3/SPEC.md)
1. [`docs/v0.3/STDLIB.md`](docs/v0.3/STDLIB.md)
1. [`docs/v0.4/SPEC.md`](docs/v0.4/SPEC.md)
1. [`docs/v0.5/SPEC.md`](docs/v0.5/SPEC.md)
1. [`docs/v0.6/SPEC.md`](docs/v0.6/SPEC.md)
1. [`docs/v0.7/SPEC.md`](docs/v0.7/SPEC.md)
1. [`docs/v0.8/SPEC.md`](docs/v0.8/SPEC.md)
1. [`docs/v0.9/SPEC.md`](docs/v0.9/SPEC.md)
1. [`docs/v0.10/SPEC.md`](docs/v0.10/SPEC.md)

The v0.5 reference implementation remains:

```text
Go lexer
Go parser
Go AST
Go checker
Go C emitter
C runtime
v0.5 specification tests
```

Go interpreter behavior, ASTMODE, and legacy archived node-string experiments
are not v0.5 authority. The maintained `selfhost/v01/compiler.tya` fixed point
is still required not to regress.

## Implementation Tooling Policy

The v0.5 compiler implementation should stay hand-written:

```text
Go lexer
Go parser
Go AST
Go checker
Go C emitter
```

Do not add a parser generator or large grammar framework for v0.5. In
particular, avoid introducing Participle, goyacc, Pigeon, ANTLR, or Tree-sitter
as compiler-front-end authority. They may be useful references or future editor
tooling, but the active compiler path should remain explicit Go code.

After the Go implementation reaches a complete lexer, parser, AST, checker, and
C emitter for the current specification, continue self-host work in the same
component order:

```text
Tya lexer
Tya parser
Tya AST
Tya checker
Tya C emitter
```

Each Tya component must preserve the self-host fixed point before moving to the
next component.

Use small test-support dependencies where they make the v0.5 specification
easier to verify:

```text
github.com/google/go-cmp/cmp
github.com/rogpeppe/go-internal/testscript
```

Use `go-cmp` for readable token, AST, diagnostic, and generated-output diffs.
Use `testscript` for CLI-level specification tests, especially `tya run`,
`tya build`, expected stdout/stderr, and negative examples.

## Current Roadmap

- [x] Ship v0.3 standard attached libraries
  - [x] Define v0.3 attached library scope
    - [x] Decide that JSON and CSV parsers are deferred from v0.3.
    - [x] Keep JSON and CSV out of builtins and out of initial stdlib scope.
    - [x] Specify that v0.3 adds attached libraries, not a package manager.
    - [x] Document v0.3 scope in `docs/SPEC.md` and `docs/STDLIB.md`.
  - [x] Add stdlib import search
    - [x] Add a `stdlib/` directory for shipped `.tya` modules.
    - [x] Search stdlib after the importing file's directory and `TYA_PATH`.
    - [x] Keep user modules and `TYA_PATH` entries higher priority than stdlib.
    - [x] Keep module file name and `module` declaration matching rules.
    - [x] Add tests for same-directory, `TYA_PATH`, and stdlib precedence.
  - [x] Package stdlib with installed Tya
    - [x] Make installed `tya` find `share/tya/stdlib` outside the source checkout.
    - [x] Install `stdlib/*` from the Homebrew Formula.
    - [x] Add an installed-layout test for runtime plus stdlib lookup.
  - [x] Add initial lightweight stdlib modules
    - [x] Add `stdlib/string.tya`.
    - [x] Add `string.blank(text)`.
    - [x] Add `string.present(text)`.
    - [x] Add `stdlib/array.tya`.
    - [x] Add `array.empty(items)`.
    - [x] Add `array.first(items)`.
    - [x] Add tests and examples for every initial stdlib function.
  - [x] Keep v0.3 documentation and release snapshots aligned
    - [x] Update latest `docs/SPEC.md` and `docs/STDLIB.md` when v0.3 behavior is implemented.
    - [x] Keep `docs/v0.3/` aligned with the v0.3 minor specification.
    - [x] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [x] Create a patch-tag snapshot only when an exact release archive needs one.
    - [x] Update README install, run, development, and documentation sections for v0.3.
- [x] Ship v0.4 testing and script confidence
  - [x] Decide that v0.4 focuses on tests instead of expanding stdlib.
  - [x] Keep native-backed stdlib, JSON, and CSV out of v0.4.
  - [x] Document v0.4 direction in `docs/v0.4/SPEC.md`.
  - [x] Add `tya test`.
    - [x] With no argument, discover `*_test.tya` under the current directory.
    - [x] With a directory argument, discover `*_test.tya` under that directory.
    - [x] With a file argument, run that file only.
    - [x] Exit non-zero when any test file fails.
  - [x] Add assertions.
    - [x] Add `assert value`.
    - [x] Add `assert_equal expected, actual`.
    - [x] Use deep equality for `assert_equal`.
    - [x] Emit source-oriented assertion diagnostics.
  - [x] Add stdlib tests as first-class examples.
    - [x] Add `tests/stdlib_string_test.tya`.
    - [x] Add `tests/stdlib_array_test.tya`.
    - [x] Ensure stdlib tests run through `tya test`.
  - [x] Keep v0.4 documentation and release snapshots aligned.
    - [x] Update latest docs when v0.4 behavior is implemented.
    - [x] Keep `docs/v0.4/` aligned with the v0.4 minor specification.
    - [x] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [x] Create a patch-tag snapshot only when an exact release archive needs one.
    - [x] Update README install, run, development, and documentation sections for v0.4.
- [x] Ship v0.5 minimal classes and objects
  - [x] Define v0.5 class scope
    - [x] Add `docs/v0.5/SPEC.md`.
    - [x] Specify `class Name`, constructor calls, `init`, `@field` instance fields, methods, and module class access.
    - [x] Reserve `@@field` for future class variables and keep it invalid in v0.5.
    - [x] Exclude inheritance, `super`, interfaces, class methods, class fields, and visibility from v0.5.
    - [x] Keep dictionary member access with `dict.key` out of v0.5.
  - [x] Add class syntax to the compiler front end
    - [x] Add lexer/parser support for class declarations.
    - [x] Add AST nodes for class declarations, methods, object field access, and object field assignment.
    - [x] Add checker diagnostics for class naming, duplicate methods, invalid `@field`, invalid `@@field`, and invalid dot access.
  - [x] Add class runtime and C emission
    - [x] Emit object construction through `ClassName(args...)`.
    - [x] Call `init` during construction and ignore its explicit return value.
    - [x] Support public instance field read/write through dot access.
    - [x] Support instance method calls with `object.method(args...)`.
  - [x] Keep modules and self-host compatible
    - [x] Expose classes declared inside modules as PascalCase module members.
    - [x] Preserve existing module member access behavior.
    - [x] Preserve the `selfhost/v01/compiler.tya` fixed point.
  - [x] Keep v0.5 documentation and tests aligned
    - [x] Update latest docs when v0.5 behavior is implemented.
    - [x] Keep `docs/v0.5/` aligned with the v0.5 minor specification.
    - [x] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [x] Add compiler, runtime, module, and negative tests for v0.5 classes.
    - [x] Update README examples when v0.5 is implemented.
- [x] Ship v0.6 class-level members and field defaults
  - [x] Define v0.6 class-level member scope
    - [x] Add `docs/v0.6/SPEC.md`.
    - [x] Specify `@@field` class variables.
    - [x] Specify `@@method = args ->` class methods.
    - [x] Specify `field = value` instance field defaults.
    - [x] Specify public class variable and class method access through `ClassName.member`.
    - [x] Keep inheritance, `super`, interfaces, visibility, and private class members out of v0.6.
    - [x] Keep dictionary member access with `dict.key` out of v0.6.
  - [x] Add class variables to the compiler front end
    - [x] Add lexer/parser support for `@@field` class member declarations.
    - [x] Add AST nodes for class variable declaration, read, and assignment.
    - [x] Add checker diagnostics for invalid `@@field` usage and duplicate class members.
  - [x] Add instance field defaults to the compiler front end
    - [x] Parse class body `field = value` as an instance field default.
    - [x] Add AST nodes for instance field defaults.
    - [x] Reject duplicate instance member names across field defaults and methods.
  - [x] Add class methods to the compiler front end
    - [x] Add parser support for `@@method = args ->` declarations.
    - [x] Add AST nodes for class method declarations and class method calls.
    - [x] Reject `@field` inside class methods.
  - [x] Add class-level runtime and C emission
    - [x] Initialize class variables once when the class is defined.
    - [x] Copy instance field defaults into each new object before `init` runs.
    - [x] Support class variable read/write through `ClassName.field`.
    - [x] Support class methods through `ClassName.method(args...)`.
    - [x] Support class variables from instance methods.
    - [x] Support module class access such as `module_name.ClassName.method(...)`.
  - [x] Keep modules and self-host compatible
    - [x] Preserve v0.5 instance class behavior.
    - [x] Preserve existing module member access behavior.
    - [x] Preserve the `selfhost/v01/compiler.tya` fixed point.
  - [x] Keep v0.6 documentation and tests aligned
    - [x] Update latest docs when v0.6 behavior is implemented.
    - [x] Keep `docs/v0.6/` aligned with the v0.6 minor specification.
    - [x] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [x] Add compiler, runtime, module, and negative tests for v0.6 class-level members and field defaults.
    - [x] Update README examples when v0.6 is implemented.
- [x] Ship v0.7 single inheritance
  - [x] Define v0.7 inheritance scope
    - [x] Add `docs/v0.7/SPEC.md`.
    - [x] Specify `class Child extends Parent`.
    - [x] Specify `super(args...)` in `init` and overridden instance methods.
    - [x] Specify inherited instance field defaults and instance methods.
    - [x] Keep class variable inheritance, class method inheritance, interfaces, and mixins out of v0.7.
  - [x] Add inheritance to the compiler front end
    - [x] Parse `extends` clauses with local and module-qualified parent classes.
    - [x] Add AST fields for parent class references.
    - [x] Detect unknown parent classes and inheritance cycles.
  - [x] Add inherited instance behavior
    - [x] Apply parent field defaults before child field defaults.
    - [x] Inherit instance methods from parent classes.
    - [x] Support instance method overriding with matching arity.
  - [x] Add `super`
    - [x] Support `super(args...)` in subclass `init`.
    - [x] Support `super(args...)` in overridden instance methods.
    - [x] Reject `super` outside `init` and instance methods.
    - [x] Reject `super` inside class methods.
  - [x] Keep class-level inheritance out of scope
    - [x] Keep `@@field` class variables local to the declaring class.
    - [x] Keep `@@method` class methods local to the declaring class.
    - [x] Reject or report missing subclass class-level member access when only the parent declares it.
  - [x] Keep v0.7 documentation and tests aligned
    - [x] Update latest docs when v0.7 behavior is implemented.
    - [x] Keep `docs/v0.7/` aligned with the v0.7 minor specification.
    - [x] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [x] Add compiler, runtime, module, and negative tests for v0.7 inheritance.
    - [x] Preserve the `selfhost/v01/compiler.tya` fixed point.
- [x] Ship v0.8 class-level inheritance
  - [x] Define v0.8 class-level inheritance scope
    - [x] Add `docs/v0.8/SPEC.md`.
    - [x] Specify inherited class variables.
    - [x] Specify inherited class methods.
    - [x] Specify subclass-local class variable shadowing on assignment.
    - [x] Specify `self` inside class methods as the receiving class.
    - [x] Specify `super(args...)` inside overridden class methods.
    - [x] Specify `object.class` and `object.class_name`.
    - [x] Specify `ClassName.name` and `ClassName.parent`.
  - [x] Add inherited class variable lookup
    - [x] Resolve `ClassName.field` through the class inheritance chain.
    - [x] Resolve `@@field` in class methods from the receiving class.
    - [x] Resolve `@@field` in instance methods from the instance's class.
    - [x] Create or update subclass-owned class variables on subclass assignment.
  - [x] Add inherited class method lookup
    - [x] Resolve `ClassName.method(args...)` through the class inheritance chain.
    - [x] Bind inherited class methods to the receiving class.
    - [x] Support class method overriding with matching arity.
  - [x] Add class-method `self` and `super`
    - [x] Support `self` inside class methods.
    - [x] Reject `self` inside instance methods.
    - [x] Support `super(args...)` inside overridden class methods.
    - [x] Reject class-method `super` when no parent class method exists.
  - [x] Add small class introspection
    - [x] Support `object.class` as the object's actual class.
    - [x] Support `object.class_name` as the object's actual class name string.
    - [x] Support `ClassName.name` as the class name string.
    - [x] Support `ClassName.parent` as the parent class or `nil`.
    - [x] Reject assignment to read-only introspection members.
  - [x] Keep v0.8 documentation and tests aligned
    - [x] Update latest docs when v0.8 behavior is implemented.
    - [x] Keep `docs/v0.8/` aligned with the v0.8 minor specification.
    - [x] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [x] Add compiler, runtime, module, and negative tests for v0.8 class-level inheritance and introspection.
    - [x] Preserve the `selfhost/v01/compiler.tya` fixed point.
- [x] Ship v0.9 class visibility and private members
  - [x] Define v0.9 private member scope
    - [x] Add `docs/v0.9/SPEC.md`.
    - [x] Specify private instance fields with `@_field`.
    - [x] Specify private instance methods with `_method = args ->`.
    - [x] Specify private class variables with `@@_field`.
    - [x] Specify private class methods with `@@_method = args ->`.
    - [x] Specify private constructors with `_init`.
    - [x] Specify `abstract class Name` as directly non-constructible.
    - [x] Keep protected visibility, visibility keywords, interfaces, abstract methods, and mixins out of v0.9.
  - [x] Add private instance member checks
    - [x] Reject external access to private instance fields.
    - [x] Reject external calls to private instance methods.
    - [x] Allow private instance access from methods declared in the same class.
    - [x] Reject subclass direct access to parent private instance members.
  - [x] Add private class member checks
    - [x] Reject external access to private class variables.
    - [x] Reject external calls to private class methods.
    - [x] Allow private class access from methods declared in the same class.
    - [x] Reject subclass direct access to parent private class members.
  - [x] Keep inheritance and introspection compatible
    - [x] Treat subclass private members with the same name as parent private members as separate members.
    - [x] Reject `super` calls that target private parent methods.
    - [x] Reject `super` calls that target parent `_init`.
    - [x] Keep v0.8 introspection from exposing private member lists.
  - [x] Add constructor visibility and abstract class checks
    - [x] Support `_init` as a private constructor.
    - [x] Reject external construction of classes with `_init`.
    - [x] Allow construction from methods declared in the same class.
    - [x] Reject classes declaring both `init` and `_init`.
    - [x] Parse `abstract class Name`.
    - [x] Reject direct construction of abstract classes.
    - [x] Allow construction of non-abstract subclasses of abstract classes.
  - [x] Keep v0.9 documentation and tests aligned
    - [x] Update latest docs when v0.9 behavior is implemented.
    - [x] Keep `docs/v0.9/` aligned with the v0.9 minor specification.
    - [x] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [x] Add compiler, runtime, module, and negative tests for v0.9 private members, `_init`, and abstract classes.
    - [x] Preserve the `selfhost/v01/compiler.tya` fixed point.
- [x] Ship v0.10 abstract methods and final classes
  - [x] Define v0.10 abstract method and final class scope
    - [x] Add `docs/v0.10/SPEC.md`.
    - [x] Specify `abstract method = args ->`.
    - [x] Specify `abstract @@method = args ->`.
    - [x] Specify concrete subclass implementation checks.
    - [x] Specify `final class Name`.
    - [x] Keep interfaces, `implements`, abstract fields, final methods, sealed classes, base classes, type annotations, and generics out of v0.10.
  - [x] Add abstract method parsing and checking
    - [x] Parse abstract instance method declarations without bodies.
    - [x] Parse abstract class method declarations without bodies.
    - [x] Reject abstract methods outside abstract classes.
    - [x] Reject abstract methods with bodies.
  - [x] Add abstract implementation checks
    - [x] Require concrete subclasses to implement inherited abstract instance methods.
    - [x] Require concrete subclasses to implement inherited abstract class methods.
    - [x] Allow abstract subclasses to leave abstract methods unimplemented.
    - [x] Check implementation arity against abstract method arity.
  - [x] Add final class checks
    - [x] Parse `final class Name`.
    - [x] Reject extending final classes.
    - [x] Reject classes declared as both abstract and final.
  - [x] Keep v0.10 documentation and tests aligned
    - [x] Update latest docs when v0.10 behavior is implemented.
    - [x] Keep `docs/v0.10/` aligned with the v0.10 minor specification.
    - [x] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [x] Add compiler, runtime, module, and negative tests for v0.10 abstract methods and final classes.
    - [x] Preserve the `selfhost/v01/compiler.tya` fixed point.
- [x] Ship v0.11 explicit interfaces
  - [x] Define v0.11 explicit interface scope
    - [x] Add `docs/v0.11/SPEC.md`.
    - [x] Specify `interface Name`.
    - [x] Specify `class Name implements InterfaceName`.
    - [x] Specify multiple interfaces.
    - [x] Specify `extends` with `implements`.
    - [x] Specify concrete and abstract class implementation checks.
    - [x] Keep implicit interfaces, class-as-interface conformance, interface fields, interface class methods, interface inheritance, default interface methods, type annotations, and generics out of v0.11.
  - [x] Add interface parsing and checking
    - [x] Parse interface declarations.
    - [x] Parse body-free interface method requirements.
    - [x] Reject invalid members inside interface bodies.
    - [x] Parse `implements` lists.
    - [x] Reject `implements` targets that are not interfaces.
  - [x] Add interface implementation checks
    - [x] Require concrete classes to implement required interface methods.
    - [x] Allow inherited instance methods to satisfy interface requirements.
    - [x] Allow abstract classes to leave interface methods unimplemented.
    - [x] Check implementation arity against interface method arity.
    - [x] Reject conflicting interface method arity requirements.
  - [x] Keep v0.11 documentation and tests aligned
    - [x] Update latest docs when v0.11 behavior is implemented.
    - [x] Keep `docs/v0.11/` aligned with the v0.11 minor specification.
    - [x] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [x] Add compiler, runtime, module, and negative tests for v0.11 interfaces.
    - [x] Preserve the `selfhost/v01/compiler.tya` fixed point.
- [x] Ship v0.12 interface inheritance
  - [x] Define v0.12 interface inheritance scope
    - [x] Add `docs/v0.12/SPEC.md`.
    - [x] Specify `interface Child extends Parent`.
    - [x] Specify multiple interface inheritance.
    - [x] Specify transitive interface inheritance.
    - [x] Specify inherited interface implementation checks.
    - [x] Specify interface inheritance cycle errors.
    - [x] Specify conflict diagnostics for incompatible method requirements.
    - [x] Keep class-as-interface conformance, classes extending interfaces, interfaces extending classes, default interface methods, interface fields, interface class methods, type annotations, and generics out of v0.12.
  - [x] Add interface inheritance parsing and checking
    - [x] Parse interface `extends` lists.
    - [x] Resolve parent interface names, including module-qualified names.
    - [x] Reject interfaces extending classes.
    - [x] Reject classes extending interfaces.
    - [x] Reject interface inheritance cycles.
  - [x] Add inherited requirement checks
    - [x] Collect direct and inherited interface method requirements.
    - [x] Treat duplicate method requirements with matching arity as compatible.
    - [x] Reject duplicate method requirements with conflicting arity.
    - [x] Require concrete classes to implement inherited interface requirements.
    - [x] Allow abstract classes to leave inherited interface requirements unimplemented.
  - [x] Improve interface conflict diagnostics
    - [x] Include child interface name in conflict errors.
    - [x] Include conflicting method name in conflict errors.
    - [x] Include parent interface names in conflict errors.
    - [x] Include conflicting arities in conflict errors.
  - [x] Keep v0.12 documentation and tests aligned
    - [x] Update latest docs when v0.12 behavior is implemented.
    - [x] Keep `docs/v0.12/` aligned with the v0.12 minor specification.
    - [x] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [x] Add compiler, runtime, module, and negative tests for v0.12 interface inheritance.
    - [x] Preserve the `selfhost/v01/compiler.tya` fixed point.
- [x] Ship v0.13 override and constructor chaining checks
  - [x] Define v0.13 override and constructor chaining scope
    - [x] Add `docs/v0.13/SPEC.md`.
    - [x] Specify `override method = args ->`.
    - [x] Specify `override @@method = args ->`.
    - [x] Specify override target and arity checks.
    - [x] Keep `override` optional in v0.13.
    - [x] Specify required parent `init` chaining when subclass `init` exists.
    - [x] Specify constructor `super(...)` count, placement, and arity checks.
    - [x] Keep mandatory `override`, final methods, final fields, duplicate method definition errors, default interface methods, type annotations, and generics out of v0.13.
  - [x] Add override parsing and checking
    - [x] Parse `override` instance method declarations.
    - [x] Parse `override` class method declarations.
    - [x] Reject `override` declarations with no inherited class method target.
    - [x] Reject `override` arity mismatches.
    - [x] Reject instance/class method kind mismatches.
    - [x] Reject `override` used only to satisfy interface requirements.
  - [x] Add constructor chaining checks
    - [x] Require subclass `init` to call parent public `init` when it exists.
    - [x] Reject more than one constructor `super(...)` call.
    - [x] Reject constructor `super(...)` when parent public `init` does not exist.
    - [x] Reject instance field assignment before constructor `super(...)`.
    - [x] Reject explicit `return` before constructor `super(...)`.
    - [x] Check constructor `super(...)` arity against parent public `init`.
    - [x] Reject constructor `super(...)` targeting parent `_init`.
  - [x] Keep v0.13 documentation and tests aligned
    - [x] Update latest docs when v0.13 behavior is implemented.
    - [x] Keep `docs/v0.13/` aligned with the v0.13 minor specification.
    - [x] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [x] Add compiler, runtime, module, and negative tests for v0.13 override and constructor chaining checks.
    - [x] Preserve the `selfhost/v01/compiler.tya` fixed point.
- [x] Ship v0.14 destructuring assignment
  - [x] Define v0.14 destructuring assignment scope
    - [x] Add `docs/v0.14/SPEC.md`.
    - [x] Specify array destructuring assignment.
    - [x] Specify dictionary destructuring assignment with explicit string keys.
    - [x] Specify nested destructuring patterns.
    - [x] Specify `_` discard targets.
    - [x] Specify runtime mismatch and missing-key errors.
    - [x] Keep rest destructuring, default values, dictionary key shorthand, function parameter destructuring, `for` destructuring, pattern matching, class object destructuring, type annotations, and generics out of v0.14.
  - [x] Add destructuring parsing and checking
    - [x] Parse array destructuring assignment targets.
    - [x] Parse dictionary destructuring assignment targets.
    - [x] Parse nested destructuring targets.
    - [x] Reject non-string dictionary keys in destructuring patterns.
    - [x] Reject destructuring assignment used as an expression.
  - [x] Add destructuring runtime behavior
    - [x] Assign array elements by position.
    - [x] Assign dictionary values by explicit string key.
    - [x] Ignore `_` discard targets without creating or updating `_`.
    - [x] Evaluate the right-hand expression once before destructuring.
    - [x] Report runtime array length mismatches.
    - [x] Report runtime dictionary missing-key errors.
    - [x] Report runtime nested shape mismatches.
  - [x] Keep v0.14 documentation and tests aligned
    - [x] Update latest docs when v0.14 behavior is implemented.
    - [x] Keep `docs/v0.14/` aligned with the v0.14 minor specification.
    - [x] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [x] Add compiler, runtime, module, and negative tests for v0.14 destructuring assignment.
    - [x] Preserve the `selfhost/v01/compiler.tya` fixed point.
- [x] Ship v0.15 structured error handling
  - [x] Define v0.15 structured error handling scope
    - [x] Add `docs/v0.15/SPEC.md`.
    - [x] Specify `raise expression`.
    - [x] Specify block `try ... catch name ...`.
    - [x] Specify `_` catch discard binding.
    - [x] Specify raised value propagation and re-raise.
    - [x] Keep existing `try expression` behavior unchanged.
    - [x] Keep `finally`, `ensure`, typed catch, multiple catch clauses, catch filters, try/catch expressions, destructuring catch bindings, error class hierarchy, and stack trace API out of v0.15.
  - [x] Add `raise` parsing and runtime behavior
    - [x] Parse `raise expression`.
    - [x] Reject `raise` without an expression.
    - [x] Propagate raised values to the nearest enclosing block `try/catch`.
    - [x] Report uncaught raised values.
  - [x] Add block `try/catch`
    - [x] Parse block `try ... catch name ...`.
    - [x] Reject `catch` without a binding name.
    - [x] Reject `catch` outside block `try`.
    - [x] Reject block `try` without `catch`.
    - [x] Bind caught values only inside the catch block.
    - [x] Treat `_` as a discard catch binding.
    - [x] Allow catch blocks to re-raise.
  - [x] Preserve existing error propagation semantics
    - [x] Keep `try expression` as `value, err` propagation.
    - [x] Ensure `try expression` does not catch raised values.
    - [x] Keep `return`, `break`, and `continue` separate from raised values.
  - [x] Keep v0.15 documentation and tests aligned
    - [x] Update latest docs when v0.15 behavior is implemented.
    - [x] Keep `docs/v0.15/` aligned with the v0.15 minor specification.
    - [x] Regenerate HTML documentation with `node scripts/build_docs_pages.js`.
    - [x] Add compiler, runtime, module, and negative tests for v0.15 structured error handling.
    - [x] Preserve the `selfhost/v01/compiler.tya` fixed point.
## Verification Reference

Default verification:

```sh
go test ./... -count=1
```

Focused verification should prefer tests for the touched lexer, parser, checker,
C emitter, runtime, examples, stdlib, or docs. The self-host fixed-point gate is
part of the maintained project invariant and must stay green.
