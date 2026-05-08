package eval

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
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

type raisedSignal struct {
	value Value
}

func (r *raisedSignal) Error() string {
	return "uncaught raised value: " + stringify(r.value)
}

type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit %d", e.Code)
}

type Value any

type Dict map[string]Value
type Array struct {
	items []Value
}
type Tuple struct {
	items []Value
}
type ErrorValue struct {
	Message string
}
type Module struct {
	Name    string
	Members Dict
}

type Function struct {
	Params []string
	Body   []ast.Stmt
	Expr   ast.Expr
	Env    *Env
	Name   string
}

type Builtin func([]Value) (Value, error)

type Env struct {
	parent    *Env
	vars      map[string]Value
	inFunc    bool
	runeCache map[string][]rune
}

func NewEnv() *Env {
	return &Env{vars: map[string]Value{}, runeCache: map[string][]rune{}}
}

func (e *Env) child() *Env {
	return &Env{parent: e, vars: map[string]Value{}, inFunc: e.inFunc, runeCache: e.runeCache}
}

func (e *Env) runes(text string) []rune {
	if runes, ok := e.runeCache[text]; ok {
		return runes
	}
	runes := []rune(text)
	e.runeCache[text] = runes
	return runes
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
	return RunWithArgs(prog, out, nil)
}

func RunWithArgs(prog *ast.Program, out io.Writer, args []string) error {
	return RunWithIO(prog, nil, out, args)
}

func RunWithIO(prog *ast.Program, in io.Reader, out io.Writer, args []string) error {
	env := NewEnv()
	installBuiltins(env, in, out, args)
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
	var raised *raisedSignal
	if errors.As(err, &raised) {
		return fmt.Errorf("uncaught raised value: %s", stringify(raised.value))
	}
	return err
}

