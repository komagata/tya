#include "tya_runtime.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

struct TyaArray {
  int len;
  int cap;
  TyaValue *items;
};

struct TyaDict {
  int len;
  TyaDictEntry *entries;
};

struct TyaFunction {
  TyaFunctionPtr fn;
  TyaValue receiver;
};

typedef struct {
  char *text;
  size_t len;
  size_t cap;
} TyaStringBuilder;

static char *tya_substr(const char *text, int start, int len);
static int tya_string_len(const char *text);
static void tya_write_value(FILE *out, TyaValue value);
static void tya_build_value(TyaStringBuilder *builder, TyaValue value);
static void tya_builder_append(TyaStringBuilder *builder, const char *text);

TyaValue tya_nil(void) {
  return (TyaValue){.kind = TYA_NIL};
}

TyaValue tya_bool(bool value) {
  return (TyaValue){.kind = TYA_BOOL, .boolean = value};
}

TyaValue tya_number(double value) {
  return (TyaValue){.kind = TYA_NUMBER, .number = value};
}

TyaValue tya_string(const char *value) {
  return (TyaValue){.kind = TYA_STRING, .string = value};
}

TyaValue tya_array(const TyaValue *items, int count) {
  TyaArray *array = malloc(sizeof(TyaArray));
  int cap = count > 0 ? count : 4;
  array->len = count;
  array->cap = cap;
  array->items = malloc(sizeof(TyaValue) * cap);
  for (int i = 0; i < count; i++) {
    array->items[i] = items[i];
  }
  return (TyaValue){.kind = TYA_ARRAY, .array = array};
}

TyaValue tya_dict(const TyaDictEntry *entries, int count) {
  TyaDict *dict = malloc(sizeof(TyaDict));
  dict->len = count;
  dict->entries = malloc(sizeof(TyaDictEntry) * count);
  for (int i = 0; i < count; i++) {
    dict->entries[i] = entries[i];
  }
  return (TyaValue){.kind = TYA_DICT, .dict = dict};
}

TyaValue tya_function(TyaFunctionPtr fn) {
  TyaFunction *function = malloc(sizeof(TyaFunction));
  function->fn = fn;
  function->receiver = tya_nil();
  return (TyaValue){.kind = TYA_FUNCTION, .function = function};
}

TyaValue tya_error(TyaValue message) {
  if (message.kind != TYA_STRING) {
    return (TyaValue){.kind = TYA_ERROR, .error = ""};
  }
  return (TyaValue){.kind = TYA_ERROR, .error = message.string};
}

TyaValue tya_call1(TyaValue fn, TyaValue arg) {
  if (fn.kind != TYA_FUNCTION || fn.function == NULL || fn.function->fn == NULL) {
    return tya_nil();
  }
  return fn.function->fn(fn.function->receiver, arg, tya_nil(), tya_nil(), tya_nil());
}

TyaValue tya_call2(TyaValue fn, TyaValue first, TyaValue second) {
  if (fn.kind != TYA_FUNCTION || fn.function == NULL || fn.function->fn == NULL) {
    return tya_nil();
  }
  return fn.function->fn(fn.function->receiver, first, second, tya_nil(), tya_nil());
}

TyaValue tya_call3(TyaValue fn, TyaValue first, TyaValue second, TyaValue third) {
  if (fn.kind != TYA_FUNCTION || fn.function == NULL || fn.function->fn == NULL) {
    return tya_nil();
  }
  return fn.function->fn(fn.function->receiver, first, second, third, tya_nil());
}

TyaValue tya_call4(TyaValue fn, TyaValue first, TyaValue second, TyaValue third, TyaValue fourth) {
  if (fn.kind != TYA_FUNCTION || fn.function == NULL || fn.function->fn == NULL) {
    return tya_nil();
  }
  return fn.function->fn(fn.function->receiver, first, second, third, fourth);
}

