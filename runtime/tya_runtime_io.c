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
  tya_set_member(out, "mode", tya_number((double)(st.st_mode & 0777)));
  return out;
}

TyaValue tya_dir_mkdir_all(TyaValue path) {
  if (path.kind != TYA_STRING || path.string == NULL) {
    tya_raise(tya_string("filesystem.mkdir_all: path must be a string"));
    return tya_nil();
  }
  char *tmp = tya_dup_cstr(path.string);
  size_t len = strlen(tmp);
  for (size_t i = 1; i <= len; i++) {
    if (tmp[i] == '/' || tmp[i] == '\0') {
      char saved = tmp[i];
      tmp[i] = '\0';
      if (tmp[0] != '\0' && mkdir(tmp, 0755) != 0 && errno != EEXIST) {
        free(tmp);
        tya_raise(tya_string("filesystem.mkdir_all: mkdir failed"));
        return tya_nil();
      }
      tmp[i] = saved;
    }
  }
  free(tmp);
  return tya_nil();
}

static bool tya_dangerous_path(const char *path) {
  return path == NULL || path[0] == '\0' || strcmp(path, ".") == 0 || strcmp(path, "/") == 0;
}

static void tya_remove_all_cstr(const char *path) {
  struct stat st;
  if (lstat(path, &st) != 0) {
    return;
  }
  if (S_ISDIR(st.st_mode)) {
    DIR *dir = opendir(path);
    if (dir != NULL) {
      struct dirent *entry;
      while ((entry = readdir(dir)) != NULL) {
        if (strcmp(entry->d_name, ".") == 0 || strcmp(entry->d_name, "..") == 0) continue;
        size_t n = strlen(path) + 1 + strlen(entry->d_name) + 1;
        char *child = malloc(n);
        snprintf(child, n, "%s/%s", path, entry->d_name);
        tya_remove_all_cstr(child);
        free(child);
      }
      closedir(dir);
    }
    rmdir(path);
  } else {
    unlink(path);
  }
}

TyaValue tya_dir_remove_all(TyaValue path) {
  if (path.kind != TYA_STRING || path.string == NULL) {
    tya_raise(tya_string("filesystem.remove_all: path must be a string"));
    return tya_nil();
  }
  if (tya_dangerous_path(path.string)) {
    tya_raise(tya_string("filesystem.remove_all: dangerous path"));
    return tya_nil();
  }
  tya_remove_all_cstr(path.string);
  return tya_nil();
}

TyaValue tya_dir_temp_dir(TyaValue prefix) {
  const char *p = (prefix.kind == TYA_STRING && prefix.string != NULL) ? prefix.string : "tya";
  char templ[512];
  snprintf(templ, sizeof(templ), "/tmp/%sXXXXXX", p);
  char *path = tya_dup_cstr(templ);
  if (mkdtemp(path) == NULL) {
    free(path);
    tya_raise(tya_string("filesystem.temp_dir: mkdtemp failed"));
    return tya_nil();
  }
  return tya_string(path);
}

static TyaValue tya_stat_dict_for_path(const char *path) {
  struct stat st;
  if (stat(path, &st) != 0) return tya_dict(NULL, 0);
  TyaValue out = tya_dict(NULL, 0);
  const char *kind = S_ISDIR(st.st_mode) ? "dir" : (S_ISREG(st.st_mode) ? "file" : "other");
  tya_set_member(out, "kind", tya_string(kind));
  tya_set_member(out, "size", tya_number((double)st.st_size));
  tya_set_member(out, "mode", tya_number((double)(st.st_mode & 0777)));
  return out;
}

static void tya_walk_collect(const char *root, TyaValue paths) {
  DIR *dir = opendir(root);
  if (dir == NULL) return;
  struct dirent *entry;
  while ((entry = readdir(dir)) != NULL) {
    if (strcmp(entry->d_name, ".") == 0 || strcmp(entry->d_name, "..") == 0) continue;
    size_t n = strlen(root) + 1 + strlen(entry->d_name) + 1;
    char *child = malloc(n);
    snprintf(child, n, "%s/%s", root, entry->d_name);
    tya_push(paths, tya_string(child));
    struct stat st;
    if (stat(child, &st) == 0 && S_ISDIR(st.st_mode)) {
      tya_walk_collect(child, paths);
    }
    free(child);
  }
  closedir(dir);
}

