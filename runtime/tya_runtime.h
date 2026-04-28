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
} TyaKind;

typedef struct TyaArray TyaArray;
typedef struct TyaObject TyaObject;
typedef struct TyaObjectEntry TyaObjectEntry;

typedef struct {
  TyaKind kind;
  bool boolean;
  double number;
  const char *string;
  TyaArray *array;
  TyaObject *object;
} TyaValue;

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
TyaValue tya_len(TyaValue value);
TyaValue tya_index(TyaValue value, TyaValue index);
void tya_set_index(TyaValue value, TyaValue index, TyaValue item);
TyaValue tya_member(TyaValue object, const char *key);
TyaValue tya_contains(TyaValue text, TyaValue part);
bool tya_equal(TyaValue left, TyaValue right);
TyaValue tya_add(TyaValue left, TyaValue right);
TyaValue tya_args(int argc, char **argv);
TyaValue tya_read_file(TyaValue path);
TyaValue tya_split(TyaValue text, TyaValue sep);
TyaValue tya_join(TyaValue array, TyaValue sep);
TyaValue tya_to_string(TyaValue value);
TyaValue tya_to_int(TyaValue value);
void tya_push(TyaValue array, TyaValue value);
TyaValue tya_pop(TyaValue array);
void tya_print(TyaValue value);
bool tya_truthy(TyaValue value);

#endif
