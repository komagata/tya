#ifdef __clang__
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
#endif

#include "tya_http_server.h"

#include <errno.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#ifdef TYA_ENABLE_OPENSSL
#include <openssl/err.h>
#include <openssl/ssl.h>
#endif

// Mirror of the internal struct layouts in tya_runtime.c. The public
// header exposes `TyaBytes *`, `TyaDict *`, and `TyaArray *` opaquely;
// this translation unit needs to read their fields directly to
// serialize HTTP responses and iterate route tables. Keep these
// definitions in sync with `runtime/tya_runtime.c`.
typedef struct TyaGcHeaderLocal {
  unsigned char mark;
  unsigned char kind;
  size_t size;
  struct TyaGcHeaderLocal *next;
} TyaGcHeaderLocal;

struct TyaBytes {
  TyaGcHeaderLocal gc;
  int len;
  uint8_t *data;
};

struct TyaDict {
  TyaGcHeaderLocal gc;
  int len;
  TyaDictEntry *entries;
};

struct TyaArray {
  TyaGcHeaderLocal gc;
  int len;
  int cap;
  TyaValue *items;
};

#ifdef _WIN32
#include <winsock2.h>
#include <ws2tcpip.h>
typedef SOCKET TyaHttpSocket;
#define TYA_HTTP_INVALID_SOCKET INVALID_SOCKET
#define tya_http_socket_errno() WSAGetLastError()
#define tya_http_close_socket closesocket
#else
#include <arpa/inet.h>
#include <netinet/in.h>
#include <signal.h>
#include <sys/select.h>
#include <sys/socket.h>
#include <sys/syscall.h>
#include <sys/types.h>
#include <unistd.h>
typedef int TyaHttpSocket;
#define TYA_HTTP_INVALID_SOCKET (-1)
#define tya_http_socket_errno() errno
#define tya_http_close_socket close
#endif

typedef struct {
  TyaHttpSocket fd;
  void *ssl;
} TyaHttpConn;

// Defensive limits — keep one connection from exhausting memory.
#define TYA_HTTP_MAX_HEADER_BYTES (16 * 1024)
#define TYA_HTTP_MAX_BODY_BYTES (10 * 1024 * 1024)
#define TYA_HTTP_READ_CHUNK 4096
#define TYA_HTTP_MAX_REQUESTS_PER_CONNECTION 100

static void handle_connection(TyaHttpConn *conn, TyaValue routes);

static TyaValue http_connection_task(TyaValue self, TyaValue fd, TyaValue routes, TyaValue c, TyaValue d, TyaValue e, TyaValue f) {
  (void)self;
  (void)c;
  (void)d;
  (void)e;
  (void)f;
  if (fd.kind == TYA_NUMBER) {
    TyaHttpConn conn;
    conn.fd = (TyaHttpSocket)fd.number;
    conn.ssl = NULL;
    handle_connection(&conn, routes);
  }
  return tya_nil();
}

static void tya_http_socket_init(void) {
#ifdef _WIN32
  static int initialized = 0;
  if (initialized) return;
  WSADATA data;
  if (WSAStartup(MAKEWORD(2, 2), &data) != 0) {
    tya_panic(tya_string("net/http: WSAStartup failed"));
    return;
  }
  initialized = 1;
#endif
}

static int tya_http_socket_interrupted(void) {
#ifdef _WIN32
  return WSAGetLastError() == WSAEINTR;
#else
  return errno == EINTR;
#endif
}

static TyaHttpSocket tya_http_socket_open(int family, int type, int protocol) {
#ifdef _WIN32
  return socket(family, type, protocol);
#else
  return (TyaHttpSocket)syscall(SYS_socket, family, type, protocol);
#endif
}

static int tya_http_conn_read(TyaHttpConn *conn, char *buf, int len) {
#ifdef TYA_ENABLE_OPENSSL
  if (conn->ssl != NULL) return SSL_read((SSL *)conn->ssl, buf, len);
#endif
  return recv(conn->fd, buf, len, 0);
}

static int tya_http_conn_write(TyaHttpConn *conn, const char *buf, int len) {
#ifdef TYA_ENABLE_OPENSSL
  if (conn->ssl != NULL) return SSL_write((SSL *)conn->ssl, buf, len);
#endif
  return send(conn->fd, buf, len, 0);
}

static void tya_http_conn_close(TyaHttpConn *conn) {
#ifdef TYA_ENABLE_OPENSSL
  if (conn->ssl != NULL) {
    SSL_shutdown((SSL *)conn->ssl);
    SSL_free((SSL *)conn->ssl);
    conn->ssl = NULL;
  }
#endif
  if (conn->fd != TYA_HTTP_INVALID_SOCKET) {
    tya_http_close_socket(conn->fd);
    conn->fd = TYA_HTTP_INVALID_SOCKET;
  }
}

typedef struct {
  char *buf;
  int len;
  int cap;
} byte_buffer;

typedef struct http_arena_block {
  void *ptr;
  struct http_arena_block *next;
} http_arena_block;

typedef struct {
  http_arena_block *head;
} http_arena;

static void arena_init(http_arena *arena) {
  arena->head = NULL;
}

static void *arena_alloc(http_arena *arena, size_t size) {
  void *ptr = malloc(size);
  if (ptr == NULL) return NULL;
  http_arena_block *block = malloc(sizeof(http_arena_block));
  if (block == NULL) {
    free(ptr);
    return NULL;
  }
  block->ptr = ptr;
  block->next = arena->head;
  arena->head = block;
  return ptr;
}

static char *arena_dup_bytes(http_arena *arena, const char *src, int len) {
  char *out = arena_alloc(arena, len + 1);
  if (out == NULL) return NULL;
  memcpy(out, src, len);
  out[len] = '\0';
  return out;
}

static void arena_free(http_arena *arena) {
  http_arena_block *block = arena->head;
  while (block != NULL) {
    http_arena_block *next = block->next;
    free(block->ptr);
    free(block);
    block = next;
  }
  arena->head = NULL;
}

static void buf_init(byte_buffer *b) {
  b->buf = NULL;
  b->len = 0;
  b->cap = 0;
}

static int buf_append(byte_buffer *b, const char *data, int n) {
  if (n <= 0) return 0;
  if (b->len + n > b->cap) {
    int new_cap = b->cap == 0 ? 1024 : b->cap;
    while (new_cap < b->len + n) new_cap *= 2;
    char *nb = realloc(b->buf, new_cap);
    if (nb == NULL) return -1;
    b->buf = nb;
    b->cap = new_cap;
  }
  memcpy(b->buf + b->len, data, n);
  b->len += n;
  return 0;
}

static void buf_free(byte_buffer *b) {
  free(b->buf);
  b->buf = NULL;
  b->len = 0;
  b->cap = 0;
}

// dup_bytes copies len bytes from src into a freshly allocated
// NUL-terminated string. Request-owned strings use http_arena instead;
// this helper is for short-lived temporary buffers that callers free.
static char *dup_bytes(const char *src, int len) {
  char *out = malloc(len + 1);
  if (out == NULL) return NULL;
  memcpy(out, src, len);
  out[len] = '\0';
  return out;
}

// lowercase_inplace ASCII-lowercases s. Header names are
// canonicalized to lowercase before exposing to the handler.
static void lowercase_inplace(char *s) {
  for (; *s; s++) {
    if (*s >= 'A' && *s <= 'Z') *s += 32;
  }
}

