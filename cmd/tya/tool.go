package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"tya/internal/pkg"
	"tya/internal/runner"
)

type toolOptions struct {
	List    bool
	Offline bool
	Path    string
	Git     string
	Tag     string
	Rev     string
	Branch  string
	Command string
	Args    []string
}

type toolEntry struct {
	Command string
	Package string
	Root    string
	Path    string
}

func toolCommand(args []string) (int, error) {
	opts, err := parseToolArgs(args)
	if err != nil {
		return 2, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return 1, err
	}
	projectRoot, manifestPath, err := pkg.FindManifest(cwd)
	if err != nil {
		return 1, fmt.Errorf("[TYA-E0940] tya tool requires a tya.toml project")
	}
	if opts.Path != "" || opts.Git != "" {
		entry, err := resolveOneShotTool(projectRoot, opts)
		if err != nil {
			return 1, err
		}
		return runToolEntry(projectRoot, entry, opts.Args)
	}

	m, err := pkg.ReadManifest(manifestPath)
	if err != nil {
		return 1, err
	}
	lf, err := readFreshToolLockfile(projectRoot, m)
	if err != nil {
		return 1, err
	}
	entries, err := lockedToolEntries(projectRoot, lf)
	if err != nil {
		return 1, err
	}
	if opts.List {
		listToolEntries(entries)
		return 0, nil
	}
	entry, err := selectToolEntry(entries, opts.Command)
	if err != nil {
		return 1, err
	}
	return runToolEntry(projectRoot, entry, opts.Args)
}

func parseToolArgs(args []string) (toolOptions, error) {
	opts := toolOptions{}
	for len(args) > 0 {
		a := args[0]
		if a == "--" {
			args = args[1:]
			break
		}
		if !strings.HasPrefix(a, "--") {
			break
		}
		switch a {
		case "--list":
			opts.List = true
			args = args[1:]
		case "--offline":
			opts.Offline = true
			args = args[1:]
		case "--path":
			if len(args) < 2 {
				return opts, fmt.Errorf("missing value for --path")
			}
			opts.Path = args[1]
			args = args[2:]
		case "--git":
			if len(args) < 2 {
				return opts, fmt.Errorf("missing value for --git")
			}
			opts.Git = args[1]
			args = args[2:]
		case "--tag":
			if len(args) < 2 {
				return opts, fmt.Errorf("missing value for --tag")
			}
			opts.Tag = args[1]
			args = args[2:]
		case "--rev":
			if len(args) < 2 {
				return opts, fmt.Errorf("missing value for --rev")
			}
			opts.Rev = args[1]
			args = args[2:]
		case "--branch":
			if len(args) < 2 {
				return opts, fmt.Errorf("missing value for --branch")
			}
			opts.Branch = args[1]
			args = args[2:]
		default:
			return opts, fmt.Errorf("unknown tool option: %s", a)
		}
	}
	if opts.Path != "" && opts.Git != "" {
		return opts, fmt.Errorf("tya tool accepts only one of --path or --git")
	}
	if opts.Branch != "" {
		return opts, fmt.Errorf("[TYA-E0946] tya tool --git rejects branches; use --tag or --rev")
	}
	if opts.Git != "" && opts.Tag == "" && opts.Rev == "" {
		return opts, fmt.Errorf("[TYA-E0946] tya tool --git requires --tag or --rev")
	}
	if opts.List {
		if len(args) > 0 {
			return opts, fmt.Errorf("tya tool --list does not accept a command")
		}
		return opts, nil
	}
	if len(args) == 0 {
		return opts, fmt.Errorf("usage: tya tool [--list] [--offline] [--path P | --git URL (--tag T|--rev R)] <command> [args...]")
	}
	opts.Command = args[0]
	opts.Args = args[1:]
	return opts, nil
}

func readFreshToolLockfile(projectRoot string, m *pkg.Manifest) (*pkg.Lockfile, error) {
	lockPath := filepath.Join(projectRoot, pkg.LockfileName)
	lf, err := pkg.ReadLockfile(lockPath)
	if err != nil {
		return nil, fmt.Errorf("[TYA-E0941] tya.lock is missing or unreadable; run `tya install`")
	}
	if !lf.SatisfiesManifest(m) {
		return nil, fmt.Errorf("[TYA-E0941] tya.lock is stale; run `tya install`")
	}
	return lf, nil
}

func lockedToolEntries(projectRoot string, lf *pkg.Lockfile) ([]toolEntry, error) {
	entries := []toolEntry{}
	for i := range lf.Packages {
		lp := &lf.Packages[i]
		root := pkg.PackageDir(projectRoot, lp)
		m, err := pkg.ReadManifest(filepath.Join(root, pkg.ManifestName))
		if err != nil {
			return nil, fmt.Errorf("[TYA-E0942] package %s is not materialized; run `tya install`", lp.Name)
		}
		for _, name := range m.ToolOrder {
			entry, err := makeToolEntry(m.Name, root, name, m.Tools[name])
			if err != nil {
				return nil, err
			}
			entries = append(entries, entry)
		}
	}
	sortToolEntries(entries)
	return entries, nil
}

