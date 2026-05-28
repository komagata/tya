package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// newCommand implements `tya new [flags] <name>`.
//
//	flags:
//	  --here              initialise the current directory (no <name>)
//	  --template app|lib  template (default: app)
//	  --native            add native C wrapper scaffold (lib template only)
//	  --force             overwrite an existing target
//	  --no-git            skip the default `git init` step
//
// app template:
//
//	<name>/
//	  tya.toml          (with sample [tasks] table)
//	  src/main.tya
//	  tests/main_test.tya
//	  .gitignore
//	  README.md
//
// lib template: src/<name>/<name>.tya (directory package) plus the
// corresponding tests/<name>_test.tya. Other files are identical.
func newCommand(args []string) error {
	opts, err := parseNewArgs(args)
	if err != nil {
		return err
	}
	target, name, err := resolveTarget(opts)
	if err != nil {
		return err
	}
	if !opts.here && !opts.force {
		if _, err := os.Stat(target); err == nil {
			return fmt.Errorf("[TYA-E0911] %s already exists", target)
		}
	}
	if err := writeScaffold(target, name, opts.template, opts.native); err != nil {
		return err
	}
	if !opts.noGit {
		runGitInit(target)
	}
	if opts.here {
		fmt.Fprintf(os.Stdout, "Initialised tya project in %s\n", target)
	} else {
		task := "run"
		if opts.template == "lib" {
			task = "test"
		}
		fmt.Fprintf(os.Stdout, "Created %s\n", target)
		fmt.Fprintf(os.Stdout, "  cd %s && tya task %s\n", name, task)
	}
	return nil
}

type newOpts struct {
	target   string
	here     bool
	template string
	force    bool
	noGit    bool
	native   bool
}

func parseNewArgs(args []string) (newOpts, error) {
	opts := newOpts{template: "app"}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--here":
			opts.here = true
		case "--force":
			opts.force = true
		case "--no-git":
			opts.noGit = true
		case "--native":
			opts.native = true
		case "--template":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --template")
			}
			opts.template = args[i+1]
			i++
		default:
			if strings.HasPrefix(a, "--template=") {
				opts.template = strings.TrimPrefix(a, "--template=")
			} else if strings.HasPrefix(a, "-") {
				return opts, fmt.Errorf("unknown option: %s", a)
			} else if opts.target == "" {
				opts.target = a
			} else {
				return opts, fmt.Errorf("unexpected argument: %s", a)
			}
		}
	}
	if opts.template != "app" && opts.template != "lib" {
		return opts, fmt.Errorf("[TYA-E0912] invalid --template %q (must be app or lib)", opts.template)
	}
	if opts.native && opts.template != "lib" {
		return opts, fmt.Errorf("[TYA-E0914] --native requires --template lib")
	}
	if opts.here && opts.target != "" {
		return opts, fmt.Errorf("[TYA-E0913] --here cannot be combined with a target name")
	}
	if !opts.here && opts.target == "" {
		return opts, fmt.Errorf("usage: tya new <name> | tya new --here")
	}
	if opts.target != "" && strings.ContainsRune(opts.target, os.PathSeparator) {
		return opts, fmt.Errorf("[TYA-E0910] invalid project name %q", opts.target)
	}
	return opts, nil
}

func resolveTarget(opts newOpts) (target, name string, err error) {
	if opts.here {
		cwd, werr := os.Getwd()
		if werr != nil {
			return "", "", werr
		}
		return cwd, filepath.Base(cwd), nil
	}
	return filepath.Join(".", opts.target), opts.target, nil
}

func writeScaffold(target, name, template string, native bool) error {
	if err := os.MkdirAll(filepath.Join(target, "src"), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(target, "tests"), 0755); err != nil {
		return err
	}
	tasks := `run = "tya run src/main.tya"
test = "tya test tests"
`
	if template == "lib" {
		tasks = `test = "TYA_PATH=src tya test tests"
`
	}
	manifest := fmt.Sprintf(`name = "%s"
version = "0.1.0"

[tasks]
%s`, name, tasks)
	if err := os.WriteFile(filepath.Join(target, "tya.toml"), []byte(manifest), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(target, ".gitignore"), []byte(".tya/\ndist/\n"), 0644); err != nil {
		return err
	}
	readme := fmt.Sprintf("# %s\n\nA Tya project. Run `tya task run` to execute, `tya task test` to run tests.\n", name)
	if template == "lib" {
		readme = fmt.Sprintf("# %s\n\nA Tya library package. Run `tya task test` to run tests.\n", name)
	}
	if err := os.WriteFile(filepath.Join(target, "README.md"), []byte(readme), 0644); err != nil {
		return err
	}
	if template == "lib" {
		if native {
			return writeNativeLibTemplate(target, name)
		}
		return writeLibTemplate(target, name)
	}
	return writeAppTemplate(target, name)
}

