package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"tya/internal/checker"
	"tya/internal/eval"
	"tya/internal/lexer"
	"tya/internal/parser"
)

const version = "0.1.0"

var fileNameRE = regexp.MustCompile(`^[a-z][a-z0-9_]*\.tya$`)

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
	if os.Args[1] == "--tokens" {
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		dumpTokens = true
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
	}
	path := os.Args[1]
	if filepath.Ext(path) != ".tya" || !fileNameRE.MatchString(filepath.Base(path)) {
		fmt.Fprintf(os.Stderr, "invalid Tya file name: %s\n", filepath.Base(path))
		os.Exit(1)
	}
	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	toks, errs := lexer.Lex(string(src))
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
	if err := eval.RunWithIO(prog, os.Stdin, os.Stdout, os.Args[2:]); err != nil {
		if exitErr, ok := err.(*eval.ExitError); ok {
			os.Exit(exitErr.Code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: tya [--version] [--tokens] <file.tya> [args...]")
}
