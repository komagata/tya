# Tya v0.44 Release Notes

> **Status:** shipped. The `tya version` constant is `0.44.0` and
> `ROADMAP.md` carries the matching `Released` entry.

## TL;DR

v0.44 introduces a **class-oriented namespace and entry-file model**.
Libraries become directories of PascalCase class files; entries
become lowercase script files. The `module` keyword is on the way
out (full removal lands at M9, after M8 self-host migration).

```sh
# v0.44 layout
project/
  main.tya                    # entry script (lowercase)
  Greeter.tya                 # sibling class file (PascalCase)
  lib/
    util/
      Logger.tya              # package member
      Cache.tya
```

```tya
# main.tya
import lib/util

logger = util.Logger()       # cross-package: <pkg>.<Class>
cache = util.Cache()
greeter = Greeter()          # same-directory sibling: bare ref
print(greeter.greet())
```

## What's new

### Directory-as-package import

`import path/to/pkg` resolves to a directory containing class files.
The path's terminal segment becomes the in-scope binding. Packages
nest to arbitrary depth via directories.

### Class files

Every PascalCase `.tya` file declares one public class whose name
matches the filename. Additional classes in the same file are
private to the file. Class files may also declare interfaces.

### Script files

Lowercase `.tya` files run via `tya run`. They may declare any
number of private classes used at top level. Imports appear at
the top.

### Within-package bare references

Inside a package, a class file calls / extends / implements its
sibling class files using the bare PascalCase name. The
within-package fallback in the checker and codegen makes
`Request(...)` / `extends Animal` / `implements Encodable` resolve
to the sibling class without a prefix.

### Same-directory siblings for entry scripts

A script entry sees PascalCase class files in its own directory
without an explicit `import`. Useful for tiny multi-file projects
that don't yet need a package directory.

### CLI surface (all six commands work with v0.44)

| Command            | Script | Class | Notes                                            |
| ------------------ | ------ | ----- | ------------------------------------------------ |
| `tya run`          | yes    | no    | entry only                                       |
| `tya build`        | yes    | no    | entry only                                       |
| `tya check`        | yes    | yes   | class file → CheckClassFile                      |
| `tya format`       | yes    | yes   | canonical syntax                                 |
| `tya test`         | yes    | —     | `*_test.tya`                                     |
| `tya --tokens`     | yes    | yes   | lexer dump                                       |
| `tya --emit-c`     | yes    | yes   | class file → standalone-compilable C             |
| `tya --check-unused` | yes  | yes   | strict pass on class files too                   |

### Import path validation

The resolver rejects `..`, absolute paths, leading dotted prefixes,
empty path segments, and PascalCase terminal segments. Cycle
detection catches `a → b → a` through directory packages too.

### Same-segment package collision

Two different directories whose paths share the terminal segment
synthesize the same module name and would clobber each other in
the merged source. The runner catches this at synthesis time with
`[TYA-E0855] package name conflict` for both unaliased and aliased
imports.

### Diagnostic codes

v0.44 wires `[TYA-EXXXX]` inline prefixes on every new runtime
error so users can grep for them in logs and cross-reference the
SPEC table:

- `[TYA-E0400]` — class file must define matching public class
- `[TYA-E0402]` — class file may only contain import / class /
  interface
- `[TYA-E0403]` — imports must precede classes
- `[TYA-E0404]` — class file's filename is not PascalCase
- `[TYA-E0405]` — duplicate public class declaration
- `[TYA-E0850]` — `tya run` invoked on a class file
- `[TYA-E0851]` — invalid module name (path validation)
- `[TYA-E0852]` — package contains a script file
- `[TYA-E0853]` — package contains no class files
- `[TYA-E0854]` — package directory name is not snake_case
- `[TYA-E0855]` — same-segment package conflict

`E0406` (cross-file private class) and `E0200` (`module` keyword
removal) remain reserved pending M5 and M9 respectively.

### Stdlib migration (19 of 27 packages)

- `path`, `random`, `hex`, `base64`, `digest`, `secure_random`
- `process`, `dir`, `file`, `os`, `math`, `csv`, `matrix`, `json`,
  `toml`, `url`
- `unittest`, `value`, `markdown`

Held back for v0.1 self-host compatibility:

- `string`, `array`, `dict` — referenced by the v0.1 self-host
  compiler tests

Held back for working-tree cleanup:

- `runtime`, `time`, `channel`, `sync`, `task`

The held-back packages migrate alongside M8 self-host migration
(string/array/dict) and M7 examples migration (the rest).

## Compatibility notes

- The legacy `module name + functions` shape continues to work
  through v0.44 to keep the v0.1 self-host fixed point green and
  to give users an incremental migration path. M9 removes `module`
  entirely.
- Existing examples, the self-host compiler, and held-back stdlib
  packages still use the legacy form.
- The `tya format` and `tya check` paths now accept class files;
  this is a new surface, not a behavior change.

## Migration guide

A practical companion document lives at
[`docs/v0.44/MIGRATION.md`](MIGRATION.md). It walks the mechanical
recipe: file naming, the `module name` → `class Name` + `@@method`
conversion, internal cross-method calls, sibling auto-visibility,
package directory rules, and the CLI command matrix.

## Known limitations / follow-ups

- **M5 cross-file private enforcement** — the SPEC says private
  classes must not be referenced from other files; the current
  source-concat synthesis pipeline doesn't carry the necessary
  `OriginFile` metadata to enforce this. Fix pairs with M8.
- **String / array / dict** still in legacy `module` form.
- **`tya emit-c` on class file** emits a trivial main; the output
  compiles but does nothing. This is not a bug, just a consequence
  of class files having no entry-point semantics.

## Testdata

22 testscript files under `tests/testdata/v44/` pin the new
behavior end-to-end. See `tests/testdata/v44/README.md` for the
catalogue. Add a new pin by creating a new `*.txtar` (don't
expand existing ones).
