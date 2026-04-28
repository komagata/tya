#include "tya_runtime.h"

#include <stdio.h>

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
