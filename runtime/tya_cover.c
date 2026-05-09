// Tya v0.30 coverage runtime support.
//
// The instrumented C program calls tya_cov_init(N) once at startup
// and tya_cov_inc(id) before each instrumented statement. At process
// exit, an atexit-registered writer dumps an H-record fragment to
// the path in TYA_COVERAGE_FRAGMENT (when set). When the env var is
// unset the writer does nothing, so non-cover invocations of an
// instrumented binary are still cheap and silent.

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static int *tya_cov_table = (int *)0;
static int tya_cov_size = 0;
static int tya_cov_atexit_registered = 0;

static void tya_cov_write(void) {
    const char *path = getenv("TYA_COVERAGE_FRAGMENT");
    if (path == (const char *)0 || path[0] == '\0') {
        return;
    }
    FILE *f = fopen(path, "w");
    if (f == (FILE *)0) {
        return;
    }
    fputs("# tya-cover 1\n", f);
    for (int i = 0; i < tya_cov_size; i++) {
        if (tya_cov_table[i] != 0) {
            fprintf(f, "H %d %d\n", i, tya_cov_table[i]);
        }
    }
    fclose(f);
}

void tya_cov_init(int n) {
    if (tya_cov_table != (int *)0) {
        return;
    }
    tya_cov_size = n;
    if (n <= 0) {
        return;
    }
    tya_cov_table = (int *)calloc((size_t)n, sizeof(int));
    if (tya_cov_table == (int *)0) {
        tya_cov_size = 0;
        return;
    }
    if (!tya_cov_atexit_registered) {
        atexit(tya_cov_write);
        tya_cov_atexit_registered = 1;
    }
}

void tya_cov_inc(int id) {
    if (tya_cov_table == (int *)0 || id < 0 || id >= tya_cov_size) {
        return;
    }
    tya_cov_table[id]++;
}
