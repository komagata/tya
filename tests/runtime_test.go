package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"testing"
)

func TestCRuntimeValuesCollectionsAndFunctions(t *testing.T) {
	src := `#include "tya_runtime.h"

TyaValue double_value(TyaValue self, TyaValue item, TyaValue unused1, TyaValue unused2, TyaValue unused3) {
  return tya_number(item.number * 2);
}

int main(void) {
  TyaValue items = tya_array((TyaValue[]){tya_number(1), tya_number(2)}, 2);
  tya_push(items, tya_number(3));
  tya_print(tya_len(items));
  tya_print(tya_index(items, tya_number(2)));

  TyaValue dict = tya_dict((TyaDictEntry[]){{"name", tya_string("Tya")}}, 1);
  tya_set_member(dict, "version", tya_number(1));
  tya_print(dict);
  tya_print(tya_index(dict, tya_string("name")));
  tya_print(tya_index(dict, tya_string("version")));

  TyaValue doubled = tya_call1(tya_function(double_value), tya_index(items, tya_number(2)));
  tya_print(doubled);

  TyaValue err = tya_error(tya_string("bad"));
  tya_print(err);
  tya_print(tya_member(err, "message"));
  return 0;
}
`
	out := compileAndRunRuntime(t, src)
	want := "3\n3\n{name: Tya, version: 1}\nTya\n1\n6\nbad\nbad\n"
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

  TyaValue many = tya_array(0, 0);
  for (int i = 0; i < 10000; i++) {
    tya_push(many, tya_string("abc"));
  }
  tya_print(tya_len(tya_join(many, tya_string(","))));
  return 0;
}
`
	out := compileAndRunRuntime(t, src, path)
	want := "hello-tya\nhello,Tya\ntrue\ntrue\ntrue\n42\ntrue\nfile text\n39999\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestCRuntimeDictHashTablePreservesOrder(t *testing.T) {
	src := `#include "tya_runtime.h"
#include <stdio.h>
#include <stdlib.h>

int main(void) {
  TyaValue dict = tya_dict(0, 0);
  for (int i = 0; i < 40; i++) {
    char *key = malloc(16);
    snprintf(key, 16, "k%d", i);
    tya_dict_set(dict, tya_string(key), tya_number(i));
  }
  tya_dict_set(dict, tya_string("k10"), tya_string("ten"));
  tya_dict_delete(dict, tya_string("k5"));
  tya_dict_set(dict, tya_string("later"), tya_number(99));

  tya_print(tya_len(dict));
  tya_print(tya_dict_get(dict, tya_string("k10"), tya_nil(), false));
  tya_print(tya_has(dict, tya_string("k5")));
  tya_print(tya_dict_key_at(dict, tya_number(5)));
  tya_print(tya_dict_value_at(dict, tya_number(5)));
  tya_print(tya_dict_key_at(dict, tya_number(39)));
  tya_print(tya_values(dict));
  return 0;
}
`
	out := compileAndRunRuntime(t, src)
	want := "40\nten\nfalse\nk6\n6\nlater\n[0, 1, 2, 3, 4, 6, 7, 8, 9, ten, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 99]\n"
	if string(out) != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestCRuntimeCollectsIntermediateStringConcatenations(t *testing.T) {
	src := `#include "tya_runtime.h"
#include <stdio.h>

int main(void) {
  TyaValue out = tya_string("");
  tya_gc_register_root(&out);
  for (int i = 0; i < 100000; i++) {
    out = tya_add(out, tya_string("x"));
    tya_gc_maybe_collect();
  }
  tya_gc_collect();
  TyaValue stats = tya_gc_stats();
  printf("%.0f\n", tya_len(out).number);
  printf("%.0f\n", tya_dict_get(stats, tya_string("live_count"), tya_nil(), false).number);
  return 0;
}
`
	out := compileAndRunRuntime(t, src)
	if string(out) != "100000\n1\n" {
		t.Fatalf("got %q, want bounded live string count", out)
	}
}

func TestCRuntimeDictMapOwnsDynamicStringKeys(t *testing.T) {
	src := `#include "tya_runtime.h"

int main(void) {
  TyaValue dict = tya_dict(0, 0);
  tya_gc_register_root(&dict);
  TyaValue key = tya_add(tya_string("na"), tya_string("me"));
  tya_dict_set(dict, key, tya_string("kept"));
  tya_gc_collect();
  tya_print(tya_dict_get(dict, tya_string("name"), tya_nil(), false));
  return 0;
}
`
	out := compileAndRunRuntime(t, src)
	if string(out) != "kept\n" {
		t.Fatalf("got %q", out)
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
		t.Fatalf("exit(got) output %q and code %d", out, code)
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
		t.Fatalf("exit(code) %d\n%s", code, out)
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
	gccArgs := []string{cfile, runtime, "-I", include, "-o", bin}
	switch goruntime.GOOS {
	case "linux":
		gccArgs = append(gccArgs, "-lpthread", "-lm", "-lz")
	case "windows":
		// no extra flags
	default:
		gccArgs = append(gccArgs, "-lm", "-lz")
	}
	if out, err := exec.Command("gcc", gccArgs...).CombinedOutput(); err != nil {
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
