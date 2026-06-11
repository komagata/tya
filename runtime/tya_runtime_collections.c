TyaValue tya_member(TyaValue dict, const char *key) {
  TyaValue primitive = tya_primitive_member(dict, key);
  if (primitive.kind != TYA_NIL) {
    return primitive;
  }
  if (dict.kind == TYA_ERROR && strcmp(key, "message") == 0) {
    return tya_string(dict.error == NULL ? "" : dict.error);
  }
  if (dict.kind == TYA_ERROR && strcmp(key, "kind") == 0) {
    return tya_string(dict.error_kind == NULL ? "" : dict.error_kind);
  }
  if (dict.kind == TYA_ERROR && strcmp(key, "code") == 0) {
    return tya_string(dict.error_code == NULL ? "" : dict.error_code);
  }
  if (dict.kind == TYA_ERROR && strcmp(key, "data") == 0) {
    return dict.dict == NULL ? tya_dict(NULL, 0) : (TyaValue){.kind = TYA_DICT, .dict = dict.dict};
  }
  if (dict.kind == TYA_ERROR && strcmp(key, "cause") == 0) {
    return dict.error_cause == NULL ? tya_nil() : *dict.error_cause;
  }
  if (dict.kind == TYA_FUNCTION && dict.function != NULL && dict.function->members != NULL) {
    if (dict.function->is_class && key != NULL && strcmp(key, "name") == 0) {
      return tya_string(dict.function->class_name == NULL ? "" : dict.function->class_name);
    }
    if (dict.function->is_class && key != NULL && strcmp(key, "parent") == 0) {
      return dict.function->parent;
    }
    TyaDictEntry *entry = tya_dict_find_entry(dict.function->members, key);
    if (entry != NULL) {
      return entry->value;
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
  if (dict.kind == TYA_DICT) {
    bool module_namespace = false;
    TyaDictEntry *entry = tya_dict_find_entry(dict.dict, "__module_namespace");
    if (entry != NULL) {
      module_namespace = tya_truthy(entry->value);
    }
    if (!module_namespace) {
      fprintf(stderr, "cannot use . access on dictionary; use index access\n");
      exit(1);
    }
  }
  TyaDictEntry *entry = tya_dict_find_entry(dict.dict, key);
  if (entry != NULL) {
    return entry->value;
  }
  if (dict.kind == TYA_OBJECT) {
    if (strcmp(key, "to_string") == 0) return tya_bind_method(dict, tya_method_to_string);
    if (strcmp(key, "inspect") == 0) return tya_bind_method(dict, tya_method_inspect);
    fprintf(stderr, "missing object field or method: %s\n", key == NULL ? "" : key);
    exit(1);
  }
  return tya_nil();
}

void tya_set_member(TyaValue dict, const char *key, TyaValue value) {
  if (dict.kind == TYA_FUNCTION && dict.function != NULL && dict.function->members != NULL) {
    tya_dict_set_entry(dict.function->members, key, value);
    return;
  }
  if ((dict.kind != TYA_DICT && dict.kind != TYA_OBJECT) || dict.dict == NULL) {
    return;
  }
  if (dict.kind == TYA_OBJECT && value.kind != TYA_FUNCTION && key != NULL && key[0] != '@') {
    size_t hidden_len = strlen(key) + 2;
    char *hidden_key = malloc(hidden_len);
    snprintf(hidden_key, hidden_len, "@%s", key);
    TyaDictEntry *hidden_entry = tya_dict_find_entry(dict.dict, hidden_key);
    if (hidden_entry != NULL) {
      hidden_entry->value = value;
    }
    free(hidden_key);
  }
  TyaDictEntry *entry = tya_dict_find_entry(dict.dict, key);
  if (entry != NULL) {
    if (dict.kind == TYA_OBJECT && value.kind != TYA_FUNCTION && entry->value.kind == TYA_FUNCTION && key != NULL && key[0] != '@') {
      return;
    }
    entry->value = value;
    return;
  }
  tya_dict_set_entry(dict.dict, key, value);
}

TyaValue tya_dict_key_at(TyaValue dict, TyaValue index) {
  if (dict.kind != TYA_DICT || dict.dict == NULL) {
    return tya_nil();
  }
  int target = (int)index.number;
  if (target >= 0 && target < dict.dict->len) {
    return tya_string(dict.dict->entries[target].key);
  }
  return tya_nil();
}

TyaValue tya_dict_value_at(TyaValue dict, TyaValue index) {
  if (dict.kind != TYA_DICT || dict.dict == NULL) {
    return tya_nil();
  }
  int target = (int)index.number;
  if (target >= 0 && target < dict.dict->len) {
    return dict.dict->entries[target].value;
  }
  return tya_nil();
}

TyaValue tya_has(TyaValue dict, TyaValue key) {
  if (key.kind != TYA_STRING || key.string == NULL || dict.kind != TYA_DICT || dict.dict == NULL) {
    return tya_bool(false);
  }
  return tya_bool(tya_dict_find_entry(dict.dict, key.string) != NULL);
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
  TyaDictEntry *entry = tya_dict_find_entry(dict.dict, key.string);
  if (entry != NULL) {
    return entry->value;
  }
  return has_fallback ? fallback : tya_nil();
}

TyaValue tya_dict_set(TyaValue dict, TyaValue key, TyaValue value) {
  if (key.kind != TYA_STRING || key.string == NULL || dict.kind != TYA_DICT || dict.dict == NULL) {
    return tya_nil();
  }
  tya_set_index(dict, key, value);
  return tya_nil();
}

TyaValue tya_dict_delete(TyaValue dict, TyaValue key) {
  if (key.kind != TYA_STRING || key.string == NULL || dict.kind != TYA_DICT || dict.dict == NULL) {
    return tya_nil();
  }
  tya_dict_delete_entry(dict.dict, key.string);
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

TyaValue tya_dict_merge_bang(TyaValue left, TyaValue right) {
  if (left.kind == TYA_DICT && left.dict != NULL && right.kind == TYA_DICT && right.dict != NULL) {
    for (int i = 0; i < right.dict->len; i++) {
      if (right.dict->entries[i].key != NULL) {
        tya_set_index(left, tya_string(right.dict->entries[i].key), right.dict->entries[i].value);
      }
    }
  }
  return tya_nil();
}

TyaValue tya_dict_entries(TyaValue dict) {
  TyaValue out = tya_array(NULL, 0);
  if (dict.kind != TYA_DICT || dict.dict == NULL) {
    return out;
  }
  for (int i = 0; i < dict.dict->len; i++) {
    if (dict.dict->entries[i].key != NULL) {
      TyaValue pair_items[2] = {tya_string(dict.dict->entries[i].key), dict.dict->entries[i].value};
      tya_push(out, tya_array(pair_items, 2));
    }
  }
  return out;
}

TyaValue tya_dict_entry_at(TyaValue dict, TyaValue index) {
  TyaValue key = tya_dict_key_at(dict, index);
  if (key.kind != TYA_STRING) {
    return tya_nil();
  }
  TyaValue value = tya_dict_value_at(dict, index);
  TyaDictEntry entries[2] = {{"key", key}, {"value", value}};
  return tya_dict(entries, 2);
}

static TyaValue tya_iterator_object(const char *kind, TyaValue source) {
  TyaValue iter = tya_object();
  tya_set_member(iter, "__iter_kind", tya_string(kind));
  tya_set_member(iter, "source", source);
  tya_set_member(iter, "index", tya_number(0));
  tya_set_member(iter, "has_next?", tya_bind_method(iter, tya_method_iterator_has_next));
  tya_set_member(iter, "next", tya_bind_method(iter, tya_method_iterator_next));
  return iter;
}

static TyaValue tya_sequence_object(const char *kind, TyaValue source, TyaValue fn, TyaValue n) {
  TyaValue seq = tya_object();
  tya_set_member(seq, "__sequence_kind", tya_string(kind));
  tya_set_member(seq, "source", source);
  tya_set_member(seq, "fn", fn);
  tya_set_member(seq, "n", n);
  tya_set_member(seq, "iter", tya_bind_method(seq, tya_method_sequence_iter));
  tya_set_member(seq, "map", tya_bind_method(seq, tya_method_sequence_map));
  tya_set_member(seq, "filter", tya_bind_method(seq, tya_method_sequence_filter));
  tya_set_member(seq, "take", tya_bind_method(seq, tya_method_sequence_take));
  tya_set_member(seq, "drop", tya_bind_method(seq, tya_method_sequence_drop));
  tya_set_member(seq, "reduce", tya_bind_method(seq, tya_method_sequence_reduce));
  tya_set_member(seq, "each", tya_bind_method(seq, tya_method_sequence_each));
  tya_set_member(seq, "any?", tya_bind_method(seq, tya_method_sequence_any_p));
  tya_set_member(seq, "all?", tya_bind_method(seq, tya_method_sequence_all_p));
  tya_set_member(seq, "find", tya_bind_method(seq, tya_method_sequence_find));
  tya_set_member(seq, "to_a", tya_bind_method(seq, tya_method_sequence_to_a));
  return seq;
}

static TyaValue tya_sequence_iterator(const char *kind, TyaValue source_iter, TyaValue fn, TyaValue n) {
  TyaValue iter = tya_iterator_object(kind, source_iter);
  tya_set_member(iter, "fn", fn);
  tya_set_member(iter, "n", n);
  tya_set_member(iter, "count", tya_number(0));
  tya_set_member(iter, "pending_ready", tya_bool(false));
  tya_set_member(iter, "pending", tya_nil());
  if (strcmp(kind, "drop") == 0) {
    int limit = n.kind == TYA_NUMBER ? (int)n.number : 0;
    for (int i = 0; i < limit && tya_truthy(tya_iterator_has_next(source_iter)); i++) {
      (void)tya_iterator_next(source_iter);
    }
  }
  return iter;
}

static TyaValue tya_member_optional(TyaValue value, const char *key) {
  if ((value.kind != TYA_DICT && value.kind != TYA_OBJECT) || value.dict == NULL || key == NULL) {
    return tya_nil();
  }
  for (int i = 0; i < value.dict->len; i++) {
    if (value.dict->entries[i].key != NULL && strcmp(value.dict->entries[i].key, key) == 0) {
      return value.dict->entries[i].value;
    }
  }
  return tya_nil();
}

TyaValue tya_iter(TyaValue value) {
  if (value.kind == TYA_ARRAY) return tya_iterator_object("array", value);
  if (value.kind == TYA_STRING) return tya_iterator_object("string", value);
  if (value.kind == TYA_BYTES) return tya_iterator_object("bytes", value);
  if (value.kind == TYA_DICT) return tya_iterator_object("dict", value);
  TyaValue method = tya_member(value, "iter");
  if (method.kind == TYA_FUNCTION) return tya_call0(method);
  tya_raise(tya_string("for: value is not iterable"));
  return tya_nil();
}

TyaValue tya_iterator_has_next(TyaValue iter) {
  if (iter.kind != TYA_DICT && iter.kind != TYA_OBJECT) {
    TyaValue method = tya_member(iter, "has_next?");
    if (method.kind == TYA_FUNCTION) return tya_call0(method);
    return tya_bool(false);
  }
  TyaValue kind = tya_member_optional(iter, "__iter_kind");
  if (kind.kind != TYA_STRING || kind.string == NULL) {
    TyaValue method = tya_member(iter, "has_next?");
    if (method.kind == TYA_FUNCTION) return tya_call0(method);
    return tya_bool(false);
  }
  TyaValue source = tya_member(iter, "source");
  int index = (int)tya_member(iter, "index").number;
  if (strcmp(kind.string, "array") == 0 || strcmp(kind.string, "string") == 0 || strcmp(kind.string, "bytes") == 0 || strcmp(kind.string, "dict") == 0) {
    return tya_bool(index < (int)tya_len(source).number);
  }
  if (strcmp(kind.string, "map") == 0 || strcmp(kind.string, "drop") == 0) {
    return tya_iterator_has_next(source);
  }
  if (strcmp(kind.string, "take") == 0) {
    TyaValue n = tya_member(iter, "n");
    int limit = n.kind == TYA_NUMBER ? (int)n.number : 0;
    int count = (int)tya_member(iter, "count").number;
    return tya_bool(count < limit && tya_truthy(tya_iterator_has_next(source)));
  }
  if (strcmp(kind.string, "filter") == 0) {
    if (tya_truthy(tya_member(iter, "pending_ready"))) return tya_bool(true);
    TyaValue fn = tya_member(iter, "fn");
    while (tya_truthy(tya_iterator_has_next(source))) {
      TyaValue item = tya_iterator_next(source);
      if (tya_truthy(tya_call1(fn, item))) {
        tya_set_member(iter, "pending", item);
        tya_set_member(iter, "pending_ready", tya_bool(true));
        return tya_bool(true);
      }
    }
    return tya_bool(false);
  }
  return tya_bool(false);
}

TyaValue tya_iterator_next(TyaValue iter) {
  if (!tya_truthy(tya_iterator_has_next(iter))) {
    tya_raise(tya_string("iterator.next: iterator exhausted"));
    return tya_nil();
  }
  TyaValue kind = tya_member_optional(iter, "__iter_kind");
  if (kind.kind != TYA_STRING || kind.string == NULL) {
    TyaValue method = tya_member(iter, "next");
    if (method.kind == TYA_FUNCTION) return tya_call0(method);
    tya_raise(tya_string("iterator.next: iterator exhausted"));
    return tya_nil();
  }
  TyaValue source = tya_member_optional(iter, "source");
  int index = (int)tya_member_optional(iter, "index").number;
  tya_set_member(iter, "index", tya_number((double)(index + 1)));
  if (strcmp(kind.string, "array") == 0 || strcmp(kind.string, "string") == 0 || strcmp(kind.string, "bytes") == 0) {
    return tya_index(source, tya_number((double)index));
  }
  if (strcmp(kind.string, "dict") == 0) {
    return tya_dict_entry_at(source, tya_number((double)index));
  }
  if (strcmp(kind.string, "map") == 0) {
    return tya_call1(tya_member(iter, "fn"), tya_iterator_next(source));
  }
  if (strcmp(kind.string, "filter") == 0) {
    TyaValue item = tya_member(iter, "pending");
    tya_set_member(iter, "pending", tya_nil());
    tya_set_member(iter, "pending_ready", tya_bool(false));
    return item;
  }
  if (strcmp(kind.string, "take") == 0) {
    int count = (int)tya_member(iter, "count").number;
    tya_set_member(iter, "count", tya_number((double)(count + 1)));
    return tya_iterator_next(source);
  }
  if (strcmp(kind.string, "drop") == 0) {
    return tya_iterator_next(source);
  }
  return tya_nil();
}

TyaValue tya_sequence(TyaValue value) {
  return tya_sequence_object("iterable", value, tya_nil(), tya_nil());
}

TyaValue tya_contains(TyaValue text, TyaValue part) {
  if (text.kind != TYA_STRING || part.kind != TYA_STRING || text.string == NULL || part.string == NULL) {
    return tya_bool(false);
  }
  return tya_bool(strstr(text.string, part.string) != NULL);
}

TyaValue tya_contains_method(TyaValue receiver, TyaValue value) {
  if (receiver.kind == TYA_ARRAY) {
    return tya_array_contains(receiver, value);
  }
  return tya_contains(receiver, value);
}

TyaValue tya_string_index_of(TyaValue text, TyaValue needle, TyaValue start) {
  if (text.kind != TYA_STRING || needle.kind != TYA_STRING || text.string == NULL || needle.string == NULL) {
    return tya_number(-1);
  }
  int rune_start = 0;
  if (start.kind != TYA_NIL && start.kind != TYA_MISSING) {
    if (start.kind != TYA_NUMBER) {
      return tya_number(-1);
    }
    rune_start = (int)start.number;
  }
  if (rune_start < 0) {
    tya_panic(tya_string("string.index_of does not support negative indexes"));
  }
  int start_byte = tya_utf8_byte_offset_at(text.string, rune_start);
  if (start_byte < 0) {
    return tya_number(-1);
  }
  char *found = strstr(text.string + start_byte, needle.string);
  if (found == NULL) {
    return tya_number(-1);
  }
  return tya_number(tya_utf8_rune_index_at_byte(text.string, (int)(found - text.string)));
}

TyaValue tya_starts_with(TyaValue text, TyaValue prefix) {
  if (text.kind != TYA_STRING || prefix.kind != TYA_STRING || text.string == NULL || prefix.string == NULL) {
    return tya_bool(false);
  }
  int text_len = tya_string_byte_len_value(text);
  int prefix_len = tya_string_byte_len_value(prefix);
  return tya_bool(prefix_len <= text_len && memcmp(text.string, prefix.string, (size_t)prefix_len) == 0);
}

TyaValue tya_ends_with(TyaValue text, TyaValue suffix) {
  if (text.kind != TYA_STRING || suffix.kind != TYA_STRING || text.string == NULL || suffix.string == NULL) {
    return tya_bool(false);
  }
  int text_len = tya_string_byte_len_value(text);
  int suffix_len = tya_string_byte_len_value(suffix);
  if (suffix_len > text_len) {
    return tya_bool(false);
  }
  return tya_bool(memcmp(text.string + text_len - suffix_len, suffix.string, (size_t)suffix_len) == 0);
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
  size_t old_len = (size_t)tya_string_byte_len_value(old);
  if (old_len == 0) {
    return text;
  }
  size_t replacement_len = (size_t)tya_string_byte_len_value(replacement);
  size_t count = 0;
  const char *cursor = text.string;
  while ((cursor = strstr(cursor, old.string)) != NULL) {
    count++;
    cursor += old_len;
  }
  size_t text_len = (size_t)tya_string_byte_len_value(text);
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
  return tya_string_with_len(out, (int)out_len, tya_ascii_only_bytes(out, (int)out_len));
}

TyaValue tya_byte_len(TyaValue text) {
  if (text.kind != TYA_STRING || text.string == NULL) {
    return tya_number(0);
  }
  return tya_number((double)tya_string_byte_len_value(text));
}

TyaValue tya_string_slice(TyaValue text, TyaValue start, TyaValue end) {
  if (text.kind != TYA_STRING || text.string == NULL || start.kind != TYA_NUMBER || end.kind != TYA_NUMBER) {
    return tya_string("");
  }
  int s = (int)start.number;
  int e = (int)end.number;
  if (s < 0 || e < 0) {
    tya_panic(tya_string("string.slice does not support negative indexes"));
  }
  if (s > e) {
    tya_panic(tya_string("string.slice index out of range"));
  }
  int start_byte = tya_utf8_byte_offset_at(text.string, s);
  int end_byte = tya_utf8_byte_offset_at(text.string, e);
  if (start_byte < 0 || end_byte < 0) {
    tya_panic(tya_string("string.slice index out of range"));
  }
  int len = end_byte - start_byte;
  char *out = malloc((size_t)len + 1);
  memcpy(out, text.string + start_byte, (size_t)len);
  out[len] = '\0';
  return tya_string_with_len(out, len, tya_ascii_only_bytes(out, len));
}

TyaValue tya_ord(TyaValue text) {
  if (text.kind != TYA_STRING || text.string == NULL || text.string[0] == '\0') {
    tya_raise(tya_string("ord: argument must be a non-empty string"));
    return tya_nil();
  }
  int code = tya_utf8_decode_first(text.string);
  if (code < 0) {
    tya_raise(tya_string("ord: invalid UTF-8"));
    return tya_nil();
  }
  return tya_number((double)code);
}

TyaValue tya_kind(TyaValue value) {
  switch (value.kind) {
  case TYA_NIL:
  case TYA_MISSING:
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
      case TYA_RES_STREAM: return tya_string("stream");
      case TYA_RES_SOCKET: return tya_string("socket");
      case TYA_RES_SOCKET_SERVER: return tya_string("socket_server");
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
  char *out = tya_utf8_from_codepoint(v);
  if (out == NULL) {
    tya_raise(tya_string("chr: code point out of range"));
    return tya_nil();
  }
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
  case TYA_MISSING:
    return true;
  case TYA_BOOL:
    return left.boolean == right.boolean;
  case TYA_NUMBER:
    return left.number == right.number;
  case TYA_STRING:
    return tya_string_equal_value(left, right);
  case TYA_ARRAY:
    return tya_deep_equal_bool(left, right);
  case TYA_DICT:
    return tya_deep_equal_bool(left, right);
  case TYA_OBJECT:
    if (tya_member_optional(left, "__data_type").kind == TYA_STRING || tya_member_optional(right, "__data_type").kind == TYA_STRING) {
      return tya_data_object_equal(left, right);
    }
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

static bool tya_data_object_equal(TyaValue left, TyaValue right) {
  if (left.kind != TYA_OBJECT || right.kind != TYA_OBJECT || left.dict == NULL || right.dict == NULL) {
    return false;
  }
  TyaValue left_type = tya_member_optional(left, "__data_type");
  TyaValue right_type = tya_member_optional(right, "__data_type");
  if (left_type.kind != TYA_STRING || right_type.kind != TYA_STRING || left_type.string == NULL || right_type.string == NULL) {
    return false;
  }
  if (!tya_string_equal_value(left_type, right_type)) {
    return false;
  }
  for (int i = 0; i < left.dict->len; i++) {
    const char *key = left.dict->entries[i].key;
    if (key == NULL || strncmp(key, "__", 2) == 0 || strcmp(key, "with") == 0) {
      continue;
    }
    bool found = false;
    TyaValue right_value = tya_nil();
    for (int j = 0; j < right.dict->len; j++) {
      if (right.dict->entries[j].key != NULL && strcmp(right.dict->entries[j].key, key) == 0) {
        found = true;
        right_value = right.dict->entries[j].value;
        break;
      }
    }
    if (!found) return false;
    if (!tya_deep_equal_bool(left.dict->entries[i].value, right_value)) {
      return false;
    }
  }
  for (int i = 0; i < right.dict->len; i++) {
    const char *key = right.dict->entries[i].key;
    if (key == NULL || strncmp(key, "__", 2) == 0 || strcmp(key, "with") == 0) {
      continue;
    }
    bool found = false;
    for (int j = 0; j < left.dict->len; j++) {
      if (left.dict->entries[j].key != NULL && strcmp(left.dict->entries[j].key, key) == 0) {
        found = true;
        break;
      }
    }
    if (!found) return false;
  }
  return true;
}

static bool tya_deep_equal_bool(TyaValue left, TyaValue right) {
  if (left.kind != right.kind) {
    return false;
  }
  switch (left.kind) {
  case TYA_NIL:
  case TYA_MISSING:
    return true;
  case TYA_BOOL:
    return left.boolean == right.boolean;
  case TYA_NUMBER:
    return left.number == right.number;
  case TYA_STRING:
    return tya_string_equal_value(left, right);
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
      if (!tya_truthy(tya_has(right, tya_string(key)))) {
        return false;
      }
      TyaValue right_value = tya_dict_get(right, tya_string(key), tya_nil(), false);
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

TyaValue tya_compare(TyaValue left, TyaValue right) {
  if (left.kind == TYA_NUMBER && right.kind == TYA_NUMBER) {
    if (left.number < right.number) return tya_number(-1);
    if (left.number > right.number) return tya_number(1);
    return tya_number(0);
  }
  if (left.kind == TYA_STRING && right.kind == TYA_STRING && left.string != NULL && right.string != NULL) {
    int result = strcmp(left.string, right.string);
    if (result < 0) return tya_number(-1);
    if (result > 0) return tya_number(1);
    return tya_number(0);
  }
  tya_raise(tya_string("compare: values are not comparable"));
  return tya_nil();
}

TyaValue tya_order_compare(TyaValue left, TyaValue right) {
  if (left.kind == TYA_NUMBER && right.kind == TYA_NUMBER) {
    if (left.number < right.number) return tya_number(-1);
    if (left.number > right.number) return tya_number(1);
    return tya_number(0);
  }
  if (left.kind == TYA_STRING && right.kind == TYA_STRING) {
    int cmp = strcmp(left.string == NULL ? "" : left.string, right.string == NULL ? "" : right.string);
    if (cmp < 0) return tya_number(-1);
    if (cmp > 0) return tya_number(1);
    return tya_number(0);
  }
  if (left.kind == TYA_NIL && right.kind == TYA_NUMBER) {
    return tya_number(-1);
  }
  if (left.kind == TYA_NUMBER && right.kind == TYA_NIL) {
    return tya_number(1);
  }
  tya_raise(tya_string("< expects numbers or strings of the same kind"));
  return tya_nil();
}

TyaValue tya_mod(TyaValue left, TyaValue right) {
  if (left.kind != TYA_NUMBER || right.kind != TYA_NUMBER) {
    tya_raise(tya_string("% expects numbers"));
    return tya_nil();
  }
  long l = (long)left.number;
  long r = (long)right.number;
  if (!left.number_is_int || !right.number_is_int || (double)l != left.number || (double)r != right.number) {
    tya_raise(tya_string("% expects integers"));
    return tya_nil();
  }
  if (r == 0) {
    tya_raise(tya_string("modulo by zero"));
    return tya_nil();
  }
  return tya_number(l % r);
}

TyaValue tya_div(TyaValue left, TyaValue right) {
  if (left.kind != TYA_NUMBER || right.kind != TYA_NUMBER) {
    tya_raise(tya_string("/ expects numbers"));
    return tya_nil();
  }
  if (right.number == 0) {
    tya_raise(tya_string("division by zero"));
    return tya_nil();
  }
  long l = (long)left.number;
  long r = (long)right.number;
  if (left.number_is_int && right.number_is_int && (double)l == left.number && (double)r == right.number) {
    return tya_number(l / r);
  }
  return tya_float(left.number / right.number);
}

TyaValue tya_add(TyaValue left, TyaValue right) {
  if (left.kind == TYA_BYTES && right.kind == TYA_BYTES && left.bytes != NULL && right.bytes != NULL) {
    return tya_bytes_concat(left, right);
  }
  if (left.kind == TYA_STRING && right.kind == TYA_STRING && left.string != NULL && right.string != NULL) {
    size_t left_len = (size_t)tya_string_byte_len_value(left);
    size_t right_len = (size_t)tya_string_byte_len_value(right);
    bool ascii_only = tya_string_ascii_only_value(left) && tya_string_ascii_only_value(right);
    TyaValue out = tya_string_alloc_len((int)(left_len + right_len), ascii_only);
    memcpy((char *)out.string, left.string, left_len);
    memcpy((char *)out.string + left_len, right.string, right_len);
    return out;
  }
  if (tya_legacy_modules_enabled() && (left.kind == TYA_STRING || right.kind == TYA_STRING)) {
    return tya_add(tya_to_string(left), tya_to_string(right));
  }
  if (left.kind == TYA_STRING || right.kind == TYA_STRING || left.kind == TYA_BYTES || right.kind == TYA_BYTES) {
    tya_raise(tya_string("+ expects numbers, strings, or bytes of the same kind"));
  }
  if (left.kind != TYA_NUMBER || right.kind != TYA_NUMBER) {
    tya_raise(tya_string("+ expects numbers, strings, or bytes of the same kind"));
  }
  TyaValue out = tya_number(left.number + right.number);
  out.number_is_int = left.number_is_int && right.number_is_int;
  return out;
}

TyaValue tya_sub(TyaValue left, TyaValue right) {
  TyaValue left_time = tya_member_optional(left, "__time_seconds");
  TyaValue right_time = tya_member_optional(right, "__time_seconds");
  if (left_time.kind == TYA_NUMBER && right_time.kind == TYA_NUMBER) {
    return tya_float(left_time.number - right_time.number);
  }
  if (left.kind != TYA_NUMBER || right.kind != TYA_NUMBER) {
    tya_raise(tya_string("- expects numbers"));
    return tya_nil();
  }
  TyaValue out = tya_number(left.number - right.number);
  out.number_is_int = left.number_is_int && right.number_is_int;
  return out;
}

TyaValue tya_mul(TyaValue left, TyaValue right) {
  if (left.kind != TYA_NUMBER || right.kind != TYA_NUMBER) {
    tya_raise(tya_string("* expects numbers"));
    return tya_nil();
  }
  TyaValue out = tya_number(left.number * right.number);
  out.number_is_int = left.number_is_int && right.number_is_int;
  return out;
}

TyaValue tya_and(TyaValue left, TyaValue right) {
  return tya_bool(tya_truthy(left) && tya_truthy(right));
}

TyaValue tya_or(TyaValue left, TyaValue right) {
  return tya_bool(tya_truthy(left) || tya_truthy(right));
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
    return tya_chars(text);
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

TyaValue tya_chars(TyaValue text) {
  TyaValue out = tya_array(NULL, 0);
  if (text.kind != TYA_STRING || text.string == NULL) {
    return out;
  }
  int i = 0;
  while (text.string[i] != '\0') {
    tya_push(out, tya_string(tya_substr(text.string, i, 1)));
    i++;
  }
  return out;
}

TyaValue tya_join(TyaValue array, TyaValue sep) {
  if (array.kind != TYA_ARRAY || array.array == NULL || sep.kind != TYA_STRING || sep.string == NULL) {
    return tya_string("");
  }
  size_t sep_len = strlen(sep.string);
  size_t total = 0;
  for (int i = 0; i < array.array->len; i++) {
    if (i > 0) {
      total += sep_len;
    }
    TyaValue item = tya_to_string(array.array->items[i]);
    if (item.string != NULL) {
      total += strlen(item.string);
    }
  }
  char *out = malloc(total + 1);
  if (out == NULL) {
    fprintf(stderr, "tya: out of memory\n");
    exit(1);
  }
  char *dst = out;
  for (int i = 0; i < array.array->len; i++) {
    if (i > 0 && sep_len > 0) {
      memcpy(dst, sep.string, sep_len);
      dst += sep_len;
    }
    TyaValue item = tya_to_string(array.array->items[i]);
    if (item.string != NULL) {
      size_t item_len = strlen(item.string);
      memcpy(dst, item.string, item_len);
      dst += item_len;
    }
  }
  *dst = '\0';
  return tya_string(out);
}

TyaValue tya_to_string(TyaValue value) {
  if (value.kind == TYA_OBJECT) {
    TyaValue data_type = tya_member_optional(value, "__data_type");
    if (data_type.kind == TYA_STRING) return tya_inspect(value);
    TyaValue method = tya_member_optional(value, "to_string");
    if (method.kind == TYA_FUNCTION) {
      TyaValue result = tya_call0(method);
      if (result.kind != TYA_STRING) {
        fprintf(stderr, "to_string must return string\n");
        exit(1);
      }
      return result;
    }
  }
  if (value.kind == TYA_STRING) {
    return value;
  }
  TyaStringBuilder builder = {.text = malloc(64), .len = 0, .cap = 64};
  builder.text[0] = '\0';
  tya_build_value(&builder, value);
  return tya_string(builder.text);
}

static void tya_build_inspect(TyaStringBuilder *builder, TyaValue value);

static void tya_builder_append_quoted(TyaStringBuilder *builder, const char *text) {
  tya_builder_append(builder, "\"");
  if (text != NULL) {
    for (const char *p = text; *p != '\0'; p++) {
      switch (*p) {
      case '\\':
        tya_builder_append(builder, "\\\\");
        break;
      case '"':
        tya_builder_append(builder, "\\\"");
        break;
      case '\n':
        tya_builder_append(builder, "\\n");
        break;
      case '\r':
        tya_builder_append(builder, "\\r");
        break;
      case '\t':
        tya_builder_append(builder, "\\t");
        break;
      default: {
        char one[2] = {*p, '\0'};
        tya_builder_append(builder, one);
        break;
      }
      }
    }
  }
  tya_builder_append(builder, "\"");
}

static void tya_build_object_inspect(TyaStringBuilder *builder, TyaValue value, const char *type_name) {
  tya_builder_append(builder, type_name == NULL || type_name[0] == '\0' ? "Object" : type_name);
  tya_builder_append(builder, "(");
  if (value.dict != NULL) {
    int written = 0;
    for (int i = 0; i < value.dict->len; i++) {
      const char *key = value.dict->entries[i].key;
      TyaValue item = value.dict->entries[i].value;
      if (key == NULL || key[0] == '_' || key[0] == '@' || item.kind == TYA_FUNCTION || strcmp(key, "class") == 0 || strcmp(key, "class_name") == 0) continue;
      if (written > 0) tya_builder_append(builder, ", ");
      tya_builder_append(builder, key);
      tya_builder_append(builder, ": ");
      tya_build_inspect(builder, item);
      written++;
    }
  }
  tya_builder_append(builder, ")");
}

TyaValue tya_inspect(TyaValue value) {
  if (value.kind == TYA_OBJECT) {
    TyaValue method = tya_member_optional(value, "inspect");
    if (method.kind == TYA_FUNCTION) {
      TyaValue result = tya_call0(method);
      if (result.kind != TYA_STRING) {
        fprintf(stderr, "inspect must return string\n");
        exit(1);
      }
      return result;
    }
  }
  TyaStringBuilder builder = {.text = malloc(64), .len = 0, .cap = 64};
  builder.text[0] = '\0';
  tya_build_inspect(&builder, value);
  return tya_string(builder.text);
}

static void tya_build_inspect(TyaStringBuilder *builder, TyaValue value) {
  char scratch[64];
  switch (value.kind) {
  case TYA_STRING:
    tya_builder_append_quoted(builder, value.string);
    break;
  case TYA_ARRAY:
    tya_builder_append(builder, "[");
    if (value.array != NULL) {
      for (int i = 0; i < value.array->len; i++) {
        if (i > 0) tya_builder_append(builder, ", ");
        tya_build_inspect(builder, value.array->items[i]);
      }
    }
    tya_builder_append(builder, "]");
    break;
  case TYA_DICT:
    tya_builder_append(builder, "{");
    if (value.dict != NULL) {
      int written = 0;
      for (int i = 0; i < value.dict->len; i++) {
        const char *key = value.dict->entries[i].key;
        if (key == NULL || key[0] == '_') continue;
        if (written > 0) tya_builder_append(builder, ", ");
        tya_builder_append_quoted(builder, key);
        tya_builder_append(builder, ": ");
        tya_build_inspect(builder, value.dict->entries[i].value);
        written++;
      }
    }
    tya_builder_append(builder, "}");
    break;
  case TYA_OBJECT: {
    TyaValue data_type = tya_member_optional(value, "__data_type");
    TyaValue class_name = tya_member_optional(value, "__class_name");
    if (class_name.kind != TYA_STRING) class_name = tya_member_optional(value, "class_name");
    const char *name = data_type.kind == TYA_STRING ? data_type.string : (class_name.kind == TYA_STRING ? class_name.string : "Object");
    tya_build_object_inspect(builder, value, name);
    break;
  }
  case TYA_NIL:
  case TYA_MISSING:
    tya_builder_append(builder, "nil");
    break;
  case TYA_BOOL:
    tya_builder_append(builder, value.boolean ? "true" : "false");
    break;
  case TYA_NUMBER:
    snprintf(scratch, sizeof(scratch), "%g", value.number);
    tya_builder_append(builder, scratch);
    break;
  default:
    tya_build_value(builder, value);
    break;
  }
}

static void tya_build_value(TyaStringBuilder *builder, TyaValue value) {
  char scratch[64];
  switch (value.kind) {
  case TYA_NIL:
  case TYA_MISSING:
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
    return tya_float(value.number);
  }
  if (value.kind == TYA_STRING && value.string != NULL) {
    return tya_float(strtod(value.string, NULL));
  }
  return tya_float(0);
}

TyaValue tya_to_number(TyaValue value) {
  return tya_to_float(value);
}

