package lsp

import (
	"tya/internal/ast"
	"tya/internal/token"
)

// DocumentSymbolsFor returns the hierarchical DocumentSymbol list
// for the document at doc. Classes report their methods as
// children; module / interface bodies expose member functions /
// methods the same way.
func DocumentSymbolsFor(doc *Document) []DocumentSymbol {
	if doc == nil {
		return nil
	}
	prog := parseOrNil(doc.Text)
	if prog == nil {
		return nil
	}
	out := []DocumentSymbol{}
	for _, stmt := range prog.Stmts {
		switch n := stmt.(type) {
		case *ast.ClassDecl:
			out = append(out, classSymbol(n))
		case *ast.ModuleDecl:
			out = append(out, moduleSymbol(n))
		case *ast.InterfaceDecl:
			out = append(out, interfaceSymbol(n))
		case *ast.AssignStmt:
			if id, ok := assignFunctionTarget(n); ok {
				if fn, ok := n.Values[0].(*ast.FuncLit); ok {
					out = append(out, DocumentSymbol{
						Name:           id.Name,
						Kind:           SymbolKindFunction,
						Detail:         identifierSignature(id.Name, fn),
						Range:          tokenRange(id.Tok, len(id.Name)),
						SelectionRange: tokenRange(id.Tok, len(id.Name)),
					})
				}
			}
		}
	}
	return out
}

func assignFunctionTarget(a *ast.AssignStmt) (*ast.Ident, bool) {
	if len(a.Targets) != 1 || len(a.Values) != 1 {
		return nil, false
	}
	id, ok := a.Targets[0].(*ast.Ident)
	if !ok {
		return nil, false
	}
	if _, ok := a.Values[0].(*ast.FuncLit); !ok {
		return nil, false
	}
	return id, true
}

func identifierSignature(name string, fn *ast.FuncLit) string {
	if fn == nil {
		return name
	}
	return name + "(" + joinParams(fn.Params) + ")"
}

func joinParams(params []string) string {
	out := ""
	for i, p := range params {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}

func classSymbol(c *ast.ClassDecl) DocumentSymbol {
	children := []DocumentSymbol{}
	for _, m := range c.Methods {
		if m.Func == nil {
			continue
		}
		children = append(children, DocumentSymbol{
			Name:           m.Name,
			Kind:           SymbolKindMethod,
			Detail:         identifierSignature(m.Name, m.Func),
			Range:          tokenRange(m.Tok, len(m.Name)),
			SelectionRange: tokenRange(m.Tok, len(m.Name)),
		})
	}
	for _, f := range c.Fields {
		children = append(children, DocumentSymbol{
			Name:           f.Name,
			Kind:           SymbolKindProperty,
			Range:          tokenRange(f.Tok, len(f.Name)),
			SelectionRange: tokenRange(f.Tok, len(f.Name)),
		})
	}
	for _, constant := range c.Constants {
		children = append(children, DocumentSymbol{
			Name:           constant.Name,
			Kind:           SymbolKindConstant,
			Range:          tokenRange(constant.Tok, len(constant.Name)),
			SelectionRange: tokenRange(constant.Tok, len(constant.Name)),
		})
	}
	return DocumentSymbol{
		Name:           c.Name,
		Kind:           SymbolKindClass,
		Range:          tokenRange(c.NameTok, len(c.Name)),
		SelectionRange: tokenRange(c.NameTok, len(c.Name)),
		Children:       children,
	}
}

func moduleSymbol(m *ast.ModuleDecl) DocumentSymbol {
	children := []DocumentSymbol{}
	for _, mem := range m.Members {
		kind := SymbolKindVariable
		detail := ""
		if fn, ok := mem.Value.(*ast.FuncLit); ok {
			kind = SymbolKindFunction
			detail = identifierSignature(mem.Name, fn)
		}
		children = append(children, DocumentSymbol{
			Name:           mem.Name,
			Kind:           kind,
			Detail:         detail,
			Range:          tokenRange(mem.Tok, len(mem.Name)),
			SelectionRange: tokenRange(mem.Tok, len(mem.Name)),
		})
	}
	return DocumentSymbol{
		Name:           m.Name,
		Kind:           SymbolKindModule,
		Range:          tokenRange(m.NameTok, len(m.Name)),
		SelectionRange: tokenRange(m.NameTok, len(m.Name)),
		Children:       children,
	}
}

func interfaceSymbol(i *ast.InterfaceDecl) DocumentSymbol {
	children := []DocumentSymbol{}
	for _, mth := range i.Methods {
		children = append(children, DocumentSymbol{
			Name:           mth.Name,
			Kind:           SymbolKindMethod,
			Range:          tokenRange(mth.Tok, len(mth.Name)),
			SelectionRange: tokenRange(mth.Tok, len(mth.Name)),
		})
	}
	return DocumentSymbol{
		Name:           i.Name,
		Kind:           SymbolKindInterface,
		Range:          tokenRange(i.NameTok, len(i.Name)),
		SelectionRange: tokenRange(i.NameTok, len(i.Name)),
		Children:       children,
	}
}

func tokenRange(t token.Token, length int) Range {
	return rangeAt(t.Line, t.Col, length)
}
