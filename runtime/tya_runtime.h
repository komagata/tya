#ifndef TYA_RUNTIME_H
#define TYA_RUNTIME_H

#include <stdbool.h>
#include <setjmp.h>

typedef enum {
  TYA_NIL,
  TYA_BOOL,
  TYA_NUMBER,
  TYA_STRING,
  TYA_ARRAY,
  TYA_DICT,
  TYA_OBJECT,
  TYA_FUNCTION,
  TYA_ERROR,
  TYA_BYTES,
  TYA_TASK,
  TYA_CHANNEL,
  TYA_RESOURCE,
} TyaKind;

typedef struct TyaBytes TyaBytes;

typedef struct TyaArray TyaArray;
typedef struct TyaDict TyaDict;
typedef struct TyaDictEntry TyaDictEntry;
typedef struct TyaFunction TyaFunction;
typedef struct TyaRaiseFrame TyaRaiseFrame;
typedef struct TyaTask TyaTask;
typedef struct TyaChannel TyaChannel;
typedef struct TyaResource TyaResource;

typedef struct {
  TyaKind kind;
  bool boolean;
  double number;
  const char *string;
  TyaArray *array;
  TyaDict *dict;
  TyaFunction *function;
  const char *error;
  TyaBytes *bytes;
  TyaTask *task;
  TyaChannel *channel;
  TyaResource *resource;
} TyaValue;

typedef TyaValue (*TyaFunctionPtr)(TyaValue, TyaValue, TyaValue, TyaValue, TyaValue, TyaValue, TyaValue);

struct TyaDictEntry {
  const char *key;
  TyaValue value;
};

struct TyaRaiseFrame {
  jmp_buf env;
  TyaValue value;
  TyaRaiseFrame *prev;
};