// trim_ws moves the start pointer past leading ASCII whitespace and
// truncates trailing ASCII whitespace by overwriting with NUL.
// Returns the new start.
static char *trim_ws(char *s) {
  while (*s == ' ' || *s == '\t') s++;
  int n = (int)strlen(s);
  while (n > 0 && (s[n - 1] == ' ' || s[n - 1] == '\t' ||
                   s[n - 1] == '\r' || s[n - 1] == '\n')) {
    s[--n] = '\0';
  }
  return s;
}

static const char *find_bytes(const char *haystack, int haystack_len, const char *needle, int needle_len) {
  if (needle_len <= 0) return haystack;
  if (haystack_len < needle_len) return NULL;
  for (int i = 0; i <= haystack_len - needle_len; i++) {
    if (memcmp(haystack + i, needle, needle_len) == 0) return haystack + i;
  }
  return NULL;
}

// segment_match tries to match a pattern segment (e.g. ":id" or
// "users") against a request segment. When the pattern segment is a
// `:name` parameter, captures the request segment as the param.
//
// out_param_name / out_param_value are set when a capture happens;
// out_param_name points into the pattern (caller copies if it needs
// to outlive the pattern).
static int segment_match(const char *pat, int pat_len,
                         const char *req, int req_len,
                         const char **out_param_name, int *out_param_name_len,
                         const char **out_param_value, int *out_param_value_len) {
  if (pat_len > 0 && pat[0] == ':') {
    *out_param_name = pat + 1;
    *out_param_name_len = pat_len - 1;
    *out_param_value = req;
    *out_param_value_len = req_len;
    return 1;
  }
  if (pat_len != req_len) return 0;
  if (memcmp(pat, req, pat_len) != 0) return 0;
  return 1;
}

static char *strip_trailing_slash_copy(const char *s) {
  int n = (int)strlen(s == NULL ? "" : s);
  while (n > 1 && s[n - 1] == '/') n--;
  return dup_bytes(s, n);
}

// path_match compares a route pattern against a request path,
// optionally collecting :name captures into `params`. Returns 1 on
// match (and writes params into the dict via tya_dict_set).
static int path_match(const char *pat, const char *req, TyaValue params, http_arena *arena) {
  // Split both on '/' and walk segment by segment.
  const char *p = pat;
  const char *r = req;
  while (*p == '/') p++;
  while (*r == '/') r++;
  while (1) {
    const char *p_end = p;
    while (*p_end && *p_end != '/') p_end++;
    const char *r_end = r;
    while (*r_end && *r_end != '/') r_end++;
    int p_len = (int)(p_end - p);
    int r_len = (int)(r_end - r);
    int p_done = (p_len == 0 && *p_end == '\0');
    int r_done = (r_len == 0 && *r_end == '\0');
    if (p_done && r_done) return 1;
    if (p_done != r_done) return 0;

    if (p_len > 0 && p[0] == '*') {
      char *name = arena_dup_bytes(arena, p + 1, p_len - 1);
      const char *value_start = r;
      int value_len = (int)strlen(r);
      char *value = arena_dup_bytes(arena, value_start, value_len);
      if (name != NULL && value != NULL) {
        tya_dict_set(params, tya_string(name), tya_string(value));
      }
      return 1;
    }

    const char *param_name = NULL;
    int param_name_len = 0;
    const char *param_value = NULL;
    int param_value_len = 0;
    if (!segment_match(p, p_len, r, r_len,
                       &param_name, &param_name_len,
                       &param_value, &param_value_len)) {
      return 0;
    }
    if (param_name != NULL && param_name_len > 0) {
      char *name = arena_dup_bytes(arena, param_name, param_name_len);
      char *value = arena_dup_bytes(arena, param_value, param_value_len);
      if (name != NULL && value != NULL) {
        tya_dict_set(params, tya_string(name), tya_string(value));
      }
    }
    p = p_end;
    r = r_end;
    while (*p == '/') p++;
    while (*r == '/') r++;
  }
}

static int route_path_match(TyaValue route, const char *path, TyaValue params, http_arena *arena) {
  TyaValue r_path = tya_dict_get(route, tya_string("path"), tya_nil(), true);
  if (r_path.kind != TYA_STRING || r_path.string == NULL || path == NULL) return 0;
  TyaValue slash = tya_dict_get(route, tya_string("trailing_slash"), tya_string("strict"), true);
  if (slash.kind == TYA_STRING && slash.string != NULL && strcmp(slash.string, "ignore") == 0) {
    char *rp = strip_trailing_slash_copy(r_path.string);
    char *pp = strip_trailing_slash_copy(path);
    int ok = path_match(rp, pp, params, arena);
    free(rp);
    free(pp);
    return ok;
  }
  return path_match(r_path.string, path, params, arena);
}

static int route_method_match(const char *route_method, const char *request_method, int allow_head_fallback) {
  if (route_method == NULL || request_method == NULL) return 0;
  if (strcmp(route_method, "ANY") == 0) return 1;
  if (strcmp(route_method, request_method) == 0) return 1;
  if (allow_head_fallback && strcmp(request_method, "HEAD") == 0 && strcmp(route_method, "GET") == 0) return 1;
  return 0;
}

static TyaValue find_special_handler(TyaValue routes, const char *method) {
  if (routes.kind != TYA_ARRAY || routes.array == NULL) return tya_nil();
  for (int i = 0; i < routes.array->len; i++) {
    TyaValue r = routes.array->items[i];
    if (r.kind != TYA_DICT || r.dict == NULL) continue;
    TyaValue r_method = tya_dict_get(r, tya_string("method"), tya_nil(), true);
    if (r_method.kind == TYA_STRING && r_method.string != NULL && strcmp(r_method.string, method) == 0) {
      return tya_dict_get(r, tya_string("handler"), tya_nil(), true);
    }
  }
  return tya_nil();
}

static char *allow_header_for(TyaValue routes, const char *path) {
  char buf[256];
  buf[0] = '\0';
  if (routes.kind != TYA_ARRAY || routes.array == NULL) return strdup("");
  for (int i = 0; i < routes.array->len; i++) {
    TyaValue r = routes.array->items[i];
    if (r.kind != TYA_DICT || r.dict == NULL) continue;
    TyaValue r_method = tya_dict_get(r, tya_string("method"), tya_nil(), true);
    if (r_method.kind != TYA_STRING || r_method.string == NULL || r_method.string[0] == '_') continue;
    TyaValue params = tya_dict(NULL, 0);
    http_arena tmp_arena;
    arena_init(&tmp_arena);
    int matched = route_path_match(r, path, params, &tmp_arena);
    arena_free(&tmp_arena);
    if (!matched) continue;
    const char *method = strcmp(r_method.string, "ANY") == 0 ? "GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD" : r_method.string;
    if (strstr(buf, method) == NULL) {
      if (buf[0] != '\0') strncat(buf, ", ", sizeof(buf) - strlen(buf) - 1);
      strncat(buf, method, sizeof(buf) - strlen(buf) - 1);
    }
    if (strcmp(r_method.string, "GET") == 0 && strstr(buf, "HEAD") == NULL) {
      if (buf[0] != '\0') strncat(buf, ", ", sizeof(buf) - strlen(buf) - 1);
      strncat(buf, "HEAD", sizeof(buf) - strlen(buf) - 1);
    }
  }
  return strdup(buf);
}

