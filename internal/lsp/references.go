package lsp

import (
	"sort"

	"tya/internal/lexer"
	"tya/internal/parser"
	"tya/internal/token"
)

// References answers textDocument/references. Scope analysis
// drives the behaviour:
//   - top-level binding  → workspace-wide scan (when a Workspace is
//                          supplied; otherwise same-file only)
//   - local / param      → enclosing FuncLit body only
//   - unknown            → no references
func References(ctx DefinitionContext, line, character int, includeDecl bool) ([]Location, error) {
	doc := ctx.Doc
	if doc == nil {
		return nil, nil
	}
	prog := parseOrNil(doc.Text)
	if prog == nil {
		return nil, nil
	}
	kind, name := ScopeKindAt(prog, line+1, character+1)
	if kind == ScopeKindUnknown || name == "" {
		return nil, nil
	}
	switch kind {
	case ScopeKindTopLevel:
		return topLevelReferences(ctx, doc, prog, name, includeDecl)
	case ScopeKindLocal, ScopeKindParam:
		fn := EnclosingFunc(prog, line+1, character+1)
		if fn == nil {
			return nil, nil
		}
		toks := FindAllParamRefs(fn, name)
		return sortLocations(tokensToLocations(doc.URI, toks, name)), nil
	}
	return nil, nil
}

func topLevelReferences(ctx DefinitionContext, doc *Document, prog any, name string, includeDecl bool) ([]Location, error) {
	out := []Location{}

	// Same-file occurrences.
	addRefs := func(uri, text string) {
		toks, lcomments, lerrs := lexer.LexWithComments(text)
		if len(lerrs) > 0 {
			return
		}
		fileProg, err := parser.ParseWithComments(toks, toCommentInfos(lcomments))
		if err != nil || fileProg == nil {
			return
		}
		for _, t := range FindAllIdentRefs(fileProg, name) {
			out = append(out, Location{URI: uri, Range: rangeAt(t.Line, t.Col, len(name))})
		}
	}
	addRefs(doc.URI, doc.Text)

	// Cross-file occurrences from the workspace.
	if ctx.Workspace != nil {
		for _, pf := range ctx.Workspace.AllFiles() {
			uri, _ := PathToURI(pf.Path)
			if uri == doc.URI {
				continue
			}
			addRefs(uri, pf.Text)
		}
	}

	if !includeDecl {
		// LSP's includeDeclaration=false should hide the *defining*
		// occurrence. v0.53 returns every occurrence (clients
		// rarely set false in practice); a future revision can
		// dedupe against BuildSymbols.
	}
	return sortLocations(out), nil
}

func tokensToLocations(uri string, toks []token.Token, name string) []Location {
	out := make([]Location, 0, len(toks))
	for _, t := range toks {
		out = append(out, Location{URI: uri, Range: rangeAt(t.Line, t.Col, len(name))})
	}
	return out
}

func sortLocations(locs []Location) []Location {
	sort.SliceStable(locs, func(i, j int) bool {
		if locs[i].URI != locs[j].URI {
			return locs[i].URI < locs[j].URI
		}
		if locs[i].Range.Start.Line != locs[j].Range.Start.Line {
			return locs[i].Range.Start.Line < locs[j].Range.Start.Line
		}
		return locs[i].Range.Start.Character < locs[j].Range.Start.Character
	})
	return locs
}
