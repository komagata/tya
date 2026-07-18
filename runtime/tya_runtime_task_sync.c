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
    tya_scheduler_run_until_task_done(t);
    if (t->raised && !had_raise) {
      first_raise = t->raise_value;
      had_raise = true;
    }
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
    tya_scheduler_run_until_task_done(t);
  }
  free(scope->tasks);
  scope->tasks = NULL;
  scope->len = 0;
  scope->cap = 0;
  tya_current_scope = scope->prev;
  tya_raise(value);
}

/* Add a freshly created task to the live-tasks list so the task is reachable
 * as a root from the moment it exists. */
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
  t->done = false;
  t->joined = false;
  t->raised = false;
  t->queued = false;
  t->waiting = false;
  atomic_store(&t->cancelled, false);
  t->callee = callee;
  t->argc = argc;
  t->argv[0] = a;
  t->argv[1] = b;
  t->argv[2] = c;
  t->argv[3] = d;
  t->result = tya_nil();
  t->raise_value = tya_nil();
  t->pending_value = tya_nil();
  t->waiting_value = tya_nil();
  t->gc_roots = NULL;
  t->raise_frame = NULL;
  t->channel_send_failed = false;
  t->sleeping = false;
  t->wake_time = 0.0;
  t->prev_live = NULL;
  t->next_live = NULL;
  t->next_ready = NULL;
  t->next_sleep = NULL;
  t->next_waiter = NULL;
  t->next_channel_waiter = NULL;
  t->in_live_list = false;
  t->stack_size = 256 * 1024;
  t->stack = malloc(t->stack_size);
  if (t->stack == NULL) {
    tya_raise(tya_string("spawn: task stack allocation failed"));
    return tya_nil();
  }
  getcontext(&t->ctx);
  t->ctx.uc_stack.ss_sp = t->stack;
  t->ctx.uc_stack.ss_size = t->stack_size;
  t->ctx.uc_link = &tya_scheduler_ctx;
  uintptr_t raw = (uintptr_t)t;
  makecontext(&t->ctx, (void (*)(void))tya_task_fiber_trampoline, 2, (uint32_t)raw, (uint32_t)(raw >> 32));
  tya_live_tasks_add(t);
  tya_scope_register_task(t);
  tya_task_enqueue(t);
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
  if (!t->done) {
    TyaTask *current = tya_current_task_ptr;
    if (current != NULL) {
      current->waiting = true;
      current->next_waiter = t->next_waiter;
      t->next_waiter = current;
      tya_task_yield(false);
    } else {
      tya_scheduler_run_until_task_done(t);
    }
  }
  bool raised = t->raised;
  TyaValue value = raised ? t->raise_value : t->result;
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
  c->recv_waiters = NULL;
  c->send_waiters = NULL;
  return (TyaValue){.kind = TYA_CHANNEL, .channel = c};
}

static void tya_channel_waiter_push(TyaTask **head, TyaTask *t) {
  t->next_channel_waiter = NULL;
  if (*head == NULL) {
    *head = t;
    return;
  }
  TyaTask *tail = *head;
  while (tail->next_channel_waiter != NULL) tail = tail->next_channel_waiter;
  tail->next_channel_waiter = t;
}

static TyaTask *tya_channel_waiter_pop(TyaTask **head) {
  TyaTask *t = *head;
  if (t == NULL) return NULL;
  *head = t->next_channel_waiter;
  t->next_channel_waiter = NULL;
  t->waiting = false;
  return t;
}

static void tya_channel_wake_one_sender(TyaChannel *c) {
  while (c->send_waiters != NULL && c->len < c->capacity && !c->closed) {
    TyaTask *sender = tya_channel_waiter_pop(&c->send_waiters);
    int tail = (c->head + c->len) % c->capacity;
    c->buffer[tail] = sender->waiting_value;
    c->len++;
    sender->channel_send_failed = false;
    tya_task_enqueue(sender);
  }
}

