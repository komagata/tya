package tests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

var scriptParallel = make(chan struct{}, 8)
var tyaGoWrapperOnce sync.Once
var tyaGoWrapperDir string
var tyaGoWrapperErr error

type limitedScriptT struct {
	*testing.T
}

func (t limitedScriptT) Parallel() {
	t.T.Parallel()
	scriptParallel <- struct{}{}
	t.T.Cleanup(func() {
		<-scriptParallel
	})
}

func (t limitedScriptT) Run(name string, f func(testscript.T)) {
	t.T.Run(name, func(st *testing.T) {
		f(limitedScriptT{T: st})
	})
}

func (t limitedScriptT) Verbose() bool {
	return testing.Verbose()
}

func setupTyaGoWrapper(t *testing.T, env *testscript.Env, repo string) {
	t.Helper()
	dir, err := tyaGoWrapper(t, repo)
	if err != nil {
		t.Fatalf("build tya test wrapper: %v", err)
	}
	env.Setenv("PATH", dir+string(os.PathListSeparator)+env.Getenv("PATH"))
}

func tyaGoWrapper(t *testing.T, repo string) (string, error) {
	t.Helper()
	tyaGoWrapperOnce.Do(func() {
		realGo, err := exec.LookPath("go")
		if err != nil {
			tyaGoWrapperErr = err
			return
		}
		dir, err := os.MkdirTemp("", "tya-test-go-wrapper-*")
		if err != nil {
			tyaGoWrapperErr = err
			return
		}
		bin := filepath.Join(dir, "tya")
		cmd := exec.Command(realGo, "build", "-o", bin, "./cmd/tya")
		cmd.Dir = repo
		out, err := cmd.CombinedOutput()
		if err != nil {
			tyaGoWrapperErr = fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
			return
		}
		wrapper := filepath.Join(dir, "go")
		script := fmt.Sprintf(`#!/bin/sh
if [ "$1" = "run" ] && [ "$2" = "./cmd/tya" ]; then
  shift 2
  exec %q "$@"
fi
exec %q "$@"
`, bin, realGo)
		if err := os.WriteFile(wrapper, []byte(script), 0755); err != nil {
			tyaGoWrapperErr = err
			return
		}
		tyaGoWrapperDir = dir
	})
	return tyaGoWrapperDir, tyaGoWrapperErr
}