func installBuiltins(env *Env, in io.Reader, out io.Writer, processArgs []string) {
	var lineReader *bufio.Reader
	if in != nil {
		lineReader = bufio.NewReader(in)
	}
	env.set("print", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("print expects 1 argument")
		}
		fmt.Fprintln(out, stringify(args[0]))
		return nil, nil
	}))
	env.set("println", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("println expects 1 argument")
		}
		fmt.Fprintln(out, stringify(args[0]))
		return nil, nil
	}))
	env.set("read_line", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("read_line expects 0 arguments")
		}
		if lineReader == nil {
			return nil, nil
		}
		line, err := lineReader.ReadString('\n')
		if errors.Is(err, io.EOF) && line == "" {
			return nil, nil
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
		return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
	}))
	env.set("len", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("len expects 1 argument")
		}
		switch v := args[0].(type) {
		case string:
			return int64(len(env.runes(v))), nil
		case *Array:
			return int64(len(v.items)), nil
		case Dict:
			return int64(len(v)), nil
		default:
			return nil, fmt.Errorf("len expects string, array, or dictionary")
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
	env.set("map", Builtin(func(args []Value) (Value, error) {
		arr, fn, err := arrayAndFunction("map", args)
		if err != nil {
			return nil, err
		}
		out := &Array{items: make([]Value, 0, len(arr.items))}
		for _, item := range arr.items {
			mapped, err := callValue(fn, []Value{item})
			if err != nil {
				return nil, err
			}
			out.items = append(out.items, mapped)
		}
		return out, nil
	}))
	env.set("filter", Builtin(func(args []Value) (Value, error) {
		arr, fn, err := arrayAndFunction("filter", args)
		if err != nil {
			return nil, err
		}
		out := &Array{}
		for _, item := range arr.items {
			keep, err := callValue(fn, []Value{item})
			if err != nil {
				return nil, err
			}
			if truthy(keep) {
				out.items = append(out.items, item)
			}
		}
		return out, nil
	}))
	env.set("find", Builtin(func(args []Value) (Value, error) {
		arr, fn, err := arrayAndFunction("find", args)
		if err != nil {
			return nil, err
		}
		for _, item := range arr.items {
			found, err := callValue(fn, []Value{item})
			if err != nil {
				return nil, err
			}
			if truthy(found) {
				return item, nil
			}
		}
		return nil, nil
	}))
	env.set("any", Builtin(func(args []Value) (Value, error) {
		arr, fn, err := arrayAndFunction("any", args)
		if err != nil {
			return nil, err
		}
		for _, item := range arr.items {
			ok, err := callValue(fn, []Value{item})
			if err != nil {
				return nil, err
			}
			if truthy(ok) {
				return true, nil
			}
		}
		return false, nil
	}))
	env.set("all", Builtin(func(args []Value) (Value, error) {
		arr, fn, err := arrayAndFunction("all", args)
		if err != nil {
			return nil, err
		}
		for _, item := range arr.items {
			ok, err := callValue(fn, []Value{item})
			if err != nil {
				return nil, err
			}
			if !truthy(ok) {
				return false, nil
			}
		}
		return true, nil
	}))
	env.set("each", Builtin(func(args []Value) (Value, error) {
		arr, fn, err := arrayAndFunction("each", args)
		if err != nil {
			return nil, err
		}
		for _, item := range arr.items {
			if _, err := callValue(fn, []Value{item}); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}))
	env.set("reduce", Builtin(func(args []Value) (Value, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("reduce expects 3 arguments")
		}
		arr, ok := args[0].(*Array)
		if !ok {
			return nil, fmt.Errorf("reduce expects array")
		}
		fn, ok := args[2].(*Function)
		if !ok {
			return nil, fmt.Errorf("reduce expects function")
		}
		acc := args[1]
		for _, item := range arr.items {
			next, err := callValue(fn, []Value{acc, item})
			if err != nil {
				return nil, err
			}
			acc = next
		}
		return acc, nil
	}))
	env.set("keys", Builtin(func(args []Value) (Value, error) {
		obj, err := oneDict("keys", args)
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
		obj, err := oneDict("values", args)
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
		obj, ok := args[0].(Dict)
		if !ok {
			return nil, fmt.Errorf("has expects dictionary")
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
		obj, ok := args[0].(Dict)
		if !ok {
			return nil, fmt.Errorf("delete expects dictionary")
		}
		key, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("delete expects string key")
		}
		delete(obj, key)
		return nil, nil
	}))
	env.set("equal", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("equal expects 2 arguments")
		}
		return deepEqual(args[0], args[1]), nil
	}))
	env.set("to_string", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("to_string expects 1 argument")
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
	env.set("starts_with", Builtin(func(args []Value) (Value, error) {
		text, prefix, err := twoStrings("starts_with", args)
		if err != nil {
			return nil, err
		}
		return strings.HasPrefix(text, prefix), nil
	}))
	env.set("ends_with", Builtin(func(args []Value) (Value, error) {
		text, suffix, err := twoStrings("ends_with", args)
		if err != nil {
			return nil, err
		}
		return strings.HasSuffix(text, suffix), nil
	}))
	env.set("byte_len", Builtin(func(args []Value) (Value, error) {
		text, err := oneString("byte_len", args)
		if err != nil {
			return nil, err
		}
		return int64(len(text)), nil
	}))
	env.set("char_len", Builtin(func(args []Value) (Value, error) {
		text, err := oneString("char_len", args)
		if err != nil {
			return nil, err
		}
		return int64(len([]rune(text))), nil
	}))
	env.set("read_file", Builtin(func(args []Value) (Value, error) {
		path, err := oneString("read_file", args)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return string(data), nil
	}))
	env.set("write_file", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("write_file expects 2 arguments")
		}
		path, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("write_file expects string path")
		}
		text, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("write_file expects string text")
		}
		if err := os.WriteFile(path, []byte(text), 0644); err != nil {
			return nil, err
		}
		return nil, nil
	}))
	env.set("file_exists", Builtin(func(args []Value) (Value, error) {
		path, err := oneString("file_exists", args)
		if err != nil {
			return nil, err
		}
		_, statErr := os.Stat(path)
		if statErr == nil {
			return true, nil
		}
		if errors.Is(statErr, os.ErrNotExist) {
			return false, nil
		}
		return nil, statErr
	}))
	env.set("args", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("args expects 0 arguments")
		}
		arr := &Array{items: make([]Value, 0, len(processArgs))}
		for _, arg := range processArgs {
			arr.items = append(arr.items, arg)
		}
		return arr, nil
	}))
	env.set("env", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("env expects 1 argument")
		}
		name, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("env expects string name")
		}
		value, ok := os.LookupEnv(name)
		if !ok {
			return nil, nil
		}
		return value, nil
	}))
	env.set("exit", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("exit expects 1 argument")
		}
		code, ok := args[0].(int64)
		if !ok {
			return nil, fmt.Errorf("exit expects int code")
		}
		return nil, &ExitError{Code: int(code)}
	}))
	env.set("panic", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("panic expects 1 argument")
		}
		return nil, fmt.Errorf("panic: %s", stringify(args[0]))
	}))
	env.set("error", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("error expects 1 argument")
		}
		message, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("error expects string message")
		}
		return &ErrorValue{Message: message}, nil
	}))
	env.set("div", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("div expects 2 arguments")
		}
		left, ok := args[0].(int64)
		if !ok {
			return nil, fmt.Errorf("div expects int left operand")
		}
		right, ok := args[1].(int64)
		if !ok {
			return nil, fmt.Errorf("div expects int right operand")
		}
		if right == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return left / right, nil
	}))
	env.set("to_int", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("to_int expects 1 argument")
		}
		return parseIntValue(args[0])
	}))
	env.set("to_float", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("to_float expects 1 argument")
		}
		return parseFloatValue(args[0])
	}))
	env.set("to_number", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("to_number expects 1 argument")
		}
		if i, err := parseIntValue(args[0]); err == nil {
			return i, nil
		}
		return parseFloatValue(args[0])
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
	case *ast.ImportStmt:
		if n.Alias != "" {
			value, ok := env.get(n.ModuleName())
			if !ok {
				return nil, fmt.Errorf("undefined imported module %s", n.ModuleName())
			}
			env.set(n.Alias, value)
			return value, nil
		}
		return nil, nil
	case *ast.AssignStmt:
		values, err := evalValues(n.Values, env)
		if err != nil {
			return nil, err
		}
		if len(values) != len(n.Targets) {
			return nil, fmt.Errorf("assignment expects %d values, got %d", len(n.Targets), len(values))
		}
		for i, target := range n.Targets {
			if id, ok := target.(*ast.Ident); ok {
				values[i] = nameFunction(id.Name, values[i])
			}
			if err := assign(target, values[i], env); err != nil {
				return nil, err
			}
		}
		if len(values) == 0 {
			return nil, nil
		}
		return values[len(values)-1], nil
	case *ast.ModuleDecl:
		module := &Module{Name: n.Name, Members: Dict{}}
		env.set(n.Name, module)
		for _, member := range n.Members {
			value, err := evalExpr(member.Value, env)
			if err != nil {
				return nil, err
			}
			module.Members[member.Name] = nameFunction(member.Name, value)
		}
		return module, nil
	case *ast.InterfaceDecl:
		return nil, nil
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
		if n.Kind == "of" {
			return evalDictFor(n, iterable, env)
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
		if len(n.Values) == 0 {
			return nil, &returnSignal{}
		}
		values, err := evalValues(n.Values, env)
		if err != nil {
			return nil, err
		}
		if len(values) == 1 {
			return nil, &returnSignal{value: values[0]}
		}
		return nil, &returnSignal{value: &Tuple{items: values}}
	case *ast.RaiseStmt:
		value, err := evalExpr(n.Value, env)
		if err != nil {
			return nil, err
		}
		return nil, &raisedSignal{value: value}
	case *ast.TryCatchStmt:
		last, err := evalStmts(n.Try, env)
		var raised *raisedSignal
		if !errors.As(err, &raised) {
			return last, err
		}
		catchEnv := env.child()
		if n.CatchName != "_" {
			catchEnv.set(n.CatchName, raised.value)
		}
		return evalStmts(n.Catch, catchEnv)
	case *ast.MatchStmt:
		value, err := evalExpr(n.Value, env)
		if err != nil {
			return nil, err
		}
		for _, c := range n.Cases {
			bindings := map[string]Value{}
			if !matchPattern(c.Pattern, value, bindings) {
				continue
			}
			caseEnv := env.child()
			for name, value := range bindings {
				caseEnv.vars[name] = value
			}
			return evalStmts(c.Body, caseEnv)
		}
		return nil, nil
	}
	return nil, fmt.Errorf("unknown statement")
}

