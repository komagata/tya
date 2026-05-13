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
	writeFile(t, filepath.Join(dir, "greeting.tya"), "hello = name -> \"Hello, {name}\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import greeting\nprint(greeting.hello(\"komagata\"))\n")
	var out strings.Builder
	if _, err := RunFile(main, nil, &out, nil); err != nil {
		t.Fatal(err)
	}
	if out.String() != "Hello, komagata\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunFileLoadsImportedModuleDeclaration(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "util.tya"), "foo = \"foo\"\nbar = -> \"bar\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import util\nprint(util.foo)\nprint(util.bar())\n")

	var out strings.Builder
	if _, err := RunFile(main, nil, &out, nil); err != nil {
		t.Fatal(err)
	}
	if out.String() != "foo\nbar\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunFileLoadsImportedModuleAlias(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "util.tya"), "foo = \"foo\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import util as u\nprint(u.foo)\n")

	var out strings.Builder
	if _, err := RunFile(main, nil, &out, nil); err != nil {
		t.Fatal(err)
	}
	if out.String() != "foo\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestRunFileRejectsTopLevelClassInImportedModule(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "user.tya"), "class User\n  initialize = name ->\n    name\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import user\nuser = User(\"komagata\")\nprint(user.name)\n")

	var out strings.Builder
	_, err := RunFile(main, nil, &out, nil)
	if err == nil {
		t.Fatal("expected module shape rejection")
	}
	if !strings.Contains(err.Error(), "import name conflict: user") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSourceRejectsImportNameConflict(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "util.tya"), "foo = \"foo\"\n")
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

func TestLoadSourceBindsImportedFileByFileName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "greeting.tya"), "text = \"hello\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import greeting\nprint(greeting.text)\n")

	source, err := LoadSource(main)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source, "greeting[\"text\"] = text") {
		t.Fatalf("source does not synthesize import namespace:\n%s", source)
	}
}

func TestLoadSourceSynthesizesMultiplePublicBindings(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "greeting.tya"), "text = \"hello\"\nextra = \"extra\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import greeting\nprint(greeting.text)\nprint(greeting.extra)\n")

	source, err := LoadSource(main)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source, "greeting[\"text\"] = text") || !strings.Contains(source, "greeting[\"extra\"] = extra") {
		t.Fatalf("source does not synthesize all public bindings:\n%s", source)
	}
}

func TestLoadSourceRejectsTopLevelClassInImportedModule(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "user.tya"), "class User\n  initialize = name ->\n    name\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import user\nuser = User(\"komagata\")\nprint(user.name)\n")

	_, err := LoadSource(main)
	if err == nil {
		t.Fatal("expected module shape rejection")
	}
	if !strings.Contains(err.Error(), "import name conflict: user") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSourceLoadsModuleClassDeclaration(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "user.tya"), "class User\n  initialize = name ->\n    self.name = name\n")
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
	writeFile(t, filepath.Join(dir, "greeting.tya"), "text = \"hello\"\n_helper = \"ok\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import greeting\n")

	source, err := LoadSource(main)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source, "greeting[\"text\"] = text") || strings.Contains(source, "greeting[\"_helper\"]") {
		t.Fatalf("unexpected synthesized namespace:\n%s", source)
	}
}

func TestLoadSourceAcceptsClassInEntryFile(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "class User\n  initialize = name ->\n    self.name = name\n")

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
	writeFile(t, filepath.Join(dir, "greeting.tya"), "text = \"hello\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import greeting as g\nprint(greeting.text)\n")

	var out strings.Builder
	_, err := RunFile(main, nil, &out, nil)
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
	writeFile(t, filepath.Join(httpDir, "server.tya"), "listen = port -> \"listening on {port}\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import http/server\nprint(server.listen(8080))\n")

	var out strings.Builder
	if _, err := RunFile(main, nil, &out, nil); err != nil {
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
	writeFile(t, filepath.Join(httpDir, "server.tya"), "listen = port -> \"listening on {port}\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import http/server as http_server\nprint(http_server.listen(8080))\n")

	var out strings.Builder
	if _, err := RunFile(main, nil, &out, nil); err != nil {
		t.Fatal(err)
	}
	if out.String() != "listening on 8080\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestLoadSourceRejectsImportBindingConflict(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "alpha.tya"), "text = \"alpha\"\n")
	writeFile(t, filepath.Join(dir, "beta.tya"), "text = \"beta\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import alpha as util\nimport beta as util\n")

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
	writeFile(t, filepath.Join(dir, "util.tya"), "text = \"hello\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import util\nimport util as u\nprint(util.text)\nprint(u.text)\n")

	source, modules, err := LoadSourceWithModules(main)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(source, "util[\"text\"] = text") != 1 {
		t.Fatalf("expected one loaded module, got source:\n%s", source)
	}
	if strings.Join(modules, ",") != "util,u" {
		t.Fatalf("got modules %v", modules)
	}
}

