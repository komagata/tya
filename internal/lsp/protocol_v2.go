package lsp

// This file extends protocol.go with the wire types introduced by
// v0.53's LSP v2. Splitting them across two files keeps the v0.52
// surface intact and easy to diff.

// WorkspaceEdit is the shape returned by rename and code action
// responses. tya v0.53 only emits the Changes map; the optional
// DocumentChanges (versioned) field is queued for v0.54+.
type WorkspaceEdit struct {
	Changes map[string][]TextEdit `json:"changes"`
}

// RenameParams matches LSP textDocument/rename.
type RenameParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	NewName      string                 `json:"newName"`
}

// ReferenceContext is the LSP textDocument/references options.
type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

// ReferenceParams matches LSP textDocument/references.
type ReferenceParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	Context      ReferenceContext       `json:"context"`
}

// DocumentRangeFormattingParams matches LSP textDocument/rangeFormatting.
type DocumentRangeFormattingParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
}

// CodeActionKind values used by tya.
const (
	CodeActionKindQuickFix = "quickfix"
)

// CodeActionContext is the third leg of textDocument/codeAction.
type CodeActionContext struct {
	Diagnostics []Diagnostic `json:"diagnostics"`
	Only        []string     `json:"only,omitempty"`
}

// CodeActionParams matches LSP textDocument/codeAction.
type CodeActionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
	Context      CodeActionContext      `json:"context"`
}

// CodeAction is a single proposed fix returned by codeAction.
type CodeAction struct {
	Title       string        `json:"title"`
	Kind        string        `json:"kind,omitempty"`
	Diagnostics []Diagnostic  `json:"diagnostics,omitempty"`
	Edit        WorkspaceEdit `json:"edit"`
	IsPreferred bool          `json:"isPreferred,omitempty"`
}

// CodeActionOptions advertises which kinds the server can produce.
type CodeActionOptions struct {
	CodeActionKinds []string `json:"codeActionKinds,omitempty"`
}

// SemanticTokensParams matches LSP textDocument/semanticTokens/full.
type SemanticTokensParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// SemanticTokens is the wire encoding of v0.53's semantic-token data.
type SemanticTokens struct {
	Data []uint32 `json:"data"`
}

// SemanticTokensLegend tells the client how to interpret the
// numeric token type / modifier values inside Data.
type SemanticTokensLegend struct {
	TokenTypes     []string `json:"tokenTypes"`
	TokenModifiers []string `json:"tokenModifiers"`
}

// SemanticTokensOptions advertises full-document semantic tokens.
type SemanticTokensOptions struct {
	Legend SemanticTokensLegend `json:"legend"`
	Range  bool                 `json:"range"`
	Full   bool                 `json:"full"`
}

// DocumentSymbolKind values (subset).
const (
	SymbolKindFunction  = 12
	SymbolKindClass     = 5
	SymbolKindModule    = 2
	SymbolKindInterface = 11
	SymbolKindMethod    = 6
	SymbolKindVariable  = 13
	SymbolKindProperty  = 7
)

// DocumentSymbolParams matches LSP textDocument/documentSymbol.
type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DocumentSymbol is the hierarchical-symbol shape.
type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           int              `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

// SymbolInformation is the flat-symbol shape used by workspace symbols.
type SymbolInformation struct {
	Name          string   `json:"name"`
	Kind          int      `json:"kind"`
	Location      Location `json:"location"`
	ContainerName string   `json:"containerName,omitempty"`
}

// WorkspaceSymbolParams matches LSP workspace/symbol.
type WorkspaceSymbolParams struct {
	Query string `json:"query"`
}