TyaValue tya_len(TyaValue value) {
  if (value.kind == TYA_STRING && value.string != NULL) {
    return tya_number(tya_string_len(value.string));
  }
  if (value.kind == TYA_ARRAY && value.array != NULL) {
    return tya_number(value.array->len);
  }
  if (value.kind == TYA_DICT && value.dict != NULL) {
    int count = 0;
    for (int i = 0; i < value.dict->len; i++) {
      if (value.dict->entries[i].key != NULL) {
        count++;
      }
    }
    return tya_number(count);
  }
  return tya_number(0);
}

TyaValue tya_index(TyaValue value, TyaValue index) {
  int i = (int)index.number;
  if (value.kind == TYA_ARRAY && value.array != NULL && i >= 0 && i < value.array->len) {
    return value.array->items[i];
  }
  if (value.kind == TYA_STRING && value.string != NULL && i >= 0) {
    int n = tya_string_len(value.string);
    if (i < n) {
      char *out = malloc(2);
      out[0] = value.string[i];
      out[1] = '\0';
      return tya_string(out);
    }
  }
  if (value.kind == TYA_DICT && value.dict != NULL && index.kind == TYA_STRING && index.string != NULL) {
    return tya_member(value, index.string);
  }
  if (value.kind == TYA_ERROR && index.kind == TYA_STRING && index.string != NULL) {
    return tya_member(value, index.string);
  }
  return tya_nil();
}

static int tya_string_len(const char *text) {
  static const char *last_string = NULL;
  static int last_len = 0;
  if (text == last_string) {
    return last_len;
  }
  int n = 0;
  while (text[n] != '\0') {
    n++;
  }
  last_string = text;
  last_len = n;
  return n;
}

void tya_set_index(TyaValue value, TyaValue index, TyaValue item) {
  int i = (int)index.number;
  if (value.kind == TYA_ARRAY && value.array != NULL && i >= 0 && i < value.array->len) {
    value.array->items[i] = item;
  }
  if (value.kind == TYA_DICT && value.dict != NULL && index.kind == TYA_STRING && index.string != NULL) {
    tya_set_member(value, index.string, item);
  }
}

TyaValue tya_member(TyaValue dict, const char *key) {
  if (dict.kind == TYA_ERROR && strcmp(key, "message") == 0) {
    return tya_string(dict.error == NULL ? "" : dict.error);
  }
  if (dict.kind != TYA_DICT || dict.dict == NULL) {
    return tya_nil();
  }
  for (int i = 0; i < dict.dict->len; i++) {
    if (dict.dict->entries[i].key != NULL && strcmp(dict.dict->entries[i].key, key) == 0) {
      return dict.dict->entries[i].value;
    }
  }
  return tya_nil();
}

void tya_set_member(TyaValue dict, const char *key, TyaValue value) {
  if (dict.kind != TYA_DICT || dict.dict == NULL) {
    return;
  }
  for (int i = 0; i < dict.dict->len; i++) {
    if (dict.dict->entries[i].key != NULL && strcmp(dict.dict->entries[i].key, key) == 0) {
      dict.dict->entries[i].value = value;
      return;
    }
  }
  dict.dict->entries = realloc(dict.dict->entries, sizeof(TyaDictEntry) * (dict.dict->len + 1));
  dict.dict->entries[dict.dict->len] = (TyaDictEntry){key, value};
  dict.dict->len++;
}

TyaValue tya_dict_key_at(TyaValue dict, TyaValue index) {
  if (dict.kind != TYA_DICT || dict.dict == NULL) {
    return tya_nil();
  }
  int target = (int)index.number;
  int seen = 0;
  for (int i = 0; i < dict.dict->len; i++) {
    if (dict.dict->entries[i].key == NULL) {
      continue;
    }
    if (seen == target) {
      return tya_string(dict.dict->entries[i].key);
    }
    seen++;
  }
  return tya_nil();
}

TyaValue tya_dict_value_at(TyaValue dict, TyaValue index) {
  if (dict.kind != TYA_DICT || dict.dict == NULL) {
    return tya_nil();
  }
  int target = (int)index.number;
  int seen = 0;
  for (int i = 0; i < dict.dict->len; i++) {
    if (dict.dict->entries[i].key == NULL) {
      continue;
    }
    if (seen == target) {
      return dict.dict->entries[i].value;
    }
    seen++;
  }
  return tya_nil();
}

