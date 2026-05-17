package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tya/internal/codegen"
	"tya/internal/runner"
)

func buildWasmExecutable(path string, output string, target string) (*codegen.CoverageRegistry, error) {
	zig, err := resolveZigToolchain()
	if err != nil {
		return nil, fmt.Errorf("WASM target %s requires managed Zig: %w", target, err)
	}
	if plan, err := nativePlanForPath(path); err != nil {
		return nil, err
	} else if plan != nil && len(plan.Packages) > 0 {
		return nil, fmt.Errorf("WASM target %s does not support native packages yet", target)
	}
	if err := checkWasmImports(path, target); err != nil {
		return nil, err
	}
	csrc, reg, err := compileToCWithCover(path, nil)
	if err != nil {
		return nil, err
	}
	if output == "" {
		output = defaultWasmOutput(path, target)
	}
	if target == "wasm32-browser" {
		if isExistingDir(output) || filepath.Ext(output) == "" {
			output = filepath.Join(output, "app.wasm")
		}
	}
	outDir, err := os.MkdirTemp("", "tya-wasm-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(outDir)
	cfile := filepath.Join(outDir, "main.c")
	if err := os.WriteFile(cfile, []byte(csrc), 0644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(outDir, "tya_runtime.h"), []byte(wasmRuntimeHeader), 0644); err != nil {
		return nil, err
	}
	runtimeSource := wasiRuntimeSource
	zigTarget := "wasm32-wasi"
	args := []string{"cc", "-target", zigTarget, cfile, filepath.Join(outDir, "runtime.c"), "-I", outDir, "-o", output}
	if target == "wasm32-browser" {
		runtimeSource = browserRuntimeSource
		zigTarget = "wasm32-freestanding"
		args = []string{"cc", "-target", zigTarget, "-nostdlib", "-fno-sanitize=undefined", cfile, filepath.Join(outDir, "runtime.c"), "-I", outDir, "-Wl,--no-entry", "-Wl,--export=main", "-Wl,--export=tya_output_ptr", "-Wl,--export=tya_output_len", "-Wl,--export=tya_output_reset", "-Wl,--export-memory", "-o", output}
	}
	if err := os.WriteFile(filepath.Join(outDir, "runtime.c"), []byte(runtimeSource), 0644); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return nil, err
	}
	cmd := zigCommand(zig.Path, args...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	if target == "wasm32-browser" {
		if err := writeBrowserLoader(output); err != nil {
			return nil, err
		}
	}
	return reg, nil
}

func checkWasmImports(path string, target string) error {
	source, modules, _, err := runner.LoadSourceWithOrigins(path)
	if err != nil {
		return err
	}
	all := source + "\n" + strings.Join(modules, "\n")
	blocked := []string{"import process", "import net", "import socket", "import http"}
	if target == "wasm32-browser" {
		blocked = append(blocked, "import file", "import io")
	}
	for _, marker := range blocked {
		if strings.Contains(all, marker) {
			return fmt.Errorf("import %q is not supported for target %s", strings.TrimPrefix(marker, "import "), target)
		}
	}
	return nil
}

func defaultWasmOutput(path string, target string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if target == "wasm32-browser" {
		return filepath.Join(base+"-browser", base+".wasm")
	}
	return base + ".wasm"
}

func isExistingDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func writeBrowserLoader(wasmPath string) error {
	jsPath := strings.TrimSuffix(wasmPath, filepath.Ext(wasmPath)) + ".js"
	wasmName := filepath.Base(wasmPath)
	loader := fmt.Sprintf(`export async function instantiateTya(options = {}) {
  const output = options.output || ((text) => console.log(text));
  const source = await WebAssembly.instantiateStreaming(fetch(new URL(%q, import.meta.url)), {});
  const instance = source.instance;
  return {
    instance,
    run(args = []) {
      instance.exports.tya_output_reset();
      const status = instance.exports.main(args.length, 0);
      const ptr = instance.exports.tya_output_ptr();
      const len = instance.exports.tya_output_len();
      if (len > 0) {
        const bytes = new Uint8Array(instance.exports.memory.buffer, ptr, len);
        output(new TextDecoder().decode(bytes));
      }
      return status;
    }
  };
}
`, wasmName)
	return os.WriteFile(jsPath, []byte(loader), 0644)
}

const wasmRuntimeHeader = `#ifndef TYA_RUNTIME_H
#define TYA_RUNTIME_H
#include <stdbool.h>
typedef enum { TYA_NIL, TYA_BOOL, TYA_NUMBER, TYA_STRING, TYA_ARRAY, TYA_DICT, TYA_OBJECT, TYA_FUNCTION, TYA_ERROR, TYA_BYTES, TYA_TASK, TYA_CHANNEL, TYA_RESOURCE, TYA_MISSING } TyaKind;
typedef struct TyaArray TyaArray;
typedef struct TyaDict TyaDict;
typedef struct TyaFunction TyaFunction;
typedef struct TyaValue TyaValue;
struct TyaValue { TyaKind kind; bool boolean; double number; const char *string; TyaArray *array; TyaDict *dict; TyaFunction *function; };
typedef struct { const char *key; TyaValue value; } TyaDictEntry;
typedef TyaValue (*TyaFunctionPtr)(TyaValue, TyaValue, TyaValue, TyaValue, TyaValue, TyaValue, TyaValue);
TyaValue tya_nil(void);
TyaValue tya_missing(void);
TyaValue tya_bool(bool value);
TyaValue tya_number(double value);
TyaValue tya_string(const char *value);
TyaValue tya_array(const TyaValue *items, int count);
TyaValue tya_dict(const TyaDictEntry *entries, int count);
TyaValue tya_object(void);
TyaValue tya_function_raw(TyaFunctionPtr fn);
TyaValue tya_class_raw(TyaFunctionPtr fn, const char *name, TyaValue parent);
TyaValue tya_bind_method_raw(TyaValue receiver, TyaFunctionPtr fn);
#define tya_function(fn) tya_function_raw((TyaFunctionPtr)(fn))
#define tya_class(fn, name, parent) tya_class_raw((TyaFunctionPtr)(fn), name, parent)
#define tya_bind_method(receiver, fn) tya_bind_method_raw(receiver, (TyaFunctionPtr)(fn))
TyaValue tya_call0(TyaValue fn);
TyaValue tya_call1(TyaValue fn, TyaValue arg);
TyaValue tya_call2(TyaValue fn, TyaValue first, TyaValue second);
TyaValue tya_index(TyaValue value, TyaValue index);
void tya_set_index(TyaValue value, TyaValue index, TyaValue item);
TyaValue tya_member(TyaValue value, const char *key);
void tya_set_member(TyaValue value, const char *key, TyaValue item);
TyaValue tya_args(int argc, char **argv);
TyaValue tya_env(TyaValue name);
TyaValue tya_read_file(TyaValue path);
void tya_write_file(TyaValue path, TyaValue text);
TyaValue tya_file_exists(TyaValue path);
TyaValue tya_file_remove(TyaValue path);
TyaValue tya_file_rename(TyaValue old_path, TyaValue new_path);
TyaValue tya_file_stat(TyaValue path);
TyaValue tya_file_read_bytes(TyaValue path);
TyaValue tya_file_write_bytes(TyaValue path, TyaValue b);
TyaValue tya_file_append(TyaValue path, TyaValue text);
void tya_panic(TyaValue value);
void tya_gc_register_root(TyaValue *root);
void tya_gc_maybe_collect(void);
void tya_print(TyaValue value);
TyaValue tya_to_string(TyaValue value);
TyaValue tya_add(TyaValue left, TyaValue right);
#endif
`

const wasiRuntimeSource = wasmRuntimeCommon + `#include <stdio.h>
void tya_write_out(const char *ptr, int len) { fwrite(ptr, 1, len, stdout); }
TyaValue tya_read_file(TyaValue path) {
  if (path.kind != TYA_STRING) return tya_nil();
  FILE *fp = fopen(path.string, "rb");
  if (!fp) return tya_nil();
  char *buf = tya_alloc(4096);
  int n = (int)fread(buf, 1, 4095, fp);
  fclose(fp);
  buf[n] = 0;
  return tya_string(buf);
}
void tya_write_file(TyaValue path, TyaValue text) {
  if (path.kind != TYA_STRING || text.kind != TYA_STRING) return;
  FILE *fp = fopen(path.string, "wb");
  if (!fp) return;
  fwrite(text.string, 1, tya_strlen2(text.string), fp);
  fclose(fp);
}
TyaValue tya_file_exists(TyaValue path) {
  if (path.kind != TYA_STRING) return tya_bool(false);
  FILE *fp = fopen(path.string, "rb");
  if (!fp) return tya_bool(false);
  fclose(fp);
  return tya_bool(true);
}
TyaValue tya_file_remove(TyaValue path) { (void)path; return tya_nil(); }
TyaValue tya_file_rename(TyaValue old_path, TyaValue new_path) { (void)old_path; (void)new_path; return tya_nil(); }
TyaValue tya_file_stat(TyaValue path) { (void)path; return tya_dict(NULL, 0); }
TyaValue tya_file_read_bytes(TyaValue path) { (void)path; return tya_nil(); }
TyaValue tya_file_write_bytes(TyaValue path, TyaValue b) { (void)path; (void)b; return tya_nil(); }
TyaValue tya_file_append(TyaValue path, TyaValue text) { (void)path; (void)text; return tya_nil(); }
`

const wasmRuntimeCommon = `#include "tya_runtime.h"
#include <stddef.h>
void tya_write_out(const char *ptr, int len);
struct TyaArray { int len; TyaValue *items; };
struct TyaDict { int len; int cap; TyaDictEntry *entries; };
struct TyaFunction { TyaFunctionPtr fn; TyaValue receiver; bool bound; TyaDict *members; };
static unsigned char tya_arena[65536];
static int tya_arena_pos = 0;
static void *tya_alloc(int size) {
  int aligned = (size + 7) & ~7;
  if (tya_arena_pos + aligned > 65536) return 0;
  void *ptr = tya_arena + tya_arena_pos;
  tya_arena_pos += aligned;
  return ptr;
}
static int tya_streq(const char *a, const char *b) {
  int i = 0;
  while (a && b && a[i] && b[i]) { if (a[i] != b[i]) return 0; i++; }
  return a && b && a[i] == b[i];
}
TyaValue tya_nil(void) { return (TyaValue){.kind = TYA_NIL}; }
TyaValue tya_missing(void) { return (TyaValue){.kind = TYA_MISSING}; }
TyaValue tya_bool(bool value) { return (TyaValue){.kind = TYA_BOOL, .boolean = value}; }
TyaValue tya_number(double value) { return (TyaValue){.kind = TYA_NUMBER, .number = value}; }
TyaValue tya_string(const char *value) { return (TyaValue){.kind = TYA_STRING, .string = value}; }
TyaValue tya_array(const TyaValue *items, int count) {
  TyaArray *arr = tya_alloc(sizeof(TyaArray));
  arr->len = count;
  arr->items = count == 0 ? NULL : tya_alloc(sizeof(TyaValue) * count);
  for (int i = 0; i < count; i++) arr->items[i] = items[i];
  return (TyaValue){.kind = TYA_ARRAY, .array = arr};
}
TyaValue tya_dict(const TyaDictEntry *entries, int count) {
  TyaDict *dict = tya_alloc(sizeof(TyaDict));
  dict->len = count;
  dict->cap = count + 16;
  dict->entries = tya_alloc(sizeof(TyaDictEntry) * dict->cap);
  for (int i = 0; i < count; i++) dict->entries[i] = entries[i];
  return (TyaValue){.kind = TYA_DICT, .dict = dict};
}
TyaValue tya_object(void) { TyaValue v = tya_dict(NULL, 0); v.kind = TYA_OBJECT; return v; }
TyaValue tya_function_raw(TyaFunctionPtr fn) {
  TyaFunction *f = tya_alloc(sizeof(TyaFunction));
  f->fn = fn; f->receiver = tya_nil(); f->bound = false; f->members = tya_dict(NULL, 0).dict;
  return (TyaValue){.kind = TYA_FUNCTION, .function = f};
}
TyaValue tya_class_raw(TyaFunctionPtr fn, const char *name, TyaValue parent) {
  (void)name; (void)parent;
  return tya_function_raw(fn);
}
TyaValue tya_bind_method_raw(TyaValue receiver, TyaFunctionPtr fn) {
  TyaFunction *f = tya_alloc(sizeof(TyaFunction));
  f->fn = fn; f->receiver = receiver; f->bound = true; f->members = tya_dict(NULL, 0).dict;
  return (TyaValue){.kind = TYA_FUNCTION, .function = f};
}
TyaValue tya_call0(TyaValue fn) {
  if (fn.kind != TYA_FUNCTION || fn.function == NULL || fn.function->fn == NULL) return tya_nil();
  return fn.function->fn(fn.function->receiver, tya_nil(), tya_nil(), tya_nil(), tya_nil(), tya_nil(), tya_nil());
}
TyaValue tya_call1(TyaValue fn, TyaValue arg) {
  if (fn.kind != TYA_FUNCTION || fn.function == NULL) return tya_nil();
  if (fn.function->bound) return fn.function->fn(fn.function->receiver, arg, tya_nil(), tya_nil(), tya_nil(), tya_nil(), tya_nil());
  return fn.function->fn(tya_nil(), arg, tya_nil(), tya_nil(), tya_nil(), tya_nil(), tya_nil());
}
TyaValue tya_call2(TyaValue fn, TyaValue first, TyaValue second) {
  if (fn.kind != TYA_FUNCTION || fn.function == NULL) return tya_nil();
  if (fn.function->bound) return fn.function->fn(fn.function->receiver, first, second, tya_nil(), tya_nil(), tya_nil(), tya_nil());
  return fn.function->fn(tya_nil(), first, second, tya_nil(), tya_nil(), tya_nil(), tya_nil());
}
static int dict_find(TyaDict *dict, const char *key) {
  if (dict == NULL) return -1;
  for (int i = 0; i < dict->len; i++) if (tya_streq(dict->entries[i].key, key)) return i;
  return -1;
}
TyaValue tya_index(TyaValue value, TyaValue index) {
  if (value.kind == TYA_ARRAY && value.array && index.kind == TYA_NUMBER) {
    int i = (int)index.number;
    if (i >= 0 && i < value.array->len) return value.array->items[i];
  }
  if ((value.kind == TYA_DICT || value.kind == TYA_OBJECT) && index.kind == TYA_STRING) {
    int i = dict_find(value.dict, index.string);
    if (i >= 0) return value.dict->entries[i].value;
  }
  return tya_nil();
}
void tya_set_index(TyaValue value, TyaValue index, TyaValue item) {
  if (value.kind == TYA_ARRAY && value.array && index.kind == TYA_NUMBER) {
    int i = (int)index.number;
    if (i >= 0 && i < value.array->len) value.array->items[i] = item;
    return;
  }
  if ((value.kind == TYA_DICT || value.kind == TYA_OBJECT) && index.kind == TYA_STRING) {
    tya_set_member(value, index.string, item);
  }
}
static TyaValue array_len_method(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)a; (void)b; (void)c; (void)d; (void)e; (void)f;
  return tya_number(receiver.kind == TYA_ARRAY && receiver.array ? receiver.array->len : 0);
}
TyaValue tya_member(TyaValue value, const char *key) {
  if (value.kind == TYA_ARRAY && tya_streq(key, "len")) return tya_bind_method(value, array_len_method);
  if (value.kind == TYA_DICT || value.kind == TYA_OBJECT || value.kind == TYA_FUNCTION) {
    TyaDict *dict = value.kind == TYA_FUNCTION && value.function ? value.function->members : value.dict;
    int i = dict_find(dict, key);
    if (i >= 0) return dict->entries[i].value;
  }
  return tya_nil();
}
void tya_set_member(TyaValue value, const char *key, TyaValue item) {
  if (value.kind != TYA_DICT && value.kind != TYA_OBJECT && value.kind != TYA_FUNCTION) return;
  TyaDict *dict = value.kind == TYA_FUNCTION && value.function ? value.function->members : value.dict;
  int i = dict_find(dict, key);
  if (i >= 0) { dict->entries[i].value = item; return; }
  if (dict->len >= dict->cap) return;
  dict->entries[dict->len++] = (TyaDictEntry){key, item};
}
void tya_gc_register_root(TyaValue *root) { (void)root; }
void tya_gc_maybe_collect(void) {}
TyaValue tya_args(int argc, char **argv) {
  TyaValue *items = tya_alloc(sizeof(TyaValue) * argc);
  for (int i = 0; i < argc; i++) items[i] = tya_string(argv[i]);
  return tya_array(items, argc);
}
TyaValue tya_env(TyaValue name) { (void)name; return tya_nil(); }
void tya_panic(TyaValue value) { tya_print(value); }
TyaValue tya_to_string(TyaValue value) { return value.kind == TYA_STRING ? value : tya_string(""); }
TyaValue tya_add(TyaValue left, TyaValue right) { (void)right; return left; }
static int tya_strlen2(const char *s) { int n = 0; while (s && s[n]) n++; return n; }
void tya_print(TyaValue value) {
  char buf[64];
  if (value.kind == TYA_STRING && value.string) tya_write_out(value.string, tya_strlen2(value.string));
  else if (value.kind == TYA_NUMBER) {
    long n = (long)value.number;
    int pos = 63; buf[pos--] = 0;
    if (n == 0) buf[pos--] = '0';
    bool neg = n < 0; if (neg) n = -n;
    while (n > 0 && pos >= 0) { buf[pos--] = (char)('0' + (n % 10)); n /= 10; }
    if (neg) buf[pos--] = '-';
    tya_write_out(&buf[pos + 1], 62 - pos);
  }
  else if (value.kind == TYA_BOOL) { if (value.boolean) tya_write_out("true", 4); else tya_write_out("false", 5); }
  else tya_write_out("nil", 3);
  tya_write_out("\n", 1);
}
`

const oldWasiRuntimeSource = `#include "tya_runtime.h"
#include <stdio.h>
TyaValue tya_nil(void) { return (TyaValue){.kind = TYA_NIL}; }
TyaValue tya_bool(bool value) { return (TyaValue){.kind = TYA_BOOL, .boolean = value}; }
TyaValue tya_number(double value) { return (TyaValue){.kind = TYA_NUMBER, .number = value}; }
TyaValue tya_string(const char *value) { return (TyaValue){.kind = TYA_STRING, .string = value}; }
TyaValue tya_to_string(TyaValue value) { return value.kind == TYA_STRING ? value : tya_string(""); }
TyaValue tya_add(TyaValue left, TyaValue right) { (void)right; return left; }
void tya_print(TyaValue value) {
  if (value.kind == TYA_STRING && value.string) fputs(value.string, stdout);
  else if (value.kind == TYA_NUMBER) printf("%g", value.number);
  else if (value.kind == TYA_BOOL) fputs(value.boolean ? "true" : "false", stdout);
  else fputs("nil", stdout);
  fputc('\n', stdout);
}
`

const browserRuntimeSource = wasmRuntimeCommon + `
static char tya_output[4096];
static int tya_output_n = 0;
int tya_output_ptr(void) { return (int)(unsigned long)tya_output; }
int tya_output_len(void) { return tya_output_n; }
void tya_output_reset(void) { tya_output_n = 0; }
void tya_write_out(const char *ptr, int len) {
  for (int i = 0; i < len && tya_output_n < 4096; i++) tya_output[tya_output_n++] = ptr[i];
}
`
