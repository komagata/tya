package lsp

import (
	"strings"

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
		case *ast.StructDecl:
			out = append(out, structSymbol(n))
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
						Detail:         functionSignature(fn),
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

func functionSignature(fn *ast.FuncLit) string {
	if fn == nil {
		return ""
	}
	return "(" + joinParams(fn.Params) + ")"
}

func symbolDetail(base string, parts ...string) string {
	out := []string{}
	if base != "" {
		out = append(out, base)
	}
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	return strings.Join(out, " ")
}

func privateModifier(private bool) string {
	if private {
		return "private"
	}
	return ""
}

func methodVisibilityModifier(m ast.ClassMethod) string {
	if m.Private {
		return "private"
	}
	if m.Protected {
		return "protected"
	}
	return ""
}

func classMethodDetail(m ast.ClassMethod) string {
	parts := []string{functionSignature(m.Func), methodVisibilityModifier(m)}
	if m.Class {
		parts = append(parts, "static")
	}
	if m.Abstract {
		parts = append(parts, "abstract")
	}
	if m.Override {
		parts = append(parts, "override")
	}
	return symbolDetail("", parts...)
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
		kind := SymbolKindMethod
		if m.Name == "initialize" {
			kind = SymbolKindConstructor
		}
		children = append(children, DocumentSymbol{
			Name:           m.Name,
			Kind:           kind,
			Detail:         classMethodDetail(m),
			Range:          tokenRange(m.Tok, len(m.Name)),
			SelectionRange: tokenRange(m.Tok, len(m.Name)),
		})
	}
	for _, f := range c.Fields {
		children = append(children, DocumentSymbol{
			Name:           f.Name,
			Kind:           SymbolKindField,
			Detail:         privateModifier(f.Private),
			Range:          tokenRange(f.Tok, len(f.Name)),
			SelectionRange: tokenRange(f.Tok, len(f.Name)),
		})
	}
	for _, constant := range c.Constants {
		children = append(children, DocumentSymbol{
			Name:           constant.Name,
			Kind:           SymbolKindConstant,
			Detail:         privateModifier(constant.Private),
			Range:          tokenRange(constant.Tok, len(constant.Name)),
			SelectionRange: tokenRange(constant.Tok, len(constant.Name)),
		})
	}
	for _, variable := range c.Vars {
		children = append(children, DocumentSymbol{
			Name:           variable.Name,
			Kind:           SymbolKindVariable,
			Detail:         symbolDetail("", privateModifier(variable.Private), "static"),
			Range:          tokenRange(variable.Tok, len(variable.Name)),
			SelectionRange: tokenRange(variable.Tok, len(variable.Name)),
		})
	}
	selection := tokenRange(c.NameTok, len(c.Name))
	return DocumentSymbol{
		Name:           c.Name,
		Kind:           SymbolKindClass,
		Range:          enclosingRange(selection, children),
		SelectionRange: selection,
		Children:       children,
	}
}

func structSymbol(s *ast.StructDecl) DocumentSymbol {
	children := []DocumentSymbol{}
	for _, f := range s.Fields {
		children = append(children, DocumentSymbol{
			Name:           f.Name,
			Kind:           SymbolKindField,
			Range:          tokenRange(f.Tok, len(f.Name)),
			SelectionRange: tokenRange(f.Tok, len(f.Name)),
		})
	}
	selection := tokenRange(s.NameTok, len(s.Name))
	kind := SymbolKindStruct
	if s.Record {
		kind = SymbolKindObject
	}
	return DocumentSymbol{
		Name:           s.Name,
		Kind:           kind,
		Range:          enclosingRange(selection, children),
		SelectionRange: selection,
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
			detail = functionSignature(fn)
		}
		children = append(children, DocumentSymbol{
			Name:           mem.Name,
			Kind:           kind,
			Detail:         detail,
			Range:          tokenRange(mem.Tok, len(mem.Name)),
			SelectionRange: tokenRange(mem.Tok, len(mem.Name)),
		})
	}
	selection := tokenRange(m.NameTok, len(m.Name))
	return DocumentSymbol{
		Name:           m.Name,
		Kind:           SymbolKindModule,
		Range:          enclosingRange(selection, children),
		SelectionRange: selection,
		Children:       children,
	}
}

func interfaceSymbol(i *ast.InterfaceDecl) DocumentSymbol {
	children := []DocumentSymbol{}
	for _, mth := range i.Methods {
		children = append(children, DocumentSymbol{
			Name:           mth.Name,
			Kind:           SymbolKindMethod,
			Detail:         functionSignature(mth.Func),
			Range:          tokenRange(mth.Tok, len(mth.Name)),
			SelectionRange: tokenRange(mth.Tok, len(mth.Name)),
		})
	}
	selection := tokenRange(i.NameTok, len(i.Name))
	return DocumentSymbol{
		Name:           i.Name,
		Kind:           SymbolKindInterface,
		Range:          enclosingRange(selection, children),
		SelectionRange: selection,
		Children:       children,
	}
}

func tokenRange(t token.Token, length int) Range {
	return rangeAt(t.Line, t.Col, length)
}

func enclosingRange(base Range, children []DocumentSymbol) Range {
	out := base
	for _, child := range children {
		out = expandRange(out, child.Range)
	}
	return out
}

func expandRange(a, b Range) Range {
	if positionBefore(b.Start, a.Start) {
		a.Start = b.Start
	}
	if positionBefore(a.End, b.End) {
		a.End = b.End
	}
	return a
}

func positionBefore(a, b Position) bool {
	if a.Line != b.Line {
		return a.Line < b.Line
	}
	return a.Character < b.Character
}
