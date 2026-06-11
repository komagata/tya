static TyaValue tya_socket_value(TyaSocketHandle fd, TyaResourceSubkind subkind, bool binary, double timeout) {
  TyaResource *r = tya_resource_new(subkind);
  r->socket_fd = fd;
  r->socket_binary = binary;
  r->socket_closed = false;
  r->socket_timeout = timeout;
  return (TyaValue){.kind = TYA_RESOURCE, .resource = r};
}

static bool tya_socket_binary_option(TyaValue options) {
  if (options.kind != TYA_DICT && options.kind != TYA_OBJECT) return false;
  TyaValue mode = tya_index(options, tya_string("mode"));
  return mode.kind == TYA_STRING && mode.string != NULL && strcmp(mode.string, "binary") == 0;
}

static double tya_socket_timeout_option(TyaValue options) {
  if (options.kind != TYA_DICT && options.kind != TYA_OBJECT) return 0.0;
  TyaValue timeout = tya_index(options, tya_string("timeout"));
  if (timeout.kind != TYA_NUMBER || timeout.number <= 0) return 0.0;
  return timeout.number;
}

static void tya_socket_init(void) {
#ifdef _WIN32
  static bool initialized = false;
  if (initialized) return;
  WSADATA data;
  if (WSAStartup(MAKEWORD(2, 2), &data) != 0) {
    tya_raise(tya_string("net/socket: WSAStartup failed"));
    return;
  }
  initialized = true;
#endif
}

static void tya_socket_close_handle(TyaSocketHandle fd) {
#ifdef _WIN32
  closesocket(fd);
#else
  close(fd);
#endif
}

static TyaSocketHandle tya_socket_open(int family, int type, int protocol) {
#ifdef _WIN32
  return socket(family, type, protocol);
#else
  return (TyaSocketHandle)syscall(SYS_socket, family, type, protocol);
#endif
}

static void tya_socket_apply_timeout(TyaSocketHandle fd, double seconds) {
  if (seconds <= 0) return;
#ifdef _WIN32
  DWORD ms = (DWORD)(seconds * 1000.0);
  if (ms == 0) ms = 1;
  setsockopt(fd, SOL_SOCKET, SO_RCVTIMEO, (const char *)&ms, sizeof(ms));
  setsockopt(fd, SOL_SOCKET, SO_SNDTIMEO, (const char *)&ms, sizeof(ms));
#else
  struct timeval tv;
  tv.tv_sec = (time_t)seconds;
  tv.tv_usec = (suseconds_t)((seconds - (double)tv.tv_sec) * 1000000.0);
  if (tv.tv_sec == 0 && tv.tv_usec == 0) tv.tv_usec = 1;
  setsockopt(fd, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof(tv));
  setsockopt(fd, SOL_SOCKET, SO_SNDTIMEO, &tv, sizeof(tv));
#endif
}

static void tya_socket_raise_errno(const char *op) {
#ifdef _WIN32
  int err = WSAGetLastError();
  if (err == WSAETIMEDOUT || err == WSAEWOULDBLOCK) {
    char buf[128];
    snprintf(buf, sizeof(buf), "%s: timeout", op);
    tya_raise(tya_string(buf));
    return;
  }
  char buf[160];
  snprintf(buf, sizeof(buf), "%s: WSA error %d", op, err);
  tya_raise(tya_string(buf));
#else
  if (errno == EAGAIN || errno == EWOULDBLOCK) {
    char buf[128];
    snprintf(buf, sizeof(buf), "%s: timeout", op);
    tya_raise(tya_string(buf));
    return;
  }
  tya_raise(tya_string(strerror(errno)));
#endif
}

static int tya_socket_port(TyaValue port, const char *op) {
  if (port.kind != TYA_NUMBER) {
    char buf[96];
    snprintf(buf, sizeof(buf), "%s: port must be a number", op);
    tya_raise(tya_string(buf));
    return -1;
  }
  int p = (int)port.number;
  if (p < 0 || p > 65535) {
    char buf[96];
    snprintf(buf, sizeof(buf), "%s: invalid port", op);
    tya_raise(tya_string(buf));
    return -1;
  }
  return p;
}

