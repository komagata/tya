package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"tya/internal/pkg"
)

func doctorCommand(args []string) error {
	if len(args) != 1 || (args[0] != "native" && args[0] != "wasm") {
		return fmt.Errorf("usage: tya doctor native|wasm")
	}
	if args[0] == "wasm" {
		return doctorWasm()
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

func doctorWasm() error {
	path, err := exec.LookPath("zig")
	if err != nil {
		fmt.Fprintln(os.Stdout, "Zig: not found")
		return fmt.Errorf("tya doctor wasm: Zig is required for wasm targets")
	}
	fmt.Fprintf(os.Stdout, "Zig: %s\n", path)
	out, err := exec.Command(path, "version").Output()
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Zig version: %s", string(out))
	fmt.Fprintln(os.Stdout, "Supported Tya wasm targets:")
	fmt.Fprintln(os.Stdout, "  wasm32-wasi")
	fmt.Fprintln(os.Stdout, "  wasm32-browser")
	return nil
}
