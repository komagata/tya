# Tya v0.60 Specification

> **Status:** released. v0.60 turns the existing concurrency surface into a
> class-style API and starts the runtime path toward C10K-capable lightweight
> tasks. `spawn`, `await`, and `scope` remain language constructs. Channel,
> task, and synchronization operations move from class-static helper style to
> value method style.

## Theme

Tya already has the right concurrency vocabulary:

```tya
scope
  t = spawn work()
  result = await t
```

The weakness is the implementation and the surrounding API shape. v0.42 shipped
`spawn` as one `pthread` per task, and v0.45 migrated the concurrency modules to
class namespaces such as `channel.Channel.send(c, value)`. That is usable, but
it does not match the class-style direction of the rest of the language and it
cannot scale to C10K-style workloads.

v0.60 keeps the language surface small:

- `spawn`, `await`, and `scope` stay as keywords.
- `Channel`, `Task`, `Mutex`, `AtomicInteger`, and `WaitGroup` become the
  public class-style concurrency API.
- Operations on concurrency values become instance methods.
- Runtime work begins toward lightweight tasks scheduled by Tya rather than
  one OS thread per task.

The HTTP server is deliberately out of scope for this specification. The runtime
work here is the foundation that a later `net/http` C10K implementation will
use.

## Goals

- Keep structured concurrency as the default model.
- Preserve the existing `spawn` / `await` / `scope` syntax.
- Make concurrency stdlib APIs consistent with class-style Tya.
- Prefer bounded queues and explicit backpressure.
- Define the runtime direction: Tya tasks are lightweight runtime tasks, not OS
  threads.
- Keep implementation incremental so existing tests and the self-host fixed
  point can stay green between steps.

## Non-goals

- No HTTP server changes in v0.60.
- No `async fn` or function-coloring model.
- No preemptive user-code scheduling guarantee.
- No promise that CPU-bound Tya code is fairly preempted.
- No native dependency manager for C libraries.
- No public event-loop API.

## Language Surface

These remain language constructs, not stdlib class methods:

```tya
spawn expr
await task
scope
  ...
select
  ...
```

They should not be replaced by `Task.spawn(...)`, `Task.await(...)`, or
`Scope.run(...)`. `spawn`, `await`, and `scope` control evaluation and task
lifetime, so they belong with `if`, `try`, and `return`, not with ordinary
library calls.

## Class-Style Concurrency API

### Channel

The new preferred API:

```tya
import channel

c = channel.Channel(100)

c.send("hello")
msg = c.receive()
msg = c.receive_timeout(1.0)
c.close()
print(c.closed?())
```

`channel.Channel(capacity)` constructs a bounded FIFO channel. `capacity` must
be a non-negative integer. A capacity of `0` continues to behave as capacity
`1` until true rendezvous channels are specified.

Methods:

```text
Channel(capacity)
c.send(value)
c.receive()
c.receive_timeout(seconds)
c.close()
c.closed?()
```

`send` blocks the current Tya task while the channel is full. `receive` blocks
the current Tya task while the channel is empty. Blocking a channel operation
must not require one OS thread per blocked task once the v0.60 runtime work is
complete.

`receive_timeout(seconds)` returns `nil` on timeout. `seconds = 0` means a
single best-effort poll.

`close` marks the channel closed and wakes all waiting senders and receivers.
After close, receivers drain buffered values and then receive `nil`.

## `select` Statement

v0.60 adds `select` as a language statement for waiting on multiple channel
operations and timeouts without polling.

```tya
select
  value = receive inbox
    handle(value)
  send outbox, "ready"
    sent = true
  timeout 1.0
    raise "timeout"
  default
    idle()
```

Each arm has an operation line and an indented body. The first operation that
can complete runs its body. If multiple operations are ready at the same time,
the runtime may choose any ready arm; programs must not depend on tie order.

Supported operation forms:

```tya
value = receive channel_expr
receive channel_expr
send channel_expr, value_expr
timeout seconds_expr
default
```