static TyaResource *tya_socket_check(TyaValue socket, TyaResourceSubkind subkind, const char *op) {
  TyaResource *r = tya_resource_check(socket, subkind, op);
  if (r == NULL) return NULL;
  if (r->socket_closed || r->socket_fd == TYA_INVALID_SOCKET) {
    char buf[128];
    snprintf(buf, sizeof(buf), "%s: socket is closed", op);
    tya_raise(tya_string(buf));
    return NULL;
  }
  return r;
}

static TyaValue tya_sockaddr_value(struct sockaddr_storage *addr, socklen_t len) {
  char host[NI_MAXHOST];
  char serv[NI_MAXSERV];
  if (getnameinfo((struct sockaddr *)addr, len, host, sizeof(host), serv, sizeof(serv), NI_NUMERICHOST | NI_NUMERICSERV) != 0) {
    return tya_dict((TyaDictEntry[]){{"host", tya_string("")}, {"port", tya_number(0)}}, 2);
  }
  return tya_dict((TyaDictEntry[]){{"host", tya_string(strdup(host))}, {"port", tya_number((double)atoi(serv))}}, 2);
}

#ifdef TYA_ENABLE_OPENSSL
static void tya_tls_raise(const char *op) {
  unsigned long err = ERR_get_error();
  char buf[256];
  if (err != 0) {
    char detail[160];
    ERR_error_string_n(err, detail, sizeof(detail));
    snprintf(buf, sizeof(buf), "%s: %s", op, detail);
  } else {
    snprintf(buf, sizeof(buf), "%s: TLS error", op);
  }
  tya_raise(tya_string(buf));
}

static bool tya_tls_bool_option(TyaValue options, const char *name) {
  if (options.kind != TYA_DICT && options.kind != TYA_OBJECT) return false;
  TyaValue value = tya_index(options, tya_string(name));
  return value.kind == TYA_BOOL && value.boolean;
}

static const char *tya_tls_string_option(TyaValue options, const char *name) {
  if (options.kind != TYA_DICT && options.kind != TYA_OBJECT) return NULL;
  TyaValue value = tya_index(options, tya_string(name));
  if (value.kind != TYA_STRING || value.string == NULL) return NULL;
  return value.string;
}
#endif

TyaValue tya_socket_connect(TyaValue host, TyaValue port, TyaValue options) {
  tya_socket_init();
  if (host.kind != TYA_STRING || host.string == NULL) {
    tya_raise(tya_string("socket.connect: host must be a string"));
    return tya_nil();
  }
  int p = tya_socket_port(port, "socket.connect");
  if (p < 0) return tya_nil();
  char port_s[16];
  snprintf(port_s, sizeof(port_s), "%d", p);
  struct addrinfo hints;
  memset(&hints, 0, sizeof(hints));
  hints.ai_family = AF_UNSPEC;
  hints.ai_socktype = SOCK_STREAM;
  struct addrinfo *res = NULL;
  int rc = getaddrinfo(host.string, port_s, &hints, &res);
  if (rc != 0) {
    tya_raise(tya_string(gai_strerror(rc)));
    return tya_nil();
  }
  TyaSocketHandle fd = TYA_INVALID_SOCKET;
  for (struct addrinfo *ai = res; ai != NULL; ai = ai->ai_next) {
    fd = tya_socket_open(ai->ai_family, ai->ai_socktype, ai->ai_protocol);
    if (fd == TYA_INVALID_SOCKET) continue;
    if (connect(fd, ai->ai_addr, ai->ai_addrlen) == 0) break;
    tya_socket_close_handle(fd);
    fd = TYA_INVALID_SOCKET;
  }
  freeaddrinfo(res);
  if (fd == TYA_INVALID_SOCKET) {
#ifdef _WIN32
    tya_socket_raise_errno("socket.connect");
#else
    tya_raise(tya_string(strerror(errno)));
#endif
    return tya_nil();
  }
  double timeout = tya_socket_timeout_option(options);
  tya_socket_apply_timeout(fd, timeout);
  return tya_socket_value(fd, TYA_RES_SOCKET, tya_socket_binary_option(options), timeout);
}

