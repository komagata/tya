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
#ifdef __APPLE__
#ifndef _DARWIN_C_SOURCE
#define _DARWIN_C_SOURCE
#endif
#endif
#ifdef __clang__
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
#endif

#include "tya_runtime.h"

#include <ctype.h>
#include <dirent.h>
#include <errno.h>
#include <fcntl.h>
#include <locale.h>
#include <math.h>
#include <pthread.h>
#ifndef _WIN32
#include <regex.h>
#else
typedef struct { size_t re_nsub; } regex_t;
typedef struct { int rm_so; int rm_eo; } regmatch_t;
#define REG_EXTENDED 0
#define REG_ICASE 0
#define REG_NEWLINE 0
static int regcomp(regex_t *re, const char *pattern, int flags) { (void)re; (void)pattern; (void)flags; return 1; }
static int regexec(const regex_t *re, const char *text, size_t nmatch, regmatch_t matches[], int flags) { (void)re; (void)text; (void)nmatch; (void)matches; (void)flags; return 1; }
static void regfree(regex_t *re) { (void)re; }
#endif
#include <signal.h>
#include <stdatomic.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/time.h>
#include <sys/types.h>
#ifndef _WIN32
#include <sys/wait.h>
#endif
#include <time.h>
#ifndef _WIN32
#include <ucontext.h>
#endif
#include <unistd.h>
#ifdef TYA_ENABLE_ZLIB
#include <zlib.h>
#endif
#ifdef TYA_ENABLE_OPENSSL
#include <openssl/err.h>
#include <openssl/ssl.h>
#endif

extern char **environ;

static char *tya_dup_cstr(const char *s);
static int tya_legacy_modules_enabled(void);

#ifdef _WIN32
#include <winsock2.h>
#include <ws2tcpip.h>
typedef SOCKET TyaSocketHandle;
#define TYA_INVALID_SOCKET INVALID_SOCKET
#else
#include <arpa/inet.h>
#include <netdb.h>
#include <netinet/in.h>
#include <sys/socket.h>
#include <sys/syscall.h>
typedef int TyaSocketHandle;
#define TYA_INVALID_SOCKET (-1)
#endif

#ifdef __APPLE__
#ifndef NI_MAXHOST
#define NI_MAXHOST 1025
#endif
#ifndef NI_MAXSERV
#define NI_MAXSERV 32
#endif
extern char *mkdtemp(char *);
extern int mkstemps(char *, int);
extern time_t timegm(struct tm *);
#endif

#if defined(__APPLE__) || defined(__FreeBSD__) || defined(__OpenBSD__)
#include <sys/random.h>
#endif


#include "tya_runtime_core.c"
#include "tya_runtime_collections.c"
#include "tya_runtime_io.c"
#include "tya_runtime_compiler_regex.c"
#include "tya_runtime_crypto_bytes.c"
#include "tya_runtime_net.c"
#include "tya_runtime_task_sync.c"
