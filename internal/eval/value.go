package eval

import (
	"fmt"
	mathpkg "math"
	mathrand "math/rand"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"tya/internal/ast"
	"tya/internal/interp"
	"tya/internal/lexer"
	"tya/internal/parser"
)

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

type RegexValue struct {
	pattern string
	re      *regexp.Regexp
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

type boundMethod struct {
	method   Value
	receiver Value
	name     string
}

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

func regexError(message string, code string) *ErrorValue {
	return &ErrorValue{Message: message, Kind: "regex", Code: code, Data: Dict{}, Cause: nil}
}

func regexRaised(message string, code string) *raisedSignal {
	return &raisedSignal{value: regexError(message, code)}
}

func compileRegexValue(pattern string, options Dict) (*RegexValue, error) {
	prefix := ""
	for key, value := range options {
		flag, ok := value.(bool)
		if !ok {
			return nil, regexRaised("regex.compile: option "+key+" must be bool", "invalid_option_kind")
		}
		switch key {
		case "ignore_case":
			if flag {
				prefix += "i"
			}
		case "multi_line":
			if flag {
				prefix += "m"
			}
		case "dot_all":
			if flag {
				prefix += "s"
			}
		default:
			return nil, regexRaised("regex.compile: unknown option "+key, "unknown_option")
		}
	}
	expr := pattern
	if prefix != "" {
		expr = "(?" + prefix + ")" + pattern
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, regexRaised("regex.compile: invalid pattern", "invalid_pattern")
	}
	return &RegexValue{pattern: pattern, re: re}, nil
}

func regexObject(compiled *RegexValue) Dict {
	obj := Dict{"__module_namespace": true, "__regex": compiled}
	obj["match?"] = Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("regex.match? expects 1 argument")
		}
		text, ok := args[0].(string)
		if !ok {
			return nil, regexRaised("regex.match?: text must be a string", "invalid_text")
		}
		return compiled.re.FindStringIndex(text) != nil, nil
	})
	obj["find"] = Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("regex.find expects 1 argument")
		}
		text, ok := args[0].(string)
		if !ok {
			return nil, regexRaised("regex.find: text must be a string", "invalid_text")
		}
		return regexMatchDict(compiled.re, text, compiled.re.FindStringSubmatchIndex(text)), nil
	})
	obj["find_all"] = Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("regex.find_all expects 1 argument")
		}
		text, ok := args[0].(string)
		if !ok {
			return nil, regexRaised("regex.find_all: text must be a string", "invalid_text")
		}
		matches := compiled.re.FindAllStringSubmatchIndex(text, -1)
		items := make([]Value, 0, len(matches))
		for _, match := range matches {
			items = append(items, regexMatchDict(compiled.re, text, match))
		}
		return &Array{items: items}, nil
	})
	obj["split"] = Builtin(func(args []Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, fmt.Errorf("regex.split expects 1 or 2 arguments")
		}
		text, ok := args[0].(string)
		if !ok {
			return nil, regexRaised("regex.split: text must be a string", "invalid_text")
		}
		limit := -1
		if len(args) == 2 && args[1] != nil {
			n, ok := numberAsInt(args[1])
			if !ok {
				return nil, regexRaised("regex.split: limit must be an integer", "invalid_limit")
			}
			limit = int(n)
		}
		parts := compiled.re.Split(text, limit)
		items := make([]Value, 0, len(parts))
		for _, part := range parts {
			items = append(items, part)
		}
		return &Array{items: items}, nil
	})
	obj["replace"] = Builtin(func(args []Value) (Value, error) {
		if len(args) < 2 || len(args) > 3 {
			return nil, fmt.Errorf("regex.replace expects 2 or 3 arguments")
		}
		text, ok := args[0].(string)
		if !ok {
			return nil, regexRaised("regex.replace: text must be a string", "invalid_text")
		}
		replacement, ok := args[1].(string)
		if !ok {
			return nil, regexRaised("regex.replace: replacement must be a string", "invalid_replacement")
		}
		limit := -1
		if len(args) == 3 && args[2] != nil {
			n, ok := numberAsInt(args[2])
			if !ok {
				return nil, regexRaised("regex.replace: limit must be an integer", "invalid_limit")
			}
			limit = int(n)
		}
		return regexReplace(compiled.re, text, replacement, limit)
	})
	return obj
}

func regexMatchDict(re *regexp.Regexp, text string, match []int) Value {
	if match == nil {
		return nil
	}
	groups := make([]Value, 0, len(match)/2-1)
	for i := 2; i < len(match); i += 2 {
		if match[i] < 0 {
			groups = append(groups, nil)
			continue
		}
		groups = append(groups, text[match[i]:match[i+1]])
	}
	return Dict{
		"text":   text[match[0]:match[1]],
		"start":  int64(utf8.RuneCountInString(text[:match[0]])),
		"end":    int64(utf8.RuneCountInString(text[:match[1]])),
		"groups": &Array{items: groups},
	}
}

