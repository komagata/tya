---
status: completed
goal_ready: false
---

# Feature: Flakewatch External Tool

## Goal

Create `flakewatch` as an external Tya-built CLI tool that ingests JUnit XML
test results from GitHub Actions, stores historical test outcomes in SQLite, and
reports likely flaky tests without requiring a hosted SaaS dashboard.

## Context

Many teams want flaky-test visibility but do not want to adopt a paid service or
send test history outside their repository. GitHub Actions already produces
useful run metadata, and most language test runners can emit JUnit XML directly
or through a small adapter.

Tya is a good fit for this tool because the final tool can be distributed as a
single binary and can use Tya's planned XML stdlib, SQLite external package,
HTML/template support, and asset embedding. The first version should optimize
for easy GitHub Actions adoption: store the SQLite database as a workflow
artifact or cache, ingest the current run's JUnit files, and write a useful job
summary.

Assumed repository and package identity:

- repository: `https://github.com/komagata/flakewatch`
- executable: `flakewatch`
- first release target: `v0.1.0`

## Behavior

- Provide a standalone external tool repository:

  ```text
  flakewatch/
    tya.toml
    src/
      main.tya
      flakewatch/
        Cli.tya
        Database.tya
        Ingest.tya
        JUnit.tya
        Report.tya
        Score.tya
        Server.tya
    assets/
    tests/
    examples/
    README.md
  ```

- The repository builds one `flakewatch` binary.
- The tool stores data in a local SQLite database.
- The tool ingests one or more JUnit XML files per CI run.
- The tool reports flaky-test suspects using historical pass/fail behavior.
- The tool is usable locally and inside GitHub Actions.
- The first version does not require a hosted server, cloud account, or SaaS.

## CLI

- `flakewatch init --db flakewatch.sqlite3`
  - creates the SQLite schema if missing.
- `flakewatch ingest --db flakewatch.sqlite3 --junit <glob> ...`
  - parses JUnit XML files and records one run.
- `flakewatch report --db flakewatch.sqlite3`
  - prints a human-readable flaky-test ranking.
- `flakewatch report --db flakewatch.sqlite3 --format markdown`
  - prints Markdown.
- `flakewatch report --db flakewatch.sqlite3 --format github-summary`
  - prints Markdown suitable for `$GITHUB_STEP_SUMMARY`.
- `flakewatch list --db flakewatch.sqlite3`
  - lists known test cases and current flake scores.
- `flakewatch history --db flakewatch.sqlite3 --test <test-id-or-name>`
  - shows recent outcomes for one test.
- `flakewatch serve --db flakewatch.sqlite3 --host 127.0.0.1 --port 8787`
  - serves a local HTML UI backed by the SQLite database.
- `flakewatch doctor`
  - checks SQLite, XML parsing, and GitHub Actions environment assumptions.

## Ingest Metadata

- `ingest` accepts explicit metadata flags:
  - `--repo`
  - `--workflow`
  - `--run-id`
  - `--run-attempt`
  - `--job`
  - `--sha`
  - `--branch`
  - `--event`
  - `--os`
  - `--runtime`
  - `--started-at`
- When running in GitHub Actions, missing metadata is read from environment
  variables where available:
  - `GITHUB_REPOSITORY`
  - `GITHUB_WORKFLOW`
  - `GITHUB_RUN_ID`
  - `GITHUB_RUN_ATTEMPT`
  - `GITHUB_JOB`
  - `GITHUB_SHA`
  - `GITHUB_REF_NAME`
  - `GITHUB_EVENT_NAME`
  - `RUNNER_OS`
- Explicit flags override environment-derived values.
- The ingest command can be run multiple times for the same run/job and should
  upsert idempotently by `(repo, run_id, run_attempt, job, junit file,
  test identity)`.

## JUnit XML Support

- Parse JUnit XML using the planned `xml` stdlib.
- Support top-level `<testsuite>` and `<testsuites>`.
- Record each `<testcase>` with:
  - suite name,
  - classname,
  - test name,
  - file,
  - line,
  - duration,
  - status,
  - failure/error/skipped message,
  - failure/error/skipped text body,
  - system-out,
  - system-err.
- Status mapping:
  - no `failure`, `error`, or `skipped` child means `passed`,
  - `failure` means `failed`,
  - `error` means `error`,
  - `skipped` means `skipped`.
- A testcase with both `failure` and `error` is recorded as `error` and keeps
  both child payloads if practical.
- JUnit files with zero test cases are allowed but reported as warnings.
- Malformed XML causes ingest failure by default.
- `--ignore-bad-junit` records a warning and continues with other files.

## SQLite Data Model

- Store at least these tables:
  - `runs`
  - `jobs`
  - `junit_files`
  - `test_cases`
  - `test_results`
  - `flake_scores`
- `runs` stores repository, workflow, run ID, attempt, SHA, branch, event, and
  timestamps.
- `jobs` stores job name, OS, runtime labels, and run association.
- `junit_files` stores path, digest, parse warnings, and job association.
- `test_cases` stores stable identity fields:
  - suite,
  - classname,
  - name,
  - file.
- `test_results` stores one observed result per run/job/test.
- `flake_scores` stores derived scores for fast reporting.
- Schema migrations are versioned in the database.
- `flakewatch init` and `flakewatch ingest` apply migrations automatically.

## Flake Scoring

- A flaky suspect is a test with mixed recent outcomes, such as both pass and
  fail/error observations in a time window.
