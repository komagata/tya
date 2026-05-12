package lsp

import (
	"sort"
	"strings"
)

// WorkspaceSymbolsFor returns every top-level symbol in the
// workspace whose name contains `query` (case-insensitive
// substring match). When query is empty all symbols are returned.
// v0.53 ranks by name ascending; future revisions may add fuzzy
// scoring.
func WorkspaceSymbolsFor(ws *Workspace, query string) []SymbolInformation {
	if ws == nil {
		return nil
	}
	q := strings.ToLower(query)
	out := []SymbolInformation{}
	for _, pf := range ws.AllFiles() {
		if pf.Prog == nil {
			continue
		}
		uri, _ := PathToURI(pf.Path)
		idx := BuildSymbols(pf.Prog)
		for _, sym := range idx.All() {
			if q != "" && !strings.Contains(strings.ToLower(sym.Name), q) {
				continue
			}
			out = append(out, SymbolInformation{
				Name: sym.Name,
				Kind: symbolKindFor(sym.Kind),
				Location: Location{
					URI:   uri,
					Range: rangeAt(sym.NameTok.Line, sym.NameTok.Col, len(sym.Name)),
				},
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func symbolKindFor(kind string) int {
	switch kind {
	case "function":
		return SymbolKindFunction
	case "class":
		return SymbolKindClass
	case "module":
		return SymbolKindModule
	case "interface":
		return SymbolKindInterface
	}
	return SymbolKindVariable
}
