---
status: completed
goal_ready: false
---

# Feature: Log Stdlib Library

## Goal

Add a `log` standard library package for structured application logging with
levels, deterministic formatting, and stderr output by default.

## Context

Tya currently has `print`, process execution, and ad hoc stderr output inside
the toolchain. User programs lack a small standard way to write operational
logs. CLI tools, HTTP servers, package tools, and long-running tasks all need
level-filtered logs without inventing formatting conventions.

## Behavior

- Add `lib/log/Logger.tya`.
- `log.Logger.default()` returns a logger writing text logs to stderr.
- `log.Logger.new(options)` creates a logger.
- Supported levels: `debug`, `info`, `warn`, `error`.
- Default level is `info`.
- Methods:
  - `debug(message, fields)`
  - `info(message, fields)`
  - `warn(message, fields)`
  - `error(message, fields)`
  - `with(fields)`
  - `level(value)`
- `fields` is optional and defaults to `{}`.
- Text format is deterministic: timestamp, level, message, sorted fields.
- JSON format is available with `{ format: "json" }`.
- Logger writes to stderr by default.
- Logger can write to a file path through `{ file: "path" }`.
- `with(fields)` returns a child logger that includes base fields on every
  record.

## Scope

- `lib/log/Logger.tya`
- host/runtime support only if stderr/file append primitives are missing
- `docs/STDLIB.md`
- next release docs
- stdlib tests and/or script tests
- `ROADMAP.md`

## Out of Scope

- Log rotation.
- Async/buffered logging.
- Syslog/journald integrations.
- Distributed tracing.
- Terminal colors.

## Acceptance Criteria

- `import log` exposes `log.Logger`.
- Default logger writes `info`, `warn`, and `error`, but suppresses `debug`.
- `level("debug")` enables debug logs.
- JSON format emits valid JSON with stable keys.
- Fields are sorted deterministically in text output.
- `with` preserves base fields and allows per-message fields.
- File output appends records.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```

## Open Questions

None.