TyaValue tya_nil(void);
TyaValue tya_bool(bool value);
TyaValue tya_number(double value);
TyaValue tya_string(const char *value);
TyaValue tya_array(const TyaValue *items, int count);
TyaValue tya_dict(const TyaDictEntry *entries, int count);
TyaValue tya_object(void);
TyaValue tya_function_raw(TyaFunctionPtr fn);
TyaValue tya_class_raw(TyaFunctionPtr fn, const char *name, TyaValue parent);
TyaValue tya_primitive_class(const char *name);
TyaValue tya_class_of(TyaValue value);
TyaValue tya_bind_method_raw(TyaValue receiver, TyaFunctionPtr fn);
#define tya_function(fn) tya_function_raw((TyaFunctionPtr)(fn))
#define tya_class(fn, name, parent) tya_class_raw((TyaFunctionPtr)(fn), name, parent)
#define tya_bind_method(receiver, fn) tya_bind_method_raw(receiver, (TyaFunctionPtr)(fn))
TyaValue tya_error(TyaValue message);
TyaValue tya_call1(TyaValue fn, TyaValue arg);
TyaValue tya_call2(TyaValue fn, TyaValue first, TyaValue second);
TyaValue tya_call3(TyaValue fn, TyaValue first, TyaValue second, TyaValue third);
TyaValue tya_call4(TyaValue fn, TyaValue first, TyaValue second, TyaValue third, TyaValue fourth);
TyaValue tya_call5(TyaValue fn, TyaValue first, TyaValue second, TyaValue third, TyaValue fourth, TyaValue fifth);
TyaValue tya_call6(TyaValue fn, TyaValue first, TyaValue second, TyaValue third, TyaValue fourth, TyaValue fifth, TyaValue sixth);
TyaValue tya_len(TyaValue value);
TyaValue tya_index(TyaValue value, TyaValue index);
TyaValue tya_destructure_array(TyaValue value, int expected, int index);
TyaValue tya_destructure_dict(TyaValue value, const char *key);
void tya_set_index(TyaValue value, TyaValue index, TyaValue item);
TyaValue tya_member(TyaValue dict, const char *key);
void tya_set_member(TyaValue dict, const char *key, TyaValue value);
TyaValue tya_dict_key_at(TyaValue dict, TyaValue index);
TyaValue tya_dict_value_at(TyaValue dict, TyaValue index);
TyaValue tya_has(TyaValue dict, TyaValue key);
TyaValue tya_keys(TyaValue dict);
TyaValue tya_values(TyaValue dict);
void tya_delete(TyaValue dict, TyaValue key);
TyaValue tya_dict_get(TyaValue dict, TyaValue key, TyaValue fallback, bool has_fallback);
TyaValue tya_dict_set(TyaValue dict, TyaValue key, TyaValue value);
TyaValue tya_dict_delete(TyaValue dict, TyaValue key);
TyaValue tya_dict_merge(TyaValue left, TyaValue right);
TyaValue tya_dict_entries(TyaValue dict);
TyaValue tya_contains(TyaValue text, TyaValue part);
TyaValue tya_contains_method(TyaValue receiver, TyaValue value);
TyaValue tya_starts_with(TyaValue text, TyaValue prefix);
TyaValue tya_ends_with(TyaValue text, TyaValue suffix);
TyaValue tya_trim(TyaValue text);
TyaValue tya_replace(TyaValue text, TyaValue old, TyaValue replacement);
TyaValue tya_byte_len(TyaValue text);
TyaValue tya_ord(TyaValue text);
TyaValue tya_chr(TyaValue code);
TyaValue tya_kind(TyaValue value);
TyaValue tya_lines(TyaValue text);
TyaValue tya_chars(TyaValue text);
TyaValue tya_upcase(TyaValue text);
TyaValue tya_downcase(TyaValue text);
bool tya_equal(TyaValue left, TyaValue right);
TyaValue tya_deep_equal(TyaValue left, TyaValue right);
TyaValue tya_add(TyaValue left, TyaValue right);
TyaValue tya_and(TyaValue left, TyaValue right);
TyaValue tya_or(TyaValue left, TyaValue right);
TyaValue tya_args(int argc, char **argv);
TyaValue tya_env(TyaValue name);
TyaValue tya_read_file(TyaValue path);
void tya_write_file(TyaValue path, TyaValue text);
TyaValue tya_split(TyaValue text, TyaValue sep);
TyaValue tya_join(TyaValue array, TyaValue sep);
TyaValue tya_to_string(TyaValue value);
TyaValue tya_to_int(TyaValue value);
TyaValue tya_to_float(TyaValue value);
TyaValue tya_to_number(TyaValue value);
TyaValue tya_file_exists(TyaValue path);
TyaValue tya_dir_list(TyaValue path);
TyaValue tya_dir_mkdir(TyaValue path);
TyaValue tya_dir_rmdir(TyaValue path);
TyaValue tya_file_remove(TyaValue path);
TyaValue tya_file_rename(TyaValue old_path, TyaValue new_path);
TyaValue tya_file_stat(TyaValue path);
TyaValue tya_path_expand_user(TyaValue value);
TyaValue tya_cwd(void);
TyaValue tya_chdir(TyaValue path);

TyaValue tya_time_now(void);
TyaValue tya_time_sleep(TyaValue seconds);
TyaValue tya_time_format(TyaValue t, TyaValue layout, bool has_layout);
TyaValue tya_time_parse(TyaValue text);
TyaValue tya_time_since(TyaValue t);

TyaValue tya_random_seed(TyaValue value);
TyaValue tya_random_int(TyaValue min, TyaValue max);
TyaValue tya_random_float(void);
TyaValue tya_serialization_kind(TyaValue value);
TyaValue tya_serialization_id(TyaValue value);
TyaValue tya_serialization_public_fields(TyaValue value);
TyaValue tya_serialization_has_member(TyaValue value, TyaValue key);

TyaValue tya_compiler_lexer_lex(TyaValue source);
TyaValue tya_compiler_lexer_lex_with_comments(TyaValue source);
TyaValue tya_compiler_parser_parse(TyaValue source);
TyaValue tya_compiler_parser_parse_tokens(TyaValue tokens);
TyaValue tya_compiler_ast_children(TyaValue node);
TyaValue tya_compiler_ast_kind(TyaValue node);
TyaValue tya_compiler_ast_span(TyaValue node);
TyaValue tya_compiler_checker_check(TyaValue source);
TyaValue tya_compiler_checker_check_ast(TyaValue program);
TyaValue tya_compiler_format_format(TyaValue source);
TyaValue tya_compiler_format_unparse(TyaValue program);

