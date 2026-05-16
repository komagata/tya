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
	if cc := os.Getenv("CC"); cc != "" {
		if path, err := exec.LookPath(cc); err == nil {
			fmt.Fprintf(os.Stdout, "Native compiler: %s (CC escape hatch)\n", path)
		} else {
			fmt.Fprintf(os.Stdout, "Native compiler: %s (CC escape hatch, not found)\n", cc)
		}
	} else {
		if zig, err := resolveZigToolchain(); err == nil {
			fmt.Fprintf(os.Stdout, "Native compiler: managed zig cc\n")
			fmt.Fprintf(os.Stdout, "Zig: %s\n", zig.Path)
			fmt.Fprintf(os.Stdout, "Zig version: %s\n", zig.Version)
			fmt.Fprintf(os.Stdout, "Zig source: %s\n", zig.Source)
		} else {
			fmt.Fprintln(os.Stdout, "Native compiler: managed zig cc (not found)")
			return fmt.Errorf("tya doctor native: %w", err)
		}
	}
	if path, err := exec.LookPath("pkg-config"); err == nil {
		fmt.Fprintf(os.Stdout, "pkg-config: %s\n", path)
		if err := exec.Command("pkg-config", "--exists", "openssl").Run(); err == nil {
			fmt.Fprintln(os.Stdout, "OpenSSL: available")
		} else {
			fmt.Fprintln(os.Stdout, "OpenSSL: not found by pkg-config")
		}
	} else {
		fmt.Fprintln(os.Stdout, "pkg-config: not found")
		if shouldEnableOpenSSL() {
			fmt.Fprintln(os.Stdout, "OpenSSL: available")
		} else {
			fmt.Fprintln(os.Stdout, "OpenSSL: not found")
		}
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
	zig, err := resolveZigToolchain()
	if err != nil {
		fmt.Fprintln(os.Stdout, "Zig: managed zig cc (not found)")
		return fmt.Errorf("tya doctor wasm: %w", err)
	}
	fmt.Fprintf(os.Stdout, "Zig: %s\n", zig.Path)
	fmt.Fprintf(os.Stdout, "Zig version: %s\n", zig.Version)
	fmt.Fprintf(os.Stdout, "Zig source: %s\n", zig.Source)
	fmt.Fprintln(os.Stdout, "Supported Tya wasm targets:")
	fmt.Fprintln(os.Stdout, "  wasm32-wasi")
	fmt.Fprintln(os.Stdout, "  wasm32-browser")
	return nil
}
