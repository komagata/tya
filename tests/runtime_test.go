package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCRuntimeValuesCollectionsAndFunctions(t *testing.T) {
	src := `#include "tya_runtime.h"

TyaValue double_value(TyaValue self, TyaValue item, TyaValue unused1, TyaValue unused2) {
  return tya_number(item.number * 2);
}

int main(void) {
  TyaValue items = tya_array((TyaValue[]){tya_number(1), tya_number(2)}, 2);
  tya_push(items, tya_number(3));
  tya_print(tya_len(items));
  tya_print(tya_index(items, tya_number(2)));

  TyaValue object = tya_object((TyaObjectEntry[]){{"name", tya_string("Tya")}}, 1);
  tya_set_member(object, "version", tya_number(1));
  tya_print(object);
  tya_print(tya_member(object, "name"));
  tya_print(tya_member(object, "version"));

  TyaValue doubled = tya_map(items, tya_function(double_value));
  tya_print(tya_index(doubled, tya_number(2)));

  TyaValue err = tya_error(tya_string("bad"));
  tya_print(err);
  tya_print(tya_member(err, "message"));
  return 0;
}
`
	out := compileAndRunRuntime(t, src)
	want := "3\n3\n{name: Tya, version: 1}\nTya\n1\n6\nerror: bad\nbad\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestCRuntimeStringsFilesAndConversions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runtime.txt")
	src := `#include "tya_runtime.h"

int main(int argc, char **argv) {
  TyaValue text = tya_trim(tya_string("  hello,tya  "));
  TyaValue parts = tya_split(text, tya_string(","));
  tya_print(tya_join(parts, tya_string("-")));
  tya_print(tya_replace(text, tya_string("tya"), tya_string("Tya")));
  tya_print(tya_contains(text, tya_string("hello")));
  tya_print(tya_starts_with(text, tya_string("hello")));
  tya_print(tya_ends_with(text, tya_string("tya")));
  tya_print(tya_to_int(tya_string("42")));

  TyaValue path = tya_string(argv[1]);
  tya_write_file(path, tya_string("file text"));
  tya_print(tya_file_exists(path));
  tya_print(tya_read_file(path));
  return 0;
}
`
	out := compileAndRunRuntime(t, src, path)
	want := "hello-tya\nhello,Tya\ntrue\ntrue\ntrue\n42\ntrue\nfile text\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestCRuntimeProcessExitAndPanic(t *testing.T) {
	out, code := compileAndRunRuntimeAllowExit(t, `#include "tya_runtime.h"

int main(void) {
  tya_exit(tya_number(7));
  return 0;
}
`)
	if string(out) != "" || code != 7 {
		t.Fatalf("exit got output %q and code %d", out, code)
	}

	out, code = compileAndRunRuntimeAllowExit(t, `#include "tya_runtime.h"

int main(void) {
  tya_panic(tya_string("bad state"));
  return 0;
}
`)
	if string(out) != "panic: bad state\n" || code != 1 {
		t.Fatalf("panic got output %q and code %d", out, code)
	}
}

func compileAndRunRuntime(t *testing.T, src string, args ...string) []byte {
	t.Helper()
	out, code := compileAndRunRuntimeAllowExit(t, src, args...)
	if code != 0 {
		t.Fatalf("exit code %d\n%s", code, out)
	}
	return out
}

func compileAndRunRuntimeAllowExit(t *testing.T, src string, args ...string) ([]byte, int) {
	t.Helper()
	dir := t.TempDir()
	cfile := filepath.Join(dir, "main.c")
	bin := filepath.Join(dir, "main")
	if err := os.WriteFile(cfile, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	runtime := filepath.Join("..", "runtime", "tya_runtime.c")
	include := filepath.Join("..", "runtime")
	if out, err := exec.Command("gcc", cfile, runtime, "-I", include, "-o", bin).CombinedOutput(); err != nil {
		t.Fatalf("gcc: %v\n%s", err, out)
	}
	out, err := exec.Command(bin, args...).CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return out, exitErr.ExitCode()
		}
		t.Fatal(err)
	}
	return out, 0
}