TyaValue tya_math_sqrt(TyaValue x);
TyaValue tya_math_pow(TyaValue x, TyaValue y);
TyaValue tya_math_floor(TyaValue x);
TyaValue tya_math_ceil(TyaValue x);
TyaValue tya_math_round(TyaValue x);
TyaValue tya_math_trunc(TyaValue x);
TyaValue tya_math_log(TyaValue x);
TyaValue tya_math_log2(TyaValue x);
TyaValue tya_math_log10(TyaValue x);
TyaValue tya_math_exp(TyaValue x);
TyaValue tya_math_sin(TyaValue x);
TyaValue tya_math_cos(TyaValue x);
TyaValue tya_math_tan(TyaValue x);
TyaValue tya_math_asin(TyaValue x);
TyaValue tya_math_acos(TyaValue x);
TyaValue tya_math_atan(TyaValue x);
TyaValue tya_math_atan2(TyaValue y, TyaValue x);
TyaValue tya_number_integer_p(TyaValue x);
TyaValue tya_number_finite_p(TyaValue x);
TyaValue tya_number_nan_p(TyaValue x);

TyaValue tya_process_run(TyaValue command, TyaValue options);

TyaValue tya_digest_md5(TyaValue text);
TyaValue tya_digest_sha1(TyaValue text);
TyaValue tya_digest_sha256(TyaValue text);
TyaValue tya_digest_sha384(TyaValue text);
TyaValue tya_digest_sha512(TyaValue text);

TyaValue tya_secure_random_bytes(TyaValue n);
TyaValue tya_secure_random_int(TyaValue min, TyaValue max);

/* Concurrency API (v0.42).
 *
 * tya_task_new      creates a task, allocates a TyaTask, spawns a
 *                   pthread that runs the callable with the given
 *                   arguments (0 to 4), and returns a TyaValue of
 *                   kind TYA_TASK. The arguments are evaluated in
 *                   the spawning thread before the new pthread
 *                   starts.
 * tya_task_await    blocks the current thread until the task
 *                   completes, then returns the task's return value
 *                   or re-raises the propagated raise.
 * tya_scope_enter / tya_scope_exit
 *                   structured-concurrency block. tya_scope_enter
 *                   pushes a fresh scope onto the thread-local scope
 *                   stack so that subsequent tya_task_new calls
 *                   register the new task in that scope.
 *                   tya_scope_exit joins every task in the scope
 *                   (in spawn order) before returning, and re-raises
 *                   the first raise it observes if any. */
typedef struct TyaScope {
  struct TyaTask **tasks;
  int len;
  int cap;
  struct TyaScope *prev;
} TyaScope;

TyaValue tya_task_new(TyaValue callee, int argc, TyaValue a, TyaValue b, TyaValue c, TyaValue d);
TyaValue tya_task_await(TyaValue task);
void tya_task_run_ready(void);
bool tya_task_has_ready(void);
double tya_task_next_wake_delay(double max_seconds);
/* Cooperative cancellation (v0.43). The cancel flag is set to true;
 * worker code is expected to poll task_is_cancelled at safe points and
 * return early. tya_scope_exit also sets the cancel flag on every
 * sibling when one task raises. */
TyaValue tya_task_cancel(TyaValue task);
TyaValue tya_task_is_cancelled(TyaValue task);
/* tya_current_task returns the task value of the currently-running
 * task (or nil for the main thread). Worker bodies use it to poll
 * task.is_cancelled() on themselves. */
TyaValue tya_current_task(void);
void tya_scope_enter(TyaScope *scope);
void tya_scope_exit(TyaScope *scope);
/* tya_scope_raise is invoked by codegen when control unwinds out of
 * a scope body via raise. It joins outstanding tasks first and then
 * re-raises the original raise value, taking precedence over any
 * subsequent task raise. */
void tya_scope_raise(TyaScope *scope, TyaValue value);

