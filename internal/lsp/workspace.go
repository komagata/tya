package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"tya/internal/ast"
	"tya/internal/lexer"
	"tya/internal/parser"
	"tya/internal/pkg"
)

// Workspace is the per-server view of a tya project. It is created
// lazily on the first incoming document (didOpen) so single-file
// edits in a non-project directory stay fast.
type Workspace struct {
	mu       sync.RWMutex
	Root     string // absolute path of the directory holding tya.toml; "" when none
	Manifest *pkg.Manifest

	// parsed caches per-file AST keyed by absolute filesystem path.
	parsed map[string]*ParsedFile
}

// ParsedFile is the cached lex/parse outcome for one source file.
type ParsedFile struct {
	Path string
	Text string
	Prog *ast.Program
}

// NewWorkspace returns an empty workspace. EnsureRoot is the
// idempotent entry point that callers invoke when they have a URI
// they want to resolve.
func NewWorkspace() *Workspace {
	return &Workspace{parsed: map[string]*ParsedFile{}}
}

// EnsureRoot makes sure the workspace has discovered (or
// remembered the absence of) the enclosing tya.toml relative to
// the document at uri. Subsequent calls are no-ops.
func (w *Workspace) EnsureRoot(uri string) {
	path, err := URIToPath(uri)
	if err != nil {
		return
	}
	dir := filepath.Dir(path)
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.Root != "" {
		return
	}
	if root, manifestPath, err := pkg.FindManifest(dir); err == nil {
		w.Root = root
		if m, merr := pkg.ReadManifest(manifestPath); merr == nil {
			w.Manifest = m
		}
	}
}

// PutText stores src as the latest known text for path and parses
// it eagerly. This is the path the server uses when it has the
// authoritative buffer in hand (didOpen / didChange).
func (w *Workspace) PutText(path, src string) {
	prog := parseOrNil(src)
	w.mu.Lock()
	defer w.mu.Unlock()
	w.parsed[path] = &ParsedFile{Path: path, Text: src, Prog: prog}
}

// Drop removes a cached file (used on didClose).
func (w *Workspace) Drop(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.parsed, path)
}

// Get returns the cached file or nil when nothing is known.
func (w *Workspace) Get(path string) *ParsedFile {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.parsed[path]
}

// LoadFromDisk returns the parsed file for path, reading from disk
// when no in-memory buffer exists. Successful reads are cached so
// subsequent lookups skip the I/O.
func (w *Workspace) LoadFromDisk(path string) *ParsedFile {
	if f := w.Get(path); f != nil {
		return f
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	src := string(data)
	prog := parseOrNil(src)
	pf := &ParsedFile{Path: path, Text: src, Prog: prog}
	w.mu.Lock()
	w.parsed[path] = pf
	w.mu.Unlock()
	return pf
}

// AllFiles returns every cached file plus any sibling `.tya` files
// under the workspace root that are not yet cached, loading them
// on demand. The order is deterministic (sorted by path).
func (w *Workspace) AllFiles() []*ParsedFile {
	w.mu.RLock()
	root := w.Root
	w.mu.RUnlock()
	paths := []string{}
	if root != "" {
		_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				name := d.Name()
				if name == ".tya" || name == "node_modules" || name == ".git" || name == "dist" {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(p, ".tya") {
				paths = append(paths, p)
			}
			return nil
		})
	}
	// Always include cached files (covers buffers outside Root).
	seen := map[string]bool{}
	for _, p := range paths {
		seen[p] = true
	}
	w.mu.RLock()
	for p := range w.parsed {
		if !seen[p] {
			paths = append(paths, p)
		}
	}
	w.mu.RUnlock()
	out := make([]*ParsedFile, 0, len(paths))
	for _, p := range paths {
		if pf := w.LoadFromDisk(p); pf != nil {
			out = append(out, pf)
		}
	}
	return out
}

func parseOrNil(src string) *ast.Program {
	toks, lcomments, lerrs := lexer.LexWithComments(src)
	if len(lerrs) > 0 {
		return nil
	}
	prog, _, err := parser.ParseWithComments(toks, toCommentInfos(lcomments))
	if err != nil {
		return nil
	}
	return prog
}