TyaValue tya_has(TyaValue dict, TyaValue key) {
  if (key.kind != TYA_STRING || key.string == NULL || dict.kind != TYA_DICT || dict.dict == NULL) {
    return tya_bool(false);
  }
  for (int i = 0; i < dict.dict->len; i++) {
    if (dict.dict->entries[i].key != NULL && strcmp(dict.dict->entries[i].key, key.string) == 0) {
      return tya_bool(true);
    }
  }
  return tya_bool(false);
}

TyaValue tya_keys(TyaValue dict) {
  TyaValue out = tya_array(0, 0);
  if (dict.kind != TYA_DICT || dict.dict == NULL) {
    return out;
  }
  for (int i = 0; i < dict.dict->len; i++) {
    if (dict.dict->entries[i].key != NULL) {
      tya_push(out, tya_string(dict.dict->entries[i].key));
    }
  }
  return out;
}

TyaValue tya_values(TyaValue dict) {
  TyaValue out = tya_array(0, 0);
  if (dict.kind != TYA_DICT || dict.dict == NULL) {
    return out;
  }
  for (int i = 0; i < dict.dict->len; i++) {
    if (dict.dict->entries[i].key != NULL) {
      tya_push(out, dict.dict->entries[i].value);
    }
  }
  return out;
}

void tya_delete(TyaValue dict, TyaValue key) {
  if (key.kind != TYA_STRING || key.string == NULL || dict.kind != TYA_DICT || dict.dict == NULL) {
    return;
  }
  for (int i = 0; i < dict.dict->len; i++) {
    if (dict.dict->entries[i].key != NULL && strcmp(dict.dict->entries[i].key, key.string) == 0) {
      dict.dict->entries[i].key = NULL;
      dict.dict->entries[i].value = tya_nil();
      return;
    }
  }
}

TyaValue tya_contains(TyaValue text, TyaValue part) {
  if (text.kind != TYA_STRING || part.kind != TYA_STRING || text.string == NULL || part.string == NULL) {
    return tya_bool(false);
  }
  return tya_bool(strstr(text.string, part.string) != NULL);
}

TyaValue tya_starts_with(TyaValue text, TyaValue prefix) {
  if (text.kind != TYA_STRING || prefix.kind != TYA_STRING || text.string == NULL || prefix.string == NULL) {
    return tya_bool(false);
  }
  return tya_bool(strncmp(text.string, prefix.string, strlen(prefix.string)) == 0);
}

TyaValue tya_ends_with(TyaValue text, TyaValue suffix) {
  if (text.kind != TYA_STRING || suffix.kind != TYA_STRING || text.string == NULL || suffix.string == NULL) {
    return tya_bool(false);
  }
  size_t text_len = strlen(text.string);
  size_t suffix_len = strlen(suffix.string);
  if (suffix_len > text_len) {
    return tya_bool(false);
  }
  return tya_bool(strcmp(text.string + text_len - suffix_len, suffix.string) == 0);
}

TyaValue tya_trim(TyaValue text) {
  if (text.kind != TYA_STRING || text.string == NULL) {
    return tya_string("");
  }
  int start = 0;
  int end = (int)strlen(text.string);
  while (start < end && (text.string[start] == ' ' || text.string[start] == '\n' || text.string[start] == '\t')) {
    start++;
  }
  while (end > start && (text.string[end - 1] == ' ' || text.string[end - 1] == '\n' || text.string[end - 1] == '\t')) {
    end--;
  }
  return tya_string(tya_substr(text.string, start, end - start));
}

