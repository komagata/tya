package eval

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	mathpkg "math"
	mathrand "math/rand"
	"net"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"tya/internal/ast"
	"tya/internal/interp"
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
type Bytes struct {
	data []byte
}
type IOStream struct {
	file     *os.File
	binary   bool
	readable bool
	writable bool
	borrowed bool
	closed   bool
}
type TCPSocket struct {
	conn     net.Conn
	listener net.Listener
	binary   bool
	timeout  time.Duration
	closed   bool
}
type Tuple struct {
	items []Value
}
type ErrorValue struct {
	Message string
	Kind    string
	Code    string
	Data    Dict
	Cause   *ErrorValue
}
type Module struct {
	Name    string
	Members Dict
}

type Function struct {
	Params   []string
	Defaults []ast.Expr
	Body     []ast.Stmt
	Expr     ast.Expr
	Env      *Env
	Name     string
}

type Builtin func([]Value) (Value, error)

func applyErrorOptions(errValue *ErrorValue, options Dict) error {
	for key, value := range options {
		switch key {
		case "message":
			message, ok := value.(string)
			if !ok {
				return fmt.Errorf("error options message must be string")
			}
			errValue.Message = message
		case "kind":
			kind, ok := value.(string)
			if !ok {
				return fmt.Errorf("error options kind must be string")
			}
			errValue.Kind = kind
		case "code":
			code, ok := value.(string)
			if !ok {
				return fmt.Errorf("error options code must be string")
			}
			errValue.Code = code
		case "data":
			data, ok := value.(Dict)
			if !ok {
				return fmt.Errorf("error options data must be dictionary")
			}
			errValue.Data = data
		case "cause":
			if value == nil {
				errValue.Cause = nil
				continue
			}
			cause, ok := value.(*ErrorValue)
			if !ok {
				return fmt.Errorf("error options cause must be error or nil")
			}
			errValue.Cause = cause
		default:
			return fmt.Errorf("error options unknown key %s", key)
		}
	}
	return nil
}

type Env struct {
	parent    *Env
	vars      map[string]Value
	kinds     map[string]string
	inFunc    bool
	runeCache map[string][]rune
}

func NewEnv() *Env {
	return &Env{vars: map[string]Value{}, kinds: map[string]string{}, runeCache: map[string][]rune{}}
}

func (e *Env) child() *Env {
	return &Env{parent: e, vars: map[string]Value{}, kinds: map[string]string{}, inFunc: e.inFunc, runeCache: e.runeCache}
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
		e.rememberKind(name, v)
		return
	}
	if e.parent.assignInNonRoot(name, v) {
		return
	}
	if !e.inFunc {
		if _, ok := e.parent.get(name); ok {
			e.parent.assign(name, v)
			return
		}
	}
	e.vars[name] = v
	e.rememberKind(name, v)
}

func (e *Env) assignInNonRoot(name string, v Value) bool {
	if e.parent == nil {
		return false
	}
	if _, ok := e.vars[name]; ok {
		e.vars[name] = v
		e.rememberKind(name, v)
		return true
	}
	if e.parent != nil {
		return e.parent.assignInNonRoot(name, v)
	}
	return false
}

func (e *Env) assign(name string, v Value) bool {
	if _, ok := e.vars[name]; ok {
		e.vars[name] = v
		e.rememberKind(name, v)
		return true
	}
	if e.parent != nil {
		return e.parent.assign(name, v)
	}
	return false
}

func (e *Env) kind(name string) (string, bool) {
	if kind, ok := e.kinds[name]; ok {
		return kind, true
	}
	if e.parent != nil {
		return e.parent.kind(name)
	}
	return "", false
}

func (e *Env) kindInNonRoot(name string) (string, bool) {
	if e.parent == nil {
		return "", false
	}
	if kind, ok := e.kinds[name]; ok {
		return kind, true
	}
	if e.parent != nil {
		return e.parent.kindInNonRoot(name)
	}
	return "", false
}

func (e *Env) rememberKind(name string, v Value) {
	kind := runtimeKind(v)
	if kind == "nil" {
		return
	}
	if _, ok := e.vars[name]; ok {
		e.kinds[name] = kind
		return
	}
	if e.parent != nil && e.parent.rememberKindInNonRoot(name, kind) {
		return
	}
	e.kinds[name] = kind
}

