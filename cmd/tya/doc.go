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
	htmlOut, paths, err := parseDocArgs(args)
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
	items, extractErr := doc.ExtractFiles(files)
	if extractErr != nil {
		fmt.Fprintln(os.Stderr, extractErr)
	}
	if htmlOut != "" {
		site := &doc.Site{Title: "API", Items: items}
		if err := site.Generate(htmlOut, os.Stderr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		return 0
	}
	if err := doc.FormatText(items, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	return 0
}

// parseDocArgs parses the v0.51 `tya doc` argument shape:
// `[--html <out>] [paths...]`. Anything after `--` is treated as a
// literal path so files literally named `--html` remain reachable.
func parseDocArgs(args []string) (htmlOut string, paths []string, err error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--html":
			i++
			if i >= len(args) {
				return "", nil, fmt.Errorf("[TYA-E0920] --html requires a directory argument")
			}
			htmlOut = args[i]
			if strings.TrimSpace(htmlOut) == "" {
				return "", nil, fmt.Errorf("[TYA-E0920] --html requires a non-empty directory argument")
			}
		case strings.HasPrefix(a, "--html="):
			htmlOut = strings.TrimPrefix(a, "--html=")
			if strings.TrimSpace(htmlOut) == "" {
				return "", nil, fmt.Errorf("[TYA-E0920] --html requires a non-empty directory argument")
			}
		case a == "--":
			paths = append(paths, args[i+1:]...)
			return htmlOut, paths, nil
		default:
			paths = append(paths, a)
		}
	}
	return htmlOut, paths, nil
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
