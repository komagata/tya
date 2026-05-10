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

### STEP 4 — `scope` block (planned)

Codegen for the structured-concurrency `scope` block. Track tasks
created inside the block and await all on exit. On `raise`, signal
cancel to outstanding tasks and re-raise after they settle. Add
`task.is_cancelled()` poll API for cooperative cancel.

### STEP 5 — `channel` stdlib module (planned)

`channel.new`, `channel.send`, `channel.receive`, `channel.close`,
`channel.closed?`. Buffered and unbuffered channels.

### STEP 6 — `channel.receive_timeout` and `channel.select` (planned)

`channel.receive_timeout(c, seconds)` and `channel.select([...])`.

### STEP 7 — `sync` stdlib module (planned)

`sync.mutex`, `sync.atomic_integer`, `sync.wait_group`.

### STEP 8 — Documentation, examples, v0.42 SPEC finalization (planned)

`docs/CONCURRENCY.md` companion document, `examples/concurrent/`,
release prep.

## Observable language behavior

After STEP 1, three keywords are reserved and three syntactic forms are
recognized. Programs that try to evaluate any of the forms get a
structured "not yet implemented" error. STEP 2 onward will progressively
make them functional.