func (e *Env) rememberKindInNonRoot(name string, kind string) bool {
	if e.parent == nil {
		return false
	}
	if _, ok := e.vars[name]; ok {
		e.kinds[name] = kind
		return true
	}
	if e.parent != nil {
		return e.parent.rememberKindInNonRoot(name, kind)
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
	env.set("ord", Builtin(func(args []Value) (Value, error) {
		s, err := oneString("ord", args)
		if err != nil {
			return nil, err
		}
		if len(s) == 0 {
			return nil, &raisedSignal{value: "ord: argument must be a non-empty string"}
		}
		return int64(s[0]), nil
	}))
	env.set("chr", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("chr expects 1 argument")
		}
		v, ok := args[0].(int64)
		if !ok {
			return nil, fmt.Errorf("chr expects int argument")
		}
		if v < 0 || v > 255 {
			return nil, &raisedSignal{value: "chr: byte value out of range (0..255)"}
		}
		return string([]byte{byte(v)}), nil
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
		if !utf8.Valid(data) {
			return nil, &raisedSignal{value: "read_file: invalid UTF-8"}
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
	env.set("dir_list", Builtin(func(args []Value) (Value, error) {
		path, err := oneString("dir_list", args)
		if err != nil {
			return nil, err
		}
		entries, listErr := os.ReadDir(path)
		if listErr != nil {
			return nil, &raisedSignal{value: listErr.Error()}
		}
		arr := &Array{items: make([]Value, 0, len(entries))}
		for _, e := range entries {
			arr.items = append(arr.items, e.Name())
		}
		return arr, nil
	}))
	env.set("dir_mkdir", Builtin(func(args []Value) (Value, error) {
		path, err := oneString("dir_mkdir", args)
		if err != nil {
			return nil, err
		}
		if mkErr := os.Mkdir(path, 0755); mkErr != nil {
			return nil, &raisedSignal{value: mkErr.Error()}
		}
		return nil, nil
	}))
	env.set("dir_rmdir", Builtin(func(args []Value) (Value, error) {
		path, err := oneString("dir_rmdir", args)
		if err != nil {
			return nil, err
		}
		if rmErr := os.Remove(path); rmErr != nil {
			return nil, &raisedSignal{value: rmErr.Error()}
		}
		return nil, nil
	}))
	env.set("file_remove", Builtin(func(args []Value) (Value, error) {
		path, err := oneString("file_remove", args)
		if err != nil {
			return nil, err
		}
		info, statErr := os.Stat(path)
		if statErr != nil {
			return nil, &raisedSignal{value: statErr.Error()}
		}
		if info.IsDir() {
			return nil, &raisedSignal{value: "file.remove: target is a directory"}
		}
		if rmErr := os.Remove(path); rmErr != nil {
			return nil, &raisedSignal{value: rmErr.Error()}
		}
		return nil, nil
	}))
	env.set("file_rename", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("file_rename expects 2 arguments")
		}
		oldPath, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("file_rename expects string old path")
		}
		newPath, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("file_rename expects string new path")
		}
		if rnErr := os.Rename(oldPath, newPath); rnErr != nil {
			return nil, &raisedSignal{value: rnErr.Error()}
		}
		return nil, nil
	}))
	env.set("file_stat", Builtin(func(args []Value) (Value, error) {
		path, err := oneString("file_stat", args)
		if err != nil {
			return nil, err
		}
		info, statErr := os.Stat(path)
		if statErr != nil {
			return nil, &raisedSignal{value: statErr.Error()}
		}
		kind := "other"
		if info.Mode().IsRegular() {
			kind = "file"
		} else if info.IsDir() {
			kind = "dir"
		}
		out := Dict{}
		out["kind"] = kind
		out["size"] = int64(info.Size())
		mode := info.Mode()
		out["readable"] = mode&0444 != 0
		out["writable"] = mode&0222 != 0
		out["executable"] = mode&0111 != 0
		return out, nil
	}))
	env.set("path_expand_user", Builtin(func(args []Value) (Value, error) {
		v, err := oneString("path_expand_user", args)
		if err != nil {
			return nil, err
		}
		if v == "" || v[0] != '~' {
			return v, nil
		}
		home := os.Getenv("HOME")
		if v == "~" {
			return home, nil
		}
		if len(v) > 1 && v[1] == '/' {
			return home + v[1:], nil
		}
		return v, nil
	}))
	env.set("cwd", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("cwd expects 0 arguments")
		}
		dir, cwdErr := os.Getwd()
		if cwdErr != nil {
			return nil, &raisedSignal{value: cwdErr.Error()}
		}
		return dir, nil
	}))
	env.set("chdir", Builtin(func(args []Value) (Value, error) {
		path, err := oneString("chdir", args)
		if err != nil {
			return nil, err
		}
		if cdErr := os.Chdir(path); cdErr != nil {
			return nil, &raisedSignal{value: cdErr.Error()}
		}
		return nil, nil
	}))
	registerV24Builtins(env)
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
	env.set("environ", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("environ expects 0 arguments")
		}
		out := Dict{}
		for _, item := range os.Environ() {
			key, value, ok := strings.Cut(item, "=")
			if ok {
				out[key] = value
			}
		}
		return out, nil
	}))
	env.set("setenv", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("setenv expects 2 arguments")
		}
		name, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("setenv expects string name")
		}
		value, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("setenv expects string value")
		}
		if strings.ContainsRune(name, 0) || strings.ContainsRune(value, 0) {
			return nil, &raisedSignal{value: "os.env: NUL byte not allowed"}
		}
		if err := os.Setenv(name, value); err != nil {
			return nil, &raisedSignal{value: "os.env: " + err.Error()}
		}
		return nil, nil
	}))
	env.set("unsetenv", Builtin(func(args []Value) (Value, error) {
		name, err := oneString("unsetenv", args)
		if err != nil {
			return nil, err
		}
		if strings.ContainsRune(name, 0) {
			return nil, &raisedSignal{value: "os.env: NUL byte not allowed"}
		}
		if err := os.Unsetenv(name); err != nil {
			return nil, &raisedSignal{value: "os.env: " + err.Error()}
		}
		return nil, nil
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
		if len(args) < 1 || len(args) > 2 {
			return nil, fmt.Errorf("error expects 1 or 2 arguments")
		}
		if message, ok := args[0].(string); ok {
			errValue := &ErrorValue{Message: message, Kind: "error", Code: "", Data: Dict{}, Cause: nil}
			if len(args) == 2 {
				options, ok := args[1].(Dict)
				if !ok {
					return nil, fmt.Errorf("error options must be a dictionary")
				}
				if err := applyErrorOptions(errValue, options); err != nil {
					return nil, err
				}
			}
			return errValue, nil
		}
		if len(args) != 1 {
			return nil, fmt.Errorf("error message must be string")
		}
		options, ok := args[0].(Dict)
		if !ok {
			return nil, fmt.Errorf("error expects string message or dictionary options")
		}
		message, ok := options["message"].(string)
		if !ok {
			return nil, fmt.Errorf("error options message must be string")
		}
		errValue := &ErrorValue{Message: message, Kind: "error", Code: "", Data: Dict{}, Cause: nil}
		if err := applyErrorOptions(errValue, options); err != nil {
			return nil, err
		}
		return errValue, nil
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
}

func evalStmts(stmts []ast.Stmt, env *Env) (Value, error) {
	var last Value
	for _, s := range stmts {
		v, err := evalStmt(s, env)
		if err != nil {
			return last, err
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
	case *ast.ClassDecl:
		class := Dict{"__module_namespace": true, "__class_name": n.Name, "name": n.Name}
		env.set(n.Name, class)
		for _, method := range n.Methods {
			if !method.Class || method.Abstract {
				continue
			}
			value, err := evalExpr(method.Func, env)
			if err != nil {
				return nil, err
			}
			class[method.Name] = nameFunction(method.Name, value)
		}
		return class, nil
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
			return evalStmts(n.Then, env.child())
		}
		return evalStmts(n.Else, env.child())
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
			bodyValue, err := evalStmts(n.Body, env.child())
			if errors.Is(err, errBreak) {
				return bodyValue, nil
			}
			if errors.Is(err, errContinue) {
				continue
			}
			if err != nil {
				return nil, err
			}
			last = bodyValue
		}
	case *ast.ForInStmt:
		iterable, err := evalExpr(n.Iterable, env)
		if err != nil {
			return nil, err
		}
		arr, ok := iterable.(*Array)
		if !ok {
			if obj, ok := iterable.(Dict); ok {
				var last Value
				i := 0
				for key, value := range obj {
					loopEnv := env.child()
					loopEnv.vars[n.ValueName] = Dict{"key": key, "value": value}
					loopEnv.rememberKind(n.ValueName, loopEnv.vars[n.ValueName])
					if n.IndexName != "" {
						loopEnv.vars[n.IndexName] = int64(i)
						loopEnv.rememberKind(n.IndexName, loopEnv.vars[n.IndexName])
					}
					i++
					bodyValue, err := evalStmts(n.Body, loopEnv)
					if errors.Is(err, errBreak) {
						return bodyValue, nil
					}
					if errors.Is(err, errContinue) {
						continue
					}
					if err != nil {
						return nil, err
					}
					last = bodyValue
				}
				return last, nil
			}
			return nil, fmt.Errorf("for in expects array")
		}
		var last Value
		for i, item := range arr.items {
			loopEnv := env.child()
			loopEnv.vars[n.ValueName] = item
			loopEnv.rememberKind(n.ValueName, item)
			if n.IndexName != "" {
				loopEnv.vars[n.IndexName] = int64(i)
				loopEnv.rememberKind(n.IndexName, loopEnv.vars[n.IndexName])
			}
			bodyValue, err := evalStmts(n.Body, loopEnv)
			if errors.Is(err, errBreak) {
				return bodyValue, nil
			}
			if errors.Is(err, errContinue) {
				continue
			}
			if err != nil {
				return nil, err
			}
			last = bodyValue
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
		if _, ok := value.(*ErrorValue); !ok {
			return nil, fmt.Errorf("raise expects error value")
		}
		return nil, &raisedSignal{value: value}
	case *ast.TryCatchStmt:
		last, err := evalStmts(n.Try, env.child())
		var raised *raisedSignal
		if errors.As(err, &raised) && n.Catch != nil {
			catchEnv := env.child()
			if n.CatchName != "_" && n.CatchName != "" {
				catchEnv.set(n.CatchName, raised.value)
			}
			last, err = evalStmts(n.Catch, catchEnv)
		}
		if len(n.Finally) > 0 {
			finallyValue, finallyErr := evalStmts(n.Finally, env.child())
			if finallyErr != nil {
				return finallyValue, finallyErr
			}
		}
		return last, err
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
				caseEnv.rememberKind(name, value)
			}
			return evalStmts(c.Body, caseEnv)
		}
		return nil, nil
	case *ast.ScopeBlock:
		return nil, fmt.Errorf("%d:%d: scope block: not yet implemented (v0.42 STEP 1 only added syntax)", n.Tok.Line, n.Tok.Col)
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
		_, exists := env.get(t.Name)
		prevKind, hasKind := env.kind(t.Name)
		if env.inFunc {
			_, exists = env.vars[t.Name]
			prevKind, hasKind = env.kindInNonRoot(t.Name)
		}
		nextKind := runtimeKind(v)
		if exists && hasKind && prevKind != "nil" && nextKind != "nil" && prevKind != nextKind {
			return fmt.Errorf("cannot reassign %s from %s to %s", t.Name, prevKind, nextKind)
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
	case *ast.BytesLit:
		return &Bytes{data: []byte(n.Value)}, nil
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
		return &Function{Params: n.Params, Defaults: n.Defaults, Body: n.Body, Expr: n.Expr, Env: env}, nil
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
		if n.Op.Lexeme == "~" {
			i, ok := numberAsInt(v)
			if !ok {
				return nil, fmt.Errorf("~ expects integer")
			}
			return ^i, nil
		}
		return nil, fmt.Errorf("unknown unary operator %s", n.Op.Lexeme)
	case *ast.SpawnExpr:
		return nil, fmt.Errorf("%d:%d: spawn: not yet implemented (v0.42 STEP 1 only added syntax)", n.Tok.Line, n.Tok.Col)
	case *ast.AwaitExpr:
		return nil, fmt.Errorf("%d:%d: await: not yet implemented (v0.42 STEP 1 only added syntax)", n.Tok.Line, n.Tok.Col)
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
			switch n.Name {
			case "message":
				return errValue.Message, nil
			case "kind":
				return errValue.Kind, nil
			case "code":
				return errValue.Code, nil
			case "data":
				return errValue.Data, nil
			case "cause":
				if errValue.Cause == nil {
					return nil, nil
				}
				return errValue.Cause, nil
			}
			return nil, nil
		}
		if module, ok := obj.(*Module); ok {
			return module.Members[n.Name], nil
		}
		if n.Name == "class" {
			return classOf(obj), nil
		}
		o, ok := obj.(Dict)
		if !ok {
			return nil, fmt.Errorf("unknown member %s on %s", n.Name, runtimeKind(obj))
		}
		if isModuleNamespace(o) {
			if value, ok := o[n.Name]; ok {
				return value, nil
			}
			return nil, fmt.Errorf("unknown member %s on %s", n.Name, runtimeKind(o))
		}
		return nil, fmt.Errorf("unknown member %s on dictionary", n.Name)
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
			switch key {
			case "message":
				return errValue.Message, nil
			case "kind":
				return errValue.Kind, nil
			case "code":
				return errValue.Code, nil
			case "data":
				return errValue.Data, nil
			case "cause":
				if errValue.Cause == nil {
					return nil, nil
				}
				return errValue.Cause, nil
			}
			return nil, nil
		}
		if bytesVal, ok := obj.(*Bytes); ok {
			i, ok := idx.(int64)
			if !ok {
				return nil, fmt.Errorf("bytes index must be int")
			}
			if i < 0 {
				return nil, fmt.Errorf("negative indexes are invalid")
			}
			if i >= int64(len(bytesVal.data)) {
				return nil, nil
			}
			return int64(bytesVal.data[i]), nil
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
			if i < 0 {
				return nil, fmt.Errorf("negative indexes are invalid")
			}
			if i >= int64(len(runes)) {
				return nil, nil
			}
			return string(runes[i]), nil
		}
		i, ok := idx.(int64)
		if !ok {
			return nil, fmt.Errorf("array index must be int")
		}
		if i < 0 {
			return nil, fmt.Errorf("negative indexes are invalid")
		}
		if i >= int64(len(arr.items)) {
			return nil, nil
		}
		return arr.items[i], nil
	case *ast.CallExpr:
		return evalCall(n, env)
	}
	return nil, fmt.Errorf("unknown expression")
}

