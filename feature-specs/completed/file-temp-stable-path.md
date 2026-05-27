# Feature: File Temp Stable Path

## Goal

`file.File().temp(prefix, suffix)` should return a stable path string that continues to point at the temporary file created by the runtime.

## Context

Issue #14 reports that the C runtime implementation for `file.File().temp(prefix, suffix)` can return a string backed by unstable local buffer memory:

```tya
import file

tmp = file.File().temp("tya-fs", ".tmp")
print(file.File().exists?(tmp))
file.File().remove(tmp)
```

The expected output is:

```text
true
```

The filesystem standard library documents `file/File().temp(prefix = "tya", suffix = "")` as creating an empty temporary file under the operating-system temporary directory and returning its path. After the class-style stdlib migration, filesystem tests exposed cases where the returned path later became corrupted, causing `exists?` or `remove` to inspect the wrong path.

The runtime function `tya_file_temp` must ensure that the returned `TyaValue` owns a stable copy of the path string after `mkstemps` succeeds.

## Behavior

- `file.File().temp(prefix, suffix)` creates an empty temporary file.
- The returned value is a stable Tya string that remains valid after `tya_file_temp` returns.
- `file.File().exists?(tmp)` returns `true` for the returned path immediately after creation.
- `file.File().remove(tmp)` removes the created temporary file using the returned path.
- The suffix argument remains part of the generated path when provided.
- Runtime failure behavior remains unchanged: if temporary file creation fails, the runtime raises the existing filesystem temp error.
- `dir.Dir().temp_dir(prefix)` behavior is not changed, but may be used as a reference because it already owns a copied path buffer before returning.

## Scope

- C runtime implementation of `tya_file_temp` in `runtime/tya_runtime.c`.
- Any helper needed to duplicate path strings safely before wrapping them as Tya strings.
- Generated-C/runtime filesystem tests for `file.File().temp`.
- Existing stdlib filesystem tests that exercise temp file creation, existence checks, and removal.

## Out of Scope

- Changing the public `file.File().temp` API shape.
- Changing the operating-system temporary directory selection policy.
- Changing random filename generation or collision handling beyond the existing `mkstemps` behavior.
- Adding automatic cleanup for temporary files.
- Changing `dir.Dir().temp_dir` unless verification exposes the same ownership bug there.
- Reworking filesystem error types or messages except where needed to preserve the current temp creation error.

## Acceptance Criteria

- The issue #14 reproduction program prints `true` and removes the created file.
- A generated-C test covers `file.File().temp("tya-fs", ".tmp")`, then `exists?`, then `remove`.
- The returned path keeps the requested suffix.
- The temp path remains usable across at least one subsequent runtime call before `exists?` or `remove`.
- No address, stack buffer, or freed local storage is used as the returned string backing memory.
- Existing filesystem utility tests continue to pass.

## Verification

```sh
go test ./internal/codegen -run File -count=1
go test ./tests -run 'TestV65Scripts/v1_stdlib_filesystem_utilities|TestSelfhostV01Scripts' -count=1
```
