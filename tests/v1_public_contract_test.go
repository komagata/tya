package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tya/internal/runner"
)

func TestV1StdlibBlockersImplemented(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "main.tya")
	src := strings.Join([]string{
		"import regex as regex",
		"import file as file",
		"import dir as dir",
		"import time as time",
		"import os as os",
		"import process as process",
		"import hmac as hmac",
		"print(regex.Regex(\"[0-9]+\").search(\"v1\")[\"text\"])",
		"tmp = file.File().temp(\"tya-v1\", \".txt\")",
		"print(file.File(tmp).exists?())",
		"file.File(tmp).remove()",
		"root = dir.Dir(nil).temp_dir(\"tya-v1\")",
		"print(file.File(root).exists?())",
		"dir.Dir(root).remove_all()",
		"print(time.Time().unix(0).format(\"unix\"))",
		"print(os.Os().env(\"TYA_NO_SUCH_V1_ENV\") == nil)",
		"print(process.Process([\"sh\", \"-c\", \"printf ok\"]).run({})[\"stdout\"])",
		"print(hmac.Hmac(\"sha256\", \"key\").verify(\"The quick brown fox jumps over the lazy dog\", \"f7bc83f430538424b13298e6aa6fb143ef4d59a14946175997479dbc2d1a3cd8\"))",
		"",
	}, "\n")
	if err := os.WriteFile(main, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TYA_STDLIB_DIR", filepath.Join(repoRoot(t), "stdlib"))
	out := strings.Builder{}
	if _, err := runner.RunFile(main, strings.NewReader(""), &out, nil); err != nil {
		t.Fatal(err)
	}
	want := "1\ntrue\ntrue\n0\ntrue\nok\ntrue\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestPlatformDependentStdlibImportsEverywhere(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "main.tya")
	src := strings.Join([]string{
		"import net/socket as socket",
		"import process as process",
		"print(true)",
		"",
	}, "\n")
	if err := os.WriteFile(main, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TYA_STDLIB_DIR", filepath.Join(repoRoot(t), "stdlib"))
	out := strings.Builder{}
	if _, err := runner.RunFile(main, strings.NewReader(""), &out, nil); err != nil {
		t.Fatal(err)
	}
	if out.String() != "true\n" {
		t.Fatalf("got %q", out.String())
	}
}

func TestReleaseGateIsRepoInternal(t *testing.T) {
	spec := readTestFile(t, "docs", "SPEC.md")
	if !strings.Contains(spec, "`tya conformance` command") {
		t.Fatal("SPEC.md must keep v1 conformance as a repo-internal release gate")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func readTestFile(t *testing.T, elems ...string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(append([]string{".."}, elems...)...))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
