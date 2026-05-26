package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestRoadmapDocumentsLatestSelfhostReleaseGate(t *testing.T) {
	roadmap := readRepoFile(t, "ROADMAP.md")
	for _, required := range []string{
		"latest self-host compiler line is the v1.0.0 release-critical self-host gate",
		"`selfhost/v01` remains maintained legacy regression coverage",
		"does not define v1.0.0 language validity or release readiness",
		"`scripts/dev_selfhost_smoke.sh` as the fast self-host smoke gate",
		"`scripts/release_gate.sh`",
	} {
		if !containsNormalized(roadmap, required) {
			t.Fatalf("ROADMAP.md missing latest self-host release-gate text %q", required)
		}
	}
}

func TestSpecDocumentsV1AuthorityOrder(t *testing.T) {
	for _, path := range [][]string{
		{"docs", "SPEC.md"},
		{"docs", "v1.0", "SPEC.md"},
	} {
		spec := readRepoFile(t, path...)
		for _, required := range []string{
			"Specification authority for v1.0.0 is, in order",
			"this specification and the frozen `docs/v1.0/SPEC.md`",
			"latest self-host compiler behavior",
			"Go implementation as reference and bootstrap recovery path",
			"behavior matching the v1 specification is authoritative",
		} {
			if !containsNormalized(spec, required) {
				t.Fatalf("%s missing v1 authority-order text %q", filepath.Join(path...), required)
			}
		}
	}
}

func TestSpecDocumentsV1SyntaxEvolutionPolicy(t *testing.T) {
	for _, path := range [][]string{
		{"docs", "SPEC.md"},
		{"docs", "v1.0", "SPEC.md"},
	} {
		spec := readRepoFile(t, path...)
		for _, required := range []string{
			"New syntax is not added in ordinary v1.x releases",
			"v1.x may add standard-library APIs, package APIs, tooling improvements, diagnostics, and compatible runtime behavior",
			"New syntax requires v2 planning or an explicitly accepted experimental feature path",
			"outside the stable v1 public contract",
		} {
			if !containsNormalized(spec, required) {
				t.Fatalf("%s missing v1 syntax-evolution text %q", filepath.Join(path...), required)
			}
		}
	}
}

func TestDiagnosticReferenceCoversPublicCodes(t *testing.T) {
	reference := readRepoFile(t, "docs", "DIAGNOSTICS.md")
	for _, code := range publicDiagnosticCodes(t) {
		if !strings.Contains(reference, code) {
			t.Fatalf("docs/DIAGNOSTICS.md missing public diagnostic code %s", code)
		}
	}
}

func TestReleaseGateUsesLatestSelfhost(t *testing.T) {
	script := readRepoFile(t, "scripts", "release_gate.sh")
	for _, required := range []string{
		"TestSelfhostV02Scripts",
		"TestBootstrapNoGoSelfhostV02FixedPoint",
		"go test ./... -count=1",
	} {
		if !strings.Contains(script, required) {
			t.Fatalf("release_gate.sh missing %q", required)
		}
	}
	if strings.Contains(script, "TestSelfhostV01Scripts") {
		t.Fatal("release_gate.sh must use the latest selfhost line, not v01")
	}
}

func TestDevelopmentGateCanUseFastSelfhostSmoke(t *testing.T) {
	dev := readRepoFile(t, "scripts", "dev_selfhost_smoke.sh")
	release := readRepoFile(t, "scripts", "release_gate.sh")
	for _, required := range []string{
		"TestSelfhostV02Scripts/fixed_point",
		"-timeout=20m",
	} {
		if !strings.Contains(dev, required) {
			t.Fatalf("dev_selfhost_smoke.sh missing %q", required)
		}
	}
	if dev == release || strings.Contains(dev, "TestBootstrapNoGoSelfhostV02FixedPoint") || strings.Contains(dev, "go test ./...") {
		t.Fatal("development selfhost smoke gate is not distinct from the full release gate")
	}
}

func TestPublicV1RejectsLegacySyntaxOutsideLegacyPaths(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"module", "module old\n", "module declarations were removed"},
		{"instance_variable", "class Counter\n  initialize: ->\n    @count = 0\n", "is removed; use self."},
		{"removed_helper", "print(len([1]))\n", "top-level builtin len was removed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "legacy.tya")
			if strings.HasPrefix(tc.src, "class ") {
				path = filepath.Join(dir, "Counter.tya")
			}
			if err := os.WriteFile(path, []byte(tc.src), 0644); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("go", "run", "./cmd/tya", "check", path)
			cmd.Dir = ".."
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("expected legacy syntax rejection for %s", tc.name)
			}
			if !strings.Contains(string(out), tc.want) || !strings.Contains(string(out), "TYA-E") {
				t.Fatalf("legacy diagnostic missing %q and stable code:\n%s", tc.want, out)
			}
		})
	}
}

