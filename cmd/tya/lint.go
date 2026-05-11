package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tya/internal/checker"
	"tya/internal/lexer"
	"tya/internal/parser"
)

// lintCommand implements `tya lint [paths...]`. Each path is a single
// .tya source file or a directory; directories are walked recursively
// for files ending in ".tya". With no paths, lint defaults to the
// current directory.
//
// v0.49 ships exactly one rule: TYAL0001 unused local. Findings are
// printed in `path:line:col: TYAL0001 unused local 'name'` form, one
// per line, sorted by path/line/column. tya exits 1 when any finding
// was reported, 0 when the corpus is clean, and 2 on argument or I/O
// errors.
func lintCommand(args []string) int {
	files, err := lintCollectFiles(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "tya lint: no .tya files found")
		return 2
	}
	findings := []string{}
	for _, path := range files {
		out, err := lintFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		findings = append(findings, out...)
	}
	sort.Strings(findings)
	for _, line := range findings {
		fmt.Fprintln(os.Stdout, line)
	}
	if len(findings) > 0 {
		return 1
	}
	return 0
}

func lintCollectFiles(paths []string) ([]string, error) {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	out := []string{}
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			err = filepath.WalkDir(p, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				if strings.HasSuffix(path, ".tya") {
					out = append(out, path)
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out, nil
}

func lintFile(path string) ([]string, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	tokens, lexErrs := lexer.Lex(string(src))
	if len(lexErrs) > 0 {
		return nil, fmt.Errorf("%s: %v", path, lexErrs[0])
	}
	prog, err := parser.Parse(tokens)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	out := []string{}
	for _, b := range checker.CollectUnused(prog) {
		out = append(out, fmt.Sprintf("%s:%d:%d: TYAL0001 unused local %q", path, b.Line, b.Col, b.Name))
	}
	return out, nil
}