TyaValue tya_replace(TyaValue text, TyaValue old, TyaValue replacement) {
  if (text.kind != TYA_STRING || old.kind != TYA_STRING || replacement.kind != TYA_STRING || text.string == NULL || old.string == NULL || replacement.string == NULL) {
    return tya_string("");
  }
  size_t old_len = strlen(old.string);
  if (old_len == 0) {
    return text;
  }
  size_t replacement_len = strlen(replacement.string);
  size_t count = 0;
  const char *cursor = text.string;
  while ((cursor = strstr(cursor, old.string)) != NULL) {
    count++;
    cursor += old_len;
  }
  size_t text_len = strlen(text.string);
  size_t out_len = text_len + count * (replacement_len - old_len);
  char *out = malloc(out_len + 1);
  char *dst = out;
  cursor = text.string;
  const char *next;
  while ((next = strstr(cursor, old.string)) != NULL) {
    size_t prefix_len = (size_t)(next - cursor);
    memcpy(dst, cursor, prefix_len);
    dst += prefix_len;
    memcpy(dst, replacement.string, replacement_len);
    dst += replacement_len;
    cursor = next + old_len;
  }
  strcpy(dst, cursor);
  return tya_string(out);
}

bool tya_equal(TyaValue left, TyaValue right) {
  if (left.kind != right.kind) {
    return false;
  }
  switch (left.kind) {
  case TYA_NIL:
    return true;
  case TYA_BOOL:
    return left.boolean == right.boolean;
  case TYA_NUMBER:
    return left.number == right.number;
  case TYA_STRING:
    if (left.string == NULL || right.string == NULL) {
      return left.string == right.string;
    }
    return strcmp(left.string, right.string) == 0;
  case TYA_ARRAY:
    return left.array == right.array;
  case TYA_DICT:
    return left.dict == right.dict;
  case TYA_FUNCTION:
    return left.function == right.function;
  case TYA_ERROR:
    if (left.error == NULL || right.error == NULL) {
      return left.error == right.error;
    }
    return strcmp(left.error, right.error) == 0;
  }
  return false;
}

TyaValue tya_add(TyaValue left, TyaValue right) {
  if (left.kind == TYA_STRING && right.kind == TYA_STRING && left.string != NULL && right.string != NULL) {
    int left_len = 0;
    int right_len = 0;
    while (left.string[left_len] != '\0') {
      left_len++;
    }
    while (right.string[right_len] != '\0') {
      right_len++;
    }
    char *out = malloc(left_len + right_len + 1);
    for (int i = 0; i < left_len; i++) {
      out[i] = left.string[i];
    }
    for (int i = 0; i < right_len; i++) {
      out[left_len + i] = right.string[i];
    }
    out[left_len + right_len] = '\0';
    return tya_string(out);
  }
  return tya_number(left.number + right.number);
}

TyaValue tya_and(TyaValue left, TyaValue right) {
  if (!tya_truthy(left)) {
    return left;
  }
  return right;
}

TyaValue tya_or(TyaValue left, TyaValue right) {
  if (tya_truthy(left)) {
    return left;
  }
  return right;
}

TyaValue tya_args(int argc, char **argv) {
  TyaValue out = tya_array(0, 0);
  for (int i = 1; i < argc; i++) {
    tya_push(out, tya_string(argv[i]));
  }
  return out;
}

TyaValue tya_env(TyaValue name) {
  if (name.kind != TYA_STRING || name.string == NULL) {
    return tya_nil();
  }
  const char *value = getenv(name.string);
  if (value == NULL) {
    return tya_nil();
  }
  return tya_string(value);
}

TyaValue tya_read_file(TyaValue path) {
  if (path.kind != TYA_STRING || path.string == NULL) {
    return tya_string("");
  }
  FILE *file = fopen(path.string, "rb");
  if (file == NULL) {
    return tya_string("");
  }
  fseek(file, 0, SEEK_END);
  long size = ftell(file);
  fseek(file, 0, SEEK_SET);
  char *buffer = malloc((size_t)size + 1);
  size_t read = fread(buffer, 1, (size_t)size, file);
  buffer[read] = '\0';
  fclose(file);
  return tya_string(buffer);
}