func evalCall(c *ast.CallExpr, env *Env) (Value, error) {
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
		if os.Getenv("TYA_LEGACY_MODULES") == "1" {
			return arr, nil
		}
		return nil, nil
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
		return nil, nil
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
		delete(obj, key)
		return nil, nil
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
	case Bytes:
		return int64(len(v.data)), nil
	default:
		return nil, fmt.Errorf("%s expects string, array, dictionary, or bytes", name)
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
		required := len(f.Params)
		for i, def := range f.Defaults {
			if i < len(f.Params) && def != nil {
				required = i
				break
			}
		}
		if len(args) < required || len(args) > len(f.Params) {
			if required == len(f.Params) {
				return nil, fmt.Errorf("function expects %d arguments, got %d", len(f.Params), len(args))
			}
			return nil, fmt.Errorf("function expects %d to %d arguments, got %d", required, len(f.Params), len(args))
		}
		callEnv := f.Env.child()
		callEnv.inFunc = true
		for i, name := range f.Params {
			var value Value
			if i < len(args) {
				value = args[i]
			} else if i < len(f.Defaults) && f.Defaults[i] != nil {
				v, err := evalExpr(f.Defaults[i], callEnv)
				if err != nil {
					return nil, err
				}
				value = v
			} else {
				if required == len(f.Params) {
					return nil, fmt.Errorf("function expects %d arguments, got %d", len(f.Params), len(args))
				}
				return nil, fmt.Errorf("function expects %d to %d arguments, got %d", required, len(f.Params), len(args))
			}
			callEnv.vars[name] = value
			callEnv.rememberKind(name, value)
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

func evalCallee(e ast.Expr, env *Env) (Value, error) {
	if m, ok := e.(*ast.MemberExpr); ok {
		obj, err := evalExpr(m.Target, env)
		if err != nil {
			return nil, err
		}
		if module, ok := obj.(*Module); ok {
			return module.Members[m.Name], nil
		}
		if dict, ok := obj.(Dict); ok {
			if isModuleNamespace(dict) {
				return dict[m.Name], nil
			}
		}
		if method := primitiveMethodValue(obj, m.Name, env); method != nil {
			return method, nil
		}
		return nil, fmt.Errorf("unknown method %s on %s", m.Name, runtimeKind(obj))
	}
	return evalExpr(e, env)
}

func primitiveMethodValue(receiver Value, name string, env *Env) Value {
	call := func(fn func([]Value) (Value, error)) Builtin {
		return Builtin(func(args []Value) (Value, error) {
			return fn(append([]Value{receiver}, args...))
		})
	}
	switch receiver.(type) {
	case string:
		switch name {
		case "upper":
			return call(func(args []Value) (Value, error) { return evalStringModuleCall("upcase", args, env) })
		case "lower":
			return call(func(args []Value) (Value, error) { return evalStringModuleCall("downcase", args, env) })
		case "blank?":
			return call(func(args []Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("string.blank? expects 0 arguments")
				}
				text, ok := args[0].(string)
				if !ok {
					return nil, fmt.Errorf("string.blank? expects string")
				}
				return strings.TrimSpace(text) == "", nil
			})
		case "present?":
			return call(func(args []Value) (Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("string.present? expects 0 arguments")
				}
				text, ok := args[0].(string)
				if !ok {
					return nil, fmt.Errorf("string.present? expects string")
				}
				return strings.TrimSpace(text) != "", nil
			})
		case "to_s":
			return Builtin(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("to_s expects 0 arguments")
				}
				return stringify(receiver), nil
			})
		case "to_i":
			return Builtin(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("to_i expects 0 arguments")
				}
				return toIntValue(receiver)
			})
		case "to_f", "to_number":
			return Builtin(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("%s expects 0 arguments", name)
				}
				return toNumberValue(receiver)
			})
		default:
			if isStandardStringCall(name) {
				return call(func(args []Value) (Value, error) { return evalStringModuleCall(name, args, env) })
			}
		}
	case *Array:
		if name == "to_s" {
			return Builtin(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("to_s expects 0 arguments")
				}
				return stringify(receiver), nil
			})
		}
		if isStandardArrayCall(name) {
			return call(func(args []Value) (Value, error) { return evalArrayModuleCall(name, args, env) })
		}
	case Dict:
		if name == "to_s" {
			return Builtin(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("to_s expects 0 arguments")
				}
				return stringify(receiver), nil
			})
		}
		if isStandardDictCall(name) {
			return call(func(args []Value) (Value, error) { return evalDictModuleCall(name, args, env) })
		}
	case Bytes:
		if name == "len" {
			return Builtin(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("bytes.len expects 0 arguments")
				}
				return int64(len(receiver.(Bytes).data)), nil
			})
		}
	case int64, float64, bool, nil:
		if name == "to_s" {
			return Builtin(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("to_s expects 0 arguments")
				}
				return stringify(receiver), nil
			})
		}
		if name == "to_i" {
			return Builtin(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("to_i expects 0 arguments")
				}
				return toIntValue(receiver)
			})
		}
		if name == "to_f" || name == "to_number" {
			return Builtin(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("%s expects 0 arguments", name)
				}
				return toNumberValue(receiver)
			})
		}
	}
	return nil
}

func isModuleNamespace(dict Dict) bool {
	marker, ok := dict["__module_namespace"].(bool)
	return ok && marker
}

func classOf(value Value) Value {
	switch value.(type) {
	case nil:
		return primitiveClass("Nil")
	case bool:
		return primitiveClass("Boolean")
	case int64, float64:
		return primitiveClass("Number")
	case string:
		return primitiveClass("String")
	case *Array:
		return primitiveClass("Array")
	case Dict:
		return primitiveClass("Dict")
	default:
		return nil
	}
}

func primitiveClass(name string) Dict {
	return Dict{"__module_namespace": true, "__class_name": name, "name": name}
}

func toIntValue(value Value) (Value, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("to_i: invalid integer")
		}
		return i, nil
	default:
		return nil, fmt.Errorf("to_i expects number or string")
	}
}

func toNumberValue(value Value) (Value, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case float64:
		return v, nil
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("to_number: invalid number")
		}
		if mathpkg.Trunc(f) == f {
			return int64(f), nil
		}
		return f, nil
	default:
		return nil, fmt.Errorf("to_number expects number or string")
	}
}