func TestV1StdlibBlockersRemainImplemented(t *testing.T) {
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
	want := "1\ntrue\ntrue\n0\ntrue\nok\ntrue\n"
	run := exec.Command("go", "run", "./cmd/tya", "run", main)
	run.Dir = ".."
	run.Env = append(os.Environ(), "TYA_STDLIB_DIR="+filepath.Join(repoRoot(t), "stdlib"))
	if out, err := run.CombinedOutput(); err != nil || string(out) != want {
		t.Fatalf("tya run stdlib blockers: %v\nout=%q\nwant=%q", err, out, want)
	}

	bin := filepath.Join(dir, "v1-stdlib")
	build := exec.Command("go", "run", "./cmd/tya", "build", "--target", "native", main, "-o", bin)
	build.Dir = ".."
	build.Env = append(os.Environ(), "TYA_STDLIB_DIR="+filepath.Join(repoRoot(t), "stdlib"))
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("tya build stdlib blockers: %v\n%s", err, out)
	}
	if out, err := exec.Command(bin).CombinedOutput(); err != nil || string(out) != want {
		t.Fatalf("generated binary stdlib blockers: %v\nout=%q\nwant=%q", err, out, want)
	}

	for _, required := range []string{
		"`regex/Regex`",
		"`file/File` and `dir/Dir`",
		"`time/Time`",
		"`os/Os`",
		"`process/Process`",
		"`hmac/Hmac`",
		"v1.0.0 stdlib blocker set",
	} {
		if !containsNormalized(readRepoFile(t, "docs", "SPEC.md"), required) {
			t.Fatalf("SPEC.md missing stdlib blocker documentation %q", required)
		}
	}
}

func TestPackageManagerV1Contract(t *testing.T) {
	spec := readRepoFile(t, "docs", "SPEC.md")
	strict := readRepoFile(t, "docs", "STRICT_SEMANTICS.md")
	for _, required := range []string{
		"Unknown top-level keys are errors",
		"Git and explicit local path dependencies are supported",
		"`tya.lock` records resolved dependency sources, revisions, and content hashes",
		"Native package metadata lives under `[native]`",
		"no central package registry and no `tya publish` command",
	} {
		if !containsNormalized(spec, required) {
			t.Fatalf("SPEC.md missing package-manager v1 contract text %q", required)
		}
	}
	for _, required := range []string{
		"internal/pkg.TestManifestRejectsUnknownKeys",
		"internal/pkg.TestManifestAcceptsExplicitDependenciesOnly",
		"internal/pkg.TestNativePackageRejectsBuildScripts",
		"internal/runner.TestDependencyHashMismatchFails",
		"tests.TestPublishCommandUnavailable",
	} {
		if !strings.Contains(strict, required) {
			t.Fatalf("STRICT_SEMANTICS.md missing package-manager test coverage %q", required)
		}
	}
}

func publicDiagnosticCodes(t *testing.T) []string {
	t.Helper()
	re := regexp.MustCompile(`TYA-E[0-9]{4}`)
	seen := map[string]bool{}
	for _, root := range []string{"internal", "cmd"} {
		cmd := exec.Command("rg", "-o", `TYA-E[0-9]{4}`, root)
		cmd.Dir = ".."
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("rg diagnostic codes in %s: %v", root, err)
		}
		for _, code := range re.FindAllString(string(out), -1) {
			seen[code] = true
		}
	}
	out := make([]string, 0, len(seen))
	for code := range seen {
		out = append(out, code)
	}
	sort.Strings(out)
	return out
}

func TestV1ReleaseGatePolicyTxtarExists(t *testing.T) {
	path := filepath.Join("..", "tests", "testdata", "v65_strict", "v1_release_gate_policy.txtar")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"scripts/dev_selfhost_smoke.sh",
		"scripts/release_gate.sh",
		"docs/DIAGNOSTICS.md",
		"legacy_module.tya",
	} {
		if !bytes.Contains(data, []byte(required)) {
			t.Fatalf("v1_release_gate_policy.txtar missing %q", required)
		}
	}
}
