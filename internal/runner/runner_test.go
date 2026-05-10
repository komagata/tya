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

func TestRunFileLoadsImportedModuleAlias(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "util.tya"), "module util\n  foo = \"foo\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import util as u\nprint u.foo\n")

	var out strings.Builder
	if err := RunFile(main, nil, &out, nil); err != nil {
		t.Fatal(err)
	}
	if out.String() != "foo\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunFileRejectsTopLevelClassInImportedModule(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "user.tya"), "class User\n  init = name ->\n    name\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import user\nuser = User(\"komagata\")\nprint user.name\n")

	var out strings.Builder
	err := RunFile(main, nil, &out, nil)
	if err == nil {
		t.Fatal("expected module shape rejection")
	}
	if !strings.Contains(err.Error(), "user.tya may only contain imports and one module declaration") {
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

func TestLoadSourceRejectsTopLevelClassInImportedModule(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "user.tya"), "class User\n  init = name ->\n    name\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import user\nuser = User(\"komagata\")\nprint user.name\n")

	_, err := LoadSource(main)
	if err == nil {
		t.Fatal("expected module shape rejection")
	}
	if !strings.Contains(err.Error(), "user.tya may only contain imports and one module declaration") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSourceLoadsModuleClassDeclaration(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "user.tya"), "module user\n  class User\n    init = name ->\n      @name = name\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import user\nitem = user.User(\"komagata\")\n")

	source, err := LoadSource(main)
	if err != nil {
		t.Fatalf("load source: %v", err)
	}
	if !strings.Contains(source, "class User") {
		t.Fatalf("source does not include module class: %s", source)
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

func TestLoadSourceAcceptsClassInEntryFile(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "class User\n  init = name ->\n    @name = name\n")

	source, err := LoadSource(main)
	if err != nil {
		t.Fatalf("load source: %v", err)
	}
	if !strings.Contains(source, "class User") {
		t.Fatalf("source does not include class: %s", source)
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

func TestRunFileBindsOnlyImportAlias(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "greeting.tya"), "module greeting\n  text = \"hello\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import greeting as g\nprint greeting.text\n")

	var out strings.Builder
	err := RunFile(main, nil, &out, nil)
	if err == nil {
		t.Fatal("expected unbound original module name")
	}
	if !strings.Contains(err.Error(), "undefined variable greeting") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunFileLoadsSlashModulePath(t *testing.T) {
	dir := t.TempDir()
	httpDir := filepath.Join(dir, "http")
	if err := os.Mkdir(httpDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(httpDir, "server.tya"), "module server\n  listen = port -> \"listening on {port}\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import http/server\nprint server.listen(8080)\n")

	var out strings.Builder
	if err := RunFile(main, nil, &out, nil); err != nil {
		t.Fatal(err)
	}
	if out.String() != "listening on 8080\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunFileLoadsSlashModulePathAlias(t *testing.T) {
	dir := t.TempDir()
	httpDir := filepath.Join(dir, "http")
	if err := os.Mkdir(httpDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(httpDir, "server.tya"), "module server\n  listen = port -> \"listening on {port}\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import http/server as http_server\nprint http_server.listen(8080)\n")

	var out strings.Builder
	if err := RunFile(main, nil, &out, nil); err != nil {
		t.Fatal(err)
	}
	if out.String() != "listening on 8080\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestLoadSourceRejectsImportBindingConflict(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "string.tya"), "module string\n  text = \"string\"\n")
	writeFile(t, filepath.Join(dir, "array.tya"), "module array\n  text = \"array\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import string as util\nimport array as util\n")

	_, err := LoadSource(main)
	if err == nil {
		t.Fatal("expected import conflict")
	}
	if !strings.Contains(err.Error(), "import name conflict: util") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSourceLoadsResolvedModuleOnce(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "util.tya"), "module util\n  text = \"hello\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import util\nimport util as u\nprint util.text\nprint u.text\n")

	source, modules, err := LoadSourceWithModules(main)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(source, "module util") != 1 {
		t.Fatalf("expected one loaded module, got source:\n%s", source)
	}
	if strings.Join(modules, ",") != "util,u" {
		t.Fatalf("got modules %v", modules)
	}
}

func TestLoadSourceRejectsImportCycle(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.tya"), "import b\nmodule a\n  text = \"a\"\n")
	writeFile(t, filepath.Join(dir, "b.tya"), "import a\nmodule b\n  text = \"b\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import a\n")

	_, err := LoadSource(main)
	if err == nil {
		t.Fatal("expected import cycle")
	}
	if !strings.Contains(err.Error(), "import cycle: a -> b -> a") {
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

func TestValidateFileNameRejectsClassFileWithSpecificMessage(t *testing.T) {
	err := ValidateFileName("Hello.tya")
	if err == nil {
		t.Fatal("expected class file rejection")
	}
	if !strings.Contains(err.Error(), "class file") {
		t.Fatalf("expected class-file diagnostic, got: %v", err)
	}
	if !strings.Contains(err.Error(), "script files") {
		t.Fatalf("expected script-file hint, got: %v", err)
	}
}

func TestValidateFileNameRejectsOtherInvalidNamesGenerically(t *testing.T) {
	cases := []string{"user-utils.tya", "userUtils.tya", "main.txt"}
	for _, name := range cases {
		err := ValidateFileName(name)
		if err == nil {
			t.Fatalf("%s: expected error", name)
		}
		if strings.Contains(err.Error(), "class file") {
			t.Fatalf("%s: should not get class-file diagnostic, got: %v", name, err)
		}
	}
}

func TestResolvePackageDirFindsClassFiles(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "net", "http")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(pkgDir, "Request.tya"), "class Request\n  init = ->\n    @url = nil\n")
	writeFile(t, filepath.Join(pkgDir, "Response.tya"), "class Response\n  init = ->\n    @status = 200\n")
	importer := filepath.Join(dir, "main.tya")
	writeFile(t, importer, "")

	gotDir, files, err := resolvePackageDir(importer, "net/http")
	if err != nil {
		t.Fatal(err)
	}
	if gotDir == "" {
		t.Fatal("expected package directory to resolve, got empty")
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 class files, got %d: %v", len(files), files)
	}
}

func TestResolvePackageDirReturnsNothingWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	importer := filepath.Join(dir, "main.tya")
	writeFile(t, importer, "")

	gotDir, files, err := resolvePackageDir(importer, "missing/pkg")
	if err != nil {
		t.Fatal(err)
	}
	if gotDir != "" || files != nil {
		t.Fatalf("expected no resolution for missing package, got dir=%q files=%v", gotDir, files)
	}
}

func TestResolvePackageDirRejectsPackageWithScriptFile(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "bad")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(pkgDir, "Helper.tya"), "class Helper\n  init = ->\n    @x = 1\n")
	writeFile(t, filepath.Join(pkgDir, "script.tya"), "print(\"hi\")\n")
	importer := filepath.Join(dir, "main.tya")
	writeFile(t, importer, "")

	_, _, err := resolvePackageDir(importer, "bad")
	if err == nil {
		t.Fatal("expected script-file rejection")
	}
	if !strings.Contains(err.Error(), "script file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolvePackageDirIgnoresFileOnlyMatch(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "math.tya"), "module math\n  pi = 3.14\n")
	importer := filepath.Join(dir, "main.tya")
	writeFile(t, importer, "")

	gotDir, _, err := resolvePackageDir(importer, "math")
	if err != nil {
		t.Fatal(err)
	}
	if gotDir != "" {
		t.Fatalf("expected no directory match for module-file-only path, got %q", gotDir)
	}
}

func writeFile(t *testing.T, path string, src string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
}
