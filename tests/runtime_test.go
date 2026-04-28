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
	want := "3\n3\nTya\n1\n6\nerror: bad\nbad\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func compileAndRunRuntime(t *testing.T, src string) []byte {
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
	out, err := exec.Command(bin).CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	return out
}
