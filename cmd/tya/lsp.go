package main

import (
	"fmt"
	"os"

	"tya/internal/lsp"
)

// lspCommand implements `tya lsp`. It speaks LSP JSON-RPC 2.0 on
// stdio. The only flag is `--log <file>` for debug logging; by
// default no logs are written so stderr stays free of noise that
// could confuse editor integrations.
func lspCommand(args []string) int {
	logPath := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--stdio":
			// vscode-languageclient appends this for stdio transports.
		case a == "--log":
			i++
			if i >= len(args) {
				fmt.Fprintln(os.Stderr, "[TYA-E0930] --log requires a path")
				return 2
			}
			logPath = args[i]
		case a == "--help" || a == "-h":
			fmt.Fprintln(os.Stdout, "usage: tya lsp [--log <file>]")
			return 0
		case len(a) > 6 && a[:6] == "--log=":
			logPath = a[6:]
		default:
			fmt.Fprintf(os.Stderr, "[TYA-E0930] unknown argument %q\n", a)
			return 2
		}
	}

	log := lsp.NullLogger
	if logPath != "" {
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[TYA-E0930] %v\n", err)
			return 2
		}
		defer f.Close()
		log = lsp.NewLogger(f)
	}
	lsp.Version = version
	if err := lsp.Run(os.Stdin, os.Stdout, log); err != nil {
		log.Errorf("[TYA-E0931] %v", err)
		return 1
	}
	return 0
}