// parse_query splits "?a=1&b=2" style content (without the leading
// '?'), populating `query` with string-typed entries.
static void parse_query(const char *qs, TyaValue query, http_arena *arena) {
  if (qs == NULL || *qs == '\0') return;
  const char *p = qs;
  while (*p) {
    const char *amp = strchr(p, '&');
    int pair_len = amp == NULL ? (int)strlen(p) : (int)(amp - p);
    const char *eq = memchr(p, '=', pair_len);
    if (eq != NULL) {
      int key_len = (int)(eq - p);
      int val_len = pair_len - key_len - 1;
      char *key = arena_dup_bytes(arena, p, key_len);
      char *val = arena_dup_bytes(arena, eq + 1, val_len);
      if (key != NULL && val != NULL) {
        tya_dict_set(query, tya_string(key), tya_string(val));
      }
    } else {
      char *key = arena_dup_bytes(arena, p, pair_len);
      if (key != NULL) {
        tya_dict_set(query, tya_string(key), tya_string(""));
      }
    }
    if (amp == NULL) break;
    p = amp + 1;
  }
}

static int request_keep_alive(const char *version, int version_len, const char *headers_text, int headers_len) {
  int http11 = version_len == 8 && memcmp(version, "HTTP/1.1", 8) == 0;
  int keep_alive = http11;
  const char *cursor = headers_text;
  const char *limit = headers_text + headers_len;
  int first_line = 1;
  while (cursor < limit) {
    const char *line_end = memchr(cursor, '\n', limit - cursor);
    if (line_end == NULL) line_end = limit;
    int line_len = (int)(line_end - cursor);
    while (line_len > 0 && cursor[line_len - 1] == '\r') line_len--;
    if (first_line) {
      first_line = 0;
    } else if (line_len > 0) {
      const char *colon = memchr(cursor, ':', line_len);
      if (colon != NULL) {
        int name_len = (int)(colon - cursor);
        char name[64];
        int n = name_len < (int)sizeof(name) - 1 ? name_len : (int)sizeof(name) - 1;
        memcpy(name, cursor, n);
        name[n] = '\0';
        lowercase_inplace(name);
        if (strcmp(name, "connection") == 0) {
          char *value = dup_bytes(colon + 1, line_len - name_len - 1);
          if (value != NULL) {
            lowercase_inplace(value);
            char *trimmed = trim_ws(value);
            if (strstr(trimmed, "close") != NULL) keep_alive = 0;
            if (strstr(trimmed, "keep-alive") != NULL) keep_alive = 1;
            free(value);
          }
        }
      }
    }
    if (line_end == limit) break;
    cursor = line_end + 1;
  }
  return keep_alive;
}

static void parse_cookie_header(const char *header, TyaValue cookies, http_arena *arena) {
  if (header == NULL || *header == '\0') return;
  char *copy = arena_dup_bytes(arena, header, (int)strlen(header));
  if (copy == NULL) return;
  char *p = copy;
  while (*p) {
    char *semi = strchr(p, ';');
    if (semi != NULL) *semi = '\0';
    char *pair = trim_ws(p);
    char *eq = strchr(pair, '=');
    if (eq != NULL) {
      *eq = '\0';
      char *name = trim_ws(pair);
      char *value = trim_ws(eq + 1);
      if (*name != '\0') {
        tya_dict_set(cookies, tya_string(name), tya_string(value));
      }
    }
    if (semi == NULL) break;
    p = semi + 1;
  }
}

static char *header_param_value(const char *header, const char *param, http_arena *arena) {
  if (header == NULL || param == NULL) return NULL;
  int param_len = (int)strlen(param);
  const char *p = header;
  while ((p = strstr(p, param)) != NULL) {
    if ((p == header || p[-1] == ';' || p[-1] == ' ' || p[-1] == '\t') && p[param_len] == '=') {
      p += param_len + 1;
      while (*p == ' ' || *p == '\t') p++;
      if (*p == '"') {
        p++;
        const char *end = strchr(p, '"');
        if (end == NULL) return NULL;
        return arena_dup_bytes(arena, p, (int)(end - p));
      }
      const char *end = p;
      while (*end && *end != ';' && *end != '\r' && *end != '\n') end++;
      char *value = arena_dup_bytes(arena, p, (int)(end - p));
      if (value == NULL) return NULL;
      return trim_ws(value);
    }
    p += param_len;
  }
  return NULL;
}

static char *multipart_boundary(const char *content_type, http_arena *arena) {
  if (content_type == NULL) return NULL;
  if (strstr(content_type, "multipart/form-data") == NULL) return NULL;
  return header_param_value(content_type, "boundary", arena);
}

static char *part_header_value(const char *headers, int headers_len, const char *wanted, http_arena *arena) {
  int wanted_len = (int)strlen(wanted);
  const char *cursor = headers;
  const char *limit = headers + headers_len;
  while (cursor < limit) {
    const char *line_end = find_bytes(cursor, (int)(limit - cursor), "\r\n", 2);
    if (line_end == NULL) line_end = limit;
    int line_len = (int)(line_end - cursor);
    const char *colon = memchr(cursor, ':', line_len);
    if (colon != NULL) {
      int name_len = (int)(colon - cursor);
      char name[64];
      int n = name_len < (int)sizeof(name) - 1 ? name_len : (int)sizeof(name) - 1;
      memcpy(name, cursor, n);
      name[n] = '\0';
      lowercase_inplace(name);
      if (name_len == wanted_len && strcmp(name, wanted) == 0) {
        char *value = arena_dup_bytes(arena, colon + 1, line_len - name_len - 1);
        if (value == NULL) return NULL;
        return trim_ws(value);
      }
    }
    if (line_end == limit) break;
    cursor = line_end + 2;
  }
  return NULL;
}

static int parse_multipart_body(const char *content_type, const uint8_t *body, int body_len, TyaValue form, TyaValue files, http_arena *arena) {
  char *boundary = multipart_boundary(content_type, arena);
  if (boundary == NULL) return 1;
  int boundary_len = (int)strlen(boundary);
  if (boundary_len == 0) return 0;
  int marker_len = boundary_len + 2;
  char *marker = malloc(marker_len + 1);
  if (marker == NULL) return 0;
  marker[0] = '-';
  marker[1] = '-';
  memcpy(marker + 2, boundary, boundary_len);
  marker[marker_len] = '\0';

  const char *buf = (const char *)body;
  const char *limit = buf + body_len;
  const char *pos = find_bytes(buf, body_len, marker, marker_len);
  if (pos == NULL) {
    free(marker);
    return 0;
  }
  pos += marker_len;
  int saw_final = 0;
  while (pos < limit) {
    if (pos + 2 <= limit && pos[0] == '-' && pos[1] == '-') {
      saw_final = 1;
      break;
    }
    if (pos + 2 > limit || pos[0] != '\r' || pos[1] != '\n') {
      free(marker);
      return 0;
    }
    pos += 2;
    const char *header_end = find_bytes(pos, (int)(limit - pos), "\r\n\r\n", 4);
    if (header_end == NULL) {
      free(marker);
      return 0;
    }
    const char *content = header_end + 4;
    const char *next = find_bytes(content, (int)(limit - content), marker, marker_len);
    if (next == NULL || next < content + 2 || next[-2] != '\r' || next[-1] != '\n') {
      free(marker);
      return 0;
    }
    int content_len = (int)(next - content - 2);
    char *disp = part_header_value(pos, (int)(header_end - pos), "content-disposition", arena);
    if (disp != NULL && strstr(disp, "form-data") != NULL) {
      char *name = header_param_value(disp, "name", arena);
      if (name != NULL && name[0] != '\0') {
        char *filename = header_param_value(disp, "filename", arena);
        if (filename != NULL) {
          TyaValue meta = tya_dict(NULL, 0);
          tya_dict_set(meta, tya_string("filename"), tya_string(filename));
          char *ct = part_header_value(pos, (int)(header_end - pos), "content-type", arena);
          tya_dict_set(meta, tya_string("content_type"), tya_string(ct == NULL ? "" : ct));
          tya_dict_set(meta, tya_string("body"), tya_bytes_lit(content, content_len));
          tya_dict_set(meta, tya_string("size"), tya_number(content_len));
          tya_dict_set(files, tya_string(name), meta);
        } else {
          char *value = arena_dup_bytes(arena, content, content_len);
          if (value != NULL) {
            tya_dict_set(form, tya_string(name), tya_string(value));
          }
        }
      }
    }
    pos = next + marker_len;
  }
  free(marker);
  return saw_final;
}