TyaValue tya_dir_walk(TyaValue path, TyaValue fn, TyaValue options) {
  (void)options;
  if (path.kind != TYA_STRING || path.string == NULL) {
    tya_raise(tya_string("filesystem.walk: path must be string"));
    return tya_nil();
  }
  TyaValue paths = tya_array(NULL, 0);
  tya_walk_collect(path.string, paths);
  paths = tya_array_sort(paths);
  for (int i = 0; paths.array != NULL && i < paths.array->len; i++) {
    TyaValue p = paths.array->items[i];
    struct stat st;
    if (p.kind != TYA_STRING || stat(p.string, &st) != 0) continue;
    TyaValue entry = tya_dict(NULL, 0);
    const char *slash = strrchr(p.string, '/');
    tya_set_member(entry, "path", p);
    tya_set_member(entry, "name", tya_string(slash == NULL ? p.string : slash + 1));
    tya_set_member(entry, "kind", tya_string(S_ISDIR(st.st_mode) ? "dir" : "file"));
    tya_set_member(entry, "stat", tya_stat_dict_for_path(p.string));
    tya_call1(fn, entry);
  }
  return tya_nil();
}

TyaValue tya_file_copy(TyaValue src, TyaValue dst, TyaValue options) {
  if (src.kind != TYA_STRING || src.string == NULL || dst.kind != TYA_STRING || dst.string == NULL) {
    tya_raise(tya_string("filesystem.copy: paths must be strings"));
    return tya_nil();
  }
  bool overwrite = true;
  if (options.kind == TYA_DICT && options.dict != NULL) {
    TyaValue ow = tya_index(options, tya_string("overwrite"));
    if (ow.kind == TYA_BOOL) overwrite = ow.boolean;
  }
  if (!overwrite && access(dst.string, F_OK) == 0) {
    tya_raise(tya_string("filesystem.copy: destination exists"));
    return tya_nil();
  }
  FILE *in = fopen(src.string, "rb");
  if (in == NULL) {
    tya_raise(tya_string("filesystem.copy: source open failed"));
    return tya_nil();
  }
  FILE *out = fopen(dst.string, "wb");
  if (out == NULL) {
    fclose(in);
    tya_raise(tya_string("filesystem.copy: destination open failed"));
    return tya_nil();
  }
  char buf[8192];
  size_t n;
  while ((n = fread(buf, 1, sizeof(buf), in)) > 0) {
    fwrite(buf, 1, n, out);
  }
  fclose(in);
  fclose(out);
  return tya_nil();
}

TyaValue tya_file_chmod(TyaValue path, TyaValue mode) {
  if (path.kind != TYA_STRING || path.string == NULL || mode.kind != TYA_NUMBER) {
    tya_raise(tya_string("filesystem.chmod: invalid arguments"));
    return tya_nil();
  }
  if (chmod(path.string, (mode_t)mode.number) != 0) {
    tya_raise(tya_string("filesystem.chmod: chmod failed"));
    return tya_nil();
  }
  return tya_nil();
}

