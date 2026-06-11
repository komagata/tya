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

static void *tya_gc_alloc(size_t size, TyaGcKind kind);

/* Sub-tag for the multi-purpose TyaResource container. v0.42 STEP 7
 * uses one container kind to host the three sync primitives so the
 * value-kind switch table stays compact. */
typedef enum {
  TYA_RES_MUTEX = 1,
  TYA_RES_ATOMIC_INTEGER = 2,
  TYA_RES_WAIT_GROUP = 3,
  TYA_RES_STREAM = 4,
  TYA_RES_SOCKET = 5,
  TYA_RES_SOCKET_SERVER = 6,
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

struct TyaString {
  int len;
  bool ascii_only;
  const char *data;
};

typedef struct TyaDictMapEntry {
  const char *key;
  int index;
} TyaDictMapEntry;

struct TyaDict {
  TyaGcHeader gc;
  int len;
  int cap;
  TyaDictEntry *entries;
  int map_cap;
  TyaDictMapEntry *map;
};

struct TyaFunction {
  TyaGcHeader gc;
  TyaFunctionPtr fn;
  TyaValue receiver;
  TyaDict *members;
  const char *class_name;
  const char **params;
  int param_count;
  TyaValue parent;
  bool is_class;
};

static const char **tya_copy_params(const char **params, int param_count) {
  if (params == NULL || param_count <= 0) return NULL;
  const char **copy = malloc(sizeof(const char*) * param_count);
  if (copy == NULL) {
    fprintf(stderr, "tya: out of memory\n");
    exit(1);
  }
  for (int i = 0; i < param_count; i++) {
    copy[i] = params[i];
  }
  return copy;
}

static int tya_dict_entry_cap(int count) {
  int cap = 4;
  while (cap < count) {
    cap *= 2;
  }
  return cap;
}

static int tya_dict_map_cap(int count) {
  int cap = 8;
  while (cap < count * 2) {
    cap *= 2;
  }
  return cap;
}

static uint64_t tya_dict_hash(const char *key) {
  uint64_t hash = 1469598103934665603ULL;
  const unsigned char *p = (const unsigned char *)(key == NULL ? "" : key);
  while (*p != '\0') {
    hash ^= (uint64_t)(*p);
    hash *= 1099511628211ULL;
    p++;
  }
  return hash;
}

static bool tya_ascii_only_bytes(const char *data, int len) {
  if (data == NULL) return true;
  for (int i = 0; i < len; i++) {
    if (((unsigned char)data[i] & 0x80) != 0) return false;
  }
  return true;
}

static TyaString *tya_new_string_ref(const char *data, int len, bool ascii_only) {
  TyaString *s = malloc(sizeof(TyaString));
  if (s == NULL) {
    fprintf(stderr, "tya: out of memory\n");
    exit(1);
  }
  s->len = len;
  s->ascii_only = ascii_only;
  s->data = data == NULL ? "" : data;
  return s;
}

static TyaValue tya_string_with_len(const char *data, int len, bool ascii_only) {
  TyaString *s = tya_new_string_ref(data, len, ascii_only);
  return (TyaValue){.kind = TYA_STRING, .string = s->data, .string_value = s};
}

static TyaValue tya_string_alloc_len(int len, bool ascii_only) {
  TyaString *s = malloc(sizeof(TyaString) + (size_t)len + 1);
  if (s == NULL) {
    fprintf(stderr, "tya: out of memory\n");
    exit(1);
  }
  char *data = (char *)(s + 1);
  s->len = len;
  s->ascii_only = ascii_only;
  s->data = data;
  data[len] = '\0';
  return (TyaValue){.kind = TYA_STRING, .string = data, .string_value = s};
}

static int tya_string_byte_len_value(TyaValue value) {
  if (value.kind == TYA_STRING && value.string_value != NULL) return value.string_value->len;
  return value.string == NULL ? 0 : (int)strlen(value.string);
}

static bool tya_string_ascii_only_value(TyaValue value) {
  if (value.string_value != NULL) return value.string_value->ascii_only;
  static const char *last_string = NULL;
  static bool last_ascii_only = true;
  if (value.string == last_string) return last_ascii_only;
  int len = value.string == NULL ? 0 : (int)strlen(value.string);
  last_string = value.string;
  last_ascii_only = tya_ascii_only_bytes(value.string, len);
  return last_ascii_only;
}

static bool tya_ascii_only_text(const char *text) {
  static const char *last_text = NULL;
  static bool last_ascii_only = true;
  if (text == last_text) return last_ascii_only;
  int len = text == NULL ? 0 : (int)strlen(text);
  last_text = text;
  last_ascii_only = tya_ascii_only_bytes(text, len);
  return last_ascii_only;
}

static bool tya_string_equal_value(TyaValue left, TyaValue right) {
  if (left.string == NULL || right.string == NULL) {
    return left.string == right.string;
  }
  if (left.string_value == NULL && right.string_value == NULL) {
    return strcmp(left.string, right.string) == 0;
  }
  int left_len = tya_string_byte_len_value(left);
  int right_len = tya_string_byte_len_value(right);
  return left_len == right_len && memcmp(left.string, right.string, (size_t)left_len) == 0;
}

static void tya_dict_init(TyaDict *dict, int cap) {
  dict->len = 0;
  dict->cap = tya_dict_entry_cap(cap);
  dict->entries = malloc(sizeof(TyaDictEntry) * dict->cap);
  dict->map_cap = 0;
  dict->map = NULL;
}

static int tya_dict_map_slot(TyaDict *dict, const char *key) {
  if (dict == NULL || dict->map_cap <= 0 || key == NULL) return -1;
  uint64_t hash = tya_dict_hash(key);
  int mask = dict->map_cap - 1;
  for (int probe = 0; probe < dict->map_cap; probe++) {
    int slot = (int)((hash + (uint64_t)probe) & (uint64_t)mask);
    const char *entry_key = dict->map[slot].key;
    if (entry_key == NULL || strcmp(entry_key, key) == 0) {
      return slot;
    }
  }
  return -1;
}

static void tya_dict_rebuild_map(TyaDict *dict) {
  if (dict == NULL) return;
  int map_cap = tya_dict_map_cap(dict->len > 0 ? dict->len : 1);
  TyaDictMapEntry *map = calloc((size_t)map_cap, sizeof(TyaDictMapEntry));
  for (int i = 0; i < dict->len; i++) {
    const char *key = dict->entries[i].key;
    if (key == NULL) continue;
    uint64_t hash = tya_dict_hash(key);
    int mask = map_cap - 1;
    for (int probe = 0; probe < map_cap; probe++) {
      int slot = (int)((hash + (uint64_t)probe) & (uint64_t)mask);
      if (map[slot].key == NULL) {
        map[slot] = (TyaDictMapEntry){key, i};
        break;
      }
      if (strcmp(map[slot].key, key) == 0) {
        break;
      }
    }
  }
  free(dict->map);
  dict->map_cap = map_cap;
  dict->map = map;
}

static TyaDictEntry *tya_dict_find_entry(TyaDict *dict, const char *key) {
  if (dict == NULL || key == NULL || dict->len <= 0) return NULL;
  if (dict->map == NULL || dict->map_cap <= 0) {
    tya_dict_rebuild_map(dict);
  }
  int slot = tya_dict_map_slot(dict, key);
  if (slot < 0 || dict->map[slot].key == NULL) return NULL;
  return &dict->entries[dict->map[slot].index];
}

static void tya_dict_ensure_entry_cap(TyaDict *dict, int needed) {
  if (dict->cap >= needed) return;
  int cap = dict->cap <= 0 ? 4 : dict->cap;
  while (cap < needed) {
    cap *= 2;
  }
  dict->entries = realloc(dict->entries, sizeof(TyaDictEntry) * cap);
  dict->cap = cap;
}

static void tya_dict_set_entry(TyaDict *dict, const char *key, TyaValue value) {
  if (dict == NULL || key == NULL) return;
  TyaDictEntry *entry = tya_dict_find_entry(dict, key);
  if (entry != NULL) {
    entry->value = value;
    return;
  }
  tya_dict_ensure_entry_cap(dict, dict->len + 1);
  dict->entries[dict->len] = (TyaDictEntry){key, value};
  dict->len++;
  if (dict->map == NULL || dict->len * 2 >= dict->map_cap) {
    tya_dict_rebuild_map(dict);
    return;
  }
  int slot = tya_dict_map_slot(dict, key);
  if (slot >= 0) {
    dict->map[slot] = (TyaDictMapEntry){key, dict->len - 1};
  }
}

static void tya_dict_delete_entry(TyaDict *dict, const char *key) {
  TyaDictEntry *entry = tya_dict_find_entry(dict, key);
  if (dict == NULL || entry == NULL) return;
  int index = (int)(entry - dict->entries);
  for (int i = index + 1; i < dict->len; i++) {
    dict->entries[i - 1] = dict->entries[i];
  }
  dict->len--;
  tya_dict_rebuild_map(dict);
}

static TyaValue tya_class_number;
static TyaValue tya_class_string;
static TyaValue tya_class_array;
static TyaValue tya_class_dict;
static TyaValue tya_class_boolean;
static TyaValue tya_class_nil;
static bool tya_primitive_classes_initialized = false;

static TyaValue tya_primitive_member(TyaValue receiver, const char *key);
static char *tya_utf8_char_at(const char *text, int rune_index);
static int tya_utf8_byte_offset_at(const char *text, int rune_index);
static int tya_utf8_rune_index_at_byte(const char *text, int byte_index);
void tya_raise(TyaValue value);
static TyaValue tya_method_len(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_empty_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_to_string(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_inspect(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_to_i(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_to_f(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_compare(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_equal_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_lt_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_lte_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_gt_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_gte_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_between_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_abs(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_floor(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_ceil(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_round(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_trunc(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_sqrt(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_pow(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_log(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_log2(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_log10(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_exp(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_sin(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_cos(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_tan(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_asin(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_acos(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_atan(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_atan2(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_integer_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_finite_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_nan_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_byte_len(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_string_slice(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_trim(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_contains(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_string_index_of(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_starts_with(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_ends_with(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_replace(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_split(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_lines(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_chars(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_bytes(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_upper(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_lower(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_blank_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_present_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_first(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_last(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_push(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_pop(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_slice(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_reverse(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_sort(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_sort_by(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_join(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_map(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_filter(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_find(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_any(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_all(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_reduce(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_iter(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_sequence(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_has(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_get(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_set(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_delete(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_keys(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_values(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_entries(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_merge(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_merge_bang(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_channel_send(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_channel_receive(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_channel_receive_timeout(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_channel_close(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_channel_closed_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_task_cancel(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_task_cancelled_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_mutex_lock(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_mutex_unlock(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_mutex_with_lock(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_atomic_integer_add(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_atomic_integer_load(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_atomic_integer_store(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_atomic_integer_compare_and_swap(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_wait_group_add(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_wait_group_done(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_wait_group_wait(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_iterator_has_next(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_iterator_next(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_sequence_iter(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_sequence_map(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_sequence_filter(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_sequence_take(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_sequence_drop(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_sequence_reduce(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_sequence_each(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_sequence_any_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_sequence_all_p(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_sequence_find(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);
static TyaValue tya_method_sequence_to_a(TyaValue receiver, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f);

/* TyaResource owns a sync primitive (mutex / atomic / wait group).
 * The subkind drives which fields are valid. */
struct TyaResource {
  TyaGcHeader gc;
  TyaResourceSubkind subkind;
  pthread_mutex_t mu;       /* mutex + wait_group */
  pthread_cond_t cv;        /* wait_group only */
  long counter;             /* wait_group counter */
  atomic_long atomic_value; /* atomic_integer only */
  FILE *stream;             /* stream only */
  bool stream_borrowed;     /* stdin/stdout/stderr are not closed */
  bool stream_binary;
  bool stream_readable;
  bool stream_writable;
  bool stream_closed;
  TyaSocketHandle socket_fd; /* socket / socket_server only */
  bool socket_binary;
  bool socket_closed;
  double socket_timeout;
  void *tls_ssl;
  void *tls_ctx;
  bool mu_initialized;
  bool cv_initialized;
};

static TyaResource *tya_resource_new(TyaResourceSubkind sub);
static TyaResource *tya_resource_check(TyaValue v, TyaResourceSubkind want, const char *op);

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
  TyaTask *recv_waiters;
  TyaTask *send_waiters;
};

/* TyaTask is the runtime representation of a task value (v0.42).
 * v0.42 STEP 2 only declares the struct and links it through the GC;
 * STEP 3 wires spawn / await codegen against this layout. */
struct TyaTask {
  TyaGcHeader gc;
  pthread_t thread;
  pthread_mutex_t mu;
  pthread_cond_t cv;
  ucontext_t ctx;
  void *stack;
  size_t stack_size;
  bool done;
  bool joined;
  bool raised;
  bool queued;
  bool waiting;
  atomic_bool cancelled;
  TyaValue callee;       /* the callable that the task runs */
  int argc;              /* number of arguments (0..4) */
  TyaValue argv[4];      /* arguments evaluated in the spawning thread */
  TyaValue result;       /* return value when done && !raised */
  TyaValue raise_value;  /* in-flight raise to propagate to await */
  TyaValue pending_value;
  TyaValue waiting_value;
  bool channel_send_failed;
  bool sleeping;
  double wake_time;
  TyaTask *next_ready;
  TyaTask *next_sleep;
  TyaTask *next_waiter;
  TyaTask *next_channel_waiter;
  /* Every not-yet-joined task lives in a global doubly-linked list so
   * the collector treats them as roots. Without this, a top-level
   * spawn whose handle is dropped before the worker finishes would
   * be reclaimed mid-flight, freeing its mutex and pthread state. */
  struct TyaTask *prev_live;
  struct TyaTask *next_live;
  bool in_live_list;
};

static TyaTask *tya_live_tasks = NULL;
static TyaTask *tya_ready_head = NULL;
static TyaTask *tya_ready_tail = NULL;
static TyaTask *tya_sleep_head = NULL;
static ucontext_t tya_scheduler_ctx;
static bool tya_scheduler_ctx_valid = false;
static _Thread_local TyaTask *tya_current_task_ptr = NULL;

static void tya_live_tasks_add(TyaTask *t);
static void tya_live_tasks_remove(TyaTask *t);
static void tya_task_enqueue(TyaTask *t);
static void tya_task_sleep_until(TyaTask *t, double wake_time);
static void tya_task_wake_sleepers(void);
static void tya_scheduler_run_one(void);
static void tya_scheduler_run_until_task_done(TyaTask *t);
static void tya_task_yield(bool requeue);
static double tya_now_seconds(void);

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
      for (TyaTask *t = c->recv_waiters; t != NULL; t = t->next_channel_waiter) {
        tya_gc_mark_header((TyaGcHeader *)t);
      }
      for (TyaTask *t = c->send_waiters; t != NULL; t = t->next_channel_waiter) {
        tya_gc_mark_header((TyaGcHeader *)t);
        tya_gc_mark_value(t->waiting_value);
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
    case TYA_ERROR:
      if (v.dict) tya_gc_mark_header((TyaGcHeader *)v.dict);
      if (v.error_cause) tya_gc_mark_value(*v.error_cause);
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
      free(d->map);
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
      if (t->stack != NULL) free(t->stack);
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
#ifdef TYA_ENABLE_OPENSSL
      if (r->tls_ssl != NULL) SSL_free((SSL *)r->tls_ssl);
      if (r->tls_ctx != NULL) SSL_CTX_free((SSL_CTX *)r->tls_ctx);
#endif
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
static bool tya_data_object_equal(TyaValue left, TyaValue right);
static void tya_write_value(FILE *out, TyaValue value);
static void tya_build_value(TyaStringBuilder *builder, TyaValue value);
static void tya_builder_append(TyaStringBuilder *builder, const char *text);
static TyaValue tya_member_optional(TyaValue value, const char *key);

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

TyaValue tya_missing(void) {
  return (TyaValue){.kind = TYA_MISSING};
}

TyaValue tya_bool(bool value) {
  return (TyaValue){.kind = TYA_BOOL, .boolean = value};
}

TyaValue tya_number(double value) {
  return (TyaValue){.kind = TYA_NUMBER, .number = value, .number_is_int = floor(value) == value};
}

TyaValue tya_float(double value) {
  return (TyaValue){.kind = TYA_NUMBER, .number = value, .number_is_int = false};
}

TyaValue tya_string(const char *value) {
  const char *data = value == NULL ? "" : value;
  return (TyaValue){.kind = TYA_STRING, .string = data};
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
  tya_dict_init(dict, count);
  for (int i = 0; i < count; i++) {
    tya_dict_ensure_entry_cap(dict, dict->len + 1);
    dict->entries[dict->len] = entries[i];
    dict->len++;
  }
  tya_dict_rebuild_map(dict);
  return (TyaValue){.kind = TYA_DICT, .dict = dict};
}

TyaValue tya_object(void) {
  TyaDict *dict = tya_gc_alloc(sizeof(TyaDict), TYA_GC_DICT);
  tya_dict_init(dict, 0);
  return (TyaValue){.kind = TYA_OBJECT, .dict = dict};
}

TyaValue tya_function_raw(TyaFunctionPtr fn) {
  return tya_function_params_raw(fn, NULL, 0);
}

TyaValue tya_function_params_raw(TyaFunctionPtr fn, const char **params, int param_count) {
  TyaFunction *function = tya_gc_alloc(sizeof(TyaFunction), TYA_GC_FUNCTION);
  function->fn = fn;
  function->receiver = tya_nil();
  function->members = tya_gc_alloc(sizeof(TyaDict), TYA_GC_DICT);
  tya_dict_init(function->members, 0);
  function->class_name = NULL;
  function->params = tya_copy_params(params, param_count);
  function->param_count = param_count;
  function->parent = tya_nil();
  function->is_class = false;
  return (TyaValue){.kind = TYA_FUNCTION, .function = function};
}

TyaValue tya_class_raw(TyaFunctionPtr fn, const char *name, TyaValue parent) {
  TyaValue value = tya_function(fn);
  value.function->class_name = name;
  value.function->parent = parent;
  value.function->is_class = true;
  return value;
}

static void tya_init_primitive_classes(void) {
  if (tya_primitive_classes_initialized) {
    return;
  }
  tya_primitive_classes_initialized = true;
  tya_class_number = tya_class(NULL, "Number", tya_nil());
  tya_class_string = tya_class(NULL, "String", tya_nil());
  tya_class_array = tya_class(NULL, "Array", tya_nil());
  tya_class_dict = tya_class(NULL, "Dict", tya_nil());
  tya_class_boolean = tya_class(NULL, "Boolean", tya_nil());
  tya_class_nil = tya_class(NULL, "Nil", tya_nil());
}

TyaValue tya_primitive_class(const char *name) {
  tya_init_primitive_classes();
  if (strcmp(name, "Number") == 0) return tya_class_number;
  if (strcmp(name, "String") == 0) return tya_class_string;
  if (strcmp(name, "Array") == 0) return tya_class_array;
  if (strcmp(name, "Dict") == 0) return tya_class_dict;
  if (strcmp(name, "Boolean") == 0) return tya_class_boolean;
  if (strcmp(name, "Nil") == 0) return tya_class_nil;
  return tya_nil();
}

TyaValue tya_class_of(TyaValue value) {
  tya_init_primitive_classes();
  switch (value.kind) {
  case TYA_NIL:
    return tya_class_nil;
  case TYA_BOOL:
    return tya_class_boolean;
  case TYA_NUMBER:
    return tya_class_number;
  case TYA_STRING:
    return tya_class_string;
  case TYA_ARRAY:
    return tya_class_array;
  case TYA_DICT:
    return tya_class_dict;
  case TYA_OBJECT:
    return tya_member(value, "class");
  default:
    return tya_nil();
  }
}

TyaValue tya_bind_method_raw(TyaValue receiver, TyaFunctionPtr fn) {
  return tya_bind_method_params_raw(receiver, fn, NULL, 0);
}

TyaValue tya_bind_method_params_raw(TyaValue receiver, TyaFunctionPtr fn, const char **params, int param_count) {
  TyaFunction *function = tya_gc_alloc(sizeof(TyaFunction), TYA_GC_FUNCTION);
  function->fn = fn;
  function->receiver = receiver;
  function->members = tya_gc_alloc(sizeof(TyaDict), TYA_GC_DICT);
  tya_dict_init(function->members, 0);
  function->class_name = NULL;
  function->params = tya_copy_params(params, param_count);
  function->param_count = param_count;
  function->parent = tya_nil();
  function->is_class = false;
  return (TyaValue){.kind = TYA_FUNCTION, .function = function};
}

static void tya_error_options_panic(const char *message) {
  fprintf(stderr, "%s\n", message);
  exit(1);
}

static TyaValue tya_error_from_parts(TyaValue message, TyaValue options, bool has_options) {
  if (message.kind != TYA_STRING) {
    tya_error_options_panic("error message must be string");
  }
  TyaValue value = {
    .kind = TYA_ERROR,
    .error = message.string == NULL ? "" : message.string,
    .error_kind = "error",
    .error_code = "",
    .dict = NULL,
    .error_cause = NULL,
  };
  if (!has_options) {
    return value;
  }
  if ((options.kind != TYA_DICT && options.kind != TYA_OBJECT) || options.dict == NULL) {
    tya_error_options_panic("error options must be a dictionary");
  }
  for (int i = 0; i < options.dict->len; i++) {
    const char *key = options.dict->entries[i].key;
    TyaValue option = options.dict->entries[i].value;
    if (strcmp(key, "message") == 0) {
      if (option.kind != TYA_STRING) tya_error_options_panic("error options message must be string");
      value.error = option.string == NULL ? "" : option.string;
    } else if (strcmp(key, "kind") == 0) {
      if (option.kind != TYA_STRING) tya_error_options_panic("error options kind must be string");
      value.error_kind = option.string == NULL ? "" : option.string;
    } else if (strcmp(key, "code") == 0) {
      if (option.kind != TYA_STRING) tya_error_options_panic("error options code must be string");
      value.error_code = option.string == NULL ? "" : option.string;
    } else if (strcmp(key, "data") == 0) {
      if ((option.kind != TYA_DICT && option.kind != TYA_OBJECT) || option.dict == NULL) {
        tya_error_options_panic("error options data must be dictionary");
      }
      value.dict = option.dict;
    } else if (strcmp(key, "cause") == 0) {
      if (option.kind == TYA_NIL) {
        value.error_cause = NULL;
      } else if (option.kind == TYA_ERROR) {
        value.error_cause = malloc(sizeof(TyaValue));
        if (value.error_cause == NULL) tya_error_options_panic("error options cause allocation failed");
        *value.error_cause = option;
      } else {
        tya_error_options_panic("error options cause must be error or nil");
      }
    } else {
      tya_error_options_panic("error options unknown key");
    }
  }
  return value;
}

TyaValue tya_error(TyaValue message) {
  if ((message.kind == TYA_DICT || message.kind == TYA_OBJECT) && message.dict != NULL) {
    TyaValue text = tya_index(message, tya_string("message"));
    return tya_error_from_parts(text, message, true);
  }
  return tya_error_from_parts(message, tya_nil(), false);
}

TyaValue tya_error2(TyaValue message, TyaValue options) {
  return tya_error_from_parts(message, options, true);
}

static TyaValue tya_callable_target(TyaValue value) {
  if (value.kind == TYA_FUNCTION && value.function != NULL && value.function->fn != NULL) {
    return value;
  }
  if (value.kind == TYA_OBJECT && value.dict != NULL) {
    for (int i = 0; i < value.dict->len; i++) {
      if (value.dict->entries[i].key != NULL && strcmp(value.dict->entries[i].key, "call") == 0) {
        TyaValue method = value.dict->entries[i].value;
        if (method.kind == TYA_FUNCTION && method.function != NULL && method.function->fn != NULL) {
          return method;
        }
        break;
      }
    }
    tya_raise(tya_string("object is not callable"));
    return tya_nil();
  }
  return value;
}

TyaValue tya_call0(TyaValue fn) {
  fn = tya_callable_target(fn);
  if (fn.kind != TYA_FUNCTION || fn.function == NULL || fn.function->fn == NULL) return tya_nil();
  return fn.function->fn(fn.function->receiver, tya_nil(), tya_nil(), tya_nil(), tya_nil(), tya_nil(), tya_nil());
}

TyaValue tya_call1(TyaValue fn, TyaValue arg) {
  fn = tya_callable_target(fn);
  if (fn.kind != TYA_FUNCTION || fn.function == NULL || fn.function->fn == NULL) return tya_nil();
  return fn.function->fn(fn.function->receiver, arg, tya_nil(), tya_nil(), tya_nil(), tya_nil(), tya_nil());
}

TyaValue tya_call2(TyaValue fn, TyaValue first, TyaValue second) {
  fn = tya_callable_target(fn);
  if (fn.kind != TYA_FUNCTION || fn.function == NULL || fn.function->fn == NULL) return tya_nil();
  return fn.function->fn(fn.function->receiver, first, second, tya_nil(), tya_nil(), tya_nil(), tya_nil());
}

TyaValue tya_call3(TyaValue fn, TyaValue first, TyaValue second, TyaValue third) {
  fn = tya_callable_target(fn);
  if (fn.kind != TYA_FUNCTION || fn.function == NULL || fn.function->fn == NULL) return tya_nil();
  return fn.function->fn(fn.function->receiver, first, second, third, tya_nil(), tya_nil(), tya_nil());
}

TyaValue tya_call4(TyaValue fn, TyaValue first, TyaValue second, TyaValue third, TyaValue fourth) {
  fn = tya_callable_target(fn);
  if (fn.kind != TYA_FUNCTION || fn.function == NULL || fn.function->fn == NULL) return tya_nil();
  return fn.function->fn(fn.function->receiver, first, second, third, fourth, tya_nil(), tya_nil());
}

TyaValue tya_call5(TyaValue fn, TyaValue first, TyaValue second, TyaValue third, TyaValue fourth, TyaValue fifth) {
  fn = tya_callable_target(fn);
  if (fn.kind != TYA_FUNCTION || fn.function == NULL || fn.function->fn == NULL) return tya_nil();
  return fn.function->fn(fn.function->receiver, first, second, third, fourth, fifth, tya_nil());
}

TyaValue tya_call6(TyaValue fn, TyaValue first, TyaValue second, TyaValue third, TyaValue fourth, TyaValue fifth, TyaValue sixth) {
  fn = tya_callable_target(fn);
  if (fn.kind != TYA_FUNCTION || fn.function == NULL || fn.function->fn == NULL) return tya_nil();
  return fn.function->fn(fn.function->receiver, first, second, third, fourth, fifth, sixth);
}

TyaValue tya_call_keywords(TyaValue fn, const TyaValue *positional, int positional_count, TyaValue keywords) {
  fn = tya_callable_target(fn);
  if (fn.kind != TYA_FUNCTION || fn.function == NULL || fn.function->fn == NULL) return tya_nil();
  if (keywords.kind != TYA_DICT || keywords.dict == NULL) {
    tya_panic(tya_string("keyword expansion expects dictionary"));
  }
  TyaValue args[6] = {tya_missing(), tya_missing(), tya_missing(), tya_missing(), tya_missing(), tya_missing()};
  for (int i = 0; i < positional_count && i < 6; i++) {
    args[i] = positional[i];
  }
  for (int i = 0; i < keywords.dict->len; i++) {
    const char *key = keywords.dict->entries[i].key;
    int found = -1;
    for (int j = 0; j < fn.function->param_count; j++) {
      if (key != NULL && fn.function->params[j] != NULL && strcmp(key, fn.function->params[j]) == 0) {
        found = j;
        break;
      }
    }
    if (found < 0) {
      fprintf(stderr, "unknown keyword %s\n", key == NULL ? "" : key);
      exit(1);
    }
    if (found < positional_count || args[found].kind != TYA_MISSING) {
      fprintf(stderr, "argument %s supplied multiple times\n", key == NULL ? "" : key);
      exit(1);
    }
    args[found] = keywords.dict->entries[i].value;
  }
  return fn.function->fn(fn.function->receiver, args[0], args[1], args[2], args[3], args[4], args[5]);
}

void tya_keywords_merge(TyaValue target, TyaValue source) {
  if (target.kind != TYA_DICT || target.dict == NULL || source.kind != TYA_DICT || source.dict == NULL) {
    tya_panic(tya_string("keyword expansion expects dictionary"));
  }
  for (int i = 0; i < source.dict->len; i++) {
    tya_set_index(target, tya_string(source.dict->entries[i].key), source.dict->entries[i].value);
  }
}

static int tya_legacy_modules_enabled(void) {
  static int cached = -1;
  if (cached < 0) {
    cached = getenv("TYA_LEGACY_MODULES") != NULL ? 1 : 0;
  }
  return cached;
}

TyaValue tya_len(TyaValue value) {
  if (value.kind == TYA_STRING && value.string != NULL) {
    if (tya_legacy_modules_enabled()) {
      return tya_number((double)strlen(value.string));
    }
    if (tya_string_ascii_only_value(value)) {
      return tya_number(tya_string_len(value.string));
    }
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
  if ((value.kind == TYA_BYTES || value.kind == TYA_ARRAY || value.kind == TYA_STRING) && i < 0) {
    tya_raise(tya_string("negative indexes are invalid"));
    return tya_nil();
  }
  if (value.kind == TYA_BYTES && value.bytes != NULL && i >= 0 && i < value.bytes->len) {
    return tya_number((double)value.bytes->data[i]);
  }
  if (value.kind == TYA_ARRAY && value.array != NULL && i >= 0 && i < value.array->len) {
    return value.array->items[i];
  }
  if (value.kind == TYA_STRING && value.string != NULL && i >= 0) {
    if (tya_legacy_modules_enabled()) {
      int n = (int)strlen(value.string);
      if (i >= n) {
        return tya_nil();
      }
      char *out = malloc(2);
      out[0] = value.string[i];
      out[1] = '\0';
      return tya_string(out);
    }
    if (tya_string_ascii_only_value(value)) {
      int n = tya_string_len(value.string);
      if (i >= n) {
        return tya_nil();
      }
      char *out = malloc(2);
      out[0] = value.string[i];
      out[1] = '\0';
      return tya_string(out);
    }
    char *out = tya_utf8_char_at(value.string, i);
    return out == NULL ? tya_nil() : tya_string(out);
  }
  if ((value.kind == TYA_DICT || value.kind == TYA_OBJECT) && value.dict != NULL && index.kind == TYA_STRING && index.string != NULL) {
    for (int j = 0; j < value.dict->len; j++) {
      if (value.dict->entries[j].key != NULL && strcmp(value.dict->entries[j].key, index.string) == 0) {
        return value.dict->entries[j].value;
      }
    }
    return tya_nil();
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
  for (int i = 0; text[i] != '\0';) {
    unsigned char c = (unsigned char)text[i];
    int width = 1;
    if ((c & 0x80) == 0) {
      width = 1;
    } else if ((c & 0xE0) == 0xC0) {
      width = 2;
    } else if ((c & 0xF0) == 0xE0) {
      width = 3;
    } else if ((c & 0xF8) == 0xF0) {
      width = 4;
    }
    i += width;
    n++;
  }
  last_string = text;
  last_len = n;
  return n;
}

static int tya_utf8_valid_bytes(const unsigned char *data, int len) {
  for (int i = 0; i < len;) {
    unsigned char c = data[i];
    if ((c & 0x80) == 0) {
      i++;
      continue;
    }
    int width = 0;
    unsigned int min = 0;
    unsigned int code = 0;
    if ((c & 0xE0) == 0xC0) {
      width = 2;
      min = 0x80;
      code = c & 0x1F;
    } else if ((c & 0xF0) == 0xE0) {
      width = 3;
      min = 0x800;
      code = c & 0x0F;
    } else if ((c & 0xF8) == 0xF0) {
      width = 4;
      min = 0x10000;
      code = c & 0x07;
    } else {
      return 0;
    }
    if (i + width > len) {
      return 0;
    }
    for (int j = 1; j < width; j++) {
      unsigned char cc = data[i + j];
      if ((cc & 0xC0) != 0x80) {
        return 0;
      }
      code = (code << 6) | (cc & 0x3F);
    }
    if (code < min || code > 0x10FFFF || (code >= 0xD800 && code <= 0xDFFF)) {
      return 0;
    }
    i += width;
  }
  return 1;
}

static char *tya_utf8_char_at(const char *text, int rune_index) {
  if (text == NULL || rune_index < 0) {
    return NULL;
  }
  if (tya_ascii_only_text(text)) {
    int len = tya_string_len(text);
    if (rune_index >= len) return NULL;
    char *out = malloc(2);
    out[0] = text[rune_index];
    out[1] = '\0';
    return out;
  }
  int current = 0;
  for (int i = 0; text[i] != '\0';) {
    unsigned char c = (unsigned char)text[i];
    int width = 1;
    if ((c & 0x80) == 0) {
      width = 1;
    } else if ((c & 0xE0) == 0xC0) {
      width = 2;
    } else if ((c & 0xF0) == 0xE0) {
      width = 3;
    } else if ((c & 0xF8) == 0xF0) {
      width = 4;
    }
    if (current == rune_index) {
      char *out = malloc((size_t)width + 1);
      memcpy(out, text + i, (size_t)width);
      out[width] = '\0';
      return out;
    }
    i += width;
    current++;
  }
  return NULL;
}

static int tya_utf8_byte_offset_at(const char *text, int rune_index) {
  if (text == NULL || rune_index < 0) {
    return -1;
  }
  if (tya_ascii_only_text(text)) {
    int len = tya_string_len(text);
    return rune_index <= len ? rune_index : -1;
  }
  int current = 0;
  for (int i = 0; text[i] != '\0';) {
    if (current == rune_index) {
      return i;
    }
    unsigned char c = (unsigned char)text[i];
    int width = 1;
    if ((c & 0x80) == 0) {
      width = 1;
    } else if ((c & 0xE0) == 0xC0) {
      width = 2;
    } else if ((c & 0xF0) == 0xE0) {
      width = 3;
    } else if ((c & 0xF8) == 0xF0) {
      width = 4;
    }
    i += width;
    current++;
  }
  return current == rune_index ? (int)strlen(text) : -1;
}

static int tya_utf8_rune_index_at_byte(const char *text, int byte_index) {
  if (text == NULL || byte_index < 0) {
    return -1;
  }
  if (tya_ascii_only_text(text)) {
    int len = tya_string_len(text);
    return byte_index <= len ? byte_index : len;
  }
  int current = 0;
  for (int i = 0; text[i] != '\0' && i < byte_index;) {
    unsigned char c = (unsigned char)text[i];
    int width = 1;
    if ((c & 0x80) == 0) {
      width = 1;
    } else if ((c & 0xE0) == 0xC0) {
      width = 2;
    } else if ((c & 0xF0) == 0xE0) {
      width = 3;
    } else if ((c & 0xF8) == 0xF0) {
      width = 4;
    }
    i += width;
    current++;
  }
  return current;
}

static int tya_utf8_decode_first(const char *text) {
  if (text == NULL || text[0] == '\0') {
    return -1;
  }
  unsigned char c = (unsigned char)text[0];
  if ((c & 0x80) == 0) {
    return (int)c;
  }
  int width = 0;
  int code = 0;
  if ((c & 0xE0) == 0xC0) {
    width = 2;
    code = c & 0x1F;
  } else if ((c & 0xF0) == 0xE0) {
    width = 3;
    code = c & 0x0F;
  } else if ((c & 0xF8) == 0xF0) {
    width = 4;
    code = c & 0x07;
  } else {
    return -1;
  }
  for (int i = 1; i < width; i++) {
    unsigned char cc = (unsigned char)text[i];
    if ((cc & 0xC0) != 0x80) {
      return -1;
    }
    code = (code << 6) | (cc & 0x3F);
  }
  return code;
}

static char *tya_utf8_from_codepoint(int code) {
  if (code < 0 || code > 0x10FFFF || (code >= 0xD800 && code <= 0xDFFF)) {
    return NULL;
  }
  char *out = malloc(5);
  if (code <= 0x7F) {
    out[0] = (char)code;
    out[1] = '\0';
  } else if (code <= 0x7FF) {
    out[0] = (char)(0xC0 | (code >> 6));
    out[1] = (char)(0x80 | (code & 0x3F));
    out[2] = '\0';
  } else if (code <= 0xFFFF) {
    out[0] = (char)(0xE0 | (code >> 12));
    out[1] = (char)(0x80 | ((code >> 6) & 0x3F));
    out[2] = (char)(0x80 | (code & 0x3F));
    out[3] = '\0';
  } else {
    out[0] = (char)(0xF0 | (code >> 18));
    out[1] = (char)(0x80 | ((code >> 12) & 0x3F));
    out[2] = (char)(0x80 | ((code >> 6) & 0x3F));
    out[3] = (char)(0x80 | (code & 0x3F));
    out[4] = '\0';
  }
  return out;
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

