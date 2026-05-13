package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"tya/internal/pkg"
)

func doctorCommand(args []string) error {
	if len(args) != 1 || args[0] != "native" {
		return fmt.Errorf("usage: tya doctor native")
	}
	root, err := projectRoot()
	if err != nil {
		return err
	}
	cc := os.Getenv("CC")
	if cc == "" {
		cc = "cc"
	}
	if path, err := exec.LookPath(cc); err == nil {
		fmt.Fprintf(os.Stdout, "C compiler: %s\n", path)
	} else {
		fmt.Fprintf(os.Stdout, "C compiler: %s (not found)\n", cc)
	}
	if path, err := exec.LookPath("pkg-config"); err == nil {
		fmt.Fprintf(os.Stdout, "pkg-config: %s\n", path)
	} else {
		fmt.Fprintln(os.Stdout, "pkg-config: not found")
	}
	plan, err := pkg.CollectNative(root)
	if err != nil {
		return err
	}
	if len(plan.Packages) == 0 {
		fmt.Fprintln(os.Stdout, "Native packages: none")
		return nil
	}
	fmt.Fprintln(os.Stdout, "Native packages:")
	for _, p := range plan.Packages {
		fmt.Fprintf(os.Stdout, "  %s\n", p.Name)
	}
	fmt.Fprintf(os.Stdout, "Sources: %s\n", strings.Join(plan.Sources, " "))
	fmt.Fprintf(os.Stdout, "Include dirs: %s\n", strings.Join(plan.IncludeDirs, " "))
	fmt.Fprintf(os.Stdout, "C flags: %s\n", strings.Join(plan.CFlags, " "))
	fmt.Fprintf(os.Stdout, "LD flags: %s\n", strings.Join(plan.LDFlags, " "))
	return nil
}
