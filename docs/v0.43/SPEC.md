# Tya v0.43 Specification

## Theme

v0.43 closes three known gaps documented for v0.42:

1. Cooperative cancellation through `task.cancel`,
   `task.cancelled?`, and `task.current()`.
2. `scope` body raises now run `tya_scope_exit` before the raise
   propagates, so siblings are joined and (when applicable)
   cancelled before control unwinds.
3. `channel.select` for multi-channel waits.

The v0.42 surface (`spawn`, `await`, `scope`, `channel`, `sync`)
is unchanged. No new keywords. Two new stdlib modules (`task`),
two new entry points in existing modules (`channel.select`,
`channel.try_send`).

## Implementation status

### STEP 1 — Cooperative cancellation (landed)

`task.cancel(t)` sets the atomic cancel flag on a task.
`task.cancelled?(t)` returns the current value of the flag.
`task.current()` returns the currently-running task value
(`nil` on the main thread). Worker bodies use the pattern:

```tya
import task

worker = () ->
  me = task.current()
  while not task.cancelled?(me)
    do_one_step()
```

Cancellation is cooperative: setting the flag does not stop a
worker, only signals it. Long-running workers must poll
`task.cancelled?(me)` at safe points and return early.

The new builtins are `task_cancel`, `task_is_cancelled_p`,
`task_current`. They go through stdlib `task` module wrappers
named `cancel`, `cancelled?`, `current`.

### STEP 2 — `scope` body raise propagation (landed)

Codegen now wraps the `scope` body in `setjmp` + a raise frame.

- Normal exit:   `tya_pop_raise_frame` and `tya_scope_exit`
                 (joins siblings, may re-raise the first task raise).
- Body raise:    `tya_scope_raise(scope, value)` cancels every
                 sibling, joins them, and re-raises the body's
                 value (taking precedence over any task raise).
- Task raise during normal exit: `tya_scope_exit` already cancels
  remaining tasks once it observes the first task raise, so a
  cooperatively-cancellable worker can return early instead of
  running to completion.

The body's raise is preserved across the join, so
`try { scope { ...; raise "body" } } catch e -> e == "body"`
holds even when sibling tasks are still running.

### STEP 4 — Documentation, examples, release (landed)

`docs/STDLIB.md` gains `channel.select`, `task.cancel` /
`task.cancelled?` / `task.current`. `examples/concurrent/` gains
a cancellable worker pool. `docs/v0.42/SPEC.md` "known gaps" is
linked to this v0.43 SPEC.

### STEP 3 — `channel.select` multiplex (landed)

`channel.select(ops)` waits for the first ready operation in `ops`.
`ops` is an array of arrays:

```tya
[c, "receive"]            receive from channel c
[c, "send", value]        send value to channel c
```

The result is a dict with three keys:

- `index` — the position in `ops` that completed.
- `kind`  — `"receive"` or `"send"`.
- `value` — for `"receive"`, the dequeued element (`nil` when the
            channel is closed and drained); for `"send"`, `nil`.

v0.43 polls every operation in a tight loop and sleeps briefly when
nothing is ready. This is functional but not the most efficient
implementation; a future minor will add a proper waiter-list
mechanism inside `TyaChannel`. The current behavior preserves
fairness only in the limit (any operation that becomes ready will
eventually be picked up); ties between two simultaneously-ready ops
are resolved by index order.

A send to a closed channel from inside `select` raises (matches
`channel.send` semantics). A receive from a closed channel returns
the operation result with `value = nil`.