func matchPattern(pattern ast.Expr, value Value, bindings map[string]Value) bool {
	switch n := pattern.(type) {
	case *ast.Ident:
		if n.Name == "_" {
			return true
		}
		bindings[n.Name] = value
		return true
	case *ast.IntLit:
		return equal(value, n.Value)
	case *ast.FloatLit:
		return equal(value, n.Value)
	case *ast.StringLit:
		return equal(value, n.Value)
	case *ast.BoolLit:
		return equal(value, n.Value)
	case *ast.NilLit:
		return value == nil
	case *ast.ArrayLit:
		arr, ok := value.(*Array)
		if !ok || len(arr.items) != len(n.Elems) {
			return false
		}
		for i, elem := range n.Elems {
			if !matchPattern(elem, arr.items[i], bindings) {
				return false
			}
		}
		return true
	case *ast.DictLit:
		dict, ok := value.(Dict)
		if !ok {
			return false
		}
		for _, prop := range n.Props {
			item, ok := dict[prop.Name]
			if !ok || !matchPattern(prop.Value, item, bindings) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func evalValues(exprs []ast.Expr, env *Env) ([]Value, error) {
	var values []Value
	for _, expr := range exprs {
		value, err := evalExpr(expr, env)
		if err != nil {
			return nil, err
		}
		if tuple, ok := value.(*Tuple); ok {
			values = append(values, tuple.items...)
		} else {
			values = append(values, value)
		}
	}
	return values, nil
}

func assign(target ast.Expr, v Value, env *Env) error {
	switch t := target.(type) {
	case *ast.Ident:
		if t.Name == "_" {
			return nil
		}
		env.set(t.Name, nameFunction(t.Name, v))
		return nil
	case *ast.ArrayLit:
		arr, ok := v.(*Array)
		if !ok {
			return fmt.Errorf("array destructuring target is not array")
		}
		if len(arr.items) != len(t.Elems) {
			return fmt.Errorf("array destructuring expects %d elements, got %d", len(t.Elems), len(arr.items))
		}
		for i, elem := range t.Elems {
			if err := assign(elem, arr.items[i], env); err != nil {
				return err
			}
		}
		return nil
	case *ast.DictLit:
		dict, ok := v.(Dict)
		if !ok {
			return fmt.Errorf("dictionary destructuring target is not dictionary")
		}
		for _, prop := range t.Props {
			value, ok := dict[prop.Name]
			if !ok {
				return fmt.Errorf("dictionary destructuring missing key %s", prop.Name)
			}
			if err := assign(prop.Value, value, env); err != nil {
				return err
			}
		}
		return nil
	case *ast.MemberExpr:
		obj, err := evalExpr(t.Target, env)
		if err != nil {
			return err
		}
		o, ok := obj.(Dict)
		if !ok {
			return fmt.Errorf("cannot assign property on non-dictionary")
		}
		o[t.Name] = v
		return nil
	case *ast.IndexExpr:
		obj, err := evalExpr(t.Target, env)
		if err != nil {
			return err
		}
		idx, err := evalExpr(t.Index, env)
		if err != nil {
			return err
		}
		if dict, ok := obj.(Dict); ok {
			key, ok := idx.(string)
			if !ok {
				return fmt.Errorf("dictionary index must be string")
			}
			dict[key] = v
			return nil
		}
		arr, ok := obj.(*Array)
		if !ok {
			return fmt.Errorf("index assignment target is not array or dictionary")
		}
		i, ok := idx.(int64)
		if !ok {
			return fmt.Errorf("array index must be int")
		}
		if i < 0 || i >= int64(len(arr.items)) {
			return fmt.Errorf("array assignment index out of range")
		}
		arr.items[i] = v
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
	case *ast.DictLit:
		dict := Dict{}
		for _, p := range n.Props {
			v, err := evalExpr(p.Value, env)
			if err != nil {
				return nil, err
			}
			dict[p.Name] = nameFunction(p.Name, v)
		}
		return dict, nil
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
		if n.Op.Lexeme == "-" {
			switch x := v.(type) {
			case int64:
				return -x, nil
			case float64:
				return -x, nil
			default:
				return nil, fmt.Errorf("- expects number")
			}
		}
		return nil, fmt.Errorf("unknown unary operator %s", n.Op.Lexeme)
	case *ast.TryExpr:
		if !env.inFunc {
			return nil, fmt.Errorf("%d:%d: try used outside function", n.Tok.Line, n.Tok.Col)
		}
		v, err := evalExpr(n.Expr, env)
		if err != nil {
			return nil, err
		}
		tuple, ok := v.(*Tuple)
		if !ok || len(tuple.items) != 2 {
			return nil, fmt.Errorf("%d:%d: try expects value, err tuple", n.Tok.Line, n.Tok.Col)
		}
		if tuple.items[1] != nil {
			return nil, &returnSignal{value: &Tuple{items: []Value{nil, tuple.items[1]}}}
		}
		return tuple.items[0], nil
	case *ast.MemberExpr:
		obj, err := evalExpr(n.Target, env)
		if err != nil {
			return nil, err
		}
		if errValue, ok := obj.(*ErrorValue); ok {
			if n.Name == "message" {
				return errValue.Message, nil
			}
			return nil, nil
		}
		if module, ok := obj.(*Module); ok {
			return module.Members[n.Name], nil
		}
		o, ok := obj.(Dict)
		if !ok {
			return nil, fmt.Errorf("cannot read property %s on non-dictionary", n.Name)
		}
		return o[n.Name], nil
	case *ast.IndexExpr:
		obj, err := evalExpr(n.Target, env)
		if err != nil {
			return nil, err
		}
		idx, err := evalExpr(n.Index, env)
		if err != nil {
			return nil, err
		}
		if dict, ok := obj.(Dict); ok {
			key, ok := idx.(string)
			if !ok {
				return nil, fmt.Errorf("dictionary index must be string")
			}
			return dict[key], nil
		}
		if errValue, ok := obj.(*ErrorValue); ok {
			key, ok := idx.(string)
			if !ok {
				return nil, fmt.Errorf("error index must be string")
			}
			if key == "message" {
				return errValue.Message, nil
			}
			return nil, nil
		}
		arr, ok := obj.(*Array)
		if !ok {
			text, ok := obj.(string)
			if !ok {
				return nil, fmt.Errorf("index target is not array or string")
			}
			i, ok := idx.(int64)
			if !ok {
				return nil, fmt.Errorf("string index must be int")
			}
			runes := env.runes(text)
			if i < 0 || i >= int64(len(runes)) {
				return nil, nil
			}
			return string(runes[i]), nil
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
	if member, ok := c.Callee.(*ast.MemberExpr); ok {
		if target, ok := member.Target.(*ast.Ident); ok {
			args := make([]Value, 0, len(c.Args))
			for _, a := range c.Args {
				v, err := evalExpr(a, env)
				if err != nil {
					return nil, err
				}
				args = append(args, v)
			}
			if value, handled, err := evalStandardModuleCall(target.Name, member.Name, args, env); handled {
				return value, err
			}
		}
	}
	fnVal, err := evalCallee(c.Callee, env)
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
	case Builtin, *Function:
		return callValue(fn, args)
	}
	return nil, fmt.Errorf("value is not callable")
}

func evalStandardModuleCall(module string, name string, args []Value, env *Env) (Value, bool, error) {
	switch module {
	case "string":
		if !isStandardStringCall(name) {
			return nil, false, nil
		}
		value, err := evalStringModuleCall(name, args, env)
		return value, true, err
	case "array":
		if !isStandardArrayCall(name) {
			return nil, false, nil
		}
		value, err := evalArrayModuleCall(name, args, env)
		return value, true, err
	case "dict":
		if !isStandardDictCall(name) {
			return nil, false, nil
		}
		value, err := evalDictModuleCall(name, args, env)
		return value, true, err
	case "value":
		if name != "nil?" {
			return nil, false, nil
		}
		if len(args) != 1 {
			return nil, true, fmt.Errorf("value.nil? expects 1 argument")
		}
		return args[0] == nil, true, nil
	default:
		return nil, false, nil
	}
}

func isStandardStringCall(name string) bool {
	switch name {
	case "len", "byte_len", "char_len", "trim", "contains", "starts_with", "ends_with", "replace", "split", "join", "lines", "upcase", "downcase":
		return true
	default:
		return false
	}
}

func isStandardArrayCall(name string) bool {
	switch name {
	case "len", "empty?", "first", "last", "push", "pop", "slice", "reverse", "join", "map", "filter", "find", "any", "all", "each", "reduce":
		return true
	default:
		return false
	}
}

func isStandardDictCall(name string) bool {
	switch name {
	case "len", "has", "has?", "get", "set", "delete", "keys", "values", "merge":
		return true
	default:
		return false
	}
}

func evalStringModuleCall(name string, args []Value, env *Env) (Value, error) {
	switch name {
	case "len":
		return evalLen("string.len", args, env)
	case "byte_len":
		text, err := oneString("string.byte_len", args)
		if err != nil {
			return nil, err
		}
		return int64(len(text)), nil
	case "char_len":
		text, err := oneString("string.char_len", args)
		if err != nil {
			return nil, err
		}
		return int64(len([]rune(text))), nil
	case "trim":
		text, err := oneString("string.trim", args)
		if err != nil {
			return nil, err
		}
		return strings.TrimSpace(text), nil
	case "contains":
		text, part, err := twoStrings("string.contains", args)
		if err != nil {
			return nil, err
		}
		return strings.Contains(text, part), nil
	case "starts_with":
		text, prefix, err := twoStrings("string.starts_with", args)
		if err != nil {
			return nil, err
		}
		return strings.HasPrefix(text, prefix), nil
	case "ends_with":
		text, suffix, err := twoStrings("string.ends_with", args)
		if err != nil {
			return nil, err
		}
		return strings.HasSuffix(text, suffix), nil
	case "replace":
		if len(args) != 3 {
			return nil, fmt.Errorf("string.replace expects 3 arguments")
		}
		text, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("string.replace expects string text")
		}
		old, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("string.replace expects string old value")
		}
		newValue, ok := args[2].(string)
		if !ok {
			return nil, fmt.Errorf("string.replace expects string new value")
		}
		return strings.ReplaceAll(text, old, newValue), nil
	case "split":
		if len(args) != 2 {
			return nil, fmt.Errorf("string.split expects 2 arguments")
		}
		text, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("string.split expects string text")
		}
		sep, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("string.split expects string separator")
		}
		parts := strings.Split(text, sep)
		arr := &Array{items: make([]Value, 0, len(parts))}
		for _, part := range parts {
			arr.items = append(arr.items, part)
		}
		return arr, nil
	case "join":
		return evalJoin("string.join", args)
	case "lines":
		text, err := oneString("string.lines", args)
		if err != nil {
			return nil, err
		}
		text = strings.TrimSuffix(strings.TrimSuffix(text, "\n"), "\r")
		if text == "" {
			return &Array{}, nil
		}
		raw := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
		arr := &Array{items: make([]Value, 0, len(raw))}
		for _, line := range raw {
			arr.items = append(arr.items, strings.TrimSuffix(line, "\r"))
		}
		return arr, nil
	case "upcase":
		text, err := oneString("string.upcase", args)
		if err != nil {
			return nil, err
		}
		return strings.ToUpper(text), nil
	case "downcase":
		text, err := oneString("string.downcase", args)
		if err != nil {
			return nil, err
		}
		return strings.ToLower(text), nil
	default:
		return nil, fmt.Errorf("unknown string.%s", name)
	}
}