TyaValue tya_file_temp(TyaValue prefix, TyaValue suffix) {
  const char *p = (prefix.kind == TYA_STRING && prefix.string != NULL) ? prefix.string : "tya";
  const char *s = (suffix.kind == TYA_STRING && suffix.string != NULL) ? suffix.string : "";
  char templ[512];
  snprintf(templ, sizeof(templ), "/tmp/%sXXXXXX%s", p, s);
  int fd = mkstemps(templ, (int)strlen(s));
  if (fd < 0) {
    tya_raise(tya_string("filesystem.temp: mkstemp failed"));
    return tya_nil();
  }
  close(fd);
  return tya_string(strdup(templ));
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

TyaValue tya_environ(void) {
  TyaValue out = tya_dict(NULL, 0);
  for (char **env = environ; env != NULL && *env != NULL; env++) {
    char *eq = strchr(*env, '=');
    if (eq == NULL) continue;
    size_t key_len = (size_t)(eq - *env);
    char *key = malloc(key_len + 1);
    memcpy(key, *env, key_len);
    key[key_len] = '\0';
    tya_set_member(out, key, tya_string(eq + 1));
    free(key);
  }
  return out;
}

TyaValue tya_setenv(TyaValue name, TyaValue value) {
  if (name.kind != TYA_STRING || name.string == NULL || value.kind != TYA_STRING || value.string == NULL) {
    tya_raise(tya_string("os.env: name and value must be strings"));
    return tya_nil();
  }
  if (setenv(name.string, value.string, 1) != 0) {
    tya_raise(tya_string("os.env: setenv failed"));
    return tya_nil();
  }
  return tya_nil();
}

TyaValue tya_unsetenv(TyaValue name) {
  if (name.kind != TYA_STRING || name.string == NULL) {
    tya_raise(tya_string("os.env: name must be a string"));
    return tya_nil();
  }
  if (unsetenv(name.string) != 0) {
    tya_raise(tya_string("os.env: unsetenv failed"));
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

TyaValue tya_array_contains(TyaValue array, TyaValue value) {
  if (array.kind != TYA_ARRAY || array.array == NULL) {
    return tya_bool(false);
  }
  for (int i = 0; i < array.array->len; i++) {
    if (tya_deep_equal_bool(array.array->items[i], value)) {
      return tya_bool(true);
    }
  }
  return tya_bool(false);
}

static int tya_value_compare(TyaValue left, TyaValue right) {
  if (left.kind == TYA_NUMBER && right.kind == TYA_NUMBER) {
    if (left.number < right.number) return -1;
    if (left.number > right.number) return 1;
    return 0;
  }
  if (left.kind == TYA_STRING && right.kind == TYA_STRING) {
    const char *l = left.string == NULL ? "" : left.string;
    const char *r = right.string == NULL ? "" : right.string;
    int cmp = strcmp(l, r);
    if (cmp < 0) return -1;
    if (cmp > 0) return 1;
    return 0;
  }
  if (left.kind == TYA_BOOL && right.kind == TYA_BOOL) {
    return (left.boolean > right.boolean) - (left.boolean < right.boolean);
  }
  if (left.kind == TYA_NIL && right.kind == TYA_NIL) {
    return 0;
  }
  TyaValue l = tya_to_string(left);
  TyaValue r = tya_to_string(right);
  const char *ls = l.string == NULL ? "" : l.string;
  const char *rs = r.string == NULL ? "" : r.string;
  int cmp = strcmp(ls, rs);
  if (cmp < 0) return -1;
  if (cmp > 0) return 1;
  return 0;
}

static int tya_sort_item_compare(const void *a, const void *b) {
  const TyaValue *left = (const TyaValue *)a;
  const TyaValue *right = (const TyaValue *)b;
  return tya_value_compare(*left, *right);
}

typedef struct {
  TyaValue item;
  TyaValue key;
} TyaSortPair;

static int tya_sort_pair_compare(const void *a, const void *b) {
  const TyaSortPair *left = (const TyaSortPair *)a;
  const TyaSortPair *right = (const TyaSortPair *)b;
  return tya_value_compare(left->key, right->key);
}

TyaValue tya_array_sort(TyaValue array) {
  if (array.kind != TYA_ARRAY || array.array == NULL) {
    return tya_array(NULL, 0);
  }
  TyaValue out = tya_array(NULL, 0);
  for (int i = 0; i < array.array->len; i++) {
    tya_push(out, array.array->items[i]);
  }
  qsort(out.array->items, (size_t)out.array->len, sizeof(TyaValue), tya_sort_item_compare);
  return out;
}

TyaValue tya_array_sort_by(TyaValue array, TyaValue fn) {
  if (array.kind != TYA_ARRAY || array.array == NULL || fn.kind != TYA_FUNCTION) {
    return tya_array(NULL, 0);
  }
  int n = array.array->len;
  TyaSortPair *pairs = malloc(sizeof(TyaSortPair) * (size_t)n);
  if (pairs == NULL) {
    return tya_array(NULL, 0);
  }
  for (int i = 0; i < n; i++) {
    pairs[i].item = array.array->items[i];
    pairs[i].key = tya_call1(fn, array.array->items[i]);
  }
  qsort(pairs, (size_t)n, sizeof(TyaSortPair), tya_sort_pair_compare);
  TyaValue out = tya_array(NULL, 0);
  for (int i = 0; i < n; i++) {
    tya_push(out, pairs[i].item);
  }
  free(pairs);
  return out;
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
  if (value.kind == TYA_NIL) {
    fprintf(stderr, "raise expects non-nil value\n");
    exit(1);
  }
  if (tya_raise_frame == NULL) {
    TyaValue text = tya_to_string(value);
    fprintf(stderr, "uncaught raised value: %s\n", text.string == NULL ? "" : text.string);
    exit(1);
  }
  tya_raise_frame->value = value;
  longjmp(tya_raise_frame->env, 1);
}

void tya_raise_user(TyaValue value) {
  if (value.kind != TYA_ERROR) {
    fprintf(stderr, "TYA-E0900: raise expects error value\n");
    exit(1);
  }
  tya_raise(value);
}

void tya_print(TyaValue value) {
  TyaValue text = tya_to_string(value);
  fprintf(stdout, "%s", text.string == NULL ? "" : text.string);
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
  case TYA_MISSING:
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
    fprintf(out, "%s", value.error == NULL ? "" : value.error);
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
  if (value.kind == TYA_NIL || value.kind == TYA_MISSING) {
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

static void tya_time_raise(const char *message, const char *code) {
  TyaDictEntry entries[] = {
    {"kind", tya_string("time")},
    {"code", tya_string(code)},
  };
  tya_raise_user(tya_error2(tya_string(message), tya_dict(entries, 2)));
}

static double tya_time_value_seconds(TyaValue value, bool *monotonic, bool *ok) {
  if (value.kind == TYA_NUMBER) {
    if (monotonic) *monotonic = false;
    if (ok) *ok = true;
    return value.number;
  }
  if ((value.kind == TYA_OBJECT || value.kind == TYA_DICT) && value.dict != NULL) {
    TyaValue seconds = tya_member(value, "__time_seconds");
    if (seconds.kind == TYA_NUMBER) {
      TyaValue mono = tya_member(value, "__time_monotonic");
      if (monotonic) *monotonic = mono.kind == TYA_BOOL && mono.boolean;
      if (ok) *ok = true;
      return seconds.number;
    }
  }
  if (ok) *ok = false;
  return 0.0;
}

static double tya_duration_value_seconds(TyaValue value, bool *ok) {
  if (value.kind == TYA_NUMBER) {
    if (ok) *ok = true;
    return value.number;
  }
  if ((value.kind == TYA_OBJECT || value.kind == TYA_DICT) && value.dict != NULL) {
    TyaValue seconds = tya_member(value, "__duration_seconds");
    if (seconds.kind == TYA_NUMBER) {
      if (ok) *ok = true;
      return seconds.number;
    }
  }
  if (ok) *ok = false;
  return 0.0;
}

static TyaValue tya_time_object(double seconds, bool monotonic, bool local);
static TyaValue tya_duration_object(double seconds);

static TyaValue tya_time_method_unix(TyaValue self, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)a; (void)b; (void)c; (void)d; (void)e; (void)f;
  bool ok = false;
  double seconds = tya_time_value_seconds(self, NULL, &ok);
  return ok ? tya_number(floor(seconds)) : tya_nil();
}

static TyaValue tya_time_method_unix_nanos(TyaValue self, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)a; (void)b; (void)c; (void)d; (void)e; (void)f;
  bool ok = false;
  double seconds = tya_time_value_seconds(self, NULL, &ok);
  return ok ? tya_number(seconds * 1.0e9) : tya_nil();
}

static TyaValue tya_time_method_utc(TyaValue self, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)a; (void)b; (void)c; (void)d; (void)e; (void)f;
  bool ok = false, mono = false;
  double seconds = tya_time_value_seconds(self, &mono, &ok);
  return ok ? tya_time_object(seconds, mono, false) : tya_nil();
}

static TyaValue tya_time_method_local(TyaValue self, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)a; (void)b; (void)c; (void)d; (void)e; (void)f;
  bool ok = false, mono = false;
  double seconds = tya_time_value_seconds(self, &mono, &ok);
  return ok ? tya_time_object(seconds, mono, true) : tya_nil();
}

static TyaValue tya_time_method_format(TyaValue self, TyaValue layout, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)b; (void)c; (void)d; (void)e; (void)f;
  return tya_time_format(self, layout, true);
}

