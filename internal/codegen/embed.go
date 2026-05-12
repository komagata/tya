package codegen

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"tya/internal/ast"
)

// emitEmbed handles `embed "pattern" as name` statements at codegen
// time. Patterns without `*` resolve to a single file and produce
// a bytes-typed global; patterns with `*` or `**` produce a
// dict<string, bytes> keyed by the file's path relative to the
// source file's directory.
//
// Path resolution: pattern is interpreted relative to the directory
// of g.sourcePath. When g.sourcePath is empty (e.g. tests that emit
// from synthetic sources) the current working directory is used.
func (g *cgen) emitEmbed(stmt *ast.EmbedStmt) error {
	baseDir := embedBaseDir(g.sourcePath)
	if strings.ContainsAny(stmt.Path, "*?") {
		return g.emitEmbedGlob(stmt, baseDir)
	}
	return g.emitEmbedSingle(stmt, baseDir)
}

// embedBaseDir returns the directory used to resolve embed paths
// for a given codegen sourcePath. Falls back to "." when no source
// path is available.
func embedBaseDir(sourcePath string) string {
	if sourcePath == "" {
		return "."
	}
	abs, err := filepath.Abs(sourcePath)
	if err != nil {
		return filepath.Dir(sourcePath)
	}
	return filepath.Dir(abs)
}

func (g *cgen) emitEmbedSingle(stmt *ast.EmbedStmt, baseDir string) error {
	full := embedResolve(baseDir, stmt.Path)
	data, err := os.ReadFile(full)
	if err != nil {
		return codegenError(codeEmbedNotFound,
			fmt.Sprintf("embed source not found: %s", stmt.Path),
			stmt.PathTok.Line, stmt.PathTok.Col)
	}
	target := cName(stmt.Name)
	g.line(fmt.Sprintf("%s = %s;", target, bytesLitC(data)))
	return nil
}

func (g *cgen) emitEmbedGlob(stmt *ast.EmbedStmt, baseDir string) error {
	matches, err := embedGlob(baseDir, stmt.Path)
	if err != nil {
		return codegenError(codeEmbedNotFound,
			fmt.Sprintf("embed glob error: %v", err),
			stmt.PathTok.Line, stmt.PathTok.Col)
	}
	if len(matches) == 0 {
		return codegenError(codeEmbedGlobEmpty,
			fmt.Sprintf("embed glob matched zero files: %s", stmt.Path),
			stmt.PathTok.Line, stmt.PathTok.Col)
	}
	sort.Strings(matches)
	var entries []string
	for _, rel := range matches {
		full := filepath.Join(baseDir, filepath.FromSlash(rel))
		data, rerr := os.ReadFile(full)
		if rerr != nil {
			return codegenError(codeEmbedNotFound,
				fmt.Sprintf("embed read failed for %s: %v", rel, rerr),
				stmt.PathTok.Line, stmt.PathTok.Col)
		}
		entries = append(entries, fmt.Sprintf("{%q, %s}", rel, bytesLitC(data)))
	}
	target := cName(stmt.Name)
	g.line(fmt.Sprintf("%s = tya_dict((TyaDictEntry[]){%s}, %d);",
		target, strings.Join(entries, ", "), len(entries)))
	return nil
}

// bytesLitC renders data as a C expression returning a TyaValue
// bytes instance. Mirrors the existing BytesLit emission in c.go.
func bytesLitC(data []byte) string {
	if len(data) == 0 {
		return "tya_bytes_lit((const char *)0, 0)"
	}
	var b strings.Builder
	b.WriteString("tya_bytes_lit((const char *)(unsigned char[]){")
	for i, by := range data {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "0x%02x", by)
	}
	fmt.Fprintf(&b, "}, %d)", len(data))
	return b.String()
}

// embedResolve joins baseDir with a `/`-separated source pattern.
// Absolute patterns are returned unchanged (after FromSlash).
func embedResolve(baseDir, pattern string) string {
	if path.IsAbs(pattern) {
		return filepath.FromSlash(pattern)
	}
	return filepath.Join(baseDir, filepath.FromSlash(pattern))
}

// embedGlob expands a `/`-separated pattern relative to baseDir.
// Supports `**` (recursive walk) and `*` (single-level). Returns
// `/`-normalized paths relative to baseDir, in deterministic
// order. Patterns may combine: `static/**` matches all files under
// static/; `assets/*.png` matches one-level .png files.
func embedGlob(baseDir, pattern string) ([]string, error) {
	if strings.Contains(pattern, "**") {
		return embedGlobRecursive(baseDir, pattern)
	}
	full := filepath.Join(baseDir, filepath.FromSlash(pattern))
	hits, err := filepath.Glob(full)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, h := range hits {
		info, err := os.Stat(h)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		rel, err := filepath.Rel(baseDir, h)
		if err != nil {
			continue
		}
		out = append(out, filepath.ToSlash(rel))
	}
	return out, nil
}

// embedGlobRecursive handles patterns containing `**`. The simplest
// rule: split on `**`, walk from the prefix root, and apply the
// remaining suffix as a glob against each candidate file.
//
//   "static/**"        → walk static/, accept every file.
//   "assets/**/*.png"  → walk assets/, accept files matching *.png.
func embedGlobRecursive(baseDir, pattern string) ([]string, error) {
	idx := strings.Index(pattern, "**")
	prefix := strings.TrimRight(pattern[:idx], "/")
	suffix := strings.TrimLeft(pattern[idx+2:], "/")
	root := baseDir
	if prefix != "" {
		root = filepath.Join(baseDir, filepath.FromSlash(prefix))
	}
	out := []string{}
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(baseDir, p)
		if rerr != nil {
			return rerr
		}
		relSlash := filepath.ToSlash(rel)
		if suffix != "" {
			matched, merr := path.Match(suffix, path.Base(relSlash))
			if merr != nil {
				return merr
			}
			if !matched {
				return nil
			}
		}
		out = append(out, relSlash)
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return out, nil
}
