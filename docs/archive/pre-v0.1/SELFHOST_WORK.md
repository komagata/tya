# Self-Host Work

Self-host planning now lives in `ROADMAP.md`.

This file remains as a short reference pointer so older handoffs and tests that
mention `SELFHOST_WORK.md` still have a stable place to land.

## Canonical Planning Source

Use the self-hosting Epic in `ROADMAP.md` for:

- current status
- remaining tasks
- completion criteria
- strict repeated-stage audit work
- generated-tool fallback removal work
- structured AST migration work

Do not add a separate self-host task queue here.

## Verification

Run the complete self-host gate from the repository root:

```sh
sh scripts/selfhost_bootstrap_check.sh
```

Expected output:

```text
selfhost bootstrap: ok
```

Useful focused checks:

```sh
sh scripts/stage1_selfhost_sources_check.sh
TYA_STAGE1_SELFHOST_STRICT_REPEATED=1 sh scripts/stage1_selfhost_sources_check.sh
sh scripts/selfhost_check.sh
sh scripts/selfhost_compile_check.sh
sh scripts/go_emit_selfhost_compile_check.sh
sh scripts/go_emit_selfhost_run_check.sh
sh scripts/selfhost_fixed_point_check.sh
go test ./... -count=1
```

## Key Files

- `selfhost/lexer.tya`
- `selfhost/parser.tya`
- `selfhost/checker.tya`
- `selfhost/codegen_c.tya`
- `scripts/selfhost_examples_manifest.txt`
- `scripts/stage1_selfhost_sources_check.sh`
- `scripts/selfhost_bootstrap_check.sh`