TyaValue tya_tls_connect(TyaValue host, TyaValue port, TyaValue options) {
#ifndef TYA_ENABLE_OPENSSL
  (void)host; (void)port; (void)options;
  tya_raise(tya_string("http.tls: OpenSSL support is not enabled"));
  return tya_nil();
#else
  TyaValue socket = tya_socket_connect(host, port, options);
  TyaResource *r = tya_resource_check(socket, TYA_RES_SOCKET, "http.tls.connect");
  if (r == NULL) return tya_nil();
  SSL_CTX *ctx = SSL_CTX_new(TLS_client_method());
  if (ctx == NULL) {
    tya_socket_close(socket);
    tya_tls_raise("http.tls");
    return tya_nil();
  }
  bool insecure = tya_tls_bool_option(options, "insecure_skip_verify");
  if (insecure) {
    SSL_CTX_set_verify(ctx, SSL_VERIFY_NONE, NULL);
  } else {
    SSL_CTX_set_verify(ctx, SSL_VERIFY_PEER, NULL);
    const char *ca_file = tya_tls_string_option(options, "ca_file");
    int ok = ca_file != NULL ? SSL_CTX_load_verify_locations(ctx, ca_file, NULL) : SSL_CTX_set_default_verify_paths(ctx);
    if (ok != 1) {
      SSL_CTX_free(ctx);
      tya_socket_close(socket);
      tya_tls_raise("http.tls");
      return tya_nil();
    }
  }
  SSL *ssl = SSL_new(ctx);
  if (ssl == NULL) {
    SSL_CTX_free(ctx);
    tya_socket_close(socket);
    tya_tls_raise("http.tls");
    return tya_nil();
  }
  SSL_set_fd(ssl, (int)r->socket_fd);
  if (host.kind == TYA_STRING && host.string != NULL) {
    SSL_set_tlsext_host_name(ssl, host.string);
    if (!insecure) SSL_set1_host(ssl, host.string);
  }
  if (SSL_connect(ssl) != 1) {
    SSL_free(ssl);
    SSL_CTX_free(ctx);
    tya_socket_close(socket);
    tya_tls_raise("http.tls");
    return tya_nil();
  }
  r->tls_ssl = ssl;
  r->tls_ctx = ctx;
  return socket;
#endif
}

TyaValue tya_socket_server_listen(TyaValue host, TyaValue port, TyaValue options) {
  tya_socket_init();
  if (host.kind != TYA_STRING || host.string == NULL) {
    tya_raise(tya_string("socket.listen: host must be a string"));
    return tya_nil();
  }
  int p = tya_socket_port(port, "socket.listen");
  if (p < 0) return tya_nil();
  TyaSocketHandle fd = tya_socket_open(AF_INET, SOCK_STREAM, 0);
  if (fd == TYA_INVALID_SOCKET) {
#ifdef _WIN32
    tya_socket_raise_errno("socket.listen");
#else
    tya_raise(tya_string(strerror(errno)));
#endif
    return tya_nil();
  }
  int yes = 1;
  setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, (const char *)&yes, sizeof(yes));
  struct sockaddr_in addr;
  memset(&addr, 0, sizeof(addr));
  addr.sin_family = AF_INET;
  addr.sin_port = htons((uint16_t)p);
  if (inet_pton(AF_INET, host.string, &addr.sin_addr) != 1) {
    tya_socket_close_handle(fd);
    tya_raise(tya_string("socket.listen: host must be an IPv4 address"));
    return tya_nil();
  }
  if (bind(fd, (struct sockaddr *)&addr, sizeof(addr)) != 0 || listen(fd, 16) != 0) {
#ifdef _WIN32
    tya_socket_close_handle(fd);
    tya_socket_raise_errno("socket.listen");
#else
    char *msg = strdup(strerror(errno));
    tya_socket_close_handle(fd);
    tya_raise(tya_string(msg));
#endif
    return tya_nil();
  }
  double timeout = tya_socket_timeout_option(options);
  tya_socket_apply_timeout(fd, timeout);
  return tya_socket_value(fd, TYA_RES_SOCKET_SERVER, tya_socket_binary_option(options), timeout);
}

TyaValue tya_socket_server_accept(TyaValue server) {
  TyaResource *r = tya_socket_check(server, TYA_RES_SOCKET_SERVER, "socket.accept");
  if (r == NULL) return tya_nil();
  TyaSocketHandle fd = accept(r->socket_fd, NULL, NULL);
  if (fd == TYA_INVALID_SOCKET) {
    tya_socket_raise_errno("socket.accept");
    return tya_nil();
  }
  tya_socket_apply_timeout(fd, r->socket_timeout);
  return tya_socket_value(fd, TYA_RES_SOCKET, r->socket_binary, r->socket_timeout);
}

