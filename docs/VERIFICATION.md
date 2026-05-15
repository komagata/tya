---
layout: doc
title: Verification Commands
permalink: /verification/
---

# Tya Verification Command Specification

This document defines the shared behavior for Tya verification commands.

Verification commands inspect source code and report whether it satisfies a
specific contract. They do not define language syntax by themselves. Language
syntax and standard-library behavior remain defined by `docs/SPEC.md`.

## Command Scope

The verification command family includes:

- `tya format --check`
- `tya check`
- future `tya lint`
- future `tya test`
- future `tya verify`

`tya run` and `tya build` may share diagnostics and exit-code conventions with
verification commands, but they are execution and build commands, not
verification commands.

## Command Roles

### `tya format --check`

`tya format --check` checks whether source files already match the canonical
Tya source formatting.

It answers this question:

```text
Would `tya format` change this file?
```

It does not decide whether a program is valid Tya. A file can be formatted but
still fail parsing or checking.

Examples of issues in scope:

- CRLF or CR line endings instead of LF
- trailing spaces or tabs
- tabs in indentation
- extra blank lines at the end of a file
- missing final newline
- future canonical spacing or indentation rules once the formatter supports
  them

`tya format --check` must not rewrite files.

### `tya check`

`tya check` checks whether source files are valid Tya programs according to the
compiler front end.

It answers this question:

```text
Can Tya accept this program before C emission or execution?
```

The phrase "how far check looks" means which compiler phases are included in
`tya check`.

`tya check` includes:

- lexical analysis
- parsing
- semantic checking
- module loading needed to check the requested program

`tya check` excludes:

- C code emission
- C compiler invocation
- executable creation
- program execution
- unit test execution
- lint rules for stylistic or project-specific conventions

Examples of issues in scope:

- invalid tokens
- unterminated strings
- invalid indentation or block structure
- parser errors
- undefined names when the checker can detect them
- duplicate declarations rejected by the checker
- invalid class, interface, override, or `super(...)` usage
- invalid module, class, or member access rejected by the checker

### Future `tya lint`

`tya lint` checks rules that are not required for a program to be valid Tya.

It answers this question:

```text
Does this valid Tya program follow the selected lint rules?
```

`tya check` reports invalid Tya programs. `tya lint` reports code that is valid
Tya but violates linter rules.

Examples of possible lint rules:

- unused variables
- unused imports
- shadowed names
- unreachable code
- project-specific naming rules
- project-specific forbidden APIs
- project-specific module layout rules

Lint rules may be built in, configured by a project, or added later by tooling.
The linter must not redefine language validity. A program that fails only lint
rules is still a valid Tya program.

### Future `tya test`

`tya test` runs Tya tests.

It should be finalized after the unittest standard library is designed. The
test command should be the execution entry point for unittest-based tests, not a
separate competing test model.

`tya test` should report:

- passed tests
- failed assertions
- skipped tests, if the unittest library supports them
- runtime errors while running tests
- test discovery errors

### Future `tya verify`

`tya verify` runs the standard verification pipeline for a target.

The default order is:

```text
format -> check -> lint -> test
```

Initial implementations may support only the commands that exist at the time.
For example, before `lint` and unittest-based `test` are finalized, `tya verify`
may run only:

```text
format --check -> check
```

The command order is intentional:

1. `format --check` reports mechanical source-shape drift first.
1. `check` reports invalid Tya programs before optional rules run.
1. `lint` reports rules for valid Tya programs.
1. `test` runs code only after source formatting, language validity, and lint
   rules have passed.

## Exit Codes

Verification commands should use stable exit codes:

```text
0: verification passed
1: verification failed
2: command usage error
3: internal tool error
```

Examples:

- formatting differences found by `tya format --check`: `1`
- parser or checker errors found by `tya check`: `1`
- lint violations found by `tya lint`: `1`
- failing tests found by `tya test`: `1`
- unknown flag or missing required argument: `2`
- compiler panic or unexpected tool failure: `3`

## Target Selection

Verification commands should accept explicit file and directory targets.

When a directory is provided, the command should recursively select `.tya`
source files that are meaningful for that command.

When no target is provided, the command should use the current directory as the
default target unless the individual command has a stronger existing convention.

Future project-root behavior may refine default target selection, but it should
not change the meaning of an explicit file target.

## Human Output

Human-readable output should be concise by default.

Successful single-target verification may print nothing or a short success
message, depending on the command.

Failures should include:

- command name
- file path
- line and column when available
- short rule or diagnostic name when available
- actionable message

Multi-file commands should continue after ordinary verification failures where
practical, then report a summary.

Usage errors and internal tool errors may stop immediately.

## Reserved Options

The following options are reserved for consistent future behavior:

- `--quiet`
- `--verbose`
- `--json`

`--quiet` should minimize human output while preserving exit codes.

`--verbose` should include additional diagnostic context useful for debugging
the tool invocation.

`--json` should produce machine-readable output for editors, CI, and other
tools. JSON output should preserve the same pass/fail meaning and exit codes as
human-readable output.

## Fixing Behavior

Verification commands should distinguish checking from rewriting.

`tya format` may rewrite files.

`tya format --check`, `tya check`, `tya lint`, `tya test`, and `tya verify`
should not rewrite files by default.

If future lint rules support automatic fixes, that behavior should require an
explicit option such as `--fix`.

## Relationship To CI

The intended CI verification sequence is:

```sh
tya verify
```

Until `tya verify` exists, CI can run the equivalent supported commands
directly:

```sh
tya format --check .
tya check .
```

After `tya lint` and unittest-based `tya test` exist, they should become part of
the default `tya verify` pipeline in the documented order.
