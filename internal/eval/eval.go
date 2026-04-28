package eval

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"tya/internal/ast"
	"tya/internal/lexer"
	"tya/internal/parser"
)

type Value any

type Object map[string]Value

type Function struct {
	Params []string
	Body   []ast.Stmt
	Expr   ast.Expr
	Env    *Env
}

type Builtin func([]Value) (Value, error)

type Env struct {
	parent *Env
	vars   map[string]Value
	this   Object
}

func NewEnv() *Env {
	return &Env{vars: map[string]Value{}}
}

func (e *Env) child(this Object) *Env {
	return &Env{parent: e, vars: map[string]Value{}, this: this}
}

func (e *Env) get(name string) (Value, bool) {
	if v, ok := e.vars[name]; ok {
		return v, true
	}
	if e.parent != nil {
		return e.parent.get(name)
	}
	return nil, false
}

func (e *Env) set(name string, v Value) {
	if _, ok := e.vars[name]; ok || e.parent == nil {
		e.vars[name] = v
		return
	}
	if _, ok := e.parent.get(name); ok {
		e.parent.assign(name, v)
		return
	}
	e.vars[name] = v
}

func (e *Env) assign(name string, v Value) bool {
	if _, ok := e.vars[name]; ok {
		e.vars[name] = v
		return true
	}
	if e.parent != nil {
		return e.parent.assign(name, v)
	}
	return false
}

func Run(prog *ast.Program, out io.Writer) error {
	env := NewEnv()
	env.set("print", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("print expects 1 argument")
		}
		fmt.Fprintln(out, stringify(args[0]))
		return nil, nil
	}))
	_, err := evalStmts(prog.Stmts, env)
	return err
}

func evalStmts(stmts []ast.Stmt, env *Env) (Value, error) {
	var last Value
	for _, s := range stmts {
		v, err := evalStmt(s, env)
		if err != nil {
			return nil, err
		}
		last = v
	}
	return last, nil
}

func evalStmt(s ast.Stmt, env *Env) (Value, error) {
	switch n := s.(type) {
	case *ast.AssignStmt:
		v, err := evalExpr(n.Value, env)
		if err != nil {
			return nil, err
		}
		return v, assign(n.Target, v, env)
	case *ast.ExprStmt:
		return evalExpr(n.Expr, env)
	case *ast.IfStmt:
		cond, err := evalExpr(n.Cond, env)
		if err != nil {
			return nil, err
		}
		if truthy(cond) {
			return evalStmts(n.Then, env)
		}
		return evalStmts(n.Else, env)
	}
	return nil, fmt.Errorf("unknown statement")
}

func assign(target ast.Expr, v Value, env *Env) error {
	switch t := target.(type) {
	case *ast.Ident:
		env.set(t.Name, v)
		return nil
	case *ast.ThisProp:
		if env.this == nil {
			return fmt.Errorf("@%s used outside method", t.Name)
		}
		env.this[t.Name] = v
		return nil
	case *ast.MemberExpr:
		obj, err := evalExpr(t.Object, env)
		if err != nil {
			return err
		}
		o, ok := obj.(Object)
		if !ok {
			return fmt.Errorf("cannot assign property on non-object")
		}
		o[t.Name] = v
		return nil
	}
	return fmt.Errorf("invalid assignment target")
}

func evalExpr(e ast.Expr, env *Env) (Value, error) {
	switch n := e.(type) {
	case *ast.Ident:
		v, ok := env.get(n.Name)
		if !ok {
			return nil, fmt.Errorf("%d:%d: undefined variable %s", n.Tok.Line, n.Tok.Col, n.Name)
		}
		return v, nil
	case *ast.ThisProp:
		if env.this == nil {
			return nil, fmt.Errorf("@%s used outside method", n.Name)
		}
		return env.this[n.Name], nil
	case *ast.IntLit:
		return n.Value, nil
	case *ast.FloatLit:
		return n.Value, nil
	case *ast.StringLit:
		return interpolate(n.Value, env)
	case *ast.BoolLit:
		return n.Value, nil
	case *ast.NilLit:
		return nil, nil
	case *ast.ObjectLit:
		o := Object{}
		for _, p := range n.Props {
			v, err := evalExpr(p.Value, env)
			if err != nil {
				return nil, err
			}
			o[p.Name] = v
		}
		return o, nil
	case *ast.FuncLit:
		return &Function{Params: n.Params, Body: n.Body, Expr: n.Expr, Env: env}, nil
	case *ast.BinaryExpr:
		return evalBinary(n, env)
	case *ast.UnaryExpr:
		v, err := evalExpr(n.Expr, env)
		if err != nil {
			return nil, err
		}
		if n.Op.Lexeme == "not" {
			return !truthy(v), nil
		}
		return nil, fmt.Errorf("unknown unary operator %s", n.Op.Lexeme)
	case *ast.MemberExpr:
		obj, err := evalExpr(n.Object, env)
		if err != nil {
			return nil, err
		}
		o, ok := obj.(Object)
		if !ok {
			return nil, fmt.Errorf("cannot read property %s on non-object", n.Name)
		}
		return o[n.Name], nil
	case *ast.CallExpr:
		return evalCall(n, env)
	}
	return nil, fmt.Errorf("unknown expression")
}

func evalCall(c *ast.CallExpr, env *Env) (Value, error) {
	fnVal, recv, err := evalCallee(c.Callee, env)
	if err != nil {
		return nil, err
	}
	args := make([]Value, 0, len(c.Args))
	for _, a := range c.Args {
		v, err := evalExpr(a, env)
		if err != nil {
			return nil, err
		}
		args = append(args, v)
	}
	switch fn := fnVal.(type) {
	case Builtin:
		return fn(args)
	case *Function:
		if len(args) != len(fn.Params) {
			return nil, fmt.Errorf("function expects %d arguments, got %d", len(fn.Params), len(args))
		}
		callEnv := fn.Env.child(recv)
		for i, name := range fn.Params {
			callEnv.set(name, args[i])
		}
		if fn.Expr != nil {
			return evalExpr(fn.Expr, callEnv)
		}
		return evalStmts(fn.Body, callEnv)
	}
	return nil, fmt.Errorf("value is not callable")
}