func writeNativeLibTemplate(target, name string) error {
	pascal := pascalCase(name)
	pkgName := packageName(name)
	pkgDir := filepath.Join(target, "src", pkgName)
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(target, "native"), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(target, "include"), 0755); err != nil {
		return err
	}
	manifest := fmt.Sprintf(`name = "%s"
version = "0.1.0"

[native]
sources = ["native/%s.c"]
headers = ["include/%s.h"]
include_dirs = ["include"]
cflags = []
ldflags = []

[native.functions]
%s_version = { symbol = "tya_%s_version", arity = 0 }

[tasks]
test = "TYA_PATH=src tya test tests"
`, name, pkgName, pkgName, pkgName, pkgName)
	if err := os.WriteFile(filepath.Join(target, "tya.toml"), []byte(manifest), 0644); err != nil {
		return err
	}
	src := fmt.Sprintf(`class %s
  static version: ->
    return %s_version()
`, pascal, pkgName)
	if err := os.WriteFile(filepath.Join(pkgDir, pkgName+".tya"), []byte(src), 0644); err != nil {
		return err
	}
	header := fmt.Sprintf(`#ifndef TYA_%s_H
#define TYA_%s_H

#include "tya_runtime.h"

TyaValue tya_%s_version(TyaValue __this, TyaValue a0, TyaValue a1, TyaValue a2, TyaValue a3);

#endif
`, strings.ToUpper(pkgName), strings.ToUpper(pkgName), pkgName)
	if err := os.WriteFile(filepath.Join(target, "include", pkgName+".h"), []byte(header), 0644); err != nil {
		return err
	}
	csrc := fmt.Sprintf(`#include "%s.h"

TyaValue tya_%s_version(TyaValue __this, TyaValue a0, TyaValue a1, TyaValue a2, TyaValue a3) {
  (void)__this;
  (void)a0;
  (void)a1;
  (void)a2;
  (void)a3;
  return tya_string("0.1.0");
}
`, pkgName, pkgName)
	if err := os.WriteFile(filepath.Join(target, "native", pkgName+".c"), []byte(csrc), 0644); err != nil {
		return err
	}
	test := fmt.Sprintf(`import %s/*
import unittest/* as unittest

class %sTest extends TestCase
  test_version: ->
    self.assert_equal("0.1.0", %s.%s.version(), "native version")
`, pkgName, pascal, pkgName, pascal)
	return os.WriteFile(filepath.Join(target, "tests", pkgName+"_test.tya"), []byte(test), 0644)
}

func writeAppTemplate(target, name string) error {
	main := `print("Hello, Tya!")
`
	if err := os.WriteFile(filepath.Join(target, "src", "main.tya"), []byte(main), 0644); err != nil {
		return err
	}
	mainTest := `import unittest/* as unittest

class MainTest < TestCase
  test_main: ->
    self.assert(true, "main")
`
	return os.WriteFile(filepath.Join(target, "tests", "main_test.tya"), []byte(mainTest), 0644)
}

func writeLibTemplate(target, name string) error {
	pascal := pascalCase(name)
	pkgName := packageName(name)
	pkgDir := filepath.Join(target, "src", pkgName)
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		return err
	}
	src := fmt.Sprintf(`class %s
  name: ""

  initialize: ->
    self.name = "%s"

  greet: target ->
    return "Hello, " + target + " from " + self.name
`, pascal, name)
	if err := os.WriteFile(filepath.Join(pkgDir, pkgName+".tya"), []byte(src), 0644); err != nil {
		return err
	}
	test := fmt.Sprintf(`import %s/*
import unittest/* as unittest

class %sTest extends TestCase
  test_greet: ->
    inst = %s.%s()
    self.assert_equal("Hello, Tya from %s", inst.greet("Tya"), "greeting")
`, pkgName, pascal, pkgName, pascal, name)
	return os.WriteFile(filepath.Join(target, "tests", pkgName+"_test.tya"), []byte(test), 0644)
}

func pascalCase(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_' || r == '.'
	})
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

func packageName(name string) string {
	name = strings.ToLower(name)
	var b strings.Builder
	prevUnderscore := false
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevUnderscore = false
			continue
		}
		if !prevUnderscore {
			b.WriteByte('_')
			prevUnderscore = true
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "project"
	}
	return out
}

func runGitInit(target string) {
	cmd := exec.Command("git", "init", "--quiet", target)
	cmd.Stdout = nil
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: skipped `git init` (%v)\n", err)
	}
}
