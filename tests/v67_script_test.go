package tests

import (
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestV67Scripts(t *testing.T) {
	t.Parallel()

	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	modCache := goEnv(t, "GOMODCACHE")
	goCache := goEnv(t, "GOCACHE")
	testscript.RunT(limitedScriptT{T: t}, testscript.Params{
		Dir: "testdata/v67_callable_object",
		Setup: func(env *testscript.Env) error {
			env.Setenv("REPO", repo)
			env.Setenv("GOCACHE", goCache)
			env.Setenv("GOMODCACHE", modCache)
			setupTyaGoWrapper(t, env, repo)
			return nil
		},
	})
}