static TyaValue tya_time_method_add(TyaValue self, TyaValue duration, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)b; (void)c; (void)d; (void)e; (void)f;
  bool ok = false, mono = false, dok = false;
  double seconds = tya_time_value_seconds(self, &mono, &ok);
  double delta = tya_duration_value_seconds(duration, &dok);
  if (!ok || !dok) {
    tya_time_raise("time.add: duration must be a duration", "invalid_duration");
    return tya_nil();
  }
  TyaValue local = tya_member(self, "__time_local");
  return tya_time_object(seconds + delta, mono, local.kind == TYA_BOOL && local.boolean);
}

static TyaValue tya_time_method_sub(TyaValue self, TyaValue other, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)b; (void)c; (void)d; (void)e; (void)f;
  bool ok = false, ook = false;
  double seconds = tya_time_value_seconds(self, NULL, &ok);
  double rhs = tya_time_value_seconds(other, NULL, &ook);
  if (!ok || !ook) {
    tya_time_raise("time.sub: other must be a time", "invalid_time");
    return tya_nil();
  }
  return tya_duration_object(seconds - rhs);
}

static TyaValue tya_duration_method_seconds(TyaValue self, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)a; (void)b; (void)c; (void)d; (void)e; (void)f;
  bool ok = false;
  double seconds = tya_duration_value_seconds(self, &ok);
  return ok ? tya_number(seconds) : tya_nil();
}

