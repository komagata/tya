package tests

import (
	"os"
	"strings"
	"testing"
)

func TestReleasePackageInstallersInstallManagedZig(t *testing.T) {
	data, err := os.ReadFile("../scripts/build_release_packages.sh")
	if err != nil {
		t.Fatal(err)
	}
	script := string(data)

	for _, want := range []string{
		`zig_version="${TYA_ZIG_VERSION:-0.16.0}"`,
		`zig_bin="$prefix/zig/$zig_version/zig"`,
		`https://ziglang.org/download/$zig_version/$zig_name.tar.xz`,
		`Managed Zig installed: $zig_bin`,
		`$zigVersion = if ($env:TYA_ZIG_VERSION)`,
		`$zigBin = Join-Path $prefix "zig\$zigVersion\zig.exe"`,
		`https://ziglang.org/download/$zigVersion/$zigName.zip`,
		`Managed Zig installed: $zigBin`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("scripts/build_release_packages.sh missing %q", want)
		}
	}
}
