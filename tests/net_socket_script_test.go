package tests

import (
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestNetSocketScripts(t *testing.T) {
	t.Parallel()

	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	modCache := goEnv(t, "GOMODCACHE")
	testscript.RunT(limitedScriptT{T: t}, testscript.Params{
		Dir: "testdata/net_socket",
		Setup: func(env *testscript.Env) error {
			env.Setenv("REPO", repo)
			env.Setenv("GOCACHE", filepath.Join(env.WorkDir, ".gocache"))
			env.Setenv("GOMODCACHE", modCache)
			setupTyaGoWrapper(t, env, repo)
			return nil
		},
	})
}