void tya_write_file(TyaValue path, TyaValue text) {
  if (path.kind != TYA_STRING || path.string == NULL || text.kind != TYA_STRING || text.string == NULL) {
    return;
  }
  FILE *file = fopen(path.string, "wb");
  if (file == NULL) {
    return;
  }
  fwrite(text.string, 1, strlen(text.string), file);
  fclose(file);
}

static char *tya_substr(const char *text, int start, int len) {
  char *out = malloc((size_t)len + 1);
  for (int i = 0; i < len; i++) {
    out[i] = text[start + i];
  }
  out[len] = '\0';
  return out;
}

TyaValue tya_split(TyaValue text, TyaValue sep) {
  if (text.kind != TYA_STRING || sep.kind != TYA_STRING || text.string == NULL || sep.string == NULL) {
    return tya_array(0, 0);
  }
  TyaValue out = tya_array(0, 0);
  int sep_len = (int)strlen(sep.string);
  if (sep_len == 0) {
    tya_push(out, text);
    return out;
  }
  int start = 0;
  int i = 0;
  while (text.string[i] != '\0') {
    if (strncmp(text.string + i, sep.string, (size_t)sep_len) == 0) {
      tya_push(out, tya_string(tya_substr(text.string, start, i - start)));
      i += sep_len;
      start = i;
      continue;
    }
    i++;
  }
  tya_push(out, tya_string(tya_substr(text.string, start, i - start)));
  return out;
}

TyaValue tya_join(TyaValue array, TyaValue sep) {
  if (array.kind != TYA_ARRAY || array.array == NULL || sep.kind != TYA_STRING || sep.string == NULL) {
    return tya_string("");
  }
  TyaValue out = tya_string("");
  for (int i = 0; i < array.array->len; i++) {
    if (i > 0) {
      out = tya_add(out, sep);
    }
    out = tya_add(out, tya_to_string(array.array->items[i]));
  }
  return out;
}

TyaValue tya_to_string(TyaValue value) {
  if (value.kind == TYA_STRING) {
    return value;
  }
  TyaStringBuilder builder = {.text = malloc(64), .len = 0, .cap = 64};
  builder.text[0] = '\0';
  tya_build_value(&builder, value);
  return tya_string(builder.text);
}

static void tya_build_value(TyaStringBuilder *builder, TyaValue value) {
  char scratch[64];
  switch (value.kind) {
  case TYA_NIL:
    tya_builder_append(builder, "nil");
    break;
  case TYA_BOOL:
    tya_builder_append(builder, value.boolean ? "true" : "false");
    break;
  case TYA_NUMBER:
    snprintf(scratch, sizeof(scratch), "%g", value.number);
    tya_builder_append(builder, scratch);
    break;
  case TYA_ARRAY:
    tya_builder_append(builder, "[");
    if (value.array != NULL) {
      for (int i = 0; i < value.array->len; i++) {
        if (i > 0) {
          tya_builder_append(builder, ", ");
        }
        tya_build_value(builder, value.array->items[i]);
      }
    }
    tya_builder_append(builder, "]");
    break;
  case TYA_DICT:
    tya_builder_append(builder, "{");
    if (value.dict != NULL) {
      int written = 0;
      for (int i = 0; i < value.dict->len; i++) {
        if (value.dict->entries[i].key == NULL) {
          continue;
        }
        if (written > 0) {
          tya_builder_append(builder, ", ");
        }
        tya_builder_append(builder, value.dict->entries[i].key);
        tya_builder_append(builder, ": ");
        tya_build_value(builder, value.dict->entries[i].value);
        written++;
      }
    }
    tya_builder_append(builder, "}");
    break;
  case TYA_FUNCTION:
    tya_builder_append(builder, "[function]");
    break;
  case TYA_ERROR:
    tya_builder_append(builder, "error: ");
    tya_builder_append(builder, value.error == NULL ? "" : value.error);
    break;
  case TYA_STRING:
    tya_builder_append(builder, value.string == NULL ? "" : value.string);
    break;
  }
}