- Compute at least:
  - total observations,
  - pass count,
  - fail count,
  - error count,
  - skipped count,
  - failure rate,
  - transition count between pass and non-pass,
  - last failed at,
  - last passed at,
  - last status,
  - p50 duration,
  - p95 duration.
- Default score should prioritize tests that:
  - have both pass and fail/error outcomes,
  - failed recently,
  - have repeated pass/fail transitions,
  - are observed often enough to be meaningful.
- Provide options:
  - `--window-runs N`
  - `--window-days N`
  - `--min-observations N`
  - `--branch BRANCH`
  - `--job JOB`
  - `--include-skipped`
- The exact score formula can evolve, but it must be documented and covered by
  tests.

## GitHub Actions Artifact Workflow

- The recommended v0.1 workflow stores `flakewatch.sqlite3` as a GitHub Actions
  artifact or cache.
- Document artifact-first usage:

  ```yaml
  - name: Restore flakewatch database
    uses: actions/download-artifact@v4
    continue-on-error: true
    with:
      name: flakewatch-db
      path: .flakewatch

  - name: Initialize flakewatch database
    run: flakewatch init --db .flakewatch/flakewatch.sqlite3

  - name: Ingest JUnit
    run: |
      flakewatch ingest \
        --db .flakewatch/flakewatch.sqlite3 \
        --junit "test-results/**/*.xml"

  - name: Flake summary
    run: |
      flakewatch report \
        --db .flakewatch/flakewatch.sqlite3 \
        --format github-summary >> "$GITHUB_STEP_SUMMARY"

  - name: Save flakewatch database
    uses: actions/upload-artifact@v4
    with:
      name: flakewatch-db
      path: .flakewatch/flakewatch.sqlite3
  ```

- Also document `actions/cache` as an alternative for teams that prefer cache
  behavior.
- Artifact upload/download failures should not make the test job fail by
  default in the recommended snippet unless the user chooses strict mode.
- The tool itself does not need GitHub API access in v0.1.

## Reports

- Text and Markdown reports show:
  - top flaky suspects,
  - newly failing tests,
  - recently recovered tests,
  - slowest tests by p95 duration,
  - ingest summary and warnings.
- GitHub summary output should be compact and useful inside a CI job.
- HTML UI from `serve` should be local-only by default and expose:
  - overview,
  - flaky ranking,
  - test detail history,
  - run detail,
  - filters by branch/job/status.
- The HTML UI should be bundled into the binary through asset embedding.

## Scope

- New external repository `komagata/flakewatch`.
- Tya CLI implementation.
- SQLite schema and migrations.
- JUnit XML ingestion.
- Flake scoring.
- Text, Markdown, GitHub summary, and local HTML reports.
- GitHub Actions README examples using artifact storage.
- Test fixtures for representative JUnit XML from multiple ecosystems.
- End-to-end fixture that ingests multiple runs and reports a flaky test.
- Release build instructions for a single binary.

## Out of Scope

- Hosted SaaS dashboard.
- GitHub App or OAuth login.
- GitHub API synchronization in v0.1.
- Automatic test reruns.
- Direct parsing of non-JUnit test formats.
- Language-specific test runner integrations beyond documentation examples.
- Multi-repository central server.
- Team/user permissions.
- Notifications to Slack, email, or pull request comments.
- Storing the SQLite DB in a remote object store in v0.1.

## Acceptance Criteria

- A separate `komagata/flakewatch` repository builds a single `flakewatch`
  binary.
- `flakewatch init` creates a usable SQLite database.
- `flakewatch ingest` records JUnit XML from both `<testsuite>` and
  `<testsuites>` fixtures.
- JUnit fixtures with failures, errors, skipped tests, `system-out`,
  `system-err`, attributes, and CDATA are parsed correctly.
- Running ingest twice for the same run/job/test does not duplicate results.
- Ingesting multiple historical runs produces stable flake scores.
- A test that alternates pass/fail is ranked above an always-failing test when
  enough passing observations exist.
- `report --format github-summary` emits Markdown suitable for
  `$GITHUB_STEP_SUMMARY`.
- `serve` starts a local web UI and can show flaky ranking and test history.
- The recommended GitHub Actions artifact workflow is documented and works in a
  fixture repository or integration example.
- The tool works without network access after the binary and prior database
  artifact are available.
- Missing or malformed JUnit input produces clear diagnostics.

## Verification

In the external repository:

```sh
tya test
tya run src/main.tya init --db tmp/flakewatch.sqlite3
tya run src/main.tya ingest --db tmp/flakewatch.sqlite3 --junit "tests/fixtures/junit/**/*.xml"
tya run src/main.tya report --db tmp/flakewatch.sqlite3 --format markdown
tya run src/main.tya serve --db tmp/flakewatch.sqlite3 --host 127.0.0.1 --port 8787
```

For this repository's spec tracking only:

```sh
test -f docs/prd/flakewatch-external-tool.md
rg -n "Flakewatch External Tool" docs/prd/flakewatch-external-tool.md
```

## Dependencies

- Depends on the planned `xml` stdlib for JUnit XML ingestion.
- Depends on the planned `sqlite` external library.
- Uses Tya asset embedding for bundled local HTML UI assets.
- Benefits from routing/template stdlib work for `serve`, but the first version
  may generate simple HTML directly if those features are not available yet.

## Open Questions

None.