func evalBinary(b *ast.BinaryExpr, env *Env) (Value, error) {
	l, err := evalExpr(b.Left, env)
	if err != nil {
		return nil, err
	}
	if b.Op.Lexeme == "and" {
		if !truthy(l) {
			return false, nil
		}
		r, err := evalExpr(b.Right, env)
		if err != nil {
			return nil, err
		}
		return truthy(r), nil
	}
	if b.Op.Lexeme == "or" {
		if truthy(l) {
			return true, nil
		}
		r, err := evalExpr(b.Right, env)
		if err != nil {
			return nil, err
		}
		return truthy(r), nil
	}
	r, err := evalExpr(b.Right, env)
	if err != nil {
		return nil, err
	}
	switch b.Op.Lexeme {
	case "==":
		ok, err := equalStrict(l, r)
		return ok, err
	case "!=":
		ok, err := equalStrict(l, r)
		return !ok, err
	case "<", "<=", ">", ">=":
		return compare(b.Op.Lexeme, l, r)
	case "&", "|", "^", "<<", ">>":
		li, lok := numberAsInt(l)
		ri, rok := numberAsInt(r)
		if !lok || !rok {
			return nil, fmt.Errorf("%s expects integers", b.Op.Lexeme)
		}
		switch b.Op.Lexeme {
		case "&":
			return li & ri, nil
		case "|":
			return li | ri, nil
		case "^":
			return li ^ ri, nil
		case "<<":
			if ri < 0 {
				return nil, &raisedSignal{value: "<< : negative shift count"}
			}
			if ri >= 64 {
				return int64(0), nil
			}
			return li << uint(ri), nil
		case ">>":
			if ri < 0 {
				return nil, &raisedSignal{value: ">> : negative shift count"}
			}
			if ri >= 64 {
				if li < 0 {
					return int64(-1), nil
				}
				return int64(0), nil
			}
			return li >> uint(ri), nil
		}
	}
	if b.Op.Lexeme != "+" {
		return evalNumeric(b.Op.Lexeme, l, r)
	}
	if lb, ok := l.(*Bytes); ok {
		if rb, ok := r.(*Bytes); ok {
			out := make([]byte, len(lb.data)+len(rb.data))
			copy(out, lb.data)
			copy(out[len(lb.data):], rb.data)
			return &Bytes{data: out}, nil
		}
	}
	if ls, ok := l.(string); ok {
		if rs, ok := r.(string); ok {
			return ls + rs, nil
		}
		return nil, fmt.Errorf("%d:%d: + expects numbers, strings, or bytes of the same kind", b.Op.Line, b.Op.Col)
	}
	if _, ok := r.(string); ok {
		return nil, fmt.Errorf("%d:%d: + expects numbers, strings, or bytes of the same kind", b.Op.Line, b.Op.Col)
	}
	lf, lok := asFloat(l)
	rf, rok := asFloat(r)
	if !lok || !rok {
		return nil, fmt.Errorf("%d:%d: + expects numbers, strings, or bytes of the same kind", b.Op.Line, b.Op.Col)
	}
	if li, ok := l.(int64); ok {
		if ri, ok := r.(int64); ok {
			return li + ri, nil
		}
	}
	return lf + rf, nil
}

func equal(l, r Value) bool {
	ok, _ := equalValue(l, r, map[visitPair]bool{})
	return ok
}

func equalStrict(l, r Value) (bool, error) {
	return equalValue(l, r, map[visitPair]bool{})
}

type visitPair struct {
	left  uintptr
	right uintptr
}

func visitKey(l, r Value) visitPair {
	return visitPair{left: reflect.ValueOf(l).Pointer(), right: reflect.ValueOf(r).Pointer()}
}

func equalValue(l, r Value, seen map[visitPair]bool) (bool, error) {
	switch lv := l.(type) {
	case nil:
		return r == nil, nil
	case bool:
		rv, ok := r.(bool)
		return ok && lv == rv, nil
	case int64:
		switch rv := r.(type) {
		case int64:
			return lv == rv, nil
		case float64:
			return float64(lv) == rv, nil
		}
	case float64:
		switch rv := r.(type) {
		case int64:
			return lv == float64(rv), nil
		case float64:
			return lv == rv, nil
		}
	case string:
		rv, ok := r.(string)
		return ok && lv == rv, nil
	case Dict:
		rv, ok := r.(Dict)
		if !ok {
			return false, nil
		}
		return deepEqualValue(lv, rv, seen)
	case *Array:
		rv, ok := r.(*Array)
		if !ok {
			return false, nil
		}
		return deepEqualValue(lv, rv, seen)
	case *Function:
		return lv == r, nil
	case *ErrorValue:
		return lv == r, nil
	case *Module:
		return lv == r, nil
	}
	return false, nil
}

func runtimeKind(v Value) string {
	switch x := v.(type) {
	case nil:
		return "nil"
	case bool:
		return "bool"
	case int64:
		return "number"
	case float64:
		return "number"
	case string:
		return "string"
	case *Bytes:
		return "bytes"
	case *Array:
		return "array"
	case Dict:
		if isModuleNamespace(x) {
			if name, ok := x["__class_name"].(string); ok {
				return "class " + name
			}
		}
		return "dict"
	case *Function:
		return "function"
	case Builtin:
		return "function"
	case *ErrorValue:
		return "error"
	case *Module:
		return "package"
	default:
		return "object"
	}
}

func deepEqual(l, r Value) bool {
	ok, _ := deepEqualValue(l, r, map[visitPair]bool{})
	return ok
}

