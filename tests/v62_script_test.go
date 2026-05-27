package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestV62Scripts(t *testing.T) {
	t.Parallel()
	skipShort(t, "native package integration txtar suite is covered by the release gate")

	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	modCache := goEnv(t, "GOMODCACHE")
	goCache := goEnv(t, "GOCACHE")
	testscript.RunT(limitedScriptT{T: t}, testscript.Params{
		Dir: "testdata/v62_native",
		Setup: func(env *testscript.Env) error {
			env.Setenv("REPO", repo)
			env.Setenv("GOCACHE", goCache)
			env.Setenv("GOMODCACHE", modCache)
			if zig := os.Getenv("TYA_ZIG"); zig != "" {
				env.Setenv("TYA_ZIG", zig)
			}
			if zigDir := os.Getenv("TYA_ZIG_DIR"); zigDir != "" {
				env.Setenv("TYA_ZIG_DIR", zigDir)
			}
			return nil
		},
	})
}
