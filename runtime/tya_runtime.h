#ifndef TYA_RUNTIME_H
#define TYA_RUNTIME_H

#include <stdbool.h>

typedef enum {
  TYA_NIL,
  TYA_BOOL,
  TYA_NUMBER,
  TYA_STRING,
  TYA_ARRAY,
  TYA_OBJECT,
  TYA_SET,
  TYA_FUNCTION,
  TYA_ERROR,
} TyaKind;

typedef struct TyaArray TyaArray;
typedef struct TyaObject TyaObject;
typedef struct TyaObjectEntry TyaObjectEntry;
typedef struct TyaFunction TyaFunction;

typedef struct {
  TyaKind kind;
  bool boolean;
  double number;
  const char *string;
  TyaArray *array;
  TyaObject *object;
  TyaFunction *function;
  const char *error;
} TyaValue;

typedef TyaValue (*TyaFunctionPtr)(TyaValue, TyaValue, TyaValue, TyaValue, TyaValue);

struct TyaObjectEntry {
  const char *key;
  TyaValue value;
};

TyaValue tya_nil(void);
TyaValue tya_bool(bool value);
TyaValue tya_number(double value);
TyaValue tya_string(const char *value);
TyaValue tya_array(const TyaValue *items, int count);
TyaValue tya_object(const TyaObjectEntry *entries, int count);
TyaValue tya_set(const TyaValue *items, int count);
TyaValue tya_function(TyaFunctionPtr fn);
TyaValue tya_method(TyaFunctionPtr fn, TyaValue receiver);
TyaValue tya_error(TyaValue message);
TyaValue tya_call1(TyaValue fn, TyaValue arg);
TyaValue tya_call2(TyaValue fn, TyaValue first, TyaValue second);
TyaValue tya_call3(TyaValue fn, TyaValue first, TyaValue second, TyaValue third);
TyaValue tya_call4(TyaValue fn, TyaValue first, TyaValue second, TyaValue third, TyaValue fourth);
TyaValue tya_len(TyaValue value);
TyaValue tya_index(TyaValue value, TyaValue index);
void tya_set_index(TyaValue value, TyaValue index, TyaValue item);
TyaValue tya_member(TyaValue object, const char *key);
void tya_set_member(TyaValue object, const char *key, TyaValue value);
TyaValue tya_object_key_at(TyaValue object, TyaValue index);
TyaValue tya_object_value_at(TyaValue object, TyaValue index);
TyaValue tya_has(TyaValue object, TyaValue key);
TyaValue tya_keys(TyaValue object);
TyaValue tya_values(TyaValue object);
void tya_delete(TyaValue object, TyaValue key);
TyaValue tya_contains(TyaValue text, TyaValue part);
TyaValue tya_starts_with(TyaValue text, TyaValue prefix);
TyaValue tya_ends_with(TyaValue text, TyaValue suffix);
TyaValue tya_trim(TyaValue text);
TyaValue tya_replace(TyaValue text, TyaValue old, TyaValue replacement);
TyaValue tya_byte_len(TyaValue text);
TyaValue tya_char_len(TyaValue text);
bool tya_equal(TyaValue left, TyaValue right);
TyaValue tya_deep_equal(TyaValue left, TyaValue right);
TyaValue tya_add(TyaValue left, TyaValue right);
TyaValue tya_and(TyaValue left, TyaValue right);
TyaValue tya_or(TyaValue left, TyaValue right);
TyaValue tya_args(int argc, char **argv);
TyaValue tya_env(TyaValue name);
TyaValue tya_read_line(void);
TyaValue tya_read_file(TyaValue path);
void tya_write_file(TyaValue path, TyaValue text);
TyaValue tya_split(TyaValue text, TyaValue sep);
TyaValue tya_join(TyaValue array, TyaValue sep);
TyaValue tya_to_string(TyaValue value);
TyaValue tya_to_int(TyaValue value);
TyaValue tya_to_float(TyaValue value);
TyaValue tya_to_number(TyaValue value);
TyaValue tya_file_exists(TyaValue path);
TyaValue tya_map(TyaValue array, TyaValue fn);
TyaValue tya_filter(TyaValue array, TyaValue fn);
TyaValue tya_find(TyaValue array, TyaValue fn);
TyaValue tya_any(TyaValue array, TyaValue fn);
TyaValue tya_all(TyaValue array, TyaValue fn);
TyaValue tya_each(TyaValue array, TyaValue fn);
TyaValue tya_reduce(TyaValue array, TyaValue initial, TyaValue fn);
void tya_push(TyaValue array, TyaValue value);
TyaValue tya_pop(TyaValue array);
void tya_exit(TyaValue code);
void tya_panic(TyaValue message);
void tya_print(TyaValue value);
bool tya_truthy(TyaValue value);

#endif
