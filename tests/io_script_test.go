package tests

import (
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestIoScripts(t *testing.T) {
	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	modCache := goEnv(t, "GOMODCACHE")
	testscript.Run(t, testscript.Params{
		Dir: "testdata/io",
		Setup: func(env *testscript.Env) error {
			env.Setenv("REPO", repo)
			env.Setenv("GOCACHE", filepath.Join(env.WorkDir, ".gocache"))
			env.Setenv("GOMODCACHE", modCache)
			return nil
		},
	})
}
