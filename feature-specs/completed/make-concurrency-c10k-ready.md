---
status: completed
goal_ready: false
---

# Feature: Make Concurrency C10K-Ready

## Goal

Make Tya concurrency scale to C10K-style workloads by keeping the v0.60
language surface, moving concurrency APIs to class-style value methods, and
replacing one-pthread-per-task semantics with runtime-managed lightweight tasks.

## Context

`docs/v0.60/SPEC.md` defines the target concurrency surface. `spawn`, `await`,
`scope`, and `select` remain language constructs. `channel.Channel`,
`task.Task`, `sync.Mutex`, `sync.AtomicInteger`, and `sync.WaitGroup` are the
public class-style APIs.

The current implementation has much of the syntax and API shape in place, but
the roadmap still tracks the C10K runtime work as incomplete. The end state
must let many blocked Tya tasks wait on channels, timers, `await`, and future
I/O readiness without consuming one OS thread per blocked task.

## Behavior

- Preserve these language constructs:
  - `spawn expr`
  - `await task`
  - `scope`
  - `select`
- Keep `spawn` returning a task value and `await` resolving or re-raising that
  task's result.
- Keep `scope` as the structured-concurrency owner:
  - tasks spawned inside a scope are registered with that scope;
  - leaving the scope waits for every spawned task;
  - a task raise cancels siblings and re-raises the first raised value after
    cleanup;
  - a synchronous body raise cancels children, waits for them, and re-raises.
- Implement class-style value APIs from `docs/v0.60/SPEC.md`:
  - `channel.Channel(capacity)`
  - `c.send(value)`
  - `c.receive()`
  - `c.receive_timeout(seconds)`
  - `c.close()`
  - `c.closed?()`
  - `task.Task.current()`
  - `t.cancel()`
  - `t.cancelled?()`
  - `sync.Mutex()`
  - `m.lock()`
  - `m.unlock()`
  - `m.with_lock(fn)`
  - `sync.AtomicInteger(initial)`
  - `a.add(n)`
  - `a.load()`
  - `a.store(n)`
  - `a.cas(expected, new_value)`
  - `sync.WaitGroup()`
  - `wg.add(n)`
  - `wg.done()`
  - `wg.wait()`
- Remove helper-style concurrency APIs and diagnose them with `[TYA-E0820]`,
  including:
  - `channel.Channel.new(...)`
  - `channel.Channel.send(c, value)`
  - `channel.Channel.receive(c)`
  - `channel.Channel.close(c)`
  - `channel.Channel.select(...)`
  - `task.Task.cancel(t)`
  - `task.Task.cancelled?(t)`
  - `sync.Sync.*`
- Implement `select` without normal-case polling:
  - receive arms;
  - ignored receive arms;
  - send arms;
  - timeout arms;
  - default arms;
  - receive bindings scoped only to the selected arm body.
- Closed-channel behavior follows `docs/v0.60/SPEC.md`:
  - receive on a closed drained channel completes with `nil`;
  - send to a closed channel raises.
- Implement a runtime task abstraction whose semantic identity is not
  `pthread_t`.
- Implement a scheduler with:
  - runnable queue;
  - task wait/wake operations;
  - await waiters;
  - channel send/receive/select waiters;
  - timer waiters;
  - cancellation propagation;
  - GC-safe task roots.
- Integrate `libuv` as the private event-loop backend for timers and future I/O
  readiness.
- Use a C coroutine/fiber path for direct-style suspension before considering a
  compiler-generated state-machine emitter.
- Make blocked channel receivers, blocked senders, timeout waiters, and await
  waiters scale without one OS thread per task.
- Include CPU-bound fairness and preemption in this feature. CPU-bound Tya code
  must not permanently monopolize all scheduler progress once the C10K runtime
  work is complete.

## Scope

- Parser and AST support for `select`, if any gaps remain.
- Checker diagnostics and migration guidance for removed helper APIs.
- C codegen for `spawn`, `await`, `scope`, and `select`.
- Runtime task, scheduler, channel, sync, timer, and libuv integration.
- Stdlib wrappers under:
  - `lib/channel/`
  - `lib/task/`
  - `lib/sync/`
- Go interpreter parity where the existing tests require it.
- `docs/STDLIB.md` and release documentation.
- Tests for language syntax, stdlib API, runtime behavior, diagnostics, and
  scalability.
- `ROADMAP.md`.

## Out of Scope

- HTTP server C10K integration.
- Public event-loop APIs.
- `async fn` or function coloring.
- Promise/future APIs separate from `spawn` / `await`.
- Unbounded channel APIs.
- Native package manager changes for C dependencies.
- Replacing `select` with `Channel.select`.

## Acceptance Criteria

- Existing v0.42, v0.43, and v0.60 concurrency tests pass.
- New tests cover class-style channel, task, mutex, atomic integer, and wait
  group methods.
- Old helper-style APIs fail with `[TYA-E0820]` and actionable replacement
  messages.
- `select` supports receive, ignored receive, send, timeout, and default arms.
- `select` receive bindings are scoped to the selected arm body.
- `select` waits through runtime waiter records instead of tight polling.
- Closing a channel wakes waiting senders, receivers, and select waiters.
- `receive_timeout(0)` and `timeout 0` behave as single best-effort polls.
- `Mutex.with_lock(fn)` unlocks when `fn` raises and re-raises the same value.
- `scope` cancellation and cleanup behavior remains correct for task raises and
  synchronous body raises.
- `task.Task.current()` returns the running task value inside spawned tasks and
  `nil` at top level.
- 10,000 channel-blocked Tya tasks can exist without creating 10,000 OS
  threads.
- 10,000 timer-blocked Tya tasks can exist without creating 10,000 OS threads.
- 10,000 await-blocked Tya tasks can exist without creating 10,000 OS threads.
- CPU-bound Tya tasks do not permanently starve ready tasks, timers, or channel
  wakeups.
- `libuv` is linked or vendored in a way that keeps normal `tya run` and
  `go test` workflows reproducible on supported platforms.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run 'TestV(42|43|60).*Script' -count=1
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```

Add focused stress verification for the C10K behavior, either as Go tests,
testscript tests, or scripts under `scripts/`, covering:

```text
10,000 channel-blocked tasks
10,000 timer-blocked tasks
10,000 await-blocked tasks
CPU-bound fairness against ready task progress
```

## Dependencies

- Use `docs/v0.60/SPEC.md` as the implementation spec.
- Preserve compatibility with the existing self-host fixed-point gate.
- Coordinate any libuv build changes with existing C runtime and package
  linking behavior.

## Open Questions

None.