static void tya_builder_append(TyaStringBuilder *builder, const char *text) {
  size_t text_len = strlen(text);
  while (builder->len + text_len + 1 > builder->cap) {
    builder->cap *= 2;
    builder->text = realloc(builder->text, builder->cap);
  }
  memcpy(builder->text + builder->len, text, text_len);
  builder->len += text_len;
  builder->text[builder->len] = '\0';
}

TyaValue tya_to_int(TyaValue value) {
  if (value.kind == TYA_NUMBER) {
    return value;
  }
  if (value.kind == TYA_STRING && value.string != NULL) {
    return tya_number(strtol(value.string, NULL, 10));
  }
  return tya_number(0);
}

TyaValue tya_to_float(TyaValue value) {
  if (value.kind == TYA_NUMBER) {
    return value;
  }
  if (value.kind == TYA_STRING && value.string != NULL) {
    return tya_number(strtod(value.string, NULL));
  }
  return tya_number(0);
}

TyaValue tya_to_number(TyaValue value) {
  return tya_to_float(value);
}

TyaValue tya_file_exists(TyaValue path) {
  if (path.kind != TYA_STRING || path.string == NULL) {
    return tya_bool(false);
  }
  return tya_bool(access(path.string, F_OK) == 0);
}

void tya_push(TyaValue array, TyaValue value) {
  if (array.kind != TYA_ARRAY || array.array == NULL) {
    return;
  }
  if (array.array->len >= array.array->cap) {
    array.array->cap *= 2;
    array.array->items = realloc(array.array->items, sizeof(TyaValue) * array.array->cap);
  }
  array.array->items[array.array->len] = value;
  array.array->len++;
}

TyaValue tya_pop(TyaValue array) {
  if (array.kind != TYA_ARRAY || array.array == NULL || array.array->len == 0) {
    return tya_nil();
  }
  array.array->len--;
  return array.array->items[array.array->len];
}

void tya_exit(TyaValue code) {
  if (code.kind == TYA_NUMBER) {
    exit((int)code.number);
  }
  exit(0);
}

void tya_panic(TyaValue message) {
  TyaValue text = tya_to_string(message);
  fprintf(stderr, "panic: %s\n", text.string == NULL ? "" : text.string);
  exit(1);
}

void tya_print(TyaValue value) {
  tya_write_value(stdout, value);
  putchar('\n');
}

static void tya_write_value(FILE *out, TyaValue value) {
  switch (value.kind) {
  case TYA_NIL:
    fprintf(out, "nil");
    break;
  case TYA_BOOL:
    fprintf(out, "%s", value.boolean ? "true" : "false");
    break;
  case TYA_NUMBER:
    fprintf(out, "%g", value.number);
    break;
  case TYA_STRING:
    fprintf(out, "%s", value.string);
    break;
  case TYA_ARRAY:
    fprintf(out, "[");
    if (value.array != NULL) {
      for (int i = 0; i < value.array->len; i++) {
        if (i > 0) {
          fprintf(out, ", ");
        }
        tya_write_value(out, value.array->items[i]);
      }
    }
    fprintf(out, "]");
    break;
  case TYA_DICT:
    fprintf(out, "{");
    if (value.dict != NULL) {
      int written = 0;
      for (int i = 0; i < value.dict->len; i++) {
        if (value.dict->entries[i].key == NULL) {
          continue;
        }
        if (written > 0) {
          fprintf(out, ", ");
        }
        fprintf(out, "%s: ", value.dict->entries[i].key);
        tya_write_value(out, value.dict->entries[i].value);
        written++;
      }
    }
    fprintf(out, "}");
    break;
  case TYA_FUNCTION:
    fprintf(out, "[function]");
    break;
  case TYA_ERROR:
    fprintf(out, "error: %s", value.error == NULL ? "" : value.error);
    break;
  }
}

bool tya_truthy(TyaValue value) {
  if (value.kind == TYA_NIL) {
    return false;
  }
  if (value.kind == TYA_BOOL) {
    return value.boolean;
  }
  return true;
}
