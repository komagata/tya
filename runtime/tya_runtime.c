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
