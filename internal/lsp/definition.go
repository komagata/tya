package lsp

import (
	"path/filepath"
	"strings"

	"tya/internal/ast"
	"tya/internal/lexer"
	"tya/internal/parser"
)

// DefinitionContext is the input shape DefinitionAt needs to do
// both same-file and cross-file resolution. A nil Workspace is
// allowed — same-file behaviour is the v0.52 fallback.
type DefinitionContext struct {
	Doc       *Document
	Workspace *Workspace
}

// Definition answers textDocument/definition.
func Definition(doc *Document, line, character int) ([]Location, error) {
	return DefinitionAt(DefinitionContext{Doc: doc}, line, character)
}

// DefinitionAt is the workspace-aware definition resolver.
func DefinitionAt(ctx DefinitionContext, line, character int) ([]Location, error) {
	doc := ctx.Doc
	if doc == nil {
		return nil, nil
	}
	toks, lcomments, lerrs := lexer.LexWithComments(doc.Text)
	if len(lerrs) > 0 {
		return nil, nil
	}
	prog, err := parser.ParseWithComments(toks, toCommentInfos(lcomments))
	if err != nil || prog == nil {
		return nil, nil
	}
	id := FindIdentAt(prog, line+1, character+1)
	if id == nil {
		return nil, nil
	}

	// 1) Same-file top-level resolution wins first.
	idx := BuildSymbols(prog)
	if sym, ok := idx.Lookup(id.Name); ok {
		return []Location{{
			URI:   doc.URI,
			Range: rangeAt(sym.NameTok.Line, sym.NameTok.Col, len(sym.Name)),
		}}, nil
	}

	// 2) ImportStmt binding: jump to the imported module file.
	if ctx.Workspace != nil {
		for _, stmt := range prog.Stmts {
			imp, ok := stmt.(*ast.ImportStmt)
			if !ok {
				continue
			}
			if imp.BindingName() != id.Name {
				continue
			}
			if path := resolveModuleFile(ctx.Workspace, imp.ModuleName()); path != "" {
				uri, _ := PathToURI(path)
				return []Location{{URI: uri, Range: Range{}}}, nil
			}
		}
	}

	// 3) `mod.foo` cross-file member resolution: when the cursor
	// is on the member name and the receiver is an ImportStmt
	// binding, look the binding up in the target file's symbols.
	if ctx.Workspace != nil {
		if loc := resolveCrossFileMember(ctx.Workspace, prog, id, line+1, character+1); loc != nil {
			return []Location{*loc}, nil
		}
	}

	return nil, nil
}

// resolveModuleFile returns the absolute path of the .tya file
// that implements `name`, or "" when none is found.
func resolveModuleFile(ws *Workspace, name string) string {
	if ws == nil || ws.Root == "" {
		return ""
	}
	// Conventional layouts: src/<name>.tya, <name>.tya at root,
	// <name>/<name>.tya, src/<name>/<name>.tya.
	candidates := []string{
		filepath.Join(ws.Root, "src", name+".tya"),
		filepath.Join(ws.Root, name+".tya"),
		filepath.Join(ws.Root, name, name+".tya"),
		filepath.Join(ws.Root, "src", name, name+".tya"),
	}
	for _, c := range candidates {
		if pf := ws.LoadFromDisk(c); pf != nil {
			return c
		}
	}
	return ""
}

// resolveCrossFileMember handles the `mod.foo` shape where the
// cursor sits on `foo`. It walks the file's AST for a MemberExpr
// whose Target is an Ident matching an ImportStmt binding, and
// looks `foo` up in that imported file's symbol index.
func resolveCrossFileMember(ws *Workspace, prog *ast.Program, id *ast.Ident, line, col int) *Location {
	var match *ast.MemberExpr
	walkStmts(prog.Stmts, func(node any) {
		me, ok := node.(*ast.MemberExpr)
		if !ok {
			return
		}
		if me.Name != id.Name {
			return
		}
		if me.NameTok.Line != line {
			return
		}
		if col < me.NameTok.Col || col >= me.NameTok.Col+len(me.Name) {
			return
		}
		match = me
	})
	if match == nil {
		return nil
	}
	recv, ok := match.Target.(*ast.Ident)
	if !ok {
		return nil
	}
	importName := ""
	for _, stmt := range prog.Stmts {
		imp, ok := stmt.(*ast.ImportStmt)
		if !ok {
			continue
		}
		if imp.BindingName() == recv.Name {
			importName = imp.ModuleName()
			break
		}
	}
	if importName == "" {
		return nil
	}
	path := resolveModuleFile(ws, importName)
	if path == "" {
		return nil
	}
	pf := ws.LoadFromDisk(path)
	if pf == nil || pf.Prog == nil {
		return nil
	}
	idx := BuildSymbols(pf.Prog)
	sym, ok := idx.Lookup(id.Name)
	if !ok {
		return nil
	}
	uri, _ := PathToURI(path)
	return &Location{
		URI:   uri,
		Range: rangeAt(sym.NameTok.Line, sym.NameTok.Col, len(sym.Name)),
	}
}

// modulePathOf yields the canonical relative slash-separated path
// of `name` under root, regardless of OS path separators.
func modulePathOf(root, name string) string {
	return strings.ReplaceAll(filepath.Join(root, name+".tya"), `\`, `/`)
}
