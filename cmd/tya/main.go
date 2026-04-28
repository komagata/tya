package main

import (
	"fmt"
	"os"

	"tya/internal/checker"
	"tya/internal/codegen"
	"tya/internal/eval"
	"tya/internal/lexer"
	"tya/internal/parser"
	"tya/internal/runner"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	if os.Args[1] == "--version" {
		fmt.Fprintln(os.Stdout, version)
		return
	}
	dumpTokens := false
	emitC := false
	checkUnused := false
	args := os.Args[1:]
	for len(args) > 0 {
		switch args[0] {
		case "--tokens":
			dumpTokens = true
			args = args[1:]
		case "--emit-c":
			emitC = true
			args = args[1:]
		case "--check-unused":
			checkUnused = true
			args = args[1:]
		default:
			goto doneOptions
		}
	}
doneOptions:
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}
	path := args[0]
	processArgs := args[1:]
	if err := runner.ValidateFileName(path); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !dumpTokens && !emitC && !checkUnused {
		if err := runner.RunFile(path, os.Stdin, os.Stdout, processArgs); err != nil {
			if exitErr, ok := err.(*eval.ExitError); ok {
				os.Exit(exitErr.Code)
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	source := string(src)
	if emitC && !checkUnused {
		source = runner.WithPrelude(path, source)
	}
	toks, errs := lexer.Lex(source)
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
	if dumpTokens {
		for _, tok := range toks {
			fmt.Fprintf(os.Stdout, "%d:%d\t%s\t%q\n", tok.Line, tok.Col, tok.Type, tok.Lexeme)
		}
		return
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := checker.Check(prog); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if checkUnused {
		if err := checker.CheckUnused(prog); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if !emitC {
			return
		}
	}
	if emitC {
		toks, errs = lexer.Lex(runner.WithPrelude(path, source))
		if len(errs) > 0 {
			for _, err := range errs {
				fmt.Fprintln(os.Stderr, err)
			}
			os.Exit(1)
		}
		prog, err = parser.Parse(toks)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := checker.Check(prog); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		csrc, err := codegen.EmitC(prog)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Fprint(os.Stdout, csrc)
		return
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: tya [--version] [--tokens] [--emit-c] [--check-unused] <file.tya> [args...]")
}