/* Channel API (v0.42 STEP 5).
 *
 * tya_channel_new       creates a channel with the given buffer size.
 *                       Capacity of 0 is treated as 1 in v0.42; true
 *                       rendezvous arrives in a later minor.
 * tya_channel_send      blocks the caller while the buffer is full,
 *                       then enqueues the value. Raises if the channel
 *                       has been closed.
 * tya_channel_receive   blocks the caller while the buffer is empty,
 *                       then dequeues. After close, drains the buffer
 *                       and then returns nil for every later call.
 * tya_channel_close     marks the channel closed and wakes everyone
 *                       waiting on it.
 * tya_channel_closed    predicate for the closed flag. */
TyaValue tya_channel_new(TyaValue capacity);
TyaValue tya_channel_send(TyaValue ch, TyaValue value);
TyaValue tya_channel_receive(TyaValue ch);
TyaValue tya_channel_receive_timeout(TyaValue ch, TyaValue seconds);
TyaValue tya_channel_close(TyaValue ch);
TyaValue tya_channel_closed(TyaValue ch);
/* tya_channel_select runs a multi-channel wait. ops is an array of
 * arrays: each inner array is [channel, "receive"] or
 * [channel, "send", value]. Returns a dict {index, kind, value} for
 * the first ready operation. The runtime scheduler runs ready tasks
 * while a select is waiting so channel wakeups can make progress
 * without creating one OS thread per waiting task. */
TyaValue tya_channel_select(TyaValue ops);

/* Synchronization primitives (v0.42 STEP 7).
 *
 * Mutex:           tya_sync_mutex_new returns an unlocked mutex
 *                  resource; tya_sync_lock blocks until the mutex
 *                  is acquired; tya_sync_unlock releases it.
 * Atomic integer:  tya_sync_atomic_integer_new wraps an int64
 *                  counter; add / load / store / compare-and-swap
 *                  use stdatomic.
 * Wait group:      tya_sync_wait_group_new wraps a counter +
 *                  mutex + condvar so workers can announce that
 *                  they finished and a coordinator can block
 *                  until everyone has done so. */
TyaValue tya_sync_mutex_new(void);
TyaValue tya_sync_lock(TyaValue m);
TyaValue tya_sync_unlock(TyaValue m);
TyaValue tya_sync_atomic_integer_new(TyaValue initial);
TyaValue tya_sync_atomic_integer_add(TyaValue a, TyaValue n);
TyaValue tya_sync_atomic_integer_load(TyaValue a);
TyaValue tya_sync_atomic_integer_store(TyaValue a, TyaValue n);
TyaValue tya_sync_atomic_integer_cas(TyaValue a, TyaValue expected, TyaValue new_value);
TyaValue tya_sync_wait_group_new(void);
TyaValue tya_sync_wait_group_add(TyaValue wg, TyaValue n);
TyaValue tya_sync_wait_group_done(TyaValue wg);
TyaValue tya_sync_wait_group_wait(TyaValue wg);

/* GC API (v0.41).
 *
 * tya_gc_register_root  generated code calls this at startup for every
 *                       module-level TyaValue global so the collector can
 *                       reach them as roots.
 * tya_gc_collect        runs a full mark-and-sweep collection. Safe only
 *                       at points where every live local TyaValue is also
 *                       reachable from a registered root (e.g. between
 *                       top-level statements). See docs/v0.41/SPEC.md.
 * tya_gc_maybe_collect  threshold-driven trigger emitted by generated
 *                       code at safe points; calls tya_gc_collect when
 *                       the live set has grown past the threshold.
 * tya_gc_stats          returns a dict snapshot of the GC counters. */
void tya_gc_register_root(TyaValue *p);
void tya_gc_collect(void);
void tya_gc_maybe_collect(void);
TyaValue tya_gc_stats(void);

