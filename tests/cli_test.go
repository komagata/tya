package tests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLICheckUnusedRejectsUnusedBinding(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unused.tya")
	if err := os.WriteFile(path, []byte("name = \"Tya\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "--check-unused", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected --check-unused to fail")
	}
	if !strings.Contains(string(out), "unused variable name") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLICheckUnusedAllowsUsedBinding(t *testing.T) {
	path := filepath.Join(t.TempDir(), "used.tya")
	if err := os.WriteFile(path, []byte("name = \"Tya\"\nprint(name)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "--check-unused", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if string(out) != "" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIAllowsCombinedOptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "used.tya")
	if err := os.WriteFile(path, []byte("name = \"Tya\"\nprint(name)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "--check-unused", "--emit-c", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "tya_print") {
		t.Fatalf("expected emitted C, got: %s", out)
	}
}

func TestCLITokensCanBeCombinedWithChecks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "used.tya")
	if err := os.WriteFile(path, []byte("name = \"Tya\"\nprint(name)\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "--check-unused", "--tokens", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "IDENT") {
		t.Fatalf("expected tokens, got: %s", out)
	}
}

func TestCLIRunCompilesAndRunsProgram(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hello.tya")
	if err := os.WriteFile(path, []byte("print(\"Hello from run\")\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "run", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if string(out) != "Hello from run\n" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIDirectFileDoesNotUseInterpreterPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hello.tya")
	if err := os.WriteFile(path, []byte("print(\"direct\")\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected direct file invocation to fail")
	}
	if !strings.Contains(string(out), "usage: tya run") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIRunPassesArgs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "args.tya")
	if err := os.WriteFile(path, []byte("items = args()\nprint(items.len())\nprint(items[0])\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "run", path, "first")
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if string(out) != "1\nfirst\n" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIRunLoadsImportedModule(t *testing.T) {
	dir := t.TempDir()
	module := filepath.Join(dir, "greeting.tya")
	if err := os.WriteFile(module, []byte("hello = name -> \"Hello, {name}\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "main.tya")
	if err := os.WriteFile(path, []byte("import greeting\nprint(greeting.hello(\"komagata\"))\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "run", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if string(out) != "Hello, komagata\n" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIBuildWritesExecutable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.tya")
	if err := os.WriteFile(path, []byte("print(\"Hello from build\")\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "hello-bin")
	cmd := exec.Command("go", "run", "./cmd/tya", "build", path, "-o", bin)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	run := exec.Command(bin)
	out, err = run.CombinedOutput()
	if err != nil {
		t.Fatalf("built executable failed: %v\n%s", err, out)
	}
	if string(out) != "Hello from build\n" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIBuildUsesDefaultOutputPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "default_out.tya")
	if err := os.WriteFile(path, []byte("print(\"default output\")\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "build", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	bin := filepath.Join("..", "default_out")
	t.Cleanup(func() {
		_ = os.Remove(bin)
	})
	run := exec.Command(bin)
	out, err = run.CombinedOutput()
	if err != nil {
		t.Fatalf("built executable failed: %v\n%s", err, out)
	}
	if string(out) != "default output\n" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIBuildNativeTargetWritesExecutable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.tya")
	if err := os.WriteFile(path, []byte("print(\"native target\")\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "hello-native")
	cmd := exec.Command("go", "run", "./cmd/tya", "build", "--target", "native", path, "-o", bin)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	run := exec.Command(bin)
	out, err = run.CombinedOutput()
	if err != nil {
		t.Fatalf("built executable failed: %v\n%s", err, out)
	}
	if string(out) != "native target\n" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIBuildWasmTargets(t *testing.T) {
	if _, err := exec.LookPath("zig"); err != nil {
		t.Skip("zig not installed")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.tya")
	if err := os.WriteFile(path, []byte("print(\"hello wasm\")\n"), 0644); err != nil {
		t.Fatal(err)
	}
	wasiOut := filepath.Join(dir, "hello.wasm")
	cmd := exec.Command("go", "run", "./cmd/tya", "build", "--target", "wasm32-wasi", path, "-o", wasiOut)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wasi build failed: %v\n%s", err, out)
	}
	if info, err := os.Stat(wasiOut); err != nil || info.Size() == 0 {
		t.Fatalf("missing wasi wasm: %v", err)
	}
	browserOut := filepath.Join(dir, "browser", "hello.wasm")
	cmd = exec.Command("go", "run", "./cmd/tya", "build", "--target", "wasm32-browser", path, "-o", browserOut)
	cmd.Dir = ".."
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("browser build failed: %v\n%s", err, out)
	}
	if info, err := os.Stat(browserOut); err != nil || info.Size() == 0 {
		t.Fatalf("missing browser wasm: %v", err)
	}
	if info, err := os.Stat(filepath.Join(dir, "browser", "hello.js")); err != nil || info.Size() == 0 {
		t.Fatalf("missing browser loader: %v", err)
	}
}

func TestCLIBuildBrowserWasmSupportsBasicRuntimeValues(t *testing.T) {
	if _, err := exec.LookPath("zig"); err != nil {
		t.Skip("zig not installed")
	}
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not installed")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "basic.tya")
	source := strings.Join([]string{
		"class Box",
		"  initialize = value ->",
		"    self.value = value",
		"",
		"items = [1, 2]",
		"data = { name: \"tya\" }",
		"box = Box(\"ok\")",
		"print(items.len())",
		"print(data[\"name\"])",
		"print(box.value)",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	wasm := filepath.Join(dir, "basic.wasm")
	cmd := exec.Command("go", "run", "./cmd/tya", "build", "--target", "wasm32-browser", path, "-o", wasm)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("browser build failed: %v\n%s", err, out)
	}
	script := fmt.Sprintf(`
const fs = require('fs');
(async () => {
  const { instance } = await WebAssembly.instantiate(fs.readFileSync(%q), {});
  instance.exports.tya_output_reset();
  instance.exports.main(0, 0);
  const ptr = instance.exports.tya_output_ptr();
  const len = instance.exports.tya_output_len();
  process.stdout.write(new TextDecoder().decode(new Uint8Array(instance.exports.memory.buffer, ptr, len)));
})();
`, wasm)
	cmd = exec.Command("node", "-e", script)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node wasm failed: %v\n%s", err, out)
	}
	if string(out) != "2\ntya\nok\n" {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIBuildWasiSupportsArgsAndBasicFiles(t *testing.T) {
	if _, err := exec.LookPath("zig"); err != nil {
		t.Skip("zig not installed")
	}
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not installed")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "wasi_file.tya")
	source := strings.Join([]string{
		"import file as file",
		"print(args().len())",
		"file.File.write(\"tya-wasi-check.txt\", \"ok\")",
		"print(file.File.read(\"tya-wasi-check.txt\"))",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	wasm := filepath.Join(dir, "wasi_file.wasm")
	cmd := exec.Command("go", "run", "./cmd/tya", "build", "--target", "wasm32-wasi", path, "-o", wasm)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wasi build failed: %v\n%s", err, out)
	}
	script := fmt.Sprintf(`
const { WASI } = require('wasi');
const fs = require('fs');
(async () => {
  const wasi = new WASI({ version: 'preview1', args: ['wasi_file.wasm', 'arg1'], preopens: {'.': %q} });
  const { instance } = await WebAssembly.instantiate(fs.readFileSync(%q), wasi.getImportObject());
  wasi.start(instance);
})();
`, dir, wasm)
	cmd = exec.Command("node", "-e", script)
	cmd.Dir = dir
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node wasi failed: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "2\nok\n") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIBuildRejectsBadWasmInputs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.tya")
	if err := os.WriteFile(path, []byte("print(\"bad\")\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "build", "--target", "plan9", path)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected unsupported target to fail")
	}
	if !strings.Contains(string(out), "unsupported build target: plan9") {
		t.Fatalf("unexpected output: %s", out)
	}
	fileImport := filepath.Join(dir, "file_import.tya")
	if err := os.WriteFile(fileImport, []byte("import file as file\nprint(\"x\")\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("go", "run", "./cmd/tya", "build", "--target", "wasm32-browser", fileImport, "-o", filepath.Join(dir, "bad.wasm"))
	cmd.Dir = ".."
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected browser file import to fail")
	}
	if !strings.Contains(string(out), "not supported for target wasm32-browser") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIDoctorWasm(t *testing.T) {
	if _, err := exec.LookPath("zig"); err != nil {
		t.Skip("zig not installed")
	}
	cmd := exec.Command("go", "run", "./cmd/tya", "doctor", "wasm")
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("doctor wasm failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "wasm32-wasi") || !strings.Contains(string(out), "wasm32-browser") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCLIVersionCommand(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/tya", "version")
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if string(out) != "0.65.0\n" {
		t.Fatalf("unexpected output: %s", out)
	}
}
