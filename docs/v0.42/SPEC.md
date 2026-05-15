---
layout: doc
title: Spec
permalink: /v0.42/spec/
---

# Tya v0.42 Specification

This document is the in-progress specification for Tya v0.42. It is updated as
each STEP of the v0.42 Tya Concurrency Epic lands.

## Theme

v0.42 ships **Tya Concurrency**: lightweight tasks, structured concurrency,
and inter-task communication. The language adds three keywords (`spawn`,
`await`, `scope`); the standard library gains two modules (`channel` and
`sync`).

The runtime extends the v0.41 mark-and-sweep collector for multiple worker
threads and adds a `task` value type. v0.42 ships a 1:1 OS-thread
implementation backed by `pthread`; an M:N scheduler is deferred.

## Goals

- Express concurrent computation through `spawn`, `await`, and `scope`.
- Communicate between tasks through the `channel` stdlib module.
- Synchronize when shared state is unavoidable through the `sync` stdlib
  module.
- Bound task lifetimes through structured concurrency (`scope` block).
- Keep the language surface small: no `async` coloring, no `select`
  statement, no `<-` operator.

## Non-goals (deferred)

- M:N scheduler / virtual-thread mapping. v0.42 uses 1:1 OS threads.
- Distributed actor model.
- `link` / `monitor` style supervision.
- Selective receive on channels.

## Implementation status

v0.42 is implemented in eight STEPs. Each STEP keeps every existing test
green and preserves the self-host fixed point.

### STEP 1 — Lexer / parser / AST (in progress; surface only)

- Reserve the keywords `spawn`, `await`, `scope`. They cannot be used as
  variable, function, parameter, class, module, or class member names.
- Parse the new forms:
  - `spawn callable` is a unary expression that produces a task value.
  - `await target` is a unary expression that blocks the current task
    until the operand task completes and yields its return value (or
    re-raises a propagated raise).
  - `scope` introduces a structured-concurrency block (indented body,
    same shape as `while`).
- Add AST nodes: `SpawnExpr`, `AwaitExpr`, `ScopeBlock`.
- Checker walks the new nodes (no static rejection).
- Codegen and the eval interpreter return a structured "not yet
  implemented" error if the program tries to evaluate any of the new
  forms.
- The canonical formatter (`tya format`) emits all three forms.
- Self-host fixed point preserved.

### STEP 2 — Multi-thread GC extension and `task` value (landed)

The v0.41 allocator is now mutex-protected. `tya_gc_alloc`,
`tya_gc_register_root`, `tya_gc_collect`, `tya_gc_maybe_collect`, and
`tya_gc_stats` all serialize on a single `pthread_mutex_t`. A future
minor will move to a finer-grained design when an M:N scheduler lands.

A new `task` value kind (`TYA_TASK` / `TyaTask`) is reserved with the
following layout: `pthread_t`, `pthread_mutex_t`, `pthread_cond_t`,
`done`, `joined`, `raised`, `cancelled` (atomic), `callee` (the
callable), `result`, and `raise_value`. The collector marks `callee`,
`result`, and `raise_value`; the sweeper destroys the mutex and
condvar before freeing the task. STEP 3 will populate the struct from
`spawn` and read it from `await`.

`kind(t)` for a task value returns `"task"`. `tya_to_string` and
`tya_print` render a task as `"[task]"`. Equality (`tya_equal`) is
identity equality.

Generated programs link with `pthread` on Linux (`-lpthread`); on
macOS / BSD pthread is in libc.

### STEP 3 — `spawn` / `await` codegen and runtime (landed)

`spawn callable` is a unary expression that produces a task value.
The spawning thread evaluates the callable expression and the
arguments first, then `tya_task_new` allocates a `TyaTask`,
initializes its mutex / condvar, and starts a pthread that calls
`callable(args...)` once. The arguments are copied into the task's
`argv` array before the pthread runs. Up to four positional
arguments are supported; passing more is a structured error at
codegen time.

`await task` blocks the current thread on `pthread_join`, then
returns either the task's `result` or re-raises its
`raise_value`. The raise frame is thread-local
(`_Thread_local`), so a raise inside the spawned body never
corrupts the awaiter's raise stack.

Special forms recognized by codegen:

- `spawn fn(args)` — call form. `fn` and each argument are evaluated
  in the spawning thread; the new pthread then calls
  `fn(arg0, arg1, ...)`.
- `spawn callable_value` — bare callable. The new pthread calls
  `callable_value()` with no arguments.

Both forms return a `TyaValue` of kind `TYA_TASK`. Awaiting a task
twice is allowed; the first `await` joins the pthread, subsequent
`await`s return the cached `result` (or re-raise the cached
`raise_value`).

### STEP 4 — `scope` block (landed)

`scope` opens a structured-concurrency block. Codegen wraps the body
in a fresh C scope and brackets it with `tya_scope_enter` /
`tya_scope_exit`. The runtime maintains a thread-local stack of
`TyaScope` records, and `tya_task_new` registers each new task in
the innermost open scope.

When control leaves a `scope` block normally, `tya_scope_exit` joins
every task spawned inside the block (in spawn order). If any of
those tasks raised, the first such raise is re-raised after every
sibling has joined.

