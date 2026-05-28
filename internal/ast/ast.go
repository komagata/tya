package ast

import (
	"strings"

	"tya/internal/token"
)

type Program struct {
	Stmts []Stmt

	// HeaderComments holds the file-header comment block introduced
	// in v0.34. Per docs/SPEC.md Formatted Syntax, these are `#` lines
	// at the top of the file separated from the body by exactly one
	// blank line. The slice contains the comment texts in source
	// order, without the leading `#`.
	HeaderComments []string

	// Comments associates each Stmt with its leading and line-end
	// comments per docs/SPEC.md Formatted Syntax. The map is
	// introduced in v0.35; it is populated by ParseWithComments and
	// is nil from the default Parse path. Keyed by Stmt pointer.
	Comments map[Stmt]StmtComments
}

// StmtComments are the comments attached to a single statement.
// Leading is the block of `#` lines immediately before the statement
// at the same indentation, in source order. LineEnd is the optional
// trailing comment on the same line as the statement's start.
type StmtComments struct {
	Leading []string
	LineEnd string
}

type Stmt interface{ stmt() }
type Expr interface{ expr() }

type AssignStmt struct {
	Targets []Expr
	Values  []Expr
	Tok     token.Token
}

func (*AssignStmt) stmt() {}

type ExprStmt struct {
	Expr Expr
}

func (*ExprStmt) stmt() {}

type ImportStmt struct {
	Name     string
	NameTok  token.Token
	Alias    string
	AliasTok token.Token
	Wildcard bool
}

func (*ImportStmt) stmt() {}

type ImportBlockStmt struct {
	Imports []*ImportStmt
	Tok     token.Token
}

func (*ImportBlockStmt) stmt() {}

func (s *ImportStmt) ModuleName() string {
	if i := strings.LastIndex(s.Name, "/"); i >= 0 {
		return s.Name[i+1:]
	}
	return s.Name
}

func (s *ImportStmt) BindingName() string {
	if s.Alias != "" {
		return s.Alias
	}
	return s.ModuleName()
}

// EmbedStmt declares a top-level binding whose value is the
// contents of one or more files read at build time. Single-file
// paths produce a bytes value; paths containing `*` or `**`
// produce a dict whose keys are paths relative to the source
// file's directory and whose values are bytes. See
// docs/v0.57/SPEC.md.
type EmbedStmt struct {
	Path       string // raw pattern, "/" separated
	PathTok    token.Token
	Name       string // binding name
	NameTok    token.Token
	Transforms map[string]bool
}

func (*EmbedStmt) stmt() {}

type IfStmt struct {
	Cond Expr
	Then []Stmt
	Else []Stmt
}

func (*IfStmt) stmt() {}
func (*IfStmt) expr() {}

type WhileStmt struct {
	Cond Expr
	Body []Stmt
}

func (*WhileStmt) stmt() {}
func (*WhileStmt) expr() {}

type ForInStmt struct {
	ValueName string
	IndexName string
	ValueTok  token.Token
	IndexTok  token.Token
	Kind      string
	Iterable  Expr
	Body      []Stmt
}

func (*ForInStmt) stmt() {}
func (*ForInStmt) expr() {}

type BreakStmt struct{ Tok token.Token }

func (*BreakStmt) stmt() {}

type ContinueStmt struct{ Tok token.Token }

func (*ContinueStmt) stmt() {}

type ReturnStmt struct {
	Values []Expr
	Tok    token.Token
}

func (*ReturnStmt) stmt() {}

type RaiseStmt struct {
	Value Expr
	Tok   token.Token
}

func (*RaiseStmt) stmt() {}

type TryCatchStmt struct {
	Try       []Stmt
	CatchName string
	CatchTok  token.Token
	Catch     []Stmt
	Finally   []Stmt
	Tok       token.Token
}

func (*TryCatchStmt) stmt() {}

type MatchStmt struct {
	Value Expr
	Cases []MatchCase
	Tok   token.Token
}

func (*MatchStmt) stmt() {}
func (*MatchStmt) expr() {}

type MatchCase struct {
	Pattern Expr
	Tok     token.Token
	Body    []Stmt
}

type ModuleDecl struct {
	Name       string
	NameTok    token.Token
	Members    []ModuleMember
	Classes    []*ClassDecl
	Interfaces []*InterfaceDecl
}

func (*ModuleDecl) stmt() {}

type ModuleMember struct {
	Name  string
	Tok   token.Token
	Value Expr
}

type ClassDecl struct {
	Name       string
	NameTok    token.Token
	Parent     *ClassRef
	Implements []ClassRef
	Abstract   bool
	Final      bool
	Fields     []ClassField
	Constants  []ClassConst
	Vars       []ClassVar
	Methods    []ClassMethod
	// OriginFile is the source file (relative to the package root)
	// that this class was parsed from. Stamped by the runner after
	// per-file parsing inside a directory package; empty for classes
	// declared in a single-file script. Used by the checker to enforce
	// v0.45 cross-file private class visibility ([TYA-E0406]).
	OriginFile string
}

func (*ClassDecl) stmt() {}

type InterfaceDecl struct {
	Name    string
	NameTok token.Token
	Parents []ClassRef
	Methods []InterfaceMethod
	Fields  []ClassField
}

func (*InterfaceDecl) stmt() {}

type InterfaceMethod struct {
	Name      string
	Tok       token.Token
	Params    []string
	ParamToks []token.Token
	Func      *FuncLit
	Comments  StmtComments
}

