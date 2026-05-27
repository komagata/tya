package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tya/internal/doc"
)

// docCommand implements `tya doc`. It returns the process exit
// code (0 success, 1 partial errors, 2 argument / I/O errors).
func docCommand(args []string) int {
	mode, htmlOut, paths, err := parseDocArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if len(paths) == 0 {
		if _, statErr := os.Stat("src"); statErr != nil {
			fmt.Fprintln(os.Stderr, "[TYA-E0923] no src/ directory; pass paths explicitly")
			return 2
		}
		paths = []string{"src"}
	}
	files, err := docCollectFiles(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	report, extractErr := doc.ExtractReport(files)
	if extractErr != nil {
		fmt.Fprintln(os.Stderr, extractErr)
		return 2
	}
	if mode == "json" {
		if err := doc.FormatJSON(report, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		if doc.HasErrorDiagnostics(report.Diagnostics) {
			return 1
		}
		return 0
	}
	if err := doc.WriteDiagnostics(report.Diagnostics, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if mode == "html" {
		title := "API"
		if docsStdlib(paths) {
			title = "Standard Library API"
		}
		site := &doc.Site{Title: title, Items: report.Items}
		if err := site.Generate(htmlOut, os.Stderr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		if doc.HasErrorDiagnostics(report.Diagnostics) {
			return 1
		}
		return 0
	}
	if err := doc.FormatText(report.Items, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if doc.HasErrorDiagnostics(report.Diagnostics) {
		return 1
	}
	return 0
}

func docsStdlib(paths []string) bool {
	if len(paths) != 1 {
		return false
	}
	clean := filepath.Clean(paths[0])
	return clean == "lib" || strings.HasSuffix(filepath.ToSlash(clean), "/lib")
}

// parseDocArgs parses the v0.51 `tya doc` argument shape:
// `[--html <out>] [paths...]`. Anything after `--` is treated as a
// literal path so files literally named `--html` remain reachable.
func parseDocArgs(args []string) (mode string, htmlOut string, paths []string, err error) {
	mode = "text"
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--html":
			if mode != "text" {
				return "", "", nil, fmt.Errorf("[TYA-E0920] tya doc accepts only one output mode")
			}
			mode = "html"
			i++
			if i >= len(args) {
				return "", "", nil, fmt.Errorf("[TYA-E0920] --html requires a directory argument")
			}
			htmlOut = args[i]
			if strings.TrimSpace(htmlOut) == "" {
				return "", "", nil, fmt.Errorf("[TYA-E0920] --html requires a non-empty directory argument")
			}
		case strings.HasPrefix(a, "--html="):
			if mode != "text" {
				return "", "", nil, fmt.Errorf("[TYA-E0920] tya doc accepts only one output mode")
			}
			mode = "html"
			htmlOut = strings.TrimPrefix(a, "--html=")
			if strings.TrimSpace(htmlOut) == "" {
				return "", "", nil, fmt.Errorf("[TYA-E0920] --html requires a non-empty directory argument")
			}
		case a == "--json":
			if mode != "text" {
				return "", "", nil, fmt.Errorf("[TYA-E0920] tya doc accepts only one output mode")
			}
			mode = "json"
		case a == "--":
			paths = append(paths, args[i+1:]...)
			return mode, htmlOut, paths, nil
		default:
			paths = append(paths, a)
		}
	}
	return mode, htmlOut, paths, nil
}

// docCollectFiles expands directories to all `.tya` files inside
// them and accepts file paths as-is.
func docCollectFiles(paths []string) ([]string, error) {
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
