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

### STEP 3 — `channel.select` multiplex (planned)

Function-form select that waits on a list of channel ops and
returns the index of the operation that completed.