func regexReplace(re *regexp.Regexp, text string, replacement string, limit int) (Value, error) {
	matches := re.FindAllStringSubmatchIndex(text, -1)
	if limit >= 0 && len(matches) > limit {
		matches = matches[:limit]
	}
	var out strings.Builder
	last := 0
	for _, match := range matches {
		out.WriteString(text[last:match[0]])
		part, err := regexExpandReplacement(text, replacement, match)
		if err != nil {
			return nil, err
		}
		out.WriteString(part)
		last = match[1]
	}
	out.WriteString(text[last:])
	return out.String(), nil
}

func regexExpandReplacement(text string, replacement string, match []int) (string, error) {
	var out strings.Builder
	for i := 0; i < len(replacement); {
		if replacement[i] != '$' {
			out.WriteByte(replacement[i])
			i++
			continue
		}
		if i+1 < len(replacement) && replacement[i+1] == '$' {
			out.WriteByte('$')
			i += 2
			continue
		}
		if i+2 >= len(replacement) || replacement[i+1] != '{' {
			return "", regexRaised("regex.replace: invalid replacement capture", "invalid_replacement")
		}
		j := i + 2
		for j < len(replacement) && replacement[j] >= '0' && replacement[j] <= '9' {
			j++
		}
		if j == i+2 || j >= len(replacement) || replacement[j] != '}' {
			return "", regexRaised("regex.replace: invalid replacement capture", "invalid_replacement")
		}
		index, _ := strconv.Atoi(replacement[i+2 : j])
		slot := index * 2
		if index == 0 || slot+1 >= len(match) {
			return "", regexRaised("regex.replace: unknown capture reference", "unknown_capture")
		}
		if match[slot] >= 0 {
			out.WriteString(text[match[slot]:match[slot+1]])
		}
		i = j + 1
	}
	return out.String(), nil
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
		if isDataObject(lv) || isDataObject(rv) {
			return dataObjectEqual(lv, rv, seen)
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

func isDataObject(v Dict) bool {
	_, ok := v["__data_type"].(string)
	return ok
}

func dataObjectEqual(l, r Dict, seen map[visitPair]bool) (bool, error) {
	leftType, leftOK := l["__data_type"].(string)
	rightType, rightOK := r["__data_type"].(string)
	if !leftOK || !rightOK || leftType != rightType {
		return false, nil
	}
	pair := visitKey(l, r)
	if seen[pair] {
		return false, fmt.Errorf("cyclic equality is invalid")
	}
	seen[pair] = true
	defer delete(seen, pair)
	for key, leftValue := range l {
		if strings.HasPrefix(key, "__") || key == "with" {
			continue
		}
		rightValue, ok := r[key]
		if !ok {
			return false, nil
		}
		ok, err := equalValue(leftValue, rightValue, seen)
		if err != nil || !ok {
			return ok, err
		}
	}
	for key := range r {
		if strings.HasPrefix(key, "__") || key == "with" {
			continue
		}
		if _, ok := l[key]; !ok {
			return false, nil
		}
	}
	return true, nil
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
		if secs, ok := numberAsFloat(x["__duration_seconds"]); ok {
			return fmt.Sprintf("%gs", secs)
		}
		if secs, ok := numberAsFloat(x["__time_seconds"]); ok {
			monotonic, _ := x["__time_monotonic"].(bool)
			if monotonic {
				return "<monotonic time>"
			}
			value, err := formatTimeValue(secs, false, false, "rfc3339")
			if err == nil {
				return value.(string)
			}
		}
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
	if ls, ok := l.(string); ok {
		rs, ok := r.(string)
		if !ok {
			return nil, fmt.Errorf("%s expects numbers or strings of the same kind", op)
		}
		cmp := strings.Compare(ls, rs)
		switch op {
		case "<":
			return cmp < 0, nil
		case "<=":
			return cmp <= 0, nil
		case ">":
			return cmp > 0, nil
		case ">=":
			return cmp >= 0, nil
		}
	}
	lf, lok := asFloat(l)
	rf, rok := asFloat(r)
	if !lok || !rok {
		return nil, fmt.Errorf("%s expects numbers or strings of the same kind", op)
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
		if li, ok := l.(int64); ok {
			if ri, ok := r.(int64); ok {
				return li / ri, nil
			}
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

func fileInfoDict(info os.FileInfo) Dict {
	kind := "other"
	if info.Mode().IsRegular() {
		kind = "file"
	} else if info.IsDir() {
		kind = "dir"
	}
	mode := info.Mode()
	return Dict{
		"kind":       kind,
		"size":       int64(info.Size()),
		"readable":   mode&0444 != 0,
		"writable":   mode&0222 != 0,
		"executable": mode&0111 != 0,
		"mode":       int64(mode.Perm()),
	}
}

func filesystemDangerousPath(path string) bool {
	clean := filepath.Clean(path)
	return path == "" || clean == "." || clean == string(filepath.Separator) || filepath.VolumeName(clean) == clean
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
			text, err := displayString(v, env)
			if err != nil {
				return "", err
			}
			out.WriteString(text)
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

func displayString(v Value, env *Env) (string, error) {
	return displayValue(v, env, map[uintptr]bool{})
}

func displayValue(v Value, env *Env, seen map[uintptr]bool) (string, error) {
	if dict, ok := v.(Dict); ok {
		if isDataObject(dict) {
			return inspectValue(v, env, seen)
		}
		if method, ok := instanceMethod(dict, "to_string"); ok {
			value, err := callMethod(method, dict, nil)
			if err != nil {
				return "", err
			}
			text, ok := value.(string)
			if !ok {
				return "", fmt.Errorf("to_string must return string")
			}
			return text, nil
		}
	}
	return stringifyValue(v, seen), nil
}

func inspectString(v Value, env *Env) (string, error) {
	return inspectValue(v, env, map[uintptr]bool{})
}

func inspectValue(v Value, env *Env, seen map[uintptr]bool) (string, error) {
	switch x := v.(type) {
	case nil:
		return "nil", nil
	case string:
		return strconv.Quote(x), nil
	case int64, float64, bool, *Bytes, *Function, Builtin, *Module, *ErrorValue:
		return stringifyValue(v, seen), nil
	case *Array:
		key := reflect.ValueOf(x).Pointer()
		if seen[key] {
			return "<cycle>", nil
		}
		seen[key] = true
		defer delete(seen, key)
		parts := make([]string, 0, len(x.items))
		for _, item := range x.items {
			part, err := inspectValue(item, env, seen)
			if err != nil {
				return "", err
			}
			parts = append(parts, part)
		}
		return "[" + strings.Join(parts, ", ") + "]", nil
	case Dict:
		key := reflect.ValueOf(x).Pointer()
		if seen[key] {
			return "<cycle>", nil
		}
		seen[key] = true
		defer delete(seen, key)
		if isDataObject(x) {
			return inspectDataObject(x, env, seen)
		}
		if method, ok := instanceMethod(x, "inspect"); ok {
			value, err := callMethod(method, x, nil)
			if err != nil {
				return "", err
			}
			text, ok := value.(string)
			if !ok {
				return "", fmt.Errorf("inspect must return string")
			}
			return text, nil
		}
		if _, ok := x["__class"].(Dict); ok {
			return inspectInstanceObject(x, env, seen)
		}
		parts := make([]string, 0, len(x))
		keys := make([]string, 0, len(x))
		for k := range x {
			if strings.HasPrefix(k, "__") {
				continue
			}
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			part, err := inspectValue(x[k], env, seen)
			if err != nil {
				return "", err
			}
			parts = append(parts, strconv.Quote(k)+": "+part)
		}
		return "{" + strings.Join(parts, ", ") + "}", nil
	default:
		return stringifyValue(v, seen), nil
	}
}

func instanceMethod(dict Dict, name string) (Value, bool) {
	class, ok := dict["__class"].(Dict)
	if !ok {
		return nil, false
	}
	methods, ok := class["__instance_methods"].(Dict)
	if !ok {
		return nil, false
	}
	method, ok := methods[name]
	return method, ok
}

func inspectDataObject(dict Dict, env *Env, seen map[uintptr]bool) (string, error) {
	name, _ := dict["__data_type"].(string)
	return inspectFields(name, dict, fieldOrder(dict), nil, env, seen)
}

func inspectInstanceObject(dict Dict, env *Env, seen map[uintptr]bool) (string, error) {
	class, _ := dict["__class"].(Dict)
	name, _ := class["__class_name"].(string)
	private := map[string]bool{}
	if privateDict, ok := class["__private_fields"].(Dict); ok {
		for k := range privateDict {
			private[k] = true
		}
	}
	return inspectFields(name, dict, fieldOrder(class), private, env, seen)
}

func fieldOrder(dict Dict) []string {
	if arr, ok := dict["__field_order"].(*Array); ok {
		out := make([]string, 0, len(arr.items))
		for _, item := range arr.items {
			if name, ok := item.(string); ok {
				out = append(out, name)
			}
		}
		return out
	}
	names := make([]string, 0, len(dict))
	for k := range dict {
		if strings.HasPrefix(k, "__") || strings.HasPrefix(k, "@") {
			continue
		}
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func inspectFields(typeName string, dict Dict, fields []string, private map[string]bool, env *Env, seen map[uintptr]bool) (string, error) {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		if private != nil && private[field] {
			continue
		}
		value, ok := dict["@"+field]
		if !ok {
			value = dict[field]
		}
		part, err := inspectValue(value, env, seen)
		if err != nil {
			return "", err
		}
		parts = append(parts, field+": "+part)
	}
	return typeName + "(" + strings.Join(parts, ", ") + ")", nil
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
var tyaMonotonicStart = time.Now()

func timeError(message string, code string) *ErrorValue {
	return &ErrorValue{Message: message, Kind: "time", Code: code, Data: Dict{}, Cause: nil}
}

func timeRaised(message string, code string) *raisedSignal {
	return &raisedSignal{value: timeError(message, code)}
}

func timeSeconds(value Value) (float64, bool, bool) {
	switch v := value.(type) {
	case float64:
		return v, false, true
	case int64:
		return float64(v), false, true
	case Dict:
		secs, ok := numberAsFloat(v["__time_seconds"])
		if !ok {
			return 0, false, false
		}
		mono, _ := v["__time_monotonic"].(bool)
		return secs, mono, true
	default:
		return 0, false, false
	}
}

func durationSeconds(value Value) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int64:
		return float64(v), true
	case Dict:
		secs, ok := numberAsFloat(v["__duration_seconds"])
		return secs, ok
	default:
		return 0, false
	}
}

func timeObject(secs float64, monotonic bool, local bool) Dict {
	obj := Dict{"__module_namespace": true, "__time_seconds": secs, "__time_monotonic": monotonic, "__time_local": local}
	obj["unix"] = Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("time.unix expects 0 arguments")
		}
		return int64(mathpkg.Floor(secs)), nil
	})
	obj["unix_nanos"] = Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("time.unix_nanos expects 0 arguments")
		}
		return int64(secs * 1e9), nil
	})
	obj["utc"] = Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("time.utc expects 0 arguments")
		}
		return timeObject(secs, monotonic, false), nil
	})
	obj["local"] = Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("time.local expects 0 arguments")
		}
		return timeObject(secs, monotonic, true), nil
	})
	obj["format"] = Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("time.format expects 1 argument")
		}
		layout, ok := args[0].(string)
		if !ok {
			return nil, timeRaised("time.format: layout must be a string", "invalid_layout")
		}
		return formatTimeValue(secs, monotonic, local, layout)
	})
	obj["add"] = Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("time.add expects 1 argument")
		}
		d, ok := durationSeconds(args[0])
		if !ok {
			return nil, timeRaised("time.add: duration must be a duration", "invalid_duration")
		}
		return timeObject(secs+d, monotonic, local), nil
	})
	obj["sub"] = Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("time.sub expects 1 argument")
		}
		other, _, ok := timeSeconds(args[0])
		if !ok {
			return nil, timeRaised("time.sub: other must be a time", "invalid_time")
		}
		return durationObject(secs - other), nil
	})
	return obj
}