static TyaValue tya_duration_method_milliseconds(TyaValue self, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)a; (void)b; (void)c; (void)d; (void)e; (void)f;
  bool ok = false;
  double seconds = tya_duration_value_seconds(self, &ok);
  return ok ? tya_number(seconds * 1.0e3) : tya_nil();
}

static TyaValue tya_duration_method_microseconds(TyaValue self, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)a; (void)b; (void)c; (void)d; (void)e; (void)f;
  bool ok = false;
  double seconds = tya_duration_value_seconds(self, &ok);
  return ok ? tya_number(seconds * 1.0e6) : tya_nil();
}

static TyaValue tya_duration_method_nanoseconds(TyaValue self, TyaValue a, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)a; (void)b; (void)c; (void)d; (void)e; (void)f;
  bool ok = false;
  double seconds = tya_duration_value_seconds(self, &ok);
  return ok ? tya_number(seconds * 1.0e9) : tya_nil();
}

static TyaValue tya_duration_method_add(TyaValue self, TyaValue other, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)b; (void)c; (void)d; (void)e; (void)f;
  bool ok = false, ook = false;
  double seconds = tya_duration_value_seconds(self, &ok);
  double rhs = tya_duration_value_seconds(other, &ook);
  if (!ok || !ook) return tya_nil();
  return tya_duration_object(seconds + rhs);
}

static TyaValue tya_duration_method_sub(TyaValue self, TyaValue other, TyaValue b, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)b; (void)c; (void)d; (void)e; (void)f;
  bool ok = false, ook = false;
  double seconds = tya_duration_value_seconds(self, &ok);
  double rhs = tya_duration_value_seconds(other, &ook);
  if (!ok || !ook) return tya_nil();
  return tya_duration_object(seconds - rhs);
}

static TyaValue tya_time_object(double seconds, bool monotonic, bool local) {
  TyaValue obj = tya_object();
  tya_set_member(obj, "__time_seconds", tya_number(seconds));
  tya_set_member(obj, "__time_monotonic", tya_bool(monotonic));
  tya_set_member(obj, "__time_local", tya_bool(local));
  tya_set_member(obj, "unix", tya_bind_method(obj, tya_time_method_unix));
  tya_set_member(obj, "unix_nanos", tya_bind_method(obj, tya_time_method_unix_nanos));
  tya_set_member(obj, "utc", tya_bind_method(obj, tya_time_method_utc));
  tya_set_member(obj, "local", tya_bind_method(obj, tya_time_method_local));
  tya_set_member(obj, "format", tya_bind_method(obj, tya_time_method_format));
  tya_set_member(obj, "add", tya_bind_method(obj, tya_time_method_add));
  tya_set_member(obj, "sub", tya_bind_method(obj, tya_time_method_sub));
  return obj;
}

