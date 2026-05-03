package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateFileName(t *testing.T) {
	valid := []string{"main.tya", "user_utils.tya", "string_tools.tya"}
	for _, name := range valid {
		if err := ValidateFileName(name); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}

	invalid := []string{"UserUtils.tya", "user-utils.tya", "userUtils.tya", "main.txt"}
	for _, name := range invalid {
		if err := ValidateFileName(name); err == nil {
			t.Fatalf("%s: expected error", name)
		}
	}
}

func TestRunFileLoadsImportedModule(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "greeting.tya"), "greeting =\n  hello: name -> \"Hello, {name}\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import greeting\nprint greeting.hello(\"komagata\")\n")

	var out strings.Builder
	if err := RunFile(main, nil, &out, nil); err != nil {
		t.Fatal(err)
	}
	if out.String() != "Hello, komagata\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunFileLoadsImportedModuleDeclaration(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "util.tya"), "module util\n  foo: \"foo\"\n  bar: -> \"bar\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import util\nprint util.foo\nprint util.bar()\n")

	var out strings.Builder
	if err := RunFile(main, nil, &out, nil); err != nil {
		t.Fatal(err)
	}
	if out.String() != "foo\nbar\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestLoadSourceRejectsModuleWithMismatchedPublicBinding(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "greeting.tya"), "message = \"hello\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import greeting\nprint message\n")

	_, err := LoadSource(main)
	if err == nil {
		t.Fatal("expected module binding error")
	}
	if !strings.Contains(err.Error(), "greeting.tya must expose exactly one public binding named greeting") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSourceRejectsModuleWithMultiplePublicBindings(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "greeting.tya"), "greeting = \"hello\"\nmessage = \"extra\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import greeting\nprint greeting\n")

	_, err := LoadSource(main)
	if err == nil {
		t.Fatal("expected module binding error")
	}
	if !strings.Contains(err.Error(), "greeting.tya must expose exactly one public binding named greeting") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSourceRejectsInvalidModuleName(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import userUtils\n")

	_, err := LoadSource(main)
	if err == nil {
		t.Fatal("expected invalid module name error")
	}
	if !strings.Contains(err.Error(), "invalid module name: userUtils") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSourceCurrentBaselineRejectsImportAlias(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "greeting.tya"), "greeting = \"hello\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import greeting as g\n")

	_, err := LoadSource(main)
	if err == nil {
		t.Fatal("expected import alias to be rejected before alias support lands")
	}
	if !strings.Contains(err.Error(), "invalid module name: greeting as g") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSourceCurrentBaselineAllowsPrivateHelperInImportedModule(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "greeting.tya"), "greeting = _message\n_message = \"hello\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import greeting\nprint greeting\n")

	source, err := LoadUserSource(main)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source, "_message = \"hello\"") {
		t.Fatalf("expected private helper to remain loaded in current baseline:\n%s", source)
	}
}

func writeFile(t *testing.T, path string, src string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
}
