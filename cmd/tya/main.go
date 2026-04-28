package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"tya/internal/eval"
	"tya/internal/lexer"
	"tya/internal/parser"
)

var fileNameRE = regexp.MustCompile(`^[a-z][a-z0-9_]*\.tya$`)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: tya <file.tya>")
		os.Exit(2)
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
	prog, err := parser.Parse(toks)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := eval.Run(prog, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
