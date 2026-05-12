#include "tya_http_server.h"

#include <errno.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>

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
// v0.58 does not implement Windows yet. Provide a stub so the
// translation unit still builds.
TyaValue tya_http_server_run(TyaValue routes, TyaValue port) {
  (void)routes;
  (void)port;
  tya_panic(tya_string("net/http: server not supported on Windows in v0.58"));
  return tya_nil();
}
#else

#include <arpa/inet.h>
#include <netinet/in.h>
#include <signal.h>
#include <sys/socket.h>
#include <sys/types.h>
#include <unistd.h>

// Defensive limits — keep one connection from exhausting memory.
#define TYA_HTTP_MAX_HEADER_BYTES (16 * 1024)
#define TYA_HTTP_MAX_BODY_BYTES (10 * 1024 * 1024)
#define TYA_HTTP_READ_CHUNK 4096

typedef struct {
  char *buf;
  int len;
  int cap;
} byte_buffer;

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
// NUL-terminated string. The runtime's TyaValue wrappers store the
// pointer by reference (no internal copy), so strings handed to
// tya_string / tya_dict_set must outlive the request. In v0.58 we
// intentionally leak these per-request buffers; switching to a per-
// request arena or a GC-tracked string allocator is a v0.59+
// follow-up.
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

// path_match compares a route pattern against a request path,
// optionally collecting :name captures into `params`. Returns 1 on
// match (and writes params into the dict via tya_dict_set).
static int path_match(const char *pat, const char *req, TyaValue params) {
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
      char *name = dup_bytes(param_name, param_name_len);
      char *value = dup_bytes(param_value, param_value_len);
      if (name != NULL && value != NULL) {
        tya_dict_set(params, tya_string(name), tya_string(value));
      }
      // Intentional per-request leak: see dup_bytes comment.
    }
    p = p_end;
    r = r_end;
    while (*p == '/') p++;
    while (*r == '/') r++;
  }
}

// parse_query splits "?a=1&b=2" style content (without the leading
// '?'), populating `query` with string-typed entries.
static void parse_query(const char *qs, TyaValue query) {
  if (qs == NULL || *qs == '\0') return;
  const char *p = qs;
  while (*p) {
    const char *amp = strchr(p, '&');
    int pair_len = amp == NULL ? (int)strlen(p) : (int)(amp - p);
    const char *eq = memchr(p, '=', pair_len);
    if (eq != NULL) {
      int key_len = (int)(eq - p);
      int val_len = pair_len - key_len - 1;
      char *key = dup_bytes(p, key_len);
      char *val = dup_bytes(eq + 1, val_len);
      if (key != NULL && val != NULL) {
        tya_dict_set(query, tya_string(key), tya_string(val));
      }
      // Intentional per-request leak.
    } else {
      char *key = dup_bytes(p, pair_len);
      if (key != NULL) {
        tya_dict_set(query, tya_string(key), tya_string(""));
      }
      // Intentional per-request leak.
    }
    if (amp == NULL) break;
    p = amp + 1;
  }
}

// read_request fills `headers_buf` until "\r\n\r\n" is observed,
// then reads the body up to Content-Length bytes into `body_buf`.
// Returns 0 on success, -1 on I/O / parse failure, -2 on overflow.
static int read_request(int fd, byte_buffer *headers_buf,
                        byte_buffer *body_buf, int *content_length_out) {
  char chunk[TYA_HTTP_READ_CHUNK];
  int header_end = -1;
  while (header_end < 0) {
    ssize_t n = read(fd, chunk, sizeof(chunk));
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
    ssize_t n = read(fd, chunk, want);
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
                                   TyaValue params) {
  TyaValue req = tya_dict(NULL, 0);
  {
    char *m = dup_bytes(method, method_len);
    if (m != NULL) {
      // method is already uppercase from HTTP/1.1.
      tya_dict_set(req, tya_string("method"), tya_string(m));
    }
  }
  {
    char *p = dup_bytes(path, path_len);
    if (p != NULL) {
      tya_dict_set(req, tya_string("path"), tya_string(p));
    }
  }
  tya_dict_set(req, tya_string("params"), params);

  TyaValue query = tya_dict(NULL, 0);
  if (query_len > 0) {
    char *q = dup_bytes(query_str, query_len);
    if (q != NULL) {
      parse_query(q, query);
      // q itself is throwaway after parse_query (parse_query
      // duped each key/value separately); free here is OK.
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
          char *name = dup_bytes(cursor, name_len);
          int value_start = (int)(colon - cursor) + 1;
          char *value = dup_bytes(cursor + value_start, line_len - value_start);
          if (name != NULL && value != NULL) {
            lowercase_inplace(name);
            char *value_trim = trim_ws(value);
            tya_dict_set(headers, tya_string(name), tya_string(value_trim));
            // Intentional per-request leak (see dup_bytes comment).
            // value_trim points into value; value can't be freed.
          }
        }
      }
      if (line_end == limit) break;
      cursor = line_end + 1;
    }
  }
  tya_dict_set(req, tya_string("headers"), headers);

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
static int write_all(int fd, const char *buf, int len) {
  int written = 0;
  while (written < len) {
    ssize_t n = write(fd, buf + written, len - written);
    if (n <= 0) {
      if (n < 0 && errno == EINTR) continue;
      return -1;
    }
    written += (int)n;
  }
  return 0;
}