static TyaValue tya_duration_object(double seconds) {
  TyaValue obj = tya_object();
  tya_set_member(obj, "__duration_seconds", tya_number(seconds));
  tya_set_member(obj, "seconds", tya_bind_method(obj, tya_duration_method_seconds));
  tya_set_member(obj, "milliseconds", tya_bind_method(obj, tya_duration_method_milliseconds));
  tya_set_member(obj, "microseconds", tya_bind_method(obj, tya_duration_method_microseconds));
  tya_set_member(obj, "nanoseconds", tya_bind_method(obj, tya_duration_method_nanoseconds));
  tya_set_member(obj, "add", tya_bind_method(obj, tya_duration_method_add));
  tya_set_member(obj, "sub", tya_bind_method(obj, tya_duration_method_sub));
  return obj;
}

TyaValue tya_time_now(void) {
  struct timeval tv;
  gettimeofday(&tv, NULL);
  return tya_time_object((double)tv.tv_sec + (double)tv.tv_usec / 1.0e6, false, false);
}

TyaValue tya_time_monotonic(void) {
  return tya_time_object(tya_now_seconds(), true, false);
}

TyaValue tya_time_unix(TyaValue seconds, TyaValue nanos) {
  if (nanos.kind == TYA_MISSING || nanos.kind == TYA_NIL) nanos = tya_number(0);
  if (seconds.kind != TYA_NUMBER || nanos.kind != TYA_NUMBER || floor(nanos.number) != nanos.number) {
    tya_time_raise("time.unix: seconds must be a number and nanos must be an integer", "invalid_seconds");
    return tya_nil();
  }
  return tya_time_object(seconds.number + nanos.number / 1.0e9, false, false);
}

TyaValue tya_time_duration(TyaValue seconds, TyaValue options) {
  if (seconds.kind != TYA_NUMBER) {
    tya_time_raise("time.duration: seconds must be a number", "invalid_seconds");
    return tya_nil();
  }
  if (options.kind == TYA_MISSING || options.kind == TYA_NIL) {
    return tya_duration_object(seconds.number);
  }
  if (options.kind != TYA_DICT || options.dict == NULL) {
    tya_time_raise("time.duration: options must be a dictionary", "invalid_options");
    return tya_nil();
  }
  double total = seconds.number;
  for (int i = 0; i < options.dict->len; i++) {
    const char *key = options.dict->entries[i].key;
    TyaValue value = options.dict->entries[i].value;
    if (key == NULL) continue;
    if (value.kind != TYA_NUMBER) {
      tya_time_raise("time.duration: option must be a number", "invalid_option");
      return tya_nil();
    }
    if (strcmp(key, "minutes") == 0) total += value.number * 60.0;
    else if (strcmp(key, "hours") == 0) total += value.number * 3600.0;
    else if (strcmp(key, "milliseconds") == 0) total += value.number / 1.0e3;
    else if (strcmp(key, "microseconds") == 0) total += value.number / 1.0e6;
    else if (strcmp(key, "nanoseconds") == 0) total += value.number / 1.0e9;
    else {
      tya_time_raise("time.duration: unknown option", "unknown_option");
      return tya_nil();
    }
  }
  return tya_duration_object(total);
}

