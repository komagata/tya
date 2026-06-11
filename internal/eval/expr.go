package eval

import (
	"errors"
	"fmt"
	mathpkg "math"
	"os"
	"strconv"
	"strings"

	"tya/internal/ast"
)

func evalExpr(e ast.Expr, env *Env) (Value, error) {
	switch n := e.(type) {
	case *ast.Ident:
		v, ok := env.get(n.Name)
		if !ok {
			if n.ImplicitField {
				if self, found := env.get("self"); found {
					if obj, objOK := self.(Dict); objOK {
						if value, valueOK := obj["@"+n.Name]; valueOK {
							return value, nil
						}
					}
				}
			}
			if self, found := env.get("Self"); found {
				if class, classOK := self.(Dict); classOK {
					if constant, constantOK := class[n.Name]; constantOK {
						return constant, nil
					}
				}
			}
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
	case *ast.SelfExpr:
		if n.Class {
			value, ok := env.get("Self")
			if !ok {
				return nil, fmt.Errorf("Self is only valid inside a class body")
			}
			return value, nil
		}
		value, ok := env.get("self")
		if !ok {
			return nil, fmt.Errorf("self is only valid inside a method")
		}
		return value, nil
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
	case *ast.IfStmt:
		return evalStmt(n, env)
	case *ast.WhileStmt:
		return evalStmt(n, env)
	case *ast.ForInStmt:
		return evalStmt(n, env)
	case *ast.MatchStmt:
		return evalStmt(n, env)
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
		if _, ok := o["__class"]; ok {
			if selfTarget, ok := n.Target.(*ast.SelfExpr); ok && !selfTarget.Class {
				if value, ok := o["@"+n.Name]; ok {
					return value, nil
				}
			}
			if value, ok := o[n.Name]; ok {
				return value, nil
			}
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

func assignMember(o Dict, name string, value Value) {
	if _, ok := o["__class"]; ok {
		if _, isFunction := value.(*Function); !isFunction {
			o["@"+name] = value
			if _, existingFunction := o[name].(*Function); existingFunction {
				return
			}
		}
	}
	o[name] = value
}

func evalCall(c *ast.CallExpr, env *Env) (Value, error) {
	if id, ok := c.Callee.(*ast.Ident); ok && (c.ImplicitSelf || c.ImplicitClass) {
		target := ast.Expr(&ast.SelfExpr{Tok: id.Tok, Class: c.ImplicitClass})
		return evalCall(&ast.CallExpr{
			Callee:   &ast.MemberExpr{Target: target, Name: id.Name, NameTok: id.Tok},
			Args:     c.Args,
			CallArgs: c.CallArgs,
		}, env)
	}
	fnVal, err := evalCallee(c.Callee, env)
	if err != nil {
		return nil, err
	}
	args, kwargs, err := evalCallArgs(c.EffectiveArgs(), env)
	if err != nil {
		return nil, err
	}
	if member, ok := c.Callee.(*ast.MemberExpr); ok && member.Name == "with" {
		if receiver, err := evalExpr(member.Target, env); err == nil {
			if dict, ok := receiver.(Dict); ok && dict["__record"] == true {
				if len(args) != 0 {
					return nil, fmt.Errorf("record with expects keyword arguments")
				}
				next := Dict{}
				for k, v := range dict {
					next[k] = v
				}
				for _, kw := range kwargs {
					if _, ok := dict[kw.name]; !ok || strings.HasPrefix(kw.name, "__") {
						return nil, fmt.Errorf("unknown record field %s", kw.name)
					}
					next[kw.name] = kw.value
				}
				return next, nil
			}
		}
	}
	switch fn := fnVal.(type) {
	case Builtin, *Function:
		return callValueWithKeywords(fn, args, kwargs)
	case boundMethod:
		return callValueWithKeywords(fn, args, kwargs)
	case *ast.StructDecl:
		return instantiateStruct(fn, args, kwargs, env)
	case Dict:
		if isModuleNamespace(fn) {
			return instantiateClassWithKeywords(fn, args, kwargs)
		}
		if class, ok := fn["__class"].(Dict); ok {
			if methods, ok := class["__instance_methods"].(Dict); ok {
				if method, ok := methods["call"]; ok {
					return callValueWithKeywords(boundMethod{method: method, receiver: fn, name: "call"}, args, kwargs)
				}
			}
			return nil, fmt.Errorf("object is not callable")
		}
	}
	return nil, nil
}

func instantiateStruct(decl *ast.StructDecl, args []Value, keywords []keywordArgValue, env *Env) (Value, error) {
	values, err := bindStructArgs(decl, args, keywords, env)
	if err != nil {
		return nil, err
	}
	obj := Dict{"__data_type": decl.Name}
	if decl.Record {
		obj["__record"] = true
	} else {
		obj["__struct"] = true
	}
	for i, field := range decl.Fields {
		obj[field.Name] = values[i]
	}
	fieldOrder := &Array{}
	for _, field := range decl.Fields {
		fieldOrder.items = append(fieldOrder.items, field.Name)
	}
	obj["__field_order"] = fieldOrder
	return obj, nil
}

func bindStructArgs(decl *ast.StructDecl, args []Value, keywords []keywordArgValue, env *Env) ([]Value, error) {
	params := make([]string, len(decl.Fields))
	required := 0
	for i, field := range decl.Fields {
		params[i] = field.Name
		if !field.HasDefault {
			required++
		}
	}
	bound, err := bindKeywordArgs(params, args, keywords)
	if err != nil {
		return nil, err
	}
	if len(bound) > len(params) {
		return nil, fmt.Errorf("too many positional arguments")
	}
	out := make([]Value, len(params))
	for i, field := range decl.Fields {
		if i < len(bound) {
			if _, missing := bound[i].(missingKeywordArg); !missing {
				out[i] = bound[i]
				continue
			}
		}
		if field.HasDefault {
			value, err := evalExpr(field.Value, env)
			if err != nil {
				return nil, err
			}
			out[i] = value
			continue
		}
		if i < required {
			return nil, fmt.Errorf("missing required argument %s", field.Name)
		}
	}
	return out, nil
}

type keywordArgValue struct {
	name  string
	value Value
}

type missingKeywordArg struct{}

func evalCallArgs(callArgs []ast.CallArg, env *Env) ([]Value, []keywordArgValue, error) {
	args := []Value{}
	keywords := []keywordArgValue{}
	seenKeyword := false
	for _, arg := range callArgs {
		v, err := evalExpr(arg.Value, env)
		if err != nil {
			return nil, nil, err
		}
		if arg.Expand {
			seenKeyword = true
			dict, ok := v.(Dict)
			if !ok {
				return nil, nil, fmt.Errorf("keyword expansion expects dictionary")
			}
			for key, value := range dict {
				keywords = append(keywords, keywordArgValue{name: key, value: value})
			}
			continue
		}
		if arg.Name != "" {
			seenKeyword = true
			keywords = append(keywords, keywordArgValue{name: arg.Name, value: v})
			continue
		}
		if seenKeyword {
			return nil, nil, fmt.Errorf("positional argument after keyword argument")
		}
		args = append(args, v)
	}
	return args, keywords, nil
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
	case "len", "byte_len", "char_len", "slice", "trim", "contains", "index_of", "starts_with", "ends_with", "replace", "split", "join", "lines", "upcase", "downcase", "sequence":
		return true
	default:
		return false
	}
}

func isStandardArrayCall(name string) bool {
	switch name {
	case "len", "empty?", "first", "last", "push", "pop", "slice", "reverse", "join", "map", "filter", "find", "any", "all", "each", "reduce", "sequence":
		return true
	default:
		return false
	}
}

func isStandardDictCall(name string) bool {
	switch name {
	case "len", "has", "has?", "get", "set", "delete", "keys", "values", "merge", "merge!":
		return true
	default:
		return false
	}
}

func evalStringModuleCall(name string, args []Value, env *Env) (Value, error) {
	switch name {
	case "len":
		text, err := oneString("string.len", args)
		if err != nil {
			return nil, err
		}
		return int64(len([]rune(text))), nil
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
	case "slice":
		if len(args) != 3 {
			return nil, fmt.Errorf("string.slice expects 3 arguments")
		}
		text, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("string.slice expects string text")
		}
		start, ok := args[1].(int64)
		if !ok {
			return nil, fmt.Errorf("string.slice expects integer start")
		}
		end, ok := args[2].(int64)
		if !ok {
			return nil, fmt.Errorf("string.slice expects integer end")
		}
		if start < 0 || end < 0 {
			return nil, fmt.Errorf("string.slice does not support negative indexes")
		}
		runes := []rune(text)
		if start > end || int(end) > len(runes) {
			return nil, fmt.Errorf("string.slice index out of range")
		}
		return string(runes[int(start):int(end)]), nil
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
	case "index_of":
		if len(args) != 2 && len(args) != 3 {
			return nil, fmt.Errorf("string.index_of expects 2 or 3 arguments")
		}
		text, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("string.index_of expects string text")
		}
		needle, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("string.index_of expects string needle")
		}
		start := int64(0)
		if len(args) == 3 {
			var ok bool
			start, ok = args[2].(int64)
			if !ok {
				return nil, fmt.Errorf("string.index_of expects integer start")
			}
		}
		if start < 0 {
			return nil, fmt.Errorf("string.index_of does not support negative indexes")
		}
		runes := []rune(text)
		if int(start) > len(runes) {
			return int64(-1), nil
		}
		idx := strings.Index(string(runes[int(start):]), needle)
		if idx < 0 {
			return int64(-1), nil
		}
		return start + int64(len([]rune(string(runes[int(start):])[:idx]))), nil
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
	case "sequence":
		if len(args) != 1 {
			return nil, fmt.Errorf("string.sequence expects 1 argument")
		}
		text, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("string.sequence expects string")
		}
		return primitiveSequence(text), nil
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
	case "sequence":
		if len(args) != 1 {
			return nil, fmt.Errorf("array.sequence expects 1 argument")
		}
		arr, ok := args[0].(*Array)
		if !ok {
			return nil, fmt.Errorf("array.sequence expects array")
		}
		return primitiveSequence(arr), nil
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
	case "merge!":
		if len(args) != 2 {
			return nil, fmt.Errorf("dict.merge! expects 2 arguments")
		}
		left, ok := args[0].(Dict)
		if !ok {
			return nil, fmt.Errorf("dict.merge! expects dictionary left")
		}
		right, ok := args[1].(Dict)
		if !ok {
			return nil, fmt.Errorf("dict.merge! expects dictionary right")
		}
		for key, value := range right {
			left[key] = value
		}
		return nil, nil
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
	case *Bytes:
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

func primitiveSequence(source Value) Dict {
	seq := Dict{}
	seq["to_a"] = Builtin(func(args []Value) (Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("sequence.to_a expects 0 arguments")
		}
		return sequenceItems(source)
	})
	seq["reduce"] = Builtin(func(args []Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("sequence.reduce expects 2 arguments")
		}
		items, err := sequenceItems(source)
		if err != nil {
			return nil, err
		}
		acc := args[0]
		for _, item := range items.items {
			acc, err = callValue(args[1], []Value{acc, item})
			if err != nil {
				return nil, err
			}
		}
		return acc, nil
	})
	seq["each"] = Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("sequence.each expects 1 argument")
		}
		items, err := sequenceItems(source)
		if err != nil {
			return nil, err
		}
		for _, item := range items.items {
			if _, err := callValue(args[0], []Value{item}); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	seq["any?"] = Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("sequence.any? expects 1 argument")
		}
		items, err := sequenceItems(source)
		if err != nil {
			return nil, err
		}
		for _, item := range items.items {
			ok, err := callValue(args[0], []Value{item})
			if err != nil {
				return nil, err
			}
			if truthy(ok) {
				return true, nil
			}
		}
		return false, nil
	})
	seq["all?"] = Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("sequence.all? expects 1 argument")
		}
		items, err := sequenceItems(source)
		if err != nil {
			return nil, err
		}
		for _, item := range items.items {
			ok, err := callValue(args[0], []Value{item})
			if err != nil {
				return nil, err
			}
			if !truthy(ok) {
				return false, nil
			}
		}
		return true, nil
	})
	seq["find"] = Builtin(func(args []Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("sequence.find expects 1 argument")
		}
		items, err := sequenceItems(source)
		if err != nil {
			return nil, err
		}
		for _, item := range items.items {
			ok, err := callValue(args[0], []Value{item})
			if err != nil {
				return nil, err
			}
			if truthy(ok) {
				return item, nil
			}
		}
		return nil, nil
	})
	return seq
}

