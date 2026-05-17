package tests

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestSelfhostV01Scripts(t *testing.T) {
	t.Parallel()

	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	modCache := goEnvForSelfhostV01(t, "GOMODCACHE")
	goCache := goEnvForSelfhostV01(t, "GOCACHE")
	testscript.RunT(limitedSelfhostV01T{T: t}, testscript.Params{
		Dir: "testdata/v01_selfhost",
		Setup: func(env *testscript.Env) error {
			env.Setenv("REPO", repo)
			env.Setenv("GOCACHE", goCache)
			env.Setenv("GOMODCACHE", modCache)
			return nil
		},
	})
}

type limitedSelfhostV01T struct {
	*testing.T
}

func (t limitedSelfhostV01T) Parallel() {
	t.T.Parallel()
	selfhostScriptParallel <- struct{}{}
	t.T.Cleanup(func() {
		<-selfhostScriptParallel
	})
}

func (t limitedSelfhostV01T) Run(name string, f func(testscript.T)) {
	t.T.Run(name, func(st *testing.T) {
		f(limitedSelfhostV01T{T: st})
	})
}

func (t limitedSelfhostV01T) Verbose() bool {
	return testing.Verbose()
}

func goEnvForSelfhostV01(t *testing.T, key string) string {
	t.Helper()
	out, err := exec.Command("go", "env", key).Output()
	if err != nil {
		t.Fatalf("go env %s: %v", key, err)
	}
	return strings.TrimSpace(string(out))
}