`value = receive ch` receives from `ch` and binds the received value only inside
that arm body. `receive ch` is valid when the received value is intentionally
ignored.

`send ch, value` sends `value` to `ch` and runs the arm body after the send has
completed.

`timeout seconds` becomes ready after the timeout elapses. `seconds` must be a
non-negative number. `timeout 0` is a single best-effort poll.

`default` is ready immediately when no receive or send arm can complete at the
moment the `select` starts. It must not run if another non-timeout operation is
already ready. A `select` may have at most one `default` arm.

At least one arm is required. A `select` with only timeout arms is valid but is
equivalent to a sleep plus body dispatch.

Closed-channel behavior matches `Channel.receive()` and `Channel.send()`:

- receive on a closed drained channel completes with `nil`;
- send to a closed channel raises.

The implementation must not poll in a tight loop. Channels must keep waiter
records for send, receive, and select operations. Timers must be registered
with the scheduler / event-loop backend. When an operation becomes ready, the
runtime wakes the waiting Tya task directly and unregisters the losing arms.

### `select` Scope

A receive binding is scoped to that arm body:

```tya
select
  value = receive a
    print(value)

print(value) # invalid unless `value` existed before the select
```

If the receive target name already exists in an outer scope, the arm creates a
new nested binding and follows the existing shadowing rules.

### `Channel.select`

The old `channel.Channel.select([...])` helper is removed with the other
helper-style APIs. Use the `select` statement instead.

### Task

`spawn` returns a task value. Task operations become methods:

```tya
import task

worker = () ->
  me = task.Task.current()
  while not me.cancelled?()
    do_work()

t = spawn worker()
t.cancel()
print(t.cancelled?())
result = await t
```

Methods:

```text
Task.current()
t.cancel()
t.cancelled?()
```

`Task.current()` remains a static class method because it does not operate on an
existing task value.

Cancellation remains cooperative. `cancel` sets a flag. It does not forcibly
interrupt arbitrary Tya code. Long-running task bodies must poll
`cancelled?()` at safe points.

### Mutex

The preferred mutex API:

```tya
import sync

m = sync.Mutex()

m.lock()
try
  update_state()
catch e
  m.unlock()
  raise e
m.unlock()
```

Preferred helper:

```tya
m.with_lock(() ->
  update_state()
)
```

Methods:

```text
Mutex()
m.lock()
m.unlock()
m.with_lock(fn)
```

`with_lock` must release the mutex when `fn` raises, then re-raise the same
value.

### AtomicInteger

The preferred atomic integer API:

```tya
import sync

a = sync.AtomicInteger(0)
a.add(1)
print(a.load())
a.store(10)
ok = a.cas(10, 11)
```

Methods:

```text
AtomicInteger(initial)
a.add(n)
a.load()
a.store(n)
a.cas(expected, new_value)
```

### WaitGroup

`WaitGroup` remains available for interop with existing worker-pool patterns,
but docs should prefer `scope` for structured task lifetime.

```tya
import sync

wg = sync.WaitGroup()
wg.add(1)
spawn (() ->
  do_work()
  wg.done()
)
wg.wait()
```

Methods:

```text
WaitGroup()
wg.add(n)
wg.done()
wg.wait()
```

Without `defer` / `finally`, `WaitGroup` is easy to misuse when worker bodies
raise. New examples should prefer `scope`, channels, and task cancellation
unless a wait group is specifically needed.

## Backward Compatibility

The old class-static API is removed in v0.60:

```tya
channel.Channel.new(10)
channel.Channel.send(c, value)
channel.Channel.receive(c)
channel.Channel.close(c)

task.Task.cancel(t)
task.Task.cancelled?(t)

sync.Sync.mutex()
sync.Sync.lock(m)
sync.Sync.unlock(m)
sync.Sync.atomic_integer(0)
sync.Sync.wait_group()
```

Programs must migrate to the instance method API:

