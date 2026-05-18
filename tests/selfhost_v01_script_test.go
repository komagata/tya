package tests

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	stage := buildSelfhostScriptStage1(t, repo, "v01")
	testscript.RunT(limitedSelfhostV01T{T: t}, testscript.Params{
		Dir: "testdata/v01_selfhost",
		Setup: func(env *testscript.Env) error {
			env.Setenv("REPO", repo)
			env.Setenv("GOCACHE", goCache)
			env.Setenv("GOMODCACHE", modCache)
			env.Setenv("SELFHOST_V01_STAGE1_C", stage.cPath)
			env.Setenv("SELFHOST_V01_STAGE1_BIN", stage.binPath)
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

type selfhostScriptStage1 struct {
	cPath   string
	binPath string
}

func buildSelfhostScriptStage1(t *testing.T, repo, version string) selfhostScriptStage1 {
	t.Helper()

	key := selfhostScriptStage1CacheKey(t, repo, version)
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		cacheRoot = t.TempDir()
	}
	dir := filepath.Join(cacheRoot, "tya", "selfhost-stage1", version+"-"+key)
	cPath := filepath.Join(dir, "stage1.c")
	binPath := filepath.Join(dir, "stage1")
	if fileExists(cPath) && fileExists(binPath) {
		return selfhostScriptStage1{cPath: cPath, binPath: binPath}
	}

	tmpDir := dir + ".tmp-" + strings.ReplaceAll(t.Name(), "/", "-")
	if err := os.RemoveAll(tmpDir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tmpCPath := filepath.Join(tmpDir, "stage1.c")
	tmpBinPath := filepath.Join(tmpDir, "stage1")
	source := filepath.Join("selfhost", version, "compiler.tya")

	emit := exec.Command("go", "run", "./cmd/tya", "run", source, source)
	emit.Dir = repo
	emit.Env = append(os.Environ(), "TYA_LEGACY_MODULES=1")
	var emitStderr bytes.Buffer
	cFile, err := os.Create(tmpCPath)
	if err != nil {
		t.Fatal(err)
	}
	emit.Stdout = cFile
	emit.Stderr = &emitStderr
	emitErr := emit.Run()
	closeErr := cFile.Close()
	if emitErr != nil {
		t.Fatalf("build %s stage1 C: %v\n%s", version, emitErr, emitStderr.String())
	}
	if closeErr != nil {
		t.Fatalf("close %s stage1 C: %v", version, closeErr)
	}

	compile := exec.Command("cc", tmpCPath, filepath.Join(repo, "runtime", "tya_runtime.c"), "-I", filepath.Join(repo, "runtime"), "-o", tmpBinPath, "-lpthread", "-lm", "-lz")
	compile.Dir = repo
	compileOut, err := compile.CombinedOutput()
	if err != nil {
		t.Fatalf("compile %s stage1: %v\n%s", version, err, compileOut)
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmpDir, dir); err != nil {
		if fileExists(cPath) && fileExists(binPath) {
			if removeErr := os.RemoveAll(tmpDir); removeErr != nil {
				t.Fatal(removeErr)
			}
			return selfhostScriptStage1{cPath: cPath, binPath: binPath}
		}
		if removeErr := os.RemoveAll(dir); removeErr != nil {
			t.Fatal(removeErr)
		}
		if renameErr := os.Rename(tmpDir, dir); renameErr != nil {
			t.Fatal(renameErr)
		}
	}

	return selfhostScriptStage1{cPath: cPath, binPath: binPath}
}

func selfhostScriptStage1CacheKey(t *testing.T, repo, version string) string {
	t.Helper()

	var files []string
	for _, root := range []string{
		filepath.Join(repo, "cmd"),
		filepath.Join(repo, "internal"),
		filepath.Join(repo, "runtime"),
		filepath.Join(repo, "selfhost", version),
	} {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == ".tya" {
					return filepath.SkipDir
				}
				return nil
			}
			switch filepath.Ext(path) {
			case ".go", ".c", ".h", ".tya":
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	sort.Strings(files)

	h := sha256.New()
	for _, path := range files {
		rel, err := filepath.Rel(repo, path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(h, rel+"\n"); err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.Copy(h, f); err != nil {
			_ = f.Close()
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