// read_request fills `headers_buf` until "\r\n\r\n" is observed,
// then reads the body up to Content-Length bytes into `body_buf`.
// Returns 0 on success, -1 on I/O / parse failure, -2 on overflow.
static int read_request(TyaHttpConn *conn, byte_buffer *headers_buf,
                        byte_buffer *body_buf, int *content_length_out) {
  char chunk[TYA_HTTP_READ_CHUNK];
  int header_end = -1;
  while (header_end < 0) {
    int n = tya_http_conn_read(conn, chunk, (int)sizeof(chunk));
    if (n <= 0) return -1;
    if (headers_buf->len + (int)n > TYA_HTTP_MAX_HEADER_BYTES + TYA_HTTP_MAX_BODY_BYTES) {
      return -2;
    }
    if (buf_append(headers_buf, chunk, (int)n) < 0) return -1;
    // Look for "\r\n\r\n" in headers_buf.
    for (int i = 0; i + 3 < headers_buf->len; i++) {
      if (headers_buf->buf[i] == '\r' &&
          headers_buf->buf[i + 1] == '\n' &&
          headers_buf->buf[i + 2] == '\r' &&
          headers_buf->buf[i + 3] == '\n') {
        header_end = i;
        break;
      }
    }
    if (header_end < 0 && headers_buf->len > TYA_HTTP_MAX_HEADER_BYTES) return -2;
  }

  // Anything past header_end+4 in headers_buf belongs to body.
  int already = headers_buf->len - (header_end + 4);
  if (already > 0) {
    if (buf_append(body_buf, headers_buf->buf + header_end + 4, already) < 0) return -1;
  }
  headers_buf->len = header_end + 2; // keep "...\r\n" for line splitting; drop second \r\n

  // Parse Content-Length: scan headers_buf for the header.
  int content_length = 0;
  {
    char *cursor = headers_buf->buf;
    char *limit = headers_buf->buf + headers_buf->len;
    while (cursor < limit) {
      char *line_end = memchr(cursor, '\n', limit - cursor);
      if (line_end == NULL) line_end = limit;
      int line_len = (int)(line_end - cursor);
      if (line_len >= 15) {
        // case-insensitive Content-Length:
        char head[16];
        int n = line_len < 15 ? line_len : 15;
        memcpy(head, cursor, n);
        head[n] = '\0';
        lowercase_inplace(head);
        if (memcmp(head, "content-length:", 15) == 0) {
          char num_buf[32];
          int j = 0;
          char *p = cursor + 15;
          while (p < line_end && j < (int)sizeof(num_buf) - 1) {
            if (*p >= '0' && *p <= '9') num_buf[j++] = *p;
            p++;
          }
          num_buf[j] = '\0';
          content_length = atoi(num_buf);
        }
      }
      if (line_end == limit) break;
      cursor = line_end + 1;
    }
  }
  if (content_length < 0) content_length = 0;
  if (content_length > TYA_HTTP_MAX_BODY_BYTES) return -2;
  *content_length_out = content_length;

  while (body_buf->len < content_length) {
    int want = content_length - body_buf->len;
    if (want > (int)sizeof(chunk)) want = (int)sizeof(chunk);
    int n = tya_http_conn_read(conn, chunk, want);
    if (n <= 0) return -1;
    if (buf_append(body_buf, chunk, (int)n) < 0) return -1;
  }
  return 0;
}

// build_request_dict turns the raw headers + body into a Tya dict
// matching the SPEC shape.
static TyaValue build_request_dict(const char *method, int method_len,
                                   const char *path, int path_len,
                                   const char *query_str, int query_len,
                                   const char *headers_text, int headers_len,
                                   const uint8_t *body, int body_len,
                                   TyaValue params, int keep_alive, http_arena *arena, int *bad_request) {
  TyaValue req = tya_dict(NULL, 0);
  {
    char *m = arena_dup_bytes(arena, method, method_len);
    if (m != NULL) {
      // method is already uppercase from HTTP/1.1.
      tya_dict_set(req, tya_string("method"), tya_string(m));
    }
  }
  {
    char *p = arena_dup_bytes(arena, path, path_len);
    if (p != NULL) {
      tya_dict_set(req, tya_string("path"), tya_string(p));
    }
  }
  tya_dict_set(req, tya_string("params"), params);
  tya_dict_set(req, tya_string("path_params"), params);
  tya_dict_set(req, tya_string("keep_alive"), tya_bool(keep_alive));

  TyaValue query = tya_dict(NULL, 0);
  if (query_len > 0) {
    char *q = dup_bytes(query_str, query_len);
    if (q != NULL) {
      parse_query(q, query, arena);
      free(q);
    }
  }
  tya_dict_set(req, tya_string("query"), query);

  TyaValue headers = tya_dict(NULL, 0);
  if (headers_text != NULL && headers_len > 0) {
    const char *cursor = headers_text;
    const char *limit = headers_text + headers_len;
    int first_line = 1;
    while (cursor < limit) {
      const char *line_end = memchr(cursor, '\n', limit - cursor);
      if (line_end == NULL) line_end = limit;
      int line_len = (int)(line_end - cursor);
      // strip trailing \r
      while (line_len > 0 && (cursor[line_len - 1] == '\r')) line_len--;
      if (first_line) {
        first_line = 0;
      } else if (line_len > 0) {
        const char *colon = memchr(cursor, ':', line_len);
        if (colon != NULL) {
          int name_len = (int)(colon - cursor);
          char *name = arena_dup_bytes(arena, cursor, name_len);
          int value_start = (int)(colon - cursor) + 1;
          char *value = arena_dup_bytes(arena, cursor + value_start, line_len - value_start);
          if (name != NULL && value != NULL) {
            lowercase_inplace(name);
            char *value_trim = trim_ws(value);
            tya_dict_set(headers, tya_string(name), tya_string(value_trim));
          }
        }
      }
      if (line_end == limit) break;
      cursor = line_end + 1;
    }
  }
  tya_dict_set(req, tya_string("headers"), headers);
  TyaValue cookies = tya_dict(NULL, 0);
  TyaValue cookie_header = tya_dict_get(headers, tya_string("cookie"), tya_nil(), true);
  if (cookie_header.kind == TYA_STRING && cookie_header.string != NULL) {
    parse_cookie_header(cookie_header.string, cookies, arena);
  }
  tya_dict_set(req, tya_string("cookies"), cookies);
  TyaValue form = tya_dict(NULL, 0);
  TyaValue files = tya_dict(NULL, 0);
  TyaValue content_type = tya_dict_get(headers, tya_string("content-type"), tya_nil(), true);
  if (content_type.kind == TYA_STRING && content_type.string != NULL) {
    if (!parse_multipart_body(content_type.string, body, body_len, form, files, arena)) {
      if (bad_request != NULL) *bad_request = 1;
    }
  }
  tya_dict_set(req, tya_string("form"), form);
  tya_dict_set(req, tya_string("files"), files);

  if (body_len > 0 && body != NULL) {
    TyaValue body_value = tya_bytes_lit((const char *)body, body_len);
    tya_dict_set(req, tya_string("body"), body_value);
  } else {
    tya_dict_set(req, tya_string("body"), tya_bytes_lit((const char *)0, 0));
  }
  return req;
}

