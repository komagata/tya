package lsp

// This file extends protocol.go with the wire types introduced by
// v0.53's LSP v2. Splitting them across two files keeps the v0.52
// surface intact and easy to diff.

// WorkspaceEdit is the shape returned by rename and code action
// responses.
type WorkspaceEdit struct {
	Changes         map[string][]TextEdit `json:"changes,omitempty"`
	DocumentChanges []any                 `json:"documentChanges,omitempty"`
}

type TextDocumentEdit struct {
	TextDocument OptionalVersionedTextDocumentIdentifier `json:"textDocument"`
	Edits        []TextEdit                              `json:"edits"`
}

type OptionalVersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version *int   `json:"version"`
}

type RenameFileOperation struct {
	Kind   string `json:"kind"`
	OldURI string `json:"oldUri"`
	NewURI string `json:"newUri"`
}

// RenameParams matches LSP textDocument/rename.
type RenameParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	NewName      string                 `json:"newName"`
}

type PrepareRenameParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type PrepareRenameResult struct {
	Range       Range  `json:"range"`
	Placeholder string `json:"placeholder"`
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
	SymbolKindFunction    = 12
	SymbolKindClass       = 5
	SymbolKindStruct      = 23
	SymbolKindModule      = 2
	SymbolKindInterface   = 11
	SymbolKindMethod      = 6
	SymbolKindField       = 8
	SymbolKindConstructor = 9
	SymbolKindConstant    = 14
	SymbolKindVariable    = 13
	SymbolKindProperty    = 7
	SymbolKindObject      = 19
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

type InlayHintParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
}

type InlayHint struct {
	Position Position `json:"position"`
	Label    string   `json:"label"`
	Kind     int      `json:"kind,omitempty"`
}

const InlayHintKindParameter = 2

type CallHierarchyPrepareParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type CallHierarchyItem struct {
	Name           string `json:"name"`
	Kind           int    `json:"kind"`
	URI            string `json:"uri"`
	Range          Range  `json:"range"`
	SelectionRange Range  `json:"selectionRange"`
}

type CallHierarchyIncomingCall struct {
	From       CallHierarchyItem `json:"from"`
	FromRanges []Range           `json:"fromRanges"`
}

type CallHierarchyOutgoingCall struct {
	To         CallHierarchyItem `json:"to"`
	FromRanges []Range           `json:"fromRanges"`
}

type SelectionRangeParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Positions    []Position             `json:"positions"`
}

type SelectionRange struct {
	Range  Range           `json:"range"`
	Parent *SelectionRange `json:"parent,omitempty"`
}

type CodeLensParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type CodeLensOptions struct {
	ResolveProvider bool `json:"resolveProvider"`
}

type Command struct {
	Title     string `json:"title"`
	Command   string `json:"command"`
	Arguments []any  `json:"arguments,omitempty"`
}

type CodeLens struct {
	Range   Range   `json:"range"`
	Command Command `json:"command,omitempty"`
}

type FoldingRangeParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type FoldingRange struct {
	StartLine      int    `json:"startLine"`
	StartCharacter int    `json:"startCharacter,omitempty"`
	EndLine        int    `json:"endLine"`
	EndCharacter   int    `json:"endCharacter,omitempty"`
	Kind           string `json:"kind,omitempty"`
}

type DocumentLinkParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type DocumentLinkOptions struct {
	ResolveProvider bool `json:"resolveProvider"`
}

type DocumentLink struct {
	Range  Range  `json:"range"`
	Target string `json:"target,omitempty"`
}
