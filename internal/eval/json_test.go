package eval_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tya/internal/runner"
)

func TestRunJsonParseNonASCIIString(t *testing.T) {
	out := runJsonProgram(t, strings.Join([]string{
		"import json",
		"item = json.Json(\"{{\\\"name\\\":\\\"test_copies_learning_and_product_data_when_進捗コピー_button_is_clicked\\\",\\\"duration\\\":0}}\").parse()",
		"print(item[\"name\"])",
		"print(item[\"duration\"])",
		"",
	}, "\n"))
	want := "test_copies_learning_and_product_data_when_進捗コピー_button_is_clicked\n0\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestRunJsonParseUnicodeEscapeStringConcatenatesAsString(t *testing.T) {
	out := runJsonProgram(t, strings.Join([]string{
		"import json",
		"item = json.Json(\"{{\\\"name\\\":\\\"\\\\u9032\\\\u6357\\\\u30b3\\\\u30d4\\\\u30fc\\\"}}\").parse()",
		"print(\"prefix:\" + item[\"name\"])",
		"",
	}, "\n"))
	if out != "prefix:進捗コピー\n" {
		t.Fatalf("got %q", out)
	}
}

func runJsonProgram(t *testing.T, src string) string {
	t.Helper()
	root := filepath.Clean(filepath.Join("..", ".."))
	stdlib, err := filepath.Abs(filepath.Join(root, "lib"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TYA_LIB_DIR", stdlib)
	path := filepath.Join(t.TempDir(), "json_eval.tya")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if _, err := runner.RunFile(path, strings.NewReader(""), &out, nil); err != nil {
		t.Fatal(err)
	}
	return out.String()
}