TyaValue tya_socket_read(TyaValue socket, TyaValue size_v) {
  TyaResource *r = tya_socket_check(socket, TYA_RES_SOCKET, "socket.read");
  if (r == NULL) return tya_nil();
  if (size_v.kind != TYA_NUMBER || size_v.number < 0) {
    tya_raise(tya_string("socket.read: size must be a non-negative number"));
    return tya_nil();
  }
  int size = (int)size_v.number;
  char *buf = malloc((size_t)(size > 0 ? size : 1));
  if (buf == NULL) {
    tya_raise(tya_string("socket.read: out of memory"));
    return tya_nil();
  }
  int n = 0;
#ifdef TYA_ENABLE_OPENSSL
  if (r->tls_ssl != NULL) n = SSL_read((SSL *)r->tls_ssl, buf, size);
  else
#endif
  n = recv(r->socket_fd, buf, (int)size, 0);
  if (n < 0) {
    free(buf);
    tya_socket_raise_errno("socket.read");
    return tya_nil();
  }
  TyaValue out = r->socket_binary ? tya_bytes_lit(buf, (int)n) : tya_string_from_buffer(buf, (int)n);
  free(buf);
  return out;
}

TyaValue tya_socket_read_line(TyaValue socket) {
  TyaResource *r = tya_socket_check(socket, TYA_RES_SOCKET, "socket.read_line");
  if (r == NULL) return tya_nil();
  size_t cap = 128;
  size_t len = 0;
  char *buf = malloc(cap);
  if (buf == NULL) {
    tya_raise(tya_string("socket.read_line: out of memory"));
    return tya_nil();
  }
  char ch;
  while (true) {
    int n = 0;
#ifdef TYA_ENABLE_OPENSSL
    if (r->tls_ssl != NULL) n = SSL_read((SSL *)r->tls_ssl, &ch, 1);
    else
#endif
    n = recv(r->socket_fd, &ch, 1, 0);
    if (n == 0) break;
    if (n < 0) {
      free(buf);
      tya_socket_raise_errno("socket.read_line");
      return tya_nil();
    }
    if (len + 1 >= cap) {
      cap *= 2;
      char *next = realloc(buf, cap);
      if (next == NULL) {
        free(buf);
        tya_raise(tya_string("socket.read_line: out of memory"));
        return tya_nil();
      }
      buf = next;
    }
    buf[len++] = ch;
    if (ch == '\n') break;
  }
  if (len == 0) {
    free(buf);
    return tya_nil();
  }
  TyaValue out = r->socket_binary ? tya_bytes_lit(buf, (int)len) : tya_string_from_buffer(buf, (int)len);
  free(buf);
  return out;
}

TyaValue tya_socket_write(TyaValue socket, TyaValue value) {
  TyaResource *r = tya_socket_check(socket, TYA_RES_SOCKET, "socket.write");
  if (r == NULL) return tya_nil();
  const unsigned char *data = NULL;
  size_t len = 0;
  TyaValue text = value;
  if (value.kind == TYA_BYTES && value.bytes != NULL) {
    data = value.bytes->data;
    len = (size_t)value.bytes->len;
  } else {
    text = tya_to_string(value);
    if (text.string == NULL) return tya_number(0);
    data = (const unsigned char *)text.string;
    len = strlen(text.string);
  }
  size_t sent = 0;
  while (sent < len) {
    int n = 0;
#ifdef TYA_ENABLE_OPENSSL
    if (r->tls_ssl != NULL) n = SSL_write((SSL *)r->tls_ssl, data + sent, (int)(len - sent));
    else
#endif
    n = send(r->socket_fd, (const char *)(data + sent), (int)(len - sent), 0);
    if (n < 0) {
      tya_socket_raise_errno("socket.write");
      return tya_nil();
    }
    sent += (size_t)n;
  }
  return tya_number((double)sent);
}

TyaValue tya_socket_close(TyaValue socket) {
  TyaResource *r = tya_resource_check(socket, TYA_RES_SOCKET, "socket.close");
  if (r == NULL || r->socket_closed) return tya_nil();
#ifdef TYA_ENABLE_OPENSSL
  if (r->tls_ssl != NULL) {
    SSL_shutdown((SSL *)r->tls_ssl);
    SSL_free((SSL *)r->tls_ssl);
    r->tls_ssl = NULL;
  }
  if (r->tls_ctx != NULL) {
    SSL_CTX_free((SSL_CTX *)r->tls_ctx);
    r->tls_ctx = NULL;
  }
#endif
  tya_socket_close_handle(r->socket_fd);
  r->socket_fd = TYA_INVALID_SOCKET;
  r->socket_closed = true;
  return tya_nil();
}

