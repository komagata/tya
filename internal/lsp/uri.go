package lsp

import (
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
)

// URIToPath converts an LSP "file://" URI to a filesystem path.
// On Windows, a leading slash before the drive letter is stripped
// ("/C:/foo" → "C:/foo"). Non-file URIs are rejected.
func URIToPath(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("parse URI %q: %w", uri, err)
	}
	if u.Scheme != "file" {
		return "", fmt.Errorf("unsupported URI scheme %q", u.Scheme)
	}
	path := u.Path
	if runtime.GOOS == "windows" {
		if len(path) >= 3 && path[0] == '/' && path[2] == ':' {
			path = path[1:]
		}
		path = filepath.FromSlash(path)
	}
	return path, nil
}

// PathToURI converts a filesystem path to an LSP "file://" URI.
// Relative paths are made absolute first.
func PathToURI(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.ToSlash(abs)
	if runtime.GOOS == "windows" {
		// "C:/foo" → "/C:/foo" so file:// + that = file:///C:/foo
		if !strings.HasPrefix(abs, "/") {
			abs = "/" + abs
		}
	}
	u := url.URL{Scheme: "file", Path: abs}
	return u.String(), nil
}
