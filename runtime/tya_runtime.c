// glibc hides strptime / getrandom unless an X/Open or default-source
// feature-test macro is set. Define both so the runtime compiles with a
// stock cc invocation on Linux distributions that ship a strict default
// (e.g. Arch). Must precede every system header include.
#ifndef _XOPEN_SOURCE
#define _XOPEN_SOURCE 700
#endif
#ifndef _DEFAULT_SOURCE
#define _DEFAULT_SOURCE
#endif

#include "tya_runtime.h"

#include <ctype.h>
#include <dirent.h>
#include <errno.h>
#include <fcntl.h>
#include <math.h>
#include <pthread.h>
#include <signal.h>
#include <stdatomic.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/time.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <time.h>
#include <unistd.h>

#if defined(__APPLE__) || defined(__FreeBSD__) || defined(__OpenBSD__)
#include <sys/random.h>
#endif

/* GC infrastructure (v0.41).
 *
 * Every heap allocation that holds Tya runtime values (arrays, dicts,
 * functions, bytes) carries a TyaGcHeader as its first field. The
 * collector links headers into a single linked list rooted at
 * tya_gc_head, so it can iterate all live allocations.
 *
 * Roots: pointers to module-level TyaValue globals registered by
 * generated code at startup, plus the active raise-frame chain. Locals
 * inside user functions are NOT roots, so the collector must only run
 * at points where every live local is also reachable from these
 * globals (e.g. between top-level statements). See docs/v0.41/SPEC.md
 * for limitations and future work. */
typedef enum {
  TYA_GC_ARRAY = 1,
  TYA_GC_DICT = 2,
  TYA_GC_FUNCTION = 3,
  TYA_GC_BYTES = 4,
  TYA_GC_TASK = 5,
  TYA_GC_CHANNEL = 6,
  TYA_GC_RESOURCE = 7,
} TyaGcKind;

/* Sub-tag for the multi-purpose TyaResource container. v0.42 STEP 7
 * uses one container kind to host the three sync primitives so the
 * value-kind switch table stays compact. */
typedef enum {
  TYA_RES_MUTEX = 1,
  TYA_RES_ATOMIC_INTEGER = 2,
  TYA_RES_WAIT_GROUP = 3,
} TyaResourceSubkind;

typedef struct TyaGcHeader {
  unsigned char mark;
  unsigned char kind;
  size_t size;
  struct TyaGcHeader *next;
} TyaGcHeader;

struct TyaArray {
  TyaGcHeader gc;
  int len;
  int cap;
  TyaValue *items;
};

struct TyaBytes {
  TyaGcHeader gc;
  int len;
  uint8_t *data;
};

struct TyaDict {
  TyaGcHeader gc;
  int len;
  TyaDictEntry *entries;
};

struct TyaFunction {
  TyaGcHeader gc;
  TyaFunctionPtr fn;
  TyaValue receiver;
  TyaDict *members;
  const char *class_name;
  TyaValue parent;
  bool is_class;
};

/* TyaResource owns a sync primitive (mutex / atomic / wait group).
 * The subkind drives which fields are valid. */
struct TyaResource {
  TyaGcHeader gc;
  TyaResourceSubkind subkind;
  pthread_mutex_t mu;       /* mutex + wait_group */
  pthread_cond_t cv;        /* wait_group only */
  long counter;             /* wait_group counter */
  atomic_long atomic_value; /* atomic_integer only */
  bool mu_initialized;
  bool cv_initialized;
};

/* TyaChannel is the runtime representation of a channel value (v0.42).
 * Items are stored in a ring buffer protected by mu; sends wait on
 * not_full when the buffer is full and receives wait on not_empty when
 * empty. close() sets closed=true and broadcasts both condvars. */
struct TyaChannel {
  TyaGcHeader gc;
  TyaValue *buffer;
  int capacity;
  int len;
  int head;
  pthread_mutex_t mu;
  pthread_cond_t not_full;
  pthread_cond_t not_empty;
  bool closed;
};

/* TyaTask is the runtime representation of a task value (v0.42).
 * v0.42 STEP 2 only declares the struct and links it through the GC;
 * STEP 3 wires spawn / await codegen against this layout. */
struct TyaTask {
  TyaGcHeader gc;
  pthread_t thread;
  pthread_mutex_t mu;
  pthread_cond_t cv;
  bool done;
  bool joined;
  bool raised;
  atomic_bool cancelled;
  TyaValue callee;       /* the callable that the task runs */
  int argc;              /* number of arguments (0..4) */
  TyaValue argv[4];      /* arguments evaluated in the spawning thread */
  TyaValue result;       /* return value when done && !raised */
  TyaValue raise_value;  /* in-flight raise to propagate to await */
  /* Every not-yet-joined task lives in a global doubly-linked list so
   * the collector treats them as roots. Without this, a top-level
   * spawn whose handle is dropped before the worker finishes would
   * be reclaimed mid-flight, freeing its mutex and pthread state. */
  struct TyaTask *prev_live;
  struct TyaTask *next_live;
  bool in_live_list;
};

static TyaTask *tya_live_tasks = NULL;

static void tya_live_tasks_add(TyaTask *t);
static void tya_live_tasks_remove(TyaTask *t);

static TyaGcHeader *tya_gc_head = NULL;
static size_t tya_gc_alloc_count = 0;
static size_t tya_gc_alloc_bytes = 0;
static size_t tya_gc_freed_count = 0;
static size_t tya_gc_freed_bytes = 0;
static size_t tya_gc_collect_count = 0;
static size_t tya_gc_live_after_last = 0;
static size_t tya_gc_threshold = 1024;

static TyaValue **tya_gc_roots = NULL;
static size_t tya_gc_root_count = 0;
static size_t tya_gc_root_cap = 0;

/* tya_gc_mu serializes allocator state, the live-allocation list, the
 * global root array, and the collector. v0.42 uses a single mutex; an
 * M:N scheduler in a future minor will move this to a finer-grained
 * design. */
static pthread_mutex_t tya_gc_mu = PTHREAD_MUTEX_INITIALIZER;

static void *tya_gc_alloc(size_t size, TyaGcKind kind) {
  TyaGcHeader *header = (TyaGcHeader *)malloc(size);
  if (header == NULL) {
    fprintf(stderr, "tya: out of memory\n");
    exit(1);
  }
  header->mark = 0;
  header->kind = (unsigned char)kind;
  header->size = size;
  pthread_mutex_lock(&tya_gc_mu);
  header->next = tya_gc_head;
  tya_gc_head = header;
  tya_gc_alloc_count++;
  tya_gc_alloc_bytes += size;
  pthread_mutex_unlock(&tya_gc_mu);
  return header;
}

void tya_gc_register_root(TyaValue *p) {
  pthread_mutex_lock(&tya_gc_mu);
  if (tya_gc_root_count == tya_gc_root_cap) {
    size_t new_cap = tya_gc_root_cap == 0 ? 16 : tya_gc_root_cap * 2;
    tya_gc_roots = realloc(tya_gc_roots, sizeof(TyaValue *) * new_cap);
    tya_gc_root_cap = new_cap;
  }
  tya_gc_roots[tya_gc_root_count++] = p;
  pthread_mutex_unlock(&tya_gc_mu);
}

static void tya_gc_mark_value(TyaValue v);
static void tya_gc_mark_header(TyaGcHeader *h);

static void tya_gc_mark_header(TyaGcHeader *h) {
  if (h == NULL || h->mark != 0) return;
  h->mark = 1;
  switch ((TyaGcKind)h->kind) {
    case TYA_GC_ARRAY: {
      TyaArray *a = (TyaArray *)h;
      for (int i = 0; i < a->len; i++) {
        tya_gc_mark_value(a->items[i]);
      }
      break;
    }
    case TYA_GC_DICT: {
      TyaDict *d = (TyaDict *)h;
      for (int i = 0; i < d->len; i++) {
        if (d->entries[i].key != NULL) {
          tya_gc_mark_value(d->entries[i].value);
        }
      }
      break;
    }
    case TYA_GC_FUNCTION: {
      TyaFunction *f = (TyaFunction *)h;
      tya_gc_mark_value(f->receiver);
      tya_gc_mark_value(f->parent);
      if (f->members) {
        tya_gc_mark_header((TyaGcHeader *)f->members);
      }
      break;
    }
    case TYA_GC_BYTES:
      /* leaf */
      break;
    case TYA_GC_TASK: {
      TyaTask *t = (TyaTask *)h;
      tya_gc_mark_value(t->callee);
      for (int i = 0; i < t->argc; i++) {
        tya_gc_mark_value(t->argv[i]);
      }
      tya_gc_mark_value(t->result);
      tya_gc_mark_value(t->raise_value);
      break;
    }
    case TYA_GC_CHANNEL: {
      TyaChannel *c = (TyaChannel *)h;
      pthread_mutex_lock(&c->mu);
      for (int i = 0; i < c->len; i++) {
        int idx = (c->head + i) % c->capacity;
        tya_gc_mark_value(c->buffer[idx]);
      }
      pthread_mutex_unlock(&c->mu);
      break;
    }
    case TYA_GC_RESOURCE:
      /* leaf — sync primitives hold no Tya values */
      break;
  }
}

static void tya_gc_mark_value(TyaValue v) {
  switch (v.kind) {
    case TYA_ARRAY:
      if (v.array) tya_gc_mark_header((TyaGcHeader *)v.array);
      break;
    case TYA_DICT:
    case TYA_OBJECT:
      if (v.dict) tya_gc_mark_header((TyaGcHeader *)v.dict);
      break;
    case TYA_FUNCTION:
      if (v.function) tya_gc_mark_header((TyaGcHeader *)v.function);
      break;
    case TYA_BYTES:
      if (v.bytes) tya_gc_mark_header((TyaGcHeader *)v.bytes);
      break;
    case TYA_TASK:
      if (v.task) tya_gc_mark_header((TyaGcHeader *)v.task);
      break;
    case TYA_CHANNEL:
      if (v.channel) tya_gc_mark_header((TyaGcHeader *)v.channel);
      break;
    case TYA_RESOURCE:
      if (v.resource) tya_gc_mark_header((TyaGcHeader *)v.resource);
      break;
    default:
      break;
  }
}

static void tya_gc_free_one(TyaGcHeader *h) {
  switch ((TyaGcKind)h->kind) {
    case TYA_GC_ARRAY: {
      TyaArray *a = (TyaArray *)h;
      free(a->items);
      free(a);
      break;
    }
    case TYA_GC_DICT: {
      TyaDict *d = (TyaDict *)h;
      free(d->entries);
      free(d);
      break;
    }
    case TYA_GC_FUNCTION: {
      TyaFunction *f = (TyaFunction *)h;
      /* members is a separately tracked TyaDict; the collector frees it
       * on its own pass through the linked list if it is unreachable. */
      free(f);
      break;
    }
    case TYA_GC_BYTES: {
      TyaBytes *b = (TyaBytes *)h;
      free(b->data);
      free(b);
      break;
    }
    case TYA_GC_TASK: {
      TyaTask *t = (TyaTask *)h;
      pthread_mutex_destroy(&t->mu);
      pthread_cond_destroy(&t->cv);
      free(t);
      break;
    }
    case TYA_GC_CHANNEL: {
      TyaChannel *c = (TyaChannel *)h;
      pthread_mutex_destroy(&c->mu);
      pthread_cond_destroy(&c->not_full);
      pthread_cond_destroy(&c->not_empty);
      free(c->buffer);
      free(c);
      break;
    }
    case TYA_GC_RESOURCE: {
      TyaResource *r = (TyaResource *)h;
      if (r->mu_initialized) pthread_mutex_destroy(&r->mu);
      if (r->cv_initialized) pthread_cond_destroy(&r->cv);
      free(r);
      break;
    }
  }
}

static void tya_gc_sweep(void) {
  TyaGcHeader **prev = &tya_gc_head;
  TyaGcHeader *h = *prev;
  while (h) {
    TyaGcHeader *next = h->next;
    if (h->mark == 0) {
      size_t freed = h->size;
      tya_gc_free_one(h);
      tya_gc_freed_count++;
      tya_gc_freed_bytes += freed;
      *prev = next;
    } else {
      h->mark = 0;
      prev = &h->next;
    }
    h = next;
  }
}

typedef struct {
  char *text;
  size_t len;
  size_t cap;
} TyaStringBuilder;

static char *tya_substr(const char *text, int start, int len);
static int tya_string_len(const char *text);
static bool tya_deep_equal_bool(TyaValue left, TyaValue right);
static void tya_write_value(FILE *out, TyaValue value);
static void tya_build_value(TyaStringBuilder *builder, TyaValue value);
static void tya_builder_append(TyaStringBuilder *builder, const char *text);

/* Each task (including the main thread) has its own raise-frame chain.
 * Storing it as _Thread_local keeps tya_raise / tya_pop_raise_frame
 * unchanged in single-threaded code while letting workers raise
 * independently. The collector only walks the main thread's chain when
 * it runs, which is safe because the main thread holds tya_gc_mu
 * throughout collection and worker threads cannot allocate or raise
 * while waiting on that lock. */
static _Thread_local TyaRaiseFrame *tya_raise_frame = NULL;

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
  TyaArray *array = tya_gc_alloc(sizeof(TyaArray), TYA_GC_ARRAY);
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
  TyaDict *dict = tya_gc_alloc(sizeof(TyaDict), TYA_GC_DICT);
  dict->len = count;
  dict->entries = malloc(sizeof(TyaDictEntry) * count);
  for (int i = 0; i < count; i++) {
    dict->entries[i] = entries[i];
  }
  return (TyaValue){.kind = TYA_DICT, .dict = dict};
}

TyaValue tya_object(void) {
  TyaDict *dict = tya_gc_alloc(sizeof(TyaDict), TYA_GC_DICT);
  dict->len = 0;
  dict->entries = NULL;
  return (TyaValue){.kind = TYA_OBJECT, .dict = dict};
}

TyaValue tya_function(TyaFunctionPtr fn) {
  TyaFunction *function = tya_gc_alloc(sizeof(TyaFunction), TYA_GC_FUNCTION);
  function->fn = fn;
  function->receiver = tya_nil();
  function->members = tya_gc_alloc(sizeof(TyaDict), TYA_GC_DICT);
  function->members->len = 0;
  function->members->entries = NULL;
  function->class_name = NULL;
  function->parent = tya_nil();
  function->is_class = false;
  return (TyaValue){.kind = TYA_FUNCTION, .function = function};
}

TyaValue tya_class(TyaFunctionPtr fn, const char *name, TyaValue parent) {
  TyaValue value = tya_function(fn);
  value.function->class_name = name;
  value.function->parent = parent;
  value.function->is_class = true;
  return value;
}

