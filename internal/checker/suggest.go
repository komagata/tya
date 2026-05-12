package checker

import (
	"fmt"

	"tya/internal/util"
)

// undefinedNameError formats the legacy "line:col: undefined
// variable foo" error string but optionally appends a "did you
// mean 'bar'?" suffix when a close candidate is found in the
// current scope. The shape stays string-error to keep the v0.54
// migration footprint minimal — the checker continues to return
// `error` everywhere.
func undefinedNameError(name string, line, col int, sc *scope) error {
	pool := append([]string{}, BuiltinNames()...)
	pool = append(pool, collectScopeNames(sc)...)
	suggestions := util.Suggest(name, pool, 2, 3)
	if len(suggestions) == 0 {
		return fmt.Errorf("%d:%d: undefined variable %s", line, col, name)
	}
	hint := "did you mean "
	for i, s := range suggestions {
		if i > 0 {
			hint += " / "
		}
		hint += fmt.Sprintf("%q", s)
	}
	hint += "?"
	return fmt.Errorf("%d:%d: undefined variable %s; %s", line, col, name, hint)
}

// collectScopeNames walks the scope chain and returns every
// directly-bound name (in declaration order, parent last).
func collectScopeNames(sc *scope) []string {
	if sc == nil {
		return nil
	}
	out := []string{}
	for s := sc; s != nil; s = s.parent {
		for k := range s.names {
			out = append(out, k)
		}
	}
	return out
}