func TestLoadSourceRejectsImportCycle(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.tya"), "import b\ntext = \"a\"\n")
	writeFile(t, filepath.Join(dir, "b.tya"), "import a\ntext = \"b\"\n")
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

func TestLoadSourceAllowsPrivateHelperInImportedScript(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "greeting.tya"), "_message = \"hello\"\ntext = _message\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import greeting\nprint(greeting)\n")

	source, err := LoadUserSource(main)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source, "greeting[\"text\"] = text") || strings.Contains(source, "greeting[\"_message\"]") {
		t.Fatalf("unexpected synthesized namespace:\n%s", source)
	}
}

func TestValidateAnyTyaFileNameAcceptsBothKinds(t *testing.T) {
	cases := []string{"hello.tya", "main.tya", "Greeter.tya", "HttpClient.tya"}
	for _, name := range cases {
		if err := ValidateAnyTyaFileName(name); err != nil {
			t.Errorf("%s: expected accepted, got %v", name, err)
		}
	}
}

func TestValidateAnyTyaFileNameRejectsBadShapes(t *testing.T) {
	cases := []string{"_hidden.tya", "user-utils.tya", "userUtils.tya", "main.txt"}
	for _, name := range cases {
		if err := ValidateAnyTyaFileName(name); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
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
	writeFile(t, filepath.Join(pkgDir, "Request.tya"), "class Request\n  initialize = ->\n    self.url = nil\n")
	writeFile(t, filepath.Join(pkgDir, "Response.tya"), "class Response\n  initialize = ->\n    self.status = 200\n")
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
	writeFile(t, filepath.Join(pkgDir, "Helper.tya"), "class Helper\n  initialize = ->\n    self.x = 1\n")
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

func TestLoadSourceImportsPackagePublicClassesBare(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "net", "http")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(pkgDir, "Request.tya"), "class Request\n  initialize = url ->\n    self.url = url\n")
	writeFile(t, filepath.Join(pkgDir, "Response.tya"), "class Response\n  initialize = status ->\n    self.status = status\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import net/http\nreq = Request(\"/ok\")\nprint(req.url)\n")

	source, modules, err := LoadSourceWithModules(main)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source, "class Request") || !strings.Contains(source, "req = Request(\"/ok\")") {
		t.Fatalf("source does not expose bare package class:\n%s", source)
	}
	if strings.Contains(source, "http = {}") || strings.Contains(source, "import net/http") {
		t.Fatalf("source should not create unaliased package namespace:\n%s", source)
	}
	if len(modules) != 0 {
		t.Fatalf("unaliased package import should not bind a module, got %v", modules)
	}
}

func TestLoadSourceImportsAliasedPackageAsNamespace(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "net", "http")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(pkgDir, "Request.tya"), "class Request\n  initialize = url ->\n    self.url = url\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import net/http as http\nreq = http.Request(\"/ok\")\nprint(req.url)\n")

	source, modules, err := LoadSourceWithModules(main)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source, "http = {}") || !strings.Contains(source, "http[\"Request\"] = Request") {
		t.Fatalf("source does not synthesize aliased package namespace:\n%s", source)
	}
	if strings.Join(modules, ",") != "http" {
		t.Fatalf("got modules %v", modules)
	}
}

func TestLoadSourceRejectsAliasedPackageBareClassUse(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "net", "http")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(pkgDir, "Request.tya"), "class Request\n  initialize = url ->\n    self.url = url\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import net/http as http\nreq = Request(\"/ok\")\n")

	_, err := LoadSource(main)
	if err == nil {
		t.Fatal("expected bare class rejection")
	}
	if !strings.Contains(err.Error(), "undefined variable Request") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSourceRejectsBarePackageNamespaceUse(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "net", "http")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(pkgDir, "Request.tya"), "class Request\n  initialize = url ->\n    self.url = url\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import net/http\nreq = http.Request(\"/ok\")\n")

	_, err := LoadSource(main)
	if err == nil {
		t.Fatal("expected bare package namespace rejection")
	}
	if !strings.Contains(err.Error(), "undefined variable http") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSourceRejectsBarePackageImportNameConflict(t *testing.T) {
	dir := t.TempDir()
	for _, rel := range []string{"api", "web"} {
		pkgDir := filepath.Join(dir, rel)
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(pkgDir, "Request.tya"), "class Request\n  initialize = ->\n    self.path = \"\"\n")
	}
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import api\nimport web\n")

	_, err := LoadSource(main)
	if err == nil {
		t.Fatal("expected import name conflict")
	}
	if !strings.Contains(err.Error(), "import name conflict: Request") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadSourceRejectsLocalBarePackageImportNameConflict(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "api")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(pkgDir, "Request.tya"), "class Request\n  initialize = ->\n    self.path = \"\"\n")
	main := filepath.Join(dir, "main.tya")
	writeFile(t, main, "import api\nRequest = -> \"local\"\n")

	_, err := LoadSource(main)
	if err == nil {
		t.Fatal("expected import name conflict")
	}
	if !strings.Contains(err.Error(), "import name conflict: Request") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeFile(t *testing.T, path string, src string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
}