TyaValue tya_socket_server_close(TyaValue server) {
  TyaResource *r = tya_resource_check(server, TYA_RES_SOCKET_SERVER, "socket.server.close");
  if (r == NULL || r->socket_closed) return tya_nil();
  tya_socket_close_handle(r->socket_fd);
  r->socket_fd = TYA_INVALID_SOCKET;
  r->socket_closed = true;
  return tya_nil();
}

TyaValue tya_socket_closed(TyaValue socket) {
  TyaResource *r = tya_resource_check(socket, TYA_RES_SOCKET, "socket.closed?");
  if (r == NULL) return tya_bool(true);
  return tya_bool(r->socket_closed || r->socket_fd == TYA_INVALID_SOCKET);
}

TyaValue tya_socket_local_address(TyaValue socket) {
  TyaResource *r = tya_socket_check(socket, TYA_RES_SOCKET, "socket.local_address");
  if (r == NULL) return tya_nil();
  struct sockaddr_storage addr;
  socklen_t len = sizeof(addr);
  if (getsockname(r->socket_fd, (struct sockaddr *)&addr, &len) != 0) {
    tya_raise(tya_string(strerror(errno)));
    return tya_nil();
  }
  return tya_sockaddr_value(&addr, len);
}

TyaValue tya_socket_remote_address(TyaValue socket) {
  TyaResource *r = tya_socket_check(socket, TYA_RES_SOCKET, "socket.remote_address");
  if (r == NULL) return tya_nil();
  struct sockaddr_storage addr;
  socklen_t len = sizeof(addr);
  if (getpeername(r->socket_fd, (struct sockaddr *)&addr, &len) != 0) {
    tya_raise(tya_string(strerror(errno)));
    return tya_nil();
  }
  return tya_sockaddr_value(&addr, len);
}

