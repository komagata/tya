package checker

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"tya/internal/ast"
	"tya/internal/diag"
	"tya/internal/interp"
	"tya/internal/lexer"
	"tya/internal/parser"
	"tya/internal/token"
)

var constNameRE = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
var valueNameRE = regexp.MustCompile(`^_?[a-z][a-z0-9]*(?:_[a-z0-9]+)*$|^_$`)
var classNameRE = regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)

func isPrivateName(name string) bool {
	return strings.HasPrefix(name, "_") && name != "_"
}

func isPredicateName(name string) bool {
	return strings.HasSuffix(name, "?")
}

func isBangName(name string) bool {
	return strings.HasSuffix(name, "!")
}

func predicateBaseName(name string) string {
	return strings.TrimSuffix(name, "?")
}

func callableBaseName(name string) string {
	return strings.TrimSuffix(strings.TrimSuffix(name, "?"), "!")
}

func validCallableName(name string) bool {
	if isPredicateName(name) || isBangName(name) {
		base := callableBaseName(name)
		return base != "" && valueNameRE.MatchString(base)
	}
	return valueNameRE.MatchString(name)
}

// permissiveLegacy controls whether the v0.47 clean-cut diagnostics
// for the legacy v0.45 class-member surface (`@`, `@@`, `_`-prefix,
// `init` / `_init`) are emitted. Set to true by the runner before
// checking files under `selfhost/v01/`, which keep the v0.43 surface
// per the v0.46 / v0.47 SPECs' Self-Host Constraint. Reset to false
// for user code.
//
// Process-level global — safe because each `go run ./cmd/tya …`
// process performs at most one check pass at a time, and testscript
// fixtures spawn subprocesses so concurrent in-process tests do not
// share this flag.
var permissiveLegacy bool

// SetPermissiveLegacy turns the legacy-permission flag on or off and
// returns a function that restores the previous value when called.
// Use with `defer`:
//
//	defer checker.SetPermissiveLegacy(isV01Path)()
func SetPermissiveLegacy(value bool) func() {
	prev := permissiveLegacy
	permissiveLegacy = value
	return func() { permissiveLegacy = prev }
}

func Check(prog *ast.Program) error {
	return CheckWithModules(prog, nil)
}

func CheckWithModules(prog *ast.Program, modules []string) error {
	if err := checkStructure(prog, modules); err != nil {
		return err
	}
	return CheckStrict(prog, modules)
}

// CheckAll runs checkStructure followed by the strict pass, collecting
// every strict diagnostic when collectAll is true. file is recorded as
// Diagnostic.Primary.File for strict diagnostics.
//
// The structural check is still fail-fast in v0.29; on structural error
// it returns (nil, error). On success it returns the strict diagnostics
// (possibly empty) and a nil error.
func CheckAll(prog *ast.Program, modules []string, file string, collectAll bool) ([]diag.Diagnostic, error) {
	if err := checkStructure(prog, modules); err != nil {
		return nil, err
	}
	return CheckStrictDiagnostics(prog, modules, file, collectAll), nil
}

// checkStructure runs the structural checker (parse-tree validity, scope
// rules, class hierarchy, predicate names, ...) without the v0.28 strict
// lint pass. It is the right entry point when validating individual
// imported modules during source loading; the strict pass runs once at
// the entry-program level.
func checkStructure(prog *ast.Program, modules []string) error {
	constants := map[string]bool{}
	scope := newScope(nil)
	for _, name := range builtinNames {
		scope.define(name, kindUnknown)
	}
	for _, name := range internalBuiltinNames {
		scope.define(name, kindUnknown)
	}
	for _, name := range extraBuiltinNames {
		scope.define(name, kindUnknown)
	}
	for _, name := range primitiveClassNames {
		scope.define(name, kindClass)
		constants[name] = true
	}
	for _, name := range modules {
		scope.define(name, kindModule)
	}
	return checkStmts(prog.Stmts, constants, scope)
}

// SnakeCaseName maps a PascalCase class or interface name to the
// canonical snake_case file stem used by class/interface files.
func SnakeCaseName(name string) string {
	var out strings.Builder
	runes := []rune(name)
	for i, r := range runes {
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := runes[i-1]
			nextLower := i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z'
			prevLowerOrDigit := (prev >= 'a' && prev <= 'z') || (prev >= '0' && prev <= '9')
			prevUpper := prev >= 'A' && prev <= 'Z'
			if prevLowerOrDigit || (prevUpper && nextLower) {
				out.WriteByte('_')
			}
		}
		if r >= 'A' && r <= 'Z' {
			r = r - 'A' + 'a'
		}
		out.WriteRune(r)
	}
	return out.String()
}

// IsClassFileName reports whether the given path has the canonical
// snake_case .tya filename shape used by class/interface files. The
// file contents determine whether it is actually a class/interface
// file, because script files also use snake_case names.
func IsClassFileName(path string) bool {
	base := strings.TrimSuffix(filepath.Base(path), ".tya")
	return valueNameRE.MatchString(base) && !strings.HasPrefix(base, "_")
}

// IsScriptFileName reports whether the given path identifies a v0.44
// script file based on its leaf name: the leaf without ".tya" is a
// snake_case identifier with no leading underscore.
func IsScriptFileName(path string) bool {
	base := strings.TrimSuffix(filepath.Base(path), ".tya")
	return valueNameRE.MatchString(base) && !strings.HasPrefix(base, "_")
}

// CheckClassFileStructure validates the file-level shape of a class
// file: snake_case filename, exactly one public class or interface
// whose PascalCase name maps to the filename, additional declarations
// allowed as private siblings, imports preceding class/interface
// declarations, and no other top-level statements. It does NOT
// recurse into class bodies, so cross-class references inside method
// bodies are deferred to the full checker pass on the merged
// program. Use CheckClassFile when both file structure and body
// validity are needed in isolation (single-file inputs).
func CheckClassFileStructure(prog *ast.Program, path string) error {
	base := filepath.Base(path)
	want := strings.TrimSuffix(base, ".tya")
	if !valueNameRE.MatchString(want) || strings.HasPrefix(want, "_") {
		return fmt.Errorf("[TYA-E0404] class file %s must have a snake_case name", base)
	}
	publicSeen := false
	inDeclarations := false
	for _, stmt := range prog.Stmts {
		switch n := stmt.(type) {
		case *ast.ImportStmt, *ast.ImportBlockStmt:
			if inDeclarations {
				return fmt.Errorf("[TYA-E0403] class file %s imports must precede class and interface declarations", base)
			}
		case *ast.InterfaceDecl:
			inDeclarations = true
			if SnakeCaseName(n.Name) == want {
				if publicSeen {
					return fmt.Errorf("[TYA-E0405] class file %s declares public type %s more than once", base, want)
				}
				publicSeen = true
			}
		case *ast.ClassDecl:
			inDeclarations = true
			if SnakeCaseName(n.Name) == want {
				if publicSeen {
					return fmt.Errorf("[TYA-E0405] class file %s declares public type %s more than once", base, want)
				}
				publicSeen = true
			}
		default:
			return fmt.Errorf("[TYA-E0402] class file %s may only contain import, class, and interface declarations", base)
		}
	}
	if !publicSeen {
		return fmt.Errorf("[TYA-E0400] class file %s must define a class or interface that maps to %s", base, want)
	}
	return nil
}

// CheckClassFile validates a class/interface file at the given path.
// The file must contain exactly one public class or interface whose
// PascalCase name maps to the snake_case filename without ".tya".
// Additional declarations are private to the file (visibility is
// enforced separately). Top-level statements other than import, class,
// and interface declarations are rejected. Imports must precede any
// class or interface declaration.
//
// CheckClassFile runs both file-level structure validation and a
// body-level scope check on the file in isolation. For files that are
// part of a v0.44 package (where bodies may reference sibling classes
// from other files), call CheckClassFileStructure instead and rely on
// the merged-source checker to validate bodies after package
// synthesis.
func CheckClassFile(prog *ast.Program, path string) error {
	if err := CheckClassFileStructure(prog, path); err != nil {
		return err
	}
	return checkStructure(prog, nil)
}

func CheckModuleFile(prog *ast.Program, path string) error {
	want := strings.TrimSuffix(filepath.Base(path), ".tya")
	if !valueNameRE.MatchString(want) || strings.HasPrefix(want, "_") {
		return fmt.Errorf("invalid module file name %s", filepath.Base(path))
	}
	modules := []string{}
	seenModule := false
	for _, stmt := range prog.Stmts {
		switch n := stmt.(type) {
		case *ast.ImportStmt, *ast.ImportBlockStmt:
			if seenModule {
				return fmt.Errorf("%s may only contain imports before its module declaration", filepath.Base(path))
			}
		case *ast.ModuleDecl:
			seenModule = true
			modules = append(modules, n.Name)
			if n.Name != want {
				return fmt.Errorf("%s must define module %s", filepath.Base(path), want)
			}
		default:
			return fmt.Errorf("%s may only contain imports and one module declaration", filepath.Base(path))
		}
	}
	if len(modules) != 1 {
		return fmt.Errorf("%s must define exactly one module", filepath.Base(path))
	}
	return checkStructure(prog, nil)
}

var builtinNames = []string{
	"args", "env", "error", "exit", "print", "println",
}

var internalBuiltinNames = []string{
	"assert", "assert_equal", "chdir", "chr",
	"cwd", "delete", "dir_list", "dir_mkdir", "dir_rmdir", "dir_mkdir_all", "dir_remove_all", "dir_walk", "dir_temp_dir",
	"equal", "file_exists", "file_remove",
	"file_rename", "file_stat", "file_copy", "file_chmod", "file_temp",
	"ord", "panic",
	"path_expand_user",
	"read_file", "read_line",
	"write_file", "stderr_write", "file_append",
	"compress_gzip", "compress_gunzip", "compress_zlib", "compress_unzlib",
	"io_stdin", "io_stdout", "io_stderr", "io_open", "io_stream_read",
	"io_stream_read_line", "io_stream_eof", "io_stream_write", "io_stream_flush",
	"io_stream_close",
	"socket_connect", "tls_connect", "socket_server_listen", "socket_server_accept",
	"socket_read", "socket_read_line", "socket_write", "socket_close",
	"socket_closed", "socket_local_address", "socket_remote_address",
	"socket_server_close", "socket_server_local_address",
	// v0.25
	"bytes", "bytes_of", "bytes_text", "bytes_array", "bytes_concat", "bytes_slice",
	"file_read_bytes", "file_write_bytes",
	"binary_read_f32", "binary_read_f64", "binary_write_f32", "binary_write_f64",
	// v0.58
	"http_server_run", "http_server_run_tls",
	// v0.24
	"time_now", "time_monotonic", "time_unix", "time_duration", "time_sleep", "time_format", "time_parse", "time_since",
	"random_seed", "random_int", "random_float",
	"serialization_kind", "serialization_id", "serialization_public_fields", "serialization_has_member",
	"compiler_lexer_lex", "compiler_lexer_lex_with_comments",
	"compiler_parser_parse", "compiler_parser_parse_tokens",
	"compiler_ast_children", "compiler_ast_kind", "compiler_ast_span",
	"compiler_checker_check", "compiler_checker_check_ast",
	"compiler_format_format", "compiler_format_unparse",
	"math_sqrt", "math_pow", "math_floor", "math_ceil", "math_round",
	"math_trunc", "math_log", "math_log2", "math_log10", "math_exp",
	"math_sin", "math_cos", "math_tan", "math_asin", "math_acos", "math_atan",
	"math_atan2",
	"process_run", "process_exec", "environ", "setenv", "unsetenv",
	"digest_md5", "digest_sha1", "digest_sha256", "digest_sha384", "digest_sha512",
	"regex_compile",
	"secure_random_bytes", "secure_random_int",
	// v0.41 GC
	"runtime_gc_stats",
	"runtime_gc_collect",
	// v0.42 channel
	"channel_new", "channel_send", "channel_receive", "channel_receive_timeout", "channel_close", "channel_closed_p",
	// v0.43 channel.select
	"channel_select",
	// v0.43 task cooperative cancel
	"task_cancel", "task_is_cancelled_p", "task_current",
	// v0.42 sync
	"sync_mutex_new", "sync_lock", "sync_unlock",
	"sync_atomic_integer_new", "sync_atomic_integer_add", "sync_atomic_integer_load",
	"sync_atomic_integer_store", "sync_atomic_integer_cas",
	"sync_wait_group_new", "sync_wait_group_add", "sync_wait_group_done", "sync_wait_group_wait",
}

var extraBuiltinNames []string

func SetExtraBuiltinNames(names []string) func() {
	prev := append([]string(nil), extraBuiltinNames...)
	extraBuiltinNames = append([]string(nil), names...)
	return func() { extraBuiltinNames = prev }
}

var primitiveClassNames = []string{"Number", "String", "Array", "Dict", "Boolean", "Nil"}

func isPrimitiveClassName(name string) bool {
	for _, className := range primitiveClassNames {
		if name == className {
			return true
		}
	}
	return false
}

var removedTopLevelPrimitiveBuiltins = map[string]string{
	"chdir":                            "Process().chdir(path)",
	"chr":                              "String.chr(number)",
	"cwd":                              "Process().cwd()",
	"delete":                           "dict.delete(key)",
	"dir_list":                         "Dir().list(path)",
	"dir_mkdir":                        "Dir().mkdir(path)",
	"dir_rmdir":                        "Dir().rmdir(path)",
	"equal":                            "left == right or left.equal?(right)",
	"file_append":                      "File().append(path, text)",
	"file_exists":                      "File().exists?(path)",
	"file_remove":                      "File().remove(path)",
	"file_rename":                      "File().rename(old_path, new_path)",
	"file_stat":                        "File().stat(path)",
	"ord":                              "String.ord(string)",
	"path_expand_user":                 "Path().expand_user(path)",
	"read_file":                        "File().read(path)",
	"read_line":                        "Io().stdin().read_line()",
	"write_file":                       "File().write(path, text)",
	"stderr_write":                     "Io().stderr().write(value)",
	"compress_gzip":                    "compress.Gzip().compress(value)",
	"compress_gunzip":                  "compress.Gzip().decompress(value)",
	"compress_zlib":                    "compress.Zlib().compress(value)",
	"compress_unzlib":                  "compress.Zlib().decompress(value)",
	"io_stdin":                         "Io().stdin()",
	"io_stdout":                        "Io().stdout()",
	"io_stderr":                        "Io().stderr()",
	"io_open":                          "Io().open(path, mode)",
	"io_stream_read":                   "Reader.read(size)",
	"io_stream_read_line":              "Reader.read_line()",
	"io_stream_eof":                    "Reader.eof?()",
	"io_stream_write":                  "Writer.write(value)",
	"io_stream_flush":                  "Writer.flush()",
	"io_stream_close":                  "Reader.close() or Writer.close()",
	"socket_connect":                   "Socket.connect(host, port, options)",
	"tls_connect":                      "http.Client(https_url).get()",
	"socket_server_listen":             "Server.listen(host, port, options)",
	"socket_server_accept":             "server.accept()",
	"socket_read":                      "socket.read(size)",
	"socket_read_line":                 "socket.read_line()",
	"socket_write":                     "socket.write(value)",
	"socket_close":                     "socket.close()",
	"socket_closed":                    "socket.closed?()",
	"socket_local_address":             "socket.local_address()",
	"socket_remote_address":            "socket.remote_address()",
	"socket_server_close":              "server.close()",
	"socket_server_local_address":      "server.local_address()",
	"bytes":                            "Bytes.from_array(array)",
	"bytes_of":                         "Bytes.from_string(string)",
	"bytes_text":                       "bytes.to_string()",
	"bytes_array":                      "bytes.to_array()",
	"bytes_concat":                     "left.concat(right)",
	"bytes_slice":                      "bytes.slice(start, end)",
	"file_read_bytes":                  "File().read_bytes(path)",
	"file_write_bytes":                 "File().write_bytes(path, bytes)",
	"time_now":                         "Time().now()",
	"time_sleep":                       "Time().sleep(seconds)",
	"time_format":                      "Time().format(...)",
	"time_parse":                       "Time().parse(...)",
	"time_since":                       "Time().since(...)",
	"random_seed":                      "Random().seed(seed)",
	"random_int":                       "Random().int(min, max)",
	"random_float":                     "Random().float()",
	"compiler_lexer_lex":               "Lexer().lex(source)",
	"compiler_lexer_lex_with_comments": "Lexer().lex_with_comments(source)",
	"compiler_parser_parse":            "Parser().parse(source)",
	"compiler_parser_parse_tokens":     "Parser().parse_tokens(tokens)",
	"compiler_ast_children":            "Ast().children(node)",
	"compiler_ast_kind":                "Ast().kind(node)",
	"compiler_ast_span":                "Ast().span(node)",
	"compiler_checker_check":           "Checker().check(source)",
	"compiler_checker_check_ast":       "Checker().check_ast(program)",
	"compiler_format_format":           "Format().format(source)",
	"compiler_format_unparse":          "Format().unparse(program)",
	"digest_md5":                       "Digest().md5(value)",
	"digest_sha1":                      "Digest().sha1(value)",
	"digest_sha256":                    "Digest().sha256(value)",
	"digest_sha384":                    "Digest().sha384(value)",
	"digest_sha512":                    "Digest().sha512(value)",
	"secure_random_bytes":              "SecureRandom().bytes(size)",
	"secure_random_int":                "SecureRandom().int(min, max)",
	"kind":                             "x.class or x.class.name",
	"len":                              "value.len()",
	"byte_len":                         "text.byte_len()",
	"char_len":                         "text.len()",
	"trim":                             "text.trim()",
	"contains":                         "text.contains(part)",
	"starts_with":                      "text.starts_with(prefix)",
	"ends_with":                        "text.ends_with(suffix)",
	"replace":                          "text.replace(old, new)",
	"split":                            "text.split(separator)",
	"join":                             "items.join(separator)",
	"keys":                             "dict.keys()",
	"values":                           "dict.values()",
	"has":                              "dict.has(key)",
	"push":                             "items.push(value)",
	"pop":                              "items.pop()",
	"map":                              "items.map(fn)",
	"filter":                           "items.filter(fn)",
	"find":                             "items.find(fn)",
	"any":                              "items.any(fn)",
	"all":                              "items.all(fn)",
	"reduce":                           "items.reduce(initial, fn)",
	"to_string":                        "value.to_s()",
	"to_int":                           "value.to_i()",
	"to_float":                         "value.to_f()",
	"to_number":                        "value.to_number()",
}

