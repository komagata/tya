# Feature: stdlib Environment and Process Contract

## Goal

Complete the v1.0.0 environment and process standard-library contract so CLI
tools can inspect and control environment variables, working directories, child
processes, stdin/stdout/stderr, exit status, and timeouts predictably.

## Context

Tya already has public builtins `args()`, `env(name)`, and `exit(status)`, plus
`os/Os` and `process/Process` wrappers. `Process.run(command, options)` exists,
but the v1 public contract should make environment handling and child process
behavior explicit.

The accepted direction is to keep low-level builtins small and expose the v1
user-facing API through `os/Os` and `process/Process`.

## Behavior

- `os.Os.args()` returns process arguments.
- `os.Os.env(name)` returns the environment variable value or `nil`.
- `os.Os.environ()` returns a dictionary of current environment variables.
- `os.Os.setenv(name, value)` sets an environment variable for the current
  process and future child processes.
- `os.Os.unsetenv(name)` removes an environment variable for the current
  process and future child processes.
- `os.Os.cwd()` returns the current working directory.
- `os.Os.chdir(path)` changes the current working directory.
- `os.Os.exit(code)` exits with the integer-compatible status code.
- `process.Process.run(command, options = {})` runs a child process and returns
  a result dictionary.
  - `command` may be a string or an array of strings.
  - A string command runs through the platform shell only when
    `options["shell"] == true`.
  - An array command executes directly without shell interpretation.
  - Supported options:
    - `cwd`: working directory string;
    - `env`: dictionary of environment overrides;
    - `clear_env`: bool, default `false`;
    - `stdin`: string or bytes input;
    - `capture_stdout`: bool, default `true`;
    - `capture_stderr`: bool, default `true`;
    - `timeout`: duration value or number of seconds;
    - `shell`: bool, default `false`.
  - Unknown option keys are invalid.
  - Result dictionaries contain:
    - `status`: numeric exit status;
    - `success`: bool;
    - `stdout`: string or bytes depending on output mode;
    - `stderr`: string or bytes depending on output mode;
    - `timed_out`: bool.
- `process.Process.exec(command, options = {})` replaces the current process
  where the platform supports exec.
  - Unsupported platforms raise a structured process error.
- Environment values are strings.
  - Non-string names or values are invalid.
  - Environment variable names containing NUL bytes are invalid.
- Text capture validates UTF-8.
  - Binary capture is added later only through an explicit option if needed.
- Process failures raise structured errors with `kind: "process"` for spawn
  failures, invalid options, unsupported operations, and timeout setup errors.
  A child process exiting non-zero is not itself a raised error; it is reported
  in the result dictionary.

## Scope

- `stdlib/os/Os.tya`
- `stdlib/process/Process.tya`
- runtime-backed OS/process intrinsics for interpreter and generated C
- `docs/SPEC.md`
- `docs/STRICT_SEMANTICS.md`
- CLI/testscript fixtures for process behavior
- Linux, macOS, and Windows behavior

## Out of Scope

- Shell task language changes.
- Implicit shell execution for array commands.
- Streaming subprocess APIs.
- PTY support.
- Daemon/process supervisor APIs.
- Binary stdout/stderr capture unless accepted by a later spec.

## Acceptance Criteria

- `docs/SPEC.md` documents the environment and process v1 contract.
- `Os.environ`, `Os.setenv`, and `Os.unsetenv` work in interpreter and
  generated C.
- `Process.run` supports direct commands, optional shell commands, working
  directory, environment overrides, stdin, stdout/stderr capture, and timeout.
- Non-zero child exit returns a result dictionary and does not raise.
- Spawn/setup failures, invalid options, invalid env values, timeout failures,
  and unsupported exec raise structured process errors with stable codes.
- Direct array commands do not perform shell interpolation.
- Existing self-host fixed-point gates remain valid.

## Tests To Add

Eval/runtime tests:

- `TestRunOsEnvironmentMutation`
  - Uses `environ`, `setenv`, `env`, and `unsetenv`.
  - Expected: current process and child process environment reflect changes.

- `TestRunProcessRunDirectCommand`
  - Runs an array command with no shell.
  - Expected: stdout/stderr/status/success fields are correct.

- `TestRunProcessRunShellOptIn`
  - Runs a string command with and without `shell: true`.
  - Expected: shell interpretation only occurs when explicitly enabled.

- `TestRunProcessRunCwdEnvStdinTimeout`
  - Covers cwd, env override, stdin text, and timeout behavior.

- `TestRunProcessStructuredErrors`
  - Invalid option key, invalid env values, missing executable, and unsupported
    exec operation.
  - Expected: structured process errors with stable codes.

Codegen tests:

- `TestEmitCEnvironmentAndProcessProgram`
  - Builds and runs environment mutation and child process fixtures.

Testscript coverage:

- `v1_stdlib_environment_process.txtar`
  - Covers CLI-level behavior for environment and process helpers.

Documentation tests:

- `TestSpecDocumentsEnvironmentProcessContract`
  - Expected: `docs/SPEC.md` documents process result fields, shell opt-in,
    env override semantics, and non-zero exit behavior.

## Verification

```sh
go test ./internal/eval -run 'Os|Process|Environment' -count=1
go test ./internal/codegen -run 'Os|Process|Environment' -count=1
go test ./tests -run 'TestV.*Scripts|TestSpecDocumentsEnvironmentProcessContract|TestSelfhostV01Scripts|TestSelfhostV02Scripts' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