func sequenceItems(source Value) (*Array, error) {
	switch v := source.(type) {
	case string:
		out := &Array{items: make([]Value, 0, len([]rune(v)))}
		for _, r := range []rune(v) {
			out.items = append(out.items, string(r))
		}
		return out, nil
	case *Bytes:
		out := &Array{items: make([]Value, 0, len(v.data))}
		for _, b := range v.data {
			out.items = append(out.items, int64(b))
		}
		return out, nil
	case *Array:
		return v, nil
	default:
		return nil, fmt.Errorf("sequence source is not iterable")
	}
}

func callValue(fn Value, args []Value) (Value, error) {
	return callValueWithKeywords(fn, args, nil)
}

func callValueWithKeywords(fn Value, args []Value, keywords []keywordArgValue) (Value, error) {
	switch f := fn.(type) {
	case Builtin:
		if len(keywords) > 0 {
			return nil, fmt.Errorf("builtin function does not accept keyword arguments")
		}
		return f(args)
	case boundMethod:
		value, err := callMethodWithKeywords(f.method, f.receiver, args, keywords)
		if err != nil {
			return nil, err
		}
		if f.name == "to_string" || f.name == "inspect" {
			if _, ok := value.(string); !ok {
				return nil, fmt.Errorf("%s must return string", f.name)
			}
		}
		return value, nil
	case *Function:
		var err error
		args, err = bindKeywordArgs(f.Params, args, keywords)
		if err != nil {
			return nil, err
		}
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
				if _, missing := args[i].(missingKeywordArg); missing {
					if i < len(f.Defaults) && f.Defaults[i] != nil {
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
				} else {
					value = args[i]
				}
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
	return nil, nil
}

func bindKeywordArgs(params []string, args []Value, keywords []keywordArgValue) ([]Value, error) {
	if len(keywords) == 0 {
		return args, nil
	}
	values := make([]Value, len(params))
	filled := make([]bool, len(params))
	for i, arg := range args {
		if i >= len(params) {
			return args, nil
		}
		values[i] = arg
		filled[i] = true
	}
	indexes := map[string]int{}
	for i, name := range params {
		indexes[name] = i
	}
	for _, keyword := range keywords {
		index, ok := indexes[keyword.name]
		if !ok {
			return nil, fmt.Errorf("unknown keyword %s", keyword.name)
		}
		if filled[index] {
			return nil, fmt.Errorf("argument %s supplied multiple times", keyword.name)
		}
		values[index] = keyword.value
		filled[index] = true
	}
	last := -1
	for i := range filled {
		if filled[i] {
			last = i
		}
	}
	if last < 0 {
		return nil, nil
	}
	out := make([]Value, last+1)
	for i := 0; i <= last; i++ {
		if filled[i] {
			out[i] = values[i]
		} else {
			out[i] = missingKeywordArg{}
		}
	}
	return out, nil
}

func instantiateClass(class Dict, args []Value) (Value, error) {
	return instantiateClassWithKeywords(class, args, nil)
}

func instantiateClassWithKeywords(class Dict, args []Value, keywords []keywordArgValue) (Value, error) {
	name, _ := class["__class_name"].(string)
	obj := Dict{"__class": class, "__class_name": name}
	if fields, ok := class["__fields"].(Dict); ok {
		for key, value := range fields {
			obj["@"+key] = value
		}
	}
	methods, _ := class["__instance_methods"].(Dict)
	init, ok := methods["initialize"]
	if !ok {
		if len(args) != 0 {
			return nil, fmt.Errorf("function expects 0 arguments, got %d", len(args))
		}
		return obj, nil
	}
	if _, err := callMethodWithKeywords(init, obj, args, keywords); err != nil {
		return nil, err
	}
	return obj, nil
}

func callMethod(method Value, receiver Value, args []Value) (Value, error) {
	return callMethodWithKeywords(method, receiver, args, nil)
}

func callMethodWithKeywords(method Value, receiver Value, args []Value, keywords []keywordArgValue) (Value, error) {
	fn, ok := method.(*Function)
	if !ok {
		return callValueWithKeywords(method, args, keywords)
	}
	callEnv := fn.Env.child()
	callEnv.set("self", receiver)
	bound := *fn
	bound.Env = callEnv
	return callValueWithKeywords(&bound, args, keywords)
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
			if dict["__record"] == true && m.Name == "with" {
				return Builtin(func(args []Value) (Value, error) {
					if len(args) != 1 {
						return nil, fmt.Errorf("record with expects keyword arguments")
					}
					updates, ok := args[0].(Dict)
					if !ok {
						return nil, fmt.Errorf("record with expects keyword arguments")
					}
					next := Dict{}
					for k, v := range dict {
						next[k] = v
					}
					for k, v := range updates {
						if _, ok := dict[k]; !ok || strings.HasPrefix(k, "__") {
							return nil, fmt.Errorf("unknown record field %s", k)
						}
						next[k] = v
					}
					return next, nil
				}), nil
			}
			if isModuleNamespace(dict) {
				return dict[m.Name], nil
			}
			if class, ok := dict["__class"].(Dict); ok {
				if methods, ok := class["__instance_methods"].(Dict); ok {
					if method, ok := methods[m.Name]; ok {
						return boundMethod{method: method, receiver: dict, name: m.Name}, nil
					}
				}
			}
			if value, ok := dict[m.Name]; ok {
				return value, nil
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
		case "to_string":
			return Builtin(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("to_string expects 0 arguments")
				}
				return displayString(receiver, env)
			})
		case "inspect":
			return Builtin(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("inspect expects 0 arguments")
				}
				return inspectString(receiver, env)
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
		if name == "to_string" {
			return Builtin(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("to_string expects 0 arguments")
				}
				return displayString(receiver, env)
			})
		}
		if name == "inspect" {
			return Builtin(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("inspect expects 0 arguments")
				}
				return inspectString(receiver, env)
			})
		}
		if isStandardArrayCall(name) {
			return call(func(args []Value) (Value, error) { return evalArrayModuleCall(name, args, env) })
		}
	case Dict:
		if name == "to_string" {
			return Builtin(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("to_string expects 0 arguments")
				}
				return displayString(receiver, env)
			})
		}
		if name == "inspect" {
			return Builtin(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("inspect expects 0 arguments")
				}
				return inspectString(receiver, env)
			})
		}
		if isStandardDictCall(name) {
			return call(func(args []Value) (Value, error) { return evalDictModuleCall(name, args, env) })
		}
	case *Bytes:
		if name == "to_string" {
			return Builtin(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("to_string expects 0 arguments")
				}
				return displayString(receiver, env)
			})
		}
		if name == "inspect" {
			return Builtin(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("inspect expects 0 arguments")
				}
				return inspectString(receiver, env)
			})
		}
		if name == "len" {
			return Builtin(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("bytes.len expects 0 arguments")
				}
				return int64(len(receiver.(*Bytes).data)), nil
			})
		}
		if name == "sequence" {
			return Builtin(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("bytes.sequence expects 0 arguments")
				}
				return primitiveSequence(receiver), nil
			})
		}
	case int64, float64, bool, nil:
		if name == "to_string" {
			return Builtin(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("to_string expects 0 arguments")
				}
				return displayString(receiver, env)
			})
		}
		if name == "inspect" {
			return Builtin(func(args []Value) (Value, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("inspect expects 0 arguments")
				}
				return inspectString(receiver, env)
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
	case *Bytes:
		return primitiveClass("Bytes")
	case *Array:
		return primitiveClass("Array")
	case Dict:
		return primitiveClass("Dict")
	default:
		return nil
	}
}

func primitiveClass(name string) Dict {
	if class, ok := primitiveClasses[name]; ok {
		return class
	}
	display := displayClassName(name)
	return Dict{"__module_namespace": true, "__class_name": display, "name": display}
}

func displayClassName(name string) string {
	if i := strings.Index(name, "TyaPkg"); i > 0 {
		return name[:i]
	}
	return name
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
	if b.Op.Lexeme == "??" {
		if l != nil {
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