// status_text returns the canonical reason phrase for a status code.
static const char *status_text(int status) {
  switch (status) {
    case 200: return "OK";
    case 201: return "Created";
    case 204: return "No Content";
    case 301: return "Moved Permanently";
    case 302: return "Found";
    case 400: return "Bad Request";
    case 401: return "Unauthorized";
    case 403: return "Forbidden";
    case 404: return "Not Found";
    case 405: return "Method Not Allowed";
    case 500: return "Internal Server Error";
    default: return "OK";
  }
}

// write_all writes len bytes from buf to fd, looping past partial
// writes. Returns 0 on success, -1 on error.
static int write_all(TyaHttpConn *conn, const char *buf, int len) {
  int written = 0;
  while (written < len) {
    int n = tya_http_conn_write(conn, buf + written, len - written);
    if (n <= 0) {
      if (n < 0 && tya_http_socket_interrupted()) continue;
      return -1;
    }
    written += (int)n;
  }
  return 0;
}

static int write_chunk(TyaHttpConn *conn, const uint8_t *data, int len) {
  if (len <= 0) return 0;
  char prefix[32];
  int pn = snprintf(prefix, sizeof(prefix), "%x\r\n", len);
  if (pn <= 0 || write_all(conn, prefix, pn) < 0) return -1;
  if (write_all(conn, (const char *)data, len) < 0) return -1;
  return write_all(conn, "\r\n", 2);
}

static int write_chunk_value(TyaHttpConn *conn, TyaValue value) {
  if (value.kind == TYA_STRING && value.string != NULL) {
    return write_chunk(conn, (const uint8_t *)value.string, (int)strlen(value.string));
  }
  if (value.kind == TYA_BYTES && value.bytes != NULL) {
    return write_chunk(conn, value.bytes->data, value.bytes->len);
  }
  return -1;
}

static int write_chunked_body(TyaHttpConn *conn, TyaValue body) {
  if (body.kind == TYA_ARRAY && body.array != NULL) {
    for (int i = 0; i < body.array->len; i++) {
      if (write_chunk_value(conn, body.array->items[i]) < 0) return -1;
    }
    return write_all(conn, "0\r\n\r\n", 5);
  }
  if (body.kind == TYA_CHANNEL && body.channel != NULL) {
    while (1) {
      TyaValue item = tya_channel_receive(body);
      if (item.kind == TYA_NIL) break;
      if (write_chunk_value(conn, item) < 0) return -1;
    }
    return write_all(conn, "0\r\n\r\n", 5);
  }
  return -1;
}

// write_response serialises a handler's response dict over fd.
static void write_response(TyaHttpConn *conn, TyaValue resp, int keep_alive) {
  int status = 200;
  TyaValue status_v = tya_dict_get(resp, tya_string("status"), tya_nil(), true);
  if (status_v.kind == TYA_NUMBER) status = (int)status_v.number;
  TyaValue chunked_v = tya_dict_get(resp, tya_string("chunked"), tya_bool(false), true);
  int chunked = chunked_v.kind == TYA_BOOL && chunked_v.boolean;

  // Determine body bytes + length.
  const uint8_t *body = NULL;
  int body_len = 0;
  TyaValue body_v = tya_dict_get(resp, tya_string("body"), tya_nil(), true);
  if (body_v.kind == TYA_STRING && body_v.string != NULL) {
    body = (const uint8_t *)body_v.string;
    body_len = (int)strlen(body_v.string);
  } else if (body_v.kind == TYA_BYTES && body_v.bytes != NULL) {
    body = body_v.bytes->data;
    body_len = body_v.bytes->len;
  }

  // Write status line.
  char status_line[64];
  int n = snprintf(status_line, sizeof(status_line), "HTTP/1.1 %d %s\r\n", status, status_text(status));
  write_all(conn, status_line, n);

  // Default content-type when no headers dict.
  TyaValue headers_v = tya_dict_get(resp, tya_string("headers"), tya_nil(), true);
  int has_content_type = 0;
  if (headers_v.kind == TYA_DICT && headers_v.dict != NULL) {
    for (int i = 0; i < headers_v.dict->len; i++) {
      TyaDictEntry e = headers_v.dict->entries[i];
      if (e.key == NULL) continue;
      char lname[64];
      int klen = (int)strlen(e.key);
      if (klen >= (int)sizeof(lname)) klen = sizeof(lname) - 1;
      memcpy(lname, e.key, klen);
      lname[klen] = '\0';
      lowercase_inplace(lname);
      if (strcmp(lname, "connection") == 0) continue;
      if (chunked && strcmp(lname, "content-length") == 0) continue;
      if (chunked && strcmp(lname, "transfer-encoding") == 0) continue;
      if (strcmp(lname, "content-type") == 0) has_content_type = 1;
      const char *vstr = "";
      if (e.value.kind == TYA_STRING && e.value.string != NULL) vstr = e.value.string;
      char header_line[1024];
      int hn = snprintf(header_line, sizeof(header_line), "%s: %s\r\n", e.key, vstr);
      if (hn > 0) write_all(conn, header_line, hn);
    }
  }
  TyaValue header_values_v = tya_dict_get(resp, tya_string("header_values"), tya_nil(), true);
  if (header_values_v.kind == TYA_DICT && header_values_v.dict != NULL) {
    for (int i = 0; i < header_values_v.dict->len; i++) {
      TyaDictEntry e = header_values_v.dict->entries[i];
      if (e.key == NULL || e.value.kind != TYA_ARRAY || e.value.array == NULL) continue;
      for (int j = 0; j < e.value.array->len; j++) {
        TyaValue hv = e.value.array->items[j];
        if (hv.kind != TYA_STRING || hv.string == NULL) continue;
        char header_line[1024];
        int hn = snprintf(header_line, sizeof(header_line), "%s: %s\r\n", e.key, hv.string);
        if (hn > 0) write_all(conn, header_line, hn);
      }
    }
  }
  if (!has_content_type) {
    const char *default_ct = "Content-Type: text/plain; charset=utf-8\r\n";
    write_all(conn, default_ct, (int)strlen(default_ct));
  }

  if (chunked) {
    const char *te = "Transfer-Encoding: chunked\r\n";
    write_all(conn, te, (int)strlen(te));
  } else {
    char clbuf[64];
    int cln = snprintf(clbuf, sizeof(clbuf), "Content-Length: %d\r\n", body_len);
    write_all(conn, clbuf, cln);
  }

  const char *connection_header = keep_alive ? "Connection: keep-alive\r\n\r\n" : "Connection: close\r\n\r\n";
  write_all(conn, connection_header, (int)strlen(connection_header));

  if (chunked) {
    write_chunked_body(conn, body_v);
  } else if (body != NULL && body_len > 0) {
    write_all(conn, (const char *)body, body_len);
  }
}

