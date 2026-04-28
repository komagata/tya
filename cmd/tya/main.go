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
	if os.Args[1] == "--tokens" {
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		dumpTokens = true
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
	}
	if os.Args[1] == "--emit-c" {
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		emitC = true
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
	}
	if os.Args[1] == "--check-unused" {
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		checkUnused = true
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
	}
	path := os.Args[1]
	if err := runner.ValidateFileName(path); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !dumpTokens && !emitC && !checkUnused {
		if err := runner.RunFile(path, os.Stdin, os.Stdout, os.Args[2:]); err != nil {
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
	if emitC {
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
		return
	}
	if emitC {
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