func resolveOneShotTool(projectRoot string, opts toolOptions) (toolEntry, error) {
	root := opts.Path
	if root != "" {
		if !filepath.IsAbs(root) {
			root = filepath.Join(projectRoot, root)
		}
	} else {
		var err error
		root, err = fetchOneShotGit(projectRoot, opts)
		if err != nil {
			return toolEntry{}, err
		}
	}
	m, err := pkg.ReadManifest(filepath.Join(root, pkg.ManifestName))
	if err != nil {
		return toolEntry{}, err
	}
	path, ok := m.Tools[opts.Command]
	if !ok {
		return toolEntry{}, fmt.Errorf("[TYA-E0944] tool %q is not declared by package %s", opts.Command, m.Name)
	}
	return makeToolEntry(m.Name, root, opts.Command, path)
}

func fetchOneShotGit(projectRoot string, opts toolOptions) (string, error) {
	ref := opts.Tag
	if ref == "" {
		ref = opts.Rev
	}
	root := filepath.Join(projectRoot, ".tya", "cache", "exec", safeToolKey(opts.Git+"@"+ref))
	if opts.Offline {
		if _, err := os.Stat(filepath.Join(root, pkg.ManifestName)); err != nil {
			return "", fmt.Errorf("[TYA-E0947] cached tool package is unavailable offline")
		}
		return root, nil
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); os.IsNotExist(err) {
		_ = os.RemoveAll(root)
		if err := os.MkdirAll(filepath.Dir(root), 0755); err != nil {
			return "", err
		}
		cmd := exec.Command("git", "clone", "--quiet", opts.Git, root)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("git clone %s: %v: %s", opts.Git, err, string(out))
		}
	} else {
		_ = exec.Command("git", "-C", root, "fetch", "--quiet", "--all", "--tags").Run()
	}
	cmd := exec.Command("git", "-C", root, "checkout", "--quiet", "--detach", ref)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git checkout %s: %v: %s", ref, err, string(out))
	}
	return root, nil
}

func makeToolEntry(packageName, root, command, rel string) (toolEntry, error) {
	if err := validateToolPath(rel); err != nil {
		return toolEntry{}, fmt.Errorf("[TYA-E0943] tool %s:%s: %v", packageName, command, err)
	}
	path := filepath.Join(root, rel)
	if _, err := os.Stat(path); err != nil {
		return toolEntry{}, fmt.Errorf("[TYA-E0943] tool %s:%s missing script %s", packageName, command, rel)
	}
	return toolEntry{Command: command, Package: packageName, Root: root, Path: path}, nil
}

func validateToolPath(rel string) error {
	clean := filepath.Clean(rel)
	if filepath.IsAbs(rel) || clean == "." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) || clean == ".." {
		return fmt.Errorf("path must be relative to the package root")
	}
	if err := runner.ValidateFileName(rel); err != nil {
		return err
	}
	return nil
}

func selectToolEntry(entries []toolEntry, command string) (toolEntry, error) {
	pkgName, toolName, qualified := strings.Cut(command, ":")
	if !qualified {
		toolName = pkgName
	}
	matches := []toolEntry{}
	for _, entry := range entries {
		if entry.Command != toolName {
			continue
		}
		if qualified && entry.Package != pkgName {
			continue
		}
		matches = append(matches, entry)
	}
	if len(matches) == 0 {
		return toolEntry{}, fmt.Errorf("[TYA-E0944] no package tool %q found", command)
	}
	if len(matches) > 1 {
		pkgs := []string{}
		for _, entry := range matches {
			pkgs = append(pkgs, entry.Package)
		}
		sort.Strings(pkgs)
		return toolEntry{}, fmt.Errorf("[TYA-E0945] tool %q is declared by multiple packages: %s; use package:tool", toolName, strings.Join(pkgs, ", "))
	}
	return matches[0], nil
}

func runToolEntry(projectRoot string, entry toolEntry, args []string) (int, error) {
	abs, err := filepath.Abs(entry.Path)
	if err != nil {
		return 1, err
	}
	prevPath := os.Getenv("TYA_PATH")
	toolSrc := filepath.Join(entry.Root, "src")
	nextPath := toolSrc
	if prevPath != "" {
		nextPath += string(os.PathListSeparator) + prevPath
	}
	if err := os.Setenv("TYA_PATH", nextPath); err != nil {
		return 1, err
	}
	defer os.Setenv("TYA_PATH", prevPath)
	err = compileAndRunInDir(abs, args, projectRoot)
	if err == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), nil
	}
	printDiagnostic(abs, err)
	return 1, nil
}

func listToolEntries(entries []toolEntry) {
	for _, entry := range entries {
		fmt.Fprintf(os.Stdout, "%s\t%s\n", entry.Command, entry.Package)
	}
}

func sortToolEntries(entries []toolEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Command != entries[j].Command {
			return entries[i].Command < entries[j].Command
		}
		return entries[i].Package < entries[j].Package
	})
}

func safeToolKey(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}