func evalArrayModuleCall(name string, args []Value, env *Env) (Value, error) {
	switch name {
	case "len":
		return evalLen("array.len", args, env)
	case "empty?":
		arr, err := oneArray("array.empty?", args)
		if err != nil {
			return nil, err
		}
		return len(arr.items) == 0, nil
	case "first":
		arr, err := oneArray("array.first", args)
		if err != nil {
			return nil, err
		}
		if len(arr.items) == 0 {
			return nil, nil
		}
		return arr.items[0], nil
	case "last":
		arr, err := oneArray("array.last", args)
		if err != nil {
			return nil, err
		}
		if len(arr.items) == 0 {
			return nil, nil
		}
		return arr.items[len(arr.items)-1], nil
	case "push":
		if len(args) != 2 {
			return nil, fmt.Errorf("array.push expects 2 arguments")
		}
		arr, ok := args[0].(*Array)
		if !ok {
			return nil, fmt.Errorf("array.push expects array")
		}
		arr.items = append(arr.items, args[1])
		return arr, nil
	case "pop":
		arr, err := oneArray("array.pop", args)
		if err != nil {
			return nil, err
		}
		if len(arr.items) == 0 {
			return nil, nil
		}
		last := arr.items[len(arr.items)-1]
		arr.items = arr.items[:len(arr.items)-1]
		return last, nil
	case "slice":
		if len(args) != 3 {
			return nil, fmt.Errorf("array.slice expects 3 arguments")
		}
		arr, ok := args[0].(*Array)
		if !ok {
			return nil, fmt.Errorf("array.slice expects array")
		}
		start, ok := args[1].(int64)
		if !ok {
			return nil, fmt.Errorf("array.slice expects integer start")
		}
		end, ok := args[2].(int64)
		if !ok {
			return nil, fmt.Errorf("array.slice expects integer end")
		}
		if start < 0 || end < 0 {
			return nil, fmt.Errorf("array.slice does not support negative indexes")
		}
		if start > end || int(end) > len(arr.items) {
			return nil, fmt.Errorf("array.slice index out of range")
		}
		out := &Array{items: append([]Value{}, arr.items[int(start):int(end)]...)}
		return out, nil
	case "reverse":
		arr, err := oneArray("array.reverse", args)
		if err != nil {
			return nil, err
		}
		out := &Array{items: make([]Value, len(arr.items))}
		for i := range arr.items {
			out.items[len(arr.items)-1-i] = arr.items[i]
		}
		return out, nil
	case "join":
		return evalJoin("array.join", args)
	case "map", "filter", "find", "any", "all", "each", "reduce":
		return evalCollectionBuiltin(name, args)
	default:
		return nil, fmt.Errorf("unknown array.%s", name)
	}
}