TyaValue tya_bytes_lit(const char *data, int len);
TyaValue tya_bytes_from_array(TyaValue arr);
TyaValue tya_bytes_of(TyaValue text);
TyaValue tya_bytes_text(TyaValue b);
TyaValue tya_bytes_array(TyaValue b);
TyaValue tya_bytes_concat(TyaValue a, TyaValue b);
TyaValue tya_bytes_slice(TyaValue b, TyaValue start, TyaValue end);
TyaValue tya_file_read_bytes(TyaValue path);
TyaValue tya_file_write_bytes(TyaValue path, TyaValue b);
TyaValue tya_binary_read_f32(TyaValue b, TyaValue offset, TyaValue endian);
TyaValue tya_binary_read_f64(TyaValue b, TyaValue offset, TyaValue endian);
TyaValue tya_binary_write_f32(TyaValue value, TyaValue endian);
TyaValue tya_binary_write_f64(TyaValue value, TyaValue endian);
TyaValue tya_stderr_write(TyaValue text);
TyaValue tya_file_append(TyaValue path, TyaValue text);
TyaValue tya_compress_gzip(TyaValue value);
TyaValue tya_compress_gunzip(TyaValue value);
TyaValue tya_compress_zlib(TyaValue value);
TyaValue tya_compress_unzlib(TyaValue value);
TyaValue tya_io_stdin(void);
TyaValue tya_io_stdout(void);
TyaValue tya_io_stderr(void);
TyaValue tya_io_open(TyaValue path, TyaValue mode);
TyaValue tya_io_stream_read(TyaValue stream, TyaValue size);
TyaValue tya_io_stream_read_line(TyaValue stream);
TyaValue tya_io_stream_eof(TyaValue stream);
TyaValue tya_io_stream_write(TyaValue stream, TyaValue value);
TyaValue tya_io_stream_flush(TyaValue stream);
TyaValue tya_io_stream_close(TyaValue stream);
TyaValue tya_socket_connect(TyaValue host, TyaValue port, TyaValue options);
TyaValue tya_socket_server_listen(TyaValue host, TyaValue port, TyaValue options);
TyaValue tya_socket_server_accept(TyaValue server);
TyaValue tya_socket_read(TyaValue socket, TyaValue size);
TyaValue tya_socket_read_line(TyaValue socket);
TyaValue tya_socket_write(TyaValue socket, TyaValue value);
TyaValue tya_socket_close(TyaValue socket);
TyaValue tya_socket_closed(TyaValue socket);
TyaValue tya_socket_local_address(TyaValue socket);
TyaValue tya_socket_remote_address(TyaValue socket);
TyaValue tya_socket_server_close(TyaValue server);
TyaValue tya_socket_server_local_address(TyaValue server);
TyaValue tya_read_line(void);
TyaValue tya_map(TyaValue array, TyaValue fn);
TyaValue tya_filter(TyaValue array, TyaValue fn);
TyaValue tya_find(TyaValue array, TyaValue fn);
TyaValue tya_any(TyaValue array, TyaValue fn);
TyaValue tya_all(TyaValue array, TyaValue fn);
TyaValue tya_each(TyaValue array, TyaValue fn);
TyaValue tya_reduce(TyaValue array, TyaValue initial, TyaValue fn);
TyaValue tya_array_contains(TyaValue array, TyaValue value);
TyaValue tya_array_sort(TyaValue array);
TyaValue tya_array_sort_by(TyaValue array, TyaValue fn);
void tya_push(TyaValue array, TyaValue value);
TyaValue tya_array_push(TyaValue array, TyaValue value);
TyaValue tya_pop(TyaValue array);
TyaValue tya_first(TyaValue array);
TyaValue tya_last(TyaValue array);
TyaValue tya_slice(TyaValue array, TyaValue start, TyaValue end);
TyaValue tya_reverse(TyaValue array);
void tya_exit(TyaValue code);
void tya_panic(TyaValue message);
void tya_push_raise_frame(TyaRaiseFrame *frame);
void tya_pop_raise_frame(void);
TyaValue tya_current_raise(void);
void tya_raise(TyaValue value);
void tya_print(TyaValue value);
void tya_assert(TyaValue value, const char *path, int line);
void tya_assert_equal(TyaValue expected, TyaValue actual, const char *path, int line);
bool tya_truthy(TyaValue value);

// v0.58 net/http server. Defined in runtime/tya_http_server.c.
// `routes` is an array of dicts {method, path, handler}.
TyaValue tya_http_server_run(TyaValue routes, TyaValue port);

#endif
