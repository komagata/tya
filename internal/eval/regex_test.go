package eval_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tya/internal/runner"
)

func TestRunRegexCompileAndSearch(t *testing.T) {
	out := runRegexProgram(t, strings.Join([]string{
		"import regex/* as regex",
		"rx = regex.Regex(\"([a-z]+)([0-9]+)\").compile()",
		"found = rx.find(\"xx abc123 yy\")",
		"print(found[\"text\"])",
		"print(found[\"start\"])",
		"print(found[\"end\"])",
		"print(found[\"groups\"][0])",
		"print(found[\"groups\"][1])",
		"print(rx.match?(\"no digits\"))",
		"print(regex.Regex(\"[0-9]+\").search(\"abc123\")[\"text\"])",
		"",
	}, "\n"))
	want := "abc123\n3\n9\nabc\n123\nfalse\n123\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestRunRegexFindAllSplitReplace(t *testing.T) {
	out := runRegexProgram(t, strings.Join([]string{
		"import regex/* as regex",
		"rx = regex.Regex(\"([a-z]+)=([0-9]+)\").compile()",
		"matches = rx.find_all(\"a=1 b=22\")",
		"print(matches.len())",
		"print(matches[1][\"groups\"][0])",
		"print(regex.Regex(\", *\").compile().split(\"a, b, c\").join(\"|\"))",
		"print(rx.replace(\"a=1 b=22\", r\"${1}:$$:${2}\"))",
		"print(rx.replace(\"a=1 b=22\", r\"${2}\", 1))",
		"",
	}, "\n"))
	want := "2\nb\na|b|c\na:$:1 b:$:22\n1 b=22\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestRunRegexUtf8RuneIndexes(t *testing.T) {
	out := runRegexProgram(t, strings.Join([]string{
		"import regex/* as regex",
		"found = regex.Regex(\"い.\").search(\"あいうえ\")",
		"print(found[\"text\"])",
		"print(found[\"start\"])",
		"print(found[\"end\"])",
		"",
	}, "\n"))
	if out != "いう\n1\n3\n" {
		t.Fatalf("got %q", out)
	}
}

func TestRunRegexInvalidPatternsAndOptionsRaiseStructuredErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{name: "pattern", src: "regex.Regex(\"[\").compile()", want: "invalid pattern"},
		{name: "unknown option", src: "regex.Regex(\"a\").compile({ bad: true })", want: "unknown option bad"},
		{name: "option kind", src: "regex.Regex(\"a\").compile({ ignore_case: 1 })", want: "option ignore_case must be bool"},
		{name: "unknown capture", src: "regex.Regex(\"(a)\").compile().replace(\"a\", r\"${9}\")", want: "unknown capture reference"},
		{name: "wrong kind", src: "regex.Regex(1).compile()", want: "pattern must be a string"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runRegexProgramError(t, "import regex/* as regex\n"+tt.src+"\n")
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func runRegexProgram(t *testing.T, src string) string {
	t.Helper()
	var out bytes.Buffer
	if err := runRegexProgramWithOutput(t, src, &out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func runRegexProgramError(t *testing.T, src string) error {
	t.Helper()
	var out bytes.Buffer
	return runRegexProgramWithOutput(t, src, &out)
}

func runRegexProgramWithOutput(t *testing.T, src string, out *bytes.Buffer) error {
	t.Helper()
	root := filepath.Clean(filepath.Join("..", ".."))
	stdlib, err := filepath.Abs(filepath.Join(root, "stdlib"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TYA_STDLIB_DIR", stdlib)
	path := filepath.Join(t.TempDir(), "regex_eval.tya")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	_, err = runner.RunFile(path, strings.NewReader(""), out, nil)
	return err
}