func durationObject(secs float64) Dict {
	obj := Dict{"__module_namespace": true, "__duration_seconds": secs}
	obj["seconds"] = Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("duration.seconds expects 0 arguments")
		}
		return secs, nil
	})
	obj["milliseconds"] = Builtin(func(args []Value) (Value, error) { return secs * 1e3, nil })
	obj["microseconds"] = Builtin(func(args []Value) (Value, error) { return secs * 1e6, nil })
	obj["nanoseconds"] = Builtin(func(args []Value) (Value, error) { return int64(secs * 1e9), nil })
	obj["add"] = Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("duration.add expects 1 argument")
		}
		other, ok := durationSeconds(args[0])
		if !ok {
			return nil, timeRaised("duration.add: other must be a duration", "invalid_duration")
		}
		return durationObject(secs + other), nil
	})
	obj["sub"] = Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("duration.sub expects 1 argument")
		}
		other, ok := durationSeconds(args[0])
		if !ok {
			return nil, timeRaised("duration.sub: other must be a duration", "invalid_duration")
		}
		return durationObject(secs - other), nil
	})
	return obj
}

func formatTimeValue(secs float64, monotonic bool, local bool, layout string) (Value, error) {
	if monotonic {
		return nil, timeRaised("time.format: monotonic time cannot be formatted", "monotonic_format")
	}
	t := time.Unix(int64(secs), int64((secs-mathpkg.Floor(secs))*1e9))
	if local {
		t = t.Local()
	} else {
		t = t.UTC()
	}
	switch layout {
	case "rfc3339", "iso":
		return t.Format(time.RFC3339), nil
	case "date":
		return t.Format("2006-01-02"), nil
	case "time":
		return t.Format("15:04:05"), nil
	case "unix":
		return strconv.FormatInt(t.Unix(), 10), nil
	}
	return nil, timeRaised("time.format: unknown layout", "unknown_layout")
}