func deepEqualValue(l, r Value, seen map[visitPair]bool) (bool, error) {
	switch lv := l.(type) {
	case *Array:
		rv, ok := r.(*Array)
		if !ok || len(lv.items) != len(rv.items) {
			return false, nil
		}
		pair := visitKey(lv, rv)
		if seen[pair] {
			return false, fmt.Errorf("cyclic equality is invalid")
		}
		seen[pair] = true
		defer delete(seen, pair)
		for i := range lv.items {
			ok, err := equalValue(lv.items[i], rv.items[i], seen)
			if err != nil || !ok {
				return ok, err
			}
		}
		return true, nil
	case Dict:
		rv, ok := r.(Dict)
		if !ok || len(lv) != len(rv) {
			return false, nil
		}
		pair := visitKey(lv, rv)
		if seen[pair] {
			return false, fmt.Errorf("cyclic equality is invalid")
		}
		seen[pair] = true
		defer delete(seen, pair)
		for key, leftValue := range lv {
			rightValue, ok := rv[key]
			if !ok {
				return false, nil
			}
			ok, err := equalValue(leftValue, rightValue, seen)
			if err != nil || !ok {
				return ok, err
			}
		}
		return true, nil
	default:
		return equalValue(l, r, seen)
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
			close := interp.FindExprEnd(s, i)
			if close < 0 {
				return "", fmt.Errorf("unclosed interpolation")
			}
			expr := strings.TrimSpace(s[i+1 : close])
			if expr == "" {
				return "", fmt.Errorf("empty interpolation")
			}
			v, err := evalInterpolationExpr(expr, env)
			if err != nil {
				return "", err
			}
			out.WriteString(stringify(v))
			i = close + 1
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
	prog, _, err := parser.Parse(toks)
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
	return stringifyValue(v, map[uintptr]bool{})
}

func stringifyValue(v Value, seen map[uintptr]bool) string {
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
		return x.Message
	case *Array:
		key := reflect.ValueOf(x).Pointer()
		if seen[key] {
			return "<cycle>"
		}
		seen[key] = true
		defer delete(seen, key)
		parts := make([]string, 0, len(x.items))
		for _, item := range x.items {
			parts = append(parts, stringifyValue(item, seen))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case Dict:
		if isModuleNamespace(x) {
			if name, ok := x["__class_name"].(string); ok {
				return "<class " + name + ">"
			}
		}
		key := reflect.ValueOf(x).Pointer()
		if seen[key] {
			return "<cycle>"
		}
		seen[key] = true
		defer delete(seen, key)
		parts := make([]string, 0, len(x))
		for k, item := range x {
			if strings.HasPrefix(k, "__") {
				continue
			}
			parts = append(parts, k+": "+stringifyValue(item, seen))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case *Tuple:
		parts := make([]string, 0, len(x.items))
		for _, item := range x.items {
			parts = append(parts, stringifyValue(item, seen))
		}
		return strings.Join(parts, ", ")
	case *Module:
		return "<package " + x.Name + ">"
	case *Function:
		if x.Name != "" {
			return "<function " + x.Name + ">"
		}
		return "<function>"
	case Builtin:
		return "<function>"
	case *Bytes:
		return fmt.Sprintf("<bytes:%d>", len(x.data))
	default:
		return fmt.Sprintf("%v", x)
	}
}

// v0.24 builtins
var tyaRng = mathrand.New(mathrand.NewSource(time.Now().UnixNano()))

func registerV24Builtins(env *Env) {
	env.set("time_now", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("time_now expects 0 arguments")
		}
		return float64(time.Now().UnixNano()) / 1e9, nil
	}))
	env.set("time_sleep", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("time_sleep expects 1 argument")
		}
		secs, ok := numberAsFloat(args[0])
		if !ok {
			return nil, fmt.Errorf("time_sleep expects number")
		}
		if secs < 0 {
			return nil, &raisedSignal{value: "time.sleep: negative duration"}
		}
		time.Sleep(time.Duration(secs * float64(time.Second)))
		return nil, nil
	}))
	env.set("time_format", Builtin(func(args []Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, fmt.Errorf("time_format expects 1 or 2 arguments")
		}
		secs, ok := numberAsFloat(args[0])
		if !ok {
			return nil, fmt.Errorf("time_format expects number")
		}
		layout := "iso"
		if len(args) == 2 {
			s, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("time_format layout must be string")
			}
			layout = s
		}
		t := time.Unix(int64(secs), int64((secs-mathpkg.Floor(secs))*1e9)).UTC()
		switch layout {
		case "iso":
			return t.Format("2006-01-02T15:04:05Z"), nil
		case "date":
			return t.Format("2006-01-02"), nil
		case "time":
			return t.Format("15:04:05"), nil
		case "unix":
			return strconv.FormatInt(t.Unix(), 10), nil
		}
		return nil, &raisedSignal{value: "time.format: unknown layout"}
	}))
	env.set("time_parse", Builtin(func(args []Value) (Value, error) {
		s, err := oneString("time_parse", args)
		if err != nil {
			return nil, err
		}
		layouts := []string{"2006-01-02T15:04:05Z", "2006-01-02"}
		for _, l := range layouts {
			if t, err := time.Parse(l, s); err == nil {
				return float64(t.Unix()), nil
			}
		}
		return nil, &raisedSignal{value: "time.parse: invalid timestamp"}
	}))
	env.set("time_since", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("time_since expects 1 argument")
		}
		t, ok := numberAsFloat(args[0])
		if !ok {
			return nil, fmt.Errorf("time_since expects number")
		}
		return float64(time.Now().UnixNano())/1e9 - t, nil
	}))

	env.set("random_seed", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("random_seed expects 1 argument")
		}
		var seed int64
		switch v := args[0].(type) {
		case int64:
			seed = v
		case float64:
			seed = int64(v)
		case string:
			h := uint64(14695981039346656037)
			for _, b := range []byte(v) {
				h ^= uint64(b)
				h *= 1099511628211
			}
			seed = int64(h)
		default:
			return nil, fmt.Errorf("random_seed expects int or string")
		}
		tyaRng = mathrand.New(mathrand.NewSource(seed))
		return nil, nil
	}))
	env.set("random_int", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("random_int expects 2 arguments")
		}
		mn, ok1 := numberAsInt(args[0])
		mx, ok2 := numberAsInt(args[1])
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("random_int expects ints")
		}
		if mx < mn {
			return nil, &raisedSignal{value: "random.int: max < min"}
		}
		return mn + tyaRng.Int63n(mx-mn+1), nil
	}))
	env.set("random_float", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("random_float expects 0 arguments")
		}
		return tyaRng.Float64(), nil
	}))

	addMath := func(name string, fn func(float64) float64) {
		env.set(name, Builtin(func(args []Value) (Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("%s expects 1 argument", name)
			}
			x, ok := numberAsFloat(args[0])
			if !ok {
				return nil, fmt.Errorf("%s expects number", name)
			}
			return fn(x), nil
		}))
	}
	env.set("math_sqrt", Builtin(func(args []Value) (Value, error) {
		x, _ := numberAsFloat(args[0])
		if x < 0 {
			return nil, &raisedSignal{value: "math.sqrt: negative argument"}
		}
		return mathpkg.Sqrt(x), nil
	}))
	env.set("math_pow", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("math_pow expects 2 arguments")
		}
		x, _ := numberAsFloat(args[0])
		y, _ := numberAsFloat(args[1])
		return mathpkg.Pow(x, y), nil
	}))
	addMath("math_floor", mathpkg.Floor)
	addMath("math_ceil", mathpkg.Ceil)
	env.set("math_round", Builtin(func(args []Value) (Value, error) {
		x, _ := numberAsFloat(args[0])
		if x >= 0 {
			return mathpkg.Floor(x + 0.5), nil
		}
		return -mathpkg.Floor(-x + 0.5), nil
	}))
	addMath("math_trunc", mathpkg.Trunc)
	addLog := func(name string, fn func(float64) float64) {
		env.set(name, Builtin(func(args []Value) (Value, error) {
			x, _ := numberAsFloat(args[0])
			if x <= 0 {
				return nil, &raisedSignal{value: "math: non-positive argument to log"}
			}
			return fn(x), nil
		}))
	}
	addLog("math_log", mathpkg.Log)
	addLog("math_log2", mathpkg.Log2)
	addLog("math_log10", mathpkg.Log10)
	addMath("math_exp", mathpkg.Exp)
	addMath("math_sin", mathpkg.Sin)
	addMath("math_cos", mathpkg.Cos)
	addMath("math_tan", mathpkg.Tan)
	addMath("math_asin", mathpkg.Asin)
	addMath("math_acos", mathpkg.Acos)
	addMath("math_atan", mathpkg.Atan)
	env.set("math_atan2", Builtin(func(args []Value) (Value, error) {
		y, _ := numberAsFloat(args[0])
		x, _ := numberAsFloat(args[1])
		return mathpkg.Atan2(y, x), nil
	}))

	env.set("process_run", Builtin(func(args []Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, fmt.Errorf("process_run expects 1 or 2 arguments")
		}
		opts := Dict{}
		if len(args) == 2 && args[1] != nil {
			var ok bool
			opts, ok = args[1].(Dict)
			if !ok {
				return nil, &raisedSignal{value: "process.run: options must be a dictionary"}
			}
		}
		allowed := map[string]bool{"cwd": true, "env": true, "clear_env": true, "stdin": true, "input": true, "capture_stdout": true, "capture_stderr": true, "timeout": true, "shell": true}
		for key := range opts {
			if !allowed[key] {
				return nil, &raisedSignal{value: "process.run: unknown option " + key}
			}
		}
		shell := false
		if v, has := opts["shell"]; has {
			var ok bool
			shell, ok = v.(bool)
			if !ok {
				return nil, &raisedSignal{value: "process.run: shell must be bool"}
			}
		}
		var cmdArgs []string
		switch command := args[0].(type) {
		case string:
			if command == "" {
				return nil, &raisedSignal{value: "process.run: command must be non-empty"}
			}
			if !shell {
				return nil, &raisedSignal{value: "process.run: string command requires shell option"}
			}
			cmdArgs = []string{"sh", "-c", command}
		case *Array:
			if len(command.items) == 0 {
				return nil, &raisedSignal{value: "process.run: command must be a non-empty array"}
			}
			cmdArgs = make([]string, len(command.items))
			for i, v := range command.items {
				s, ok := v.(string)
				if !ok {
					return nil, &raisedSignal{value: "process.run: command items must be strings"}
				}
				cmdArgs[i] = s
			}
		default:
			return nil, &raisedSignal{value: "process.run: command must be a string or array"}
		}
		var ctx context.Context = context.Background()
		var cancel context.CancelFunc
		if v, has := opts["timeout"]; has {
			secs, ok := numberAsFloat(v)
			if !ok {
				return nil, &raisedSignal{value: "process.run: timeout must be a number"}
			}
			ctx, cancel = context.WithTimeout(ctx, time.Duration(secs*float64(time.Second)))
			defer cancel()
		}
		cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
		var stdoutBuf, stderrBuf bytes.Buffer
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf
		if cwd, has := opts["cwd"].(string); has {
			cmd.Dir = cwd
		}
		input := opts["stdin"]
		if input == nil {
			input = opts["input"]
		}
		if input != nil {
			switch v := input.(type) {
			case string:
				cmd.Stdin = strings.NewReader(v)
			case *Bytes:
				cmd.Stdin = bytes.NewReader(v.data)
			default:
				return nil, &raisedSignal{value: "process.run: stdin must be string or bytes"}
			}
		}
		if clear, has := opts["clear_env"].(bool); has && clear {
			cmd.Env = []string{}
		} else {
			cmd.Env = os.Environ()
		}
		if envDict, has := opts["env"].(Dict); has {
			for k, v := range envDict {
				s, ok := v.(string)
				if !ok {
					return nil, &raisedSignal{value: "process.run: env values must be strings"}
				}
				cmd.Env = append(cmd.Env, k+"="+s)
			}
		}
		err := cmd.Run()
		exitCode := 0
		timedOut := ctx.Err() == context.DeadlineExceeded
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				return nil, &raisedSignal{value: "process.run: " + err.Error()}
			}
		}
		out := Dict{}
		out["status"] = int64(exitCode)
		out["exit_code"] = int64(exitCode)
		out["success"] = exitCode == 0
		out["stdout"] = stdoutBuf.String()
		out["stderr"] = stderrBuf.String()
		out["timed_out"] = timedOut
		return out, nil
	}))
	env.set("process_exec", Builtin(func(args []Value) (Value, error) {
		return nil, &raisedSignal{value: "process.exec: unsupported on this runtime"}
	}))

	digestInput := func(name string, args []Value) ([]byte, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("%s expects 1 argument", name)
		}
		switch v := args[0].(type) {
		case string:
			return []byte(v), nil
		case *Bytes:
			return v.data, nil
		}
		return nil, &raisedSignal{value: name + ": argument must be a string or bytes"}
	}
	env.set("digest_md5", Builtin(func(args []Value) (Value, error) {
		data, err := digestInput("digest.md5", args)
		if err != nil {
			return nil, err
		}
		h := md5.Sum(data)
		return hex.EncodeToString(h[:]), nil
	}))
	env.set("digest_sha1", Builtin(func(args []Value) (Value, error) {
		data, err := digestInput("digest.sha1", args)
		if err != nil {
			return nil, err
		}
		h := sha1.Sum(data)
		return hex.EncodeToString(h[:]), nil
	}))
	env.set("digest_sha256", Builtin(func(args []Value) (Value, error) {
		data, err := digestInput("digest.sha256", args)
		if err != nil {
			return nil, err
		}
		h := sha256.Sum256(data)
		return hex.EncodeToString(h[:]), nil
	}))
	env.set("digest_sha384", Builtin(func(args []Value) (Value, error) {
		data, err := digestInput("digest.sha384", args)
		if err != nil {
			return nil, err
		}
		h := sha512.Sum384(data)
		return hex.EncodeToString(h[:]), nil
	}))
	env.set("digest_sha512", Builtin(func(args []Value) (Value, error) {
		data, err := digestInput("digest.sha512", args)
		if err != nil {
			return nil, err
		}
		h := sha512.Sum512(data)
		return hex.EncodeToString(h[:]), nil
	}))

	env.set("secure_random_bytes", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("secure_random_bytes expects 1 argument")
		}
		n, ok := numberAsInt(args[0])
		if !ok || n < 0 || n > 4096 {
			return nil, &raisedSignal{value: "secure_random.bytes: count out of range"}
		}
		buf := make([]byte, n)
		if _, err := rand.Read(buf); err != nil {
			return nil, &raisedSignal{value: "secure_random: entropy unavailable"}
		}
		return &Bytes{data: buf}, nil
	}))
	env.set("secure_random_int", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("secure_random_int expects 2 arguments")
		}
		mn, ok1 := numberAsInt(args[0])
		mx, ok2 := numberAsInt(args[1])
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("secure_random_int expects ints")
		}
		if mx < mn {
			return nil, &raisedSignal{value: "secure_random.int: max < min"}
		}
		rng := uint64(mx - mn + 1)
		threshold := uint64(0)
		if rng != 0 {
			threshold = (^uint64(0) - rng + 1) % rng
		}
		for {
			var b [8]byte
			if _, err := rand.Read(b[:]); err != nil {
				return nil, &raisedSignal{value: "secure_random.int: entropy unavailable"}
			}
			r := binary.BigEndian.Uint64(b[:])
			if r >= threshold {
				return mn + int64(r%rng), nil
			}
		}
	}))
	registerV25Builtins(env)
	registerV41Builtins(env)
}

