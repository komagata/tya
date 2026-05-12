#ifndef TYA_HTTP_SERVER_H
#define TYA_HTTP_SERVER_H

#include "tya_runtime.h"

// tya_http_server_run starts a single-threaded HTTP/1.1 server on
// `port` (0 = let the OS pick a free port) and dispatches each
// request to the first matching route.
//
// `routes` is a Tya array of dicts shaped:
//   {method: "GET", path: "/users/:id", handler: <fn>}
//
// The handler receives a request dict:
//   {method, path, params, query, headers, body}
//
// and must return a response dict:
//   {status: 200, headers: {...}, body: "..." | <bytes>}
//
// The function blocks forever (or until SIGINT). It always returns
// tya_nil() on graceful exit.
//
// When port == 0, the chosen port is printed to stderr as
// "listening on <port>\n" so test harnesses can latch onto it.
TyaValue tya_http_server_run(TyaValue routes, TyaValue port);

#endif
