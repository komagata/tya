#ifndef TYA_RUNTIME_H
#define TYA_RUNTIME_H

#include <stdbool.h>

typedef enum {
  TYA_NIL,
  TYA_BOOL,
  TYA_NUMBER,
  TYA_STRING,
  TYA_ARRAY,
} TyaKind;

typedef struct TyaArray TyaArray;

typedef struct {
  TyaKind kind;
  bool boolean;
  double number;
  const char *string;
  TyaArray *array;
} TyaValue;

TyaValue tya_nil(void);
TyaValue tya_bool(bool value);
TyaValue tya_number(double value);
TyaValue tya_string(const char *value);
TyaValue tya_array(const TyaValue *items, int count);
TyaValue tya_len(TyaValue value);
TyaValue tya_index(TyaValue value, TyaValue index);
TyaValue tya_contains(TyaValue text, TyaValue part);
bool tya_equal(TyaValue left, TyaValue right);
void tya_push(TyaValue array, TyaValue value);
void tya_print(TyaValue value);
bool tya_truthy(TyaValue value);

#endif
