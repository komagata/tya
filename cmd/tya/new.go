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
// lib template: src/<name>/<PascalName>.tya (directory package) plus the
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
	if err := writeScaffold(target, name, opts.template); err != nil {
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

func writeScaffold(target, name, template string) error {
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
		return writeLibTemplate(target, name)
	}
	return writeAppTemplate(target, name)
}

func writeAppTemplate(target, name string) error {
	main := `print("Hello, Tya!")
`
	if err := os.WriteFile(filepath.Join(target, "src", "main.tya"), []byte(main), 0644); err != nil {
		return err
	}
	mainTest := `import unittest

class MainTest < TestCase
  test_main = ->
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
  initialize = ->
    self.name = "%s"

  greet = target ->
    return "Hello, " + target + " from " + self.name
`, pascal, name)
	if err := os.WriteFile(filepath.Join(pkgDir, pascal+".tya"), []byte(src), 0644); err != nil {
		return err
	}
	test := fmt.Sprintf(`import %s
import unittest

class %sTest extends TestCase
  test_greet = ->
    inst = %s()
    self.assert_equal("Hello, Tya from %s", inst.greet("Tya"), "greeting")
`, pkgName, pascal, pascal, name)
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
