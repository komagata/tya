package eval

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"tya/internal/ast"
	"tya/internal/lexer"
	"tya/internal/parser"
)

var (
	errBreak    = errors.New("break")
	errContinue = errors.New("continue")
)

type returnSignal struct {
	value Value
}

func (r *returnSignal) Error() string { return "return" }

type Value any

type Object map[string]Value
type Array struct {
	items []Value
}

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
	installBuiltins(env, out)
	_, err := evalStmts(prog.Stmts, env)
	if errors.Is(err, errBreak) {
		return fmt.Errorf("break used outside loop")
	}
	if errors.Is(err, errContinue) {
		return fmt.Errorf("continue used outside loop")
	}
	var ret *returnSignal
	if errors.As(err, &ret) {
		return fmt.Errorf("return used outside function")
	}
	return err
}

func installBuiltins(env *Env, out io.Writer) {
	env.set("print", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("print expects 1 argument")
		}
		fmt.Fprintln(out, stringify(args[0]))
		return nil, nil
	}))
	env.set("len", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("len expects 1 argument")
		}
		switch v := args[0].(type) {
		case string:
			return int64(len([]rune(v))), nil
		case *Array:
			return int64(len(v.items)), nil
		case Object:
			return int64(len(v)), nil
		default:
			return nil, fmt.Errorf("len expects string, array, or object")
		}
	}))
	env.set("push", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("push expects 2 arguments")
		}
		arr, ok := args[0].(*Array)
		if !ok {
			return nil, fmt.Errorf("push expects array")
		}
		arr.items = append(arr.items, args[1])
		return arr, nil
	}))
	env.set("pop", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("pop expects 1 argument")
		}
		arr, ok := args[0].(*Array)
		if !ok {
			return nil, fmt.Errorf("pop expects array")
		}
		if len(arr.items) == 0 {
			return nil, nil
		}
		last := arr.items[len(arr.items)-1]
		arr.items = arr.items[:len(arr.items)-1]
		return last, nil
	}))
	env.set("keys", Builtin(func(args []Value) (Value, error) {
		obj, err := oneObject("keys", args)
		if err != nil {
			return nil, err
		}
		arr := &Array{}
		for key := range obj {
			arr.items = append(arr.items, key)
		}
		return arr, nil
	}))
	env.set("values", Builtin(func(args []Value) (Value, error) {
		obj, err := oneObject("values", args)
		if err != nil {
			return nil, err
		}
		arr := &Array{}
		for _, value := range obj {
			arr.items = append(arr.items, value)
		}
		return arr, nil
	}))
	env.set("has", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("has expects 2 arguments")
		}
		obj, ok := args[0].(Object)
		if !ok {
			return nil, fmt.Errorf("has expects object")
		}
		key, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("has expects string key")
		}
		_, exists := obj[key]
		return exists, nil
	}))
	env.set("delete", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("delete expects 2 arguments")
		}
		obj, ok := args[0].(Object)
		if !ok {
			return nil, fmt.Errorf("delete expects object")
		}
		key, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("delete expects string key")
		}
		delete(obj, key)
		return nil, nil
	}))
	env.set("toString", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("toString expects 1 argument")
		}
		return stringify(args[0]), nil
	}))
	env.set("split", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("split expects 2 arguments")
		}
		text, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("split expects string text")
		}
		sep, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("split expects string separator")
		}
		parts := strings.Split(text, sep)
		arr := &Array{items: make([]Value, 0, len(parts))}
		for _, part := range parts {
			arr.items = append(arr.items, part)
		}
		return arr, nil
	}))
	env.set("join", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("join expects 2 arguments")
		}
		arr, ok := args[0].(*Array)
		if !ok {
			return nil, fmt.Errorf("join expects array")
		}
		sep, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("join expects string separator")
		}
		parts := make([]string, 0, len(arr.items))
		for _, item := range arr.items {
			parts = append(parts, stringify(item))
		}
		return strings.Join(parts, sep), nil
	}))
	env.set("trim", Builtin(func(args []Value) (Value, error) {
		text, err := oneString("trim", args)
		if err != nil {
			return nil, err
		}
		return strings.TrimSpace(text), nil
	}))
	env.set("replace", Builtin(func(args []Value) (Value, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("replace expects 3 arguments")
		}
		text, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("replace expects string text")
		}
		old, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("replace expects string old value")
		}
		newValue, ok := args[2].(string)
		if !ok {
			return nil, fmt.Errorf("replace expects string new value")
		}
		return strings.ReplaceAll(text, old, newValue), nil
	}))
	env.set("contains", Builtin(func(args []Value) (Value, error) {
		text, part, err := twoStrings("contains", args)
		if err != nil {
			return nil, err
		}
		return strings.Contains(text, part), nil
	}))
	env.set("startsWith", Builtin(func(args []Value) (Value, error) {
		text, prefix, err := twoStrings("startsWith", args)
		if err != nil {
			return nil, err
		}
		return strings.HasPrefix(text, prefix), nil
	}))
	env.set("endsWith", Builtin(func(args []Value) (Value, error) {
		text, suffix, err := twoStrings("endsWith", args)
		if err != nil {
			return nil, err
		}
		return strings.HasSuffix(text, suffix), nil
	}))
	env.set("byteLen", Builtin(func(args []Value) (Value, error) {
		text, err := oneString("byteLen", args)
		if err != nil {
			return nil, err
		}
		return int64(len(text)), nil
	}))
	env.set("charLen", Builtin(func(args []Value) (Value, error) {
		text, err := oneString("charLen", args)
		if err != nil {
			return nil, err
		}
		return int64(len([]rune(text))), nil
	}))
	env.set("toInt", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("toInt expects 1 argument")
		}
		return toInt(args[0])
	}))
	env.set("toFloat", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("toFloat expects 1 argument")
		}
		return toFloat(args[0])
	}))
	env.set("toNumber", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("toNumber expects 1 argument")
		}
		if i, err := toInt(args[0]); err == nil {
			return i, nil
		}
		return toFloat(args[0])
	}))
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
	case *ast.WhileStmt:
		var last Value
		for {
			cond, err := evalExpr(n.Cond, env)
			if err != nil {
				return nil, err
			}
			if !truthy(cond) {
				return last, nil
			}
			last, err = evalStmts(n.Body, env)
			if errors.Is(err, errBreak) {
				return last, nil
			}
			if errors.Is(err, errContinue) {
				continue
			}
			if err != nil {
				return nil, err
			}
		}
	case *ast.ForInStmt:
		iterable, err := evalExpr(n.Iterable, env)
		if err != nil {
			return nil, err
		}
		arr, ok := iterable.(*Array)
		if !ok {
			return nil, fmt.Errorf("for in expects array")
		}
		var last Value
		for i, item := range arr.items {
			env.set(n.ValueName, item)
			if n.IndexName != "" {
				env.set(n.IndexName, int64(i))
			}
			last, err = evalStmts(n.Body, env)
			if errors.Is(err, errBreak) {
				return last, nil
			}
			if errors.Is(err, errContinue) {
				continue
			}
			if err != nil {
				return nil, err
			}
		}
		return last, nil
	case *ast.BreakStmt:
		return nil, errBreak
	case *ast.ContinueStmt:
		return nil, errContinue
	case *ast.ReturnStmt:
		if n.Value == nil {
			return nil, &returnSignal{}
		}
		v, err := evalExpr(n.Value, env)
		if err != nil {
			return nil, err
		}
		return nil, &returnSignal{value: v}
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
	case *ast.ArrayLit:
		arr := &Array{}
		for _, elem := range n.Elems {
			v, err := evalExpr(elem, env)
			if err != nil {
				return nil, err
			}
			arr.items = append(arr.items, v)
		}
		return arr, nil
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
	case *ast.IndexExpr:
		obj, err := evalExpr(n.Object, env)
		if err != nil {
			return nil, err
		}
		idx, err := evalExpr(n.Index, env)
		if err != nil {
			return nil, err
		}
		arr, ok := obj.(*Array)
		if !ok {
			return nil, fmt.Errorf("index target is not array")
		}
		i, ok := idx.(int64)
		if !ok {
			return nil, fmt.Errorf("array index must be int")
		}
		if i < 0 || i >= int64(len(arr.items)) {
			return nil, nil
		}
		return arr.items[i], nil
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
		v, err := evalStmts(fn.Body, callEnv)
		var ret *returnSignal
		if errors.As(err, &ret) {
			return ret.value, nil
		}
		return v, err
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
	case *Array:
		return lv == r
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

