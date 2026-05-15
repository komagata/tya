package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const managedZigVersion = "0.16.0"

type zigToolchain struct {
	Path    string
	Version string
	Source  string
}

func resolveZigToolchain() (zigToolchain, error) {
	candidates := zigCandidates()
	for _, c := range candidates {
		if c.path == "" {
			continue
		}
		toolchain, err := readZigToolchain(c.path, c.source)
		if err == nil {
			return toolchain, nil
		}
		if c.required {
			return zigToolchain{}, managedZigError(err)
		}
	}
	return zigToolchain{}, managedZigError(nil)
}

type zigCandidate struct {
	path     string
	source   string
	required bool
}

func zigCandidates() []zigCandidate {
	var candidates []zigCandidate
	if path := os.Getenv("TYA_ZIG"); path != "" {
		candidates = append(candidates, zigCandidate{path: path, source: "TYA_ZIG", required: true})
	}
	if dir := os.Getenv("TYA_ZIG_DIR"); dir != "" {
		candidates = append(candidates, zigCandidate{path: filepath.Join(dir, zigExecutableName()), source: "TYA_ZIG_DIR", required: true})
	}
	if path := managedZigPath(); path != "" {
		candidates = append(candidates, zigCandidate{path: path, source: "managed"})
	}
	if path, err := exec.LookPath("zig"); err == nil {
		candidates = append(candidates, zigCandidate{path: path, source: "PATH"})
	}
	return candidates
}

func managedZigPath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	prefix := filepath.Dir(filepath.Dir(exe))
	return filepath.Join(prefix, "zig", managedZigVersion, zigExecutableName())
}

func zigExecutableName() string {
	if runtime.GOOS == "windows" {
		return "zig.exe"
	}
	return "zig"
}

func readZigToolchain(path string, source string) (zigToolchain, error) {
	out, err := exec.Command(path, "version").Output()
	if err != nil {
		return zigToolchain{}, err
	}
	version := strings.TrimSpace(string(out))
	if version == "" {
		return zigToolchain{}, fmt.Errorf("empty Zig version output")
	}
	return zigToolchain{Path: path, Version: version, Source: source}, nil
}

func zigCommand(path string, args ...string) *exec.Cmd {
	cmd := exec.Command(path, args...)
	if os.Getenv("ZIG_GLOBAL_CACHE_DIR") == "" {
		cmd.Env = append(os.Environ(), "ZIG_GLOBAL_CACHE_DIR="+filepath.Join(os.TempDir(), "tya-zig-cache"))
	}
	return cmd
}

func managedZigError(cause error) error {
	msg := fmt.Sprintf("managed Zig %s is required; reinstall or repair Tya", managedZigVersion)
	if cause != nil {
		return fmt.Errorf("%s: %w", msg, cause)
	}
	return fmt.Errorf("%s", msg)
}
