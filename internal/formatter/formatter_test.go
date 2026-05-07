package formatter

import "testing"

func TestFormatSourceIsIdempotent(t *testing.T) {
	input := "name = \"Tya\"  \n\tprint name\t\n\n"
	want := "name = \"Tya\"\n  print name\n"
	got := FormatSource(input)
	if got != want {
		t.Fatalf("unexpected format:\n%q", got)
	}
	if again := FormatSource(got); again != got {
		t.Fatalf("format is not idempotent:\n%q", again)
	}
}