// write_simple_response writes a status + plain-text body without
// invoking a handler. Used for routing failures and parse errors.
static void write_simple_response(TyaHttpConn *conn, int status, const char *body) {
  TyaValue resp = tya_dict(NULL, 0);
  tya_dict_set(resp, tya_string("status"), tya_number(status));
  tya_dict_set(resp, tya_string("body"), tya_string(body == NULL ? status_text(status) : body));
  write_response(conn, resp, 0);
}

static const char *static_content_type_for_path(const char *path) {
  if (path == NULL) return "application/octet-stream";
  int n = (int)strlen(path);
  if (n >= 5 && strcmp(path + n - 5, ".html") == 0) return "text/html; charset=utf-8";
  if (n >= 4 && strcmp(path + n - 4, ".css") == 0) return "text/css; charset=utf-8";
  if (n >= 3 && strcmp(path + n - 3, ".js") == 0) return "application/javascript; charset=utf-8";
  if (n >= 5 && strcmp(path + n - 5, ".json") == 0) return "application/json";
  if (n >= 4 && strcmp(path + n - 4, ".svg") == 0) return "image/svg+xml";
  return "application/octet-stream";
}

static TyaValue build_static_response(TyaValue route, TyaValue req) {
  TyaValue asset = tya_dict_get(route, tya_string("static_asset"), tya_nil(), true);
  TyaValue route_path = tya_dict_get(route, tya_string("path"), tya_nil(), true);
  TyaValue content_type = route_path.kind == TYA_STRING ? tya_string(static_content_type_for_path(route_path.string)) : tya_string("application/octet-stream");
  TyaValue hashed = tya_dict_get(route, tya_string("static_hashed"), tya_bool(false), true);

  TyaValue body = asset;
  TyaValue headers = tya_dict(NULL, 0);
  if (content_type.kind == TYA_STRING) {
    tya_dict_set(headers, tya_string("Content-Type"), content_type);
  }

  if (asset.kind == TYA_DICT && asset.dict != NULL) {
    body = tya_dict_get(asset, tya_string("content"), tya_nil(), true);
    TyaValue hash = tya_dict_get(asset, tya_string("hash"), tya_nil(), true);
    if (hash.kind == TYA_STRING) {
      tya_dict_set(headers, tya_string("ETag"), hash);
    }
    TyaValue asset_ct = tya_dict_get(asset, tya_string("content_type"), tya_nil(), true);
    if (asset_ct.kind == TYA_STRING) {
      tya_dict_set(headers, tya_string("Content-Type"), asset_ct);
    }
    TyaValue req_headers = tya_dict_get(req, tya_string("headers"), tya_nil(), true);
    TyaValue accept = tya_dict_get(req_headers, tya_string("accept-encoding"), tya_nil(), true);
    TyaValue gzip_body = tya_dict_get(asset, tya_string("gzip"), tya_nil(), true);
    if (accept.kind == TYA_STRING && gzip_body.kind == TYA_BYTES &&
        strstr(accept.string, "gzip") != NULL) {
      body = gzip_body;
      tya_dict_set(headers, tya_string("Content-Encoding"), tya_string("gzip"));
    }
  }

  if (hashed.kind == TYA_BOOL && hashed.boolean) {
    tya_dict_set(headers, tya_string("Cache-Control"), tya_string("public, max-age=31536000, immutable"));
  } else {
    tya_dict_set(headers, tya_string("Cache-Control"), tya_string("no-cache"));
  }

  TyaValue resp = tya_dict(NULL, 0);
  tya_dict_set(resp, tya_string("status"), tya_number(200));
  tya_dict_set(resp, tya_string("headers"), headers);
  tya_dict_set(resp, tya_string("body"), body);
  return resp;
}

