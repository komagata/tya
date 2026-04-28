#include "tya_runtime.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

struct TyaArray {
  int len;
  int cap;
  TyaValue *items;
};

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

TyaValue tya_len(TyaValue value) {
  if (value.kind == TYA_STRING && value.string != NULL) {
    int n = 0;
    while (value.string[n] != '\0') {
      n++;
    }
    return tya_number(n);
  }
  if (value.kind == TYA_ARRAY && value.array != NULL) {
    return tya_number(value.array->len);
  }
  return tya_number(0);
}

TyaValue tya_index(TyaValue value, TyaValue index) {
  int i = (int)index.number;
  if (value.kind == TYA_ARRAY && value.array != NULL && i >= 0 && i < value.array->len) {
    return value.array->items[i];
  }
  if (value.kind == TYA_STRING && value.string != NULL && i >= 0) {
    int n = 0;
    while (value.string[n] != '\0') {
      n++;
    }
    if (i < n) {
      char *out = malloc(2);
      out[0] = value.string[i];
      out[1] = '\0';
      return tya_string(out);
    }
  }
  return tya_nil();
}

TyaValue tya_contains(TyaValue text, TyaValue part) {
  if (text.kind != TYA_STRING || part.kind != TYA_STRING || text.string == NULL || part.string == NULL) {
    return tya_bool(false);
  }
  return tya_bool(strstr(text.string, part.string) != NULL);
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

TyaValue tya_args(int argc, char **argv) {
  TyaValue out = tya_array(0, 0);
  for (int i = 1; i < argc; i++) {
    tya_push(out, tya_string(argv[i]));
  }
  return out;
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
  char *out = malloc(64);
  switch (value.kind) {
  case TYA_NIL:
    snprintf(out, 64, "nil");
    break;
  case TYA_BOOL:
    snprintf(out, 64, "%s", value.boolean ? "true" : "false");
    break;
  case TYA_NUMBER:
    snprintf(out, 64, "%g", value.number);
    break;
  case TYA_ARRAY:
    snprintf(out, 64, "[array len=%d]", value.array == NULL ? 0 : value.array->len);
    break;
  case TYA_STRING:
    break;
  }
  return tya_string(out);
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

void tya_print(TyaValue value) {
  switch (value.kind) {
  case TYA_NIL:
    puts("nil");
    break;
  case TYA_BOOL:
    puts(value.boolean ? "true" : "false");
    break;
  case TYA_NUMBER:
    printf("%g\n", value.number);
    break;
  case TYA_STRING:
    puts(value.string);
    break;
  case TYA_ARRAY:
    printf("[array len=%d]\n", value.array == NULL ? 0 : value.array->len);
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
