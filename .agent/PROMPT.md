> Complete one task from @.agent/tasks.json, commit it, output `<promise>TASK-{ID}:DONE</promise>`, and stop. Do not start another task in the same invocation.

## Project

You are working on Tya, a small indentation-based dynamic programming language implemented in Go.

Read these first:

- @SELFHOST_WORK.md
- @ROADMAP.md
- @docs/SELFHOST.md
- @README.md
- @docs/NAMING.md
- @docs/STDLIB.md

## Self-Hosting Goal

Drive Tya toward a complete self-hosted compiler: the Tya-written lexer, parser, checker, and C code generator should compile themselves through the bootstrap stages, execute the repository examples through generated tools, and reach a deterministic fixed point where regenerated generated C is stable.

## Task Flow

1. Check @.agent/STEERING.md for urgent instructions. Finish those first if present.
2. Pick the highest-priority task with `"passes": false` from @.agent/tasks.json.
3. Read the matching task file at `.agent/tasks/TASK-{ID}.json`.
4. Inspect current self-host implementation and tests before editing.
5. Implement exactly one task. Keep the slice as small as possible while satisfying the task.
6. Add or update focused tests and scripts for the changed behavior.
7. Run focused verification first.
8. Run `gofmt -w` on changed Go files.
9. Run `go test ./... -count=1`.
10. Run `sh scripts/selfhost_bootstrap_check.sh` unless the task file gives a narrower reason not to.
11. Update @SELFHOST_WORK.md, @ROADMAP.md, and @docs/SELFHOST.md when the self-hosting status changes.
12. If verification passes, mark the task as `"passes": true` in @.agent/tasks.json.
13. Add a newest-first entry to @.agent/logs/LOG.md with date, task id, summary, and verification commands.
14. Commit the task with a concise Conventional Commit message.

## Completion Tags

- When the task is committed, output exactly `<promise>TASK-{ID}:DONE</promise>` and stop.
- When every task in @.agent/tasks.json already has `"passes": true`, output exactly `<promise>COMPLETE</promise>`.
- If blocked by missing credentials, Docker/Sandbox problems, or unavailable system services, output `<promise>BLOCKED:brief reason</promise>`.
- If a product or compatibility decision is required, output `<promise>DECIDE:question</promise>`.

## Rules

- Work on one task per invocation.
- Do not push.
- Do not change git remotes or initialize a new repository.
- Do not rewrite unrelated user changes.
- This is not a web app. Do not start a dev server, install Playwright, run Vitest, run ESLint, or take screenshots.
- Prefer existing Go/Tya patterns over new dependencies.