func evalCallee(e ast.Expr, env *Env) (Value, Object, error) {
	if m, ok := e.(*ast.MemberExpr); ok {
		obj, err := evalExpr(m.Object, env)
		if err != nil {
			return nil, nil, err
		}
		o, ok := obj.(Object)
		if !ok {
			return nil, nil, fmt.Errorf("method receiver is not object")
		}
		return o[m.Name], o, nil
	}
	v, err := evalExpr(e, env)
	return v, nil, err
}

func evalBinary(b *ast.BinaryExpr, env *Env) (Value, error) {
	l, err := evalExpr(b.Left, env)
	if err != nil {
		return nil, err
	}
	if b.Op.Lexeme == "and" {
		if !truthy(l) {
			return l, nil
		}
		return evalExpr(b.Right, env)
	}
	if b.Op.Lexeme == "or" {
		if truthy(l) {
			return l, nil
		}
		return evalExpr(b.Right, env)
	}
	r, err := evalExpr(b.Right, env)
	if err != nil {
		return nil, err
	}
	switch b.Op.Lexeme {
	case "==":
		return equal(l, r), nil
	case "!=":
		return !equal(l, r), nil
	case "<", "<=", ">", ">=":
		return compare(b.Op.Lexeme, l, r)
	}
	if b.Op.Lexeme != "+" {
		return evalNumeric(b.Op.Lexeme, l, r)
	}
	if _, ok := l.(string); ok {
		return stringify(l) + stringify(r), nil
	}
	if _, ok := r.(string); ok {
		return stringify(l) + stringify(r), nil
	}
	lf, lok := asFloat(l)
	rf, rok := asFloat(r)
	if !lok || !rok {
		return nil, fmt.Errorf("+ expects numbers or strings")
	}
	if li, ok := l.(int64); ok {
		if ri, ok := r.(int64); ok {
			return li + ri, nil
		}
	}
	return lf + rf, nil
}

func equal(l, r Value) bool {
	switch lv := l.(type) {
	case nil:
		return r == nil
	case bool:
		rv, ok := r.(bool)
		return ok && lv == rv
	case int64:
		switch rv := r.(type) {
		case int64:
			return lv == rv
		case float64:
			return float64(lv) == rv
		}
	case float64:
		switch rv := r.(type) {
		case int64:
			return lv == float64(rv)
		case float64:
			return lv == rv
		}
	case string:
		rv, ok := r.(string)
		return ok && lv == rv
	case Object:
		rv, ok := r.(Object)
		return ok && fmt.Sprintf("%p", lv) == fmt.Sprintf("%p", rv)
	case *Function:
		return lv == r
	}
	return false
}

func compare(op string, l, r Value) (Value, error) {
	lf, lok := asFloat(l)
	rf, rok := asFloat(r)
	if !lok || !rok {
		return nil, fmt.Errorf("%s expects numbers", op)
	}
	switch op {
	case "<":
		return lf < rf, nil
	case "<=":
		return lf <= rf, nil
	case ">":
		return lf > rf, nil
	case ">=":
		return lf >= rf, nil
	}
	return nil, fmt.Errorf("unknown operator %s", op)
}

func evalNumeric(op string, l, r Value) (Value, error) {
	lf, lok := asFloat(l)
	rf, rok := asFloat(r)
	if !lok || !rok {
		return nil, fmt.Errorf("%s expects numbers", op)
	}
	if op == "/" {
		if rf == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return lf / rf, nil
	}
	li, lint := l.(int64)
	ri, rint := r.(int64)
	if lint && rint {
		switch op {
		case "-":
			return li - ri, nil
		case "*":
			return li * ri, nil
		case "%":
			if ri == 0 {
				return nil, fmt.Errorf("modulo by zero")
			}
			return li % ri, nil
		}
	}
	switch op {
	case "-":
		return lf - rf, nil
	case "*":
		return lf * rf, nil
	case "%":
		return nil, fmt.Errorf("%% expects integers")
	}
	return nil, fmt.Errorf("unknown operator %s", op)
}

func asFloat(v Value) (float64, bool) {
	switch n := v.(type) {
	case int64:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}

func truthy(v Value) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return true
}

var interp = regexp.MustCompile(`\{([^{}]+)\}`)

func interpolate(s string, env *Env) (string, error) {
	var first error
	out := interp.ReplaceAllStringFunc(s, func(m string) string {
		if first != nil {
			return ""
		}
		expr := strings.TrimSpace(m[1 : len(m)-1])
		toks, errs := lexer.Lex(expr)
		if len(errs) > 0 {
			first = errs[0]
			return ""
		}
		prog, err := parser.Parse(toks)
		if err != nil {
			first = err
			return ""
		}
		if len(prog.Stmts) != 1 {
			first = fmt.Errorf("interpolation must contain one expression")
			return ""
		}
		stmt, ok := prog.Stmts[0].(*ast.ExprStmt)
		if !ok {
			first = fmt.Errorf("interpolation must contain an expression")
			return ""
		}
		v, err := evalExpr(stmt.Expr, env)
		if err != nil {
			first = err
			return ""
		}
		return stringify(v)
	})
	return out, first
}

func stringify(v Value) string {
	switch x := v.(type) {
	case nil:
		return "nil"
	case string:
		return x
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		if x {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", x)
	}
}
