package tests

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tya/internal/lsp"
)

func TestSpecDocumentsSingleErrorModel(t *testing.T) {
	spec := readRepoFile(t, "docs", "SPEC.md")
	for _, forbidden := range []string{
		"try` may be used as an expression",
		"try can also be an expression",
		"return a documented `value, err`",
		"return nil, error(",
	} {
		if strings.Contains(spec, forbidden) {
			t.Fatalf("SPEC.md still documents forbidden error model text %q", forbidden)
		}
	}
	for _, required := range []string{
		"`try` is a statement only",
		"`catch err` is the only catch syntax",
		"raise structured error values",
		"error(message, options = {})",
	} {
		if !strings.Contains(spec, required) {
			t.Fatalf("SPEC.md missing %q", required)
		}
	}
}

func TestStdlibDocsUseStructuredRaisedErrors(t *testing.T) {
	spec := readRepoFile(t, "docs", "SPEC.md")
	for _, required := range []string{
		"kind",
		"code",
		"data",
		"cause",
		"raise structured error values",
	} {
		if !strings.Contains(spec, required) {
			t.Fatalf("structured error docs missing %q", required)
		}
	}
}

func TestSpecDocumentsV1CompatibilityBoundary(t *testing.T) {
	spec := readRepoFile(t, "docs", "SPEC.md")
	for _, required := range []string{
		"## v1.0.0 Compatibility Boundary",
		"documented public standard-library and package APIs",
		"CLI JSON diagnostic schema",
		"not compatibility guarantees",
		"bootstrap recovery path",
	} {
		if !strings.Contains(spec, required) {
			t.Fatalf("SPEC.md missing v1 compatibility text %q", required)
		}
	}
}

func TestStrictSemanticsDocumentsDynamicAllowances(t *testing.T) {
	strict := readRepoFile(t, "docs", "STRICT_SEMANTICS.md")
	for _, required := range []string{
		"## Dynamic Allowances",
		"Runtime-kind checks",
		"`nil` may be returned",
		"compare any two values without coercion",
		"Runtime errors are valid",
		"do not permit implicit conversion",
	} {
		if !strings.Contains(strict, required) {
			t.Fatalf("STRICT_SEMANTICS.md missing dynamic allowance text %q", required)
		}
	}
}

func TestRoadmapDocumentsV1GoRecoveryPath(t *testing.T) {
	roadmap := readRepoFile(t, "ROADMAP.md")
	for _, required := range []string{
		"Go implementation remains the reference implementation",
		"bootstrap recovery path",
		"no-Go self-host bootstrap path",
		"not a v1.0.0 release requirement",
	} {
		if !containsNormalized(roadmap, required) {
			t.Fatalf("ROADMAP.md missing Go recovery path text %q", required)
		}
	}
}

func TestSpecDocumentsWasmAsNonBlockingTarget(t *testing.T) {
	spec := readRepoFile(t, "docs", "SPEC.md")
	for _, required := range []string{
		"WebAssembly is documented for v1.0.0",
		"not a release-blocking target",
		"WASM-specific gaps are tracked separately",
	} {
		if !containsNormalized(spec, required) {
			t.Fatalf("SPEC.md missing WASM release boundary text %q", required)
		}
	}
}

func TestSelfhostCoverageManifestCoversSpec(t *testing.T) {
	manifest := readRepoFile(t, "docs", "SELFHOST_COVERAGE.md")
	for _, required := range []string{
		"SPEC feature",
		"Lexer",
		"Parser",
		"AST",
		"Checker",
		"C emitter",
		"Runtime",
		"v1 release gate",
		"`gap` release-gate value",
	} {
		if !strings.Contains(manifest, required) {
			t.Fatalf("SELFHOST_COVERAGE.md missing %q", required)
		}
	}
	for _, feature := range []string{
		"Indentation, comments, identifiers, and literals",
		"Functions, calls, returns, defaults, and multiple returns",
		"`raise`, `try`, `catch`, `finally`, and structured errors",
		"`spawn`, `await`, `scope`, channels, and `select`",
	} {
		if !strings.Contains(manifest, feature) {
			t.Fatalf("SELFHOST_COVERAGE.md missing feature row %q", feature)
		}
	}
}

func TestReleaseNoGoBootstrapRequiredOnUnix(t *testing.T) {
	script := readRepoFile(t, "scripts", "bootstrap_no_go.sh")
	for _, required := range []string{
		"TYA_BOOTSTRAP_TYA",
		"no-Go violation",
		"stage-2 emit",
		"stage-3 emit",
		"fixed-point compare passed",
	} {
		if !strings.Contains(script, required) {
			t.Fatalf("bootstrap_no_go.sh missing %q", required)
		}
	}
}

func TestReleaseWindowsSmokeCoverage(t *testing.T) {
	script := readRepoFile(t, "scripts", "build_release_packages.sh")
	for _, required := range []string{
		"build_package windows amd64",
		"build_package windows arm64",
		"install.ps1",
		"zig.exe",
	} {
		if !strings.Contains(script, required) {
			t.Fatalf("build_release_packages.sh missing Windows release text %q", required)
		}
	}
}