TyaValue tya_channel_send(TyaValue ch, TyaValue value) {
  if (ch.kind != TYA_CHANNEL || ch.channel == NULL) {
    tya_raise(tya_string("channel.send: first argument must be a channel"));
    return tya_nil();
  }
  TyaChannel *c = ch.channel;
  pthread_mutex_lock(&c->mu);
  if (c->recv_waiters != NULL && c->len == 0 && !c->closed) {
    TyaTask *receiver = tya_channel_waiter_pop(&c->recv_waiters);
    receiver->pending_value = value;
    tya_task_enqueue(receiver);
    pthread_mutex_unlock(&c->mu);
    return tya_nil();
  }
  while (c->len == c->capacity && !c->closed) {
    TyaTask *current = tya_current_task_ptr;
    if (current == NULL) {
      pthread_mutex_unlock(&c->mu);
      tya_scheduler_run_one();
      pthread_mutex_lock(&c->mu);
      continue;
    }
    current->waiting = true;
    current->waiting_value = value;
    current->channel_send_failed = false;
    tya_channel_waiter_push(&c->send_waiters, current);
    pthread_mutex_unlock(&c->mu);
    tya_task_yield(false);
    if (current->channel_send_failed) {
      tya_raise(tya_string("channel.send: channel is closed"));
    }
    return tya_nil();
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
    TyaTask *current = tya_current_task_ptr;
    if (current == NULL) {
      pthread_mutex_unlock(&c->mu);
      tya_scheduler_run_one();
      pthread_mutex_lock(&c->mu);
      if (tya_ready_head == NULL && c->len == 0 && !c->closed) {
        pthread_mutex_unlock(&c->mu);
        return tya_nil();
      }
      continue;
    }
    current->waiting = true;
    tya_channel_waiter_push(&c->recv_waiters, current);
    pthread_mutex_unlock(&c->mu);
    tya_task_yield(false);
    return current->pending_value;
  }
  if (c->len == 0 && c->closed) {
    pthread_mutex_unlock(&c->mu);
    return tya_nil();
  }
  TyaValue value = c->buffer[c->head];
  c->buffer[c->head] = tya_nil();
  c->head = (c->head + 1) % c->capacity;
  c->len--;
  tya_channel_wake_one_sender(c);
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
    if (tya_current_task_ptr != NULL) {
      if (tya_now_seconds() >= ((double)deadline.tv_sec + ((double)deadline.tv_nsec / 1000000000.0))) {
        pthread_mutex_unlock(&c->mu);
        return tya_nil();
      }
      pthread_mutex_unlock(&c->mu);
      tya_task_yield(true);
      pthread_mutex_lock(&c->mu);
      continue;
    }
    if (tya_ready_head != NULL) {
      pthread_mutex_unlock(&c->mu);
      tya_scheduler_run_one();
      pthread_mutex_lock(&c->mu);
      continue;
    }
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

static double tya_now_seconds(void) {
#if defined(__APPLE__)
  struct timeval now;
  gettimeofday(&now, NULL);
  return (double)now.tv_sec + ((double)now.tv_usec / 1000000.0);
#else
  struct timespec now;
  clock_gettime(CLOCK_REALTIME, &now);
  return (double)now.tv_sec + ((double)now.tv_nsec / 1000000000.0);
#endif
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
  int default_index = -1;
  double start = tya_now_seconds();
  /* Validate every op once before entering the polling loop. */
  for (int i = 0; i < n; i++) {
    TyaValue op = ops.array->items[i];
    if (op.kind != TYA_ARRAY || op.array == NULL || op.array->len < 2) {
      tya_raise(tya_string("channel.select: operation must be [channel, \"receive\"] or [channel, \"send\", value]"));
      return tya_nil();
    }
    TyaValue ch = op.array->items[0];
    TyaValue kind_v = op.array->items[1];
    if (kind_v.kind != TYA_STRING) {
      tya_raise(tya_string("channel.select: operation kind must be \"receive\", \"send\", \"timeout\", or \"default\""));
      return tya_nil();
    }
    bool is_timeout = strcmp(kind_v.string, "timeout") == 0;
    bool is_default = strcmp(kind_v.string, "default") == 0;
    if (!is_timeout && !is_default && (ch.kind != TYA_CHANNEL || ch.channel == NULL)) {
      tya_raise(tya_string("channel.select: operation channel must be a channel"));
      return tya_nil();
    }
    bool is_receive = strcmp(kind_v.string, "receive") == 0;
    bool is_send = strcmp(kind_v.string, "send") == 0;
    if (!is_receive && !is_send && !is_timeout && !is_default) {
      tya_raise(tya_string("channel.select: operation kind must be \"receive\", \"send\", \"timeout\", or \"default\""));
      return tya_nil();
    }
    if (is_send && op.array->len < 3) {
      tya_raise(tya_string("channel.select: send operation must include the value"));
      return tya_nil();
    }
    if (is_timeout) {
      if (op.array->len < 3 || op.array->items[2].kind != TYA_NUMBER || op.array->items[2].number < 0.0) {
        tya_raise(tya_string("channel.select: timeout operation must include non-negative seconds"));
        return tya_nil();
      }
    }
    if (is_default) {
      if (default_index >= 0) {
        tya_raise(tya_string("channel.select: at most one default operation is allowed"));
        return tya_nil();
      }
      default_index = i;
    }
  }
  /* Try each operation non-blocking, then let the scheduler run any ready
   * tasks before falling back to a short host sleep at top level. */
  while (true) {
    for (int i = 0; i < n; i++) {
      TyaValue op = ops.array->items[i];
      const char *kind_s = op.array->items[1].string;
      if (strcmp(kind_s, "timeout") == 0 || strcmp(kind_s, "default") == 0) {
        continue;
      }
      TyaChannel *c = op.array->items[0].channel;
      if (strcmp(kind_s, "receive") == 0) {
        pthread_mutex_lock(&c->mu);
        if (c->len > 0) {
          TyaValue v = c->buffer[c->head];
          c->buffer[c->head] = tya_nil();
          c->head = (c->head + 1) % c->capacity;
          c->len--;
          tya_channel_wake_one_sender(c);
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
        if (c->recv_waiters != NULL && c->len == 0) {
          TyaTask *receiver = tya_channel_waiter_pop(&c->recv_waiters);
          receiver->pending_value = value;
          tya_task_enqueue(receiver);
          pthread_mutex_unlock(&c->mu);
          TyaDictEntry entries[3] = {
            {"index", tya_number((double)i)},
            {"kind", tya_string("send")},
            {"value", tya_nil()},
          };
          return tya_dict(entries, 3);
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
    if (default_index >= 0) {
      TyaDictEntry entries[3] = {
        {"index", tya_number((double)default_index)},
        {"kind", tya_string("default")},
        {"value", tya_nil()},
      };
      return tya_dict(entries, 3);
    }
    double elapsed = tya_now_seconds() - start;
    for (int i = 0; i < n; i++) {
      TyaValue op = ops.array->items[i];
      const char *kind_s = op.array->items[1].string;
      if (strcmp(kind_s, "timeout") != 0) continue;
      if (elapsed >= op.array->items[2].number) {
        TyaDictEntry entries[3] = {
          {"index", tya_number((double)i)},
          {"kind", tya_string("timeout")},
          {"value", tya_nil()},
        };
        return tya_dict(entries, 3);
      }
    }
    if (tya_ready_head != NULL) {
      tya_scheduler_run_one();
      continue;
    }
    if (tya_current_task_ptr != NULL) {
      tya_task_yield(true);
      continue;
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
  while (c->recv_waiters != NULL) {
    TyaTask *receiver = tya_channel_waiter_pop(&c->recv_waiters);
    receiver->pending_value = tya_nil();
    tya_task_enqueue(receiver);
  }
  while (c->send_waiters != NULL) {
    TyaTask *sender = tya_channel_waiter_pop(&c->send_waiters);
    sender->channel_send_failed = true;
    tya_task_enqueue(sender);
  }
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
  r->stream = NULL;
  r->stream_borrowed = false;
  r->stream_binary = false;
  r->stream_readable = false;
  r->stream_writable = false;
  r->stream_closed = false;
  r->socket_fd = -1;
  r->socket_binary = false;
  r->socket_closed = false;
  r->socket_timeout = 0.0;
  r->tls_ssl = NULL;
  r->tls_ctx = NULL;
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
      case TYA_RES_STREAM: expected = "stream"; break;
      case TYA_RES_SOCKET: expected = "socket"; break;
      case TYA_RES_SOCKET_SERVER: expected = "socket_server"; break;
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
    if (tya_ready_head != NULL) {
      pthread_mutex_unlock(&r->mu);
      tya_scheduler_run_one();
      pthread_mutex_lock(&r->mu);
      continue;
    }
    pthread_cond_wait(&r->cv, &r->mu);
  }
  pthread_mutex_unlock(&r->mu);
  return tya_nil();
}
