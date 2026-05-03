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

type Ident struct {
	Name string
	Tok  token.Token
}

func (*Ident) expr() {}

type ThisProp struct {
	Name string
	Tok  token.Token
}

func (*ThisProp) expr() {}

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

type ObjectLit = DictLit

type ArrayLit struct {
	Elems []Expr
}

func (*ArrayLit) expr() {}

type SetLit struct {
	Elems []Expr
}

func (*SetLit) expr() {}

type DictProp struct {
	Name  string
	Tok   token.Token
	Value Expr
}

type ObjectProp = DictProp

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
	Object Expr
	Name   string
}

func (*MemberExpr) expr() {}

type IndexExpr struct {
	Object Expr
	Index  Expr
}

func (*IndexExpr) expr() {}

type CallExpr struct {
	Callee Expr
	Args   []Expr
}

func (*CallExpr) expr() {}
