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

// lintCommand implements
// `tya lint [--fix] [--format=text|json] [paths...]`. Each path is
// a single .tya source file or a directory; directories are walked
// recursively for files ending in ".tya". With no paths, lint
// defaults to the current directory.
//
// v0.55 rules:
//
//	TYAL0001 unused local              (autofix: line removal)
//	TYAL0002 dead code after return    (warn only)
//	TYAL0003 redundant if true/false   (autofix: unwrap-if since v0.55)
//	TYAL0004 deeply nested blocks       (warn only)
//	TYAL0005 very long functions        (warn only)
//
// `--format=text` (default) prints `path:line:col: CODE message` one
// per line. `--format=json` emits a single JSON object with a
// `findings` array. Findings on lines bearing
// `# tya-lint-ignore[: CODE[, CODE...]]` are filtered before output.
// Exit status: 0 clean / 1 findings remain / 2 arg or I/O error.
func lintCommand(args []string) int {
	fix, format, paths, err := parseLintArgs(args)
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
	var all []lintFinding
	for _, path := range files {
		out, err := lintOneFile(path, fix)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		all = append(all, out...)
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].Path != all[j].Path {
			return all[i].Path < all[j].Path
		}
		if all[i].Line != all[j].Line {
			return all[i].Line < all[j].Line
		}
		if all[i].Col != all[j].Col {
			return all[i].Col < all[j].Col
		}
		return all[i].Code < all[j].Code
	})
	switch format {
	case "json":
		report := lintReport{Version: version, Findings: all}
		if report.Findings == nil {
			report.Findings = []lintFinding{}
		}
		if err := writeLintJSON(os.Stdout, report); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
	default:
		for _, f := range all {
			fmt.Fprintf(os.Stdout, "%s:%d:%d: %s %s\n", f.Path, f.Line, f.Col, f.Code, f.Message)
		}
	}
	if len(all) > 0 {
		return 1
	}
	return 0
}

func parseLintArgs(args []string) (fix bool, format string, paths []string, err error) {
	format = "text"
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--fix":
			fix = true
		case a == "-h" || a == "--help":
			return false, "", nil, fmt.Errorf("usage: tya lint [--fix] [--format=text|json] [paths...]")
		case strings.HasPrefix(a, "--format="):
			format = a[len("--format="):]
		case a == "--format":
			if i+1 >= len(args) {
				return false, "", nil, fmt.Errorf("--format requires a value (text|json)")
			}
			i++
			format = args[i]
		case strings.HasPrefix(a, "-"):
			return false, "", nil, fmt.Errorf("unknown lint option: %s", a)
		default:
			paths = append(paths, a)
		}
	}
	if format != "text" && format != "json" {
		return false, "", nil, fmt.Errorf("--format must be text or json (got %q)", format)
	}
	return fix, format, paths, nil
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
// true, TYAL0003 `if true/false` blocks are first unwrapped in
// place, then TYAL0001 unused-local lines are dropped; both rounds
// rewrite the file on disk and only the non-autofixed remainder
// reaches the returned slice. Per-line opt-out comments suppress
// matching findings before they are returned.
func lintOneFile(path string, fix bool) ([]lintFinding, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	source := string(src)

	if fix {
		// Apply TYAL0003 unwrap-if first because it shifts line
		// numbers; TYAL0001 line-delete is computed afterwards on
		// the rewritten source.
		newSrc, changed, ferr := lintApplyUnwrapIf(source)
		if ferr != nil {
			return nil, fmt.Errorf("%s: %v", path, ferr)
		}
		if changed > 0 {
			if err := os.WriteFile(path, []byte(newSrc), 0644); err != nil {
				return nil, err
			}
			source = newSrc
		}
	}

	tokens, comments, lexErrs := lexer.LexWithComments(source)
	if len(lexErrs) > 0 {
		return nil, fmt.Errorf("%s: %v", path, lexErrs[0])
	}
	prog, _, err := parser.Parse(tokens)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}

	unused := checker.CollectUnused(prog)
	other := checker.CollectLintFindings(prog)
	optouts := buildOptouts(comments)

	if fix && len(unused) > 0 {
		newSource, fixed := fixUnusedLines(source, unused)
		if fixed > 0 {
			if err := os.WriteFile(path, []byte(newSource), 0644); err != nil {
				return nil, err
			}
		}
		unused = nil
	}

	out := []lintFinding{}
	for _, b := range unused {
		if optouts.suppressed(b.Line, "TYAL0001") {
			continue
		}
		out = append(out, lintFinding{
			Path:        path,
			Line:        b.Line,
			Col:         b.Col,
			Code:        "TYAL0001",
			Message:     fmt.Sprintf("unused local %q", b.Name),
			Autofixable: true,
		})
	}
	for _, f := range other {
		if optouts.suppressed(f.Line, f.Code) {
			continue
		}
		out = append(out, lintFinding{
			Path:        path,
			Line:        f.Line,
			Col:         f.Col,
			Code:        f.Code,
			Message:     f.Message,
			Autofixable: f.Code == "TYAL0003",
		})
	}
	return out, nil
}

// lintApplyUnwrapIf parses source, collects TYAL0003 unwrap-if hints,
// and applies them via applyUnwrapIf. Returns the rewritten source
// and the number of hints applied.
func lintApplyUnwrapIf(source string) (string, int, error) {
	tokens, lexErrs := lexer.Lex(source)
	if len(lexErrs) > 0 {
		return source, 0, lexErrs[0]
	}
	prog, _, err := parser.Parse(tokens)
	if err != nil {
		return source, 0, err
	}
	hints := checker.LintAutofixHints(prog)
	newSrc, n := applyUnwrapIf(source, hints)
	return newSrc, n, nil
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