// handle_connection reads one HTTP/1.1 request, dispatches it to
// the matching route, and writes the response. Closes fd on exit.
static void handle_connection(TyaHttpConn *conn, TyaValue routes) {
  for (int request_count = 0; request_count < TYA_HTTP_MAX_REQUESTS_PER_CONNECTION; request_count++) {
  byte_buffer headers_buf;
  byte_buffer body_buf;
  buf_init(&headers_buf);
  buf_init(&body_buf);
  http_arena arena;
  arena_init(&arena);
  int content_length = 0;
  if (read_request(conn, &headers_buf, &body_buf, &content_length) != 0) {
    if (request_count == 0) write_simple_response(conn, 400, "Bad Request");
    arena_free(&arena);
    buf_free(&headers_buf);
    buf_free(&body_buf);
    tya_http_conn_close(conn);
    return;
  }

  // Parse request line "METHOD PATH HTTP/1.1\r\n".
  char *line_end = memchr(headers_buf.buf, '\n', headers_buf.len);
  if (line_end == NULL) {
    write_simple_response(conn, 400, "Bad Request");
    arena_free(&arena);
    buf_free(&headers_buf);
    buf_free(&body_buf);
    tya_http_conn_close(conn);
    return;
  }
  int rl_len = (int)(line_end - headers_buf.buf);
  if (rl_len > 0 && headers_buf.buf[rl_len - 1] == '\r') rl_len--;

  // Tokenize: method SP path SP version.
  const char *method = headers_buf.buf;
  int method_len = 0;
  while (method_len < rl_len && method[method_len] != ' ') method_len++;
  if (method_len == 0 || method_len == rl_len) {
    write_simple_response(conn, 400, "Bad Request");
    arena_free(&arena);
    buf_free(&headers_buf);
    buf_free(&body_buf);
    tya_http_conn_close(conn);
    return;
  }
  const char *path_start = method + method_len + 1;
  int rem = rl_len - method_len - 1;
  int path_len = 0;
  while (path_len < rem && path_start[path_len] != ' ') path_len++;
  const char *version = path_start + path_len;
  int version_len = 0;
  if (path_len < rem && *version == ' ') {
    version++;
    version_len = rem - path_len - 1;
  }
  int keep_alive = request_keep_alive(version, version_len, (const char *)headers_buf.buf, headers_buf.len);
  if (request_count + 1 >= TYA_HTTP_MAX_REQUESTS_PER_CONNECTION) keep_alive = 0;

  // Split path and query.
  const char *query_str = NULL;
  int query_len = 0;
  const char *q_mark = memchr(path_start, '?', path_len);
  int path_only_len = path_len;
  if (q_mark != NULL) {
    path_only_len = (int)(q_mark - path_start);
    query_str = q_mark + 1;
    query_len = path_len - path_only_len - 1;
  }

  // Iterate routes and find first match by (method, path).
  if (routes.kind != TYA_ARRAY || routes.array == NULL) {
    write_simple_response(conn, 500, "Server misconfigured: routes missing");
    arena_free(&arena);
    buf_free(&headers_buf);
    buf_free(&body_buf);
    tya_http_conn_close(conn);
    return;
  }
  TyaValue matched_handler = tya_nil();
  TyaValue matched_params = tya_nil();
  TyaValue matched_route = tya_nil();
  char *path_cstr = dup_bytes(path_start, path_only_len);
  char *method_cstr = dup_bytes(method, method_len);
  int head_fallback = 0;
  for (int i = 0; i < routes.array->len; i++) {
    TyaValue r = routes.array->items[i];
    if (r.kind != TYA_DICT || r.dict == NULL) continue;
    TyaValue r_method = tya_dict_get(r, tya_string("method"), tya_nil(), true);
    TyaValue r_handler = tya_dict_get(r, tya_string("handler"), tya_nil(), true);
    if (r_method.kind != TYA_STRING || r_method.string == NULL || r_method.string[0] == '_') continue;
    if (method_cstr == NULL) continue;
    if (!route_method_match(r_method.string, method_cstr, 0)) continue;
    TyaValue params = tya_dict(NULL, 0);
    if (route_path_match(r, path_cstr, params, &arena)) {
      matched_handler = r_handler;
      matched_params = params;
      matched_route = r;
      break;
    }
  }
  if (matched_route.kind == TYA_NIL && method_cstr != NULL && strcmp(method_cstr, "HEAD") == 0) {
    for (int i = 0; i < routes.array->len; i++) {
      TyaValue r = routes.array->items[i];
      if (r.kind != TYA_DICT || r.dict == NULL) continue;
      TyaValue r_method = tya_dict_get(r, tya_string("method"), tya_nil(), true);
      TyaValue r_handler = tya_dict_get(r, tya_string("handler"), tya_nil(), true);
      if (r_method.kind != TYA_STRING || r_method.string == NULL) continue;
      if (!route_method_match(r_method.string, method_cstr, 1)) continue;
      TyaValue params = tya_dict(NULL, 0);
      if (route_path_match(r, path_cstr, params, &arena)) {
        matched_handler = r_handler;
        matched_params = params;
        matched_route = r;
        head_fallback = 1;
        break;
      }
    }
  }
  if (matched_route.kind == TYA_NIL && method_cstr != NULL && strcmp(method_cstr, "OPTIONS") == 0) {
    char *allow = allow_header_for(routes, path_cstr);
    if (allow != NULL && allow[0] != '\0') {
      TyaValue headers = tya_dict(NULL, 0);
      tya_dict_set(headers, tya_string("Allow"), tya_string(allow));
      TyaValue resp = tya_dict(NULL, 0);
      tya_dict_set(resp, tya_string("status"), tya_number(204));
      tya_dict_set(resp, tya_string("headers"), headers);
      tya_dict_set(resp, tya_string("body"), tya_string(""));
      write_response(conn, resp, keep_alive);
      free(method_cstr);
      free(path_cstr);
      arena_free(&arena);
      buf_free(&headers_buf);
      buf_free(&body_buf);
      if (keep_alive) continue;
      tya_http_conn_close(conn);
      return;
    }
    free(allow);
  }
  free(method_cstr);
  free(path_cstr);

  TyaValue static_asset = tya_dict_get(matched_route, tya_string("static_asset"), tya_nil(), true);
  if (matched_handler.kind != TYA_FUNCTION && static_asset.kind == TYA_NIL) {
    TyaValue not_found = find_special_handler(routes, "__NOT_FOUND__");
    if (not_found.kind == TYA_FUNCTION) {
      int bad_request = 0;
      TyaValue req = build_request_dict(method, method_len,
                                        path_start, path_only_len,
                                        query_str, query_len,
                                        (const char *)(headers_buf.buf), headers_buf.len,
                                        (const uint8_t *)body_buf.buf, body_buf.len,
                                        tya_dict(NULL, 0), keep_alive, &arena, &bad_request);
      if (bad_request) {
        write_simple_response(conn, 400, "Bad Request");
        arena_free(&arena);
        buf_free(&headers_buf);
        buf_free(&body_buf);
        tya_http_conn_close(conn);
        return;
      }
      TyaValue resp = tya_call1(not_found, req);
      if (resp.kind == TYA_DICT) write_response(conn, resp, keep_alive);
      else write_simple_response(conn, 404, "Not Found");
    } else {
      write_simple_response(conn, 404, "Not Found");
    }
    arena_free(&arena);
    buf_free(&headers_buf);
    buf_free(&body_buf);
    if (keep_alive) continue;
    tya_http_conn_close(conn);
    return;
  }

  // Headers text begins after the request line.
  const char *headers_text = (const char *)(headers_buf.buf);
  int headers_text_len = headers_buf.len;

  int bad_request = 0;
  TyaValue req = build_request_dict(method, method_len,
                                    path_start, path_only_len,
                                    query_str, query_len,
                                    headers_text, headers_text_len,
                                    (const uint8_t *)body_buf.buf, body_buf.len,
                                    matched_params, keep_alive, &arena, &bad_request);
  if (bad_request) {
    write_simple_response(conn, 400, "Bad Request");
    arena_free(&arena);
    buf_free(&headers_buf);
    buf_free(&body_buf);
    tya_http_conn_close(conn);
    return;
  }
  tya_dict_set(req, tya_string("route"), matched_route);
  TyaValue resp = tya_nil();
  if (matched_handler.kind == TYA_FUNCTION) {
    TyaRaiseFrame frame;
    if (setjmp(frame.env) == 0) {
      tya_push_raise_frame(&frame);
      resp = tya_call1(matched_handler, req);
      tya_pop_raise_frame();
    } else {
      tya_pop_raise_frame();
      TyaValue error_handler = find_special_handler(routes, "__ERROR__");
      if (error_handler.kind == TYA_FUNCTION) {
        resp = tya_call2(error_handler, frame.value, req);
      } else {
        resp = tya_dict(NULL, 0);
        tya_dict_set(resp, tya_string("status"), tya_number(500));
        tya_dict_set(resp, tya_string("body"), tya_string("Internal Server Error"));
      }
    }
  } else {
    resp = build_static_response(matched_route, req);
  }
  if (head_fallback && resp.kind == TYA_DICT) {
    tya_dict_set(resp, tya_string("body"), tya_string(""));
  }
  if (resp.kind != TYA_DICT) {
    write_simple_response(conn, 500, "Handler returned non-dict");
  } else {
    write_response(conn, resp, keep_alive);
  }
  arena_free(&arena);
  buf_free(&headers_buf);
  buf_free(&body_buf);
  if (keep_alive) continue;
  tya_http_conn_close(conn);
  return;
  }
  tya_http_conn_close(conn);
}

