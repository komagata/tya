---
layout: doc
title: Release Notes
permalink: /v0.60/release-notes/
---

# Tya v0.60 Release Notes

v0.60 makes Tya concurrency match the class-style direction of the standard
library while starting the runtime path toward C10K workloads.

## Highlights

- `channel.Channel(capacity)` replaces `channel.Channel.new(capacity)`.
- Channel operations are instance methods: `send`, `receive`,
  `receive_timeout`, `close`, and `closed?`.
- Task cancellation is instance-based: `t.cancel()` and `t.cancelled?()`.
- Synchronization primitives are classes: `sync.Mutex`,
  `sync.AtomicInteger`, and `sync.WaitGroup`.
- `select` is now a language statement for channel receive, send, timeout, and
  default arms.
- `spawn` tasks are scheduled as cooperative lightweight runtime tasks rather
  than one OS thread per task.

## Breaking Changes

The old helper APIs are removed. Use the instance-method forms instead.

```tya
channel.Channel.new(10)        # removed
channel.Channel.send(c, value) # removed
channel.Channel.select([...])  # removed
sync.Sync.mutex()              # removed
task.Task.cancel(t)            # removed
```

Removed helper calls report `TYA-E0820`.

## Verification

The release includes script coverage for:

- class-style channel, task, mutex, atomic integer, and wait group APIs;
- `select` receive, send, and default arms;
- removed helper diagnostics;
- 10,000 blocked channel receiver tasks without creating 10,000 OS threads.