Open question for a later STEP: a synchronous raise from within the
`scope` body itself bypasses the cleanup (the raise frame walks
back without running `tya_scope_exit`). Cooperative cancel
(`task.is_cancelled()`) is not yet wired. Both will land in a
follow-up.

### STEP 5 — `channel` stdlib module (landed)

The new stdlib `channel` module exposes:

```tya
channel.new(capacity)
channel.send(c, value)
channel.receive(c)
channel.close(c)
channel.closed?(c)
```

Channels are FIFO bounded queues backed by a ring buffer guarded by a
`pthread_mutex_t` plus two condition variables (`not_full`,
`not_empty`). `channel.send` blocks while the buffer is full and
raises if the channel has been closed. `channel.receive` blocks
while the buffer is empty; once the channel is closed, it drains the
remaining elements and then returns `nil` for every later call.
`channel.close` marks the channel closed and broadcasts both
condvars so every waiter wakes up.

`capacity = 0` is treated as `1` in v0.42; true rendezvous
(synchronous) channels arrive in a later minor.

Two operational fixes ride this STEP:

- Tasks that have not yet been joined are kept in a global
  doubly-linked list so the collector treats them as roots; without
  this, a top-level `spawn` whose handle is dropped before the
  worker finishes would be reclaimed mid-flight, freeing its mutex
  and pthread state.
- `tya format` already kept side-effecting expression statements;
  codegen now also emits the `(void)expr;` form for `spawn` and
  `await` when they appear at statement position, so a fire-and-
  forget `spawn fn(args)` actually runs.

A new `TYA_CHANNEL` value kind is reserved with the matching
`TyaChannel` struct. `kind(c)` returns `"channel"`,
`tya_to_string` and `tya_print` render a channel as `[channel]`,
and equality is identity equality.

### STEP 6 — `channel.receive_timeout` (landed)

`channel.receive_timeout(c, seconds)` blocks until either a value is
available or the wall-clock deadline elapses, then returns the
dequeued value (on success) or `nil` (on timeout). `seconds` is a
non-negative number; `0` means "do not wait" (a single best-effort
poll). The implementation uses `pthread_cond_timedwait` against
`CLOCK_REALTIME`. On macOS / BSD the deadline is computed from
`gettimeofday`; elsewhere it is read from `clock_gettime`.

`channel.select` (a multiplexed select-like primitive) is deferred
to a follow-up STEP. The minimum-effort substitute today is
`receive_timeout` with a small budget plus polling, or one
forwarding task per source channel that funnels into a shared
inbox.

### STEP 7 — `sync` stdlib module (landed)

The new stdlib `sync` module exposes three families:

```tya
sync.mutex()
sync.lock(m); sync.unlock(m); sync.with_lock(m, fn)
sync.atomic_integer(initial)
sync.atomic_add(a, n); sync.atomic_load(a); sync.atomic_store(a, n)
sync.atomic_cas(a, expected, new)
sync.wait_group()
sync.wait_group_add(wg, n); sync.wait_group_done(wg); sync.wait_group_wait(wg)
```

The runtime uses one `TyaResource` value kind for all three
primitives (sub-tagged `MUTEX`, `ATOMIC_INTEGER`, `WAIT_GROUP`),
backed by `pthread_mutex_t`, `stdatomic.h` `atomic_long`, and a
counter + condvar respectively. `kind(r)` returns `"mutex"`,
`"atomic_integer"`, or `"wait_group"`. Equality is identity
equality.

Note on Tya closure semantics. Tya closures cannot write back to
outer variables. To share mutable state across `spawn`ed tasks,
pass a dict / array as an argument and mutate it through indexed
assignment:

```tya
state = {}
state["count"] = 0
m = sync.mutex()

inc = mref, sref ->
  sync.lock(mref)
  sref["count"] = sref["count"] + 1
  sync.unlock(mref)

t = spawn inc(m, state)
await t
```

The `tests/testdata/v42/sync.txtar` testscript covers a
mutex-protected dict counter, atomic add / load / cas, and
`wait_group_wait` blocking until every spawned worker has
called `done`.

### STEP 8 — Documentation, examples, release (landed)

`examples/concurrent/` ships representative end-to-end programs:
parallel fetch via `scope`, a long-lived `Counter` actor that owns
its state and answers requests over a `channel`, a worker pool
co-ordinated through `sync.wait_group`, and a producer / consumer
streaming through a buffered channel. `docs/STDLIB.md` gains
sections for the new `channel` and `sync` modules; the v0.42 SPEC
above is the canonical surface description.

## Observable language behavior summary

- Three new keywords: `spawn`, `await`, `scope`.
- Two new value kinds: `task`, `channel`. A third (`resource`)
  hosts the sync primitives and surfaces as `kind(r) ==
  "mutex" | "atomic_integer" | "wait_group"`.
- Two new stdlib modules: `channel`, `sync`.
- Generated programs link with `pthread` on Linux.
- `runtime.gc()` is now thread-safe.

The full safety contract (locals are still not roots; collections
inside function bodies are not safe in v0.42 either; the existing
v0.41 limitations still apply) is documented in
`docs/v0.41/SPEC.md` and inherited by v0.42 unchanged.