func evalDictModuleCall(name string, args []Value, env *Env) (Value, error) {
	switch name {
	case "len":
		return evalLen("dict.len", args, env)
	case "has", "has?":
		return evalHas("dict.has", args)
	case "get":
		if len(args) != 2 && len(args) != 3 {
			return nil, fmt.Errorf("dict.get expects 2 or 3 arguments")
		}
		obj, ok := args[0].(Dict)
		if !ok {
			return nil, fmt.Errorf("dict.get expects dictionary")
		}
		key, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("dict.get expects string key")
		}
		value, exists := obj[key]
		if exists {
			return value, nil
		}
		if len(args) == 3 {
			return args[2], nil
		}
		return nil, nil
	case "set":
		if len(args) != 3 {
			return nil, fmt.Errorf("dict.set expects 3 arguments")
		}
		obj, ok := args[0].(Dict)
		if !ok {
			return nil, fmt.Errorf("dict.set expects dictionary")
		}
		key, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("dict.set expects string key")
		}
		obj[key] = args[2]
		return obj, nil
	case "delete":
		if len(args) != 2 {
			return nil, fmt.Errorf("dict.delete expects 2 arguments")
		}
		obj, ok := args[0].(Dict)
		if !ok {
			return nil, fmt.Errorf("dict.delete expects dictionary")
		}
		key, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("dict.delete expects string key")
		}
		value := obj[key]
		delete(obj, key)
		return value, nil
	case "keys":
		obj, err := oneDict("dict.keys", args)
		if err != nil {
			return nil, err
		}
		arr := &Array{}
		for key := range obj {
			arr.items = append(arr.items, key)
		}
		return arr, nil
	case "values":
		obj, err := oneDict("dict.values", args)
		if err != nil {
			return nil, err
		}
		arr := &Array{}
		for _, value := range obj {
			arr.items = append(arr.items, value)
		}
		return arr, nil
	case "merge":
		if len(args) != 2 {
			return nil, fmt.Errorf("dict.merge expects 2 arguments")
		}
		left, ok := args[0].(Dict)
		if !ok {
			return nil, fmt.Errorf("dict.merge expects dictionary left")
		}
		right, ok := args[1].(Dict)
		if !ok {
			return nil, fmt.Errorf("dict.merge expects dictionary right")
		}
		out := Dict{}
		for key, value := range left {
			out[key] = value
		}
		for key, value := range right {
			out[key] = value
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unknown dict.%s", name)
	}
}

