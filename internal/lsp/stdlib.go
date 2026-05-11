package lsp

// stdlibModules is the snapshot of attached standard-library
// module names as of v0.52. Used for completion only — actual
// resolution still goes through the runner / pkg module loader.
// Update this list when adding or removing stdlib modules.
var stdlibModules = []string{
	"array",
	"base64",
	"channel",
	"csv",
	"dict",
	"digest",
	"dir",
	"file",
	"hex",
	"json",
	"markdown",
	"math",
	"matrix",
	"os",
	"path",
	"process",
	"random",
	"runtime",
	"secure_random",
	"string",
	"sync",
	"task",
	"time",
	"unittest",
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
	"abstract", "final", "private", "static",
	"initialize", "self", "super",
	"import", "as",
	"match", "when",
	"true", "false", "nil",
	"spawn", "await", "scope",
}

// Keywords returns a copy of the keyword list.
func Keywords() []string {
	out := make([]string, len(keywords))
	copy(out, keywords)
	return out
}
