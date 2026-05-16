package lsp

// stdlibModules is the snapshot of attached standard-library
// module names. Used for completion only — actual
// resolution still goes through the runner / pkg module loader.
// Update this list when adding or removing stdlib modules.
var stdlibModules = []string{
	"array",
	"base64",
	"binary",
	"channel",
	"cli",
	"collections",
	"color",
	"comparable",
	"compiler/ast",
	"compiler/checker",
	"compiler/format",
	"compiler/lexer",
	"compiler/parser",
	"compress",
	"csv",
	"dict",
	"digest",
	"dir",
	"drop_sequence",
	"equatable",
	"file",
	"filter_sequence",
	"geometry",
	"hex",
	"image",
	"io",
	"iterable",
	"iterable_sequence",
	"iterator",
	"json",
	"log",
	"map_sequence",
	"markdown",
	"math",
	"matrix",
	"net/http",
	"net/ip",
	"net/socket",
	"os",
	"path",
	"process",
	"random",
	"runtime",
	"secure_random",
	"serialization",
	"sequence",
	"stringable",
	"string",
	"sync",
	"task",
	"template",
	"time",
	"toml",
	"transform2d",
	"unittest",
	"url",
	"value",
	"xml",
}

// StdlibModules returns a copy of the registered module names.
func StdlibModules() []string {
	out := make([]string, len(stdlibModules))
	copy(out, stdlibModules)
	return out
}

// keywords lists tya control-flow / declaration keywords surfaced
// via completion. Order matches docs/SPEC.md §Lexical for
// determinism.
var keywords = []string{
	"if", "elseif", "else",
	"while", "for", "in", "break", "continue",
	"return", "raise", "try", "catch",
	"class", "module", "interface", "implements", "extends",
	"abstract", "final", "override", "private", "static",
	"initialize", "self", "Self", "super",
	"import", "as", "embed",
	"match", "case", "when",
	"true", "false", "nil",
	"spawn", "await", "scope", "select", "receive", "send", "timeout", "default",
}

// Keywords returns a copy of the keyword list.
func Keywords() []string {
	out := make([]string, len(keywords))
	copy(out, keywords)
	return out
}