// write_response serialises a handler's response dict over fd.
static void write_response(int fd, TyaValue resp) {
  int status = 200;
  TyaValue status_v = tya_dict_get(resp, tya_string("status"), tya_nil(), true);
  if (status_v.kind == TYA_NUMBER) status = (int)status_v.number;

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
  write_all(fd, status_line, n);

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
      if (strcmp(lname, "content-type") == 0) has_content_type = 1;
      const char *vstr = "";
      if (e.value.kind == TYA_STRING && e.value.string != NULL) vstr = e.value.string;
      char header_line[1024];
      int hn = snprintf(header_line, sizeof(header_line), "%s: %s\r\n", e.key, vstr);
      if (hn > 0) write_all(fd, header_line, hn);
    }
  }
  if (!has_content_type) {
    const char *default_ct = "Content-Type: text/plain; charset=utf-8\r\n";
    write_all(fd, default_ct, (int)strlen(default_ct));
  }

  char clbuf[64];
  int cln = snprintf(clbuf, sizeof(clbuf), "Content-Length: %d\r\n", body_len);
  write_all(fd, clbuf, cln);

  const char *connection_close = "Connection: close\r\n\r\n";
  write_all(fd, connection_close, (int)strlen(connection_close));

  if (body != NULL && body_len > 0) {
    write_all(fd, (const char *)body, body_len);
  }
}

// write_simple_response writes a status + plain-text body without
// invoking a handler. Used for routing failures and parse errors.
static void write_simple_response(int fd, int status, const char *body) {
  TyaValue resp = tya_dict(NULL, 0);
  tya_dict_set(resp, tya_string("status"), tya_number(status));
  tya_dict_set(resp, tya_string("body"), tya_string(body == NULL ? status_text(status) : body));
  write_response(fd, resp);
}

