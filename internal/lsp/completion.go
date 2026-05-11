package lsp

import (
	"tya/internal/checker"
	"tya/internal/lexer"
	"tya/internal/parser"
)

// Completion answers textDocument/completion. The MVP returns a
// flat candidate list: top-level bindings from the same file plus
// every stdlib module name, builtin function name, and keyword.
// The client filters the list by prefix.
func Completion(doc *Document) (CompletionList, error) {
	out := CompletionList{IsIncomplete: false, Items: []CompletionItem{}}
	if doc != nil {
		toks, lcomments, _ := lexer.LexWithComments(doc.Text)
		if prog, err := parser.ParseWithComments(toks, toCommentInfos(lcomments)); err == nil && prog != nil {
			idx := BuildSymbols(prog)
			for _, s := range idx.All() {
				out.Items = append(out.Items, CompletionItem{
					Label:  s.Name,
					Kind:   completionKindOf(s.Kind),
					Detail: s.Signature,
				})
			}
		}
	}
	for _, m := range StdlibModules() {
		out.Items = append(out.Items, CompletionItem{
			Label:  m,
			Kind:   CompletionKindModule,
			Detail: "stdlib module",
		})
	}
	for _, b := range checker.BuiltinNames() {
		out.Items = append(out.Items, CompletionItem{
			Label:  b,
			Kind:   CompletionKindFunction,
			Detail: "built-in",
		})
	}
	for _, k := range Keywords() {
		out.Items = append(out.Items, CompletionItem{
			Label: k,
			Kind:  CompletionKindKeyword,
		})
	}
	return out, nil
}

func completionKindOf(kind string) int {
	switch kind {
	case "function":
		return CompletionKindFunction
	case "class":
		return CompletionKindClass
	case "module":
		return CompletionKindModule
	case "interface":
		return CompletionKindClass
	}
	return CompletionKindVariable
}

func isStdlibModule(name string) bool {
	for _, m := range stdlibModules {
		if m == name {
			return true
		}
	}
	return false
}

func isBuiltinName(name string) bool {
	for _, b := range checker.BuiltinNames() {
		if b == name {
			return true
		}
	}
	return false
}
