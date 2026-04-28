package ast

import "tya/internal/token"

type Program struct {
	Stmts []Stmt
}

type Stmt interface{ stmt() }
type Expr interface{ expr() }

type AssignStmt struct {
	Target Expr
	Value  Expr
	Tok    token.Token
}

func (*AssignStmt) stmt() {}

type ExprStmt struct {
	Expr Expr
}

func (*ExprStmt) stmt() {}

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

type ObjectLit struct {
	Props []ObjectProp
}

func (*ObjectLit) expr() {}

type ObjectProp struct {
	Name  string
	Value Expr
}

type FuncLit struct {
	Params []string
	Body   []Stmt
	Expr   Expr
}

func (*FuncLit) expr() {}

type BinaryExpr struct {
	Left  Expr
	Op    token.Token
	Right Expr
}

func (*BinaryExpr) expr() {}

type MemberExpr struct {
	Object Expr
	Name   string
}

func (*MemberExpr) expr() {}

type CallExpr struct {
	Callee Expr
	Args   []Expr
}

func (*CallExpr) expr() {}
