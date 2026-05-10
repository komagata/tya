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

### STEP 2 — `scope` body raise propagation (planned)

Codegen wraps the `scope` body in a setjmp/longjmp pair so a
synchronous raise from the body still calls `tya_scope_exit`
before unwinding.

### STEP 3 — `channel.select` multiplex (planned)

Function-form select that waits on a list of channel ops and
returns the index of the operation that completed.
