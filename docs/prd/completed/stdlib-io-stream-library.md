---
status: completed
goal_ready: false
---

# Feature: IO Stream Stdlib Library

## Goal

Add an `io` standard library package for line-oriented and chunk-oriented input
and output, filling the gap between whole-file helpers and low-level process
I/O.

## Context

Tya has `read_line`, `file.read`, `file.write`, and `process.run`, but no
general reader/writer abstraction. That makes large files, line processing,
copying, and stdin/stdout/stderr handling awkward or memory-heavy.

## Behavior

- Add `stdlib/io/Reader.tya`, `Writer.tya`, and `Io.tya` if multiple public
  classes are useful.
- `io.Io.stdin()`, `stdout()`, and `stderr()` expose process streams.
- `io.Io.open(path, mode)` opens a file stream.
- Reader methods:
  - `read(size)`
  - `read_line()`
  - `each_line(fn)`
  - `eof?()`
  - `close()`
- Writer methods:
  - `write(value)`
  - `write_line(value)`
  - `flush()`
  - `close()`
- `io.Io.copy(reader, writer)` copies until EOF and returns byte count.
- Text mode is default.
- Binary mode is available through mode `"rb"` / `"wb"` and returns/writes
  `bytes`.
- Closing stdin/stdout/stderr is a no-op or raises a clear error; it must not
  break the host process unexpectedly.

## Scope

- stdlib `io` package
- runtime/builtin support for stream handles if needed
- `docs/STDLIB.md`
- next release docs
- tests for stdin/stdout/file streams and line iteration
- `ROADMAP.md`

## Out of Scope

- Non-blocking/evented I/O.
- Network sockets.
- Compression streams.
- Random-access file seeking in the first version.
- Encoding conversion beyond bytes/text distinction.

## Acceptance Criteria

- `import io` exposes process stream accessors.
- A file can be opened, read line by line, and closed.
- A file can be written incrementally and flushed.
- `Io.copy` copies text and binary files without reading the whole file into a
  Tya string.
- EOF behavior is deterministic.
- Existing whole-file `file` helpers continue to work.
- The self-host fixed point remains green.

## Verification

```sh
go test ./tests -run TestSelfhostV01Scripts -count=1
go test ./... -count=1
```

## Open Questions

None.
