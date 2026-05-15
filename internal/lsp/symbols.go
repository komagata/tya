package lsp

import (
	"strings"

	"tya/internal/ast"
	"tya/internal/doc"
	"tya/internal/token"
)

// Symbol is one indexed top-level declaration. The struct is
// intentionally LSP-agnostic so hover, definition, and completion
// share the same lookup table.
type Symbol struct {
	Name      string
	Kind      string // "function" | "class" | "module" | "interface"
	Signature string
	NameTok   token.Token
	DocBody   string // raw Markdown from the leading comment block
}

// SymbolIndex maps a binding name to its definition. The store is
// rebuilt for each request — programs are small enough that this
// is cheaper than maintaining incremental state.
type SymbolIndex struct {
	byName map[string]Symbol
	order  []string
}

// BuildSymbols walks prog and returns an index of every
// documentable top-level binding.
func BuildSymbols(prog *ast.Program) *SymbolIndex {
	idx := &SymbolIndex{byName: map[string]Symbol{}}
	if prog == nil {
		return idx
	}
	for _, stmt := range prog.Stmts {
		sym, ok := symbolFromStmt(stmt, prog)
		if !ok {
			continue
		}
		if _, exists := idx.byName[sym.Name]; !exists {
			idx.order = append(idx.order, sym.Name)
		}
		idx.byName[sym.Name] = sym
	}
	return idx
}

// Lookup returns the indexed symbol for name (exact match).
func (idx *SymbolIndex) Lookup(name string) (Symbol, bool) {
	s, ok := idx.byName[name]
	return s, ok
}

// All returns every indexed symbol in source order.
func (idx *SymbolIndex) All() []Symbol {
	out := make([]Symbol, 0, len(idx.order))
	for _, n := range idx.order {
		out = append(out, idx.byName[n])
	}
	return out
}

func symbolFromStmt(stmt ast.Stmt, prog *ast.Program) (Symbol, bool) {
	leading := leadingFor(stmt, prog)
	switch d := stmt.(type) {
	case *ast.ClassDecl:
		return Symbol{Name: d.Name, Kind: "class", Signature: "class " + d.Name, NameTok: d.NameTok, DocBody: leading}, true
	case *ast.ModuleDecl:
		return Symbol{Name: d.Name, Kind: "module", Signature: "module " + d.Name, NameTok: d.NameTok, DocBody: leading}, true
	case *ast.InterfaceDecl:
		return Symbol{Name: d.Name, Kind: "interface", Signature: "interface " + d.Name, NameTok: d.NameTok, DocBody: leading}, true
	case *ast.AssignStmt:
		if len(d.Targets) != 1 || len(d.Values) != 1 {
			return Symbol{}, false
		}
		id, ok := d.Targets[0].(*ast.Ident)
		if !ok {
			return Symbol{}, false
		}
		fn, ok := d.Values[0].(*ast.FuncLit)
		if !ok {
			return Symbol{}, false
		}
		return Symbol{
			Name:      id.Name,
			Kind:      "function",
			Signature: doc.FuncSignature(id.Name, fn),
			NameTok:   id.Tok,
			DocBody:   leading,
		}, true
	}
	return Symbol{}, false
}

func leadingFor(stmt ast.Stmt, prog *ast.Program) string {
	if prog == nil || prog.Comments == nil {
		return ""
	}
	sc, ok := prog.Comments[stmt]
	if !ok || len(sc.Leading) == 0 {
		return ""
	}
	lines := make([]string, len(sc.Leading))
	for i, l := range sc.Leading {
		if strings.HasPrefix(l, " ") {
			lines[i] = l[1:]
		} else {
			lines[i] = l
		}
	}
	return strings.Join(lines, "\n")
}