type ClassField struct {
	Name     string
	Tok      token.Token
	Value    Expr
	Comments StmtComments
	// v0.46 G1: explicit visibility marker. true when the field was
	// declared with `private` keyword. The legacy `_`-prefix
	// convention writes the same flag during parsing (transitional
	// dual recognition; the checker reads only this flag).
	Private bool
}

type ClassVar struct {
	Name     string
	Tok      token.Token
	Value    Expr
	Private  bool // v0.46 G1; see ClassField.Private.
	Comments StmtComments
}

type ClassConst struct {
	Name     string
	Tok      token.Token
	Value    Expr
	Private  bool // v0.46 G1; see ClassField.Private.
	Comments StmtComments
}

type ClassMethod struct {
	Name     string
	Tok      token.Token
	Func     *FuncLit
	Class    bool
	Abstract bool
	Override bool
	Private  bool // v0.46 G1; see ClassField.Private.
	Comments StmtComments
}

type ClassRef struct {
	Module string
	Name   string
	Tok    token.Token
}

type Ident struct {
	Name string
	Tok  token.Token
}

func (*Ident) expr() {}

type IntLit struct{ Value int64 }

func (*IntLit) expr() {}

type FloatLit struct{ Value float64 }

func (*FloatLit) expr() {}

type StringLit struct {
	Value  string
	Form   string
	Lang   string
	Marker string
}

func (*StringLit) expr() {}

type BytesLit struct {
	Value  string
	Form   string
	Marker string
}

func (*BytesLit) expr() {}

type BoolLit struct{ Value bool }

func (*BoolLit) expr() {}

type NilLit struct{}

func (*NilLit) expr() {}

type DictLit struct {
	Props []DictProp
}

func (*DictLit) expr() {}

type ArrayLit struct {
	Elems []Expr
}

func (*ArrayLit) expr() {}

type DictProp struct {
	Name  string
	Tok   token.Token
	Value Expr
}

type FuncLit struct {
	Params    []string
	ParamToks []token.Token
	Defaults  []Expr
	Body      []Stmt
	Expr      Expr
}

func (*FuncLit) expr() {}

type BinaryExpr struct {
	Left  Expr
	Op    token.Token
	Right Expr
}

func (*BinaryExpr) expr() {}

type UnaryExpr struct {
	Op   token.Token
	Expr Expr
}

func (*UnaryExpr) expr() {}

type TryExpr struct {
	Expr Expr
	Tok  token.Token
}

func (*TryExpr) expr() {}

type SuperExpr struct {
	Tok token.Token
}

func (*SuperExpr) expr() {}

type SelfExpr struct {
	Tok token.Token
	// v0.46 G2: Class is true for the `Self` (capital S) form, which
	// always denotes the lexically enclosing class (statically
	// resolved). Class is false for the `self` (lowercase) form —
	// in v0.45 / v0.46 transition this means "the class object" in
	// class methods (legacy semantics retained until the final
	// clean-cut flips it to "the instance" in instance methods).
	Class bool
}

func (*SelfExpr) expr() {}

type MemberExpr struct {
	Target  Expr
	Name    string
	NameTok token.Token
}

func (*MemberExpr) expr() {}

type InstanceFieldExpr struct {
	Name    string
	NameTok token.Token
}

func (*InstanceFieldExpr) expr() {}

type ClassVarExpr struct {
	Name    string
	NameTok token.Token
}

func (*ClassVarExpr) expr() {}

type IndexExpr struct {
	Target Expr
	Index  Expr
}

func (*IndexExpr) expr() {}

type CallExpr struct {
	Callee        Expr
	Args          []Expr
	CallArgs      []CallArg
	ImplicitSelf  bool
	ImplicitClass bool
}

func (*CallExpr) expr() {}

type CallArg struct {
	Name    string
	NameTok token.Token
	Value   Expr
	Expand  bool
}

func (c *CallExpr) EffectiveArgs() []CallArg {
	if len(c.CallArgs) > 0 {
		return c.CallArgs
	}
	args := make([]CallArg, 0, len(c.Args))
	for _, arg := range c.Args {
		args = append(args, CallArg{Value: arg})
	}
	return args
}

func (c *CallExpr) PositionalArgsOnly() bool {
	for _, arg := range c.CallArgs {
		if arg.Name != "" || arg.Expand {
			return false
		}
	}
	return true
}

// v0.42 Tya Concurrency

// SpawnExpr starts a concurrent task. Its operand is a callable
// expression that the runtime will invoke in a fresh task. The
// expression evaluates to a task value.
type SpawnExpr struct {
	Callee Expr
	Tok    token.Token
}

func (*SpawnExpr) expr() {}

// AwaitExpr blocks the current task until the operand task completes
// and yields its return value (or re-raises a propagated raise).
type AwaitExpr struct {
	Target Expr
	Tok    token.Token
}

func (*AwaitExpr) expr() {}

// ScopeBlock is a structured-concurrency block. Tasks spawned inside
// the block are guaranteed to complete before control leaves the
// block; if any task raises, sibling tasks are cancelled and the raise
// propagates out of the scope.
type ScopeBlock struct {
	Body []Stmt
	Tok  token.Token
}

func (*ScopeBlock) stmt() {}

type SelectStmt struct {
	Arms []SelectArm
	Tok  token.Token
}

func (*SelectStmt) stmt() {}

type SelectArm struct {
	Kind     string
	BindName string
	BindTok  token.Token
	Channel  Expr
	Value    Expr
	Seconds  Expr
	Body     []Stmt
	Tok      token.Token
}
