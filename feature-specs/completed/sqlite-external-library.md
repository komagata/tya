---
status: completed
goal_ready: false
---

# Feature: SQLite External Library

## Goal

Create an external Tya package that exposes SQLite through the native package
system, giving applications a small, dependable embedded database API without
adding SQLite to the Tya standard library.

## Context

Tya supports external packages through `tya.toml`, `tya.lock`, path/git
dependencies, and native package metadata. SQLite is a good fit for that model:
it is useful for many applications, but it brings C linking, host dependency,
transaction, statement, and error-surface decisions that should evolve outside
the core stdlib first.

Assumed repository and package identity:

- repository: `https://github.com/komagata/tya-sqlite`
- package name: `sqlite`
- import path: `import sqlite as sqlite`
- first release target: `v0.1.0`

## Behavior

- Provide an external native package with this layout:

  ```text
  tya-sqlite/
    tya.toml
    src/sqlite/
      Connection.tya
      Statement.tya
      Transaction.tya
      Row.tya
      Result.tya
    native/sqlite_binding.c
    include/sqlite_binding.h
    tests/sqlite_test.tya
    examples/
    README.md
  ```

- Applications consume the package through a git dependency:

  ```toml
  [dependencies]
  sqlite = { git = "https://github.com/komagata/tya-sqlite", tag = "v0.1.0" }
  ```

- The package manifest declares the native SQLite dependency through
  `pkg-config`:

  ```toml
  [native]
  sources = ["native/sqlite_binding.c"]
  headers = ["include/sqlite_binding.h"]
  include_dirs = ["include"]
  pkg_config = ["sqlite3"]
  cflags = []
  ldflags = []
  ```

- Public API:

  ```tya
  import sqlite as sqlite

  db = sqlite.Connection.open("app.db")
  db.exec("create table if not exists users (id integer primary key, name text)")
  db.exec("insert into users (name) values (?)", ["Tya"])

  rows = db.query("select id, name from users order by id")
  for row in rows
    println row["name"]

  db.close()
  ```

- `Connection.open(path)` opens or creates a database file.
- `Connection.open(path, options)` accepts:
  - `readonly: true`,
  - `create: true | false`,
  - `memory: true`,
  - `busy_timeout_ms: number`.
- `Connection.memory()` opens an in-memory database.
- `conn.close()` closes the connection. Closing twice is a no-op.
- Operations on a closed connection raise a SQLite error.
- `conn.exec(sql)` runs one or more SQL statements and returns a `Result`.
- `conn.exec(sql, params)` binds positional parameters and returns a `Result`.
- `conn.query(sql)` returns an array of row dictionaries.
- `conn.query(sql, params)` binds positional parameters and returns rows.
- `conn.prepare(sql)` returns a `Statement`.
- `conn.transaction(fn)` runs `fn(tx)` inside a transaction:
  - commit when `fn` returns normally,
  - rollback when `fn` raises,
  - return the function result after commit.
- `conn.begin()` returns a `Transaction` for manual transaction control.

## Statements

- `stmt.bind(params)` binds positional parameters and returns the statement.
- `stmt.exec()` executes a non-query statement and returns `Result`.
- `stmt.query()` returns an array of row dictionaries.
- `stmt.each(fn)` calls `fn(row)` for each row and returns `nil`.
- `stmt.reset()` resets the statement so it can be reused.
- `stmt.close()` finalizes the statement. Closing twice is a no-op.
- A statement is invalid after its connection is closed.

## Values and Rows

- Supported parameter and column values:
  - `nil` ↔ SQL `NULL`,
  - boolean ↔ integer `0` / `1`,
  - number ↔ SQLite numeric value,
  - string ↔ SQLite text,
  - bytes ↔ SQLite blob.
- Row dictionaries use column names as keys.
- Duplicate column names are disambiguated by suffixing later duplicates with
  `_2`, `_3`, and so on.
- `Row` may be implemented as a plain dictionary in v0.1.0. A dedicated
  `sqlite.Row` class is optional unless it removes real complexity.
- `Result` exposes:
  - `changes`,
  - `last_insert_rowid`.

## Errors

- SQLite failures raise package errors with messages that include:
  - operation,
  - SQLite result code name when available,
  - SQLite error message.
- Constraint violations, busy/locked databases, invalid SQL, bind-count
  mismatch, closed resources, and missing host dependencies must be
  distinguishable by message at minimum.
- The first version does not need a full error-code class hierarchy.

## Native Boundary

- Native functions are thin wrappers around SQLite C APIs:
  - open/close,
  - exec,
  - prepare/finalize/reset,
  - bind,
  - step,
  - column metadata/value extraction,
  - changes,
  - last insert rowid,
  - busy timeout.
- Native resources must be represented as Tya resources or another package-safe
  handle type supported by the native package system.
- Prepared statements and connections must finalize/close their SQLite handles
  deterministically when `close()` is called.
- The library should avoid relying on finalizers for correctness.

## Scope

- New external repository `komagata/tya-sqlite`.
- `tya.toml` native package manifest using `pkg_config = ["sqlite3"]`.
- Tya wrapper classes under `src/sqlite/`.
- C native binding under `native/` and public package header under `include/`.
- Tests for connection lifecycle, exec, query, prepared statements,
  transactions, values, blobs, errors, and busy timeout.
- Examples:
  - basic CRUD,
  - prepared statements,
  - transaction,
  - in-memory database.
- README documenting installation, system dependency requirements, API,
  transactions, value mapping, and error behavior.

## Out of Scope

- Adding SQLite to Tya stdlib.
- A central package registry or `tya publish`.
- ORM, migrations framework, query builder, schema DSL, or connection pool.
- Async query API.
- Cross-compiling SQLite for WASM.
- Bundling the SQLite amalgamation in the first version.
- Loading SQLite extensions.
- FTS, JSON1, RTree, or other optional SQLite extension APIs as first-class
  wrappers.
- Full typed error hierarchy.

## Acceptance Criteria

- A separate `komagata/tya-sqlite` repository contains a valid native Tya
  package manifest.
- A project can depend on the package by git URL, run `tya install`, import
  `sqlite`, and compile/link against host `sqlite3` through `pkg-config`.
- `sqlite.Connection.open("test.db")` opens or creates a database file.
- `sqlite.Connection.memory()` opens an in-memory database.
- `conn.exec` can create tables and insert rows.
- `conn.query` returns row dictionaries with correct column names and values.
- Positional parameters bind `nil`, booleans, numbers, strings, and bytes.
- Prepared statements can be reused with `reset`.
- `conn.transaction(fn)` commits on normal return and rolls back on raised
  errors.
- `Result.changes` and `Result.last_insert_rowid` report correct values for
  inserts and updates.
- Closing connections and statements releases native resources and makes later
  use raise a clear error.
- Missing `sqlite3` `pkg-config` support fails with a diagnostic from the native
  package build path.
- The package test suite passes through `tya test`.
- A fixture app proves path dependency consumption from outside the package
  repository.

## Verification

```sh
pkg-config --exists sqlite3
tya install
tya doctor native
tya test
tya run examples/basic.tya
tya run examples/transaction.tya
```

For this repository's spec tracking only:

```sh
test -f feature-specs/sqlite-external-library.md
rg -n "SQLite External Library" feature-specs/sqlite-external-library.md
```

## Dependencies

- Requires completed native package support.
- Requires host `sqlite3` development files discoverable through `pkg-config`.
- Uses existing Tya bytes support for BLOB values.

## Open Questions

None.
