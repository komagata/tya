# Tya v0.22 Specification

This document is the specification for Tya v0.22 after v0.21 native-backed
`file` and `os` standard modules.

## Theme

Tya v0.22 expands native-backed filesystem standard library APIs.

v0.21 adds the minimal native-backed surface. v0.22 makes filesystem scripting
practical by adding directory operations, file metadata, path user expansion,
and current-directory helpers.

## Goals

- Add a `dir` standard module.
- Expand the `file` standard module.
- Expand the `path` standard module.
- Expand the `os` standard module with current-directory helpers.
- Keep permissions API small and metadata-based.
- Keep time/date, streaming IO, binary IO, and async IO out of v0.22.

## Included in v0.22

v0.22 includes all v0.21 behavior and adds:

- `dir.list(path)`
- `dir.mkdir(path)`
- `dir.rmdir(path)`
- `file.remove(path)`
- `file.rename(old_path, new_path)`
- `file.stat(path)`
- `path.expand_user(value)`
- `os.cwd()`
- `os.chdir(path)`
- permission booleans returned by `file.stat(path)`

## Not Included in v0.22

v0.22 does not include:

- time/date APIs
- streaming IO
- binary IO
- async IO
- recursive directory walking
- `mkdir_all`
- `remove_all`
- file copy
- symlink APIs
- chmod/chown
- file handles
- `$VAR` environment expansion in paths
- platform-specific path separators

## `dir`

The `dir` module contains directory operations.

Functions:

- `dir.list(path)`
- `dir.mkdir(path)`
- `dir.rmdir(path)`

Examples:

```tya
import dir

names = dir.list(".")
dir.mkdir("tmp")
dir.rmdir("tmp")
```

`dir.list(path)` returns an array of names directly under `path`.

`dir.list(path)` does not include `.` or `..`. The returned names are sorted in
dictionary order for stable results.

`dir.mkdir(path)` creates one directory level. It raises an error when the
parent directory does not exist or when `path` already exists.

`dir.rmdir(path)` removes an empty directory. It raises an error when the
directory is not empty.

All `dir` functions require string path arguments.

## `file` Expansion

v0.22 expands the `file` module from v0.21.

New functions:

- `file.remove(path)`
- `file.rename(old_path, new_path)`
- `file.stat(path)`

Examples:

```tya
import file

file.rename("memo.txt", "memo.old.txt")
info = file.stat("memo.old.txt")

println info["kind"]
println info["size"]
println info["readable"]

file.remove("memo.old.txt")
```

`file.remove(path)` removes a file. It does not remove directories.

`file.rename(old_path, new_path)` renames a file or directory.

`file.stat(path)` returns a dictionary with metadata:

```tya
{
  kind: "file",
  size: 120,
  readable: true,
  writable: true,
  executable: false,
}
```

`kind` is one of:

- `"file"`
- `"dir"`
- `"other"`

`size` is the file size in bytes. For directories and other filesystem objects,
`size` is implementation-defined.

`readable`, `writable`, and `executable` are booleans. They are the v0.22
permissions API.

`file.stat(path)` does not include modification time, creation time, owner,
group, inode, device, or platform-specific metadata in v0.22.

All new `file` functions require string path arguments.

## `path` Expansion

v0.22 adds user-home expansion to the `path` module.

New function:

- `path.expand_user(value)`

Example:

```tya
import path

memo = path.expand_user("~/memo.txt")
```

`path.expand_user(value)` expands `~` and `~/...` to the current user's home
directory.

Other strings are returned unchanged.

`path.expand_user(value)` is lexical except for reading the current user's home
directory. It does not check whether the resulting path exists.

`$VAR` environment expansion is not part of v0.22.

## `os` Expansion

v0.22 expands the `os` module with current-directory helpers.

New functions:

- `os.cwd()`
- `os.chdir(path)`

Examples:

```tya
import os

original = os.cwd()
os.chdir("/tmp")
println os.cwd()
os.chdir(original)
```

`os.cwd()` returns the current working directory as a string.

`os.chdir(path)` changes the process working directory and returns `nil` on
success.

`os.chdir(path)` affects the whole running process. It raises an error on
failure.

## Errors

Native-backed operation failures raise structured errors. They do not call
`panic`.

This matches v0.21 native-backed stdlib behavior.

```tya
import dir

try
  dir.rmdir("not-empty")
catch err
  println err
```

## Diagnostics

v0.22 implementations should report source-oriented errors for:

- missing imports for `dir`, `file`, `path`, or `os`
- unknown `dir`, `file`, `path`, or `os` module functions
- wrong argument counts
- wrong argument kinds
- failed `dir.list(path)`
- failed `dir.mkdir(path)`
- failed `dir.rmdir(path)`
- failed `file.remove(path)`
- failed `file.rename(old_path, new_path)`
- failed `file.stat(path)`
- failed `path.expand_user(value)`
- failed `os.cwd()`
- failed `os.chdir(path)`

Diagnostics should mention the module name, function name, expected argument
shape, and actual value kind when available.
