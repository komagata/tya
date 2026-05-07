package ast

import "tya/internal/token"

type Program struct {
	Stmts []Stmt
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
	Name    string
	NameTok token.Token
}

func (*ImportStmt) stmt() {}

type IfStmt struct {
	Cond Expr
	Then []Stmt
	Else []Stmt
}

func (*IfStmt) stmt() {}

type WhileStmt struct {
	Cond Expr
	Body []Stmt
}

func (*WhileStmt) stmt() {}

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

type BreakStmt struct{}

func (*BreakStmt) stmt() {}

type ContinueStmt struct{}

func (*ContinueStmt) stmt() {}

type ReturnStmt struct {
	Values []Expr
}

func (*ReturnStmt) stmt() {}

type ModuleDecl struct {
	Name    string
	NameTok token.Token
	Members []ModuleMember
}

func (*ModuleDecl) stmt() {}

type ModuleMember struct {
	Name  string
	Tok   token.Token
	Value Expr
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

type StringLit struct{ Value string }

func (*StringLit) expr() {}

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

type MemberExpr struct {
	Target  Expr
	Name    string
	NameTok token.Token
}

func (*MemberExpr) expr() {}

type IndexExpr struct {
	Target Expr
	Index  Expr
}

func (*IndexExpr) expr() {}

type CallExpr struct {
	Callee Expr
	Args   []Expr
}

func (*CallExpr) expr() {}