TyaValue tya_bind_method(TyaValue receiver, TyaFunctionPtr fn) {
  TyaFunction *function = tya_gc_alloc(sizeof(TyaFunction), TYA_GC_FUNCTION);
  function->fn = fn;
  function->receiver = receiver;
  function->members = tya_gc_alloc(sizeof(TyaDict), TYA_GC_DICT);
  function->members->len = 0;
  function->members->entries = NULL;
  function->class_name = NULL;
  function->parent = tya_nil();
  function->is_class = false;
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
  if (value.kind == TYA_BYTES && value.bytes != NULL) {
    return tya_number(value.bytes->len);
  }
  if ((value.kind == TYA_DICT || value.kind == TYA_OBJECT) && value.dict != NULL) {
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
  if (value.kind == TYA_BYTES && value.bytes != NULL && i >= 0 && i < value.bytes->len) {
    return tya_number((double)value.bytes->data[i]);
  }
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
  if ((value.kind == TYA_DICT || value.kind == TYA_OBJECT) && value.dict != NULL && index.kind == TYA_STRING && index.string != NULL) {
    return tya_member(value, index.string);
  }
  if (value.kind == TYA_ERROR && index.kind == TYA_STRING && index.string != NULL) {
    return tya_member(value, index.string);
  }
  return tya_nil();
}

TyaValue tya_destructure_array(TyaValue value, int expected, int index) {
  if (value.kind != TYA_ARRAY || value.array == NULL) {
    tya_panic(tya_string("array destructuring target is not array"));
  }
  if (value.array->len != expected) {
    char message[96];
    snprintf(message, sizeof(message), "array destructuring expects %d elements, got %d", expected, value.array->len);
    tya_panic(tya_string(message));
  }
  if (index < 0 || index >= value.array->len) {
    tya_panic(tya_string("array destructuring index out of range"));
  }
  return value.array->items[index];
}

TyaValue tya_destructure_dict(TyaValue value, const char *key) {
  if (value.kind != TYA_DICT || value.dict == NULL) {
    tya_panic(tya_string("dictionary destructuring target is not dictionary"));
  }
  for (int i = 0; i < value.dict->len; i++) {
    if (value.dict->entries[i].key != NULL && strcmp(value.dict->entries[i].key, key) == 0) {
      return value.dict->entries[i].value;
    }
  }
  char message[256];
  snprintf(message, sizeof(message), "dictionary destructuring missing key %s", key == NULL ? "" : key);
  tya_panic(tya_string(message));
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
  if ((value.kind == TYA_DICT || value.kind == TYA_OBJECT) && value.dict != NULL && index.kind == TYA_STRING && index.string != NULL) {
    tya_set_member(value, index.string, item);
  }
}

TyaValue tya_member(TyaValue dict, const char *key) {
  if (dict.kind == TYA_ERROR && strcmp(key, "message") == 0) {
    return tya_string(dict.error == NULL ? "" : dict.error);
  }
  if (dict.kind == TYA_FUNCTION && dict.function != NULL && dict.function->members != NULL) {
    if (dict.function->is_class && key != NULL && strcmp(key, "name") == 0) {
      return tya_string(dict.function->class_name == NULL ? "" : dict.function->class_name);
    }
    if (dict.function->is_class && key != NULL && strcmp(key, "parent") == 0) {
      return dict.function->parent;
    }
    for (int i = 0; i < dict.function->members->len; i++) {
      if (dict.function->members->entries[i].key != NULL && strcmp(dict.function->members->entries[i].key, key) == 0) {
        return dict.function->members->entries[i].value;
      }
    }
    if (dict.function->is_class && dict.function->parent.kind == TYA_FUNCTION) {
      TyaValue inherited = tya_member(dict.function->parent, key);
      if (inherited.kind == TYA_FUNCTION && inherited.function != NULL && inherited.function->fn != NULL) {
        return tya_bind_method(dict, inherited.function->fn);
      }
      return inherited;
    }
    fprintf(stderr, "missing class variable or method: %s\n", key == NULL ? "" : key);
    exit(1);
  }
  if ((dict.kind != TYA_DICT && dict.kind != TYA_OBJECT) || dict.dict == NULL) {
    return tya_nil();
  }
  for (int i = 0; i < dict.dict->len; i++) {
    if (dict.dict->entries[i].key != NULL && strcmp(dict.dict->entries[i].key, key) == 0) {
      return dict.dict->entries[i].value;
    }
  }
  if (dict.kind == TYA_OBJECT) {
    fprintf(stderr, "missing object field or method: %s\n", key == NULL ? "" : key);
    exit(1);
  }
  return tya_nil();
}

void tya_set_member(TyaValue dict, const char *key, TyaValue value) {
  if (dict.kind == TYA_FUNCTION && dict.function != NULL && dict.function->members != NULL) {
    for (int i = 0; i < dict.function->members->len; i++) {
      if (dict.function->members->entries[i].key != NULL && strcmp(dict.function->members->entries[i].key, key) == 0) {
        dict.function->members->entries[i].value = value;
        return;
      }
    }
    dict.function->members->entries = realloc(dict.function->members->entries, sizeof(TyaDictEntry) * (dict.function->members->len + 1));
    dict.function->members->entries[dict.function->members->len] = (TyaDictEntry){key, value};
    dict.function->members->len++;
    return;
  }
  if ((dict.kind != TYA_DICT && dict.kind != TYA_OBJECT) || dict.dict == NULL) {
    return;
  }
  if (dict.kind == TYA_OBJECT && value.kind != TYA_FUNCTION && key != NULL && key[0] != '@') {
    size_t hidden_len = strlen(key) + 2;
    char *hidden_key = malloc(hidden_len);
    snprintf(hidden_key, hidden_len, "@%s", key);
    for (int i = 0; i < dict.dict->len; i++) {
      if (dict.dict->entries[i].key != NULL && strcmp(dict.dict->entries[i].key, hidden_key) == 0) {
        dict.dict->entries[i].value = value;
        break;
      }
    }
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
  (void)tya_dict_delete(dict, key);
}

TyaValue tya_dict_get(TyaValue dict, TyaValue key, TyaValue fallback, bool has_fallback) {
  if (key.kind != TYA_STRING || key.string == NULL || dict.kind != TYA_DICT || dict.dict == NULL) {
    return has_fallback ? fallback : tya_nil();
  }
  for (int i = 0; i < dict.dict->len; i++) {
    if (dict.dict->entries[i].key != NULL && strcmp(dict.dict->entries[i].key, key.string) == 0) {
      return dict.dict->entries[i].value;
    }
  }
  return has_fallback ? fallback : tya_nil();
}

TyaValue tya_dict_set(TyaValue dict, TyaValue key, TyaValue value) {
  if (key.kind != TYA_STRING || key.string == NULL || dict.kind != TYA_DICT || dict.dict == NULL) {
    return dict;
  }
  tya_set_index(dict, key, value);
  return dict;
}

TyaValue tya_dict_delete(TyaValue dict, TyaValue key) {
  if (key.kind != TYA_STRING || key.string == NULL || dict.kind != TYA_DICT || dict.dict == NULL) {
    return tya_nil();
  }
  for (int i = 0; i < dict.dict->len; i++) {
    if (dict.dict->entries[i].key != NULL && strcmp(dict.dict->entries[i].key, key.string) == 0) {
      TyaValue value = dict.dict->entries[i].value;
      dict.dict->entries[i].key = NULL;
      dict.dict->entries[i].value = tya_nil();
      return value;
    }
  }
  return tya_nil();
}

TyaValue tya_dict_merge(TyaValue left, TyaValue right) {
  TyaValue out = tya_dict(NULL, 0);
  if (left.kind == TYA_DICT && left.dict != NULL) {
    for (int i = 0; i < left.dict->len; i++) {
      if (left.dict->entries[i].key != NULL) {
        tya_set_index(out, tya_string(left.dict->entries[i].key), left.dict->entries[i].value);
      }
    }
  }
  if (right.kind == TYA_DICT && right.dict != NULL) {
    for (int i = 0; i < right.dict->len; i++) {
      if (right.dict->entries[i].key != NULL) {
        tya_set_index(out, tya_string(right.dict->entries[i].key), right.dict->entries[i].value);
      }
    }
  }
  return out;
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

TyaValue tya_byte_len(TyaValue text) {
  if (text.kind != TYA_STRING || text.string == NULL) {
    return tya_number(0);
  }
  return tya_number((double)strlen(text.string));
}

TyaValue tya_ord(TyaValue text) {
  if (text.kind != TYA_STRING || text.string == NULL || text.string[0] == '\0') {
    tya_raise(tya_string("ord: argument must be a non-empty string"));
    return tya_nil();
  }
  return tya_number((double)((unsigned char)text.string[0]));
}

TyaValue tya_kind(TyaValue value) {
  switch (value.kind) {
  case TYA_NIL:
    return tya_string("nil");
  case TYA_BOOL:
    return tya_string("bool");
  case TYA_NUMBER: {
    double d = value.number;
    if (d == (double)((long)d)) {
      return tya_string("int");
    }
    return tya_string("float");
  }
  case TYA_STRING:
    return tya_string("string");
  case TYA_ARRAY:
    return tya_string("array");
  case TYA_DICT:
    return tya_string("dict");
  case TYA_OBJECT:
    return tya_string("object");
  case TYA_FUNCTION:
    return tya_string("function");
  case TYA_ERROR:
    return tya_string("error");
  case TYA_BYTES:
    return tya_string("bytes");
  case TYA_TASK:
    return tya_string("task");
  case TYA_CHANNEL:
    return tya_string("channel");
  case TYA_RESOURCE:
    if (value.resource == NULL) return tya_string("resource");
    switch (value.resource->subkind) {
      case TYA_RES_MUTEX: return tya_string("mutex");
      case TYA_RES_ATOMIC_INTEGER: return tya_string("atomic_integer");
      case TYA_RES_WAIT_GROUP: return tya_string("wait_group");
    }
    return tya_string("resource");
  }
  return tya_string("unknown");
}

TyaValue tya_chr(TyaValue code) {
  if (code.kind != TYA_NUMBER) {
    tya_raise(tya_string("chr: argument must be an int"));
    return tya_nil();
  }
  int v = (int)code.number;
  if (v < 0 || v > 255) {
    tya_raise(tya_string("chr: byte value out of range (0..255)"));
    return tya_nil();
  }
  char *out = malloc(2);
  out[0] = (char)v;
  out[1] = '\0';
  return tya_string(out);
}

TyaValue tya_lines(TyaValue text) {
  TyaValue out = tya_array(NULL, 0);
  if (text.kind != TYA_STRING || text.string == NULL || text.string[0] == '\0') {
    return out;
  }
  int start = 0;
  int len = (int)strlen(text.string);
  while (len > 0 && (text.string[len - 1] == '\n' || text.string[len - 1] == '\r')) {
    len--;
  }
  for (int i = 0; i <= len; i++) {
    if (i == len || text.string[i] == '\n') {
      int end = i;
      if (end > start && text.string[end - 1] == '\r') {
        end--;
      }
      tya_push(out, tya_string(tya_substr(text.string, start, end - start)));
      start = i + 1;
    }
  }
  return out;
}

TyaValue tya_upcase(TyaValue text) {
  if (text.kind != TYA_STRING || text.string == NULL) {
    return tya_string("");
  }
  int len = (int)strlen(text.string);
  char *out = malloc((size_t)len + 1);
  for (int i = 0; i < len; i++) {
    out[i] = (char)toupper((unsigned char)text.string[i]);
  }
  out[len] = '\0';
  return tya_string(out);
}

TyaValue tya_downcase(TyaValue text) {
  if (text.kind != TYA_STRING || text.string == NULL) {
    return tya_string("");
  }
  int len = (int)strlen(text.string);
  char *out = malloc((size_t)len + 1);
  for (int i = 0; i < len; i++) {
    out[i] = (char)tolower((unsigned char)text.string[i]);
  }
  out[len] = '\0';
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
  case TYA_OBJECT:
    return left.dict == right.dict;
  case TYA_FUNCTION:
    return left.function == right.function;
  case TYA_ERROR:
    if (left.error == NULL || right.error == NULL) {
      return left.error == right.error;
    }
    return strcmp(left.error, right.error) == 0;
  case TYA_BYTES:
    if (left.bytes == NULL || right.bytes == NULL) {
      return left.bytes == right.bytes;
    }
    if (left.bytes->len != right.bytes->len) {
      return false;
    }
    return memcmp(left.bytes->data, right.bytes->data, (size_t)left.bytes->len) == 0;
  case TYA_TASK:
    return left.task == right.task;
  case TYA_CHANNEL:
    return left.channel == right.channel;
  case TYA_RESOURCE:
    return left.resource == right.resource;
  }
  return false;
}

TyaValue tya_deep_equal(TyaValue left, TyaValue right) {
  return tya_bool(tya_deep_equal_bool(left, right));
}

static bool tya_deep_equal_bool(TyaValue left, TyaValue right) {
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
    if (left.array == NULL || right.array == NULL) {
      return left.array == right.array;
    }
    if (left.array->len != right.array->len) {
      return false;
    }
    for (int i = 0; i < left.array->len; i++) {
      if (!tya_deep_equal_bool(left.array->items[i], right.array->items[i])) {
        return false;
      }
    }
    return true;
  case TYA_DICT:
  case TYA_OBJECT:
    if (left.dict == NULL || right.dict == NULL) {
      return left.dict == right.dict;
    }
    if ((int)tya_len(left).number != (int)tya_len(right).number) {
      return false;
    }
    for (int i = 0; i < left.dict->len; i++) {
      const char *key = left.dict->entries[i].key;
      if (key == NULL) {
        continue;
      }
      TyaValue right_value = tya_member(right, key);
      if (!tya_truthy(tya_has(right, tya_string(key)))) {
        return false;
      }
      if (!tya_deep_equal_bool(left.dict->entries[i].value, right_value)) {
        return false;
      }
    }
    return true;
  case TYA_FUNCTION:
    return left.function == right.function;
  case TYA_ERROR:
    if (left.error == NULL || right.error == NULL) {
      return left.error == right.error;
    }
    return strcmp(left.error, right.error) == 0;
  case TYA_BYTES:
    if (left.bytes == NULL || right.bytes == NULL) {
      return left.bytes == right.bytes;
    }
    if (left.bytes->len != right.bytes->len) {
      return false;
    }
    return memcmp(left.bytes->data, right.bytes->data, (size_t)left.bytes->len) == 0;
  case TYA_TASK:
    return left.task == right.task;
  case TYA_CHANNEL:
    return left.channel == right.channel;
  case TYA_RESOURCE:
    return left.resource == right.resource;
  }
  return false;
}

TyaValue tya_add(TyaValue left, TyaValue right) {
  if (left.kind == TYA_BYTES && right.kind == TYA_BYTES && left.bytes != NULL && right.bytes != NULL) {
    return tya_bytes_concat(left, right);
  }
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
  case TYA_OBJECT:
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
    if (value.function != NULL && value.function->is_class) {
      tya_builder_append(builder, value.function->class_name == NULL ? "" : value.function->class_name);
    } else {
      tya_builder_append(builder, "[function]");
    }
    break;
  case TYA_ERROR:
    tya_builder_append(builder, "error: ");
    tya_builder_append(builder, value.error == NULL ? "" : value.error);
    break;
  case TYA_STRING:
    tya_builder_append(builder, value.string == NULL ? "" : value.string);
    break;
  case TYA_TASK:
    tya_builder_append(builder, "[task]");
    break;
  case TYA_CHANNEL:
    tya_builder_append(builder, "[channel]");
    break;
  case TYA_RESOURCE:
    tya_builder_append(builder, "[resource]");
    break;
  case TYA_BYTES:
    tya_builder_append(builder, "<bytes:");
    if (value.bytes != NULL) {
      snprintf(scratch, sizeof(scratch), "%d", value.bytes->len);
      tya_builder_append(builder, scratch);
    } else {
      tya_builder_append(builder, "0");
    }
    tya_builder_append(builder, ">");
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
    return tya_number((double)((long)value.number));
  }
  if (value.kind == TYA_STRING && value.string != NULL) {
    return tya_number((double)strtol(value.string, NULL, 10));
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

static int tya_cstr_compare(const void *a, const void *b) {
  const char *const *aa = (const char *const *)a;
  const char *const *bb = (const char *const *)b;
  return strcmp(*aa, *bb);
}

TyaValue tya_dir_list(TyaValue path) {
  if (path.kind != TYA_STRING || path.string == NULL) {
    tya_raise(tya_string("dir.list: path must be a string"));
    return tya_nil();
  }
  DIR *dir = opendir(path.string);
  if (dir == NULL) {
    tya_raise(tya_string(strerror(errno)));
    return tya_nil();
  }
  int cap = 16;
  int count = 0;
  char **names = malloc(sizeof(char *) * (size_t)cap);
  struct dirent *entry;
  while ((entry = readdir(dir)) != NULL) {
    if (strcmp(entry->d_name, ".") == 0 || strcmp(entry->d_name, "..") == 0) {
      continue;
    }
    if (count >= cap) {
      cap *= 2;
      names = realloc(names, sizeof(char *) * (size_t)cap);
    }
    size_t n = strlen(entry->d_name);
    char *copy = malloc(n + 1);
    memcpy(copy, entry->d_name, n + 1);
    names[count++] = copy;
  }
  closedir(dir);
  qsort(names, (size_t)count, sizeof(char *), tya_cstr_compare);
  TyaValue out = tya_array(NULL, 0);
  for (int i = 0; i < count; i++) {
    tya_push(out, tya_string(names[i]));
  }
  free(names);
  return out;
}

TyaValue tya_dir_mkdir(TyaValue path) {
  if (path.kind != TYA_STRING || path.string == NULL) {
    tya_raise(tya_string("dir.mkdir: path must be a string"));
    return tya_nil();
  }
  if (mkdir(path.string, 0755) != 0) {
    tya_raise(tya_string(strerror(errno)));
    return tya_nil();
  }
  return tya_nil();
}

TyaValue tya_dir_rmdir(TyaValue path) {
  if (path.kind != TYA_STRING || path.string == NULL) {
    tya_raise(tya_string("dir.rmdir: path must be a string"));
    return tya_nil();
  }
  if (rmdir(path.string) != 0) {
    tya_raise(tya_string(strerror(errno)));
    return tya_nil();
  }
  return tya_nil();
}

TyaValue tya_file_remove(TyaValue path) {
  if (path.kind != TYA_STRING || path.string == NULL) {
    tya_raise(tya_string("file.remove: path must be a string"));
    return tya_nil();
  }
  struct stat st;
  if (stat(path.string, &st) != 0) {
    tya_raise(tya_string(strerror(errno)));
    return tya_nil();
  }
  if (S_ISDIR(st.st_mode)) {
    tya_raise(tya_string("file.remove: target is a directory"));
    return tya_nil();
  }
  if (unlink(path.string) != 0) {
    tya_raise(tya_string(strerror(errno)));
    return tya_nil();
  }
  return tya_nil();
}

TyaValue tya_file_rename(TyaValue old_path, TyaValue new_path) {
  if (old_path.kind != TYA_STRING || old_path.string == NULL ||
      new_path.kind != TYA_STRING || new_path.string == NULL) {
    tya_raise(tya_string("file.rename: paths must be strings"));
    return tya_nil();
  }
  if (rename(old_path.string, new_path.string) != 0) {
    tya_raise(tya_string(strerror(errno)));
    return tya_nil();
  }
  return tya_nil();
}

TyaValue tya_file_stat(TyaValue path) {
  if (path.kind != TYA_STRING || path.string == NULL) {
    tya_raise(tya_string("file.stat: path must be a string"));
    return tya_nil();
  }
  struct stat st;
  if (stat(path.string, &st) != 0) {
    tya_raise(tya_string(strerror(errno)));
    return tya_nil();
  }
  const char *kind = "other";
  if (S_ISREG(st.st_mode)) {
    kind = "file";
  } else if (S_ISDIR(st.st_mode)) {
    kind = "dir";
  }
  TyaValue out = tya_dict(NULL, 0);
  tya_set_member(out, "kind", tya_string(kind));
  tya_set_member(out, "size", tya_number((double)st.st_size));
  tya_set_member(out, "readable", tya_bool(access(path.string, R_OK) == 0));
  tya_set_member(out, "writable", tya_bool(access(path.string, W_OK) == 0));
  tya_set_member(out, "executable", tya_bool(access(path.string, X_OK) == 0));
  return out;
}

TyaValue tya_path_expand_user(TyaValue value) {
  if (value.kind != TYA_STRING || value.string == NULL) {
    tya_raise(tya_string("path.expand_user: value must be a string"));
    return tya_nil();
  }
  const char *src = value.string;
  if (src[0] != '~') {
    return value;
  }
  const char *home = getenv("HOME");
  if (home == NULL) {
    home = "";
  }
  if (src[1] == '\0') {
    return tya_string(home);
  }
  if (src[1] != '/') {
    return value;
  }
  size_t home_len = strlen(home);
  size_t rest_len = strlen(src + 1);
  char *out = malloc(home_len + rest_len + 1);
  memcpy(out, home, home_len);
  memcpy(out + home_len, src + 1, rest_len + 1);
  return tya_string(out);
}

TyaValue tya_cwd(void) {
  char buffer[4096];
  if (getcwd(buffer, sizeof(buffer)) == NULL) {
    tya_raise(tya_string(strerror(errno)));
    return tya_nil();
  }
  size_t n = strlen(buffer);
  char *out = malloc(n + 1);
  memcpy(out, buffer, n + 1);
  return tya_string(out);
}

TyaValue tya_chdir(TyaValue path) {
  if (path.kind != TYA_STRING || path.string == NULL) {
    tya_raise(tya_string("os.chdir: path must be a string"));
    return tya_nil();
  }
  if (chdir(path.string) != 0) {
    tya_raise(tya_string(strerror(errno)));
    return tya_nil();
  }
  return tya_nil();
}

TyaValue tya_read_line(void) {
  size_t cap = 128;
  size_t len = 0;
  char *buffer = malloc(cap);
  int ch = getchar();
  if (ch == EOF) {
    free(buffer);
    return tya_nil();
  }
  while (ch != EOF && ch != '\n') {
    if (len + 1 >= cap) {
      cap *= 2;
      buffer = realloc(buffer, cap);
    }
    buffer[len++] = (char)ch;
    ch = getchar();
  }
  buffer[len] = '\0';
  return tya_string(buffer);
}

TyaValue tya_map(TyaValue array, TyaValue fn) {
  TyaValue out = tya_array(0, 0);
  if (array.kind != TYA_ARRAY || array.array == NULL || fn.kind != TYA_FUNCTION) {
    return out;
  }
  for (int i = 0; i < array.array->len; i++) {
    tya_push(out, tya_call1(fn, array.array->items[i]));
  }
  return out;
}

TyaValue tya_filter(TyaValue array, TyaValue fn) {
  TyaValue out = tya_array(0, 0);
  if (array.kind != TYA_ARRAY || array.array == NULL || fn.kind != TYA_FUNCTION) {
    return out;
  }
  for (int i = 0; i < array.array->len; i++) {
    TyaValue item = array.array->items[i];
    if (tya_truthy(tya_call1(fn, item))) {
      tya_push(out, item);
    }
  }
  return out;
}

TyaValue tya_find(TyaValue array, TyaValue fn) {
  if (array.kind != TYA_ARRAY || array.array == NULL || fn.kind != TYA_FUNCTION) {
    return tya_nil();
  }
  for (int i = 0; i < array.array->len; i++) {
    TyaValue item = array.array->items[i];
    if (tya_truthy(tya_call1(fn, item))) {
      return item;
    }
  }
  return tya_nil();
}

TyaValue tya_any(TyaValue array, TyaValue fn) {
  if (array.kind != TYA_ARRAY || array.array == NULL || fn.kind != TYA_FUNCTION) {
    return tya_bool(false);
  }
  for (int i = 0; i < array.array->len; i++) {
    if (tya_truthy(tya_call1(fn, array.array->items[i]))) {
      return tya_bool(true);
    }
  }
  return tya_bool(false);
}

TyaValue tya_all(TyaValue array, TyaValue fn) {
  if (array.kind != TYA_ARRAY || array.array == NULL || fn.kind != TYA_FUNCTION) {
    return tya_bool(false);
  }
  for (int i = 0; i < array.array->len; i++) {
    if (!tya_truthy(tya_call1(fn, array.array->items[i]))) {
      return tya_bool(false);
    }
  }
  return tya_bool(true);
}

TyaValue tya_each(TyaValue array, TyaValue fn) {
  if (array.kind != TYA_ARRAY || array.array == NULL || fn.kind != TYA_FUNCTION) {
    return tya_nil();
  }
  for (int i = 0; i < array.array->len; i++) {
    (void)tya_call1(fn, array.array->items[i]);
  }
  return tya_nil();
}

TyaValue tya_reduce(TyaValue array, TyaValue initial, TyaValue fn) {
  TyaValue acc = initial;
  if (array.kind != TYA_ARRAY || array.array == NULL || fn.kind != TYA_FUNCTION) {
    return acc;
  }
  for (int i = 0; i < array.array->len; i++) {
    acc = tya_call2(fn, acc, array.array->items[i]);
  }
  return acc;
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

TyaValue tya_array_push(TyaValue array, TyaValue value) {
  tya_push(array, value);
  return array;
}

TyaValue tya_pop(TyaValue array) {
  if (array.kind != TYA_ARRAY || array.array == NULL || array.array->len == 0) {
    return tya_nil();
  }
  array.array->len--;
  return array.array->items[array.array->len];
}

TyaValue tya_first(TyaValue array) {
  if (array.kind != TYA_ARRAY || array.array == NULL || array.array->len == 0) {
    return tya_nil();
  }
  return array.array->items[0];
}

TyaValue tya_last(TyaValue array) {
  if (array.kind != TYA_ARRAY || array.array == NULL || array.array->len == 0) {
    return tya_nil();
  }
  return array.array->items[array.array->len - 1];
}

TyaValue tya_slice(TyaValue array, TyaValue start, TyaValue end) {
  if (array.kind != TYA_ARRAY || array.array == NULL || start.kind != TYA_NUMBER || end.kind != TYA_NUMBER) {
    return tya_array(NULL, 0);
  }
  int s = (int)start.number;
  int e = (int)end.number;
  if (s < 0 || e < 0) {
    tya_panic(tya_string("array.slice does not support negative indexes"));
  }
  if (s > e || e > array.array->len) {
    tya_panic(tya_string("array.slice index out of range"));
  }
  TyaValue out = tya_array(NULL, 0);
  for (int i = s; i < e; i++) {
    tya_push(out, array.array->items[i]);
  }
  return out;
}

TyaValue tya_reverse(TyaValue array) {
  TyaValue out = tya_array(NULL, 0);
  if (array.kind != TYA_ARRAY || array.array == NULL) {
    return out;
  }
  for (int i = array.array->len - 1; i >= 0; i--) {
    tya_push(out, array.array->items[i]);
  }
  return out;
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

void tya_push_raise_frame(TyaRaiseFrame *frame) {
  frame->value = tya_nil();
  frame->prev = tya_raise_frame;
  tya_raise_frame = frame;
}

void tya_pop_raise_frame(void) {
  if (tya_raise_frame != NULL) {
    tya_raise_frame = tya_raise_frame->prev;
  }
}

TyaValue tya_current_raise(void) {
  if (tya_raise_frame == NULL) {
    return tya_nil();
  }
  return tya_raise_frame->value;
}

void tya_raise(TyaValue value) {
  if (tya_raise_frame == NULL) {
    TyaValue text = tya_to_string(value);
    fprintf(stderr, "uncaught raised value: %s\n", text.string == NULL ? "" : text.string);
    exit(1);
  }
  tya_raise_frame->value = value;
  longjmp(tya_raise_frame->env, 1);
}

void tya_print(TyaValue value) {
  tya_write_value(stdout, value);
  putchar('\n');
}

void tya_assert(TyaValue value, const char *path, int line) {
  if (tya_truthy(value)) {
    return;
  }
  fprintf(stderr, "%s:%d:1: assertion failed\n", path == NULL || path[0] == '\0' ? "<unknown>" : path, line);
  exit(1);
}

void tya_assert_equal(TyaValue expected, TyaValue actual, const char *path, int line) {
  if (tya_deep_equal_bool(expected, actual)) {
    return;
  }
  fprintf(stderr, "%s:%d:1: assert_equal failed\n", path == NULL || path[0] == '\0' ? "<unknown>" : path, line);
  fprintf(stderr, "expected: ");
  tya_write_value(stderr, expected);
  fprintf(stderr, "\nactual: ");
  tya_write_value(stderr, actual);
  fprintf(stderr, "\n");
  exit(1);
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
  case TYA_OBJECT:
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
    if (value.function != NULL && value.function->is_class) {
      fprintf(out, "%s", value.function->class_name == NULL ? "" : value.function->class_name);
    } else {
      fprintf(out, "[function]");
    }
    break;
  case TYA_ERROR:
    fprintf(out, "error: %s", value.error == NULL ? "" : value.error);
    break;
  case TYA_BYTES:
    fprintf(out, "<bytes:%d>", value.bytes == NULL ? 0 : value.bytes->len);
    break;
  case TYA_TASK:
    fprintf(out, "[task]");
    break;
  case TYA_CHANNEL:
    fprintf(out, "[channel]");
    break;
  case TYA_RESOURCE:
    fprintf(out, "[resource]");
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

/* =========================================================================
 * v0.24: time
 * ========================================================================= */

TyaValue tya_time_now(void) {
  struct timeval tv;
  gettimeofday(&tv, NULL);
  return tya_number((double)tv.tv_sec + (double)tv.tv_usec / 1.0e6);
}

TyaValue tya_time_sleep(TyaValue seconds) {
  if (seconds.kind != TYA_NUMBER) {
    tya_raise(tya_string("time.sleep: argument must be a number"));
    return tya_nil();
  }
  if (seconds.number < 0) {
    tya_raise(tya_string("time.sleep: negative duration"));
    return tya_nil();
  }
  double whole = floor(seconds.number);
  double frac = seconds.number - whole;
  struct timespec req;
  req.tv_sec = (time_t)whole;
  req.tv_nsec = (long)(frac * 1.0e9);
  nanosleep(&req, NULL);
  return tya_nil();
}

TyaValue tya_time_format(TyaValue t, TyaValue layout, bool has_layout) {
  if (t.kind != TYA_NUMBER) {
    tya_raise(tya_string("time.format: argument must be a number"));
    return tya_nil();
  }
  const char *layout_name = "iso";
  if (has_layout) {
    if (layout.kind != TYA_STRING || layout.string == NULL) {
      tya_raise(tya_string("time.format: layout must be a string"));
      return tya_nil();
    }
    layout_name = layout.string;
  }
  if (strcmp(layout_name, "unix") == 0) {
    char buf[32];
    snprintf(buf, sizeof(buf), "%ld", (long)t.number);
    char *out = malloc(strlen(buf) + 1);
    strcpy(out, buf);
    return tya_string(out);
  }
  time_t tt = (time_t)t.number;
  struct tm gm;
  gmtime_r(&tt, &gm);
  char buf[64];
  if (strcmp(layout_name, "iso") == 0) {
    strftime(buf, sizeof(buf), "%Y-%m-%dT%H:%M:%SZ", &gm);
  } else if (strcmp(layout_name, "date") == 0) {
    strftime(buf, sizeof(buf), "%Y-%m-%d", &gm);
  } else if (strcmp(layout_name, "time") == 0) {
    strftime(buf, sizeof(buf), "%H:%M:%S", &gm);
  } else {
    tya_raise(tya_string("time.format: unknown layout"));
    return tya_nil();
  }
  char *out = malloc(strlen(buf) + 1);
  strcpy(out, buf);
  return tya_string(out);
}

TyaValue tya_time_parse(TyaValue text) {
  if (text.kind != TYA_STRING || text.string == NULL) {
    tya_raise(tya_string("time.parse: argument must be a string"));
    return tya_nil();
  }
  struct tm tm;
  memset(&tm, 0, sizeof(tm));
  const char *s = text.string;
  size_t n = strlen(s);
  const char *fmt;
  if (n >= 20 && s[10] == 'T' && s[n - 1] == 'Z') {
    fmt = "%Y-%m-%dT%H:%M:%SZ";
  } else if (n == 10) {
    fmt = "%Y-%m-%d";
  } else {
    tya_raise(tya_string("time.parse: unsupported format"));
    return tya_nil();
  }
  if (strptime(s, fmt, &tm) == NULL) {
    tya_raise(tya_string("time.parse: invalid timestamp"));
    return tya_nil();
  }
  time_t tt = timegm(&tm);
  return tya_number((double)tt);
}

TyaValue tya_time_since(TyaValue t) {
  if (t.kind != TYA_NUMBER) {
    tya_raise(tya_string("time.since: argument must be a number"));
    return tya_nil();
  }
  TyaValue now = tya_time_now();
  return tya_number(now.number - t.number);
}

/* =========================================================================
 * v0.24: random (xoshiro256** PRNG, seedable)
 * ========================================================================= */

static uint64_t tya_rng_state[4] = {
    0x9E3779B97F4A7C15ULL, 0xBF58476D1CE4E5B9ULL,
    0x94D049BB133111EBULL, 0x4F4A0E1D0E2A0B5DULL,
};
static int tya_rng_seeded = 0;

static uint64_t tya_rng_rotl(uint64_t x, int k) {
  return (x << k) | (x >> (64 - k));
}

static uint64_t tya_rng_next(void) {
  if (!tya_rng_seeded) {
    struct timeval tv;
    gettimeofday(&tv, NULL);
    uint64_t seed = (uint64_t)tv.tv_sec * 1000000ULL + (uint64_t)tv.tv_usec;
    seed ^= (uint64_t)getpid() << 32;
    /* splitmix64 to expand seed */
    for (int i = 0; i < 4; i++) {
      seed += 0x9E3779B97F4A7C15ULL;
      uint64_t z = seed;
      z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9ULL;
      z = (z ^ (z >> 27)) * 0x94D049BB133111EBULL;
      z = z ^ (z >> 31);
      tya_rng_state[i] = z;
    }
    tya_rng_seeded = 1;
  }
  const uint64_t result = tya_rng_rotl(tya_rng_state[1] * 5, 7) * 9;
  const uint64_t t = tya_rng_state[1] << 17;
  tya_rng_state[2] ^= tya_rng_state[0];
  tya_rng_state[3] ^= tya_rng_state[1];
  tya_rng_state[1] ^= tya_rng_state[2];
  tya_rng_state[0] ^= tya_rng_state[3];
  tya_rng_state[2] ^= t;
  tya_rng_state[3] = tya_rng_rotl(tya_rng_state[3], 45);
  return result;
}

TyaValue tya_random_seed(TyaValue value) {
  uint64_t seed = 0;
  if (value.kind == TYA_NUMBER) {
    seed = (uint64_t)(int64_t)value.number;
  } else if (value.kind == TYA_STRING && value.string != NULL) {
    /* FNV-1a 64-bit */
    seed = 14695981039346656037ULL;
    for (const unsigned char *p = (const unsigned char *)value.string; *p; p++) {
      seed ^= *p;
      seed *= 1099511628211ULL;
    }
  } else {
    tya_raise(tya_string("random.seed: argument must be int or string"));
    return tya_nil();
  }
  for (int i = 0; i < 4; i++) {
    seed += 0x9E3779B97F4A7C15ULL;
    uint64_t z = seed;
    z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9ULL;
    z = (z ^ (z >> 27)) * 0x94D049BB133111EBULL;
    z = z ^ (z >> 31);
    tya_rng_state[i] = z;
  }
  tya_rng_seeded = 1;
  return tya_nil();
}

TyaValue tya_random_int(TyaValue min, TyaValue max) {
  if (min.kind != TYA_NUMBER || max.kind != TYA_NUMBER) {
    tya_raise(tya_string("random.int: arguments must be numbers"));
    return tya_nil();
  }
  long mn = (long)min.number;
  long mx = (long)max.number;
  if (mx < mn) {
    tya_raise(tya_string("random.int: max < min"));
    return tya_nil();
  }
  uint64_t range = (uint64_t)(mx - mn) + 1ULL;
  uint64_t r = tya_rng_next();
  return tya_number((double)((long)(r % range) + mn));
}

TyaValue tya_random_float(void) {
  uint64_t r = tya_rng_next() >> 11; /* 53 bits */
  double v = (double)r / (double)(1ULL << 53);
  return tya_number(v);
}

/* =========================================================================
 * v0.24: math expansion
 * ========================================================================= */

static TyaValue tya_math_unary(double (*fn)(double), TyaValue x, const char *name) {
  if (x.kind != TYA_NUMBER) {
    tya_raise(tya_string("math: argument must be a number"));
    return tya_nil();
  }
  (void)name;
  return tya_number(fn(x.number));
}

TyaValue tya_math_sqrt(TyaValue x) {
  if (x.kind != TYA_NUMBER) {
    tya_raise(tya_string("math.sqrt: argument must be a number"));
    return tya_nil();
  }
  if (x.number < 0) {
    tya_raise(tya_string("math.sqrt: negative argument"));
    return tya_nil();
  }
  return tya_number(sqrt(x.number));
}

TyaValue tya_math_pow(TyaValue x, TyaValue y) {
  if (x.kind != TYA_NUMBER || y.kind != TYA_NUMBER) {
    tya_raise(tya_string("math.pow: arguments must be numbers"));
    return tya_nil();
  }
  return tya_number(pow(x.number, y.number));
}

TyaValue tya_math_floor(TyaValue x) { return tya_math_unary(floor, x, "floor"); }
TyaValue tya_math_ceil(TyaValue x) { return tya_math_unary(ceil, x, "ceil"); }
TyaValue tya_math_round(TyaValue x) {
  if (x.kind != TYA_NUMBER) {
    tya_raise(tya_string("math.round: argument must be a number"));
    return tya_nil();
  }
  double v = x.number;
  if (v >= 0) {
    return tya_number(floor(v + 0.5));
  }
  return tya_number(-floor(-v + 0.5));
}
TyaValue tya_math_trunc(TyaValue x) { return tya_math_unary(trunc, x, "trunc"); }

static TyaValue tya_math_log_kind(double (*fn)(double), TyaValue x, const char *name) {
  if (x.kind != TYA_NUMBER) {
    tya_raise(tya_string("math: argument must be a number"));
    return tya_nil();
  }
  if (x.number <= 0) {
    tya_raise(tya_string("math: non-positive argument to log"));
    return tya_nil();
  }
  (void)name;
  return tya_number(fn(x.number));
}

TyaValue tya_math_log(TyaValue x) { return tya_math_log_kind(log, x, "log"); }
TyaValue tya_math_log2(TyaValue x) { return tya_math_log_kind(log2, x, "log2"); }
TyaValue tya_math_log10(TyaValue x) { return tya_math_log_kind(log10, x, "log10"); }
TyaValue tya_math_exp(TyaValue x) { return tya_math_unary(exp, x, "exp"); }
TyaValue tya_math_sin(TyaValue x) { return tya_math_unary(sin, x, "sin"); }
TyaValue tya_math_cos(TyaValue x) { return tya_math_unary(cos, x, "cos"); }
TyaValue tya_math_tan(TyaValue x) { return tya_math_unary(tan, x, "tan"); }
TyaValue tya_math_asin(TyaValue x) { return tya_math_unary(asin, x, "asin"); }
TyaValue tya_math_acos(TyaValue x) { return tya_math_unary(acos, x, "acos"); }
TyaValue tya_math_atan(TyaValue x) { return tya_math_unary(atan, x, "atan"); }

TyaValue tya_math_atan2(TyaValue y, TyaValue x) {
  if (x.kind != TYA_NUMBER || y.kind != TYA_NUMBER) {
    tya_raise(tya_string("math.atan2: arguments must be numbers"));
    return tya_nil();
  }
  return tya_number(atan2(y.number, x.number));
}

/* =========================================================================
 * v0.24: process
 * ========================================================================= */

static char *tya_dup_cstr(const char *s) {
  size_t n = strlen(s) + 1;
  char *out = malloc(n);
  memcpy(out, s, n);
  return out;
}

static char *tya_read_all(int fd) {
  size_t cap = 256;
  size_t len = 0;
  char *buf = malloc(cap);
  for (;;) {
    if (len + 1 >= cap) {
      cap *= 2;
      buf = realloc(buf, cap);
    }
    ssize_t r = read(fd, buf + len, cap - len - 1);
    if (r < 0) {
      if (errno == EINTR) continue;
      free(buf);
      return NULL;
    }
    if (r == 0) break;
    len += (size_t)r;
  }
  buf[len] = '\0';
  return buf;
}

TyaValue tya_process_run(TyaValue command, TyaValue options) {
  if (command.kind != TYA_ARRAY || command.array == NULL || command.array->len == 0) {
    tya_raise(tya_string("process.run: command must be a non-empty array of strings"));
    return tya_nil();
  }
  int argc = command.array->len;
  char **argv = malloc(sizeof(char *) * (size_t)(argc + 1));
  for (int i = 0; i < argc; i++) {
    TyaValue item = command.array->items[i];
    if (item.kind != TYA_STRING || item.string == NULL) {
      for (int j = 0; j < i; j++) free(argv[j]);
      free(argv);
      tya_raise(tya_string("process.run: command items must be strings"));
      return tya_nil();
    }
    argv[i] = tya_dup_cstr(item.string);
  }
  argv[argc] = NULL;

  const char *cwd_path = NULL;
  const char *input_text = NULL;
  size_t input_len = 0;
  char **child_env = NULL;
  bool replace_env = false;
  if (options.kind == TYA_DICT && options.dict != NULL) {
    TyaValue cwd = tya_member(options, "cwd");
    if (cwd.kind == TYA_STRING && cwd.string != NULL) {
      cwd_path = cwd.string;
    }
    TyaValue inp = tya_member(options, "input");
    if (inp.kind == TYA_STRING && inp.string != NULL) {
      input_text = inp.string;
      input_len = strlen(input_text);
    }
    TyaValue env_v = tya_member(options, "env");
    if (env_v.kind == TYA_DICT && env_v.dict != NULL) {
      replace_env = true;
      int env_count = 0;
      for (int i = 0; i < env_v.dict->len; i++) {
        if (env_v.dict->entries[i].key != NULL) env_count++;
      }
      child_env = malloc(sizeof(char *) * (size_t)(env_count + 1));
      int idx = 0;
      for (int i = 0; i < env_v.dict->len; i++) {
        if (env_v.dict->entries[i].key == NULL) continue;
        TyaValue val = env_v.dict->entries[i].value;
        if (val.kind != TYA_STRING || val.string == NULL) {
          for (int j = 0; j < idx; j++) free(child_env[j]);
          free(child_env);
          for (int j = 0; j < argc; j++) free(argv[j]);
          free(argv);
          tya_raise(tya_string("process.run: env values must be strings"));
          return tya_nil();
        }
        size_t kl = strlen(env_v.dict->entries[i].key);
        size_t vl = strlen(val.string);
        char *entry = malloc(kl + 1 + vl + 1);
        memcpy(entry, env_v.dict->entries[i].key, kl);
        entry[kl] = '=';
        memcpy(entry + kl + 1, val.string, vl);
        entry[kl + 1 + vl] = '\0';
        child_env[idx++] = entry;
      }
      child_env[idx] = NULL;
    }
  }

  int in_pipe[2] = {-1, -1};
  int out_pipe[2] = {-1, -1};
  int err_pipe[2] = {-1, -1};
  if (pipe(in_pipe) < 0 || pipe(out_pipe) < 0 || pipe(err_pipe) < 0) {
    tya_raise(tya_string("process.run: pipe failed"));
    return tya_nil();
  }

  pid_t pid = fork();
  if (pid < 0) {
    tya_raise(tya_string("process.run: fork failed"));
    return tya_nil();
  }
  if (pid == 0) {
    /* child */
    dup2(in_pipe[0], 0);
    dup2(out_pipe[1], 1);
    dup2(err_pipe[1], 2);
    close(in_pipe[0]); close(in_pipe[1]);
    close(out_pipe[0]); close(out_pipe[1]);
    close(err_pipe[0]); close(err_pipe[1]);
    if (cwd_path && chdir(cwd_path) < 0) {
      _exit(127);
    }
    if (replace_env) {
      execve(argv[0], argv, child_env);
    } else {
      execvp(argv[0], argv);
    }
    _exit(127);
  }
  /* parent */
  close(in_pipe[0]);
  close(out_pipe[1]);
  close(err_pipe[1]);
  if (input_text && input_len > 0) {
    size_t written = 0;
    while (written < input_len) {
      ssize_t w = write(in_pipe[1], input_text + written, input_len - written);
      if (w < 0) {
        if (errno == EINTR) continue;
        break;
      }
      written += (size_t)w;
    }
  }
  close(in_pipe[1]);
  char *out_buf = tya_read_all(out_pipe[0]);
  char *err_buf = tya_read_all(err_pipe[0]);
  close(out_pipe[0]);
  close(err_pipe[0]);
  int status = 0;
  while (waitpid(pid, &status, 0) < 0) {
    if (errno != EINTR) break;
  }

  for (int i = 0; i < argc; i++) free(argv[i]);
  free(argv);
  if (child_env) {
    for (int i = 0; child_env[i]; i++) free(child_env[i]);
    free(child_env);
  }

  int exit_code = 0;
  if (WIFEXITED(status)) {
    exit_code = WEXITSTATUS(status);
  } else if (WIFSIGNALED(status)) {
    exit_code = 128 + WTERMSIG(status);
  }

  TyaValue result = tya_dict(NULL, 0);
  tya_set_member(result, "exit_code", tya_number((double)exit_code));
  tya_set_member(result, "stdout", tya_string(out_buf ? out_buf : ""));
  tya_set_member(result, "stderr", tya_string(err_buf ? err_buf : ""));
  if (out_buf == NULL) free(out_buf);
  if (err_buf == NULL) free(err_buf);
  return result;
}

/* =========================================================================
 * v0.24: digest (MD5, SHA1, SHA256, SHA384, SHA512)
 * Public-domain inline implementations.
 * ========================================================================= */

/* ---- MD5 ---- */
typedef struct {
  uint32_t state[4];
  uint64_t count;
  uint8_t buffer[64];
} tya_md5_ctx;

static void tya_md5_init(tya_md5_ctx *c) {
  c->state[0] = 0x67452301; c->state[1] = 0xEFCDAB89;
  c->state[2] = 0x98BADCFE; c->state[3] = 0x10325476;
  c->count = 0;
}

#define TYA_MD5_F(x, y, z) (((x) & (y)) | (~(x) & (z)))
#define TYA_MD5_G(x, y, z) (((x) & (z)) | ((y) & ~(z)))
#define TYA_MD5_H(x, y, z) ((x) ^ (y) ^ (z))
#define TYA_MD5_I(x, y, z) ((y) ^ ((x) | ~(z)))
#define TYA_MD5_ROL(x, n) (((x) << (n)) | ((x) >> (32 - (n))))
#define TYA_MD5_STEP(f, a, b, c, d, x, t, s) \
  (a) += f((b), (c), (d)) + (x) + (t); \
  (a) = TYA_MD5_ROL((a), (s)); \
  (a) += (b);

static void tya_md5_transform(tya_md5_ctx *ctx, const uint8_t block[64]) {
  uint32_t a = ctx->state[0], b = ctx->state[1], c = ctx->state[2], d = ctx->state[3];
  uint32_t x[16];
  for (int i = 0; i < 16; i++) {
    x[i] = (uint32_t)block[i * 4] | ((uint32_t)block[i * 4 + 1] << 8) |
           ((uint32_t)block[i * 4 + 2] << 16) | ((uint32_t)block[i * 4 + 3] << 24);
  }
  TYA_MD5_STEP(TYA_MD5_F, a, b, c, d, x[ 0], 0xD76AA478,  7)
  TYA_MD5_STEP(TYA_MD5_F, d, a, b, c, x[ 1], 0xE8C7B756, 12)
  TYA_MD5_STEP(TYA_MD5_F, c, d, a, b, x[ 2], 0x242070DB, 17)
  TYA_MD5_STEP(TYA_MD5_F, b, c, d, a, x[ 3], 0xC1BDCEEE, 22)
  TYA_MD5_STEP(TYA_MD5_F, a, b, c, d, x[ 4], 0xF57C0FAF,  7)
  TYA_MD5_STEP(TYA_MD5_F, d, a, b, c, x[ 5], 0x4787C62A, 12)
  TYA_MD5_STEP(TYA_MD5_F, c, d, a, b, x[ 6], 0xA8304613, 17)
  TYA_MD5_STEP(TYA_MD5_F, b, c, d, a, x[ 7], 0xFD469501, 22)
  TYA_MD5_STEP(TYA_MD5_F, a, b, c, d, x[ 8], 0x698098D8,  7)
  TYA_MD5_STEP(TYA_MD5_F, d, a, b, c, x[ 9], 0x8B44F7AF, 12)
  TYA_MD5_STEP(TYA_MD5_F, c, d, a, b, x[10], 0xFFFF5BB1, 17)
  TYA_MD5_STEP(TYA_MD5_F, b, c, d, a, x[11], 0x895CD7BE, 22)
  TYA_MD5_STEP(TYA_MD5_F, a, b, c, d, x[12], 0x6B901122,  7)
  TYA_MD5_STEP(TYA_MD5_F, d, a, b, c, x[13], 0xFD987193, 12)
  TYA_MD5_STEP(TYA_MD5_F, c, d, a, b, x[14], 0xA679438E, 17)
  TYA_MD5_STEP(TYA_MD5_F, b, c, d, a, x[15], 0x49B40821, 22)
  TYA_MD5_STEP(TYA_MD5_G, a, b, c, d, x[ 1], 0xF61E2562,  5)
  TYA_MD5_STEP(TYA_MD5_G, d, a, b, c, x[ 6], 0xC040B340,  9)
  TYA_MD5_STEP(TYA_MD5_G, c, d, a, b, x[11], 0x265E5A51, 14)
  TYA_MD5_STEP(TYA_MD5_G, b, c, d, a, x[ 0], 0xE9B6C7AA, 20)
  TYA_MD5_STEP(TYA_MD5_G, a, b, c, d, x[ 5], 0xD62F105D,  5)
  TYA_MD5_STEP(TYA_MD5_G, d, a, b, c, x[10], 0x02441453,  9)
  TYA_MD5_STEP(TYA_MD5_G, c, d, a, b, x[15], 0xD8A1E681, 14)
  TYA_MD5_STEP(TYA_MD5_G, b, c, d, a, x[ 4], 0xE7D3FBC8, 20)
  TYA_MD5_STEP(TYA_MD5_G, a, b, c, d, x[ 9], 0x21E1CDE6,  5)
  TYA_MD5_STEP(TYA_MD5_G, d, a, b, c, x[14], 0xC33707D6,  9)
  TYA_MD5_STEP(TYA_MD5_G, c, d, a, b, x[ 3], 0xF4D50D87, 14)
  TYA_MD5_STEP(TYA_MD5_G, b, c, d, a, x[ 8], 0x455A14ED, 20)
  TYA_MD5_STEP(TYA_MD5_G, a, b, c, d, x[13], 0xA9E3E905,  5)
  TYA_MD5_STEP(TYA_MD5_G, d, a, b, c, x[ 2], 0xFCEFA3F8,  9)
  TYA_MD5_STEP(TYA_MD5_G, c, d, a, b, x[ 7], 0x676F02D9, 14)
  TYA_MD5_STEP(TYA_MD5_G, b, c, d, a, x[12], 0x8D2A4C8A, 20)
  TYA_MD5_STEP(TYA_MD5_H, a, b, c, d, x[ 5], 0xFFFA3942,  4)
  TYA_MD5_STEP(TYA_MD5_H, d, a, b, c, x[ 8], 0x8771F681, 11)
  TYA_MD5_STEP(TYA_MD5_H, c, d, a, b, x[11], 0x6D9D6122, 16)
  TYA_MD5_STEP(TYA_MD5_H, b, c, d, a, x[14], 0xFDE5380C, 23)
  TYA_MD5_STEP(TYA_MD5_H, a, b, c, d, x[ 1], 0xA4BEEA44,  4)
  TYA_MD5_STEP(TYA_MD5_H, d, a, b, c, x[ 4], 0x4BDECFA9, 11)
  TYA_MD5_STEP(TYA_MD5_H, c, d, a, b, x[ 7], 0xF6BB4B60, 16)
  TYA_MD5_STEP(TYA_MD5_H, b, c, d, a, x[10], 0xBEBFBC70, 23)
  TYA_MD5_STEP(TYA_MD5_H, a, b, c, d, x[13], 0x289B7EC6,  4)
  TYA_MD5_STEP(TYA_MD5_H, d, a, b, c, x[ 0], 0xEAA127FA, 11)
  TYA_MD5_STEP(TYA_MD5_H, c, d, a, b, x[ 3], 0xD4EF3085, 16)
  TYA_MD5_STEP(TYA_MD5_H, b, c, d, a, x[ 6], 0x04881D05, 23)
  TYA_MD5_STEP(TYA_MD5_H, a, b, c, d, x[ 9], 0xD9D4D039,  4)
  TYA_MD5_STEP(TYA_MD5_H, d, a, b, c, x[12], 0xE6DB99E5, 11)
  TYA_MD5_STEP(TYA_MD5_H, c, d, a, b, x[15], 0x1FA27CF8, 16)
  TYA_MD5_STEP(TYA_MD5_H, b, c, d, a, x[ 2], 0xC4AC5665, 23)
  TYA_MD5_STEP(TYA_MD5_I, a, b, c, d, x[ 0], 0xF4292244,  6)
  TYA_MD5_STEP(TYA_MD5_I, d, a, b, c, x[ 7], 0x432AFF97, 10)
  TYA_MD5_STEP(TYA_MD5_I, c, d, a, b, x[14], 0xAB9423A7, 15)
  TYA_MD5_STEP(TYA_MD5_I, b, c, d, a, x[ 5], 0xFC93A039, 21)
  TYA_MD5_STEP(TYA_MD5_I, a, b, c, d, x[12], 0x655B59C3,  6)
  TYA_MD5_STEP(TYA_MD5_I, d, a, b, c, x[ 3], 0x8F0CCC92, 10)
  TYA_MD5_STEP(TYA_MD5_I, c, d, a, b, x[10], 0xFFEFF47D, 15)
  TYA_MD5_STEP(TYA_MD5_I, b, c, d, a, x[ 1], 0x85845DD1, 21)
  TYA_MD5_STEP(TYA_MD5_I, a, b, c, d, x[ 8], 0x6FA87E4F,  6)
  TYA_MD5_STEP(TYA_MD5_I, d, a, b, c, x[15], 0xFE2CE6E0, 10)
  TYA_MD5_STEP(TYA_MD5_I, c, d, a, b, x[ 6], 0xA3014314, 15)
  TYA_MD5_STEP(TYA_MD5_I, b, c, d, a, x[13], 0x4E0811A1, 21)
  TYA_MD5_STEP(TYA_MD5_I, a, b, c, d, x[ 4], 0xF7537E82,  6)
  TYA_MD5_STEP(TYA_MD5_I, d, a, b, c, x[11], 0xBD3AF235, 10)
  TYA_MD5_STEP(TYA_MD5_I, c, d, a, b, x[ 2], 0x2AD7D2BB, 15)
  TYA_MD5_STEP(TYA_MD5_I, b, c, d, a, x[ 9], 0xEB86D391, 21)
  ctx->state[0] += a; ctx->state[1] += b;
  ctx->state[2] += c; ctx->state[3] += d;
}

static void tya_md5_update(tya_md5_ctx *c, const uint8_t *data, size_t len) {
  size_t buf_used = (size_t)((c->count >> 3) & 0x3F);
  c->count += (uint64_t)len << 3;
  size_t need = 64 - buf_used;
  if (len >= need) {
    memcpy(c->buffer + buf_used, data, need);
    tya_md5_transform(c, c->buffer);
    data += need; len -= need;
    while (len >= 64) {
      tya_md5_transform(c, data);
      data += 64; len -= 64;
    }
    buf_used = 0;
  }
  memcpy(c->buffer + buf_used, data, len);
}

static void tya_md5_final(tya_md5_ctx *c, uint8_t out[16]) {
  size_t buf_used = (size_t)((c->count >> 3) & 0x3F);
  c->buffer[buf_used++] = 0x80;
  if (buf_used > 56) {
    memset(c->buffer + buf_used, 0, 64 - buf_used);
    tya_md5_transform(c, c->buffer);
    buf_used = 0;
  }
  memset(c->buffer + buf_used, 0, 56 - buf_used);
  for (int i = 0; i < 8; i++) {
    c->buffer[56 + i] = (uint8_t)((c->count >> (i * 8)) & 0xFF);
  }
  tya_md5_transform(c, c->buffer);
  for (int i = 0; i < 4; i++) {
    out[i * 4] = (uint8_t)(c->state[i] & 0xFF);
    out[i * 4 + 1] = (uint8_t)((c->state[i] >> 8) & 0xFF);
    out[i * 4 + 2] = (uint8_t)((c->state[i] >> 16) & 0xFF);
    out[i * 4 + 3] = (uint8_t)((c->state[i] >> 24) & 0xFF);
  }
}

/* ---- SHA1 ---- */
typedef struct {
  uint32_t state[5];
  uint64_t count;
  uint8_t buffer[64];
} tya_sha1_ctx;

static void tya_sha1_init(tya_sha1_ctx *c) {
  c->state[0] = 0x67452301; c->state[1] = 0xEFCDAB89;
  c->state[2] = 0x98BADCFE; c->state[3] = 0x10325476;
  c->state[4] = 0xC3D2E1F0;
  c->count = 0;
}

#define TYA_SHA1_ROL(x, n) (((x) << (n)) | ((x) >> (32 - (n))))

static void tya_sha1_transform(tya_sha1_ctx *ctx, const uint8_t block[64]) {
  uint32_t w[80];
  for (int i = 0; i < 16; i++) {
    w[i] = ((uint32_t)block[i * 4] << 24) | ((uint32_t)block[i * 4 + 1] << 16) |
           ((uint32_t)block[i * 4 + 2] << 8) | (uint32_t)block[i * 4 + 3];
  }
  for (int i = 16; i < 80; i++) {
    w[i] = TYA_SHA1_ROL(w[i - 3] ^ w[i - 8] ^ w[i - 14] ^ w[i - 16], 1);
  }
  uint32_t a = ctx->state[0], b = ctx->state[1], c = ctx->state[2], d = ctx->state[3], e = ctx->state[4];
  for (int i = 0; i < 80; i++) {
    uint32_t f, k;
    if (i < 20) { f = (b & c) | (~b & d); k = 0x5A827999; }
    else if (i < 40) { f = b ^ c ^ d; k = 0x6ED9EBA1; }
    else if (i < 60) { f = (b & c) | (b & d) | (c & d); k = 0x8F1BBCDC; }
    else { f = b ^ c ^ d; k = 0xCA62C1D6; }
    uint32_t t = TYA_SHA1_ROL(a, 5) + f + e + k + w[i];
    e = d; d = c; c = TYA_SHA1_ROL(b, 30); b = a; a = t;
  }
  ctx->state[0] += a; ctx->state[1] += b;
  ctx->state[2] += c; ctx->state[3] += d;
  ctx->state[4] += e;
}

static void tya_sha1_update(tya_sha1_ctx *c, const uint8_t *data, size_t len) {
  size_t buf_used = (size_t)((c->count >> 3) & 0x3F);
  c->count += (uint64_t)len << 3;
  size_t need = 64 - buf_used;
  if (len >= need) {
    memcpy(c->buffer + buf_used, data, need);
    tya_sha1_transform(c, c->buffer);
    data += need; len -= need;
    while (len >= 64) {
      tya_sha1_transform(c, data);
      data += 64; len -= 64;
    }
    buf_used = 0;
  }
  memcpy(c->buffer + buf_used, data, len);
}

static void tya_sha1_final(tya_sha1_ctx *c, uint8_t out[20]) {
  size_t buf_used = (size_t)((c->count >> 3) & 0x3F);
  c->buffer[buf_used++] = 0x80;
  if (buf_used > 56) {
    memset(c->buffer + buf_used, 0, 64 - buf_used);
    tya_sha1_transform(c, c->buffer);
    buf_used = 0;
  }
  memset(c->buffer + buf_used, 0, 56 - buf_used);
  for (int i = 0; i < 8; i++) {
    c->buffer[56 + i] = (uint8_t)((c->count >> (56 - i * 8)) & 0xFF);
  }
  tya_sha1_transform(c, c->buffer);
  for (int i = 0; i < 5; i++) {
    out[i * 4] = (uint8_t)((c->state[i] >> 24) & 0xFF);
    out[i * 4 + 1] = (uint8_t)((c->state[i] >> 16) & 0xFF);
    out[i * 4 + 2] = (uint8_t)((c->state[i] >> 8) & 0xFF);
    out[i * 4 + 3] = (uint8_t)(c->state[i] & 0xFF);
  }
}

/* ---- SHA-256 ---- */
typedef struct {
  uint32_t state[8];
  uint64_t count;
  uint8_t buffer[64];
} tya_sha256_ctx;

static const uint32_t tya_sha256_k[64] = {
  0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
  0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
  0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
  0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
  0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
  0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
  0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
  0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
};

static void tya_sha256_init(tya_sha256_ctx *c) {
  c->state[0] = 0x6a09e667; c->state[1] = 0xbb67ae85;
  c->state[2] = 0x3c6ef372; c->state[3] = 0xa54ff53a;
  c->state[4] = 0x510e527f; c->state[5] = 0x9b05688c;
  c->state[6] = 0x1f83d9ab; c->state[7] = 0x5be0cd19;
  c->count = 0;
}

#define TYA_SHA256_ROR(x, n) (((x) >> (n)) | ((x) << (32 - (n))))

static void tya_sha256_transform(tya_sha256_ctx *ctx, const uint8_t block[64]) {
  uint32_t w[64];
  for (int i = 0; i < 16; i++) {
    w[i] = ((uint32_t)block[i * 4] << 24) | ((uint32_t)block[i * 4 + 1] << 16) |
           ((uint32_t)block[i * 4 + 2] << 8) | (uint32_t)block[i * 4 + 3];
  }
  for (int i = 16; i < 64; i++) {
    uint32_t s0 = TYA_SHA256_ROR(w[i - 15], 7) ^ TYA_SHA256_ROR(w[i - 15], 18) ^ (w[i - 15] >> 3);
    uint32_t s1 = TYA_SHA256_ROR(w[i - 2], 17) ^ TYA_SHA256_ROR(w[i - 2], 19) ^ (w[i - 2] >> 10);
    w[i] = w[i - 16] + s0 + w[i - 7] + s1;
  }
  uint32_t a = ctx->state[0], b = ctx->state[1], c = ctx->state[2], d = ctx->state[3];
  uint32_t e = ctx->state[4], f = ctx->state[5], g = ctx->state[6], h = ctx->state[7];
  for (int i = 0; i < 64; i++) {
    uint32_t S1 = TYA_SHA256_ROR(e, 6) ^ TYA_SHA256_ROR(e, 11) ^ TYA_SHA256_ROR(e, 25);
    uint32_t ch = (e & f) ^ (~e & g);
    uint32_t t1 = h + S1 + ch + tya_sha256_k[i] + w[i];
    uint32_t S0 = TYA_SHA256_ROR(a, 2) ^ TYA_SHA256_ROR(a, 13) ^ TYA_SHA256_ROR(a, 22);
    uint32_t mj = (a & b) ^ (a & c) ^ (b & c);
    uint32_t t2 = S0 + mj;
    h = g; g = f; f = e; e = d + t1;
    d = c; c = b; b = a; a = t1 + t2;
  }
  ctx->state[0] += a; ctx->state[1] += b;
  ctx->state[2] += c; ctx->state[3] += d;
  ctx->state[4] += e; ctx->state[5] += f;
  ctx->state[6] += g; ctx->state[7] += h;
}

static void tya_sha256_update(tya_sha256_ctx *c, const uint8_t *data, size_t len) {
  size_t buf_used = (size_t)((c->count >> 3) & 0x3F);
  c->count += (uint64_t)len << 3;
  size_t need = 64 - buf_used;
  if (len >= need) {
    memcpy(c->buffer + buf_used, data, need);
    tya_sha256_transform(c, c->buffer);
    data += need; len -= need;
    while (len >= 64) {
      tya_sha256_transform(c, data);
      data += 64; len -= 64;
    }
    buf_used = 0;
  }
  memcpy(c->buffer + buf_used, data, len);
}

static void tya_sha256_final(tya_sha256_ctx *c, uint8_t out[32]) {
  size_t buf_used = (size_t)((c->count >> 3) & 0x3F);
  c->buffer[buf_used++] = 0x80;
  if (buf_used > 56) {
    memset(c->buffer + buf_used, 0, 64 - buf_used);
    tya_sha256_transform(c, c->buffer);
    buf_used = 0;
  }
  memset(c->buffer + buf_used, 0, 56 - buf_used);
  for (int i = 0; i < 8; i++) {
    c->buffer[56 + i] = (uint8_t)((c->count >> (56 - i * 8)) & 0xFF);
  }
  tya_sha256_transform(c, c->buffer);
  for (int i = 0; i < 8; i++) {
    out[i * 4] = (uint8_t)((c->state[i] >> 24) & 0xFF);
    out[i * 4 + 1] = (uint8_t)((c->state[i] >> 16) & 0xFF);
    out[i * 4 + 2] = (uint8_t)((c->state[i] >> 8) & 0xFF);
    out[i * 4 + 3] = (uint8_t)(c->state[i] & 0xFF);
  }
}

/* ---- SHA-512 (and SHA-384) ---- */
typedef struct {
  uint64_t state[8];
  uint64_t count_lo;
  uint64_t count_hi;
  uint8_t buffer[128];
} tya_sha512_ctx;

static const uint64_t tya_sha512_k[80] = {
  0x428a2f98d728ae22ULL, 0x7137449123ef65cdULL, 0xb5c0fbcfec4d3b2fULL, 0xe9b5dba58189dbbcULL,
  0x3956c25bf348b538ULL, 0x59f111f1b605d019ULL, 0x923f82a4af194f9bULL, 0xab1c5ed5da6d8118ULL,
  0xd807aa98a3030242ULL, 0x12835b0145706fbeULL, 0x243185be4ee4b28cULL, 0x550c7dc3d5ffb4e2ULL,
  0x72be5d74f27b896fULL, 0x80deb1fe3b1696b1ULL, 0x9bdc06a725c71235ULL, 0xc19bf174cf692694ULL,
  0xe49b69c19ef14ad2ULL, 0xefbe4786384f25e3ULL, 0x0fc19dc68b8cd5b5ULL, 0x240ca1cc77ac9c65ULL,
  0x2de92c6f592b0275ULL, 0x4a7484aa6ea6e483ULL, 0x5cb0a9dcbd41fbd4ULL, 0x76f988da831153b5ULL,
  0x983e5152ee66dfabULL, 0xa831c66d2db43210ULL, 0xb00327c898fb213fULL, 0xbf597fc7beef0ee4ULL,
  0xc6e00bf33da88fc2ULL, 0xd5a79147930aa725ULL, 0x06ca6351e003826fULL, 0x142929670a0e6e70ULL,
  0x27b70a8546d22ffcULL, 0x2e1b21385c26c926ULL, 0x4d2c6dfc5ac42aedULL, 0x53380d139d95b3dfULL,
  0x650a73548baf63deULL, 0x766a0abb3c77b2a8ULL, 0x81c2c92e47edaee6ULL, 0x92722c851482353bULL,
  0xa2bfe8a14cf10364ULL, 0xa81a664bbc423001ULL, 0xc24b8b70d0f89791ULL, 0xc76c51a30654be30ULL,
  0xd192e819d6ef5218ULL, 0xd69906245565a910ULL, 0xf40e35855771202aULL, 0x106aa07032bbd1b8ULL,
  0x19a4c116b8d2d0c8ULL, 0x1e376c085141ab53ULL, 0x2748774cdf8eeb99ULL, 0x34b0bcb5e19b48a8ULL,
  0x391c0cb3c5c95a63ULL, 0x4ed8aa4ae3418acbULL, 0x5b9cca4f7763e373ULL, 0x682e6ff3d6b2b8a3ULL,
  0x748f82ee5defb2fcULL, 0x78a5636f43172f60ULL, 0x84c87814a1f0ab72ULL, 0x8cc702081a6439ecULL,
  0x90befffa23631e28ULL, 0xa4506cebde82bde9ULL, 0xbef9a3f7b2c67915ULL, 0xc67178f2e372532bULL,
  0xca273eceea26619cULL, 0xd186b8c721c0c207ULL, 0xeada7dd6cde0eb1eULL, 0xf57d4f7fee6ed178ULL,
  0x06f067aa72176fbaULL, 0x0a637dc5a2c898a6ULL, 0x113f9804bef90daeULL, 0x1b710b35131c471bULL,
  0x28db77f523047d84ULL, 0x32caab7b40c72493ULL, 0x3c9ebe0a15c9bebcULL, 0x431d67c49c100d4cULL,
  0x4cc5d4becb3e42b6ULL, 0x597f299cfc657e2aULL, 0x5fcb6fab3ad6faecULL, 0x6c44198c4a475817ULL,
};

#define TYA_SHA512_ROR(x, n) (((x) >> (n)) | ((x) << (64 - (n))))

static void tya_sha512_transform(tya_sha512_ctx *ctx, const uint8_t block[128]) {
  uint64_t w[80];
  for (int i = 0; i < 16; i++) {
    w[i] = 0;
    for (int j = 0; j < 8; j++) {
      w[i] = (w[i] << 8) | block[i * 8 + j];
    }
  }
  for (int i = 16; i < 80; i++) {
    uint64_t s0 = TYA_SHA512_ROR(w[i - 15], 1) ^ TYA_SHA512_ROR(w[i - 15], 8) ^ (w[i - 15] >> 7);
    uint64_t s1 = TYA_SHA512_ROR(w[i - 2], 19) ^ TYA_SHA512_ROR(w[i - 2], 61) ^ (w[i - 2] >> 6);
    w[i] = w[i - 16] + s0 + w[i - 7] + s1;
  }
  uint64_t a = ctx->state[0], b = ctx->state[1], c = ctx->state[2], d = ctx->state[3];
  uint64_t e = ctx->state[4], f = ctx->state[5], g = ctx->state[6], h = ctx->state[7];
  for (int i = 0; i < 80; i++) {
    uint64_t S1 = TYA_SHA512_ROR(e, 14) ^ TYA_SHA512_ROR(e, 18) ^ TYA_SHA512_ROR(e, 41);
    uint64_t ch = (e & f) ^ (~e & g);
    uint64_t t1 = h + S1 + ch + tya_sha512_k[i] + w[i];
    uint64_t S0 = TYA_SHA512_ROR(a, 28) ^ TYA_SHA512_ROR(a, 34) ^ TYA_SHA512_ROR(a, 39);
    uint64_t mj = (a & b) ^ (a & c) ^ (b & c);
    uint64_t t2 = S0 + mj;
    h = g; g = f; f = e; e = d + t1;
    d = c; c = b; b = a; a = t1 + t2;
  }
  ctx->state[0] += a; ctx->state[1] += b;
  ctx->state[2] += c; ctx->state[3] += d;
  ctx->state[4] += e; ctx->state[5] += f;
  ctx->state[6] += g; ctx->state[7] += h;
}

static void tya_sha512_init(tya_sha512_ctx *c) {
  c->state[0] = 0x6a09e667f3bcc908ULL; c->state[1] = 0xbb67ae8584caa73bULL;
  c->state[2] = 0x3c6ef372fe94f82bULL; c->state[3] = 0xa54ff53a5f1d36f1ULL;
  c->state[4] = 0x510e527fade682d1ULL; c->state[5] = 0x9b05688c2b3e6c1fULL;
  c->state[6] = 0x1f83d9abfb41bd6bULL; c->state[7] = 0x5be0cd19137e2179ULL;
  c->count_lo = 0; c->count_hi = 0;
}

static void tya_sha384_init(tya_sha512_ctx *c) {
  c->state[0] = 0xcbbb9d5dc1059ed8ULL; c->state[1] = 0x629a292a367cd507ULL;
  c->state[2] = 0x9159015a3070dd17ULL; c->state[3] = 0x152fecd8f70e5939ULL;
  c->state[4] = 0x67332667ffc00b31ULL; c->state[5] = 0x8eb44a8768581511ULL;
  c->state[6] = 0xdb0c2e0d64f98fa7ULL; c->state[7] = 0x47b5481dbefa4fa4ULL;
  c->count_lo = 0; c->count_hi = 0;
}

static void tya_sha512_update(tya_sha512_ctx *c, const uint8_t *data, size_t len) {
  size_t buf_used = (size_t)((c->count_lo >> 3) & 0x7F);
  uint64_t add = (uint64_t)len << 3;
  uint64_t old_lo = c->count_lo;
  c->count_lo += add;
  if (c->count_lo < old_lo) c->count_hi++;
  c->count_hi += (uint64_t)len >> 61;
  size_t need = 128 - buf_used;
  if (len >= need) {
    memcpy(c->buffer + buf_used, data, need);
    tya_sha512_transform(c, c->buffer);
    data += need; len -= need;
    while (len >= 128) {
      tya_sha512_transform(c, data);
      data += 128; len -= 128;
    }
    buf_used = 0;
  }
  memcpy(c->buffer + buf_used, data, len);
}

static void tya_sha512_final_n(tya_sha512_ctx *c, uint8_t *out, int out_words) {
  size_t buf_used = (size_t)((c->count_lo >> 3) & 0x7F);
  c->buffer[buf_used++] = 0x80;
  if (buf_used > 112) {
    memset(c->buffer + buf_used, 0, 128 - buf_used);
    tya_sha512_transform(c, c->buffer);
    buf_used = 0;
  }
  memset(c->buffer + buf_used, 0, 112 - buf_used);
  for (int i = 0; i < 8; i++) {
    c->buffer[112 + i] = (uint8_t)((c->count_hi >> (56 - i * 8)) & 0xFF);
  }
  for (int i = 0; i < 8; i++) {
    c->buffer[120 + i] = (uint8_t)((c->count_lo >> (56 - i * 8)) & 0xFF);
  }
  tya_sha512_transform(c, c->buffer);
  for (int i = 0; i < out_words; i++) {
    for (int j = 0; j < 8; j++) {
      out[i * 8 + j] = (uint8_t)((c->state[i] >> (56 - j * 8)) & 0xFF);
    }
  }
}

static const char tya_hex_digits[] = "0123456789abcdef";

static TyaValue tya_hex_string(const uint8_t *data, size_t n) {
  char *out = malloc(n * 2 + 1);
  for (size_t i = 0; i < n; i++) {
    out[i * 2] = tya_hex_digits[(data[i] >> 4) & 0xF];
    out[i * 2 + 1] = tya_hex_digits[data[i] & 0xF];
  }
  out[n * 2] = '\0';
  return tya_string(out);
}

TyaValue tya_digest_md5(TyaValue text) {
  const uint8_t *data;
  size_t dlen;
  if (text.kind == TYA_STRING && text.string != NULL) {
    data = (const uint8_t *)text.string;
    dlen = strlen(text.string);
  } else if (text.kind == TYA_BYTES && text.bytes != NULL) {
    data = text.bytes->data;
    dlen = (size_t)text.bytes->len;
  } else {
    tya_raise(tya_string("digest.md5: argument must be a string or bytes"));
    return tya_nil();
  }
  tya_md5_ctx c;
  tya_md5_init(&c);
  tya_md5_update(&c, data, dlen);
  uint8_t digest[16];
  tya_md5_final(&c, digest);
  return tya_hex_string(digest, 16);
}

static int tya_digest_input(TyaValue v, const uint8_t **data, size_t *dlen, const char *err_msg) {
  if (v.kind == TYA_STRING && v.string != NULL) {
    *data = (const uint8_t *)v.string;
    *dlen = strlen(v.string);
    return 0;
  }
  if (v.kind == TYA_BYTES && v.bytes != NULL) {
    *data = v.bytes->data;
    *dlen = (size_t)v.bytes->len;
    return 0;
  }
  tya_raise(tya_string(err_msg));
  return -1;
}

TyaValue tya_digest_sha1(TyaValue text) {
  const uint8_t *data;
  size_t dlen;
  if (tya_digest_input(text, &data, &dlen, "digest.sha1: argument must be a string or bytes") < 0) {
    return tya_nil();
  }
  tya_sha1_ctx c;
  tya_sha1_init(&c);
  tya_sha1_update(&c, data, dlen);
  uint8_t digest[20];
  tya_sha1_final(&c, digest);
  return tya_hex_string(digest, 20);
}

TyaValue tya_digest_sha256(TyaValue text) {
  const uint8_t *data;
  size_t dlen;
  if (tya_digest_input(text, &data, &dlen, "digest.sha256: argument must be a string or bytes") < 0) {
    return tya_nil();
  }
  tya_sha256_ctx c;
  tya_sha256_init(&c);
  tya_sha256_update(&c, data, dlen);
  uint8_t digest[32];
  tya_sha256_final(&c, digest);
  return tya_hex_string(digest, 32);
}

TyaValue tya_digest_sha384(TyaValue text) {
  const uint8_t *data;
  size_t dlen;
  if (tya_digest_input(text, &data, &dlen, "digest.sha384: argument must be a string or bytes") < 0) {
    return tya_nil();
  }
  tya_sha512_ctx c;
  tya_sha384_init(&c);
  tya_sha512_update(&c, data, dlen);
  uint8_t digest[48];
  tya_sha512_final_n(&c, digest, 6);
  return tya_hex_string(digest, 48);
}

TyaValue tya_digest_sha512(TyaValue text) {
  const uint8_t *data;
  size_t dlen;
  if (tya_digest_input(text, &data, &dlen, "digest.sha512: argument must be a string or bytes") < 0) {
    return tya_nil();
  }
  tya_sha512_ctx c;
  tya_sha512_init(&c);
  tya_sha512_update(&c, data, dlen);
  uint8_t digest[64];
  tya_sha512_final_n(&c, digest, 8);
  return tya_hex_string(digest, 64);
}

/* =========================================================================
 * v0.24: secure_random
 * ========================================================================= */

static int tya_secure_random_fill(uint8_t *buf, size_t n) {
#if defined(__APPLE__) || defined(__FreeBSD__) || defined(__OpenBSD__)
  while (n > 0) {
    size_t chunk = n > 256 ? 256 : n;
    if (getentropy(buf, chunk) < 0) return -1;
    buf += chunk; n -= chunk;
  }
  return 0;
#else
  int fd = open("/dev/urandom", O_RDONLY);
  if (fd < 0) return -1;
  while (n > 0) {
    ssize_t r = read(fd, buf, n);
    if (r < 0) {
      if (errno == EINTR) continue;
      close(fd);
      return -1;
    }
    if (r == 0) { close(fd); return -1; }
    buf += r; n -= (size_t)r;
  }
  close(fd);
  return 0;
#endif
}

TyaValue tya_secure_random_bytes(TyaValue n) {
  if (n.kind != TYA_NUMBER) {
    tya_raise(tya_string("secure_random: count must be a number"));
    return tya_nil();
  }
  long count = (long)n.number;
  if (count < 0 || count > 4096) {
    tya_raise(tya_string("secure_random: count out of range"));
    return tya_nil();
  }
  TyaBytes *bb = tya_gc_alloc(sizeof(TyaBytes), TYA_GC_BYTES);
  bb->len = (int)count;
  bb->data = malloc((size_t)(count > 0 ? count : 1));
  if (tya_secure_random_fill(bb->data, (size_t)count) < 0) {
    free(bb->data);
    /* bb is GC-tracked; leak now, the next collection will reclaim it. */
    tya_raise(tya_string("secure_random: entropy source unavailable"));
    return tya_nil();
  }
  return (TyaValue){.kind = TYA_BYTES, .bytes = bb};
}

TyaValue tya_secure_random_int(TyaValue min, TyaValue max) {
  if (min.kind != TYA_NUMBER || max.kind != TYA_NUMBER) {
    tya_raise(tya_string("secure_random.int: arguments must be numbers"));
    return tya_nil();
  }
  long mn = (long)min.number;
  long mx = (long)max.number;
  if (mx < mn) {
    tya_raise(tya_string("secure_random.int: max < min"));
    return tya_nil();
  }
  uint64_t range = (uint64_t)(mx - mn) + 1ULL;
  uint64_t threshold = (uint64_t)(-(int64_t)range) % range;
  for (;;) {
    uint64_t r;
    if (tya_secure_random_fill((uint8_t *)&r, sizeof(r)) < 0) {
      tya_raise(tya_string("secure_random.int: entropy source unavailable"));
      return tya_nil();
    }
    if (r >= threshold) {
      return tya_number((double)(long)((r % range) + (uint64_t)mn));
    }
  }
}

/* =========================================================================
 * v0.25: bytes type and binary I/O
 * ========================================================================= */

TyaValue tya_bytes_lit(const char *data, int len) {
  TyaBytes *b = tya_gc_alloc(sizeof(TyaBytes), TYA_GC_BYTES);
  b->len = len;
  b->data = malloc((size_t)(len > 0 ? len : 1));
  if (len > 0) memcpy(b->data, data, (size_t)len);
  return (TyaValue){.kind = TYA_BYTES, .bytes = b};
}

TyaValue tya_bytes_from_array(TyaValue arr) {
  if (arr.kind != TYA_ARRAY || arr.array == NULL) {
    tya_raise(tya_string("bytes: argument must be an array of ints"));
    return tya_nil();
  }
  int n = arr.array->len;
  TyaBytes *b = tya_gc_alloc(sizeof(TyaBytes), TYA_GC_BYTES);
  b->len = n;
  b->data = malloc((size_t)(n > 0 ? n : 1));
  for (int i = 0; i < n; i++) {
    TyaValue item = arr.array->items[i];
    if (item.kind != TYA_NUMBER) {
      free(b->data);
      /* b is GC-tracked; leak now, the next collection will reclaim it. */
      tya_raise(tya_string("bytes: items must be ints"));
      return tya_nil();
    }
    int v = (int)item.number;
    if (v < 0 || v > 255) {
      free(b->data);
      /* b is GC-tracked; leak now, the next collection will reclaim it. */
      tya_raise(tya_string("bytes: item out of 0..255"));
      return tya_nil();
    }
    b->data[i] = (uint8_t)v;
  }
  return (TyaValue){.kind = TYA_BYTES, .bytes = b};
}

TyaValue tya_bytes_of(TyaValue text) {
  if (text.kind != TYA_STRING || text.string == NULL) {
    tya_raise(tya_string("bytes_of: argument must be a string"));
    return tya_nil();
  }
  int n = (int)strlen(text.string);
  return tya_bytes_lit(text.string, n);
}

TyaValue tya_bytes_text(TyaValue b) {
  if (b.kind != TYA_BYTES || b.bytes == NULL) {
    tya_raise(tya_string("bytes_text: argument must be bytes"));
    return tya_nil();
  }
  for (int i = 0; i < b.bytes->len; i++) {
    if (b.bytes->data[i] == 0) {
      tya_raise(tya_string("bytes_text: NUL byte not allowed in string"));
      return tya_nil();
    }
  }
  char *out = malloc((size_t)b.bytes->len + 1);
  memcpy(out, b.bytes->data, (size_t)b.bytes->len);
  out[b.bytes->len] = '\0';
  return tya_string(out);
}

TyaValue tya_bytes_array(TyaValue b) {
  if (b.kind != TYA_BYTES || b.bytes == NULL) {
    tya_raise(tya_string("bytes_array: argument must be bytes"));
    return tya_nil();
  }
  TyaValue out = tya_array(NULL, 0);
  for (int i = 0; i < b.bytes->len; i++) {
    tya_push(out, tya_number((double)b.bytes->data[i]));
  }
  return out;
}

TyaValue tya_bytes_concat(TyaValue a, TyaValue b) {
  if (a.kind != TYA_BYTES || b.kind != TYA_BYTES || a.bytes == NULL || b.bytes == NULL) {
    tya_raise(tya_string("bytes_concat: arguments must be bytes"));
    return tya_nil();
  }
  int total = a.bytes->len + b.bytes->len;
  TyaBytes *out = tya_gc_alloc(sizeof(TyaBytes), TYA_GC_BYTES);
  out->len = total;
  out->data = malloc((size_t)(total > 0 ? total : 1));
  if (a.bytes->len > 0) memcpy(out->data, a.bytes->data, (size_t)a.bytes->len);
  if (b.bytes->len > 0) memcpy(out->data + a.bytes->len, b.bytes->data, (size_t)b.bytes->len);
  return (TyaValue){.kind = TYA_BYTES, .bytes = out};
}

TyaValue tya_bytes_slice(TyaValue b, TyaValue start_v, TyaValue end_v) {
  if (b.kind != TYA_BYTES || b.bytes == NULL) {
    tya_raise(tya_string("bytes_slice: first argument must be bytes"));
    return tya_nil();
  }
  if (start_v.kind != TYA_NUMBER || end_v.kind != TYA_NUMBER) {
    tya_raise(tya_string("bytes_slice: indices must be ints"));
    return tya_nil();
  }
  int s = (int)start_v.number;
  int e = (int)end_v.number;
  if (s < 0 || e < s || e > b.bytes->len) {
    tya_raise(tya_string("bytes_slice: index out of range"));
    return tya_nil();
  }
  return tya_bytes_lit((const char *)(b.bytes->data + s), e - s);
}

TyaValue tya_file_read_bytes(TyaValue path) {
  if (path.kind != TYA_STRING || path.string == NULL) {
    tya_raise(tya_string("file.read_bytes: path must be a string"));
    return tya_nil();
  }
  FILE *fp = fopen(path.string, "rb");
  if (fp == NULL) {
    tya_raise(tya_string(strerror(errno)));
    return tya_nil();
  }
  fseek(fp, 0, SEEK_END);
  long size = ftell(fp);
  fseek(fp, 0, SEEK_SET);
  if (size < 0) size = 0;
  TyaBytes *bb = tya_gc_alloc(sizeof(TyaBytes), TYA_GC_BYTES);
  bb->len = (int)size;
  bb->data = malloc((size_t)(size > 0 ? size : 1));
  size_t got = fread(bb->data, 1, (size_t)size, fp);
  fclose(fp);
  bb->len = (int)got;
  return (TyaValue){.kind = TYA_BYTES, .bytes = bb};
}

TyaValue tya_file_write_bytes(TyaValue path, TyaValue b) {
  if (path.kind != TYA_STRING || path.string == NULL) {
    tya_raise(tya_string("file.write_bytes: path must be a string"));
    return tya_nil();
  }
  if (b.kind != TYA_BYTES || b.bytes == NULL) {
    tya_raise(tya_string("file.write_bytes: data must be bytes"));
    return tya_nil();
  }
  FILE *fp = fopen(path.string, "wb");
  if (fp == NULL) {
    tya_raise(tya_string(strerror(errno)));
    return tya_nil();
  }
  if (b.bytes->len > 0) {
    fwrite(b.bytes->data, 1, (size_t)b.bytes->len, fp);
  }
  fclose(fp);
  return tya_nil();
}

TyaValue tya_stderr_write(TyaValue text) {
  if (text.kind != TYA_STRING || text.string == NULL) {
    tya_raise(tya_string("stderr.write: text must be a string"));
    return tya_nil();
  }
  fputs(text.string, stderr);
  fflush(stderr);
  return tya_nil();
}

TyaValue tya_file_append(TyaValue path, TyaValue text) {
  if (path.kind != TYA_STRING || path.string == NULL) {
    tya_raise(tya_string("file.append: path must be a string"));
    return tya_nil();
  }
  if (text.kind != TYA_STRING || text.string == NULL) {
    tya_raise(tya_string("file.append: text must be a string"));
    return tya_nil();
  }
  FILE *fp = fopen(path.string, "ab");
  if (fp == NULL) {
    tya_raise(tya_string(strerror(errno)));
    return tya_nil();
  }
  fputs(text.string, fp);
  fclose(fp);
  return tya_nil();
}

/* =========================================================================
 * v0.41 GC API
 * ========================================================================= */

void tya_gc_collect(void) {
  pthread_mutex_lock(&tya_gc_mu);
  /* Mark from registered globals. */
  for (size_t i = 0; i < tya_gc_root_count; i++) {
    tya_gc_mark_value(*tya_gc_roots[i]);
  }
  /* Mark in-flight raise values. */
  for (TyaRaiseFrame *frame = tya_raise_frame; frame != NULL; frame = frame->prev) {
    tya_gc_mark_value(frame->value);
  }
  /* Mark every not-yet-joined task so its mutex / pthread state
   * cannot be reclaimed while the worker thread is still alive. */
  for (TyaTask *t = tya_live_tasks; t != NULL; t = t->next_live) {
    tya_gc_mark_header((TyaGcHeader *)t);
  }
  /* Sweep. */
  tya_gc_sweep();
  tya_gc_collect_count++;
  tya_gc_live_after_last = tya_gc_alloc_count - tya_gc_freed_count;
  /* Recompute threshold: collect again when allocations grow by another
   * factor over what survived. Minimum 1024 to avoid thrashing on tiny
   * programs. */
  size_t target = tya_gc_live_after_last * 2;
  tya_gc_threshold = (target > 1024) ? target : 1024;
  pthread_mutex_unlock(&tya_gc_mu);
}

void tya_gc_maybe_collect(void) {
  /* Called by generated code at safe points (between top-level
   * statements). Triggers a collection when allocations since the last
   * collection exceed the threshold. */
  pthread_mutex_lock(&tya_gc_mu);
  size_t live = tya_gc_alloc_count - tya_gc_freed_count;
  size_t threshold = tya_gc_threshold;
  pthread_mutex_unlock(&tya_gc_mu);
  if (live >= threshold) {
    tya_gc_collect();
  }
}

TyaValue tya_gc_stats(void) {
  pthread_mutex_lock(&tya_gc_mu);
  size_t alloc_count = tya_gc_alloc_count;
  size_t alloc_bytes = tya_gc_alloc_bytes;
  size_t freed_count = tya_gc_freed_count;
  size_t freed_bytes = tya_gc_freed_bytes;
  size_t collect_count = tya_gc_collect_count;
  size_t threshold = tya_gc_threshold;
  pthread_mutex_unlock(&tya_gc_mu);
  size_t live_count = alloc_count - freed_count;
  size_t live_bytes = alloc_bytes - freed_bytes;
  TyaDictEntry entries[8] = {
    {"alloc_count",   tya_number((double)alloc_count)},
    {"alloc_bytes",   tya_number((double)alloc_bytes)},
    {"freed_count",   tya_number((double)freed_count)},
    {"freed_bytes",   tya_number((double)freed_bytes)},
    {"live_count",    tya_number((double)live_count)},
    {"live_bytes",    tya_number((double)live_bytes)},
    {"collect_count", tya_number((double)collect_count)},
    {"threshold",     tya_number((double)threshold)},
  };
  return tya_dict(entries, 8);
}


/* =========================================================================
 * v0.42 STEP 3: spawn / await runtime
 * ========================================================================= */

static TyaValue tya_task_invoke(TyaValue callee, int argc, TyaValue *argv) {
  switch (argc) {
    case 0:
      return tya_call1(callee, tya_nil());
    case 1:
      return tya_call1(callee, argv[0]);
    case 2:
      return tya_call2(callee, argv[0], argv[1]);
    case 3:
      return tya_call3(callee, argv[0], argv[1], argv[2]);
    case 4:
      return tya_call4(callee, argv[0], argv[1], argv[2], argv[3]);
  }
  return tya_nil();
}

/* Pointer to the currently-running task; nil on the main thread.
 * tya_current_task() reads it; tya_task_thread_main sets it on entry. */
static _Thread_local TyaTask *tya_current_task_ptr = NULL;

TyaValue tya_current_task(void) {
  TyaTask *t = tya_current_task_ptr;
  if (t == NULL) return tya_nil();
  return (TyaValue){.kind = TYA_TASK, .task = t};
}

static void *tya_task_thread_main(void *arg) {
  TyaTask *t = (TyaTask *)arg;
  tya_current_task_ptr = t;
  TyaRaiseFrame frame;
  frame.prev = NULL;
  if (setjmp(frame.env) == 0) {
    tya_push_raise_frame(&frame);
    TyaValue result = tya_task_invoke(t->callee, t->argc, t->argv);
    tya_pop_raise_frame();
    pthread_mutex_lock(&t->mu);
    t->result = result;
    t->done = true;
    pthread_cond_broadcast(&t->cv);
    pthread_mutex_unlock(&t->mu);
  } else {
    /* The body raised; capture the value and propagate it from the
     * awaiter. The raise frame is the one this task pushed, so it has
     * already been longjmp'd back to; no pop is needed. */
    pthread_mutex_lock(&t->mu);
    t->raise_value = frame.value;
    t->raised = true;
    t->done = true;
    pthread_cond_broadcast(&t->cv);
    pthread_mutex_unlock(&t->mu);
  }
  return NULL;
}

/* Per-thread chain of structured-concurrency scopes. tya_task_new
 * registers each new task in the innermost scope (if any) so that
 * tya_scope_exit can wait for it before returning. */
static _Thread_local TyaScope *tya_current_scope = NULL;

static void tya_scope_register_task(TyaTask *t) {
  TyaScope *s = tya_current_scope;
  if (s == NULL) return;
  if (s->len == s->cap) {
    int new_cap = s->cap == 0 ? 8 : s->cap * 2;
    s->tasks = realloc(s->tasks, sizeof(TyaTask *) * (size_t)new_cap);
    s->cap = new_cap;
  }
  s->tasks[s->len++] = t;
}

void tya_scope_enter(TyaScope *scope) {
  scope->tasks = NULL;
  scope->len = 0;
  scope->cap = 0;
  scope->prev = tya_current_scope;
  tya_current_scope = scope;
}

void tya_scope_exit(TyaScope *scope) {
  TyaValue first_raise = tya_nil();
  bool had_raise = false;
  for (int i = 0; i < scope->len; i++) {
    TyaTask *t = scope->tasks[i];
    /* Once any sibling has raised, request cancel on every remaining
     * task so a cooperative worker can return early instead of running
     * to completion before the scope can re-raise. */
    if (had_raise) {
      atomic_store(&t->cancelled, true);
    }
    pthread_mutex_lock(&t->mu);
    bool already_joined = t->joined;
    pthread_mutex_unlock(&t->mu);
    if (!already_joined) {
      pthread_join(t->thread, NULL);
      pthread_mutex_lock(&t->mu);
      t->joined = true;
      pthread_mutex_unlock(&t->mu);
      tya_live_tasks_remove(t);
    }
    pthread_mutex_lock(&t->mu);
    if (t->raised && !had_raise) {
      first_raise = t->raise_value;
      had_raise = true;
    }
    pthread_mutex_unlock(&t->mu);
  }
  free(scope->tasks);
  scope->tasks = NULL;
  scope->len = 0;
  scope->cap = 0;
  tya_current_scope = scope->prev;
  if (had_raise) {
    tya_raise(first_raise);
  }
}

/* tya_scope_raise is called when control unwinds out of a scope body
 * via raise. It cancels every sibling, joins them, and then re-raises
 * the original raise value (taking precedence over any task raise). */
void tya_scope_raise(TyaScope *scope, TyaValue value) {
  for (int i = 0; i < scope->len; i++) {
    atomic_store(&scope->tasks[i]->cancelled, true);
  }
  for (int i = 0; i < scope->len; i++) {
    TyaTask *t = scope->tasks[i];
    pthread_mutex_lock(&t->mu);
    bool already_joined = t->joined;
    pthread_mutex_unlock(&t->mu);
    if (!already_joined) {
      pthread_join(t->thread, NULL);
      pthread_mutex_lock(&t->mu);
      t->joined = true;
      pthread_mutex_unlock(&t->mu);
      tya_live_tasks_remove(t);
    }
  }
  free(scope->tasks);
  scope->tasks = NULL;
  scope->len = 0;
  scope->cap = 0;
  tya_current_scope = scope->prev;
  tya_raise(value);
}

/* Add a freshly created task to the live-tasks list; called once
 * before pthread_create so the task is reachable as a root from
 * the moment it exists. */
static void tya_live_tasks_add(TyaTask *t) {
  pthread_mutex_lock(&tya_gc_mu);
  if (!t->in_live_list) {
    t->prev_live = NULL;
    t->next_live = tya_live_tasks;
    if (tya_live_tasks) tya_live_tasks->prev_live = t;
    tya_live_tasks = t;
    t->in_live_list = true;
  }
  pthread_mutex_unlock(&tya_gc_mu);
}

/* Remove a task from the live-tasks list; called once the task has
 * been joined and its result is either consumed or otherwise reachable
 * through normal value plumbing. After this call the GC may reclaim
 * the task struct as soon as nothing else references it. */
static void tya_live_tasks_remove(TyaTask *t) {
  pthread_mutex_lock(&tya_gc_mu);
  if (t->in_live_list) {
    if (t->prev_live) t->prev_live->next_live = t->next_live;
    else tya_live_tasks = t->next_live;
    if (t->next_live) t->next_live->prev_live = t->prev_live;
    t->prev_live = NULL;
    t->next_live = NULL;
    t->in_live_list = false;
  }
  pthread_mutex_unlock(&tya_gc_mu);
}

TyaValue tya_task_new(TyaValue callee, int argc, TyaValue a, TyaValue b, TyaValue c, TyaValue d) {
  if (callee.kind != TYA_FUNCTION || callee.function == NULL) {
    tya_raise(tya_string("spawn: argument must be a callable"));
    return tya_nil();
  }
  if (argc < 0 || argc > 4) {
    tya_raise(tya_string("spawn: at most 4 arguments are supported"));
    return tya_nil();
  }
  TyaTask *t = tya_gc_alloc(sizeof(TyaTask), TYA_GC_TASK);
  pthread_mutex_init(&t->mu, NULL);
  pthread_cond_init(&t->cv, NULL);
  t->done = false;
  t->joined = false;
  t->raised = false;
  atomic_store(&t->cancelled, false);
  t->callee = callee;
  t->argc = argc;
  t->argv[0] = a;
  t->argv[1] = b;
  t->argv[2] = c;
  t->argv[3] = d;
  t->result = tya_nil();
  t->raise_value = tya_nil();
  t->prev_live = NULL;
  t->next_live = NULL;
  t->in_live_list = false;
  tya_live_tasks_add(t);
  if (pthread_create(&t->thread, NULL, tya_task_thread_main, t) != 0) {
    tya_live_tasks_remove(t);
    tya_raise(tya_string("spawn: pthread_create failed"));
    return tya_nil();
  }
  tya_scope_register_task(t);
  return (TyaValue){.kind = TYA_TASK, .task = t};
}

TyaValue tya_task_cancel(TyaValue v) {
  if (v.kind != TYA_TASK || v.task == NULL) {
    tya_raise(tya_string("task.cancel: argument must be a task"));
    return tya_nil();
  }
  atomic_store(&v.task->cancelled, true);
  return tya_nil();
}

TyaValue tya_task_is_cancelled(TyaValue v) {
  if (v.kind != TYA_TASK || v.task == NULL) {
    tya_raise(tya_string("task.is_cancelled: argument must be a task"));
    return tya_nil();
  }
  return tya_bool(atomic_load(&v.task->cancelled));
}

TyaValue tya_task_await(TyaValue v) {
  if (v.kind != TYA_TASK || v.task == NULL) {
    tya_raise(tya_string("await: argument must be a task"));
    return tya_nil();
  }
  TyaTask *t = v.task;
  pthread_mutex_lock(&t->mu);
  bool already_joined = t->joined;
  pthread_mutex_unlock(&t->mu);
  if (!already_joined) {
    pthread_join(t->thread, NULL);
    pthread_mutex_lock(&t->mu);
    t->joined = true;
    pthread_mutex_unlock(&t->mu);
    tya_live_tasks_remove(t);
  }
  pthread_mutex_lock(&t->mu);
  bool raised = t->raised;
  TyaValue value = raised ? t->raise_value : t->result;
  pthread_mutex_unlock(&t->mu);
  if (raised) {
    tya_raise(value);
    return tya_nil();
  }
  return value;
}


/* =========================================================================
 * v0.42 STEP 5: channel runtime
 * ========================================================================= */

TyaValue tya_channel_new(TyaValue capacity) {
  if (capacity.kind != TYA_NUMBER) {
    tya_raise(tya_string("channel.new: capacity must be a number"));
    return tya_nil();
  }
  int cap = (int)capacity.number;
  if (cap < 0) {
    tya_raise(tya_string("channel.new: capacity must be >= 0"));
    return tya_nil();
  }
  /* v0.42 STEP 5: capacity 0 is treated as 1. True rendezvous arrives
   * in a later minor. */
  if (cap == 0) cap = 1;
  TyaChannel *c = tya_gc_alloc(sizeof(TyaChannel), TYA_GC_CHANNEL);
  c->buffer = malloc(sizeof(TyaValue) * (size_t)cap);
  for (int i = 0; i < cap; i++) c->buffer[i] = tya_nil();
  c->capacity = cap;
  c->len = 0;
  c->head = 0;
  pthread_mutex_init(&c->mu, NULL);
  pthread_cond_init(&c->not_full, NULL);
  pthread_cond_init(&c->not_empty, NULL);
  c->closed = false;
  return (TyaValue){.kind = TYA_CHANNEL, .channel = c};
}

TyaValue tya_channel_send(TyaValue ch, TyaValue value) {
  if (ch.kind != TYA_CHANNEL || ch.channel == NULL) {
    tya_raise(tya_string("channel.send: first argument must be a channel"));
    return tya_nil();
  }
  TyaChannel *c = ch.channel;
  pthread_mutex_lock(&c->mu);
  while (c->len == c->capacity && !c->closed) {
    pthread_cond_wait(&c->not_full, &c->mu);
  }
  if (c->closed) {
    pthread_mutex_unlock(&c->mu);
    tya_raise(tya_string("channel.send: channel is closed"));
    return tya_nil();
  }
  int tail = (c->head + c->len) % c->capacity;
  c->buffer[tail] = value;
  c->len++;
  pthread_cond_signal(&c->not_empty);
  pthread_mutex_unlock(&c->mu);
  return tya_nil();
}

TyaValue tya_channel_receive(TyaValue ch) {
  if (ch.kind != TYA_CHANNEL || ch.channel == NULL) {
    tya_raise(tya_string("channel.receive: argument must be a channel"));
    return tya_nil();
  }
  TyaChannel *c = ch.channel;
  pthread_mutex_lock(&c->mu);
  while (c->len == 0 && !c->closed) {
    pthread_cond_wait(&c->not_empty, &c->mu);
  }
  if (c->len == 0 && c->closed) {
    pthread_mutex_unlock(&c->mu);
    return tya_nil();
  }
  TyaValue value = c->buffer[c->head];
  c->buffer[c->head] = tya_nil();
  c->head = (c->head + 1) % c->capacity;
  c->len--;
  pthread_cond_signal(&c->not_full);
  pthread_mutex_unlock(&c->mu);
  return value;
}

TyaValue tya_channel_receive_timeout(TyaValue ch, TyaValue seconds) {
  if (ch.kind != TYA_CHANNEL || ch.channel == NULL) {
    tya_raise(tya_string("channel.receive_timeout: first argument must be a channel"));
    return tya_nil();
  }
  if (seconds.kind != TYA_NUMBER) {
    tya_raise(tya_string("channel.receive_timeout: seconds must be a number"));
    return tya_nil();
  }
  if (seconds.number < 0.0) {
    tya_raise(tya_string("channel.receive_timeout: seconds must be >= 0"));
    return tya_nil();
  }
  TyaChannel *c = ch.channel;
  struct timespec deadline;
#if defined(__APPLE__)
  struct timeval now;
  gettimeofday(&now, NULL);
  deadline.tv_sec = now.tv_sec + (time_t)seconds.number;
  long add_nsec = (long)((seconds.number - (double)((long)seconds.number)) * 1e9) + now.tv_usec * 1000;
  if (add_nsec >= 1000000000L) {
    deadline.tv_sec += add_nsec / 1000000000L;
    add_nsec %= 1000000000L;
  }
  deadline.tv_nsec = add_nsec;
#else
  clock_gettime(CLOCK_REALTIME, &deadline);
  deadline.tv_sec += (time_t)seconds.number;
  long add_nsec = (long)((seconds.number - (double)((long)seconds.number)) * 1e9) + deadline.tv_nsec;
  if (add_nsec >= 1000000000L) {
    deadline.tv_sec += add_nsec / 1000000000L;
    add_nsec %= 1000000000L;
  }
  deadline.tv_nsec = add_nsec;
#endif
  pthread_mutex_lock(&c->mu);
  while (c->len == 0 && !c->closed) {
    int rc = pthread_cond_timedwait(&c->not_empty, &c->mu, &deadline);
    if (rc == ETIMEDOUT) {
      pthread_mutex_unlock(&c->mu);
      return tya_nil();
    }
  }
  if (c->len == 0 && c->closed) {
    pthread_mutex_unlock(&c->mu);
    return tya_nil();
  }
  TyaValue value = c->buffer[c->head];
  c->buffer[c->head] = tya_nil();
  c->head = (c->head + 1) % c->capacity;
  c->len--;
  pthread_cond_signal(&c->not_full);
  pthread_mutex_unlock(&c->mu);
  return value;
}

TyaValue tya_channel_select(TyaValue ops) {
  if (ops.kind != TYA_ARRAY || ops.array == NULL) {
    tya_raise(tya_string("channel.select: argument must be an array of operations"));
    return tya_nil();
  }
  int n = ops.array->len;
  if (n == 0) {
    tya_raise(tya_string("channel.select: at least one operation is required"));
    return tya_nil();
  }
  /* Validate every op once before entering the polling loop. */
  for (int i = 0; i < n; i++) {
    TyaValue op = ops.array->items[i];
    if (op.kind != TYA_ARRAY || op.array == NULL || op.array->len < 2) {
      tya_raise(tya_string("channel.select: operation must be [channel, \"receive\"] or [channel, \"send\", value]"));
      return tya_nil();
    }
    TyaValue ch = op.array->items[0];
    TyaValue kind_v = op.array->items[1];
    if (ch.kind != TYA_CHANNEL || ch.channel == NULL) {
      tya_raise(tya_string("channel.select: operation channel must be a channel"));
      return tya_nil();
    }
    if (kind_v.kind != TYA_STRING) {
      tya_raise(tya_string("channel.select: operation kind must be \"receive\" or \"send\""));
      return tya_nil();
    }
    bool is_receive = strcmp(kind_v.string, "receive") == 0;
    bool is_send = strcmp(kind_v.string, "send") == 0;
    if (!is_receive && !is_send) {
      tya_raise(tya_string("channel.select: operation kind must be \"receive\" or \"send\""));
      return tya_nil();
    }
    if (is_send && op.array->len < 3) {
      tya_raise(tya_string("channel.select: send operation must include the value"));
      return tya_nil();
    }
  }
  /* Polling loop: try each op non-blocking; sleep briefly when nothing
   * is ready. */
  while (true) {
    for (int i = 0; i < n; i++) {
      TyaValue op = ops.array->items[i];
      TyaChannel *c = op.array->items[0].channel;
      const char *kind_s = op.array->items[1].string;
      if (strcmp(kind_s, "receive") == 0) {
        pthread_mutex_lock(&c->mu);
        if (c->len > 0) {
          TyaValue v = c->buffer[c->head];
          c->buffer[c->head] = tya_nil();
          c->head = (c->head + 1) % c->capacity;
          c->len--;
          pthread_cond_signal(&c->not_full);
          pthread_mutex_unlock(&c->mu);
          TyaDictEntry entries[3] = {
            {"index", tya_number((double)i)},
            {"kind", tya_string("receive")},
            {"value", v},
          };
          return tya_dict(entries, 3);
        }
        if (c->closed) {
          pthread_mutex_unlock(&c->mu);
          TyaDictEntry entries[3] = {
            {"index", tya_number((double)i)},
            {"kind", tya_string("receive")},
            {"value", tya_nil()},
          };
          return tya_dict(entries, 3);
        }
        pthread_mutex_unlock(&c->mu);
      } else {
        TyaValue value = op.array->items[2];
        pthread_mutex_lock(&c->mu);
        if (c->closed) {
          pthread_mutex_unlock(&c->mu);
          tya_raise(tya_string("channel.select: send on closed channel"));
          return tya_nil();
        }
        if (c->len < c->capacity) {
          int tail = (c->head + c->len) % c->capacity;
          c->buffer[tail] = value;
          c->len++;
          pthread_cond_signal(&c->not_empty);
          pthread_mutex_unlock(&c->mu);
          TyaDictEntry entries[3] = {
            {"index", tya_number((double)i)},
            {"kind", tya_string("send")},
            {"value", tya_nil()},
          };
          return tya_dict(entries, 3);
        }
        pthread_mutex_unlock(&c->mu);
      }
    }
    /* Nothing ready — sleep 1 ms then retry. */
    usleep(1000);
  }
}

TyaValue tya_channel_close(TyaValue ch) {
  if (ch.kind != TYA_CHANNEL || ch.channel == NULL) {
    tya_raise(tya_string("channel.close: argument must be a channel"));
    return tya_nil();
  }
  TyaChannel *c = ch.channel;
  pthread_mutex_lock(&c->mu);
  c->closed = true;
  pthread_cond_broadcast(&c->not_full);
  pthread_cond_broadcast(&c->not_empty);
  pthread_mutex_unlock(&c->mu);
  return tya_nil();
}

TyaValue tya_channel_closed(TyaValue ch) {
  if (ch.kind != TYA_CHANNEL || ch.channel == NULL) {
    tya_raise(tya_string("channel.closed?: argument must be a channel"));
    return tya_nil();
  }
  TyaChannel *c = ch.channel;
  pthread_mutex_lock(&c->mu);
  bool closed = c->closed;
  pthread_mutex_unlock(&c->mu);
  return tya_bool(closed);
}

/* =========================================================================
 * v0.42 STEP 7: sync primitives (mutex, atomic_integer, wait_group)
 * ========================================================================= */

static TyaResource *tya_resource_new(TyaResourceSubkind sub) {
  TyaResource *r = tya_gc_alloc(sizeof(TyaResource), TYA_GC_RESOURCE);
  r->subkind = sub;
  r->counter = 0;
  atomic_store(&r->atomic_value, 0);
  r->mu_initialized = false;
  r->cv_initialized = false;
  return r;
}

static TyaResource *tya_resource_check(TyaValue v, TyaResourceSubkind want, const char *op) {
  if (v.kind != TYA_RESOURCE || v.resource == NULL || v.resource->subkind != want) {
    char buf[128];
    const char *expected = "resource";
    switch (want) {
      case TYA_RES_MUTEX: expected = "mutex"; break;
      case TYA_RES_ATOMIC_INTEGER: expected = "atomic_integer"; break;
      case TYA_RES_WAIT_GROUP: expected = "wait_group"; break;
    }
    snprintf(buf, sizeof(buf), "%s: argument must be a %s", op, expected);
    tya_raise(tya_string(buf));
    return NULL;
  }
  return v.resource;
}

TyaValue tya_sync_mutex_new(void) {
  TyaResource *r = tya_resource_new(TYA_RES_MUTEX);
  pthread_mutex_init(&r->mu, NULL);
  r->mu_initialized = true;
  return (TyaValue){.kind = TYA_RESOURCE, .resource = r};
}

TyaValue tya_sync_lock(TyaValue m) {
  TyaResource *r = tya_resource_check(m, TYA_RES_MUTEX, "sync.lock");
  if (r == NULL) return tya_nil();
  pthread_mutex_lock(&r->mu);
  return tya_nil();
}

TyaValue tya_sync_unlock(TyaValue m) {
  TyaResource *r = tya_resource_check(m, TYA_RES_MUTEX, "sync.unlock");
  if (r == NULL) return tya_nil();
  pthread_mutex_unlock(&r->mu);
  return tya_nil();
}

TyaValue tya_sync_atomic_integer_new(TyaValue initial) {
  if (initial.kind != TYA_NUMBER) {
    tya_raise(tya_string("sync.atomic_integer: initial value must be a number"));
    return tya_nil();
  }
  TyaResource *r = tya_resource_new(TYA_RES_ATOMIC_INTEGER);
  atomic_store(&r->atomic_value, (long)initial.number);
  return (TyaValue){.kind = TYA_RESOURCE, .resource = r};
}

TyaValue tya_sync_atomic_integer_add(TyaValue a, TyaValue n) {
  TyaResource *r = tya_resource_check(a, TYA_RES_ATOMIC_INTEGER, "sync.atomic_integer.add");
  if (r == NULL) return tya_nil();
  if (n.kind != TYA_NUMBER) {
    tya_raise(tya_string("sync.atomic_integer.add: delta must be a number"));
    return tya_nil();
  }
  long old = atomic_fetch_add(&r->atomic_value, (long)n.number);
  return tya_number((double)(old + (long)n.number));
}

TyaValue tya_sync_atomic_integer_load(TyaValue a) {
  TyaResource *r = tya_resource_check(a, TYA_RES_ATOMIC_INTEGER, "sync.atomic_integer.load");
  if (r == NULL) return tya_nil();
  long v = atomic_load(&r->atomic_value);
  return tya_number((double)v);
}

TyaValue tya_sync_atomic_integer_store(TyaValue a, TyaValue n) {
  TyaResource *r = tya_resource_check(a, TYA_RES_ATOMIC_INTEGER, "sync.atomic_integer.store");
  if (r == NULL) return tya_nil();
  if (n.kind != TYA_NUMBER) {
    tya_raise(tya_string("sync.atomic_integer.store: value must be a number"));
    return tya_nil();
  }
  atomic_store(&r->atomic_value, (long)n.number);
  return tya_nil();
}

TyaValue tya_sync_atomic_integer_cas(TyaValue a, TyaValue expected, TyaValue new_value) {
  TyaResource *r = tya_resource_check(a, TYA_RES_ATOMIC_INTEGER, "sync.atomic_integer.compare_and_swap");
  if (r == NULL) return tya_nil();
  if (expected.kind != TYA_NUMBER || new_value.kind != TYA_NUMBER) {
    tya_raise(tya_string("sync.atomic_integer.compare_and_swap: expected and new must be numbers"));
    return tya_nil();
  }
  long e = (long)expected.number;
  long n = (long)new_value.number;
  bool ok = atomic_compare_exchange_strong(&r->atomic_value, &e, n);
  return tya_bool(ok);
}

TyaValue tya_sync_wait_group_new(void) {
  TyaResource *r = tya_resource_new(TYA_RES_WAIT_GROUP);
  pthread_mutex_init(&r->mu, NULL);
  pthread_cond_init(&r->cv, NULL);
  r->mu_initialized = true;
  r->cv_initialized = true;
  r->counter = 0;
  return (TyaValue){.kind = TYA_RESOURCE, .resource = r};
}

TyaValue tya_sync_wait_group_add(TyaValue wg, TyaValue n) {
  TyaResource *r = tya_resource_check(wg, TYA_RES_WAIT_GROUP, "sync.wait_group.add");
  if (r == NULL) return tya_nil();
  if (n.kind != TYA_NUMBER) {
    tya_raise(tya_string("sync.wait_group.add: count must be a number"));
    return tya_nil();
  }
  pthread_mutex_lock(&r->mu);
  r->counter += (long)n.number;
  if (r->counter < 0) {
    pthread_mutex_unlock(&r->mu);
    tya_raise(tya_string("sync.wait_group.add: counter would go negative"));
    return tya_nil();
  }
  if (r->counter == 0) {
    pthread_cond_broadcast(&r->cv);
  }
  pthread_mutex_unlock(&r->mu);
  return tya_nil();
}

TyaValue tya_sync_wait_group_done(TyaValue wg) {
  return tya_sync_wait_group_add(wg, tya_number(-1));
}

TyaValue tya_sync_wait_group_wait(TyaValue wg) {
  TyaResource *r = tya_resource_check(wg, TYA_RES_WAIT_GROUP, "sync.wait_group.wait");
  if (r == NULL) return tya_nil();
  pthread_mutex_lock(&r->mu);
  while (r->counter > 0) {
    pthread_cond_wait(&r->cv, &r->mu);
  }
  pthread_mutex_unlock(&r->mu);
  return tya_nil();
}
