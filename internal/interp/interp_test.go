package interp

import "testing"

func TestFindExprEndBalancesBracesAndStrings(t *testing.T) {
	src := `{ {"kind": "{ok}"}["kind"] } tail`
	got := FindExprEnd(src, 0)
	if got != len(`{ {"kind": "{ok}"}["kind"] }`)-1 {
		t.Fatalf("end: got %d in %q", got, src)
	}
}

func TestFindExprEndHandlesEscapedQuotes(t *testing.T) {
	src := `{user["\"quoted\""]} tail`
	got := FindExprEnd(src, 0)
	if got != len(`{user["\"quoted\""]}`)-1 {
		t.Fatalf("end: got %d in %q", got, src)
	}
}

func TestFindExprEndReportsUnclosed(t *testing.T) {
	if got := FindExprEnd(`{user["name"]`, 0); got != -1 {
		t.Fatalf("end: got %d, want -1", got)
	}
}
