package eval

import (
	"errors"
	"fmt"
	"io"

	"tya/internal/ast"
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

type Env struct {
	parent    *Env
	vars      map[string]Value
	kinds     map[string]string
	inFunc    bool
	runeCache map[string][]rune
}

var primitiveClasses = map[string]Dict{
	"Number":  {"__module_namespace": true, "__class_name": "Number", "name": "Number"},
	"String":  {"__module_namespace": true, "__class_name": "String", "name": "String"},
	"Bytes":   {"__module_namespace": true, "__class_name": "Bytes", "name": "Bytes"},
	"Array":   {"__module_namespace": true, "__class_name": "Array", "name": "Array"},
	"Dict":    {"__module_namespace": true, "__class_name": "Dict", "name": "Dict"},
	"Boolean": {"__module_namespace": true, "__class_name": "Boolean", "name": "Boolean"},
	"Nil":     {"__module_namespace": true, "__class_name": "Nil", "name": "Nil"},
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