func toInt(v Value) (Value, error) {
	switch x := v.(type) {
	case int64:
		return x, nil
	case float64:
		return int64(x), nil
	case string:
		i, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to int", x)
		}
		return i, nil
	default:
		return nil, fmt.Errorf("cannot convert %s to int", stringify(v))
	}
}

func toFloat(v Value) (Value, error) {
	switch x := v.(type) {
	case int64:
		return float64(x), nil
	case float64:
		return x, nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to float", x)
		}
		return f, nil
	default:
		return nil, fmt.Errorf("cannot convert %s to float", stringify(v))
	}
}

func oneString(name string, args []Value) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("%s expects 1 argument", name)
	}
	text, ok := args[0].(string)
	if !ok {
		return "", fmt.Errorf("%s expects string", name)
	}
	return text, nil
}

func twoStrings(name string, args []Value) (string, string, error) {
	if len(args) != 2 {
		return "", "", fmt.Errorf("%s expects 2 arguments", name)
	}
	first, ok := args[0].(string)
	if !ok {
		return "", "", fmt.Errorf("%s expects string first argument", name)
	}
	second, ok := args[1].(string)
	if !ok {
		return "", "", fmt.Errorf("%s expects string second argument", name)
	}
	return first, second, nil
}

func oneObject(name string, args []Value) (Object, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s expects 1 argument", name)
	}
	obj, ok := args[0].(Object)
	if !ok {
		return nil, fmt.Errorf("%s expects object", name)
	}
	return obj, nil
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
	case *Array:
		parts := make([]string, 0, len(x.items))
		for _, item := range x.items {
			parts = append(parts, stringify(item))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		return fmt.Sprintf("%v", x)
	}
}
