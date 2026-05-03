> Complete one task from @.agent/tasks.json, commit it, output the Ralph task-done promise tag for that task id, and stop. Do not start another task in the same invocation.

## Project

You are working on Tya, a small indentation-based dynamic programming language implemented in Go.

Read these first:

- @docs/CLASS_MODULE_DESIGN.md
- @README.md
- @docs/NAMING.md
- @docs/STDLIB.md
- @ROADMAP.md
- @SELFHOST_WORK.md only when a change touches self-hosting fixtures or gates

## Class/Module Goal

Implement the planned semantics in @docs/CLASS_MODULE_DESIGN.md: dictionaries, sets, classes, modules, imports, one-file-one-definition rules, entry-file semantics, inheritance, interfaces, and final documentation/testing.

## Task Flow

1. Check @.agent/STEERING.md for urgent instructions. Finish those first if present.
2. Pick the highest-priority task with `"passes": false` from @.agent/tasks.json.
3. Read the matching task file at `.agent/tasks/TASK-{ID}.json`.
4. Inspect the current lexer, parser, AST, checker, eval, runner, codegen, runtime, tests, and examples before editing.
5. Implement exactly one task. Keep the slice small and compatible with existing behavior.
6. Add or update focused tests and examples for the changed behavior.
7. Run focused verification first.
8. Run `gofmt -w` on changed Go files.
9. Run `go test ./... -count=1`.
10. Run self-host/bootstrap checks only when the task affects self-hosting scripts, manifests, or generated-C behavior.
11. Update docs when semantics or user-facing behavior changes.
12. If verification passes, mark the task as `"passes": true` in @.agent/tasks.json.
13. Add a newest-first entry to @.agent/logs/LOG.md with date, task id, summary, and verification commands.
14. Commit the task with a concise Conventional Commit message.

## Completion Tags

- When the task is committed, output a promise tag whose content is `TASK-`, the task id, then `:DONE`, and stop.
- When every task in @.agent/tasks.json already has `"passes": true`, output a promise tag whose content is `COMPLETE`.
- If blocked by missing credentials, Docker/Sandbox problems, or unavailable system services, output a promise tag whose content starts with `BLOCKED:` and includes a brief reason.
- If a product or compatibility decision is required, output a promise tag whose content starts with `DECIDE:` and includes the question.

## Rules

- Work on one task per invocation.
- Do not push.
- Do not change git remotes or initialize a new repository.
- Do not rewrite unrelated user changes.
- This is not a web app. Do not start a dev server, install Playwright, run Vitest, run ESLint, or take screenshots.
- Prefer existing Go/Tya patterns over new dependencies.