// handle_connection reads one HTTP/1.1 request, dispatches it to
// the matching route, and writes the response. Closes fd on exit.
static void handle_connection(int fd, TyaValue routes) {
  byte_buffer headers_buf;
  byte_buffer body_buf;
  buf_init(&headers_buf);
  buf_init(&body_buf);
  int content_length = 0;
  if (read_request(fd, &headers_buf, &body_buf, &content_length) != 0) {
    write_simple_response(fd, 400, "Bad Request");
    buf_free(&headers_buf);
    buf_free(&body_buf);
    close(fd);
    return;
  }

  // Parse request line "METHOD PATH HTTP/1.1\r\n".
  char *line_end = memchr(headers_buf.buf, '\n', headers_buf.len);
  if (line_end == NULL) {
    write_simple_response(fd, 400, "Bad Request");
    buf_free(&headers_buf);
    buf_free(&body_buf);
    close(fd);
    return;
  }
  int rl_len = (int)(line_end - headers_buf.buf);
  if (rl_len > 0 && headers_buf.buf[rl_len - 1] == '\r') rl_len--;

  // Tokenize: method SP path SP version.
  const char *method = headers_buf.buf;
  int method_len = 0;
  while (method_len < rl_len && method[method_len] != ' ') method_len++;
  if (method_len == 0 || method_len == rl_len) {
    write_simple_response(fd, 400, "Bad Request");
    buf_free(&headers_buf);
    buf_free(&body_buf);
    close(fd);
    return;
  }
  const char *path_start = method + method_len + 1;
  int rem = rl_len - method_len - 1;
  int path_len = 0;
  while (path_len < rem && path_start[path_len] != ' ') path_len++;

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
    write_simple_response(fd, 500, "Server misconfigured: routes missing");
    buf_free(&headers_buf);
    buf_free(&body_buf);
    close(fd);
    return;
  }
  TyaValue matched_handler = tya_nil();
  TyaValue matched_params = tya_nil();
  char *path_cstr = dup_bytes(path_start, path_only_len);
  char *method_cstr = dup_bytes(method, method_len);
  for (int i = 0; i < routes.array->len; i++) {
    TyaValue r = routes.array->items[i];
    if (r.kind != TYA_DICT || r.dict == NULL) continue;
    TyaValue r_method = tya_dict_get(r, tya_string("method"), tya_nil(), true);
    TyaValue r_path = tya_dict_get(r, tya_string("path"), tya_nil(), true);
    TyaValue r_handler = tya_dict_get(r, tya_string("handler"), tya_nil(), true);
    if (r_method.kind != TYA_STRING || r_path.kind != TYA_STRING) continue;
    if (method_cstr == NULL) continue;
    if (strcmp(r_method.string, method_cstr) != 0) continue;
    TyaValue params = tya_dict(NULL, 0);
    if (path_match(r_path.string, path_cstr, params)) {
      matched_handler = r_handler;
      matched_params = params;
      break;
    }
  }
  free(method_cstr);
  free(path_cstr);

  if (matched_handler.kind != TYA_FUNCTION) {
    write_simple_response(fd, 404, "Not Found");
    buf_free(&headers_buf);
    buf_free(&body_buf);
    close(fd);
    return;
  }

  // Headers text begins after the request line.
  const char *headers_text = (const char *)(headers_buf.buf);
  int headers_text_len = headers_buf.len;

  TyaValue req = build_request_dict(method, method_len,
                                    path_start, path_only_len,
                                    query_str, query_len,
                                    headers_text, headers_text_len,
                                    (const uint8_t *)body_buf.buf, body_buf.len,
                                    matched_params);
  TyaValue resp = tya_call1(matched_handler, req);
  if (resp.kind != TYA_DICT) {
    write_simple_response(fd, 500, "Handler returned non-dict");
  } else {
    write_response(fd, resp);
  }
  buf_free(&headers_buf);
  buf_free(&body_buf);
  close(fd);
}

TyaValue tya_http_server_run(TyaValue routes, TyaValue port) {
  int port_num = 0;
  if (port.kind == TYA_NUMBER) port_num = (int)port.number;
  if (port_num < 0 || port_num > 65535) {
    tya_panic(tya_string("net/http: port out of range"));
    return tya_nil();
  }

  // Ignore SIGPIPE so a client disconnect mid-write doesn't kill us.
  signal(SIGPIPE, SIG_IGN);

  int sock = socket(AF_INET, SOCK_STREAM, 0);
  if (sock < 0) {
    tya_panic(tya_string("net/http: socket() failed"));
    return tya_nil();
  }
  int one = 1;
  setsockopt(sock, SOL_SOCKET, SO_REUSEADDR, &one, sizeof(one));

  struct sockaddr_in addr;
  memset(&addr, 0, sizeof(addr));
  addr.sin_family = AF_INET;
  addr.sin_addr.s_addr = htonl(INADDR_ANY);
  addr.sin_port = htons((uint16_t)port_num);
  if (bind(sock, (struct sockaddr *)&addr, sizeof(addr)) < 0) {
    char msg[128];
    snprintf(msg, sizeof(msg), "net/http: bind() failed on port %d: %s", port_num, strerror(errno));
    close(sock);
    tya_panic(tya_string(msg));
    return tya_nil();
  }
  if (listen(sock, 16) < 0) {
    close(sock);
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
    struct sockaddr_in caddr;
    socklen_t clen = sizeof(caddr);
    int c = accept(sock, (struct sockaddr *)&caddr, &clen);
    if (c < 0) {
      if (errno == EINTR) continue;
      break;
    }
    handle_connection(c, routes);
  }
  close(sock);
  return tya_nil();
}

#endif /* _WIN32 */
