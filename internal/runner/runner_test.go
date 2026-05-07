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
	writeFile(t, filepath.Join(dir, "greeting.tya"), "module greeting\n  hello = name -> \"Hello, {name}\"\n")
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
	writeFile(t, filepath.Join(dir, "util.tya"), "module util\n  foo = \"foo\"\n  bar = -> \"bar\"\n")
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

func TestRunFileRejectsImportedModuleAlias(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "util.tya"), "module util\n  foo = \"foo\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import util as u\nprint u.foo\n")

	var out strings.Builder
	err := RunFile(main, nil, &out, nil)
	if err == nil {
		t.Fatal("expected import alias error")
	}
	if !strings.Contains(err.Error(), "import aliases are not in Tya v0.1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunFileRejectsImportedClass(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "user.tya"), "class User\n  init = name ->\n    name\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import user\nuser = User(\"komagata\")\nprint user.name\n")

	var out strings.Builder
	err := RunFile(main, nil, &out, nil)
	if err == nil {
		t.Fatal("expected class rejection")
	}
	if !strings.Contains(err.Error(), "class is not in Tya v0.1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSourceRejectsImportNameConflict(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "util.tya"), "module util\n  foo = \"foo\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import util\nutil = \"conflict\"\n")

	_, err := LoadSource(main)
	if err == nil {
		t.Fatal("expected import conflict")
	}
	if !strings.Contains(err.Error(), "import name conflict: util") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSourceRejectsModuleWithMismatchedPublicBinding(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "greeting.tya"), "module message\n  text = \"hello\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import greeting\nprint message\n")

	_, err := LoadSource(main)
	if err == nil {
		t.Fatal("expected module binding error")
	}
	if !strings.Contains(err.Error(), "greeting.tya must define module greeting") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSourceRejectsModuleWithMultiplePublicBindings(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "greeting.tya"), "module greeting\n  text = \"hello\"\nmodule greeting\n  extra = \"extra\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import greeting\nprint greeting\n")

	_, err := LoadSource(main)
	if err == nil {
		t.Fatal("expected module binding error")
	}
	if !strings.Contains(err.Error(), "greeting.tya must define exactly one module") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSourceRejectsImportedClassDeclaration(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "user.tya"), "class User\n  init = name ->\n    name\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import user\nuser = User(\"komagata\")\nprint user.name\n")

	_, err := LoadSource(main)
	if err == nil {
		t.Fatal("expected class rejection")
	}
	if !strings.Contains(err.Error(), "class is not in Tya v0.1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSourceRejectsImportedClassBeforeNameValidation(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "user.tya"), "class Account\n  init: -> nil\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import user\n")

	_, err := LoadSource(main)
	if err == nil {
		t.Fatal("expected class filename mismatch error")
	}
	if !strings.Contains(err.Error(), "class is not in Tya v0.1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSourceRejectsTopLevelHelperInImportedFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "greeting.tya"), "module greeting\n  text = \"hello\"\n_helper = \"bad\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import greeting\n")

	_, err := LoadSource(main)
	if err == nil {
		t.Fatal("expected top-level helper error")
	}
	if !strings.Contains(err.Error(), "greeting.tya may only contain imports and one module declaration") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSourceRejectsClassInEntryFile(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "class User\n  init: -> nil\n")

	_, err := LoadSource(main)
	if err == nil {
		t.Fatal("expected entry class error")
	}
	if !strings.Contains(err.Error(), "class is not in Tya v0.1") {
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

func TestLoadSourceRejectsImportAlias(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "greeting.tya"), "module greeting\n  text = \"hello\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import greeting as g\n")

	var out strings.Builder
	err := RunFile(main, nil, &out, nil)
	if err == nil {
		t.Fatal("expected import alias error")
	}
	if !strings.Contains(err.Error(), "import aliases are not in Tya v0.1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSourceRejectsPrivateHelperInImportedModule(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "greeting.tya"), "module greeting\n  text = _message\n_message = \"hello\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import greeting\nprint greeting\n")

	_, err := LoadUserSource(main)
	if err == nil {
		t.Fatal("expected private helper rejection")
	}
	if !strings.Contains(err.Error(), "greeting.tya may only contain imports and one module declaration") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeFile(t *testing.T, path string, src string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
}