func TestAllUserFacingDiagnosticsHaveStableCodes(t *testing.T) {
	for _, path := range []string{
		filepath.Join("internal", "parser", "codes.go"),
		filepath.Join("internal", "checker", "strict.go"),
		filepath.Join("internal", "codegen", "errors.go"),
		filepath.Join("internal", "runner", "errors.go"),
		filepath.Join("cmd", "tya", "main.go"),
	} {
		text := readRepoFile(t, strings.Split(path, string(os.PathSeparator))...)
		if !strings.Contains(text, "TYA-E") {
			t.Fatalf("%s missing stable TYA-E diagnostic codes", path)
		}
	}
}

func TestDiagnosticJsonAndLspCodesMatchCli(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.tya")
	if err := os.WriteFile(path, []byte("return 1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "--json", "check", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected tya check failure")
	}
	var cliCode string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		var obj struct {
			Type string `json:"type"`
			Code string `json:"code"`
		}
		if json.Unmarshal([]byte(line), &obj) == nil && obj.Code != "" {
			cliCode = obj.Code
			break
		}
	}
	if cliCode == "" {
		t.Fatalf("JSON diagnostics missing code:\n%s", out)
	}
	diags := lsp.DiagnosticsFor(path, "return 1\n")
	if len(diags) == 0 {
		t.Fatal("expected LSP diagnostics")
	}
	if got := diags[0].Code; got != cliCode {
		t.Fatalf("LSP code %q, CLI code %q", got, cliCode)
	}
}

func TestRuntimeErrorsExposeStableCodes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runtime_error.tya")
	if err := os.WriteFile(path, []byte("raise nil\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "run", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected runtime failure")
	}
	for _, required := range []string{"TYA-E0900", "raise expects error value"} {
		if !strings.Contains(string(out), required) {
			t.Fatalf("runtime diagnostic missing %q:\n%s", required, out)
		}
	}
}

func TestPreV1ContradictionsRejectedWithMigrationDocs(t *testing.T) {
	notes := readRepoFile(t, "docs", "v1.0", "RELEASE_NOTES.md")
	for _, required := range []string{
		"Pre-v1 behavior that contradicts `docs/SPEC.md` is rejected",
		"Use structured `raise error(...)`",
		"`catch err` only",
		"`defer` is not v1 syntax",
		"removed top-level primitive helpers",
	} {
		if !containsNormalized(notes, required) {
			t.Fatalf("v1 release notes missing migration text %q", required)
		}
	}
}

func TestSpecDocumentsEnvironmentProcessContract(t *testing.T) {
	spec := readRepoFile(t, "docs", "SPEC.md")
	for _, required := range []string{
		"### Environment And Process",
		"`environ()`",
		"`setenv(name, value)`",
		"`unsetenv(name)`",
		"`process/Process.run(command, options = {})`",
		"`options[\"shell\"] == true`",
		"`status`, `success`, `stdout`, `stderr`, and `timed_out`",
		"Non-zero child exit status is reported in the result dictionary",
	} {
		if !containsNormalized(spec, required) {
			t.Fatalf("SPEC.md missing environment/process text %q", required)
		}
	}
}

func TestSpecDocumentsFilesystemUtilities(t *testing.T) {
	spec := readRepoFile(t, "docs", "SPEC.md")
	for _, required := range []string{
		"### Filesystem Utilities",
		"`file/File.copy(src, dst, options = {})`",
		"`file/File.chmod(path, mode)`",
		"`dir/Dir.mkdir_all(path)`",
		"`dir/Dir.remove_all(path)`",
		"`dir/Dir.walk(path, fn, options = {})`",
		"`file/File.temp(prefix = \"tya\", suffix = \"\")`",
		"Windows permissions are best-effort",
	} {
		if !containsNormalized(spec, required) {
			t.Fatalf("SPEC.md missing filesystem utility text %q", required)
		}
	}
}

func TestSpecDocumentsHmacStdlib(t *testing.T) {
	spec := readRepoFile(t, "docs", "SPEC.md")
	for _, required := range []string{
		"`hmac/Hmac`",
		"`Hmac.digest(algorithm, key, message)`",
		"`Hmac.hexdigest(algorithm, key, message)`",
		"`Hmac.base64digest(algorithm, key, message)`",
		"`sha256`, `sha384`, and `sha512`",
		"constant-time comparison",
		"General encryption, public-key cryptography",
	} {
		if !containsNormalized(spec, required) {
			t.Fatalf("SPEC.md missing HMAC stdlib text %q", required)
		}
	}
}

func readRepoFile(t *testing.T, elems ...string) string {
	t.Helper()
	parts := append([]string{".."}, elems...)
	data, err := os.ReadFile(filepath.Join(parts...))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func containsNormalized(text, want string) bool {
	return strings.Contains(strings.Join(strings.Fields(text), " "), want)
}