```tya
c = channel.Channel(10)
c.send(value)
value = c.receive()
c.close()

t.cancel()
t.cancelled?()

m = sync.Mutex()
m.lock()
m.unlock()
```

`Channel.select` is removed too. Use the `select` statement.

## Runtime Model

The long-term runtime model is:

```text
Tya code
  spawn / await / scope / channel
    ↓
Tya lightweight task
    ↓
Tya scheduler
    ↓
small OS-thread pool
    ↓
event loop / async I/O backend
```

`spawn` must no longer mean "create one pthread". It means "create one Tya
task". A Tya task is a lightweight runtime-managed execution unit with:

- task state: ready, running, waiting, done, cancelled
- result or raised value
- cancellation flag
- parent scope
- waiter list for `await`
- links needed by the scheduler and GC

The scheduler owns a runnable queue. `await`, channel operations, timers, and
future non-blocking I/O suspend only the current Tya task. They must not require
one blocked OS thread per blocked Tya task.

The first implementation may still use cooperative scheduling. CPU-bound Tya
code that never reaches a scheduling point may monopolize an OS worker. That is
acceptable for v0.60 and should be documented. Preemption can be a later runtime
epic.

## External Runtime Libraries

Tya may use C libraries to reduce implementation cost.

Recommended split:

```text
Tya runtime:
  task identity
  spawn / await / scope semantics
  channel semantics
  cancellation
  GC integration

External library:
  event loop
  timers
  async I/O readiness
  blocking / CPU worker pool
```

`libuv` is the preferred event-loop candidate because it is a C library, is
cross-platform, and hides Linux `epoll`, macOS/BSD `kqueue`, and Windows IOCP.
It also provides timers and a worker pool.

`libuv` is not itself the Tya scheduler. Tya still needs its own task model to
preserve `spawn`, `await`, `scope`, and channel semantics.

For lightweight task suspension, Tya can choose either:

1. a small stackful coroutine / fiber library in C; or
2. a compiler-generated state-machine transformation.

The v0.60 preferred implementation path is stackful coroutine first, because it
keeps the C emitter closer to the current direct-style code. A state-machine
emitter remains a future option if the runtime wants to avoid separate stacks.

C++ dependencies such as Boost.Fiber are not preferred for the default runtime
because they require a C++ toolchain and complicate distribution. A C dependency
is acceptable if it materially reduces the runtime scheduler and event-loop
burden.

## Backpressure

Unbounded work queues are not the default.

Rules:

- `Channel(capacity)` creates a bounded channel.
- `capacity = 0` is not an unbounded channel.
- Any future unbounded channel API must be explicit, e.g.
  `Channel.unbounded()`, and documented as dangerous for servers.
- Task spawning inside servers and worker pools should be paired with a scope,
  bounded channel, semaphore, or connection limit.

The language must make the safe pattern the obvious pattern:

```tya
jobs = channel.Channel(1000)

scope
  worker1 = spawn worker(jobs)
  worker2 = spawn worker(jobs)
  ...
```

## Structured Concurrency

`scope` remains the owner of task lifetime.

Rules:

- A task spawned inside a scope is registered with that scope.
- Leaving the scope waits for every spawned task.
- If a task raises, the scope cancels siblings and then re-raises the first
  task raise after cleanup.
- If the scope body raises, the scope cancels spawned children, waits for them,
  and then re-raises the body raise.
- Cancellation is cooperative but must be propagated by the runtime.

Unscoped top-level `spawn` is still allowed for compatibility, but new examples
should prefer scoped tasks unless deliberately starting a process-lifetime
actor.

## Diagnostics

New diagnostics should be stable and actionable:

| Code | Meaning |
|---|---|
| `TYA-E0820` | removed concurrency helper API; use instance method style |
| `TYA-E0821` | invalid channel capacity |
| `TYA-E0822` | invalid `select` arm |
| `TYA-E0823` | await expects a task |
| `TYA-E0824` | task operation expects a task |
| `TYA-E0825` | mutex operation expects a mutex |
| `TYA-E0826` | atomic operation expects an atomic integer |
| `TYA-E0827` | wait group operation expects a wait group |

