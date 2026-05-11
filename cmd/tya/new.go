package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// newCommand implements `tya new <name>`. It creates a directory
// named <name> in the current working directory and writes a minimal
// project scaffold:
//
//   <name>/
//     tya.toml      # name, version, sample [tasks] table
//     src/main.tya  # hello-world entry point
//     .gitignore    # .tya/ and dist/
//
// The target directory must not already exist. v0.49 keeps the
// surface intentionally small: no --template, no --here, no --force,
// no automatic `git init` (add these in a follow-up minor when the
// need is concrete).
func newCommand(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: tya new <name>")
	}
	name := args[0]
	if name == "" || strings.ContainsRune(name, os.PathSeparator) {
		return fmt.Errorf("[TYA-E0910] invalid project name %q", name)
	}
	target := filepath.Join(".", name)
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("[TYA-E0911] %s already exists", target)
	}
	if err := os.MkdirAll(filepath.Join(target, "src"), 0755); err != nil {
		return err
	}
	manifest := fmt.Sprintf(`name = "%s"
version = "0.1.0"

[tasks]
run = "tya run src/main.tya"
`, name)
	if err := os.WriteFile(filepath.Join(target, "tya.toml"), []byte(manifest), 0644); err != nil {
		return err
	}
	main := `print("Hello, Tya!")
`
	if err := os.WriteFile(filepath.Join(target, "src", "main.tya"), []byte(main), 0644); err != nil {
		return err
	}
	gitignore := ".tya/\ndist/\n"
	if err := os.WriteFile(filepath.Join(target, ".gitignore"), []byte(gitignore), 0644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Created %s\n", target)
	fmt.Fprintf(os.Stdout, "  cd %s && tya task run\n", name)
	return nil
}