func registerV41Builtins(env *Env) {
	// runtime_gc_stats() — eval has no real GC; return zeros for parity with C runtime.
	env.set("runtime_gc_stats", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("runtime_gc_stats expects 0 arguments")
		}
		out := Dict{
			"alloc_count":   float64(0),
			"alloc_bytes":   float64(0),
			"freed_count":   float64(0),
			"freed_bytes":   float64(0),
			"live_count":    float64(0),
			"live_bytes":    float64(0),
			"collect_count": float64(0),
			"threshold":     float64(0),
		}
		return out, nil
	}))
	// runtime_gc_collect() — eval has no real GC; no-op for parity.
	env.set("runtime_gc_collect", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("runtime_gc_collect expects 0 arguments")
		}
		return nil, nil
	}))
	// v0.42 channel: eval is single-threaded; the stubs below raise so
	// programs that require concurrency are routed through the C runtime.
	stub := func(name string, want int) Builtin {
		return func(args []Value) (Value, error) {
			if len(args) != want {
				return nil, fmt.Errorf("%s expects %d arguments", name, want)
			}
			return nil, &raisedSignal{value: name + ": eval interpreter does not support concurrency; use tya run"}
		}
	}
	env.set("channel_new", stub("channel_new", 1))
	env.set("channel_send", stub("channel_send", 2))
	env.set("channel_receive", stub("channel_receive", 1))
	env.set("channel_receive_timeout", stub("channel_receive_timeout", 2))
	env.set("channel_close", stub("channel_close", 1))
	env.set("channel_closed_p", stub("channel_closed_p", 1))
	env.set("channel_select", stub("channel_select", 1))
	env.set("task_cancel", stub("task_cancel", 1))
	env.set("task_is_cancelled_p", stub("task_is_cancelled_p", 1))
	env.set("task_current", stub("task_current", 0))
	env.set("sync_mutex_new", stub("sync_mutex_new", 0))
	env.set("sync_lock", stub("sync_lock", 1))
	env.set("sync_unlock", stub("sync_unlock", 1))
	env.set("sync_atomic_integer_new", stub("sync_atomic_integer_new", 1))
	env.set("sync_atomic_integer_add", stub("sync_atomic_integer_add", 2))
	env.set("sync_atomic_integer_load", stub("sync_atomic_integer_load", 1))
	env.set("sync_atomic_integer_store", stub("sync_atomic_integer_store", 2))
	env.set("sync_atomic_integer_cas", stub("sync_atomic_integer_cas", 3))
	env.set("sync_wait_group_new", stub("sync_wait_group_new", 0))
	env.set("sync_wait_group_add", stub("sync_wait_group_add", 2))
	env.set("sync_wait_group_done", stub("sync_wait_group_done", 1))
	env.set("sync_wait_group_wait", stub("sync_wait_group_wait", 1))
}

