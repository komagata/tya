# Ralph Loop

This repository is configured for [PageAI-Pro/ralph-loop](https://github.com/PageAI-Pro/ralph-loop).

## Current Use

Ralph is being used to advance Tya toward complete self-hosting. The task list lives in:

```text
.agent/tasks.json
.agent/tasks/TASK-SELFHOST-*.json
```

## Run

Run one iteration with Codex:

```sh
./ralph.sh --agent codex --once
```

Run up to five iterations:

```sh
./ralph.sh --agent codex -n 5
```

If Codex authentication inside Docker Sandboxes is needed:

```sh
sbx run --name ralph-codex-tya-$(pwd | shasum -a 256 | awk '{print substr($1, 1, 8)}') codex .
```

## Steering

Use `.agent/STEERING.md` for temporary urgent instructions that should run before normal self-host tasks.

## Verification

The normal self-host gate is:

```sh
go test ./... -count=1
sh scripts/selfhost_bootstrap_check.sh
```
