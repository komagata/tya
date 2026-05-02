# Tya Self-Hosting Summary

Tya is a small indentation-based dynamic language implemented in Go. The current repo contains a Go lexer, parser, AST, checker, interpreter, C emitter, C runtime, stdlib prelude, examples, and Tya-written compiler pieces under `selfhost/`.

The long-term target is a complete self-hosted compiler. The immediate Ralph loop should advance the Tya-written lexer, parser, checker, and C code generator from the current bootstrap subset toward full repository example parity and deterministic fixed-point regeneration.

Primary verification:

- `go test ./... -count=1`
- `sh scripts/selfhost_bootstrap_check.sh`

Useful focused checks:

- `sh scripts/selfhost_check.sh`
- `sh scripts/selfhost_compile_check.sh`
- `sh scripts/go_emit_selfhost_compile_check.sh`
- `sh scripts/go_emit_selfhost_run_check.sh`
- `sh scripts/stage1_selfhost_sources_check.sh`

Important files:

- `SELFHOST_WORK.md`: current self-host queue and restart protocol
- `ROADMAP.md`: durable project roadmap
- `docs/SELFHOST.md`: user-facing self-hosting status
- `selfhost/lexer.tya`: Tya lexer
- `selfhost/parser.tya`: Tya parser
- `selfhost/checker.tya`: Tya checker
- `selfhost/codegen_c.tya`: Tya C code generator
- `scripts/selfhost_bootstrap_check.sh`: full self-host gate
