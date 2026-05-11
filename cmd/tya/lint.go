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

// lintCommand implements `tya lint [--fix] [paths...]`. Each path is
// a single .tya source file or a directory; directories are walked
// recursively for files ending in ".tya". With no paths, lint
// defaults to the current directory.
//
// v0.50 rules:
//
//	TYAL0001 unused local           (autofix: line removal)
//	TYAL0003 redundant if true/false (warn only in v0.50)
//	TYAL0004 deeply nested blocks    (warn only)
//	TYAL0005 very long functions     (warn only)
//
// Findings stream to stdout sorted by path/line/col. tya exits 1
// when any finding was reported, 0 when clean, and 2 on argument
// or I/O errors. With --fix, autofixable findings are applied
// in-place and only the non-fixed remainder counts toward exit 1.
func lintCommand(args []string) int {
	fix, paths, err := parseLintArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	files, err := lintCollectFiles(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "tya lint: no .tya files found")
		return 2
	}
	remaining := []string{}
	for _, path := range files {
		out, err := lintOneFile(path, fix)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		remaining = append(remaining, out...)
	}
	sort.Strings(remaining)
	for _, line := range remaining {
		fmt.Fprintln(os.Stdout, line)
	}
	if len(remaining) > 0 {
		return 1
	}
	return 0
}

func parseLintArgs(args []string) (fix bool, paths []string, err error) {
	for _, a := range args {
		switch a {
		case "--fix":
			fix = true
		case "-h", "--help":
			return false, nil, fmt.Errorf("usage: tya lint [--fix] [paths...]")
		default:
			if strings.HasPrefix(a, "-") {
				return false, nil, fmt.Errorf("unknown lint option: %s", a)
			}
			paths = append(paths, a)
		}
	}
	return fix, paths, nil
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

// lintOneFile collects findings for a single source. When fix is
// true, autofixable findings are applied to the file in place and
// removed from the returned slice; only non-autofixed findings
// remain for stdout reporting.
func lintOneFile(path string, fix bool) ([]string, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	source := string(src)
	tokens, lexErrs := lexer.Lex(source)
	if len(lexErrs) > 0 {
		return nil, fmt.Errorf("%s: %v", path, lexErrs[0])
	}
	prog, err := parser.Parse(tokens)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}

	unused := checker.CollectUnused(prog)
	other := checker.CollectLintFindings(prog)

	if fix && len(unused) > 0 {
		newSource, fixed := fixUnusedLines(source, unused)
		if fixed > 0 {
			if err := os.WriteFile(path, []byte(newSource), 0644); err != nil {
				return nil, err
			}
		}
		// every unused local is autofixable in v0.50
		unused = nil
	}

	out := []string{}
	for _, b := range unused {
		out = append(out, fmt.Sprintf("%s:%d:%d: TYAL0001 unused local %q", path, b.Line, b.Col, b.Name))
	}
	for _, f := range other {
		out = append(out, fmt.Sprintf("%s:%d:%d: %s %s", path, f.Line, f.Col, f.Code, f.Message))
	}
	return out, nil
}

// fixUnusedLines removes any source line that introduces one of the
// unused bindings. We match by (line, col, name) — a binding lives
// at the head of an `assign` statement so its column points at the
// name token, and the surrounding line is safe to drop verbatim.
// Returns the new source and the number of lines removed.
func fixUnusedLines(source string, unused []checker.UnusedBinding) (string, int) {
	lines := strings.Split(source, "\n")
	dropLine := make(map[int]bool)
	for _, b := range unused {
		if b.Line < 1 || b.Line > len(lines) {
			continue
		}
		idx := b.Line - 1
		trimmed := strings.TrimSpace(lines[idx])
		// only drop if the line actually starts with the binding name
		if !strings.HasPrefix(trimmed, b.Name) {
			continue
		}
		dropLine[idx] = true
	}
	if len(dropLine) == 0 {
		return source, 0
	}
	out := make([]string, 0, len(lines)-len(dropLine))
	for i, line := range lines {
		if dropLine[i] {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n"), len(dropLine)
}