func evalLen(name string, args []Value, env *Env) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s expects 1 argument", name)
	}
	switch v := args[0].(type) {
	case string:
		return int64(len(env.runes(v))), nil
	case *Array:
		return int64(len(v.items)), nil
	case Dict:
		return int64(len(v)), nil
	default:
		return nil, fmt.Errorf("%s expects string, array, or dictionary", name)
	}
}

func evalJoin(name string, args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s expects 2 arguments", name)
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return nil, fmt.Errorf("%s expects array", name)
	}
	sep, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("%s expects string separator", name)
	}
	parts := make([]string, 0, len(arr.items))
	for _, item := range arr.items {
		parts = append(parts, stringify(item))
	}
	return strings.Join(parts, sep), nil
}

func evalHas(name string, args []Value) (Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s expects 2 arguments", name)
	}
	obj, ok := args[0].(Dict)
	if !ok {
		return nil, fmt.Errorf("%s expects dictionary", name)
	}
	key, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("%s expects string key", name)
	}
	_, exists := obj[key]
	return exists, nil
}

func evalCollectionBuiltin(name string, args []Value) (Value, error) {
	switch name {
	case "map":
		arr, fn, err := arrayAndFunction("array.map", args)
		if err != nil {
			return nil, err
		}
		out := &Array{items: make([]Value, 0, len(arr.items))}
		for _, item := range arr.items {
			mapped, err := callValue(fn, []Value{item})
			if err != nil {
				return nil, err
			}
			out.items = append(out.items, mapped)
		}
		return out, nil
	case "filter":
		arr, fn, err := arrayAndFunction("array.filter", args)
		if err != nil {
			return nil, err
		}
		out := &Array{}
		for _, item := range arr.items {
			keep, err := callValue(fn, []Value{item})
			if err != nil {
				return nil, err
			}
			if truthy(keep) {
				out.items = append(out.items, item)
			}
		}
		return out, nil
	case "find":
		arr, fn, err := arrayAndFunction("array.find", args)
		if err != nil {
			return nil, err
		}
		for _, item := range arr.items {
			found, err := callValue(fn, []Value{item})
			if err != nil {
				return nil, err
			}
			if truthy(found) {
				return item, nil
			}
		}
		return nil, nil
	case "any":
		arr, fn, err := arrayAndFunction("array.any", args)
		if err != nil {
			return nil, err
		}
		for _, item := range arr.items {
			ok, err := callValue(fn, []Value{item})
			if err != nil {
				return nil, err
			}
			if truthy(ok) {
				return true, nil
			}
		}
		return false, nil
	case "all":
		arr, fn, err := arrayAndFunction("array.all", args)
		if err != nil {
			return nil, err
		}
		for _, item := range arr.items {
			ok, err := callValue(fn, []Value{item})
			if err != nil {
				return nil, err
			}
			if !truthy(ok) {
				return false, nil
			}
		}
		return true, nil
	case "each":
		arr, fn, err := arrayAndFunction("array.each", args)
		if err != nil {
			return nil, err
		}
		for _, item := range arr.items {
			if _, err := callValue(fn, []Value{item}); err != nil {
				return nil, err
			}
		}
		return nil, nil
	case "reduce":
		if len(args) != 3 {
			return nil, fmt.Errorf("array.reduce expects 3 arguments")
		}
		arr, ok := args[0].(*Array)
		if !ok {
			return nil, fmt.Errorf("array.reduce expects array")
		}
		fn, ok := args[2].(*Function)
		if !ok {
			return nil, fmt.Errorf("array.reduce expects function")
		}
		acc := args[1]
		for _, item := range arr.items {
			next, err := callValue(fn, []Value{acc, item})
			if err != nil {
				return nil, err
			}
			acc = next
		}
		return acc, nil
	default:
		return nil, fmt.Errorf("unknown array.%s", name)
	}
}

