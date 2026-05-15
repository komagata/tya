package tests

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

// TestSelfhostV02Scripts is the v0.45 M8 gate for the next-surface
// self-host compiler at selfhost/v02/. In M8.0 v02 is a byte
// equivalent copy of v01 and this gate mirrors TestSelfhostV01Scripts
// on the v0.1 fixed-point program. Subsequent M8.x STEPs grow v02
// toward the v0.44 surface; this gate must remain green at every
// STEP. The v01 gate stays in place until M9 retires the legacy
// `module` keyword.
func TestSelfhostV02Scripts(t *testing.T) {
	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	modCache := goEnvForSelfhostV02(t, "GOMODCACHE")
	goCache := goEnvForSelfhostV02(t, "GOCACHE")
	testscript.RunT(limitedSelfhostV02T{T: t}, testscript.Params{
		Dir: "testdata/v02_selfhost",
		Setup: func(env *testscript.Env) error {
			env.Setenv("REPO", repo)
			env.Setenv("GOCACHE", goCache)
			env.Setenv("GOMODCACHE", modCache)
			return nil
		},
	})
}

type limitedSelfhostV02T struct {
	*testing.T
}

var selfhostV02Parallel = make(chan struct{}, 2)

func (t limitedSelfhostV02T) Parallel() {
	t.T.Parallel()
	selfhostV02Parallel <- struct{}{}
	t.T.Cleanup(func() {
		<-selfhostV02Parallel
	})
}

func (t limitedSelfhostV02T) Run(name string, f func(testscript.T)) {
	t.T.Run(name, func(st *testing.T) {
		f(limitedSelfhostV02T{T: st})
	})
}

func (t limitedSelfhostV02T) Verbose() bool {
	return testing.Verbose()
}

func goEnvForSelfhostV02(t *testing.T, key string) string {
	t.Helper()
	out, err := exec.Command("go", "env", key).Output()
	if err != nil {
		t.Fatalf("go env %s: %v", key, err)
	}
	return strings.TrimSpace(string(out))
}