TyaValue tya_time_sleep(TyaValue seconds) {
  bool ok = false;
  double sleep_seconds = tya_duration_value_seconds(seconds, &ok);
  if (!ok) {
    tya_time_raise("time.sleep: argument must be a duration or number", "invalid_duration");
    return tya_nil();
  }
  if (sleep_seconds < 0) {
    tya_time_raise("time.sleep: negative duration", "negative_duration");
    return tya_nil();
  }
  if (tya_current_task_ptr == NULL) {
    double deadline = tya_now_seconds() + sleep_seconds;
    while (tya_now_seconds() < deadline) {
      tya_task_wake_sleepers();
      if (tya_ready_head != NULL) {
        tya_scheduler_run_one();
        continue;
      }
      double delay = deadline - tya_now_seconds();
      if (tya_sleep_head != NULL && tya_sleep_head->wake_time < deadline) {
        delay = tya_sleep_head->wake_time - tya_now_seconds();
      }
      if (delay <= 0.0) continue;
      struct timespec req;
      req.tv_sec = (time_t)floor(delay);
      req.tv_nsec = (long)((delay - floor(delay)) * 1.0e9);
      nanosleep(&req, NULL);
    }
    return tya_nil();
  } else {
    tya_task_sleep_until(tya_current_task_ptr, tya_now_seconds() + sleep_seconds);
    tya_task_yield(false);
    return tya_nil();
  }
  return tya_nil();
}

TyaValue tya_time_format(TyaValue t, TyaValue layout, bool has_layout) {
  bool ok = false, monotonic = false;
  double seconds = tya_time_value_seconds(t, &monotonic, &ok);
  if (!ok) {
    tya_time_raise("time.format: argument must be a time", "invalid_time");
    return tya_nil();
  }
  if (monotonic) {
    tya_time_raise("time.format: monotonic time cannot be formatted", "monotonic_format");
    return tya_nil();
  }
  const char *layout_name = "rfc3339";
  if (has_layout) {
    if (layout.kind != TYA_STRING || layout.string == NULL) {
      tya_time_raise("time.format: layout must be a string", "invalid_layout");
      return tya_nil();
    }
    layout_name = layout.string;
  }
  if (strcmp(layout_name, "unix") == 0) {
    char buf[32];
    snprintf(buf, sizeof(buf), "%ld", (long)seconds);
    char *out = malloc(strlen(buf) + 1);
    strcpy(out, buf);
    return tya_string(out);
  }
  time_t tt = (time_t)seconds;
  struct tm gm;
  TyaValue local = tya_member(t, "__time_local");
  if (local.kind == TYA_BOOL && local.boolean) {
    localtime_r(&tt, &gm);
  } else {
    gmtime_r(&tt, &gm);
  }
  char buf[64];
  if (strcmp(layout_name, "iso") == 0 || strcmp(layout_name, "rfc3339") == 0) {
    strftime(buf, sizeof(buf), "%Y-%m-%dT%H:%M:%SZ", &gm);
  } else if (strcmp(layout_name, "date") == 0) {
    strftime(buf, sizeof(buf), "%Y-%m-%d", &gm);
  } else if (strcmp(layout_name, "time") == 0) {
    strftime(buf, sizeof(buf), "%H:%M:%S", &gm);
  } else {
    tya_time_raise("time.format: unknown layout", "unknown_layout");
    return tya_nil();
  }
  char *out = malloc(strlen(buf) + 1);
  strcpy(out, buf);
  return tya_string(out);
}

TyaValue tya_time_parse(TyaValue text, TyaValue layout, bool has_layout) {
  if (text.kind != TYA_STRING || text.string == NULL) {
    tya_time_raise("time.parse: text must be a string", "invalid_text");
    return tya_nil();
  }
  const char *layout_name = "rfc3339";
  if (has_layout) {
    if (layout.kind == TYA_MISSING || layout.kind == TYA_NIL) {
      layout_name = "rfc3339";
    } else
    if (layout.kind != TYA_STRING || layout.string == NULL) {
      tya_time_raise("time.parse: layout must be a string", "invalid_layout");
      return tya_nil();
    } else {
      layout_name = layout.string;
    }
  }
  struct tm tm;
  memset(&tm, 0, sizeof(tm));
  const char *s = text.string;
  const char *fmt = NULL;
  if (strcmp(layout_name, "rfc3339") == 0 || strcmp(layout_name, "iso") == 0) {
    fmt = "%Y-%m-%dT%H:%M:%SZ";
  } else if (strcmp(layout_name, "date") == 0) {
    fmt = "%Y-%m-%d";
  } else if (strcmp(layout_name, "unix") == 0) {
    char *end = NULL;
    long value = strtol(s, &end, 10);
    if (end == s || *end != '\0') {
      tya_time_raise("time.parse: invalid timestamp", "invalid_timestamp");
      return tya_nil();
    }
    return tya_time_object((double)value, false, false);
  } else {
    tya_time_raise("time.parse: unknown layout", "unknown_layout");
    return tya_nil();
  }
  if (strptime(s, fmt, &tm) == NULL) {
    tya_time_raise("time.parse: invalid timestamp", "invalid_timestamp");
    return tya_nil();
  }
  time_t tt = timegm(&tm);
  return tya_time_object((double)tt, false, false);
}