func callValue(fn Value, args []Value) (Value, error) {
	switch f := fn.(type) {
	case Builtin:
		return f(args)
	case *Function:
		if len(args) != len(f.Params) {
			return nil, fmt.Errorf("function expects %d arguments, got %d", len(f.Params), len(args))
		}
		callEnv := f.Env.child()
		callEnv.inFunc = true
		for i, name := range f.Params {
			callEnv.set(name, args[i])
		}
		checkPredicate := func(v Value, err error) (Value, error) {
			if err != nil {
				return v, err
			}
			if !strings.HasSuffix(f.Name, "?") {
				return v, nil
			}
			if _, ok := v.(bool); !ok {
				return nil, fmt.Errorf("%s must return boolean", f.Name)
			}
			return v, nil
		}
		if f.Expr != nil {
			return checkPredicate(evalExpr(f.Expr, callEnv))
		}
		v, err := evalStmts(f.Body, callEnv)
		var ret *returnSignal
		if errors.As(err, &ret) {
			return checkPredicate(ret.value, nil)
		}
		return checkPredicate(v, err)
	}
	return nil, fmt.Errorf("value is not callable")
}

func nameFunction(name string, value Value) Value {
	if fn, ok := value.(*Function); ok {
		fn.Name = name
	}
	return value
}

