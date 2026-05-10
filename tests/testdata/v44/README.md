# v0.44 testscript coverage

Each `*.txtar` here exercises a piece of the v0.44 class-oriented
namespace and entry-file model. Group the files by milestone:

## M2 — Parser/checker class file shape

- `class_file_rules.txtar` — filename match, structural rules,
  private companion class
- `interface_in_package.txtar` — interface declaration alongside a
  class in the same class file
- `script_file_classes.txtar` — script files declare private
  classes used at top level

## M3 — Resolver, packages, within-package references

- `directory_packages.txtar` — happy path + script-in-pkg + class
  file as entry rejection + sibling auto-visibility
- `nested_packages.txtar` — within-package class-to-class bare
  references under nesting
- `edge_cases.txtar` — three-level nesting, single-class package
  with matching name (`solo.Solo`)
- `single_class_distinct_name.txtar` — single-class package where
  directory and class names differ
- `within_package_extends.txtar` — class inheritance via bare
  sibling reference
- `within_package_class_method_call.txtar` — sibling class method
  call (`Helper.info()` from inside a sibling)
- `import_path_validation.txtar` — `..`/absolute/`./`/PascalCase
  rejection at parser/resolver
- `import_alias.txtar` — aliased directory import
- `import_collision.txtar` — same-segment package collision (both
  unaliased and aliased forms)
- `import_cycle.txtar` — cycle detection through directory
  packages
- `empty_package.txtar` — directory exists but contains no class
  files → clear diagnostic
- `tya_path.txtar` — `TYA_PATH` search order across multiple
  entries

## M3 — CLI surface

- `tya_check.txtar` — `tya check` applies same rules as `tya run`
- `tya_build.txtar` — `tya build` and `tya emit-c` work with
  directory packages
- `tya_test_with_pkg.txtar` — `tya test` synthesized suite
  resolves a v0.44 directory package import

## M6 — Stdlib migration sanity

- `stdlib_class_use.txtar` — entry uses migrated stdlib (math)
  through the new `pkg.Class.method` form

## Add a test

When pinning a new v0.44 behavior, prefer adding a small
`*.txtar` here over expanding an existing one — each file becomes
an independent failure target via
`go test ./tests -run TestV44Scripts/<name>`.