func removedTopLevelPrimitiveBuiltinError(name string, line, col int) error {
	replacement, ok := removedTopLevelPrimitiveBuiltins[name]
	if !ok {
		return nil
	}
	if name == "kind" {
		return fmt.Errorf("%d:%d: [TYA-E0810] kind builtin removed in v0.59; use %s", line, col, replacement)
	}
	return fmt.Errorf("%d:%d: [TYA-E0812] top-level builtin %s was removed in v0.59; use %s", line, col, name, replacement)
}

func removedPrimitiveModuleCallError(member *ast.MemberExpr, scope *scope) error {
	id, ok := member.Target.(*ast.Ident)
	if !ok {
		return nil
	}
	if scope.defined(id.Name) {
		return nil
	}
	replacement := ""
	switch id.Name {
	case "string":
		switch member.Name {
		case "len", "char_len":
			replacement = "text.len()"
		case "byte_len":
			replacement = "text.byte_len()"
		case "trim":
			replacement = "text.trim()"
		case "contains":
			replacement = "text.contains(part)"
		case "starts_with":
			replacement = "text.starts_with(prefix)"
		case "ends_with":
			replacement = "text.ends_with(suffix)"
		case "replace":
			replacement = "text.replace(old, new)"
		case "split":
			replacement = "text.split(separator)"
		case "join":
			replacement = "items.join(separator)"
		case "lines":
			replacement = "text.lines()"
		case "upcase":
			replacement = "text.upper()"
		case "downcase":
			replacement = "text.lower()"
		}
	case "array":
		switch member.Name {
		case "len":
			replacement = "items.len()"
		case "empty?":
			replacement = "items.empty?()"
		case "first":
			replacement = "items.first()"
		case "last":
			replacement = "items.last()"
		case "push":
			replacement = "items.push(value)"
		case "pop":
			replacement = "items.pop()"
		case "slice":
			replacement = "items.slice(start, end)"
		case "reverse":
			replacement = "items.reverse()"
		case "join":
			replacement = "items.join(separator)"
		case "map":
			replacement = "items.map(fn)"
		case "filter":
			replacement = "items.filter(fn)"
		case "find":
			replacement = "items.find(fn)"
		case "any":
			replacement = "items.any(fn)"
		case "all":
			replacement = "items.all(fn)"
		case "each":
			replacement = "items.each(fn)"
		case "reduce":
			replacement = "items.reduce(initial, fn)"
		}
	case "dict":
		switch member.Name {
		case "len":
			replacement = "dict.len()"
		case "has", "has?":
			replacement = "dict.has(key)"
		case "get":
			replacement = "dict.get(key)"
		case "set":
			replacement = "dict.set(key, value)"
		case "delete":
			replacement = "dict.delete(key)"
		case "keys":
			replacement = "dict.keys()"
		case "values":
			replacement = "dict.values()"
		case "entries":
			replacement = "dict.entries()"
		case "merge":
			replacement = "dict.merge(other)"
		case "merge!":
			replacement = "dict.merge!(other)"
		}
	case "value":
		if member.Name == "nil?" {
			replacement = "value == nil"
		}
	}
	if replacement == "" {
		return nil
	}
	return fmt.Errorf("%d:%d: [TYA-E0812] primitive helper %s.%s was removed in v0.59; use %s", member.NameTok.Line, member.NameTok.Col, id.Name, member.Name, replacement)
}

// BuiltinNames returns a copy of the registered builtin function
// names. Tooling that needs to surface builtins (LSP completion,
// for instance) can use this without touching the internal slice.
func BuiltinNames() []string {
	out := make([]string, 0, len(builtinNames)+len(extraBuiltinNames))
	out = append(out, builtinNames...)
	out = append(out, extraBuiltinNames...)
	return out
}

type scope struct {
	parent           *scope
	names            map[string]bool
	kinds            map[string]valueKind
	funcParams       map[string][]string
	classes          map[string]classInfo
	interfaces       map[string]interfaceInfo
	inInstanceMethod bool
	inClassMethod    bool
	inClassBody      bool
	interfaceFields  map[string]bool
	currentMethod    string
	currentClass     string
}

type classInfo struct {
	name                  string
	parent                string
	abstract              bool
	final                 bool
	hasInit               bool
	initRequired          int
	initArity             int
	initParams            []string
	privateInit           bool
	methods               map[string]int
	methodParams          map[string][]string
	classMethods          map[string]int
	classMethodParams     map[string][]string
	abstractMethods       map[string]int
	abstractClassMethods  map[string]int
	interfaceMethods      map[string]int
	interfaceInitializers bool
	fields                map[string]bool
	privateFields         map[string]bool
	privateMethods        map[string]bool
	classConstants        map[string]bool
	privateClassMembers   map[string]bool
	privateClassMethods   map[string]bool
	privateFieldAssigned  map[string]bool
	// v0.45 cross-file private enforcement metadata. originFile is
	// the basename of the source file (e.g. "util.tya") that declared
	// this class; populated from ClassDecl.OriginFile by
	// predeclareModuleClass. A class is public iff its name plus
	// ".tya" equals originFile (the v0.44 filename-matches-classname
	// rule); otherwise it is private to originFile. An empty
	// originFile means the class is not part of a synthesized
	// directory package (single-file script, legacy `module` decl),
	// and no cross-file check applies.
	originFile string
}

type arityRange struct {
	required int
	max      int
}

type interfaceInfo struct {
	name              string
	parents           []string
	methods           map[string]int
	defaults          map[string]int
	defaultsCallSuper map[string]bool
	fields            map[string]bool
	initializer       bool
	tokLine           int
	tokCol            int
}

type interfaceRequirement struct {
	arity     int
	source    string
	sourceTok ast.ClassRef
}

func newScope(parent *scope) *scope {
	s := &scope{parent: parent, names: map[string]bool{}, kinds: map[string]valueKind{}, funcParams: map[string][]string{}}
	if parent != nil {
		s.classes = parent.classes
		s.interfaces = parent.interfaces
		s.funcParams = parent.funcParams
		s.inInstanceMethod = parent.inInstanceMethod
		s.inClassMethod = parent.inClassMethod
		s.inClassBody = parent.inClassBody
		s.interfaceFields = parent.interfaceFields
		s.currentMethod = parent.currentMethod
		s.currentClass = parent.currentClass
	} else {
		s.classes = map[string]classInfo{}
		s.interfaces = map[string]interfaceInfo{}
	}
	return s
}

type valueKind int

const (
	kindUnknown valueKind = iota
	kindArray
	kindDict
	kindModule
	kindClass
	kindInterface
	kindObject
	kindNil
	kindBool
	kindNumber
	kindString
	kindBytes
	kindFunction
)

func (s *scope) define(name string, kind valueKind) {
	s.names[name] = true
	if s.kinds[name] == kindModule {
		return
	}
	if kind == kindNil && s.kinds[name] != kindUnknown {
		return
	}
	s.kinds[name] = kind
}

func (s *scope) defined(name string) bool {
	if s.names[name] {
		return true
	}
	if s.parent != nil {
		return s.parent.defined(name)
	}
	return false
}

func (s *scope) kind(name string) valueKind {
	if kind, ok := s.kinds[name]; ok {
		return kind
	}
	if s.parent != nil {
		return s.parent.kind(name)
	}
	return kindUnknown
}

