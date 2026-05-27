package lsp

import (
	"fmt"
	"os"
	"path/filepath"

	"tya/internal/ast"
	"tya/internal/checker"
	"tya/internal/token"
)

// RenameError signals a conflict the server should report to the
// client as a JSON-RPC error with code TYA-E0933 attached to the
// message.
type RenameError struct {
	msg string
}

func (e *RenameError) Error() string { return e.msg }

// Rename builds a WorkspaceEdit that renames every occurrence of
// the identifier under (line, character) to newName. The scope is
// derived from ScopeKindAt:
//
//   - "top"   → all .tya files in the workspace (and the open file)
//   - "param" → the enclosing FuncLit's parameter and its body
//   - "local" → the enclosing FuncLit's body only
//
// Conflicts (e.g. newName already bound in the target scope) raise
// `*RenameError`, which the caller renders as TYA-E0933.
func Rename(ctx DefinitionContext, line, character int, newName string) (*WorkspaceEdit, error) {
	doc := ctx.Doc
	if doc == nil {
		return nil, &RenameError{msg: "[TYA-E0933] no open document for rename"}
	}
	if !isIdentifierName(newName) {
		return nil, &RenameError{msg: "[TYA-E0933] invalid identifier: " + newName}
	}
	prog := parseOrNil(doc.Text)
	if prog == nil {
		return nil, &RenameError{msg: "[TYA-E0933] cannot rename: file has parse errors"}
	}
	kind, name := ScopeKindAt(prog, line+1, character+1)
	if kind == ScopeKindUnknown || name == "" {
		return nil, &RenameError{msg: "[TYA-E0933] cursor is not on a known binding"}
	}
	if name == newName {
		return &WorkspaceEdit{Changes: map[string][]TextEdit{}}, nil
	}

	switch kind {
	case ScopeKindTopLevel:
		return renameTopLevel(ctx, doc, prog, name, newName)
	case ScopeKindLocal, ScopeKindParam:
		fn := EnclosingFunc(prog, line+1, character+1)
		if fn == nil {
			return nil, &RenameError{msg: "[TYA-E0933] enclosing function not found"}
		}
		if scopeHasName(fn, newName) {
			return nil, fmt.Errorf("[TYA-E0933] rename conflict: %q already bound in this scope", newName)
		}
		toks := FindAllParamRefs(fn, name)
		edits := tokensToEdits(toks, name, newName)
		return &WorkspaceEdit{Changes: map[string][]TextEdit{doc.URI: edits}}, nil
	}
	return nil, &RenameError{msg: "[TYA-E0933] unsupported scope"}
}

func renameTopLevel(ctx DefinitionContext, doc *Document, prog *ast.Program, name, newName string) (*WorkspaceEdit, error) {
	changes := map[string][]TextEdit{}

	// Same-file edits.
	if scopeHasTopLevelName(prog, newName) {
		return nil, fmt.Errorf("[TYA-E0933] rename conflict: %q already a top-level binding in this file", newName)
	}
	toks := FindAllIdentRefs(prog, name)
	if len(toks) > 0 {
		changes[doc.URI] = tokensToEdits(toks, name, newName)
	}

	// Cross-file edits via workspace scan.
	if ctx.Workspace != nil {
		for _, pf := range ctx.Workspace.AllFiles() {
			uri, _ := PathToURI(pf.Path)
			if uri == doc.URI || pf.Prog == nil {
				continue
			}
			if scopeHasTopLevelName(pf.Prog, newName) {
				return nil, fmt.Errorf("[TYA-E0933] rename conflict: %q already top-level in %s", newName, pf.Path)
			}
			fileToks := FindAllIdentRefs(pf.Prog, name)
			if len(fileToks) > 0 {
				changes[uri] = tokensToEdits(fileToks, name, newName)
			}
		}
	}
	return workspaceEditWithOptionalFileRename(doc, prog, changes, name, newName)
}

func workspaceEditWithOptionalFileRename(doc *Document, prog *ast.Program, changes map[string][]TextEdit, name, newName string) (*WorkspaceEdit, error) {
	edit := &WorkspaceEdit{Changes: changes}
	docChanges := textDocumentChanges(changes)
	if oldURI, newURI, ok, err := classFileRename(doc, prog, name, newName); err != nil {
		return nil, err
	} else if ok {
		docChanges = append(docChanges, RenameFileOperation{Kind: "rename", OldURI: oldURI, NewURI: newURI})
	}
	if len(docChanges) > 0 {
		edit.DocumentChanges = docChanges
	}
	return edit, nil
}

func textDocumentChanges(changes map[string][]TextEdit) []any {
	out := []any{}
	for uri, edits := range changes {
		out = append(out, TextDocumentEdit{
			TextDocument: OptionalVersionedTextDocumentIdentifier{URI: uri, Version: nil},
			Edits:        edits,
		})
	}
	return out
}

func classFileRename(doc *Document, prog *ast.Program, name, newName string) (string, string, bool, error) {
	if !declaresClass(prog, name) {
		return "", "", false, nil
	}
	path, err := URIToPath(doc.URI)
	if err != nil {
		return "", "", false, nil
	}
	newPath := filepath.Join(filepath.Dir(path), checker.SnakeCaseName(newName)+".tya")
	if _, err := filepath.Abs(newPath); err != nil {
		return "", "", false, err
	}
	if newPath != path {
		if pf := filepath.Clean(newPath); pf != filepath.Clean(path) {
			if existsPath(pf) {
				return "", "", false, fmt.Errorf("[TYA-E0933] rename conflict: target file already exists: %s", pf)
			}
		}
	}
	newURI, err := PathToURI(newPath)
	if err != nil {
		return "", "", false, err
	}
	return doc.URI, newURI, true, nil
}

func declaresClass(prog *ast.Program, name string) bool {
	for _, stmt := range prog.Stmts {
		if classDeclaresName(stmt, name) {
			return true
		}
	}
	return false
}

func classDeclaresName(stmt ast.Stmt, name string) bool {
	switch n := stmt.(type) {
	case *ast.ClassDecl:
		return n.Name == name
	case *ast.ModuleDecl:
		for _, c := range n.Classes {
			if c.Name == name {
				return true
			}
		}
	}
	return false
}

func existsPath(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func tokensToEdits(toks []token.Token, oldName, newName string) []TextEdit {
	out := make([]TextEdit, 0, len(toks))
	for _, t := range toks {
		out = append(out, TextEdit{
			Range:   rangeAt(t.Line, t.Col, len(oldName)),
			NewText: newName,
		})
	}
	return out
}

// scopeHasTopLevelName reports whether prog already has a top-level
// declaration named `name`. Used as a conservative conflict check
// for rename.
func scopeHasTopLevelName(prog *ast.Program, name string) bool {
	idx := BuildSymbols(prog)
	_, ok := idx.Lookup(name)
	return ok
}

// scopeHasName reports whether the function fn already binds name
// either as a parameter or as a local assignment target.
func scopeHasName(fn *ast.FuncLit, name string) bool {
	for _, p := range fn.Params {
		if p == name {
			return true
		}
	}
	for _, s := range fn.Body {
		if a, ok := s.(*ast.AssignStmt); ok {
			for _, t := range a.Targets {
				if ti, ok := t.(*ast.Ident); ok && ti.Name == name {
					return true
				}
			}
		}
	}
	return false
}

// isIdentifierName loosely validates a candidate tya identifier
// (ASCII letter / underscore followed by alphanumeric / underscore).
func isIdentifierName(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			// always valid
		case i > 0 && r >= '0' && r <= '9':
			// digits ok after first char
		default:
			return false
		}
	}
	return true
}
