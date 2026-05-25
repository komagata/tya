# Feature: macOS 26.5 Runtime Portability

## Goal

`tya run`, `tya build`, and `tya test` should compile the Tya C runtime on macOS with the Xcode macOS 26.5 SDK, so real projects such as `komagata/flakewatch` can run their Tya test suites locally without delegating verification to Linux CI.

## Context

GitHub issue #19 reports that `tya test tests` in `komagata/flakewatch` fails before project tests execute on macOS with SDK path `/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX26.5.sdk`.

The observed failures come from runtime C compilation, including undeclared POSIX/GNU-style functions and missing networking constants:

- `mkdtemp`
- `mkstemps`
- `timegm`
- `syscall`
- `NI_MAXHOST`
- `NI_MAXSERV`

The runtime sources involved are primarily `runtime/tya_runtime.c` and `runtime/tya_http_server.c`. The CLI native build path in `cmd/tya/main.go` compiles generated C with the runtime sources and platform-specific linker flags.

The issue also mentions deprecation warnings for `syscall`, `swapcontext`, `setcontext`, `getcontext`, and `makecontext`. Those warnings are not the reported hard failure unless the compiler treats them as errors.

## Behavior

- On macOS with the 26.5 SDK, runtime compilation must not fail because of undeclared declarations for `mkdtemp`, `mkstemps`, `timegm`, or `syscall`.
- On macOS with the 26.5 SDK, runtime compilation must not fail because `NI_MAXHOST` or `NI_MAXSERV` are unavailable from the active headers.
- Socket creation in runtime code should prefer the standard `socket()` API on POSIX/macOS instead of calling `syscall(SYS_socket)` directly.
- Reducing `syscall` dependency is part of this feature, not only papering over missing prototypes.
- Existing Linux, Windows, and other POSIX behavior must remain unchanged.
- `tya test tests` in `komagata/flakewatch` must compile and run on the affected macOS SDK, proving the fix against the original project-level failure.
- Deprecation warnings for `swapcontext`, `setcontext`, `getcontext`, and `makecontext` may remain if they do not stop compilation.
- If deprecation warnings become hard errors under the macOS 26.5 SDK build flags, handle only the minimum required to restore successful compilation.

## Scope

- `runtime/tya_runtime.c`
- `runtime/tya_http_server.c`
- Native C compiler invocation or platform flags in `cmd/tya/main.go` only if needed for macOS SDK compatibility.
- Runtime/build tests that compile generated C against the runtime on macOS.
- Project-level verification in `/Users/komagata/Projects/komagata/flakewatch`.

## Out of Scope

- Replacing the task/fiber implementation or removing `ucontext` APIs entirely.
- Redesigning scheduler, task, channel, HTTP, or socket semantics.
- Changing public Tya stdlib APIs.
- Changing runtime behavior for Linux or Windows beyond preserving existing tests.
- Adding a new runtime abstraction layer unless the existing runtime code needs a small local helper to avoid direct `syscall` usage.
- Treating non-fatal deprecation warnings as release blockers.

## Acceptance Criteria

- A minimal Tya program can be run or built on macOS with the 26.5 SDK without undeclared-function or missing-constant runtime C errors.
- Runtime socket creation no longer depends on direct `syscall(SYS_socket)` on macOS/POSIX where `socket()` is available.
- `runtime/tya_runtime.c` and `runtime/tya_http_server.c` compile on macOS with the affected SDK.
- Existing runtime, HTTP, socket, filesystem temp, and time behavior remain compatible with current tests.
- `/Users/komagata/Projects/komagata/flakewatch` passes `tya test tests` locally on the macOS 26.5 SDK environment described in issue #19.
- Full repository verification still passes, including the self-host invariant.

## Verification

```sh
gofmt -w <changed-go-files>
go test ./... -count=1
```

On the affected macOS machine:

```sh
tya run examples/hello.tya
tya build examples/hello.tya -o /tmp/tya-hello
cd /Users/komagata/Projects/komagata/flakewatch
tya test tests
```