func checkStmts(stmts []ast.Stmt, constants map[string]bool, scope *scope) error {
	if err := predeclareFunctionBindings(stmts, scope); err != nil {
		return err
	}
	imports := map[string]token.Token{}
	for _, stmt := range stmts {
		for _, imp := range checkerImports(stmt) {
			if first, ok := imports[importCheckerKey(imp)]; ok {
				return fmt.Errorf("%d:%d: duplicate import %s first imported at %d:%d", imp.NameTok.Line, imp.NameTok.Col, importCheckerKey(imp), first.Line, first.Col)
			}
			imports[importCheckerKey(imp)] = imp.NameTok
			for _, segment := range strings.Split(imp.Name, "/") {
				if !valueNameRE.MatchString(segment) || strings.HasPrefix(segment, "_") {
					return fmt.Errorf("%d:%d: invalid module name %s", imp.NameTok.Line, imp.NameTok.Col, imp.Name)
				}
			}
			binding := imp.BindingName()
			tok := imp.NameTok
			if imp.Alias != "" {
				tok = imp.AliasTok
			}
			if !valueNameRE.MatchString(binding) || strings.HasPrefix(binding, "_") {
				return fmt.Errorf("%d:%d: invalid import binding %s", tok.Line, tok.Col, binding)
			}
			scope.define(binding, kindModule)
		}
		switch n := stmt.(type) {
		case *ast.EmbedStmt:
			if !valueNameRE.MatchString(n.Name) {
				return fmt.Errorf("%d:%d: invalid embed binding %s", n.NameTok.Line, n.NameTok.Col, n.Name)
			}
			for key := range n.Transforms {
				if key != "gzip" && key != "hash" && key != "minify" {
					return fmt.Errorf("%d:%d: [TYA-E0612] unknown embed transform %s", n.PathTok.Line, n.PathTok.Col, key)
				}
			}
			scope.define(n.Name, kindUnknown)
		case *ast.AssignStmt:
			for _, value := range n.Values {
				if err := checkExpr(value, scope); err != nil {
					return err
				}
			}
			for _, target := range n.Targets {
				name, ok := target.(*ast.Ident)
				if !ok {
					if err := checkAssignmentTarget(target, n.Values, constants, scope); err != nil {
						return err
					}
					continue
				}
				if isPredicateName(name.Name) || isBangName(name.Name) {
					if len(n.Targets) != 1 || len(n.Values) != 1 {
						if isPredicateName(name.Name) {
							return fmt.Errorf("%d:%d: predicate binding %s must be a function", name.Tok.Line, name.Tok.Col, name.Name)
						}
						return fmt.Errorf("%d:%d: callable binding %s must be a function", name.Tok.Line, name.Tok.Col, name.Name)
					}
					if _, ok := n.Values[0].(*ast.FuncLit); !ok {
						if isPredicateName(name.Name) {
							return fmt.Errorf("%d:%d: predicate binding %s must be a function", name.Tok.Line, name.Tok.Col, name.Name)
						}
						return fmt.Errorf("%d:%d: callable binding %s must be a function", name.Tok.Line, name.Tok.Col, name.Name)
					}
					if !validCallableName(name.Name) {
						if isPredicateName(name.Name) {
							return fmt.Errorf("%d:%d: invalid predicate name %s", name.Tok.Line, name.Tok.Col, name.Name)
						}
						return fmt.Errorf("%d:%d: invalid callable name %s", name.Tok.Line, name.Tok.Col, name.Name)
					}
				} else if err := checkBindingName(name.Name, name.Tok.Line, name.Tok.Col); err != nil {
					return err
				}
				if constants[name.Name] {
					return fmt.Errorf("%d:%d: cannot reassign constant %s", n.Tok.Line, n.Tok.Col, name.Name)
				}
				nextKind := exprKind(n.Values, scope)
				if prevKind := scope.kind(name.Name); prevKind != kindUnknown && nextKind != kindUnknown && prevKind != kindModule && prevKind != kindNil && nextKind != kindNil && prevKind != nextKind {
					return fmt.Errorf("%d:%d: cannot reassign %s from %s to %s", n.Tok.Line, n.Tok.Col, name.Name, valueKindName(prevKind), valueKindName(nextKind))
				}
				if constNameRE.MatchString(name.Name) {
					constants[name.Name] = true
				}
				scope.define(name.Name, nextKind)
			}
		case *ast.ModuleDecl:
			if !valueNameRE.MatchString(n.Name) || strings.HasPrefix(n.Name, "_") {
				return fmt.Errorf("%d:%d: invalid module name %s", n.NameTok.Line, n.NameTok.Col, n.Name)
			}
			seen := map[string]bool{}
			scope.define(n.Name, kindModule)
			for _, iface := range n.Interfaces {
				if !classNameRE.MatchString(iface.Name) {
					return fmt.Errorf("%d:%d: invalid interface name %s", iface.NameTok.Line, iface.NameTok.Col, iface.Name)
				}
				if seen[iface.Name] {
					return fmt.Errorf("%d:%d: duplicate module member %s", iface.NameTok.Line, iface.NameTok.Col, iface.Name)
				}
				seen[iface.Name] = true
				scope.define(n.Name+"."+iface.Name, kindInterface)
				if err := checkInterface(iface, scope, n.Name); err != nil {
					return err
				}
			}
			for _, class := range n.Classes {
				if !classNameRE.MatchString(class.Name) {
					return fmt.Errorf("%d:%d: invalid class name %s", class.NameTok.Line, class.NameTok.Col, class.Name)
				}
				if seen[class.Name] {
					return fmt.Errorf("%d:%d: duplicate module member %s", class.NameTok.Line, class.NameTok.Col, class.Name)
				}
				seen[class.Name] = true
				scope.define(n.Name+"."+class.Name, kindClass)
				if err := checkClass(class, scope, n.Name); err != nil {
					return err
				}
			}
			for _, member := range n.Members {
				if isPredicateName(member.Name) || isBangName(member.Name) {
					if _, ok := member.Value.(*ast.FuncLit); !ok {
						if isPredicateName(member.Name) {
							return fmt.Errorf("%d:%d: predicate module member %s must be a function", member.Tok.Line, member.Tok.Col, member.Name)
						}
						return fmt.Errorf("%d:%d: callable module member %s must be a function", member.Tok.Line, member.Tok.Col, member.Name)
					}
					if !validCallableName(member.Name) {
						if isPredicateName(member.Name) {
							return fmt.Errorf("%d:%d: invalid predicate name %s", member.Tok.Line, member.Tok.Col, member.Name)
						}
						return fmt.Errorf("%d:%d: invalid callable name %s", member.Tok.Line, member.Tok.Col, member.Name)
					}
				} else if !valueNameRE.MatchString(member.Name) {
					return fmt.Errorf("%d:%d: invalid module member %s", member.Tok.Line, member.Tok.Col, member.Name)
				}
				if seen[member.Name] {
					return fmt.Errorf("%d:%d: duplicate module member %s", member.Tok.Line, member.Tok.Col, member.Name)
				}
				seen[member.Name] = true
				if err := checkExpr(member.Value, scope); err != nil {
					return err
				}
			}
		case *ast.ClassDecl:
			if err := checkClass(n, scope, ""); err != nil {
				return err
			}
		case *ast.InterfaceDecl:
			if err := checkInterface(n, scope, ""); err != nil {
				return err
			}
		case *ast.IfStmt:
			if err := checkExpr(n.Cond, scope); err != nil {
				return err
			}
			if err := checkStmts(n.Then, constants, newScope(scope)); err != nil {
				return err
			}
			if err := checkStmts(n.Else, constants, newScope(scope)); err != nil {
				return err
			}
		case *ast.WhileStmt:
			if err := checkExpr(n.Cond, scope); err != nil {
				return err
			}
			if err := checkStmts(n.Body, constants, newScope(scope)); err != nil {
				return err
			}
		case *ast.ForInStmt:
			if err := checkBindingName(n.ValueName, n.ValueTok.Line, n.ValueTok.Col); err != nil {
				return err
			}
			if n.IndexName != "" {
				if err := checkBindingName(n.IndexName, n.IndexTok.Line, n.IndexTok.Col); err != nil {
					return err
				}
			}
			if err := checkExpr(n.Iterable, scope); err != nil {
				return err
			}
			child := newScope(scope)
			child.define(n.ValueName, kindUnknown)
			if n.IndexName != "" {
				child.define(n.IndexName, kindUnknown)
			}
			if err := checkStmts(n.Body, constants, child); err != nil {
				return err
			}
		case *ast.ExprStmt:
			if err := checkExpr(n.Expr, scope); err != nil {
				return err
			}
		case *ast.ReturnStmt:
			for _, value := range n.Values {
				if err := checkExpr(value, scope); err != nil {
					return err
				}
			}
		case *ast.RaiseStmt:
			if err := checkExpr(n.Value, scope); err != nil {
				return err
			}
		case *ast.TryCatchStmt:
			if err := checkStmts(n.Try, constants, newScope(scope)); err != nil {
				return err
			}
			catchScope := newScope(scope)
			if n.CatchName != "_" && n.CatchName != "" {
				if err := checkBindingName(n.CatchName, n.CatchTok.Line, n.CatchTok.Col); err != nil {
					return err
				}
				catchScope.define(n.CatchName, kindUnknown)
			}
			if err := checkStmts(n.Catch, constants, catchScope); err != nil {
				return err
			}
			if err := checkStmts(n.Finally, constants, newScope(scope)); err != nil {
				return err
			}
		case *ast.MatchStmt:
			if err := checkExpr(n.Value, scope); err != nil {
				return err
			}
			for _, c := range n.Cases {
				caseScope := newScope(scope)
				if err := checkPattern(c.Pattern, caseScope); err != nil {
					return err
				}
				if err := checkStmts(c.Body, constants, caseScope); err != nil {
					return err
				}
			}
		case *ast.ScopeBlock:
			if err := checkStmts(n.Body, constants, newScope(scope)); err != nil {
				return err
			}
		case *ast.SelectStmt:
			for _, arm := range n.Arms {
				if arm.Channel != nil {
					if err := checkExpr(arm.Channel, scope); err != nil {
						return err
					}
				}
				if arm.Value != nil {
					if err := checkExpr(arm.Value, scope); err != nil {
						return err
					}
				}
				if arm.Seconds != nil {
					if err := checkExpr(arm.Seconds, scope); err != nil {
						return err
					}
				}
				child := newScope(scope)
				if arm.BindName != "" {
					if err := checkBindingName(arm.BindName, arm.BindTok.Line, arm.BindTok.Col); err != nil {
						return err
					}
					child.define(arm.BindName, kindUnknown)
				}
				if err := checkStmts(arm.Body, constants, child); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func checkPattern(pattern ast.Expr, scope *scope) error {
	switch n := pattern.(type) {
	case *ast.Ident:
		if n.Name == "_" {
			return nil
		}
		if err := checkBindingName(n.Name, n.Tok.Line, n.Tok.Col); err != nil {
			return err
		}
		scope.define(n.Name, kindUnknown)
	case *ast.IntLit, *ast.FloatLit, *ast.StringLit, *ast.BoolLit, *ast.NilLit:
		return nil
	case *ast.ArrayLit:
		for _, elem := range n.Elems {
			if err := checkPattern(elem, scope); err != nil {
				return err
			}
		}
	case *ast.DictLit:
		seen := map[string]bool{}
		for _, prop := range n.Props {
			if prop.Name == "" {
				return fmt.Errorf("%d:%d: pattern dictionary keys must be string literals", prop.Tok.Line, prop.Tok.Col)
			}
			if seen[prop.Name] {
				return fmt.Errorf("%d:%d: duplicate pattern dictionary key %s", prop.Tok.Line, prop.Tok.Col, prop.Name)
			}
			seen[prop.Name] = true
			if err := checkPattern(prop.Value, scope); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("invalid pattern syntax")
	}
	return nil
}

func predeclareFunctionBindings(stmts []ast.Stmt, scope *scope) error {
	for _, stmt := range stmts {
		switch n := stmt.(type) {
		case *ast.AssignStmt:
			if len(n.Targets) != 1 || len(n.Values) != 1 {
				continue
			}
			name, ok := n.Targets[0].(*ast.Ident)
			if !ok {
				continue
			}
			if _, ok := n.Values[0].(*ast.FuncLit); !ok {
				continue
			}
			fn := n.Values[0].(*ast.FuncLit)
			if isPredicateName(name.Name) || isBangName(name.Name) {
				if !validCallableName(name.Name) {
					return fmt.Errorf("%d:%d: invalid callable name %s", name.Tok.Line, name.Tok.Col, name.Name)
				}
			} else {
				if err := checkBindingName(name.Name, name.Tok.Line, name.Tok.Col); err != nil {
					return err
				}
			}
			scope.define(name.Name, kindUnknown)
			scope.funcParams[name.Name] = append([]string(nil), fn.Params...)
		case *ast.ClassDecl:
			if err := predeclareClass(n, scope); err != nil {
				return err
			}
		case *ast.InterfaceDecl:
			if err := predeclareInterface("", n, scope); err != nil {
				return err
			}
		case *ast.ModuleDecl:
			for _, iface := range n.Interfaces {
				if err := predeclareInterface(n.Name, iface, scope); err != nil {
					return err
				}
			}
			for _, class := range n.Classes {
				if err := predeclareModuleClass(n.Name, class, scope); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func predeclareInterface(module string, iface *ast.InterfaceDecl, scope *scope) error {
	if !classNameRE.MatchString(iface.Name) {
		return fmt.Errorf("%d:%d: invalid interface name %s", iface.NameTok.Line, iface.NameTok.Col, iface.Name)
	}
	key := iface.Name
	if module != "" {
		key = module + "." + iface.Name
	}
	if scope.kind(key) == kindInterface {
		return fmt.Errorf("%d:%d: duplicate interface %s", iface.NameTok.Line, iface.NameTok.Col, iface.Name)
	}
	info := interfaceInfo{name: key, methods: map[string]int{}, defaults: map[string]int{}, defaultsCallSuper: map[string]bool{}, fields: map[string]bool{}, tokLine: iface.NameTok.Line, tokCol: iface.NameTok.Col}
	for _, parent := range iface.Parents {
		info.parents = append(info.parents, refKey(&parent, module, scope))
	}
	for _, method := range iface.Methods {
		if method.Func == nil {
			info.methods[method.Name] = len(method.Params)
		} else if method.Name != "initialize" {
			info.defaults[method.Name] = len(method.Params)
			if funcCallsSuper(method.Func) {
				info.defaultsCallSuper[method.Name] = true
			}
		} else {
			info.initializer = true
		}
	}
	for _, field := range iface.Fields {
		info.fields[field.Name] = true
	}
	scope.define(key, kindInterface)
	scope.interfaces[key] = info
	return nil
}

func predeclareModuleClass(module string, class *ast.ClassDecl, scope *scope) error {
	if !classNameRE.MatchString(class.Name) {
		return fmt.Errorf("%d:%d: invalid class name %s", class.NameTok.Line, class.NameTok.Col, class.Name)
	}
	key := classKey(module, class)
	info := newClassInfo(key, class)
	info.originFile = class.OriginFile
	if class.Parent != nil {
		info.parent = refKey(class.Parent, module, scope)
	}
	for _, method := range class.Methods {
		if method.Abstract {
			if method.Class {
				info.abstractClassMethods[method.Name] = len(method.Func.Params)
			} else {
				info.abstractMethods[method.Name] = len(method.Func.Params)
			}
			continue
		}
		if method.Class {
			info.classMethods[method.Name] = len(method.Func.Params)
			info.classMethodParams[method.Name] = append([]string(nil), method.Func.Params...)
			if method.Private {
				info.privateClassMembers[method.Name] = true
				info.privateClassMethods[method.Name] = true
			}
			continue
		}
		info.methods[method.Name] = len(method.Func.Params)
		info.methodParams[method.Name] = append([]string(nil), method.Func.Params...)
		if method.Private {
			info.privateMethods[method.Name] = true
		}
		// v0.46 G1: `init` + Private flag is the private constructor.
		// Legacy `_init` (no keyword) also flows through Private from
		// the parser's transitional `_`-prefix recognition.
		// v0.46 G5: `initialize` is the canonical constructor name;
		// `init` and `_init` are recognized during the transition.
		if method.Name == "init" || method.Name == "_init" || method.Name == "initialize" {
			info.hasInit = true
			info.initRequired = requiredParamCount(method.Func)
			info.initArity = len(method.Func.Params)
			info.initParams = append([]string(nil), method.Func.Params...)
			if method.Private {
				info.privateInit = true
			}
		}
	}
	collectPrivateClassMembers(&info, class)
	scope.define(module+"."+class.Name, kindClass)
	scope.classes[key] = info
	return nil
}

func predeclareClass(class *ast.ClassDecl, scope *scope) error {
	if !classNameRE.MatchString(class.Name) {
		return fmt.Errorf("%d:%d: invalid class name %s", class.NameTok.Line, class.NameTok.Col, class.Name)
	}
	key := classKey("", class)
	info := newClassInfo(key, class)
	info.originFile = class.OriginFile
	if class.Parent != nil {
		info.parent = refKey(class.Parent, "", scope)
	}
	for _, method := range class.Methods {
		if method.Abstract {
			if method.Class {
				info.abstractClassMethods[method.Name] = len(method.Func.Params)
			} else {
				info.abstractMethods[method.Name] = len(method.Func.Params)
			}
			continue
		}
		if method.Class {
			info.classMethods[method.Name] = len(method.Func.Params)
			info.classMethodParams[method.Name] = append([]string(nil), method.Func.Params...)
			if method.Private {
				info.privateClassMembers[method.Name] = true
				info.privateClassMethods[method.Name] = true
			}
			continue
		}
		info.methods[method.Name] = len(method.Func.Params)
		info.methodParams[method.Name] = append([]string(nil), method.Func.Params...)
		if method.Private {
			info.privateMethods[method.Name] = true
		}
		// v0.46 G1: see predeclareModuleClass for the same logic.
		// v0.46 G5: `initialize` is the canonical constructor name;
		// `init` and `_init` are recognized during the transition.
		if method.Name == "init" || method.Name == "_init" || method.Name == "initialize" {
			info.hasInit = true
			info.initRequired = requiredParamCount(method.Func)
			info.initArity = len(method.Func.Params)
			info.initParams = append([]string(nil), method.Func.Params...)
			if method.Private {
				info.privateInit = true
			}
		}
	}
	collectPrivateClassMembers(&info, class)
	scope.define(class.Name, kindClass)
	scope.classes[key] = info
	return nil
}

func newClassInfo(key string, class *ast.ClassDecl) classInfo {
	return classInfo{
		name:                 key,
		abstract:             class.Abstract,
		final:                class.Final,
		methods:              map[string]int{},
		methodParams:         map[string][]string{},
		classMethods:         map[string]int{},
		classMethodParams:    map[string][]string{},
		abstractMethods:      map[string]int{},
		abstractClassMethods: map[string]int{},
		interfaceMethods:     map[string]int{},
		fields:               map[string]bool{},
		privateFields:        map[string]bool{},
		privateMethods:       map[string]bool{},
		classConstants:       map[string]bool{},
		privateClassMembers:  map[string]bool{},
		privateClassMethods:  map[string]bool{},
		privateFieldAssigned: map[string]bool{},
	}
}

func collectPrivateClassMembers(info *classInfo, class *ast.ClassDecl) {
	for _, field := range class.Fields {
		info.fields[field.Name] = true
		if field.Private {
			info.privateFields[field.Name] = true
		}
	}
	for _, variable := range class.Vars {
		if variable.Private {
			info.privateClassMembers[variable.Name] = true
		}
	}
	for _, constant := range class.Constants {
		info.classConstants[constant.Name] = true
		if constant.Private {
			info.privateClassMembers[constant.Name] = true
		}
	}
	for _, method := range class.Methods {
		collectPrivateAssignments(info, method.Func)
	}
}

func collectPrivateAssignments(info *classInfo, fn *ast.FuncLit) {
	var walkExpr func(ast.Expr, bool)
	walkExpr = func(expr ast.Expr, assignmentTarget bool) {
		switch n := expr.(type) {
		case *ast.InstanceFieldExpr:
			if assignmentTarget && isPrivateName(n.Name) {
				info.privateFieldAssigned[n.Name] = true
				info.privateFields[n.Name] = true
			}
		case *ast.ClassVarExpr:
			if assignmentTarget && isPrivateName(n.Name) {
				info.privateClassMembers[n.Name] = true
			}
		case *ast.BinaryExpr:
			walkExpr(n.Left, false)
			walkExpr(n.Right, false)
		case *ast.UnaryExpr:
			walkExpr(n.Expr, false)
		case *ast.TryExpr:
			walkExpr(n.Expr, false)
		case *ast.SpawnExpr:
			walkExpr(n.Callee, false)
		case *ast.AwaitExpr:
			walkExpr(n.Target, false)
		case *ast.ArrayLit:
			for _, elem := range n.Elems {
				walkExpr(elem, false)
			}
		case *ast.DictLit:
			for _, prop := range n.Props {
				walkExpr(prop.Value, false)
			}
		case *ast.IndexExpr:
			walkExpr(n.Target, false)
			walkExpr(n.Index, false)
		case *ast.MemberExpr:
			walkExpr(n.Target, false)
		case *ast.CallExpr:
			walkExpr(n.Callee, false)
			for _, arg := range n.Args {
				walkExpr(arg, false)
			}
		case *ast.FuncLit:
			collectPrivateAssignments(info, n)
		}
	}
	for _, stmt := range fn.Body {
		switch n := stmt.(type) {
		case *ast.AssignStmt:
			for _, target := range n.Targets {
				walkExpr(target, true)
			}
			for _, value := range n.Values {
				walkExpr(value, false)
			}
		case *ast.ExprStmt:
			walkExpr(n.Expr, false)
		case *ast.ReturnStmt:
			for _, value := range n.Values {
				walkExpr(value, false)
			}
		}
	}
	if fn.Expr != nil {
		walkExpr(fn.Expr, false)
	}
}

func checkerImports(stmt ast.Stmt) []*ast.ImportStmt {
	switch n := stmt.(type) {
	case *ast.ImportStmt:
		return []*ast.ImportStmt{n}
	case *ast.ImportBlockStmt:
		return n.Imports
	default:
		return nil
	}
}

func importCheckerKey(imp *ast.ImportStmt) string {
	key := imp.Name
	if imp.Wildcard {
		key += "/*"
	}
	return key
}

func checkExpr(expr ast.Expr, scope *scope) error {
	switch n := expr.(type) {
	case *ast.Ident:
		if isPrivateName(n.Name) && scope.inInstanceMethod && scope.currentClass != "" {
			if scope.classes[scope.currentClass].privateMethods[n.Name] {
				return nil
			}
		}
		if !scope.defined(n.Name) {
			if scope.currentClass != "" {
				if info, ok := scope.classes[scope.currentClass]; ok && info.classConstants[n.Name] {
					return nil
				}
			}
			// v0.44 within-package fallback: inside a module's class
			// method body, an unqualified PascalCase reference resolves
			// to <currentModule>.<Name> when that key names a sibling
			// class or interface in the same module. This makes
			// references between class files of the same package work
			// without an explicit module prefix.
			if mod := currentModulePrefix(scope); mod != "" && classNameRE.MatchString(n.Name) {
				key := mod + "." + n.Name
				if scope.kind(key) == kindClass {
					if info, ok := scope.classes[key]; ok {
						if err := checkCrossFilePrivate(info, n.Name, n.Tok.Line, n.Tok.Col, scope); err != nil {
							return err
						}
					}
					return nil
				}
				if scope.kind(key) == kindInterface {
					return nil
				}
			}
			return undefinedNameError(n.Name, n.Tok.Line, n.Tok.Col, scope)
		}
		if scope.kind(n.Name) == kindClass {
			if info, ok := scope.classes[n.Name]; ok {
				if err := checkCrossFilePrivate(info, n.Name, n.Tok.Line, n.Tok.Col, scope); err != nil {
					return err
				}
			}
		}
	case *ast.StringLit:
		if err := checkInterpolation(n.Value, scope); err != nil {
			return err
		}
	case *ast.DictLit:
		seen := map[string]bool{}
		for _, prop := range n.Props {
			if prop.Tok.Type != token.STRING && !valueNameRE.MatchString(prop.Name) {
				return fmt.Errorf("%d:%d: invalid dictionary key %s", prop.Tok.Line, prop.Tok.Col, prop.Name)
			}
			if seen[prop.Name] {
				return fmt.Errorf("%d:%d: duplicate dictionary key %s", prop.Tok.Line, prop.Tok.Col, prop.Name)
			}
			seen[prop.Name] = true
			if err := checkExpr(prop.Value, scope); err != nil {
				return err
			}
		}
	case *ast.FuncLit:
		seen := map[string]bool{}
		child := newScope(scope)
		child.currentClass = scope.currentClass
		child.currentMethod = scope.currentMethod
		child.inClassMethod = scope.inClassMethod
		child.inInstanceMethod = scope.inInstanceMethod
		seenDefault := false
		for i, param := range n.Params {
			line := 0
			col := 0
			if i < len(n.ParamToks) {
				line = n.ParamToks[i].Line
				col = n.ParamToks[i].Col
			}
			var def ast.Expr
			if i < len(n.Defaults) {
				def = n.Defaults[i]
			}
			if def == nil && seenDefault {
				return fmt.Errorf("%d:%d: required parameter %s after default parameter", line, col, param)
			}
			if err := checkBindingName(param, line, col); err != nil {
				return err
			}
			if seen[param] && param != "_" {
				if line > 0 {
					return fmt.Errorf("%d:%d: duplicate function parameter %s", line, col, param)
				}
				return fmt.Errorf("duplicate function parameter %s", param)
			}
			seen[param] = true
			if def != nil {
				seenDefault = true
				if err := checkExpr(def, child); err != nil {
					return err
				}
			}
			child.define(param, kindUnknown)
		}
		if n.Expr != nil {
			if err := checkExpr(n.Expr, child); err != nil {
				return err
			}
		}
		if err := checkStmts(n.Body, map[string]bool{}, child); err != nil {
			return err
		}
	case *ast.ArrayLit:
		for _, elem := range n.Elems {
			if err := checkExpr(elem, scope); err != nil {
				return err
			}
		}
	case *ast.BinaryExpr:
		if err := checkExpr(n.Left, scope); err != nil {
			return err
		}
		switch n.Op.Lexeme {
		case "%":
			if isFloatLiteral(n.Left) || isFloatLiteral(n.Right) {
				return fmt.Errorf("%d:%d: %% expects integers", n.Op.Line, n.Op.Col)
			}
		case "<", "<=", ">", ">=":
			if isStringLiteral(n.Left) || isStringLiteral(n.Right) {
				return fmt.Errorf("%d:%d: %s expects numbers", n.Op.Line, n.Op.Col, n.Op.Lexeme)
			}
		}
		if n.Op.Lexeme == "+" {
			leftKind := kindOf(n.Left, scope)
			rightKind := kindOf(n.Right, scope)
			if leftKind != kindUnknown && rightKind != kindUnknown {
				if (leftKind == kindString || rightKind == kindString || leftKind == kindBytes || rightKind == kindBytes) && leftKind != rightKind {
					return fmt.Errorf("%d:%d: + expects numbers, strings, or bytes of the same kind", n.Op.Line, n.Op.Col)
				}
			}
		}
		return checkExpr(n.Right, scope)
	case *ast.UnaryExpr:
		return checkExpr(n.Expr, scope)
	case *ast.TryExpr:
		return checkExpr(n.Expr, scope)
	case *ast.IfStmt:
		if err := checkExpr(n.Cond, scope); err != nil {
			return err
		}
		if err := checkStmts(n.Then, map[string]bool{}, newScope(scope)); err != nil {
			return err
		}
		return checkStmts(n.Else, map[string]bool{}, newScope(scope))
	case *ast.WhileStmt:
		if err := checkExpr(n.Cond, scope); err != nil {
			return err
		}
		return checkStmts(n.Body, map[string]bool{}, newScope(scope))
	case *ast.ForInStmt:
		if err := checkBindingName(n.ValueName, n.ValueTok.Line, n.ValueTok.Col); err != nil {
			return err
		}
		if n.IndexName != "" {
			if err := checkBindingName(n.IndexName, n.IndexTok.Line, n.IndexTok.Col); err != nil {
				return err
			}
		}
		if err := checkExpr(n.Iterable, scope); err != nil {
			return err
		}
		child := newScope(scope)
		child.define(n.ValueName, kindUnknown)
		if n.IndexName != "" {
			child.define(n.IndexName, kindUnknown)
		}
		return checkStmts(n.Body, map[string]bool{}, child)
	case *ast.MatchStmt:
		if err := checkExpr(n.Value, scope); err != nil {
			return err
		}
		for _, c := range n.Cases {
			caseScope := newScope(scope)
			if err := checkPattern(c.Pattern, caseScope); err != nil {
				return err
			}
			if err := checkStmts(c.Body, map[string]bool{}, caseScope); err != nil {
				return err
			}
		}
	case *ast.SuperExpr:
		return fmt.Errorf("%d:%d: super must be called inside init, an instance method, or a class method", n.Tok.Line, n.Tok.Col)
	case *ast.SelfExpr:
		if n.Class {
			// v0.46 G2: `Self` is valid in any class-body context.
			if scope.currentClass == "" {
				return fmt.Errorf("%d:%d: [TYA-E0412] Self is only valid inside a class body", n.Tok.Line, n.Tok.Col)
			}
		} else {
			// v0.46 G2 / v0.47 G5: `self` (lowercase) refers to the
			// instance. Valid in instance methods and constructors.
			// v0.47 rejects `self` inside `static` methods with
			// [TYA-E0411] (legacy v0.45 allowed it as "the class").
			// Permissive legacy mode preserves the v0.45 acceptance
			// for the selfhost/v01 surface.
			if scope.inClassMethod && !scope.inInstanceMethod {
				if !permissiveLegacy {
					return fmt.Errorf("%d:%d: [TYA-E0411] self is not available in static methods (no instance receiver); use Self for the class", n.Tok.Line, n.Tok.Col)
				}
			}
			if !scope.inInstanceMethod && !scope.inClassMethod {
				return fmt.Errorf("%d:%d: self is only valid inside a class method or instance method", n.Tok.Line, n.Tok.Col)
			}
		}
	case *ast.InstanceFieldExpr:
		// v0.47 G2: reject `@field` outside permissive legacy mode.
		if !permissiveLegacy {
			return fmt.Errorf("%d:%d: [TYA-E0410] @%s is removed; use self.%s", n.NameTok.Line, n.NameTok.Col, n.Name, n.Name)
		}
		if !scope.inInstanceMethod {
			return fmt.Errorf("%d:%d: @%s is only valid inside an instance method", n.NameTok.Line, n.NameTok.Col, n.Name)
		}
		if !valueNameRE.MatchString(n.Name) {
			return fmt.Errorf("%d:%d: invalid field name %s", n.NameTok.Line, n.NameTok.Col, n.Name)
		}
		if err := checkPrivateInstanceAccess(n.Name, n.NameTok.Line, n.NameTok.Col, scope, false); err != nil {
			return err
		}
	case *ast.ClassVarExpr:
		if !scope.inClassBody && !scope.inInstanceMethod && !scope.inClassMethod {
			return fmt.Errorf("%d:%d: @@%s is only valid inside a class", n.NameTok.Line, n.NameTok.Col, n.Name)
		}
		if !valueNameRE.MatchString(n.Name) {
			return fmt.Errorf("%d:%d: invalid class variable name %s", n.NameTok.Line, n.NameTok.Col, n.Name)
		}
		if err := checkPrivateClassAccess(n.Name, n.NameTok.Line, n.NameTok.Col, scope); err != nil {
			return err
		}
	case *ast.MemberExpr:
		if err := checkExpr(n.Target, scope); err != nil {
			return err
		}
		switch kindOf(n.Target, scope) {
		case kindDict:
			if isDictMethodName(n.Name) {
				return nil
			}
			return memberAccessError(n, "dictionary")
		case kindModule:
			if classNameRE.MatchString(n.Name) {
				key := memberKey(n)
				if info, ok := scope.classes[key]; ok {
					if err := checkCrossFilePrivate(info, n.Name, n.NameTok.Line, n.NameTok.Col, scope); err != nil {
						return err
					}
				} else if info, ok := scope.classes[n.Name]; ok {
					if err := checkCrossFilePrivate(info, n.Name, n.NameTok.Line, n.NameTok.Col, scope); err != nil {
						return err
					}
				}
			}
			return nil
		case kindClass:
			if err := checkExternalClassMemberAccess(n, scope); err != nil {
				return err
			}
			return nil
		case kindObject:
			return nil
		default:
			return nil
		}
	case *ast.IndexExpr:
		if err := checkExpr(n.Target, scope); err != nil {
			return err
		}
		if isNegativeNumberLiteral(n.Index) {
			return fmt.Errorf("negative indexes are invalid")
		}
		return checkExpr(n.Index, scope)
	case *ast.CallExpr:
		if super, ok := n.Callee.(*ast.SuperExpr); ok {
			if scope.inInstanceMethod && (scope.currentClass == "" || scope.interfaces[scope.currentClass].name != "") {
				for _, arg := range n.Args {
					if err := checkExpr(arg, scope); err != nil {
						return err
					}
				}
				return nil
			}
			if (!scope.inInstanceMethod && !scope.inClassMethod) || scope.currentMethod == "" || scope.currentClass == "" {
				return fmt.Errorf("%d:%d: super must be called inside init, an instance method, or a class method", super.Tok.Line, super.Tok.Col)
			}
			parent := scope.classes[scope.currentClass].parent
			if parent == "" {
				if scope.inClassMethod {
					return fmt.Errorf("%d:%d: super has no parent class method %s", super.Tok.Line, super.Tok.Col, scope.currentMethod)
				}
				for _, arg := range n.Args {
					if err := checkExpr(arg, scope); err != nil {
						return err
					}
				}
				return nil
			}
			if scope.inClassMethod {
				arity, ok := inheritedClassMethodArity(parent, scope.currentMethod, scope)
				if !ok {
					return fmt.Errorf("%d:%d: super has no parent class method %s", super.Tok.Line, super.Tok.Col, scope.currentMethod)
				}
				if len(n.Args) != arity {
					return fmt.Errorf("%d:%d: super class method %s expects %d arguments", super.Tok.Line, super.Tok.Col, scope.currentMethod, arity)
				}
				if params, ok := effectiveClassMethodParams(parent, scope.currentMethod, scope); ok {
					if err := checkCallKeywords(n, params); err != nil {
						return err
					}
				}
			} else if scope.currentMethod == "init" || scope.currentMethod == "_init" || scope.currentMethod == "initialize" {
				if parentInfo := scope.classes[parent]; parentInfo.privateInit {
					return fmt.Errorf("%d:%d: super cannot call private parent constructor", super.Tok.Line, super.Tok.Col)
				}
				arity, ok := inheritedInitArity(parent, scope)
				if !ok {
					if !scope.classes[scope.currentClass].interfaceInitializers {
						return fmt.Errorf("%d:%d: constructor super has no parent init", super.Tok.Line, super.Tok.Col)
					}
					arity = arityRange{required: 0, max: 0}
				}
				if !arity.accepts(len(n.Args)) {
					return fmt.Errorf("%d:%d: super init expects %s", super.Tok.Line, super.Tok.Col, arity.describe())
				}
				if parentInfo, ok := scope.classes[parent]; ok {
					if err := checkCallKeywords(n, effectiveInitParams(parentInfo, scope)); err != nil {
						return err
					}
				}
			} else {
				if _, ok := inheritedAbstractMethodArity(parent, scope.currentMethod, scope); ok {
					return fmt.Errorf("%d:%d: super cannot call abstract parent method %s", super.Tok.Line, super.Tok.Col, scope.currentMethod)
				}
				arity, ok := inheritedMethodArity(parent, scope.currentMethod, scope)
				if !ok {
					return fmt.Errorf("%d:%d: super has no parent method %s", super.Tok.Line, super.Tok.Col, scope.currentMethod)
				}
				if len(n.Args) != arity {
					return fmt.Errorf("%d:%d: super method %s expects %d arguments", super.Tok.Line, super.Tok.Col, scope.currentMethod, arity)
				}
				if params, ok := effectiveMethodParams(parent, scope.currentMethod, scope); ok {
					if err := checkCallKeywords(n, params); err != nil {
						return err
					}
				}
			}
			for _, arg := range n.Args {
				if err := checkExpr(arg, scope); err != nil {
					return err
				}
			}
			return nil
		}
		if id, ok := n.Callee.(*ast.Ident); ok && scope.currentClass == "" && !permissiveLegacy && os.Getenv("TYA_LEGACY_MODULES") != "1" {
			if err := removedTopLevelPrimitiveBuiltinError(id.Name, id.Tok.Line, id.Tok.Col); err != nil {
				return err
			}
		}
		if member, ok := n.Callee.(*ast.MemberExpr); ok {
			if err := removedPrimitiveModuleCallError(member, scope); err != nil {
				return err
			}
		}
		if id, ok := n.Callee.(*ast.Ident); ok && !scope.defined(id.Name) && scope.currentClass != "" {
			if scope.inInstanceMethod {
				if arity, found := effectiveMethodArity(scope.currentClass, id.Name, scope); found {
					if len(n.Args) != arity {
						return fmt.Errorf("%d:%d: method %s expects %d arguments", id.Tok.Line, id.Tok.Col, id.Name, arity)
					}
					if params, ok := effectiveMethodParams(scope.currentClass, id.Name, scope); ok {
						if err := checkCallKeywords(n, params); err != nil {
							return err
						}
					}
					n.ImplicitSelf = true
				} else if arity, found := effectiveClassMethodArity(scope.currentClass, id.Name, scope); found {
					if len(n.Args) != arity {
						return fmt.Errorf("%d:%d: class method %s expects %d arguments", id.Tok.Line, id.Tok.Col, id.Name, arity)
					}
					if params, ok := effectiveClassMethodParams(scope.currentClass, id.Name, scope); ok {
						if err := checkCallKeywords(n, params); err != nil {
							return err
						}
					}
					n.ImplicitClass = true
				}
			} else if scope.inClassMethod {
				if arity, found := effectiveClassMethodArity(scope.currentClass, id.Name, scope); found {
					if len(n.Args) != arity {
						return fmt.Errorf("%d:%d: class method %s expects %d arguments", id.Tok.Line, id.Tok.Col, id.Name, arity)
					}
					if params, ok := effectiveClassMethodParams(scope.currentClass, id.Name, scope); ok {
						if err := checkCallKeywords(n, params); err != nil {
							return err
						}
					}
					n.ImplicitClass = true
				}
			}
		}
		if !n.ImplicitSelf && !n.ImplicitClass {
			if err := checkExpr(n.Callee, scope); err != nil {
				return err
			}
		}
		if id, ok := n.Callee.(*ast.Ident); ok && scope.kind(id.Name) == kindClass {
			if info, ok := scope.classes[id.Name]; ok {
				if err := checkCallKeywords(n, effectiveInitParams(info, scope)); err != nil {
					return err
				}
			}
			if err := checkClassCall(id.Name, id.Name, id.Tok.Line, id.Tok.Col, len(n.Args), scope); err != nil {
				return err
			}
		}
		if id, ok := n.Callee.(*ast.Ident); ok {
			if params := scope.funcParams[id.Name]; len(params) > 0 {
				if err := checkCallKeywords(n, params); err != nil {
					return err
				}
			}
		}
		if id, ok := n.Callee.(*ast.Ident); ok && scope.kind(id.Name) == kindInterface {
			return fmt.Errorf("%d:%d: cannot construct interface %s", id.Tok.Line, id.Tok.Col, id.Name)
		}
		// v0.44 within-package fallback for class calls: `Foo(args)`
		// inside a module's class method resolves to the sibling
		// `<module>.Foo` class.
		if id, ok := n.Callee.(*ast.Ident); ok && classNameRE.MatchString(id.Name) {
			if mod := currentModulePrefix(scope); mod != "" {
				key := mod + "." + id.Name
				if scope.kind(key) == kindClass {
					if err := checkClassCall(key, id.Name, id.Tok.Line, id.Tok.Col, len(n.Args), scope); err != nil {
						return err
					}
				}
				if scope.kind(key) == kindInterface {
					return fmt.Errorf("%d:%d: cannot construct interface %s", id.Tok.Line, id.Tok.Col, id.Name)
				}
			}
		}
		if member, ok := n.Callee.(*ast.MemberExpr); ok && kindOf(member.Target, scope) == kindModule && classNameRE.MatchString(member.Name) {
			key := memberKey(member)
			if scope.kind(key) == kindClass {
				if err := checkClassCall(key, member.Name, member.NameTok.Line, member.NameTok.Col, len(n.Args), scope); err != nil {
					return err
				}
			} else if scope.kind(member.Name) == kindClass {
				if err := checkClassCall(member.Name, member.Name, member.NameTok.Line, member.NameTok.Col, len(n.Args), scope); err != nil {
					return err
				}
			}
			aliasedKey := aliasedPackageSymbolKey(memberKeyModule(member), member.Name)
			if scope.kind(key) == kindInterface || scope.kind(aliasedKey) == kindInterface {
				return fmt.Errorf("%d:%d: cannot construct interface %s", member.NameTok.Line, member.NameTok.Col, member.Name)
			} else if scope.kind(member.Name) == kindInterface {
				return fmt.Errorf("%d:%d: cannot construct interface %s", member.NameTok.Line, member.NameTok.Col, member.Name)
			}
		}
		if member, ok := n.Callee.(*ast.MemberExpr); ok {
			if err := removedConcurrencyHelperError(member); err != nil {
				return err
			}
			if name := classConstantName(member.Target, scope); name != "" && isKnownMutatingMethod(member.Name) {
				return fmt.Errorf("%d:%d: cannot mutate constant %s", member.NameTok.Line, member.NameTok.Col, name)
			}
		}
		for _, arg := range n.Args {
			if err := checkExpr(arg, scope); err != nil {
				return err
			}
		}
	}
	return nil
}

func removedConcurrencyHelperError(member *ast.MemberExpr) error {
	target, ok := member.Target.(*ast.MemberExpr)
	if !ok {
		return nil
	}
	classKey := memberKey(target)
	switch classKey {
	case "channel.Channel":
		switch member.Name {
		case "new", "send", "receive", "receive_timeout", "close", "closed?", "select":
			return fmt.Errorf("%d:%d: [TYA-E0820] removed concurrency helper API; use instance method style", member.NameTok.Line, member.NameTok.Col)
		}
	case "task.Task":
		switch member.Name {
		case "cancel", "cancelled?":
			return fmt.Errorf("%d:%d: [TYA-E0820] removed concurrency helper API; use instance method style", member.NameTok.Line, member.NameTok.Col)
		}
	case "sync.Sync":
		return fmt.Errorf("%d:%d: [TYA-E0820] removed concurrency helper API; use instance method style", member.NameTok.Line, member.NameTok.Col)
	}
	return nil
}

func checkClass(class *ast.ClassDecl, scope *scope, module string) error {
	if !classNameRE.MatchString(class.Name) {
		return fmt.Errorf("%d:%d: invalid class name %s", class.NameTok.Line, class.NameTok.Col, class.Name)
	}
	if isPrimitiveClassName(class.Name) {
		return fmt.Errorf("%d:%d: [TYA-E0815] cannot rebind reserved class identifier %s", class.NameTok.Line, class.NameTok.Col, class.Name)
	}
	if class.Parent != nil && class.Parent.Module == "" && isPrimitiveClassName(class.Parent.Name) {
		return fmt.Errorf("%d:%d: [TYA-E0813] cannot inherit from built-in primitive class %s", class.Parent.Tok.Line, class.Parent.Tok.Col, class.Parent.Name)
	}
	if class.Abstract && class.Final {
		return fmt.Errorf("%d:%d: class cannot be both abstract and final", class.NameTok.Line, class.NameTok.Col)
	}
	key := classKey(module, class)
	info := scope.classes[key]
	info.interfaceInitializers = classHasInterfaceInitializers(class, module, scope)
	scope.classes[key] = info
	if class.Parent != nil {
		parentKey := refKey(class.Parent, module, scope)
		if _, ok := scope.interfaces[parentKey]; ok {
			return fmt.Errorf("%d:%d: class %s extends interface %s", class.Parent.Tok.Line, class.Parent.Tok.Col, class.Name, parentName(class.Parent))
		}
		parentInfo, ok := scope.classes[parentKey]
		if !ok {
			return fmt.Errorf("%d:%d: unknown parent class %s", class.Parent.Tok.Line, class.Parent.Tok.Col, parentName(class.Parent))
		}
		// v0.45: extends across files into a sibling-file private class
		// is rejected with [TYA-E0406]. Use the class being defined as
		// the reference site.
		if parentInfo.originFile != "" && !isPublicClassInfo(parentInfo) {
			if class.OriginFile != parentInfo.originFile {
				site := class.OriginFile
				if site == "" {
					site = "outside the package"
				}
				return fmt.Errorf("%d:%d: [TYA-E0406] private class %s is not visible from %s (declared in %s)",
					class.Parent.Tok.Line, class.Parent.Tok.Col, parentName(class.Parent), site, parentInfo.originFile)
			}
		}
		if scope.classes[parentKey].final {
			return fmt.Errorf("%d:%d: cannot extend final class %s", class.Parent.Tok.Line, class.Parent.Tok.Col, parentName(class.Parent))
		}
		if hasInheritanceCycle(key, scope) {
			return fmt.Errorf("%d:%d: inheritance cycle involving %s", class.NameTok.Line, class.NameTok.Col, class.Name)
		}
	}
	registerInterfaceContributions(class, scope, key, module)
	instanceMembers := map[string]bool{}
	instanceMethods := map[string]bool{}
	classMembers := map[string]bool{}
	hasPublicInit := false
	hasPrivateInit := false
	classBody := newScope(scope)
	classBody.inClassBody = true
	classBody.currentClass = key
	for _, field := range class.Fields {
		if !valueNameRE.MatchString(field.Name) {
			return fmt.Errorf("%d:%d: invalid field name %s", field.Tok.Line, field.Tok.Col, field.Name)
		}
		// v0.47 G3: reject `_`-prefix instance field names.
		if !permissiveLegacy && strings.HasPrefix(field.Name, "_") {
			return fmt.Errorf("%d:%d: [TYA-E0407] %s is no longer a privacy marker on class members; rename to %s or add `private`", field.Tok.Line, field.Tok.Col, field.Name, strings.TrimPrefix(field.Name, "_"))
		}
		if instanceMembers[field.Name] {
			return fmt.Errorf("%d:%d: duplicate instance member %s", field.Tok.Line, field.Tok.Col, field.Name)
		}
		instanceMembers[field.Name] = true
		if err := checkExpr(field.Value, classBody); err != nil {
			return err
		}
	}
	for _, variable := range class.Vars {
		if !valueNameRE.MatchString(variable.Name) {
			return fmt.Errorf("%d:%d: invalid class variable name %s", variable.Tok.Line, variable.Tok.Col, variable.Name)
		}
		// v0.47 G3: reject `_`-prefix class variable names.
		if !permissiveLegacy && strings.HasPrefix(variable.Name, "_") {
			return fmt.Errorf("%d:%d: [TYA-E0407] %s is no longer a privacy marker on class members; rename to %s or add `private`", variable.Tok.Line, variable.Tok.Col, variable.Name, strings.TrimPrefix(variable.Name, "_"))
		}
		if classMembers[variable.Name] {
			return fmt.Errorf("%d:%d: duplicate class member %s", variable.Tok.Line, variable.Tok.Col, variable.Name)
		}
		classMembers[variable.Name] = true
		if err := checkExpr(variable.Value, classBody); err != nil {
			return err
		}
	}
	for _, constant := range class.Constants {
		if !constNameRE.MatchString(constant.Name) {
			return fmt.Errorf("%d:%d: invalid class constant name %s", constant.Tok.Line, constant.Tok.Col, constant.Name)
		}
		if classMembers[constant.Name] {
			return fmt.Errorf("%d:%d: duplicate class member %s", constant.Tok.Line, constant.Tok.Col, constant.Name)
		}
		classMembers[constant.Name] = true
		if err := checkExpr(constant.Value, classBody); err != nil {
			return err
		}
	}
	for _, method := range class.Methods {
		if !validCallableName(method.Name) {
			return fmt.Errorf("%d:%d: invalid method name %s", method.Tok.Line, method.Tok.Col, method.Name)
		}
		if method.Abstract && !class.Abstract {
			return fmt.Errorf("%d:%d: abstract method %s must be declared inside an abstract class", method.Tok.Line, method.Tok.Col, method.Name)
		}
		if method.Abstract && method.Override {
			return fmt.Errorf("%d:%d: method %s cannot be both abstract and override", method.Tok.Line, method.Tok.Col, method.Name)
		}
		// v0.47 G4: reject `init` / `_init` as constructor names outside
		// permissive legacy mode. Only `initialize` is recognized.
		if !permissiveLegacy && !method.Class && (method.Name == "init" || method.Name == "_init") {
			return fmt.Errorf("%d:%d: [TYA-E0414] %s is removed as a constructor name; rename to initialize", method.Tok.Line, method.Tok.Col, method.Name)
		}
		// v0.47 G3: reject `_`-prefix class member names. Underscore is
		// no longer a privacy marker on class members; use `private`.
		// (Excluded: `_init` already caught above; `_` lone alone is
		// not a valid member name and would fail validCallableName.)
		if !permissiveLegacy && strings.HasPrefix(method.Name, "_") && method.Name != "_init" {
			return fmt.Errorf("%d:%d: [TYA-E0407] %s is no longer a privacy marker on class members; rename to %s or add `private`", method.Tok.Line, method.Tok.Col, method.Name, strings.TrimPrefix(method.Name, "_"))
		}
		if method.Class {
			if classMembers[method.Name] {
				return fmt.Errorf("%d:%d: duplicate class member %s", method.Tok.Line, method.Tok.Col, method.Name)
			}
			classMembers[method.Name] = true
		} else {
			if instanceMethods[method.Name] {
				return fmt.Errorf("%d:%d: duplicate instance member %s", method.Tok.Line, method.Tok.Col, method.Name)
			}
			// v0.46 G1+G5: a constructor is `init` (legacy public),
			// `_init` (legacy private), or `initialize` (canonical).
			// The Private flag (from `private` keyword or `_`-prefix)
			// distinguishes public vs private. A class may not declare
			// both a public and private constructor.
			if method.Name == "init" || method.Name == "initialize" {
				if method.Private {
					hasPrivateInit = true
				} else {
					hasPublicInit = true
				}
			}
			if method.Name == "_init" {
				hasPrivateInit = true
			}
			if hasPublicInit && hasPrivateInit {
				return fmt.Errorf("%d:%d: class cannot declare both init and _init", method.Tok.Line, method.Tok.Col)
			}
			instanceMethods[method.Name] = true
		}
		child := newScope(scope)
		child.inClassBody = true
		child.currentClass = key
		child.currentMethod = method.Name
		if method.Abstract {
			continue
		}
		if method.Class {
			child.inClassMethod = true
		} else {
			child.inInstanceMethod = true
		}
		if err := checkExpr(method.Func, child); err != nil {
			return err
		}
		if !method.Class {
			parent := scope.classes[key].parent
			if parent != "" {
				if method.Name == "init" || method.Name == "initialize" {
					if err := checkConstructorSuper(class, method, parent, scope, module); err != nil {
						return err
					}
					if _, ok := inheritedInitArity(parent, scope); ok && !funcCallsSuper(method.Func) {
						return fmt.Errorf("%d:%d: subclass init must call super", method.Tok.Line, method.Tok.Col)
					}
					if classHasInterfaceInitializers(class, module, scope) && !funcCallsSuper(method.Func) {
						return fmt.Errorf("%d:%d: class constructor must call super to run interface initialization", method.Tok.Line, method.Tok.Col)
					}
				} else {
					if err := checkOverrideMethod(class, method, parent, scope); err != nil {
						return err
					}
					if arity, ok := inheritedMethodArity(parent, method.Name, scope); ok && arity != len(method.Func.Params) {
						return fmt.Errorf("%d:%d: overriding method %s expects %d parameters", method.Tok.Line, method.Tok.Col, method.Name, arity)
					} else if arity, ok := inheritedAbstractMethodArity(parent, method.Name, scope); ok && arity != len(method.Func.Params) {
						return fmt.Errorf("%d:%d: implementing abstract method %s expects %d parameters", method.Tok.Line, method.Tok.Col, method.Name, arity)
					}
				}
			} else if method.Override {
				return fmt.Errorf("%d:%d: override method %s has no inherited method target", method.Tok.Line, method.Tok.Col, method.Name)
			}
			if parent == "" && (method.Name == "init" || method.Name == "initialize") && classHasInterfaceInitializers(class, module, scope) && !funcCallsSuper(method.Func) {
				return fmt.Errorf("%d:%d: class constructor must call super to run interface initialization", method.Tok.Line, method.Tok.Col)
			}
		} else {
			parent := scope.classes[key].parent
			if parent != "" {
				if err := checkOverrideMethod(class, method, parent, scope); err != nil {
					return err
				}
				if arity, ok := inheritedClassMethodArity(parent, method.Name, scope); ok && arity != len(method.Func.Params) {
					return fmt.Errorf("%d:%d: overriding class method %s expects %d parameters", method.Tok.Line, method.Tok.Col, method.Name, arity)
				} else if arity, ok := inheritedAbstractClassMethodArity(parent, method.Name, scope); ok && arity != len(method.Func.Params) {
					return fmt.Errorf("%d:%d: implementing abstract class method %s expects %d parameters", method.Tok.Line, method.Tok.Col, method.Name, arity)
				}
			} else if method.Override {
				return fmt.Errorf("%d:%d: override class method %s has no inherited class method target", method.Tok.Line, method.Tok.Col, method.Name)
			}
		}
	}
	if err := checkAbstractImplementations(class, scope, key); err != nil {
		return err
	}
	if err := checkInterfaceImplementations(class, scope, key, module); err != nil {
		return err
	}
	return nil
}

func checkInterface(iface *ast.InterfaceDecl, scope *scope, module string) error {
	if !classNameRE.MatchString(iface.Name) {
		return fmt.Errorf("%d:%d: invalid interface name %s", iface.NameTok.Line, iface.NameTok.Col, iface.Name)
	}
	key := refKey(&ast.ClassRef{Name: iface.Name, Tok: iface.NameTok}, module, scope)
	seen := map[string]bool{}
	interfaceFields := map[string]bool{}
	for _, field := range iface.Fields {
		interfaceFields[field.Name] = true
	}
	for _, method := range iface.Methods {
		if !validCallableName(method.Name) || isPrivateName(method.Name) {
			return fmt.Errorf("%d:%d: invalid interface method %s", method.Tok.Line, method.Tok.Col, method.Name)
		}
		if seen[method.Name] {
			return fmt.Errorf("%d:%d: duplicate interface method %s", method.Tok.Line, method.Tok.Col, method.Name)
		}
		seen[method.Name] = true
		if method.Func != nil {
			if method.Name == "initialize" && len(method.Params) != 0 {
				return fmt.Errorf("%d:%d: interface initialize must not take arguments", method.Tok.Line, method.Tok.Col)
			}
			if containsClassSelfRef(method.Func) {
				return fmt.Errorf("%d:%d: Self is not allowed in interface methods", method.Tok.Line, method.Tok.Col)
			}
			child := newScope(scope)
			child.inInstanceMethod = true
			child.currentClass = key
			child.interfaceFields = interfaceFields
			if err := checkExpr(method.Func, child); err != nil {
				return err
			}
		}
	}
	for _, field := range iface.Fields {
		if !valueNameRE.MatchString(field.Name) || isPrivateName(field.Name) {
			return fmt.Errorf("%d:%d: invalid interface field %s", field.Tok.Line, field.Tok.Col, field.Name)
		}
		if seen[field.Name] {
			return fmt.Errorf("%d:%d: duplicate interface member %s", field.Tok.Line, field.Tok.Col, field.Name)
		}
		seen[field.Name] = true
		if containsSelfRef(field.Value) {
			return fmt.Errorf("%d:%d: interface field initializers cannot reference self", field.Tok.Line, field.Tok.Col)
		}
		if err := checkExpr(field.Value, scope); err != nil {
			return err
		}
	}
	if _, err := effectiveInterfaceRequirements(key, scope, nil); err != nil {
		return err
	}
	return nil
}

func classKey(module string, class *ast.ClassDecl) string {
	if module != "" {
		return module + "." + class.Name
	}
	return class.Name
}

func containsSelfRef(expr ast.Expr) bool {
	switch n := expr.(type) {
	case *ast.SelfExpr:
		return true
	case *ast.MemberExpr:
		return containsSelfRef(n.Target)
	case *ast.IndexExpr:
		return containsSelfRef(n.Target) || containsSelfRef(n.Index)
	case *ast.CallExpr:
		if containsSelfRef(n.Callee) {
			return true
		}
		for _, arg := range n.Args {
			if containsSelfRef(arg) {
				return true
			}
		}
	case *ast.BinaryExpr:
		return containsSelfRef(n.Left) || containsSelfRef(n.Right)
	case *ast.UnaryExpr:
		return containsSelfRef(n.Expr)
	case *ast.ArrayLit:
		for _, elem := range n.Elems {
			if containsSelfRef(elem) {
				return true
			}
		}
	case *ast.DictLit:
		for _, prop := range n.Props {
			if containsSelfRef(prop.Value) {
				return true
			}
		}
	}
	return false
}

func containsClassSelfRef(expr ast.Expr) bool {
	switch n := expr.(type) {
	case *ast.SelfExpr:
		return n.Class
	case *ast.FuncLit:
		if n.Expr != nil && containsClassSelfRef(n.Expr) {
			return true
		}
		for _, stmt := range n.Body {
			if stmtContainsClassSelfRef(stmt) {
				return true
			}
		}
	case *ast.MemberExpr:
		return containsClassSelfRef(n.Target)
	case *ast.CallExpr:
		if containsClassSelfRef(n.Callee) {
			return true
		}
		for _, arg := range n.Args {
			if containsClassSelfRef(arg) {
				return true
			}
		}
	case *ast.BinaryExpr:
		return containsClassSelfRef(n.Left) || containsClassSelfRef(n.Right)
	case *ast.UnaryExpr:
		return containsClassSelfRef(n.Expr)
	}
	return false
}

func stmtContainsClassSelfRef(stmt ast.Stmt) bool {
	switch n := stmt.(type) {
	case *ast.ExprStmt:
		return containsClassSelfRef(n.Expr)
	case *ast.AssignStmt:
		for _, target := range n.Targets {
			if containsClassSelfRef(target) {
				return true
			}
		}
		for _, value := range n.Values {
			if containsClassSelfRef(value) {
				return true
			}
		}
	case *ast.ReturnStmt:
		for _, value := range n.Values {
			if containsClassSelfRef(value) {
				return true
			}
		}
	case *ast.IfStmt:
		if containsClassSelfRef(n.Cond) {
			return true
		}
		for _, st := range append(n.Then, n.Else...) {
			if stmtContainsClassSelfRef(st) {
				return true
			}
		}
	}
	return false
}

// currentModulePrefix returns the module prefix when the scope is
// positioned inside a module class method or class body. The
// module prefix is the part before the first dot in scope.currentClass
// (a module class is keyed as "module.ClassName"). Returns "" when
// the current class is at top level (no dot in the key) or no class
// context is set.
func currentModulePrefix(scope *scope) string {
	if scope == nil || scope.currentClass == "" {
		return ""
	}
	if i := strings.IndexByte(scope.currentClass, '.'); i > 0 {
		return scope.currentClass[:i]
	}
	return ""
}

func refKey(ref *ast.ClassRef, currentModule string, scope *scope) string {
	if ref.Module != "" {
		if key := aliasedPackageSymbolKey(ref.Module, ref.Name); scope.kind(key) == kindInterface || scope.kind(key) == kindClass {
			return key
		}
		return ref.Module + "." + ref.Name
	}
	if currentModule != "" {
		if _, ok := scope.classes[currentModule+"."+ref.Name]; ok {
			return currentModule + "." + ref.Name
		}
		if _, ok := scope.interfaces[currentModule+"."+ref.Name]; ok {
			return currentModule + "." + ref.Name
		}
	}
	return ref.Name
}

func aliasedPackageSymbolKey(module, name string) string {
	return name + "TyaPkg" + pascalIdentifier(module)
}

func pascalIdentifier(name string) string {
	var out strings.Builder
	capNext := true
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			if capNext && r >= 'a' && r <= 'z' {
				r = r - 'a' + 'A'
			}
			out.WriteRune(r)
			capNext = false
			continue
		}
		capNext = true
	}
	if out.Len() == 0 {
		return "Package"
	}
	return out.String()
}

func parentName(ref *ast.ClassRef) string {
	if ref.Module != "" {
		return ref.Module + "." + ref.Name
	}
	return ref.Name
}

func hasInheritanceCycle(start string, scope *scope) bool {
	seen := map[string]bool{}
	current := start
	for current != "" {
		if seen[current] {
			return true
		}
		seen[current] = true
		current = scope.classes[current].parent
	}
	return false
}

func effectiveInit(info classInfo, scope *scope) (bool, arityRange) {
	if info.hasInit {
		return true, arityRange{required: info.initRequired, max: info.initArity}
	}
	if info.parent != "" {
		rng, ok := inheritedInitArity(info.parent, scope)
		return ok, rng
	}
	return false, arityRange{}
}

func effectiveInitParams(info classInfo, scope *scope) []string {
	if info.hasInit {
		return info.initParams
	}
	if info.parent != "" {
		if parent, ok := scope.classes[info.parent]; ok {
			return effectiveInitParams(parent, scope)
		}
	}
	return nil
}

func requiredParamCount(fn *ast.FuncLit) int {
	required := len(fn.Params)
	for i, def := range fn.Defaults {
		if def != nil && i < required {
			required = i
		}
	}
	return required
}

func (r arityRange) accepts(argc int) bool {
	return argc >= r.required && argc <= r.max
}

func (r arityRange) describe() string {
	if r.required == r.max {
		return fmt.Sprintf("%d arguments", r.max)
	}
	return fmt.Sprintf("%d to %d arguments", r.required, r.max)
}

func checkCallKeywords(call *ast.CallExpr, params []string) error {
	if call.PositionalArgsOnly() {
		return nil
	}
	if len(params) == 0 {
		return nil
	}
	indexes := map[string]int{}
	for i, param := range params {
		indexes[param] = i
	}
	filled := map[int]string{}
	pos := 0
	for _, arg := range call.EffectiveArgs() {
		if arg.Expand {
			continue
		}
		if arg.Name == "" {
			if pos < len(params) {
				filled[pos] = params[pos]
			}
			pos++
			continue
		}
		index, ok := indexes[arg.Name]
		if !ok {
			return fmt.Errorf("%d:%d: unknown keyword %s", arg.NameTok.Line, arg.NameTok.Col, arg.Name)
		}
		if filled[index] != "" {
			return fmt.Errorf("%d:%d: argument %s supplied multiple times", arg.NameTok.Line, arg.NameTok.Col, arg.Name)
		}
		filled[index] = arg.Name
	}
	return nil
}

// bareClassName strips the leading "module." prefix from a class
// info key, returning just the class name segment.
func bareClassName(key string) string {
	if i := strings.LastIndexByte(key, '.'); i >= 0 {
		return key[i+1:]
	}
	return key
}

// isPublicClassInfo reports whether a class is the public class of
// its source file: its bare name maps to OriginFile in snake_case. A
// class without OriginFile is considered public (no cross-file
// metadata to enforce against).
func isPublicClassInfo(info classInfo) bool {
	if info.originFile == "" {
		return true
	}
	return SnakeCaseName(bareClassName(info.name))+".tya" == info.originFile
}

// checkCrossFilePrivate enforces the v0.45 cross-file private class
// rule. If target is a private class (filename does not match) and
// the current reference site (scope.currentClass's origin) lives in
// a different file, emit [TYA-E0406]. A site without an origin file
// (entry script, non-package code) cannot reach any non-public class
// in a package.
func checkCrossFilePrivate(target classInfo, display string, line, col int, scope *scope) error {
	if target.originFile == "" || isPublicClassInfo(target) {
		return nil
	}
	siteOrigin := ""
	if scope.currentClass != "" {
		siteOrigin = scope.classes[scope.currentClass].originFile
	}
	if siteOrigin == target.originFile {
		return nil
	}
	siteDisplay := siteOrigin
	if siteDisplay == "" {
		siteDisplay = "outside the package"
	}
	return fmt.Errorf("%d:%d: [TYA-E0406] private class %s is not visible from %s (declared in %s)",
		line, col, display, siteDisplay, target.originFile)
}

func checkClassCall(key, display string, line, col, argc int, scope *scope) error {
	info := scope.classes[key]
	if err := checkCrossFilePrivate(info, display, line, col, scope); err != nil {
		return err
	}
	if arity, ok := nativeConcurrencyConstructorArity(key); ok {
		if argc != arity {
			return fmt.Errorf("%d:%d: class %s constructor expects %d arguments", line, col, display, arity)
		}
		return nil
	}
	if info.abstract {
		return fmt.Errorf("%d:%d: cannot construct abstract class %s", line, col, display)
	}
	if info.privateInit && scope.currentClass != key {
		return fmt.Errorf("%d:%d: class %s constructor is private", line, col, display)
	}
	hasInit, arity := effectiveInit(info, scope)
	if !hasInit && argc > 0 {
		return fmt.Errorf("%d:%d: class %s has no init and takes no arguments", line, col, display)
	}
	if hasInit && !arity.accepts(argc) {
		return fmt.Errorf("%d:%d: class %s constructor expects %s", line, col, display, arity.describe())
	}
	return nil
}

func nativeConcurrencyConstructorArity(key string) (int, bool) {
	switch key {
	case "channel.Channel", "Channel":
		return 1, true
	case "sync.Mutex", "Mutex":
		return 0, true
	case "sync.AtomicInteger", "AtomicInteger":
		return 1, true
	case "sync.WaitGroup", "WaitGroup":
		return 0, true
	default:
		return 0, false
	}
}

func checkPrivateInstanceAccess(name string, line, col int, scope *scope, assignment bool) error {
	if scope.currentClass == "" {
		return nil
	}
	info := scope.classes[scope.currentClass]
	if info.privateFields[name] || info.privateMethods[name] {
		return nil
	}
	if assignment {
		return nil
	}
	// v0.46 G1: walk the ancestor chain regardless of the name's
	// underscore prefix. Privacy is recorded on classInfo from the
	// AST's Private flag (which captures both `_`-prefix and the new
	// `private` keyword).
	for parent := info.parent; parent != ""; {
		parentInfo := scope.classes[parent]
		if parentInfo.privateFields[name] || parentInfo.privateMethods[name] {
			return fmt.Errorf("%d:%d: private instance member %s is not accessible from %s", line, col, name, scope.currentClass)
		}
		parent = parentInfo.parent
	}
	return nil
}

func checkPrivateClassAccess(name string, line, col int, scope *scope) error {
	// v0.46 G1: see checkPrivateInstanceAccess. Walk ancestors
	// regardless of name shape; privacy is on classInfo.
	if scope.currentClass == "" {
		return nil
	}
	info := scope.classes[scope.currentClass]
	if info.privateClassMembers[name] || info.privateClassMethods[name] {
		return nil
	}
	for parent := info.parent; parent != ""; {
		parentInfo := scope.classes[parent]
		if parentInfo.privateClassMembers[name] || parentInfo.privateClassMethods[name] {
			return fmt.Errorf("%d:%d: private class member %s is not accessible from %s", line, col, name, scope.currentClass)
		}
		parent = parentInfo.parent
	}
	return nil
}

func checkExternalClassMemberAccess(member *ast.MemberExpr, scope *scope) error {
	key := classExprKey(member.Target, scope)
	if key == "" {
		return nil
	}
	info, ok := scope.classes[key]
	if !ok || !info.privateClassMembers[member.Name] {
		return nil
	}
	if scope.currentClass == key {
		return nil
	}
	return fmt.Errorf("%d:%d: private class member %s is not accessible from %s", member.NameTok.Line, member.NameTok.Col, member.Name, accessSite(scope))
}

func accessSite(scope *scope) string {
	if scope.currentClass != "" {
		return scope.currentClass
	}
	return "outside the class"
}

func classExprKey(expr ast.Expr, scope *scope) string {
	switch n := expr.(type) {
	case *ast.SelfExpr:
		if n.Class {
			return scope.currentClass
		}
	case *ast.Ident:
		if scope.kind(n.Name) == kindClass {
			return n.Name
		}
		if mod := currentModulePrefix(scope); mod != "" {
			key := mod + "." + n.Name
			if scope.kind(key) == kindClass {
				return key
			}
		}
	case *ast.MemberExpr:
		if kindOf(n, scope) == kindClass {
			key := memberKey(n)
			if scope.kind(key) == kindClass {
				return key
			}
			if scope.kind(n.Name) == kindClass {
				return n.Name
			}
		}
	}
	return ""
}

func classConstantName(expr ast.Expr, scope *scope) string {
	member, ok := expr.(*ast.MemberExpr)
	if !ok {
		return ""
	}
	key := classExprKey(member.Target, scope)
	if key == "" {
		return ""
	}
	info, ok := scope.classes[key]
	if !ok || !info.classConstants[member.Name] {
		return ""
	}
	return member.Name
}

func exprLineCol(expr ast.Expr) (int, int) {
	switch n := expr.(type) {
	case *ast.Ident:
		return n.Tok.Line, n.Tok.Col
	case *ast.MemberExpr:
		return n.NameTok.Line, n.NameTok.Col
	case *ast.ClassVarExpr:
		return n.NameTok.Line, n.NameTok.Col
	case *ast.InstanceFieldExpr:
		return n.NameTok.Line, n.NameTok.Col
	}
	return 0, 0
}

func isKnownMutatingMethod(name string) bool {
	switch name {
	case "push", "pop", "clear", "delete", "set", "append", "remove":
		return true
	default:
		return false
	}
}

func inheritedInitArity(className string, scope *scope) (arityRange, bool) {
	for className != "" {
		info, ok := scope.classes[className]
		if !ok {
			return arityRange{}, false
		}
		if info.hasInit && !info.privateInit {
			return arityRange{required: info.initRequired, max: info.initArity}, true
		}
		className = info.parent
	}
	return arityRange{}, false
}

func inheritedMethodArity(className string, method string, scope *scope) (int, bool) {
	for className != "" {
		info, ok := scope.classes[className]
		if !ok {
			return 0, false
		}
		if arity, ok := info.methods[method]; ok {
			return arity, true
		}
		className = info.parent
	}
	return 0, false
}

func inheritedClassMethodArity(className string, method string, scope *scope) (int, bool) {
	for className != "" {
		info, ok := scope.classes[className]
		if !ok {
			return 0, false
		}
		if arity, ok := info.classMethods[method]; ok {
			return arity, true
		}
		className = info.parent
	}
	return 0, false
}

func effectiveClassMethodArity(className string, method string, scope *scope) (int, bool) {
	info, ok := scope.classes[className]
	if !ok {
		return 0, false
	}
	if arity, ok := info.classMethods[method]; ok {
		return arity, true
	}
	return inheritedClassMethodArity(info.parent, method, scope)
}

func effectiveInstanceField(name string, className string, scope *scope) bool {
	for className != "" {
		info, ok := scope.classes[className]
		if !ok {
			return false
		}
		if info.fields[name] {
			return true
		}
		className = info.parent
	}
	return false
}

func inheritedAbstractMethodArity(className string, method string, scope *scope) (int, bool) {
	for className != "" {
		info, ok := scope.classes[className]
		if !ok {
			return 0, false
		}
		if _, ok := info.methods[method]; ok {
			return 0, false
		}
		if arity, ok := info.abstractMethods[method]; ok {
			return arity, true
		}
		className = info.parent
	}
	return 0, false
}

func inheritedAbstractClassMethodArity(className string, method string, scope *scope) (int, bool) {
	for className != "" {
		info, ok := scope.classes[className]
		if !ok {
			return 0, false
		}
		if _, ok := info.classMethods[method]; ok {
			return 0, false
		}
		if arity, ok := info.abstractClassMethods[method]; ok {
			return arity, true
		}
		className = info.parent
	}
	return 0, false
}

func checkOverrideMethod(class *ast.ClassDecl, method ast.ClassMethod, parent string, scope *scope) error {
	if !method.Override {
		return nil
	}
	if method.Class {
		if arity, ok := inheritedClassMethodArity(parent, method.Name, scope); ok {
			if arity != len(method.Func.Params) {
				return fmt.Errorf("%d:%d: override class method %s expects %d parameters", method.Tok.Line, method.Tok.Col, method.Name, arity)
			}
			return nil
		}
		if arity, ok := inheritedAbstractClassMethodArity(parent, method.Name, scope); ok {
			if arity != len(method.Func.Params) {
				return fmt.Errorf("%d:%d: override class method %s expects %d parameters", method.Tok.Line, method.Tok.Col, method.Name, arity)
			}
			return nil
		}
		if _, ok := inheritedMethodArity(parent, method.Name, scope); ok {
			return fmt.Errorf("%d:%d: override class method %s targets inherited instance method", method.Tok.Line, method.Tok.Col, method.Name)
		}
		if _, ok := inheritedAbstractMethodArity(parent, method.Name, scope); ok {
			return fmt.Errorf("%d:%d: override class method %s targets inherited instance method", method.Tok.Line, method.Tok.Col, method.Name)
		}
		return fmt.Errorf("%d:%d: override class method %s has no inherited class method target", method.Tok.Line, method.Tok.Col, method.Name)
	}
	if arity, ok := inheritedMethodArity(parent, method.Name, scope); ok {
		if arity != len(method.Func.Params) {
			return fmt.Errorf("%d:%d: override method %s expects %d parameters", method.Tok.Line, method.Tok.Col, method.Name, arity)
		}
		return nil
	}
	if arity, ok := inheritedAbstractMethodArity(parent, method.Name, scope); ok {
		if arity != len(method.Func.Params) {
			return fmt.Errorf("%d:%d: override method %s expects %d parameters", method.Tok.Line, method.Tok.Col, method.Name, arity)
		}
		return nil
	}
	if _, ok := inheritedClassMethodArity(parent, method.Name, scope); ok {
		return fmt.Errorf("%d:%d: override method %s targets inherited class method", method.Tok.Line, method.Tok.Col, method.Name)
	}
	if _, ok := inheritedAbstractClassMethodArity(parent, method.Name, scope); ok {
		return fmt.Errorf("%d:%d: override method %s targets inherited class method", method.Tok.Line, method.Tok.Col, method.Name)
	}
	return fmt.Errorf("%d:%d: override method %s has no inherited method target", method.Tok.Line, method.Tok.Col, method.Name)
}

func checkConstructorSuper(class *ast.ClassDecl, method ast.ClassMethod, parent string, scope *scope, module string) error {
	count, firstSuper, firstField, firstReturn := constructorSuperStats(method.Func)
	if count > 1 {
		return fmt.Errorf("%d:%d: constructor super called more than once", method.Tok.Line, method.Tok.Col)
	}
	parentHasInit := false
	if _, ok := inheritedInitArity(parent, scope); ok {
		parentHasInit = true
	}
	hasInterfaceInit := classHasInterfaceInitializers(class, module, scope)
	if count > 0 && !parentHasInit && !hasInterfaceInit {
		return fmt.Errorf("%d:%d: constructor super has no parent init", method.Tok.Line, method.Tok.Col)
	}
	if parentHasInit && count == 0 {
		return nil
	}
	if count == 0 {
		return nil
	}
	if firstField != 0 && (firstSuper == 0 || firstField < firstSuper) {
		return fmt.Errorf("%d:%d: instance field access before constructor super", firstField, 1)
	}
	if firstReturn != 0 && (firstSuper == 0 || firstReturn < firstSuper) {
		return fmt.Errorf("%d:%d: return before constructor super", firstReturn, 1)
	}
	if parentInfo := scope.classes[parent]; parentInfo.privateInit {
		return fmt.Errorf("%d:%d: super cannot call private parent constructor", method.Tok.Line, method.Tok.Col)
	}
	_ = class
	return nil
}

func checkAbstractImplementations(class *ast.ClassDecl, scope *scope, key string) error {
	instanceReqs, classReqs := effectiveAbstractRequirements(key, scope)
	if class.Abstract {
		return nil
	}
	if len(instanceReqs) > 0 {
		for name := range instanceReqs {
			return fmt.Errorf("%d:%d: class %s must implement abstract method %s", class.NameTok.Line, class.NameTok.Col, class.Name, name)
		}
	}
	if len(classReqs) > 0 {
		for name := range classReqs {
			return fmt.Errorf("%d:%d: class %s must implement abstract class method %s", class.NameTok.Line, class.NameTok.Col, class.Name, name)
		}
	}
	return nil
}

func checkInterfaceImplementations(class *ast.ClassDecl, scope *scope, key string, module string) error {
	reqs := map[string]int{}
	defaults := map[string]int{}
	defaultSources := map[string]string{}
	interfaceFields := map[string]string{}
	var interfaceOrder []string
	if parent := scope.classes[key].parent; parent != "" {
		for name, arity := range scope.classes[parent].interfaceMethods {
			reqs[name] = arity
		}
	}
	seen := map[string]bool{}
	for _, ref := range class.Implements {
		ifaceKey := refKey(&ref, module, scope)
		if seen[ifaceKey] {
			return fmt.Errorf("%d:%d: duplicate interface %s", ref.Tok.Line, ref.Tok.Col, parentName(&ref))
		}
		seen[ifaceKey] = true
		if scope.kind(ifaceKey) != kindInterface {
			return fmt.Errorf("%d:%d: implements target %s is not an interface", ref.Tok.Line, ref.Tok.Col, parentName(&ref))
		}
		interfaceOrder = append(interfaceOrder, effectiveInterfaceOrder(ifaceKey, scope, nil)...)
		ifaceReqs, err := effectiveInterfaceRequirements(ifaceKey, scope, nil)
		if err != nil {
			return err
		}
		for name, req := range ifaceReqs {
			if existing, ok := reqs[name]; ok && existing != req.arity {
				return fmt.Errorf("%d:%d: conflicting interface method %s arity requirements", ref.Tok.Line, ref.Tok.Col, name)
			}
			reqs[name] = req.arity
		}
		for name, arity := range effectiveInterfaceDefaults(ifaceKey, scope, nil) {
			defaults[name] = arity
		}
		for name, source := range effectiveInterfaceDefaultSources(ifaceKey, scope, nil) {
			if existing, ok := defaultSources[name]; ok && existing != source && !classDeclaresMethod(class, name) {
				return fmt.Errorf("%d:%d: conflicting interface default method %s", ref.Tok.Line, ref.Tok.Col, name)
			}
			defaultSources[name] = source
		}
		for name, source := range effectiveInterfaceFields(ifaceKey, scope, nil) {
			if existing, ok := interfaceFields[name]; ok && existing != source && !classDeclaresField(class, name) {
				return fmt.Errorf("%d:%d: conflicting interface field %s", ref.Tok.Line, ref.Tok.Col, name)
			}
			interfaceFields[name] = source
		}
	}
	seenDefaultInOrder := map[string]bool{}
	for _, ifaceKey := range interfaceOrder {
		info := scope.interfaces[ifaceKey]
		for name := range info.defaultsCallSuper {
			if !seenDefaultInOrder[name] && !classDeclaresMethod(class, name) {
				return fmt.Errorf("%d:%d: super has no next method in interface stack", class.NameTok.Line, class.NameTok.Col)
			}
		}
		for name := range info.defaults {
			seenDefaultInOrder[name] = true
		}
	}
	for name, arity := range defaults {
		if reqArity, ok := reqs[name]; ok && reqArity == arity {
			delete(reqs, name)
		}
	}
	info := scope.classes[key]
	info.interfaceMethods = reqs
	for name := range interfaceFields {
		info.fields[name] = true
	}
	scope.classes[key] = info
	if len(reqs) == 0 {
		return nil
	}
	if class.Abstract {
		for name, arity := range reqs {
			if methodArity, ok := info.methods[name]; ok && methodArity != arity {
				return fmt.Errorf("%d:%d: implementing interface method %s expects %d parameters", class.NameTok.Line, class.NameTok.Col, name, arity)
			}
			if methodArity, ok := info.abstractMethods[name]; ok && methodArity != arity {
				return fmt.Errorf("%d:%d: abstract interface method %s expects %d parameters", class.NameTok.Line, class.NameTok.Col, name, arity)
			}
		}
		return nil
	}
	for name, arity := range reqs {
		if methodArity, ok := effectiveMethodArity(key, name, scope); !ok {
			return fmt.Errorf("%d:%d: class %s must implement interface method %s", class.NameTok.Line, class.NameTok.Col, class.Name, name)
		} else if methodArity != arity {
			return fmt.Errorf("%d:%d: implementing interface method %s expects %d parameters", class.NameTok.Line, class.NameTok.Col, name, arity)
		}
	}
	return nil
}

func registerInterfaceContributions(class *ast.ClassDecl, scope *scope, key string, module string) {
	info := scope.classes[key]
	if parent := info.parent; parent != "" {
		for name := range scope.classes[parent].fields {
			info.fields[name] = true
		}
	}
	for _, ref := range class.Implements {
		ifaceKey := refKey(&ref, module, scope)
		if scope.kind(ifaceKey) != kindInterface {
			continue
		}
		for name, arity := range effectiveInterfaceDefaults(ifaceKey, scope, nil) {
			if _, ok := info.methods[name]; !ok {
				info.methods[name] = arity
			}
		}
		for name := range effectiveInterfaceFields(ifaceKey, scope, nil) {
			info.fields[name] = true
		}
	}
	scope.classes[key] = info
}

func classDeclaresField(class *ast.ClassDecl, name string) bool {
	for _, field := range class.Fields {
		if field.Name == name {
			return true
		}
	}
	return false
}

func classDeclaresMethod(class *ast.ClassDecl, name string) bool {
	for _, method := range class.Methods {
		if !method.Class && method.Name == name {
			return true
		}
	}
	return false
}

func classHasInterfaceInitializers(class *ast.ClassDecl, module string, scope *scope) bool {
	for _, ref := range class.Implements {
		if hasInterfaceInitializer(refKey(&ref, module, scope), scope, nil) {
			return true
		}
	}
	return false
}

func hasInterfaceInitializer(key string, scope *scope, stack []string) bool {
	info, ok := scope.interfaces[key]
	if !ok {
		return false
	}
	for _, active := range stack {
		if active == key {
			return false
		}
	}
	stack = append(stack, key)
	if info.initializer {
		return true
	}
	for _, parent := range info.parents {
		if hasInterfaceInitializer(parent, scope, stack) {
			return true
		}
	}
	return false
}

func effectiveInterfaceOrder(key string, scope *scope, stack []string) []string {
	info, ok := scope.interfaces[key]
	if !ok {
		return nil
	}
	for _, active := range stack {
		if active == key {
			return nil
		}
	}
	stack = append(stack, key)
	var out []string
	for _, parent := range info.parents {
		out = append(out, effectiveInterfaceOrder(parent, scope, stack)...)
	}
	out = append(out, key)
	return out
}

func effectiveInterfaceDefaults(key string, scope *scope, stack []string) map[string]int {
	info, ok := scope.interfaces[key]
	if !ok {
		return nil
	}
	for _, active := range stack {
		if active == key {
			return nil
		}
	}
	stack = append(stack, key)
	defaults := map[string]int{}
	for _, parent := range info.parents {
		for name, arity := range effectiveInterfaceDefaults(parent, scope, stack) {
			defaults[name] = arity
		}
	}
	for name, arity := range info.defaults {
		defaults[name] = arity
	}
	return defaults
}

func effectiveInterfaceFields(key string, scope *scope, stack []string) map[string]string {
	info, ok := scope.interfaces[key]
	if !ok {
		return nil
	}
	for _, active := range stack {
		if active == key {
			return nil
		}
	}
	stack = append(stack, key)
	fields := map[string]string{}
	for _, parent := range info.parents {
		for name, source := range effectiveInterfaceFields(parent, scope, stack) {
			fields[name] = source
		}
	}
	for name := range info.fields {
		fields[name] = key
	}
	return fields
}

func effectiveInterfaceDefaultSources(key string, scope *scope, stack []string) map[string]string {
	info, ok := scope.interfaces[key]
	if !ok {
		return nil
	}
	for _, active := range stack {
		if active == key {
			return nil
		}
	}
	stack = append(stack, key)
	defaults := map[string]string{}
	for _, parent := range info.parents {
		for name, source := range effectiveInterfaceDefaultSources(parent, scope, stack) {
			defaults[name] = source
		}
	}
	for name := range info.defaults {
		defaults[name] = key
	}
	return defaults
}

func effectiveInterfaceRequirements(key string, scope *scope, stack []string) (map[string]interfaceRequirement, error) {
	info, ok := scope.interfaces[key]
	if !ok {
		if scope.kind(key) == kindClass {
			return nil, fmt.Errorf("interface %s extends class %s", displayName(key), displayName(key))
		}
		return nil, fmt.Errorf("unknown interface %s", displayName(key))
	}
	for _, active := range stack {
		if active == key {
			return nil, fmt.Errorf("%d:%d: interface inheritance cycle involving %s", info.tokLine, info.tokCol, displayName(key))
		}
	}
	stack = append(stack, key)
	reqs := map[string]interfaceRequirement{}
	for name, arity := range info.methods {
		reqs[name] = interfaceRequirement{arity: arity, source: key}
	}
	for _, parent := range info.parents {
		if scope.kind(parent) == kindClass {
			return nil, fmt.Errorf("%d:%d: interface %s extends class %s", info.tokLine, info.tokCol, displayName(key), displayName(parent))
		}
		if scope.kind(parent) != kindInterface {
			return nil, fmt.Errorf("%d:%d: unknown interface %s", info.tokLine, info.tokCol, displayName(parent))
		}
		parentReqs, err := effectiveInterfaceRequirements(parent, scope, stack)
		if err != nil {
			return nil, err
		}
		for name, parentReq := range parentReqs {
			if existing, ok := reqs[name]; ok && existing.arity != parentReq.arity {
				return nil, fmt.Errorf("%d:%d: interface %s has conflicting method requirement %s: %s.%s expects %d arguments, %s.%s expects %d arguments",
					info.tokLine, info.tokCol, displayName(key), name,
					displayName(existing.source), name, existing.arity,
					displayName(parentReq.source), name, parentReq.arity)
			}
			reqs[name] = parentReq
		}
	}
	return reqs, nil
}

func displayName(key string) string {
	return key
}

func effectiveMethodArity(className string, method string, scope *scope) (int, bool) {
	if info, ok := scope.interfaces[className]; ok {
		if arity, ok := info.methods[method]; ok {
			return arity, true
		}
		if arity, ok := info.defaults[method]; ok {
			return arity, true
		}
		return 0, false
	}
	for className != "" {
		info, ok := scope.classes[className]
		if !ok {
			return 0, false
		}
		if arity, ok := info.methods[method]; ok {
			return arity, true
		}
		className = info.parent
	}
	return 0, false
}

func effectiveMethodParams(className string, method string, scope *scope) ([]string, bool) {
	for className != "" {
		info, ok := scope.classes[className]
		if !ok {
			return nil, false
		}
		if params, ok := info.methodParams[method]; ok {
			return params, true
		}
		className = info.parent
	}
	return nil, false
}

func effectiveClassMethodParams(className string, method string, scope *scope) ([]string, bool) {
	for className != "" {
		info, ok := scope.classes[className]
		if !ok {
			return nil, false
		}
		if params, ok := info.classMethodParams[method]; ok {
			return params, true
		}
		className = info.parent
	}
	return nil, false
}

func effectiveAbstractRequirements(className string, scope *scope) (map[string]int, map[string]int) {
	chain := []classInfo{}
	for className != "" {
		info := scope.classes[className]
		chain = append([]classInfo{info}, chain...)
		className = info.parent
	}
	instanceReqs := map[string]int{}
	classReqs := map[string]int{}
	for _, info := range chain {
		for name, arity := range info.abstractMethods {
			instanceReqs[name] = arity
		}
		for name := range info.methods {
			delete(instanceReqs, name)
		}
		for name, arity := range info.abstractClassMethods {
			classReqs[name] = arity
		}
		for name := range info.classMethods {
			delete(classReqs, name)
		}
	}
	return instanceReqs, classReqs
}

func funcCallsSuper(fn *ast.FuncLit) bool {
	var exprHasSuper func(ast.Expr) bool
	exprHasSuper = func(expr ast.Expr) bool {
		switch n := expr.(type) {
		case *ast.CallExpr:
			if _, ok := n.Callee.(*ast.SuperExpr); ok {
				return true
			}
			if exprHasSuper(n.Callee) {
				return true
			}
			for _, arg := range n.Args {
				if exprHasSuper(arg) {
					return true
				}
			}
		case *ast.BinaryExpr:
			return exprHasSuper(n.Left) || exprHasSuper(n.Right)
		case *ast.UnaryExpr:
			return exprHasSuper(n.Expr)
		case *ast.TryExpr:
			return exprHasSuper(n.Expr)
		case *ast.MemberExpr:
			return exprHasSuper(n.Target)
		case *ast.IndexExpr:
			return exprHasSuper(n.Target) || exprHasSuper(n.Index)
		case *ast.ArrayLit:
			for _, elem := range n.Elems {
				if exprHasSuper(elem) {
					return true
				}
			}
		case *ast.DictLit:
			for _, prop := range n.Props {
				if exprHasSuper(prop.Value) {
					return true
				}
			}
		}
		return false
	}
	if fn.Expr != nil && exprHasSuper(fn.Expr) {
		return true
	}
	for _, stmt := range fn.Body {
		switch n := stmt.(type) {
		case *ast.ExprStmt:
			if exprHasSuper(n.Expr) {
				return true
			}
		case *ast.AssignStmt:
			for _, value := range n.Values {
				if exprHasSuper(value) {
					return true
				}
			}
		case *ast.ReturnStmt:
			for _, value := range n.Values {
				if exprHasSuper(value) {
					return true
				}
			}
		}
	}
	return false
}

func constructorSuperStats(fn *ast.FuncLit) (count int, firstSuper int, firstField int, firstReturn int) {
	var exprLine func(ast.Expr) int
	exprLine = func(expr ast.Expr) int {
		switch n := expr.(type) {
		case *ast.CallExpr:
			if super, ok := n.Callee.(*ast.SuperExpr); ok {
				return super.Tok.Line
			}
			if line := exprLine(n.Callee); line != 0 {
				return line
			}
			for _, arg := range n.Args {
				if line := exprLine(arg); line != 0 {
					return line
				}
			}
		case *ast.InstanceFieldExpr:
			return n.NameTok.Line
		case *ast.BinaryExpr:
			if line := exprLine(n.Left); line != 0 {
				return line
			}
			return exprLine(n.Right)
		case *ast.UnaryExpr:
			return exprLine(n.Expr)
		case *ast.TryExpr:
			return exprLine(n.Expr)
		case *ast.MemberExpr:
			// v0.46/47 G2: `self.field` is an instance-field access
			// for the purposes of constructor-super ordering checks.
			if self, ok := n.Target.(*ast.SelfExpr); ok && !self.Class {
				return n.NameTok.Line
			}
			return exprLine(n.Target)
		case *ast.IndexExpr:
			if line := exprLine(n.Target); line != 0 {
				return line
			}
			return exprLine(n.Index)
		case *ast.ArrayLit:
			for _, elem := range n.Elems {
				if line := exprLine(elem); line != 0 {
					return line
				}
			}
		case *ast.DictLit:
			for _, prop := range n.Props {
				if line := exprLine(prop.Value); line != 0 {
					return line
				}
			}
		}
		return 0
	}
	var countExprSuper func(ast.Expr)
	countExprSuper = func(expr ast.Expr) {
		switch n := expr.(type) {
		case *ast.CallExpr:
			if super, ok := n.Callee.(*ast.SuperExpr); ok {
				count++
				if firstSuper == 0 {
					firstSuper = super.Tok.Line
				}
			}
			countExprSuper(n.Callee)
			for _, arg := range n.Args {
				countExprSuper(arg)
			}
		case *ast.BinaryExpr:
			countExprSuper(n.Left)
			countExprSuper(n.Right)
		case *ast.UnaryExpr:
			countExprSuper(n.Expr)
		case *ast.TryExpr:
			countExprSuper(n.Expr)
		case *ast.MemberExpr:
			countExprSuper(n.Target)
		case *ast.IndexExpr:
			countExprSuper(n.Target)
			countExprSuper(n.Index)
		case *ast.ArrayLit:
			for _, elem := range n.Elems {
				countExprSuper(elem)
			}
		case *ast.DictLit:
			for _, prop := range n.Props {
				countExprSuper(prop.Value)
			}
		}
	}
	observeField := func(line int) {
		if line != 0 && firstField == 0 {
			firstField = line
		}
	}
	if fn.Expr != nil {
		observeField(exprLine(fn.Expr))
		countExprSuper(fn.Expr)
	}
	for _, stmt := range fn.Body {
		switch n := stmt.(type) {
		case *ast.ExprStmt:
			observeField(exprLine(n.Expr))
			countExprSuper(n.Expr)
		case *ast.AssignStmt:
			for _, target := range n.Targets {
				observeField(exprLine(target))
			}
			for _, value := range n.Values {
				observeField(exprLine(value))
				countExprSuper(value)
			}
		case *ast.ReturnStmt:
			if firstReturn == 0 {
				firstReturn = n.Tok.Line
			}
			for _, value := range n.Values {
				observeField(exprLine(value))
				countExprSuper(value)
			}
		}
	}
	return count, firstSuper, firstField, firstReturn
}

func checkAssignmentTarget(target ast.Expr, values []ast.Expr, constants map[string]bool, scope *scope) error {
	switch n := target.(type) {
	case *ast.MemberExpr:
		if n.Name == "class" || n.Name == "class_name" || ((n.Name == "name" || n.Name == "parent") && kindOf(n.Target, scope) == kindClass) {
			return fmt.Errorf("%d:%d: cannot assign to read-only introspection member %s", n.NameTok.Line, n.NameTok.Col, n.Name)
		}
		if self, ok := n.Target.(*ast.SelfExpr); ok && !self.Class {
			if !scope.inInstanceMethod {
				return fmt.Errorf("%d:%d: self is only valid inside a class method or instance method", n.NameTok.Line, n.NameTok.Col)
			}
			if scope.currentClass == "" {
				if scope.interfaceFields != nil && scope.interfaceFields[n.Name] {
					return nil
				}
				return fmt.Errorf("%d:%d: undeclared instance field %s", n.NameTok.Line, n.NameTok.Col, n.Name)
			}
			if scope.interfaceFields != nil && scope.interfaceFields[n.Name] {
				return nil
			}
			if !effectiveInstanceField(n.Name, scope.currentClass, scope) {
				return fmt.Errorf("%d:%d: undeclared instance field %s", n.NameTok.Line, n.NameTok.Col, n.Name)
			}
			if err := checkPrivateInstanceAccess(n.Name, n.NameTok.Line, n.NameTok.Col, scope, true); err != nil {
				return err
			}
			return nil
		}
		if classConstantName(n, scope) != "" {
			return fmt.Errorf("%d:%d: cannot reassign constant %s", n.NameTok.Line, n.NameTok.Col, n.Name)
		}
		if id, ok := n.Target.(*ast.Ident); ok && isPrimitiveClassName(id.Name) {
			return fmt.Errorf("%d:%d: [TYA-E0814] cannot add or redefine method %s on built-in primitive class %s", n.NameTok.Line, n.NameTok.Col, n.Name, id.Name)
		}
		if id, ok := n.Target.(*ast.Ident); ok && constants[id.Name] {
			return fmt.Errorf("%d:%d: cannot mutate constant %s", n.NameTok.Line, n.NameTok.Col, id.Name)
		}
		if err := checkExpr(n.Target, scope); err != nil {
			return err
		}
		return nil
	case *ast.InstanceFieldExpr:
		if !permissiveLegacy {
			return fmt.Errorf("%d:%d: [TYA-E0410] @%s is removed; use self.%s", n.NameTok.Line, n.NameTok.Col, n.Name, n.Name)
		}
		if !scope.inInstanceMethod {
			return fmt.Errorf("%d:%d: @%s is only valid inside an instance method", n.NameTok.Line, n.NameTok.Col, n.Name)
		}
		if !valueNameRE.MatchString(n.Name) {
			return fmt.Errorf("%d:%d: invalid field name %s", n.NameTok.Line, n.NameTok.Col, n.Name)
		}
		if err := checkPrivateInstanceAccess(n.Name, n.NameTok.Line, n.NameTok.Col, scope, true); err != nil {
			return err
		}
	case *ast.ClassVarExpr:
		if !permissiveLegacy {
			return fmt.Errorf("%d:%d: [TYA-E0410] @@%s is removed; use Self.%s", n.NameTok.Line, n.NameTok.Col, n.Name, n.Name)
		}
		if !scope.inClassBody && !scope.inInstanceMethod && !scope.inClassMethod {
			return fmt.Errorf("%d:%d: @@%s is only valid inside a class", n.NameTok.Line, n.NameTok.Col, n.Name)
		}
		if !valueNameRE.MatchString(n.Name) {
			return fmt.Errorf("%d:%d: invalid class variable name %s", n.NameTok.Line, n.NameTok.Col, n.Name)
		}
		if err := checkPrivateClassAccess(n.Name, n.NameTok.Line, n.NameTok.Col, scope); err != nil {
			return err
		}
	case *ast.IndexExpr:
		if id, ok := n.Target.(*ast.Ident); ok && constants[id.Name] {
			return fmt.Errorf("%d:%d: cannot mutate constant %s", id.Tok.Line, id.Tok.Col, id.Name)
		}
		if name := classConstantName(n.Target, scope); name != "" {
			line, col := exprLineCol(n.Target)
			return fmt.Errorf("%d:%d: cannot mutate constant %s", line, col, name)
		}
		if err := checkExpr(n.Target, scope); err != nil {
			return err
		}
		return checkExpr(n.Index, scope)
	case *ast.ArrayLit:
		for _, elem := range n.Elems {
			if err := checkDestructuringTarget(elem, values, constants, scope); err != nil {
				return err
			}
		}
	case *ast.DictLit:
		seen := map[string]bool{}
		for _, prop := range n.Props {
			if prop.Name == "" {
				return fmt.Errorf("%d:%d: destructuring dictionary keys must be string literals", prop.Tok.Line, prop.Tok.Col)
			}
			if seen[prop.Name] {
				return fmt.Errorf("%d:%d: duplicate destructuring dictionary key %s", prop.Tok.Line, prop.Tok.Col, prop.Name)
			}
			seen[prop.Name] = true
			if err := checkDestructuringTarget(prop.Value, values, constants, scope); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkDestructuringTarget(target ast.Expr, values []ast.Expr, constants map[string]bool, scope *scope) error {
	switch n := target.(type) {
	case *ast.Ident:
		if n.Name == "_" {
			return nil
		}
		if err := checkBindingName(n.Name, n.Tok.Line, n.Tok.Col); err != nil {
			return err
		}
		if constants[n.Name] {
			return fmt.Errorf("%d:%d: cannot reassign constant %s", n.Tok.Line, n.Tok.Col, n.Name)
		}
		if constNameRE.MatchString(n.Name) {
			constants[n.Name] = true
		}
		scope.define(n.Name, exprKind(values, scope))
		return nil
	case *ast.ArrayLit, *ast.DictLit:
		return checkAssignmentTarget(target, values, constants, scope)
	default:
		return fmt.Errorf("invalid destructuring target")
	}
}

func checkInterpolation(value string, scope *scope) error {
	for i := 0; i < len(value); {
		switch value[i] {
		case '{':
			if i+1 < len(value) && value[i+1] == '{' {
				i += 2
				continue
			}
			close := interp.FindExprEnd(value, i)
			if close < 0 {
				return fmt.Errorf("unclosed interpolation")
			}
			expr := strings.TrimSpace(value[i+1 : close])
			if expr == "" {
				return fmt.Errorf("empty interpolation")
			}
			if err := checkInterpolationExpr(expr, scope); err != nil {
				return err
			}
			i = close + 1
		case '}':
			if i+1 < len(value) && value[i+1] == '}' {
				i += 2
				continue
			}
			return fmt.Errorf("unmatched '}' in string interpolation")
		default:
			i++
		}
	}
	return nil
}

func checkInterpolationExpr(expr string, scope *scope) error {
	toks, errs := lexer.Lex(expr)
	if len(errs) > 0 {
		return fmt.Errorf("invalid interpolation expression: %w", errs[0])
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		return fmt.Errorf("invalid interpolation expression: %w", err)
	}
	if len(prog.Stmts) != 1 {
		return fmt.Errorf("interpolation must contain one expression")
	}
	stmt, ok := prog.Stmts[0].(*ast.ExprStmt)
	if !ok {
		return fmt.Errorf("interpolation must contain an expression")
	}
	return checkExpr(stmt.Expr, scope)
}

func exprKind(values []ast.Expr, scope *scope) valueKind {
	if len(values) != 1 {
		return kindUnknown
	}
	if call, ok := values[0].(*ast.CallExpr); ok {
		if id, ok := call.Callee.(*ast.Ident); ok && scope.kind(id.Name) == kindClass {
			return kindObject
		}
		if member, ok := call.Callee.(*ast.MemberExpr); ok && kindOf(member.Target, scope) == kindModule && classNameRE.MatchString(member.Name) {
			return kindObject
		}
	}
	return literalKind(values[0])
}

func kindOf(expr ast.Expr, scope *scope) valueKind {
	if id, ok := expr.(*ast.Ident); ok {
		return scope.kind(id.Name)
	}
	if member, ok := expr.(*ast.MemberExpr); ok && kindOf(member.Target, scope) == kindModule {
		if scope.kind(memberKey(member)) == kindClass {
			return kindClass
		}
		if scope.kind(memberKey(member)) == kindInterface {
			return kindInterface
		}
		if scope.kind(aliasedPackageSymbolKey(memberKeyModule(member), member.Name)) == kindInterface {
			return kindInterface
		}
		if scope.kind(member.Name) == kindClass {
			return kindClass
		}
	}
	return literalKind(expr)
}

func memberKey(member *ast.MemberExpr) string {
	parts := []string{member.Name}
	for target := member.Target; ; {
		switch n := target.(type) {
		case *ast.Ident:
			parts = append([]string{n.Name}, parts...)
			return strings.Join(parts, ".")
		case *ast.MemberExpr:
			parts = append([]string{n.Name}, parts...)
			target = n.Target
		default:
			return strings.Join(parts, ".")
		}
	}
}

func memberKeyModule(member *ast.MemberExpr) string {
	if id, ok := member.Target.(*ast.Ident); ok {
		return id.Name
	}
	key := memberKey(member)
	if head, _, ok := strings.Cut(key, "."); ok {
		return head
	}
	return key
}

func literalKind(expr ast.Expr) valueKind {
	switch n := expr.(type) {
	case *ast.BinaryExpr:
		if n.Op.Lexeme != "+" {
			return kindUnknown
		}
		left := literalKind(n.Left)
		right := literalKind(n.Right)
		if left != kindUnknown && left == right && (left == kindString || left == kindBytes || left == kindNumber) {
			return left
		}
		return kindUnknown
	case *ast.ArrayLit:
		return kindArray
	case *ast.DictLit:
		if hasFunctionMember(n) {
			return kindUnknown
		}
		return kindDict
	case *ast.NilLit:
		return kindNil
	case *ast.BoolLit:
		return kindBool
	case *ast.IntLit, *ast.FloatLit:
		return kindNumber
	case *ast.StringLit:
		return kindString
	case *ast.BytesLit:
		return kindBytes
	case *ast.FuncLit:
		return kindFunction
	}
	return kindUnknown
}

func valueKindName(kind valueKind) string {
	switch kind {
	case kindArray:
		return "array"
	case kindDict:
		return "dict"
	case kindModule:
		return "module"
	case kindClass:
		return "class"
	case kindInterface:
		return "interface"
	case kindObject:
		return "object"
	case kindNil:
		return "nil"
	case kindBool:
		return "bool"
	case kindNumber:
		return "number"
	case kindString:
		return "string"
	case kindBytes:
		return "bytes"
	case kindFunction:
		return "function"
	default:
		return "unknown"
	}
}

func hasFunctionMember(dict *ast.DictLit) bool {
	for _, prop := range dict.Props {
		if _, ok := prop.Value.(*ast.FuncLit); ok {
			return true
		}
	}
	return false
}

func isFloatLiteral(expr ast.Expr) bool {
	_, ok := expr.(*ast.FloatLit)
	return ok
}

func isStringLiteral(expr ast.Expr) bool {
	_, ok := expr.(*ast.StringLit)
	return ok
}

func isNegativeNumberLiteral(expr ast.Expr) bool {
	unary, ok := expr.(*ast.UnaryExpr)
	if !ok || unary.Op.Type != token.MINUS {
		return false
	}
	switch unary.Expr.(type) {
	case *ast.IntLit, *ast.FloatLit:
		return true
	default:
		return false
	}
}

func isDictMethodName(name string) bool {
	switch name {
	case "class", "len", "empty?", "has", "has?", "get", "set", "delete", "keys", "values", "entries", "merge", "merge!", "to_s", "equal?", "iter", "sequence":
		return true
	default:
		return false
	}
}

func memberAccessError(expr *ast.MemberExpr, receiver string) error {
	line := expr.NameTok.Line
	col := expr.NameTok.Col
	if receiver == "dictionary" {
		if line > 0 {
			return fmt.Errorf("%d:%d: cannot use . access on dictionary; use index access", line, col)
		}
		return fmt.Errorf("cannot use . access on dictionary; use index access")
	}
	if receiver == "non-module value" {
		if line > 0 {
			return fmt.Errorf("%d:%d: cannot use . access on non-module value", line, col)
		}
		return fmt.Errorf("cannot use . access on non-module value")
	}
	if line > 0 {
		return fmt.Errorf("%d:%d: cannot use . access on %s", line, col, receiver)
	}
	return fmt.Errorf("cannot use . access on %s", receiver)
}

func checkBindingName(name string, line, col int) error {
	if isPrimitiveClassName(name) {
		return fmt.Errorf("%d:%d: [TYA-E0815] cannot rebind reserved class identifier %s", line, col, name)
	}
	if constNameRE.MatchString(name) || valueNameRE.MatchString(name) {
		return nil
	}
	if line > 0 {
		return fmt.Errorf("%d:%d: invalid binding name %s", line, col, name)
	}
	return fmt.Errorf("invalid binding name %s", name)
}