TyaValue tya_http_server_run(TyaValue routes, TyaValue port) {
  tya_http_socket_init();
  int port_num = 0;
  if (port.kind == TYA_NUMBER) port_num = (int)port.number;
  if (port_num < 0 || port_num > 65535) {
    tya_panic(tya_string("net/http: port out of range"));
    return tya_nil();
  }

#ifndef _WIN32
  // Ignore SIGPIPE so a client disconnect mid-write doesn't kill us.
  signal(SIGPIPE, SIG_IGN);
#endif

  TyaHttpSocket sock = tya_http_socket_open(AF_INET, SOCK_STREAM, 0);
  if (sock == TYA_HTTP_INVALID_SOCKET) {
    tya_panic(tya_string("net/http: socket() failed"));
    return tya_nil();
  }
  int one = 1;
  setsockopt(sock, SOL_SOCKET, SO_REUSEADDR, (const char *)&one, sizeof(one));

  struct sockaddr_in addr;
  memset(&addr, 0, sizeof(addr));
  addr.sin_family = AF_INET;
  addr.sin_addr.s_addr = htonl(INADDR_ANY);
  addr.sin_port = htons((uint16_t)port_num);
  if (bind(sock, (struct sockaddr *)&addr, sizeof(addr)) < 0) {
    char msg[128];
#ifdef _WIN32
    snprintf(msg, sizeof(msg), "net/http: bind() failed on port %d: WSA error %d", port_num, tya_http_socket_errno());
#else
    snprintf(msg, sizeof(msg), "net/http: bind() failed on port %d: %s", port_num, strerror(errno));
#endif
    tya_http_close_socket(sock);
    tya_panic(tya_string(msg));
    return tya_nil();
  }
  if (listen(sock, 16) < 0) {
    tya_http_close_socket(sock);
    tya_panic(tya_string("net/http: listen() failed"));
    return tya_nil();
  }

  // Recover the actual port when port_num was 0.
  socklen_t alen = sizeof(addr);
  if (getsockname(sock, (struct sockaddr *)&addr, &alen) == 0) {
    int actual = ntohs(addr.sin_port);
    fprintf(stderr, "listening on %d\n", actual);
    fflush(stderr);
  }

  for (;;) {
    while (tya_task_has_ready()) {
      tya_task_run_ready();
    }
    double delay = tya_task_next_wake_delay(0.05);
    struct timeval tv;
    tv.tv_sec = (time_t)delay;
    tv.tv_usec = (long)((delay - (double)tv.tv_sec) * 1000000.0);
    fd_set rfds;
    FD_ZERO(&rfds);
    FD_SET(sock, &rfds);
#ifdef _WIN32
    int ready = select(0, &rfds, NULL, NULL, &tv);
#else
    int ready = select(sock + 1, &rfds, NULL, NULL, &tv);
#endif
    if (ready < 0) {
      if (tya_http_socket_interrupted()) continue;
      break;
    }
    if (ready == 0) continue;
    struct sockaddr_in caddr;
    socklen_t clen = sizeof(caddr);
    TyaHttpSocket c = accept(sock, (struct sockaddr *)&caddr, &clen);
    if (c == TYA_HTTP_INVALID_SOCKET) {
      if (tya_http_socket_interrupted()) continue;
      break;
    }
    tya_task_new(tya_function(http_connection_task), 2, tya_number((double)c), routes, tya_nil(), tya_nil());
  }
  tya_http_close_socket(sock);
  return tya_nil();
}

TyaValue tya_http_server_run_tls(TyaValue routes, TyaValue port, TyaValue cert_file, TyaValue key_file, TyaValue options) {
#ifndef TYA_ENABLE_OPENSSL
  (void)routes; (void)port; (void)cert_file; (void)key_file; (void)options;
  tya_panic(tya_string("http.tls: OpenSSL support is not enabled"));
  return tya_nil();
#else
  tya_http_socket_init();
  if (cert_file.kind != TYA_STRING || cert_file.string == NULL || key_file.kind != TYA_STRING || key_file.string == NULL) {
    tya_panic(tya_string("http.tls: cert_file and key_file must be strings"));
    return tya_nil();
  }
  int port_num = 0;
  if (port.kind == TYA_NUMBER) port_num = (int)port.number;
  if (port_num < 0 || port_num > 65535) {
    tya_panic(tya_string("net/http: port out of range"));
    return tya_nil();
  }
  const char *host = "0.0.0.0";
  double timeout = 0.0;
  if (options.kind == TYA_DICT || options.kind == TYA_OBJECT) {
    TyaValue hv = tya_index(options, tya_string("host"));
    if (hv.kind == TYA_STRING && hv.string != NULL) host = hv.string;
    TyaValue tv = tya_index(options, tya_string("timeout"));
    if (tv.kind == TYA_NUMBER && tv.number > 0) timeout = tv.number;
  }
#ifndef _WIN32
  signal(SIGPIPE, SIG_IGN);
#endif
  SSL_CTX *ctx = SSL_CTX_new(TLS_server_method());
  if (ctx == NULL) {
    tya_panic(tya_string("http.tls: failed to create TLS context"));
    return tya_nil();
  }
  if (SSL_CTX_use_certificate_file(ctx, cert_file.string, SSL_FILETYPE_PEM) != 1 ||
      SSL_CTX_use_PrivateKey_file(ctx, key_file.string, SSL_FILETYPE_PEM) != 1 ||
      SSL_CTX_check_private_key(ctx) != 1) {
    SSL_CTX_free(ctx);
    tya_panic(tya_string("http.tls: failed to load certificate or private key"));
    return tya_nil();
  }
  TyaHttpSocket sock = tya_http_socket_open(AF_INET, SOCK_STREAM, 0);
  if (sock == TYA_HTTP_INVALID_SOCKET) {
    SSL_CTX_free(ctx);
    tya_panic(tya_string("net/http: socket() failed"));
    return tya_nil();
  }
  int one = 1;
  setsockopt(sock, SOL_SOCKET, SO_REUSEADDR, (const char *)&one, sizeof(one));
  struct sockaddr_in addr;
  memset(&addr, 0, sizeof(addr));
  addr.sin_family = AF_INET;
  addr.sin_port = htons((uint16_t)port_num);
  if (inet_pton(AF_INET, host, &addr.sin_addr) != 1) {
    tya_http_close_socket(sock);
    SSL_CTX_free(ctx);
    tya_panic(tya_string("http.tls: host must be an IPv4 address"));
    return tya_nil();
  }
  if (bind(sock, (struct sockaddr *)&addr, sizeof(addr)) < 0) {
    tya_http_close_socket(sock);
    SSL_CTX_free(ctx);
    tya_panic(tya_string("http.tls: bind() failed"));
    return tya_nil();
  }
  if (listen(sock, 16) < 0) {
    tya_http_close_socket(sock);
    SSL_CTX_free(ctx);
    tya_panic(tya_string("http.tls: listen() failed"));
    return tya_nil();
  }
  socklen_t alen = sizeof(addr);
  if (getsockname(sock, (struct sockaddr *)&addr, &alen) == 0) {
    int actual = ntohs(addr.sin_port);
    fprintf(stderr, "listening on %d\n", actual);
    fflush(stderr);
  }
  for (;;) {
    struct sockaddr_in caddr;
    socklen_t clen = sizeof(caddr);
    TyaHttpSocket c = accept(sock, (struct sockaddr *)&caddr, &clen);
    if (c == TYA_HTTP_INVALID_SOCKET) {
      if (tya_http_socket_interrupted()) continue;
      break;
    }
    if (timeout > 0) {
      struct timeval tv;
      tv.tv_sec = (time_t)timeout;
      tv.tv_usec = (long)((timeout - (double)tv.tv_sec) * 1000000.0);
      setsockopt(c, SOL_SOCKET, SO_RCVTIMEO, (const char *)&tv, sizeof(tv));
      setsockopt(c, SOL_SOCKET, SO_SNDTIMEO, (const char *)&tv, sizeof(tv));
    }
    SSL *ssl = SSL_new(ctx);
    if (ssl == NULL) {
      tya_http_close_socket(c);
      continue;
    }
    SSL_set_fd(ssl, (int)c);
    if (SSL_accept(ssl) != 1) {
      SSL_free(ssl);
      tya_http_close_socket(c);
      continue;
    }
    TyaHttpConn conn;
    conn.fd = c;
    conn.ssl = ssl;
    handle_connection(&conn, routes);
  }
  tya_http_close_socket(sock);
  SSL_CTX_free(ctx);
  return tya_nil();
#endif
}