TyaValue tya_time_since(TyaValue t) {
  bool ok = false;
  double seconds = tya_time_value_seconds(t, NULL, &ok);
  if (!ok) {
    tya_time_raise("time.since: argument must be a time", "invalid_time");
    return tya_nil();
  }
  return tya_duration_object(tya_now_seconds() - seconds);
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

TyaValue tya_serialization_kind(TyaValue value) {
  switch (value.kind) {
  case TYA_NIL:
  case TYA_MISSING:
    return tya_string("nil");
  case TYA_BOOL: return tya_string("bool");
  case TYA_NUMBER: return tya_string("number");
  case TYA_STRING: return tya_string("string");
  case TYA_ARRAY: return tya_string("array");
  case TYA_DICT: return tya_string("dict");
  case TYA_OBJECT: return tya_string("object");
  case TYA_FUNCTION: return tya_string(value.function != NULL && value.function->is_class ? "class" : "function");
  case TYA_ERROR: return tya_string("error");
  case TYA_BYTES: return tya_string("bytes");
  case TYA_TASK: return tya_string("task");
  case TYA_CHANNEL: return tya_string("channel");
  case TYA_RESOURCE: return tya_string("resource");
  }
  return tya_string("unknown");
}

TyaValue tya_serialization_id(TyaValue value) {
  uintptr_t p = 0;
  switch (value.kind) {
  case TYA_ARRAY: p = (uintptr_t)value.array; break;
  case TYA_DICT:
  case TYA_OBJECT: p = (uintptr_t)value.dict; break;
  case TYA_FUNCTION: p = (uintptr_t)value.function; break;
  case TYA_BYTES: p = (uintptr_t)value.bytes; break;
  case TYA_TASK: p = (uintptr_t)value.task; break;
  case TYA_CHANNEL: p = (uintptr_t)value.channel; break;
  case TYA_RESOURCE: p = (uintptr_t)value.resource; break;
  default: p = 0; break;
  }
  return tya_number((double)p);
}

TyaValue tya_serialization_public_fields(TyaValue value) {
  TyaValue out = tya_dict(NULL, 0);
  if (value.kind != TYA_OBJECT || value.dict == NULL) {
    return out;
  }
  for (int i = 0; i < value.dict->len; i++) {
    const char *key = value.dict->entries[i].key;
    TyaValue field = value.dict->entries[i].value;
    if (key == NULL || key[0] == '@' || strcmp(key, "class") == 0 || strcmp(key, "class_name") == 0 || field.kind == TYA_FUNCTION) {
      continue;
    }
    tya_set_member(out, key, field);
  }
  return out;
}

TyaValue tya_serialization_has_member(TyaValue value, TyaValue key_value) {
  if (key_value.kind != TYA_STRING || key_value.string == NULL) {
    return tya_bool(false);
  }
  const char *key = key_value.string;
  if (value.kind == TYA_OBJECT && value.dict != NULL) {
    for (int i = 0; i < value.dict->len; i++) {
      if (value.dict->entries[i].key != NULL && strcmp(value.dict->entries[i].key, key) == 0) {
        return tya_bool(true);
      }
    }
  }
  if (value.kind == TYA_FUNCTION && value.function != NULL && value.function->members != NULL) {
    if (value.function->is_class && (strcmp(key, "name") == 0 || strcmp(key, "parent") == 0)) {
      return tya_bool(true);
    }
    for (int i = 0; i < value.function->members->len; i++) {
      if (value.function->members->entries[i].key != NULL && strcmp(value.function->members->entries[i].key, key) == 0) {
        return tya_bool(true);
      }
    }
  }
  return tya_bool(false);
}

