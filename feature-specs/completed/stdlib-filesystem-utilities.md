# Feature: stdlib Filesystem Utilities

## Goal

Fill the v1.0.0 standard-library filesystem gap with recursive directory
creation, directory walking, recursive removal, file copying, permissions, and
temporary file/directory helpers.

## Context

Tya already has `file/File`, `dir/Dir`, and `path/Path`. The current surface
supports basic file reads/writes, existence checks, removal, rename, stat,
directory listing, simple directory creation/removal, and path manipulation.

General-purpose CLI tools and build scripts need a slightly broader filesystem
baseline. The accepted direction is to extend the existing class-style packages
rather than adding new syntax or shell-like glob expansion to core CLI target
handling.

## Behavior

- `file.File.copy(src, dst, options = {})` copies file contents from `src` to
  `dst`.
  - `src` and `dst` must be strings.
  - `options` must be a dictionary.
  - Supported options:
    - `overwrite`: bool, default `true`;
    - `preserve_mode`: bool, default `true`.
  - If `overwrite` is false and `dst` exists, the operation raises a structured
    filesystem error.
  - File data is copied as bytes and does not validate UTF-8.
- `file.File.chmod(path, mode)` changes file permissions where the platform
  supports POSIX-like modes.
  - `mode` must be an integer-compatible number.
  - Unsupported platforms raise a structured filesystem error with a stable
    code rather than silently succeeding.
- `dir.Dir.mkdir_all(path)` creates `path` and missing parent directories.
- `dir.Dir.remove_all(path)` removes a file or directory tree recursively.
  - Removing a missing path is a no-op.
  - Removing dangerous paths such as `""`, `"."`, `"/"`, and platform roots is
    invalid.
- `dir.Dir.walk(path, fn, options = {})` walks a directory tree.
  - `fn` is called with an entry dictionary for each visited path.
  - Entry dictionaries include `path`, `name`, `kind`, and `stat`.
  - Walk order is deterministic: entries are visited in ascending path order.
  - Supported options:
    - `follow_symlinks`: bool, default `false`;
    - `include_dirs`: bool, default `true`;
    - `include_files`: bool, default `true`.
  - Symlink loops are detected when symlink following is enabled.
- `dir.Dir.temp_dir(prefix = "tya")` creates a temporary directory and returns
  its path.
- `file.File.temp(prefix = "tya", suffix = "")` creates an empty temporary file
  and returns its path.
- Temporary helpers create paths under the operating system temporary directory,
  use unpredictable names, and never overwrite existing files.
- All filesystem failures raise structured errors with `kind: "filesystem"` and
  stable `code` values.

## Scope

- `lib/file/File.tya`
- `lib/dir/Dir.tya`
- runtime-backed filesystem intrinsics for interpreter and generated C
- `docs/SPEC.md`
- `docs/STRICT_SEMANTICS.md`
- tests under `tests/testdata/` and stdlib unittest files
- release-platform behavior for Linux, macOS, and Windows

## Out of Scope

- Shell-style glob expansion for CLI source targets.
- A full path glob package.
- File watching.
- Extended ACL APIs.
- Cross-platform promise that every platform supports every permission bit.
- Atomic directory tree transactions.

## Acceptance Criteria

- `docs/SPEC.md` documents the extended `file/File` and `dir/Dir` v1 surface.
- Recursive create/remove, copy, chmod, walk, temp file, and temp directory
  helpers work through `tya run` and generated C.
- `Dir.walk` order is deterministic.
- Recursive removal rejects dangerous roots.
- Temporary helpers do not overwrite existing files and return existing paths.
- Filesystem errors are structured and have stable diagnostic codes.
- Windows behavior is documented for permissions and path roots.
- Existing self-host fixed-point gates remain valid.

## Tests To Add

Eval/runtime tests:

- `TestRunFileCopyAndChmod`
  - Copies binary data and changes mode on supported platforms.
  - Expected: bytes are preserved; unsupported chmod raises the documented
    structured error.

- `TestRunDirMkdirAllAndRemoveAll`
  - Creates nested directories and removes a tree.
  - Expected: missing remove target is a no-op; dangerous targets are rejected.

- `TestRunDirWalkDeterministicOrder`
  - Walks a nested tree.
  - Expected: paths are visited in ascending path order with documented entry
    fields.

- `TestRunTempFileAndDir`
  - Creates temporary file and directory.
  - Expected: returned paths exist and are unique across repeated calls.

Codegen tests:

- `TestEmitCFilesystemUtilitiesProgram`
  - Builds and runs copy, mkdir_all, walk, temp, and remove_all usage.

Testscript coverage:

- `v1_stdlib_filesystem_utilities.txtar`
  - Covers CLI-level valid and invalid filesystem operations.

Documentation tests:

- `TestSpecDocumentsFilesystemUtilities`
  - Expected: `docs/SPEC.md` lists all accepted filesystem helpers and their
    platform-specific behavior.

## Verification

```sh
go test ./internal/eval -run 'File|Dir|Filesystem' -count=1
go test ./internal/codegen -run 'File|Dir|Filesystem' -count=1
go test ./tests -run 'TestV.*Scripts|TestSpecDocumentsFilesystemUtilities|TestSelfhostV01Scripts|TestSelfhostV02Scripts' -count=1 -timeout=20m
go test ./... -count=1 -timeout=20m
```