func registerV25Builtins(env *Env) {
	env.set("bytes", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("bytes expects 1 argument")
		}
		arr, ok := args[0].(*Array)
		if !ok {
			return nil, &raisedSignal{value: "bytes: argument must be an array of ints"}
		}
		out := make([]byte, len(arr.items))
		for i, item := range arr.items {
			n, ok := numberAsInt(item)
			if !ok {
				return nil, &raisedSignal{value: "bytes: items must be ints"}
			}
			if n < 0 || n > 255 {
				return nil, &raisedSignal{value: "bytes: item out of 0..255"}
			}
			out[i] = byte(n)
		}
		return &Bytes{data: out}, nil
	}))
	env.set("bytes_of", Builtin(func(args []Value) (Value, error) {
		s, err := oneString("bytes_of", args)
		if err != nil {
			return nil, err
		}
		return &Bytes{data: []byte(s)}, nil
	}))
	env.set("bytes_text", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("bytes_text expects 1 argument")
		}
		b, ok := args[0].(*Bytes)
		if !ok {
			return nil, &raisedSignal{value: "bytes_text: argument must be bytes"}
		}
		for _, c := range b.data {
			if c == 0 {
				return nil, &raisedSignal{value: "bytes_text: NUL byte not allowed"}
			}
		}
		if !utf8.Valid(b.data) {
			return nil, &raisedSignal{value: "bytes_text: invalid UTF-8"}
		}
		return string(b.data), nil
	}))
	env.set("bytes_array", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("bytes_array expects 1 argument")
		}
		b, ok := args[0].(*Bytes)
		if !ok {
			return nil, &raisedSignal{value: "bytes_array: argument must be bytes"}
		}
		out := &Array{items: make([]Value, len(b.data))}
		for i, c := range b.data {
			out.items[i] = int64(c)
		}
		return out, nil
	}))
	env.set("bytes_concat", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("bytes_concat expects 2 arguments")
		}
		a, aok := args[0].(*Bytes)
		b, bok := args[1].(*Bytes)
		if !aok || !bok {
			return nil, &raisedSignal{value: "bytes_concat: arguments must be bytes"}
		}
		out := make([]byte, len(a.data)+len(b.data))
		copy(out, a.data)
		copy(out[len(a.data):], b.data)
		return &Bytes{data: out}, nil
	}))
	env.set("bytes_slice", Builtin(func(args []Value) (Value, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("bytes_slice expects 3 arguments")
		}
		b, ok := args[0].(*Bytes)
		if !ok {
			return nil, &raisedSignal{value: "bytes_slice: first argument must be bytes"}
		}
		s, sok := numberAsInt(args[1])
		e, eok := numberAsInt(args[2])
		if !sok || !eok {
			return nil, &raisedSignal{value: "bytes_slice: indices must be ints"}
		}
		if s < 0 || e < s || int(e) > len(b.data) {
			return nil, &raisedSignal{value: "bytes_slice: index out of range"}
		}
		return &Bytes{data: append([]byte{}, b.data[s:e]...)}, nil
	}))
	env.set("file_read_bytes", Builtin(func(args []Value) (Value, error) {
		path, err := oneString("file_read_bytes", args)
		if err != nil {
			return nil, err
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil, &raisedSignal{value: rerr.Error()}
		}
		return &Bytes{data: data}, nil
	}))
	env.set("file_write_bytes", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("file_write_bytes expects 2 arguments")
		}
		path, ok := args[0].(string)
		if !ok {
			return nil, &raisedSignal{value: "file.write_bytes: path must be a string"}
		}
		b, ok := args[1].(*Bytes)
		if !ok {
			return nil, &raisedSignal{value: "file.write_bytes: data must be bytes"}
		}
		if err := os.WriteFile(path, b.data, 0644); err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		return nil, nil
	}))
	env.set("stderr_write", Builtin(func(args []Value) (Value, error) {
		text, err := oneString("stderr_write", args)
		if err != nil {
			return nil, err
		}
		_, _ = fmt.Fprint(os.Stderr, text)
		return nil, nil
	}))
	env.set("file_append", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("file_append expects 2 arguments")
		}
		path, ok := args[0].(string)
		if !ok {
			return nil, &raisedSignal{value: "file.append: path must be a string"}
		}
		text, ok := args[1].(string)
		if !ok {
			return nil, &raisedSignal{value: "file.append: text must be a string"}
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		if _, err := f.WriteString(text); err != nil {
			_ = f.Close()
			return nil, &raisedSignal{value: err.Error()}
		}
		if err := f.Close(); err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		return nil, nil
	}))
	env.set("compress_gzip", Builtin(func(args []Value) (Value, error) {
		data, err := valueBytes("compress.gzip", args)
		if err != nil {
			return nil, err
		}
		var out bytes.Buffer
		w := gzip.NewWriter(&out)
		if _, err := w.Write(data); err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		if err := w.Close(); err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		return &Bytes{data: out.Bytes()}, nil
	}))
	env.set("compress_gunzip", Builtin(func(args []Value) (Value, error) {
		data, err := valueBytes("compress.gunzip", args)
		if err != nil {
			return nil, err
		}
		r, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, &raisedSignal{value: "compress: invalid compressed data"}
		}
		out, err := io.ReadAll(r)
		_ = r.Close()
		if err != nil {
			return nil, &raisedSignal{value: "compress: invalid compressed data"}
		}
		return &Bytes{data: out}, nil
	}))
	env.set("compress_zlib", Builtin(func(args []Value) (Value, error) {
		data, err := valueBytes("compress.zlib", args)
		if err != nil {
			return nil, err
		}
		var out bytes.Buffer
		w := zlib.NewWriter(&out)
		if _, err := w.Write(data); err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		if err := w.Close(); err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		return &Bytes{data: out.Bytes()}, nil
	}))
	env.set("compress_unzlib", Builtin(func(args []Value) (Value, error) {
		data, err := valueBytes("compress.unzlib", args)
		if err != nil {
			return nil, err
		}
		r, err := zlib.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, &raisedSignal{value: "compress: invalid compressed data"}
		}
		out, err := io.ReadAll(r)
		_ = r.Close()
		if err != nil {
			return nil, &raisedSignal{value: "compress: invalid compressed data"}
		}
		return &Bytes{data: out}, nil
	}))
	env.set("io_stdin", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("io_stdin expects 0 arguments")
		}
		return &IOStream{file: os.Stdin, readable: true, borrowed: true}, nil
	}))
	env.set("io_stdout", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("io_stdout expects 0 arguments")
		}
		return &IOStream{file: os.Stdout, writable: true, borrowed: true}, nil
	}))
	env.set("io_stderr", Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("io_stderr expects 0 arguments")
		}
		return &IOStream{file: os.Stderr, writable: true, borrowed: true}, nil
	}))
	env.set("io_open", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("io_open expects 2 arguments")
		}
		path, ok := args[0].(string)
		if !ok {
			return nil, &raisedSignal{value: "io.open: path must be a string"}
		}
		mode, ok := args[1].(string)
		if !ok {
			return nil, &raisedSignal{value: "io.open: mode must be a string"}
		}
		flag := 0
		readable := strings.Contains(mode, "r")
		writable := strings.Contains(mode, "w") || strings.Contains(mode, "a")
		switch {
		case strings.Contains(mode, "a"):
			flag = os.O_CREATE | os.O_WRONLY | os.O_APPEND
		case strings.Contains(mode, "w"):
			flag = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
		case strings.Contains(mode, "r"):
			flag = os.O_RDONLY
		default:
			return nil, &raisedSignal{value: "io.open: invalid mode"}
		}
		f, err := os.OpenFile(path, flag, 0644)
		if err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		return &IOStream{file: f, binary: strings.Contains(mode, "b"), readable: readable, writable: writable}, nil
	}))
	streamArg := func(name string, args []Value) (*IOStream, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("%s expects stream argument", name)
		}
		s, ok := args[0].(*IOStream)
		if !ok || s == nil {
			return nil, &raisedSignal{value: name + ": argument must be a stream"}
		}
		if s.closed {
			return nil, &raisedSignal{value: name + ": stream is closed"}
		}
		return s, nil
	}
	env.set("io_stream_read", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("io_stream_read expects 2 arguments")
		}
		s, err := streamArg("io.read", args)
		if err != nil {
			return nil, err
		}
		if !s.readable {
			return nil, &raisedSignal{value: "io.read: stream is not readable"}
		}
		size, ok := numberAsInt(args[1])
		if !ok || size < 0 {
			return nil, &raisedSignal{value: "io.read: size must be non-negative"}
		}
		buf := make([]byte, int(size))
		n, rerr := s.file.Read(buf)
		if rerr != nil && rerr != io.EOF {
			return nil, &raisedSignal{value: rerr.Error()}
		}
		buf = buf[:n]
		if s.binary {
			return &Bytes{data: buf}, nil
		}
		return string(buf), nil
	}))
	env.set("io_stream_read_line", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("io_stream_read_line expects 1 argument")
		}
		s, err := streamArg("io.read_line", args)
		if err != nil {
			return nil, err
		}
		if !s.readable {
			return nil, &raisedSignal{value: "io.read_line: stream is not readable"}
		}
		var buf []byte
		tmp := make([]byte, 1)
		for {
			n, rerr := s.file.Read(tmp)
			if n > 0 {
				buf = append(buf, tmp[0])
				if tmp[0] == '\n' {
					break
				}
			}
			if rerr == io.EOF {
				break
			}
			if rerr != nil {
				return nil, &raisedSignal{value: rerr.Error()}
			}
		}
		if len(buf) == 0 {
			return nil, nil
		}
		if s.binary {
			return &Bytes{data: buf}, nil
		}
		return string(buf), nil
	}))
	env.set("io_stream_eof", Builtin(func(args []Value) (Value, error) {
		_, err := streamArg("io.eof?", args)
		if err != nil {
			return nil, err
		}
		return false, nil
	}))
	env.set("io_stream_write", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("io_stream_write expects 2 arguments")
		}
		s, err := streamArg("io.write", args)
		if err != nil {
			return nil, err
		}
		if !s.writable {
			return nil, &raisedSignal{value: "io.write: stream is not writable"}
		}
		var data []byte
		if b, ok := args[1].(*Bytes); ok {
			data = b.data
		} else {
			data = []byte(stringify(args[1]))
		}
		n, werr := s.file.Write(data)
		if werr != nil {
			return nil, &raisedSignal{value: werr.Error()}
		}
		return int64(n), nil
	}))
	env.set("io_stream_flush", Builtin(func(args []Value) (Value, error) {
		s, err := streamArg("io.flush", args)
		if err != nil {
			return nil, err
		}
		return nil, s.file.Sync()
	}))
	env.set("io_stream_close", Builtin(func(args []Value) (Value, error) {
		s, err := streamArg("io.close", args)
		if err != nil {
			return nil, err
		}
		s.closed = true
		if s.borrowed {
			return nil, nil
		}
		return nil, s.file.Close()
	}))
	socketOptions := func(options Value) (bool, time.Duration) {
		opts, _ := options.(Dict)
		binary := false
		timeout := time.Duration(0)
		if opts != nil {
			if mode, ok := opts["mode"].(string); ok && mode == "binary" {
				binary = true
			}
			if seconds, ok := numberAsFloat(opts["timeout"]); ok && seconds > 0 {
				timeout = time.Duration(seconds * float64(time.Second))
			}
		}
		return binary, timeout
	}
	socketArg := func(name string, args []Value) (*TCPSocket, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("%s expects socket argument", name)
		}
		s, ok := args[0].(*TCPSocket)
		if !ok || s == nil || s.conn == nil {
			return nil, &raisedSignal{value: name + ": argument must be a socket"}
		}
		if s.closed {
			return nil, &raisedSignal{value: name + ": socket is closed"}
		}
		return s, nil
	}
	serverArg := func(name string, args []Value) (*TCPSocket, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("%s expects server argument", name)
		}
		s, ok := args[0].(*TCPSocket)
		if !ok || s == nil || s.listener == nil {
			return nil, &raisedSignal{value: name + ": argument must be a socket server"}
		}
		if s.closed {
			return nil, &raisedSignal{value: name + ": socket is closed"}
		}
		return s, nil
	}
	socketAddress := func(addr net.Addr) Dict {
		host := ""
		port := int64(0)
		if tcp, ok := addr.(*net.TCPAddr); ok {
			host = tcp.IP.String()
			port = int64(tcp.Port)
		} else if addr != nil {
			host, _, _ = net.SplitHostPort(addr.String())
		}
		return Dict{"host": host, "port": port}
	}
	env.set("socket_connect", Builtin(func(args []Value) (Value, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("socket_connect expects 3 arguments")
		}
		host, ok := args[0].(string)
		if !ok {
			return nil, &raisedSignal{value: "socket.connect: host must be a string"}
		}
		port, ok := numberAsInt(args[1])
		if !ok || port < 0 || port > 65535 {
			return nil, &raisedSignal{value: "socket.connect: invalid port"}
		}
		binary, timeout := socketOptions(args[2])
		address := net.JoinHostPort(host, strconv.FormatInt(port, 10))
		var conn net.Conn
		var err error
		if timeout > 0 {
			conn, err = net.DialTimeout("tcp", address, timeout)
		} else {
			conn, err = net.Dial("tcp", address)
		}
		if err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		return &TCPSocket{conn: conn, binary: binary, timeout: timeout}, nil
	}))
	env.set("socket_server_listen", Builtin(func(args []Value) (Value, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("socket_server_listen expects 3 arguments")
		}
		host, ok := args[0].(string)
		if !ok {
			return nil, &raisedSignal{value: "socket.listen: host must be a string"}
		}
		port, ok := numberAsInt(args[1])
		if !ok || port < 0 || port > 65535 {
			return nil, &raisedSignal{value: "socket.listen: invalid port"}
		}
		binary, timeout := socketOptions(args[2])
		listener, err := net.Listen("tcp", net.JoinHostPort(host, strconv.FormatInt(port, 10)))
		if err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		return &TCPSocket{listener: listener, binary: binary, timeout: timeout}, nil
	}))
	env.set("socket_server_accept", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("socket_server_accept expects 1 argument")
		}
		s, err := serverArg("socket.accept", args)
		if err != nil {
			return nil, err
		}
		if s.timeout > 0 {
			if tcp, ok := s.listener.(*net.TCPListener); ok {
				_ = tcp.SetDeadline(time.Now().Add(s.timeout))
			}
		}
		conn, err := s.listener.Accept()
		if err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		return &TCPSocket{conn: conn, binary: s.binary, timeout: s.timeout}, nil
	}))
	env.set("socket_read", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("socket_read expects 2 arguments")
		}
		s, err := socketArg("socket.read", args)
		if err != nil {
			return nil, err
		}
		size, ok := numberAsInt(args[1])
		if !ok || size < 0 {
			return nil, &raisedSignal{value: "socket.read: size must be non-negative"}
		}
		if s.timeout > 0 {
			_ = s.conn.SetReadDeadline(time.Now().Add(s.timeout))
		}
		buf := make([]byte, int(size))
		n, err := s.conn.Read(buf)
		if err != nil && err != io.EOF {
			return nil, &raisedSignal{value: err.Error()}
		}
		buf = buf[:n]
		if s.binary {
			return &Bytes{data: buf}, nil
		}
		return string(buf), nil
	}))
	env.set("socket_read_line", Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("socket_read_line expects 1 argument")
		}
		s, err := socketArg("socket.read_line", args)
		if err != nil {
			return nil, err
		}
		if s.timeout > 0 {
			_ = s.conn.SetReadDeadline(time.Now().Add(s.timeout))
		}
		var buf []byte
		tmp := make([]byte, 1)
		for {
			n, rerr := s.conn.Read(tmp)
			if n > 0 {
				buf = append(buf, tmp[0])
				if tmp[0] == '\n' {
					break
				}
			}
			if rerr == io.EOF {
				break
			}
			if rerr != nil {
				return nil, &raisedSignal{value: rerr.Error()}
			}
		}
		if len(buf) == 0 {
			return nil, nil
		}
		if s.binary {
			return &Bytes{data: buf}, nil
		}
		return string(buf), nil
	}))
	env.set("socket_write", Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("socket_write expects 2 arguments")
		}
		s, err := socketArg("socket.write", args)
		if err != nil {
			return nil, err
		}
		var data []byte
		if b, ok := args[1].(*Bytes); ok {
			data = b.data
		} else {
			data = []byte(stringify(args[1]))
		}
		if s.timeout > 0 {
			_ = s.conn.SetWriteDeadline(time.Now().Add(s.timeout))
		}
		n, err := s.conn.Write(data)
		if err != nil {
			return nil, &raisedSignal{value: err.Error()}
		}
		return int64(n), nil
	}))
	env.set("socket_close", Builtin(func(args []Value) (Value, error) {
		s, ok := args[0].(*TCPSocket)
		if !ok || s == nil || s.conn == nil || s.closed {
			return nil, nil
		}
		s.closed = true
		return nil, s.conn.Close()
	}))
	env.set("socket_closed", Builtin(func(args []Value) (Value, error) {
		s, ok := args[0].(*TCPSocket)
		return !ok || s == nil || s.closed, nil
	}))
	env.set("socket_local_address", Builtin(func(args []Value) (Value, error) {
		s, err := socketArg("socket.local_address", args)
		if err != nil {
			return nil, err
		}
		return socketAddress(s.conn.LocalAddr()), nil
	}))
	env.set("socket_remote_address", Builtin(func(args []Value) (Value, error) {
		s, err := socketArg("socket.remote_address", args)
		if err != nil {
			return nil, err
		}
		return socketAddress(s.conn.RemoteAddr()), nil
	}))
	env.set("socket_server_close", Builtin(func(args []Value) (Value, error) {
		s, ok := args[0].(*TCPSocket)
		if !ok || s == nil || s.listener == nil || s.closed {
			return nil, nil
		}
		s.closed = true
		return nil, s.listener.Close()
	}))
	env.set("socket_server_local_address", Builtin(func(args []Value) (Value, error) {
		s, err := serverArg("socket.server.local_address", args)
		if err != nil {
			return nil, err
		}
		return socketAddress(s.listener.Addr()), nil
	}))
}

func numberAsFloat(v Value) (float64, bool) {
	switch x := v.(type) {
	case int64:
		return float64(x), true
	case float64:
		return x, true
	}
	return 0, false
}

func valueBytes(name string, args []Value) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s expects 1 argument", name)
	}
	switch v := args[0].(type) {
	case string:
		return []byte(v), nil
	case *Bytes:
		return v.data, nil
	default:
		return nil, &raisedSignal{value: name + ": value must be a string or bytes"}
	}
}

func numberAsInt(v Value) (int64, bool) {
	switch x := v.(type) {
	case int64:
		return x, true
	case float64:
		if x != mathpkg.Trunc(x) {
			return 0, false
		}
		return int64(x), true
	}
	return 0, false
}
