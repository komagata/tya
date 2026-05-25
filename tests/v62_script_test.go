package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestV62Scripts(t *testing.T) {
	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	modCache := goEnv(t, "GOMODCACHE")
	goCache := goEnv(t, "GOCACHE")
	testscript.Run(t, testscript.Params{
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
