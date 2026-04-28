#ifndef TYA_RUNTIME_H
#define TYA_RUNTIME_H

#include <stdbool.h>

typedef enum {
  TYA_NIL,
  TYA_BOOL,
  TYA_NUMBER,
  TYA_STRING,
} TyaKind;

typedef struct {
  TyaKind kind;
  bool boolean;
  double number;
  const char *string;
} TyaValue;

TyaValue tya_nil(void);
TyaValue tya_bool(bool value);
TyaValue tya_number(double value);
TyaValue tya_string(const char *value);
void tya_print(TyaValue value);
bool tya_truthy(TyaValue value);

#endif