func evalDictFor(n *ast.ForInStmt, iterable Value, env *Env) (Value, error) {
	obj, ok := iterable.(Dict)
	if !ok {
		return nil, fmt.Errorf("for of expects dictionary")
	}
	var last Value
	for key, value := range obj {
		env.set(n.ValueName, key)
		if n.IndexName != "" {
			env.set(n.IndexName, value)
		}
		var err error
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
}

func evalCallee(e ast.Expr, env *Env) (Value, error) {
	if m, ok := e.(*ast.MemberExpr); ok {
		obj, err := evalExpr(m.Target, env)
		if err != nil {
			return nil, err
		}
		if module, ok := obj.(*Module); ok {
			return module.Members[m.Name], nil
		}
		return nil, fmt.Errorf("member calls require module receiver")
	}
	return evalExpr(e, env)
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
	case Dict:
		rv, ok := r.(Dict)
		return ok && fmt.Sprintf("%p", lv) == fmt.Sprintf("%p", rv)
	case *Array:
		return lv == r
	case *Function:
		return lv == r
	case *ErrorValue:
		return lv == r
	case *Module:
		return lv == r
	}
	return false
}

func deepEqual(l, r Value) bool {
	switch lv := l.(type) {
	case *Array:
		rv, ok := r.(*Array)
		if !ok || len(lv.items) != len(rv.items) {
			return false
		}
		for i := range lv.items {
			if !deepEqual(lv.items[i], rv.items[i]) {
				return false
			}
		}
		return true
	case Dict:
		rv, ok := r.(Dict)
		if !ok || len(lv) != len(rv) {
			return false
		}
		for key, leftValue := range lv {
			rightValue, ok := rv[key]
			if !ok || !deepEqual(leftValue, rightValue) {
				return false
			}
		}
		return true
	default:
		return equal(l, r)
	}
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

func parseIntValue(v Value) (Value, error) {
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

func parseFloatValue(v Value) (Value, error) {
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

func oneDict(name string, args []Value) (Dict, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s expects 1 argument", name)
	}
	obj, ok := args[0].(Dict)
	if !ok {
		return nil, fmt.Errorf("%s expects dictionary", name)
	}
	return obj, nil
}

func oneArray(name string, args []Value) (*Array, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s expects 1 argument", name)
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return nil, fmt.Errorf("%s expects array", name)
	}
	return arr, nil
}

func arrayAndFunction(name string, args []Value) (*Array, *Function, error) {
	if len(args) != 2 {
		return nil, nil, fmt.Errorf("%s expects 2 arguments", name)
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return nil, nil, fmt.Errorf("%s expects array", name)
	}
	fn, ok := args[1].(*Function)
	if !ok {
		return nil, nil, fmt.Errorf("%s expects function", name)
	}
	return arr, fn, nil
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

func interpolate(s string, env *Env) (string, error) {
	var out strings.Builder
	for i := 0; i < len(s); {
		switch s[i] {
		case '{':
			if i+1 < len(s) && s[i+1] == '{' {
				out.WriteByte('{')
				i += 2
				continue
			}
			close := strings.IndexByte(s[i+1:], '}')
			if close < 0 {
				return "", fmt.Errorf("unclosed interpolation")
			}
			expr := strings.TrimSpace(s[i+1 : i+1+close])
			if expr == "" {
				return "", fmt.Errorf("empty interpolation")
			}
			v, err := evalInterpolationExpr(expr, env)
			if err != nil {
				return "", err
			}
			out.WriteString(stringify(v))
			i += close + 2
		case '}':
			if i+1 < len(s) && s[i+1] == '}' {
				out.WriteByte('}')
				i += 2
				continue
			}
			return "", fmt.Errorf("unmatched '}' in string interpolation")
		default:
			out.WriteByte(s[i])
			i++
		}
	}
	return out.String(), nil
}

func evalInterpolationExpr(expr string, env *Env) (Value, error) {
	toks, errs := lexer.Lex(expr)
	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid interpolation expression: %w", errs[0])
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		return nil, fmt.Errorf("invalid interpolation expression: %w", err)
	}
	if len(prog.Stmts) != 1 {
		return nil, fmt.Errorf("interpolation must contain one expression")
	}
	stmt, ok := prog.Stmts[0].(*ast.ExprStmt)
	if !ok {
		return nil, fmt.Errorf("interpolation must contain an expression")
	}
	return evalExpr(stmt.Expr, env)
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
	case *ErrorValue:
		return "error: " + x.Message
	case *Array:
		parts := make([]string, 0, len(x.items))
		for _, item := range x.items {
			parts = append(parts, stringify(item))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *Tuple:
		parts := make([]string, 0, len(x.items))
		for _, item := range x.items {
			parts = append(parts, stringify(item))
		}
		return strings.Join(parts, ", ")
	case *Module:
		return "<module " + x.Name + ">"
	default:
		return fmt.Sprintf("%v", x)
	}
}
