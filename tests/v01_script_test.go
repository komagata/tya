package tests

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestV01Scripts(t *testing.T) {
	t.Parallel()

	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	modCache := goEnv(t, "GOMODCACHE")
	testscript.RunT(limitedScriptT{T: t}, testscript.Params{
		Dir: "testdata/v01",
		Setup: func(env *testscript.Env) error {
			env.Setenv("REPO", repo)
			env.Setenv("GOCACHE", filepath.Join(env.WorkDir, ".gocache"))
			env.Setenv("GOMODCACHE", modCache)
			setupTyaGoWrapper(t, env, repo)
			return nil
		},
	})
}

func goEnv(t *testing.T, key string) string {
	t.Helper()
	out, err := exec.Command("go", "env", key).Output()
	if err != nil {
		t.Fatalf("go env %s: %v", key, err)
	}
	return strings.TrimSpace(string(out))
}