Removed helper-style APIs are compile-time or load-time errors in v0.60. The
diagnostic must point to the instance method replacement.

## Implementation Plan

The work should be split for long-running `/goal` execution.

1. **Spec and docs**
   - Add this spec.
   - Update `docs/STDLIB.md` examples to the preferred class-style API.
   - Add migration notes from static helper style to instance method style.

2. **Parser / AST**
   - Add `select` statement parsing.
   - Add AST nodes for select statements and receive / send / timeout arms.
   - Keep `spawn`, `await`, and `scope` nodes unchanged.

3. **Stdlib wrappers**
   - Add `Channel(capacity)` constructor behavior.
   - Add instance methods for channel values.
   - Add `Task` instance methods on task values.
   - Replace `sync.Sync` with `sync.Mutex`, `sync.AtomicInteger`, and
     `sync.WaitGroup`.

4. **Runtime method dispatch**
   - Extend member dispatch for `TYA_CHANNEL`, `TYA_TASK`, and sync resource
     values.
   - Preserve existing builtins as implementation targets.

5. **Channel scheduler cleanup**
   - Implement `select` statement waits with waiter-list wakeups.
   - Make send / receive / select waiters explicit runtime records.

6. **Task abstraction split**
   - Introduce a runtime task abstraction that does not expose `pthread_t` as
     the semantic identity.
   - Keep the pthread backend temporarily if needed.
   - Move public task behavior onto the abstraction.

7. **Lightweight scheduler**
   - Add runnable queue.
   - Add task wait / wake operations.
   - Add timer wait as a scheduler operation.
   - Integrate with channel and await waits.

8. **Event-loop backend**
   - Evaluate and integrate `libuv` for timers and future I/O readiness.
   - Keep the backend private to the runtime.

9. **Verification**
   - Existing v0.42 / v0.43 concurrency tests pass.
   - New tests cover instance method API and `select` syntax.
   - `select` does not spin under idle wait.
   - Spawning 10,000 sleeping or channel-blocked tasks does not create 10,000 OS
     threads.
   - Self-host fixed point remains green.

## Migration Examples

Channel:

```tya
# old
c = channel.Channel.new(10)
channel.Channel.send(c, "hello")
msg = channel.Channel.receive(c)

# new
c = channel.Channel(10)
c.send("hello")
msg = c.receive()
```

Task:

```tya
# old
task.Task.cancel(t)
if task.Task.cancelled?(t)
  ...

# new
t.cancel()
if t.cancelled?()
  ...
```

Sync:

```tya
# old
m = sync.Sync.mutex()
sync.Sync.lock(m)
sync.Sync.unlock(m)

# new
m = sync.Mutex()
m.lock()
m.unlock()
```

Atomic integer:

```tya
# old
a = sync.Sync.atomic_integer(0)
sync.Sync.atomic_add(a, 1)
v = sync.Sync.atomic_load(a)

# new
a = sync.AtomicInteger(0)
a.add(1)
v = a.load()
```

Wait group:

```tya
# old
wg = sync.Sync.wait_group()
sync.Sync.wait_group_add(wg, 1)
sync.Sync.wait_group_done(wg)
sync.Sync.wait_group_wait(wg)

# new
wg = sync.WaitGroup()
wg.add(1)
wg.done()
wg.wait()
```

## Success Criteria

v0.60 is complete when:

- class-style concurrency examples pass through `tya run`;
- old helper-style APIs produce the planned migration diagnostic;
- task, channel, and sync values support the documented methods;
- `spawn`, `await`, `scope`, and `select` have the documented user-facing
  behavior;
- `select` no longer uses polling as its normal wait mechanism;
- 10,000 blocked Tya tasks can exist without 10,000 OS threads;
- `go test ./... -count=1` passes, including the self-host fixed point.
