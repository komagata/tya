package lsp

import "testing"

func TestFormatRewritesAcceptedSyntax(t *testing.T) {
	src := "add = (a, b,) -> a + b\nname = 'Tya'\nprint(add(1, 2,))\n"
	edits, err := Format(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(edits) != 1 {
		t.Fatalf("got %d edits, want 1", len(edits))
	}
	want := "add = a, b -> a + b\nname = \"Tya\"\nprint(add(1, 2))\n"
	if edits[0].NewText != want {
		t.Fatalf("got:\n%swant:\n%s", edits[0].NewText, want)
	}
}
