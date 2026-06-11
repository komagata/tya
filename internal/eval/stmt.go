package eval

import (
	"errors"
	"fmt"

	"tya/internal/ast"
	"tya/internal/token"
)

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
		return evalImportStmt(n, env)
	case *ast.ImportBlockStmt:
		var last Value
		for _, imp := range n.Imports {
			value, err := evalImportStmt(imp, env)
			if err != nil {
				return nil, err
			}
			last = value
		}
		return last, nil
	case *ast.AssignStmt:
		if n.Tok.Type == token.NIL_ASSIGN {
			return evalNilAssign(n, env)
		}
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
		className := displayClassName(n.Name)
		class := Dict{"__module_namespace": true, "__class_name": className, "name": className, "__instance_methods": Dict{}, "__fields": Dict{}}
		env.set(n.Name, class)
		fields := class["__fields"].(Dict)
		fieldOrder := &Array{}
		privateFields := Dict{}
		for _, field := range n.Fields {
			value, err := evalExpr(field.Value, env)
			if err != nil {
				return nil, err
			}
			fields[field.Name] = value
			fieldOrder.items = append(fieldOrder.items, field.Name)
			if field.Private {
				privateFields[field.Name] = true
			}
		}
		class["__field_order"] = fieldOrder
		class["__private_fields"] = privateFields
		for _, constant := range n.Constants {
			value, err := evalExpr(constant.Value, env)
			if err != nil {
				return nil, err
			}
			class[constant.Name] = value
		}
		for _, method := range n.Methods {
			if method.Abstract {
				continue
			}
			methodEnv := env.child()
			methodEnv.set("Self", class)
			value, err := evalExpr(method.Func, methodEnv)
			if err != nil {
				return nil, err
			}
			if method.Class {
				class[method.Name] = nameFunction(method.Name, value)
			} else {
				class["__instance_methods"].(Dict)[method.Name] = nameFunction(method.Name, value)
			}
		}
		return class, nil
	case *ast.StructDecl:
		decl := n
		constructor := Builtin(func(args []Value) (Value, error) {
			return instantiateStruct(decl, args, nil, env)
		})
		env.set(n.Name, constructor)
		return constructor, nil
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
		if text, ok := iterable.(string); ok {
			var last Value
			for i, r := range []rune(text) {
				loopEnv := env.child()
				loopEnv.vars[n.ValueName] = string(r)
				loopEnv.rememberKind(n.ValueName, loopEnv.vars[n.ValueName])
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
		}
		if bytesVal, ok := iterable.(*Bytes); ok {
			var last Value
			for i, b := range bytesVal.data {
				loopEnv := env.child()
				loopEnv.vars[n.ValueName] = int64(b)
				loopEnv.rememberKind(n.ValueName, loopEnv.vars[n.ValueName])
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

func evalImportStmt(n *ast.ImportStmt, env *Env) (Value, error) {
	if n.Alias != "" {
		value, ok := env.get(n.ModuleName())
		if !ok {
			return nil, fmt.Errorf("undefined imported module %s", n.ModuleName())
		}
		env.set(n.Alias, value)
		return value, nil
	}
	return nil, nil
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

func evalNilAssign(n *ast.AssignStmt, env *Env) (Value, error) {
	if len(n.Targets) != 1 || len(n.Values) != 1 {
		return nil, fmt.Errorf("??= assignment requires exactly one target and one value")
	}
	current, err := assignmentValue(n.Targets[0], env)
	if err != nil {
		return nil, err
	}
	if current != nil {
		return current, nil
	}
	value, err := evalExpr(n.Values[0], env)
	if err != nil {
		return nil, err
	}
	if id, ok := n.Targets[0].(*ast.Ident); ok {
		value = nameFunction(id.Name, value)
	}
	if err := assign(n.Targets[0], value, env); err != nil {
		return nil, err
	}
	return value, nil
}

func assignmentValue(target ast.Expr, env *Env) (Value, error) {
	switch t := target.(type) {
	case *ast.Ident:
		if t.Name == "_" {
			return nil, nil
		}
		value, ok := env.get(t.Name)
		if !ok {
			return nil, nil
		}
		return value, nil
	case *ast.MemberExpr, *ast.IndexExpr:
		return evalExpr(target, env)
	default:
		return evalExpr(target, env)
	}
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
		if o["__record"] == true {
			return fmt.Errorf("cannot assign to record field %s", t.Name)
		}
		if _, ok := o[t.Name]; !ok {
			if o["__struct"] == true || o["__record"] == true {
				return fmt.Errorf("undeclared struct or record field %s", t.Name)
			}
		}
		assignMember(o, t.Name, v)
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