TyaValue tya_socket_server_local_address(TyaValue server) {
  TyaResource *r = tya_socket_check(server, TYA_RES_SOCKET_SERVER, "socket.server.local_address");
  if (r == NULL) return tya_nil();
  struct sockaddr_storage addr;
  socklen_t len = sizeof(addr);
  if (getsockname(r->socket_fd, (struct sockaddr *)&addr, &len) != 0) {
    tya_raise(tya_string(strerror(errno)));
    return tya_nil();
  }
  return tya_sockaddr_value(&addr, len);
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
      return tya_call0(callee);
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

TyaValue tya_current_task(void) {
  TyaTask *t = tya_current_task_ptr;
  if (t == NULL) return tya_nil();
  return (TyaValue){.kind = TYA_TASK, .task = t};
}

static void tya_task_enqueue(TyaTask *t) {
  if (t == NULL || t->done || t->queued) return;
  t->next_ready = NULL;
  if (tya_ready_tail == NULL) {
    tya_ready_head = t;
    tya_ready_tail = t;
  } else {
    tya_ready_tail->next_ready = t;
    tya_ready_tail = t;
  }
  t->queued = true;
}

static TyaTask *tya_task_dequeue(void) {
  tya_task_wake_sleepers();
  TyaTask *t = tya_ready_head;
  if (t == NULL) return NULL;
  tya_ready_head = t->next_ready;
  if (tya_ready_head == NULL) tya_ready_tail = NULL;
  t->next_ready = NULL;
  t->queued = false;
  return t;
}

static void tya_task_sleep_until(TyaTask *t, double wake_time) {
  if (t == NULL || t->done) return;
  t->sleeping = true;
  t->wake_time = wake_time;
  t->next_sleep = NULL;
  if (tya_sleep_head == NULL || wake_time < tya_sleep_head->wake_time) {
    t->next_sleep = tya_sleep_head;
    tya_sleep_head = t;
    return;
  }
  TyaTask *cur = tya_sleep_head;
  while (cur->next_sleep != NULL && cur->next_sleep->wake_time <= wake_time) {
    cur = cur->next_sleep;
  }
  t->next_sleep = cur->next_sleep;
  cur->next_sleep = t;
}

static void tya_task_wake_sleepers(void) {
  double now = tya_now_seconds();
  while (tya_sleep_head != NULL && tya_sleep_head->wake_time <= now) {
    TyaTask *t = tya_sleep_head;
    tya_sleep_head = t->next_sleep;
    t->next_sleep = NULL;
    t->sleeping = false;
    tya_task_enqueue(t);
  }
}

bool tya_task_has_ready(void) {
  tya_task_wake_sleepers();
  return tya_ready_head != NULL;
}

double tya_task_next_wake_delay(double max_seconds) {
  tya_task_wake_sleepers();
  if (tya_ready_head != NULL) return 0.0;
  if (tya_sleep_head == NULL) return max_seconds;
  double delay = tya_sleep_head->wake_time - tya_now_seconds();
  if (delay < 0.0) return 0.0;
  if (delay > max_seconds) return max_seconds;
  return delay;
}

void tya_task_run_ready(void) {
  tya_scheduler_run_one();
}

static void tya_task_yield(bool requeue) {
  TyaTask *t = tya_current_task_ptr;
  if (t == NULL) return;
  if (requeue) tya_task_enqueue(t);
  swapcontext(&t->ctx, &tya_scheduler_ctx);
}

static void tya_scheduler_run_one(void) {
  TyaTask *t = tya_task_dequeue();
  if (t == NULL) return;
  tya_scheduler_ctx_valid = true;
  TyaTask *prev = tya_current_task_ptr;
  tya_current_task_ptr = t;
  swapcontext(&tya_scheduler_ctx, &t->ctx);
  tya_current_task_ptr = prev;
}

static void tya_scheduler_run_until_task_done(TyaTask *t) {
  while (t != NULL && !t->done) {
    tya_task_wake_sleepers();
    if (tya_ready_head != NULL) {
      tya_scheduler_run_one();
      continue;
    }
    if (tya_sleep_head == NULL) break;
    double delay = tya_sleep_head->wake_time - tya_now_seconds();
    if (delay <= 0.0) continue;
    struct timespec req;
    req.tv_sec = (time_t)floor(delay);
    req.tv_nsec = (long)((delay - floor(delay)) * 1.0e9);
    nanosleep(&req, NULL);
  }
}

static void tya_task_wake_waiters(TyaTask *t) {
  TyaTask *w = t->next_waiter;
  t->next_waiter = NULL;
  while (w != NULL) {
    TyaTask *next = w->next_waiter;
    w->next_waiter = NULL;
    w->waiting = false;
    tya_task_enqueue(w);
    w = next;
  }
}

static void tya_task_finish(TyaTask *t, TyaValue result, bool raised) {
  t->result = result;
  t->raise_value = raised ? result : tya_nil();
  t->raised = raised;
  t->done = true;
  t->joined = true;
  tya_task_wake_waiters(t);
  tya_live_tasks_remove(t);
}

static void tya_task_fiber_main(uintptr_t raw) {
  TyaTask *t = (TyaTask *)raw;
  tya_current_task_ptr = t;
  TyaRaiseFrame frame;
  frame.prev = NULL;
  if (setjmp(frame.env) == 0) {
    tya_push_raise_frame(&frame);
    TyaValue result = tya_task_invoke(t->callee, t->argc, t->argv);
    tya_pop_raise_frame();
    tya_task_finish(t, result, false);
  } else {
    /* The body raised; capture the value and propagate it from the
     * awaiter. The raise frame is the one this task pushed, so it has
     * already been longjmp'd back to. Remove it before returning to
     * the scheduler so await re-raises into the awaiter's frame. */
    tya_pop_raise_frame();
    tya_task_finish(t, frame.value, true);
  }
  tya_current_task_ptr = NULL;
  setcontext(&tya_scheduler_ctx);
}

static void tya_task_fiber_trampoline(uint32_t raw_lo, uint32_t raw_hi) {
  uintptr_t raw = (uintptr_t)raw_lo;
#if UINTPTR_MAX > UINT32_MAX
  raw |= ((uintptr_t)raw_hi << 32);
#else
  (void)raw_hi;
#endif
  tya_task_fiber_main(raw);
}

/* Per-thread chain of structured-concurrency scopes. tya_task_new
 * registers each new task in the innermost scope (if any) so that
 * tya_scope_exit can wait for it before returning. */
static _Thread_local TyaScope *tya_current_scope = NULL;

