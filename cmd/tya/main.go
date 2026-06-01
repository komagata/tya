package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"tya/internal/ast"
	"tya/internal/checker"
	"tya/internal/codegen"
	"tya/internal/cover"
	"tya/internal/diag"
	"tya/internal/formatter"
	"tya/internal/lexer"
	"tya/internal/parser"
	"tya/internal/pkg"
	"tya/internal/runner"
)

const version = "0.72.9"

var cliFormat = diag.FormatHuman
var cliColor = diag.ColorAuto

var lineColErrorRE = regexp.MustCompile(`^(\d+):(\d+):\s*(.*)$`)
var errTestsFailed = errors.New("test failed")

func main() {
	if err := parseGlobalDiagFlags(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	if os.Args[1] == "version" || os.Args[1] == "--version" {
		if err := versionCommand(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		return
	}
	switch os.Args[1] {
	case "run":
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		runPath := os.Args[2]
		runArgs, err := parseRunArgs(os.Args[3:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if runPath == "-" {
			var cleanup func()
			runPath, cleanup, err = stdinSourceFile()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			defer cleanup()
		}
		if err := compileAndRun(runPath, runArgs); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			printDiagnostic(runPath, err)
			os.Exit(1)
		}
		return
	case "build":
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		buildOpts, err := parseBuildArgs(os.Args[2:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if err := buildExecutableTarget(buildOpts.Path, buildOpts.Output, buildOpts.Target); err != nil {
			printDiagnostic(buildOpts.Path, err)
			os.Exit(1)
		}
		return
	case "check":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		path := os.Args[2]
		if path == "-" {
			var cleanup func()
			var err error
			path, cleanup, err = stdinSourceFile()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			defer cleanup()
		}
		if err := checkFile(path); err != nil {
			if !errors.Is(err, errStrictReported) {
				if os.Args[2] == "-" {
					printDiagnostic("<stdin>", err)
				} else {
					printDiagnostic(path, err)
				}
			}
			os.Exit(1)
		}
		return
	case "fmt":
		fmt.Fprintln(os.Stderr, "tya fmt is no longer accepted; use `tya format`")
		os.Exit(2)
		return
	case "format":
		if err := formatCommand(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "emit-c":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		csrc, err := compileToC(os.Args[2])
		if err != nil {
			printDiagnostic(os.Args[2], err)
			os.Exit(1)
		}
		fmt.Fprint(os.Stdout, csrc)
		return
	case "test":
		root := "."
		coverEnabled := false
		profilePath := ""
		coverOpt := coverageCLIOptions{}
		args := os.Args[2:]
		for i := 0; i < len(args); i++ {
			a := args[i]
			switch {
			case a == "--cover":
				coverEnabled = true
			case a == "--include" || a == "--exclude" || a == "--min":
				if i+1 >= len(args) {
					fmt.Fprintf(os.Stderr, "missing value for %s\n", a)
					os.Exit(2)
				}
				if err := coverOpt.add(a, args[i+1]); err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(2)
				}
				i++
			case strings.HasPrefix(a, "--include="):
				coverOpt.include = append(coverOpt.include, strings.TrimPrefix(a, "--include="))
			case strings.HasPrefix(a, "--exclude="):
				coverOpt.exclude = append(coverOpt.exclude, strings.TrimPrefix(a, "--exclude="))
			case strings.HasPrefix(a, "--min="):
				if err := coverOpt.add("--min", strings.TrimPrefix(a, "--min=")); err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(2)
				}
			case strings.HasPrefix(a, "--profile="):
				profilePath = strings.TrimPrefix(a, "--profile=")
			case a == "--profile":
				if i+1 >= len(args) {
					fmt.Fprintln(os.Stderr, "missing value for --profile")
					os.Exit(2)
				}
				profilePath = args[i+1]
				i++
			default:
				if strings.HasPrefix(a, "--") {
					fmt.Fprintf(os.Stderr, "unknown test option: %s\n", a)
					os.Exit(2)
				}
				root = a
			}
		}
		if err := testCommand(root, coverEnabled, profilePath, coverOpt); err != nil {
			if errors.Is(err, errTestsFailed) {
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "cover":
		if err := coverCommand(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "embed":
		if err := embedCommand(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "install":
		if err := installCommand(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "update":
		target := ""
		if len(os.Args) > 2 {
			target = os.Args[2]
		}
		if err := updateCommand(target); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "add":
		if err := addCommand(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "remove":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		if err := removeCommand(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "outdated":
		if err := outdatedCommand(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "clean":
		if err := cleanCommand(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "task":
		code, err := taskCommand(os.Args[2:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(code)
		return
	case "tool":
		code, err := toolCommand(os.Args[2:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(code)
		return
	case "new":
		if err := newCommand(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "doctor":
		if err := doctorCommand(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	case "lint":
		os.Exit(lintCommand(os.Args[2:]))
		return
	case "doc":
		os.Exit(docCommand(os.Args[2:]))
		return
	case "lsp":
		os.Exit(lspCommand(os.Args[2:]))
		return
	}
	dumpTokens := false
	emitC := false
	checkUnused := false
	args := os.Args[1:]
	for len(args) > 0 {
		switch args[0] {
		case "--tokens":
			dumpTokens = true
			args = args[1:]
		case "--emit-c":
			emitC = true
			args = args[1:]
		case "--check-unused":
			checkUnused = true
			args = args[1:]
		default:
			goto doneOptions
		}
	}
doneOptions:
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}
	path := args[0]
	// v0.44: developer flags are read-only — accept both script and
	// class files. Strict entry-only validation (ValidateFileName)
	// only fires when the flag actually requires entry semantics.
	if err := runner.ValidateAnyTyaFileName(path); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !dumpTokens && !emitC && !checkUnused {
		usage()
		os.Exit(2)
	}
	defer checker.SetPermissiveLegacy(runner.IsLegacyV01Path(path))()
	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	source := string(src)
	modules := []string(nil)
	isClassFile, err := runner.IsClassFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if emitC && !checkUnused {
		if isClassFile {
			// v0.44: emit C for a class file in isolation. Skip the
			// entry-only LoadSourceWithModules path and validate
			// the class file via CheckClassFile. The C output is
			// partial (no main()); useful for inspection /
			// debugging.
			toks, errs := lexer.Lex(source)
			if len(errs) > 0 {
				for _, err := range errs {
					fmt.Fprintln(os.Stderr, err)
				}
				os.Exit(1)
			}
			prog, _, err := parser.Parse(toks)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			if err := checker.CheckClassFile(prog, path); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			csrc, _, err := codegen.EmitCWithPath(prog, path)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Fprint(os.Stdout, csrc)
			return
		}
		var origins map[string]map[string]string
		source, modules, origins, err = runner.LoadSourceWithOrigins(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		toks, errs := lexer.Lex(source)
		if len(errs) > 0 {
			for _, err := range errs {
				fmt.Fprintln(os.Stderr, err)
			}
			os.Exit(1)
		}
		prog, _, err := parser.Parse(toks)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		runner.StampOriginFiles(prog, origins)
		if err := checker.CheckWithModules(prog, modules); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		csrc, _, err := codegen.EmitCWithPath(prog, path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Fprint(os.Stdout, csrc)
		return
	}
	var origins map[string]map[string]string
	if checkUnused {
		if isClassFile {
			// v0.44: --check-unused on a class file uses the source
			// already read from disk; entry-only import resolution
			// is skipped because a class file is a library member.
			// The strict pass downstream walks classes + methods
			// the same way it does for entry programs.
		} else {
			source, modules, origins, err = runner.LoadSourceWithOrigins(path)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	}
	toks, errs := lexer.Lex(source)
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
	if dumpTokens {
		for _, tok := range toks {
			fmt.Fprintf(os.Stdout, "%d:%d\t%s\t%q\n", tok.Line, tok.Col, tok.Type, tok.Lexeme)
		}
		return
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	runner.StampOriginFiles(prog, origins)
	if err := checker.CheckWithModules(prog, modules); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if checkUnused {
		if err := checker.CheckUnused(prog); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if !emitC {
			return
		}
	}
	if emitC {
		source, modules, origins, err = runner.LoadSourceWithOrigins(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		toks, errs = lexer.Lex(source)
		if len(errs) > 0 {
			for _, err := range errs {
				fmt.Fprintln(os.Stderr, err)
			}
			os.Exit(1)
		}
		prog, _, err = parser.Parse(toks)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		runner.StampOriginFiles(prog, origins)
		if err := checker.CheckWithModules(prog, modules); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		csrc, _, err := codegen.EmitCWithPath(prog, path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Fprint(os.Stdout, csrc)
		return
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: tya run <file.tya> [args...]")
	fmt.Fprintln(os.Stderr, "       tya build <file.tya> [-o <output>]")
	fmt.Fprintln(os.Stderr, "       tya check <file.tya>")
	fmt.Fprintln(os.Stderr, "       tya format [--check] [-w] <file.tya>")
	fmt.Fprintln(os.Stderr, "       tya emit-c <file.tya>")
	fmt.Fprintln(os.Stderr, "       tya test [--cover [--profile FILE]] [path]")
	fmt.Fprintln(os.Stderr, "       tya cover [--format=human|json] [--profile FILE]")
	fmt.Fprintln(os.Stderr, "       tya install")
	fmt.Fprintln(os.Stderr, "       tya update [package]")
	fmt.Fprintln(os.Stderr, "       tya add <name> [<constraint>] [--git URL --tag T] [--path P] [--dev]")
	fmt.Fprintln(os.Stderr, "       tya remove <name>")
	fmt.Fprintln(os.Stderr, "       tya outdated")
	fmt.Fprintln(os.Stderr, "       tya clean [--packages]")
	fmt.Fprintln(os.Stderr, "       tya new <name>")
	fmt.Fprintln(os.Stderr, "       tya doctor native")
	fmt.Fprintln(os.Stderr, "       tya task [name] [args...]")
	fmt.Fprintln(os.Stderr, "       tya tool [--list] [--offline] [--path P | --git URL (--tag T|--rev R)] <command> [args...]")
	fmt.Fprintln(os.Stderr, "       tya lint [--fix] [--format=text|json|sarif] [paths...]")
	fmt.Fprintln(os.Stderr, "       tya doc [--html <out>] [paths...]")
	fmt.Fprintln(os.Stderr, "       tya lsp [--log <file>]")
	fmt.Fprintln(os.Stderr, "       tya version")
}

func versionCommand(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stdout, version)
		return nil
	}
	if len(args) == 1 && args[0] == "--json" {
		fmt.Fprintf(os.Stdout, "{\"compiler\":\"%s\",\"runtime\":\"%s\",\"spec\":\"1.0.0\",\"selfhost\":\"v01/v02\"}\n", version, version)
		return nil
	}
	return fmt.Errorf("unknown version option: %s", args[0])
}

func parseRunArgs(args []string) ([]string, error) {
	if len(args) == 0 {
		return nil, nil
	}
	if args[0] == "--" {
		return args[1:], nil
	}
	return args, nil
}

func stdinSourceFile() (string, func(), error) {
	src, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", nil, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", nil, err
	}
	path := filepath.Join(cwd, "tya_stdin.tya")
	if err := os.WriteFile(path, src, 0644); err != nil {
		return "", nil, err
	}
	return path, func() { _ = os.Remove(path) }, nil
}

func cleanCommand(args []string) error {
	removePackages := false
	for _, arg := range args {
		if arg != "--packages" {
			return fmt.Errorf("unknown clean option: %s", arg)
		}
		removePackages = true
	}
	root, err := projectRoot()
	if err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(root, ".tya", "build")); err != nil {
		return err
	}
	if removePackages {
		return os.RemoveAll(filepath.Join(root, ".tya", "packages"))
	}
	return nil
}

func compileAndRun(path string, args []string) error {
	return compileAndRunInDir(path, args, "")
}

func compileAndRunInDir(path string, args []string, cwd string) error {
	prevProjectRoot := os.Getenv("TYA_PROJECT_ROOT")
	if cwd != "" {
		if err := os.Setenv("TYA_PROJECT_ROOT", cwd); err != nil {
			return err
		}
		defer os.Setenv("TYA_PROJECT_ROOT", prevProjectRoot)
	}
	outDir, err := os.MkdirTemp("", "tya-run-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(outDir)
	bin := filepath.Join(outDir, "main")
	if err := buildExecutable(path, bin); err != nil {
		return err
	}
	run := exec.Command(bin, args...)
	run.Stdin = os.Stdin
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr
	if cwd != "" {
		run.Dir = cwd
	}
	return run.Run()
}

func buildExecutable(path string, output string) error {
	_, err := buildExecutableWithCoverTarget(path, output, nil, "native")
	return err
}

func buildExecutableWithCover(path string, output string, opt *codegen.CoverageOptions) (*codegen.CoverageRegistry, error) {
	return buildExecutableWithCoverTarget(path, output, opt, "native")
}

func buildExecutableTarget(path string, output string, target string) error {
	_, err := buildExecutableWithCoverTarget(path, output, nil, target)
	return err
}

func buildExecutableWithCoverTarget(path string, output string, opt *codegen.CoverageOptions, target string) (*codegen.CoverageRegistry, error) {
	if target == "" {
		target = "native"
	}
	if target != "native" {
		return buildWasmExecutable(path, output, target)
	}
	csrc, reg, nativePlan, err := compileToCWithCoverNative(path, opt)
	if err != nil {
		return nil, err
	}
	if output == "" {
		output = defaultOutputPath(path)
	}
	outDir := filepath.Join(filepath.Dir(path), ".tya", "build")
	if root, _, err := pkg.FindManifest(filepath.Dir(path)); err == nil {
		outDir = filepath.Join(root, ".tya", "build")
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, err
	}
	cfile := filepath.Join(outDir, "main.c")
	if err := os.WriteFile(cfile, []byte(csrc), 0644); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return nil, err
	}
	runtimeDir, err := findRuntimeDir()
	if err != nil {
		return nil, err
	}
	args := []string{cfile, filepath.Join(runtimeDir, "tya_runtime.c")}
	httpC := filepath.Join(runtimeDir, "tya_http_server.c")
	if _, err := os.Stat(httpC); err == nil {
		args = append(args, httpC)
	}
	if nativePlan != nil {
		args = append(args, nativePlan.Sources...)
		for _, dir := range nativePlan.IncludeDirs {
			args = append(args, "-I", dir)
		}
		args = append(args, nativePlan.CFlags...)
	}
	if opt != nil {
		coverC := filepath.Join(runtimeDir, "tya_cover.c")
		if _, err := os.Stat(coverC); err == nil {
			args = append(args, coverC)
		}
	}
	args = append(args, "-I", runtimeDir, "-o", output)
	// The runtime uses pthread primitives for GC and sync resources. libm
	// provides log2 / exp / sin / cos / atan2 etc.; glibc requires explicit
	// -lm, while macOS rolls it into libSystem so the flag is harmless.
	if runtime.GOOS == "linux" {
		args = append(args, "-lpthread", "-lm")
	} else if runtime.GOOS == "windows" {
		args = append(args, "-lws2_32")
	} else if runtime.GOOS != "windows" {
		args = append(args, "-lm")
	}
	if shouldEnableZlib() {
		args = append(args, "-DTYA_ENABLE_ZLIB", "-lz")
	}
	if shouldEnableOpenSSL() {
		args = append(args, "-DTYA_ENABLE_OPENSSL")
		args = append(args, opensslCompileFlags()...)
		args = append(args, "-lssl", "-lcrypto")
	}
	if nativePlan != nil {
		args = append(args, nativePlan.LDFlags...)
	}
	var compile *exec.Cmd
	if cc := os.Getenv("CC"); cc != "" {
		compile = exec.Command(cc, args...)
	} else if cc := isolatedTestHostCC(); cc != "" {
		compile = exec.Command(cc, args...)
	} else {
		zig, err := resolveZigToolchain()
		if err != nil {
			return nil, err
		}
		compile = zigCommand(zig.Path, append([]string{"cc"}, args...)...)
	}
	compile.Stderr = os.Stderr
	if err := compile.Run(); err != nil {
		return nil, err
	}
	return reg, nil
}

func isolatedTestHostCC() string {
	if os.Getenv("HOME") != "/no-home" {
		return ""
	}
	cc, err := exec.LookPath("cc")
	if err != nil {
		return ""
	}
	return cc
}

func shouldEnableZlib() bool {
	if os.Getenv("TYA_DISABLE_ZLIB") != "" {
		return false
	}
	if os.Getenv("TYA_ENABLE_ZLIB") != "" {
		return true
	}
	if runtime.GOOS == "darwin" {
		return true
	}
	for _, path := range []string{
		"/usr/include/zlib.h",
		"/usr/local/include/zlib.h",
		"/opt/homebrew/include/zlib.h",
		"/home/linuxbrew/.linuxbrew/include/zlib.h",
	} {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

func shouldEnableOpenSSL() bool {
	if os.Getenv("TYA_DISABLE_OPENSSL") != "" {
		return false
	}
	if os.Getenv("TYA_ENABLE_OPENSSL") != "" {
		return true
	}
	if _, err := exec.LookPath("pkg-config"); err == nil {
		if err := exec.Command("pkg-config", "--exists", "openssl").Run(); err == nil {
			return true
		}
	}
	for _, path := range []string{
		"/usr/include/openssl/ssl.h",
		"/usr/local/include/openssl/ssl.h",
		"/opt/homebrew/include/openssl/ssl.h",
		"/home/linuxbrew/.linuxbrew/include/openssl/ssl.h",
	} {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

func opensslCompileFlags() []string {
	flags := []string{}
	if _, err := exec.LookPath("pkg-config"); err == nil {
		if cflags, err := exec.Command("pkg-config", "--cflags", "openssl").Output(); err == nil {
			flags = append(flags, strings.Fields(string(cflags))...)
		}
		if ldflags, err := exec.Command("pkg-config", "--libs", "openssl").Output(); err == nil {
			for _, flag := range strings.Fields(string(ldflags)) {
				if flag == "-lssl" || flag == "-lcrypto" {
					continue
				}
				flags = append(flags, flag)
			}
		}
	}
	for _, dir := range []string{
		"/opt/homebrew/opt/openssl@3/include",
		"/opt/homebrew/include",
		"/usr/local/opt/openssl@3/include",
		"/usr/local/include",
		"/home/linuxbrew/.linuxbrew/opt/openssl@3/include",
		"/home/linuxbrew/.linuxbrew/include",
	} {
		if _, err := os.Stat(filepath.Join(dir, "openssl", "ssl.h")); err == nil {
			if !hasCompileFlag(flags, "-I"+dir) {
				flags = append(flags, "-I", dir)
			}
			break
		}
	}
	for _, dir := range []string{
		"/opt/homebrew/opt/openssl@3/lib",
		"/opt/homebrew/lib",
		"/usr/local/opt/openssl@3/lib",
		"/usr/local/lib",
		"/home/linuxbrew/.linuxbrew/opt/openssl@3/lib",
		"/home/linuxbrew/.linuxbrew/lib",
	} {
		if _, err := os.Stat(filepath.Join(dir, "libssl.dylib")); err == nil {
			if !hasCompileFlag(flags, "-L"+dir) {
				flags = append(flags, "-L", dir)
			}
			break
		}
		if _, err := os.Stat(filepath.Join(dir, "libssl.so")); err == nil {
			if !hasCompileFlag(flags, "-L"+dir) {
				flags = append(flags, "-L", dir)
			}
			break
		}
	}
	return flags
}

func hasCompileFlag(flags []string, compact string) bool {
	splitPrefix := compact[:2]
	splitValue := compact[2:]
	for i, flag := range flags {
		if flag == compact {
			return true
		}
		if flag == splitPrefix && i+1 < len(flags) && flags[i+1] == splitValue {
			return true
		}
	}
	return false
}

func findRuntimeDir() (string, error) {
	if dir := os.Getenv("TYA_RUNTIME_DIR"); dir != "" {
		if runtimeExists(dir) {
			return dir, nil
		}
		return "", fmt.Errorf("TYA_RUNTIME_DIR does not contain tya_runtime.c: %s", dir)
	}

	candidates := []string{filepath.Join("runtime")}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "runtime"),
			filepath.Join(exeDir, "..", "share", "tya", "runtime"),
		)
	}
	for _, dir := range candidates {
		if runtimeExists(dir) {
			return dir, nil
		}
	}
	return "", fmt.Errorf("could not find Tya runtime; set TYA_RUNTIME_DIR")
}

func runtimeExists(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "tya_runtime.c"))
	return err == nil && !info.IsDir()
}

func compileToC(path string) (string, error) {
	csrc, _, err := compileToCWithCover(path, nil)
	return csrc, err
}

func compileToCWithCover(path string, opt *codegen.CoverageOptions) (string, *codegen.CoverageRegistry, error) {
	csrc, reg, _, err := compileToCWithCoverNative(path, opt)
	return csrc, reg, err
}

func compileToCWithCoverNative(path string, opt *codegen.CoverageOptions) (string, *codegen.CoverageRegistry, *pkg.NativePlan, error) {
	defer checker.SetPermissiveLegacy(runner.IsLegacyV01Path(path))()
	source, modules, origins, err := runner.LoadSourceWithOrigins(path)
	if err != nil {
		return "", nil, nil, err
	}
	nativePlan, err := nativePlanForPath(path)
	if err != nil {
		return "", nil, nil, err
	}
	nativeNames := []string{}
	if nativePlan != nil {
		nativeNames = append(nativeNames, nativePlan.FuncOrder...)
	}
	defer checker.SetExtraBuiltinNames(nativeNames)()
	if nativePlan != nil {
		defer codegen.SetNativeFunctions(nativePlan.Functions)()
	} else {
		defer codegen.SetNativeFunctions(nil)()
	}
	toks, errs := lexer.Lex(source)
	if len(errs) > 0 {
		return "", nil, nil, errs[0]
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		return "", nil, nil, err
	}
	runner.StampOriginFiles(prog, origins)
	if err := checker.CheckWithModules(prog, modules); err != nil {
		return "", nil, nil, err
	}
	csrc, reg, _, err := codegen.EmitCWithCoverage(prog, path, opt)
	if err != nil {
		return "", nil, nil, err
	}
	return csrc, reg, nativePlan, nil
}

func nativePlanForPath(path string) (*pkg.NativePlan, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	root, _, err := pkg.FindManifest(filepath.Dir(abs))
	if err != nil {
		if envRoot := os.Getenv("TYA_PROJECT_ROOT"); envRoot != "" {
			root = envRoot
		} else if cwd, cwdErr := os.Getwd(); cwdErr == nil {
			if found, _, findErr := pkg.FindManifest(cwd); findErr == nil {
				root = found
			} else {
				return nil, nil
			}
		} else {
			return nil, nil
		}
	}
	plan, err := pkg.CollectNative(root)
	if err != nil {
		return nil, err
	}
	if len(plan.Packages) == 0 {
		return nil, nil
	}
	return plan, nil
}

type coverageCLIOptions struct {
	include []string
	exclude []string
	min     *float64
}

func (o *coverageCLIOptions) add(flag, value string) error {
	switch flag {
	case "--include":
		o.include = append(o.include, value)
	case "--exclude":
		o.exclude = append(o.exclude, value)
	case "--min":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid --min value: %s", value)
		}
		o.min = &v
	}
	return nil
}

func readCoverageConfig(root string) coverageCLIOptions {
	raw, err := os.ReadFile(filepath.Join(root, "tya.toml"))
	if err != nil {
		return coverageCLIOptions{}
	}
	var out coverageCLIOptions
	inCoverage := false
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inCoverage = line == "[coverage]"
			continue
		}
		if !inCoverage {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case "include":
			out.include = append(out.include, parseTomlStringArray(value)...)
		case "exclude":
			out.exclude = append(out.exclude, parseTomlStringArray(value)...)
		case "minimum":
			if v, err := strconv.ParseFloat(strings.Trim(value, `"`), 64); err == nil {
				out.min = &v
			}
		}
	}
	return out
}

func parseTomlStringArray(value string) []string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
		return nil
	}
	value = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"`)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func testCommand(root string, coverEnabled bool, profilePath string, cli coverageCLIOptions) error {
	files, err := testFiles(root)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	suiteSrc, err := synthesizeTestSuite(files)
	if err != nil {
		return err
	}
	dirSet := map[string]struct{}{}
	for _, f := range files {
		dirSet[filepath.Dir(f)] = struct{}{}
	}
	pathDirs := []string{}
	for d := range dirSet {
		pathDirs = append(pathDirs, d)
	}
	sort.Strings(pathDirs)

	suiteDir, err := os.MkdirTemp("", "tya-test-suite-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(suiteDir)
	suitePath := filepath.Join(suiteDir, "main.tya")
	if err := os.WriteFile(suitePath, []byte(suiteSrc), 0644); err != nil {
		return err
	}

	prevPath := os.Getenv("TYA_PATH")
	prevProjectRoot := os.Getenv("TYA_PROJECT_ROOT")
	combined := strings.Join(pathDirs, string(os.PathListSeparator))
	if prevPath != "" {
		combined = combined + string(os.PathListSeparator) + prevPath
	}
	if err := os.Setenv("TYA_PATH", combined); err != nil {
		return err
	}
	defer os.Setenv("TYA_PATH", prevPath)
	if cwd, err := os.Getwd(); err == nil {
		if project, _, err := pkg.FindManifest(cwd); err == nil {
			if err := os.Setenv("TYA_PROJECT_ROOT", project); err != nil {
				return err
			}
			defer os.Setenv("TYA_PROJECT_ROOT", prevProjectRoot)
		}
	}

	if !coverEnabled {
		if err := compileAndRun(suitePath, nil); err != nil {
			if _, ok := err.(*exec.ExitError); ok {
				return errTestsFailed
			}
			return err
		}
		return nil
	}
	cfg := readCoverageConfig(".")
	include := append([]string{}, cfg.include...)
	include = append(include, cli.include...)
	exclude := append([]string{}, cfg.exclude...)
	exclude = append(exclude, cli.exclude...)
	min := cfg.min
	if cli.min != nil {
		min = cli.min
	}

	// Coverage path: build with instrumentation, run with
	// TYA_COVERAGE_FRAGMENT, then merge fragment with registry.
	covDir := os.Getenv("TYA_COVERAGE_DIR")
	if covDir == "" {
		covDir = filepath.Join(".tya", "coverage")
	}
	if profilePath == "" {
		profilePath = filepath.Join(covDir, "profile")
	}
	fragDir := filepath.Join(covDir, "fragments")
	if err := os.MkdirAll(fragDir, 0o755); err != nil {
		return err
	}
	fragPath := filepath.Join(fragDir, "main.cov")
	_ = os.Remove(fragPath)

	binDir, err := os.MkdirTemp("", "tya-test-bin-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(binDir)
	bin := filepath.Join(binDir, "main")

	opt := &codegen.CoverageOptions{
		StdlibDir:   firstStdlibDir(),
		PackagesDir: firstPackagesDir(),
		Include:     include,
		Exclude:     exclude,
	}
	reg, err := buildExecutableWithCover(suitePath, bin, opt)
	if err != nil {
		return err
	}

	cmd := exec.Command(bin)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "TYA_COVERAGE_FRAGMENT="+fragPath)
	runErr := cmd.Run()

	if err := mergeCoverageFragments(profilePath, fragDir, reg); err != nil {
		return err
	}
	if len(include) > 0 || len(exclude) > 0 || min != nil {
		prof, err := cover.ReadProfile(profilePath)
		if err != nil {
			return err
		}
		filtered := cover.Filter(prof, cover.FilterOptions{Include: include, Exclude: exclude})
		if err := cover.WriteProfile(profilePath, filtered); err != nil {
			return err
		}
		if min != nil {
			if err := cover.CheckMinimum(cover.Summarize(filtered), *min); err != nil {
				return err
			}
		}
	}
	_ = os.RemoveAll(fragDir)

	if runErr != nil {
		if _, ok := runErr.(*exec.ExitError); ok {
			return errTestsFailed
		}
		return runErr
	}
	return nil
}

func firstStdlibDir() string {
	if dir := os.Getenv("TYA_LIB_DIR"); dir != "" {
		return dir
	}
	if dir := os.Getenv("TYA_STDLIB_DIR"); dir != "" {
		return dir
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		for _, c := range []string{
			filepath.Join(exeDir, "lib"),
			filepath.Clean(filepath.Join(exeDir, "..", "share", "tya", "lib")),
		} {
			if info, err := os.Stat(c); err == nil && info.IsDir() {
				return c
			}
		}
	}
	if info, err := os.Stat("lib"); err == nil && info.IsDir() {
		return "lib"
	}
	return ""
}

func firstPackagesDir() string {
	if info, err := os.Stat(".tya/packages"); err == nil && info.IsDir() {
		return ".tya/packages"
	}
	return ""
}

func mergeCoverageFragments(profilePath, fragDir string, reg *codegen.CoverageRegistry) error {
	prof := cover.New()
	if reg != nil {
		fileIDByPath := map[string]int{}
		for _, e := range reg.Entries {
			fid, ok := fileIDByPath[e.File]
			if !ok {
				fid = len(prof.Files)
				fileIDByPath[e.File] = fid
				prof.Files = append(prof.Files, cover.File{ID: fid, Path: e.File})
			}
			prof.Stmts = append(prof.Stmts, cover.Stmt{ID: e.ID, FileID: fid, Line: e.Line, Col: e.Col})
		}
	}
	entries, err := os.ReadDir(fragDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			frag, err := cover.ReadProfile(filepath.Join(fragDir, e.Name()))
			if err != nil {
				continue
			}
			cover.Merge(prof, frag)
		}
	}
	return cover.WriteProfile(profilePath, prof)
}

func coverCommand(args []string) error {
	format := cliFormat
	profilePath := filepath.Join(".tya", "coverage", "profile")
	outputPath := filepath.Join(".tya", "coverage", "index.html")
	cli := coverageCLIOptions{}
	sub := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "report" || a == "html":
			sub = a
		case a == "-o" || a == "--output":
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for %s", a)
			}
			outputPath = args[i+1]
			i++
		case strings.HasPrefix(a, "-o="):
			outputPath = strings.TrimPrefix(a, "-o=")
		case strings.HasPrefix(a, "--output="):
			outputPath = strings.TrimPrefix(a, "--output=")
		case a == "--include" || a == "--exclude" || a == "--min":
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for %s", a)
			}
			if err := cli.add(a, args[i+1]); err != nil {
				return err
			}
			i++
		case strings.HasPrefix(a, "--include="):
			cli.include = append(cli.include, strings.TrimPrefix(a, "--include="))
		case strings.HasPrefix(a, "--exclude="):
			cli.exclude = append(cli.exclude, strings.TrimPrefix(a, "--exclude="))
		case strings.HasPrefix(a, "--min="):
			if err := cli.add("--min", strings.TrimPrefix(a, "--min=")); err != nil {
				return err
			}
		case a == "--format":
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for --format")
			}
			f, err := diag.ParseFormat(args[i+1])
			if err != nil {
				return err
			}
			format = f
			i++
		case strings.HasPrefix(a, "--format="):
			f, err := diag.ParseFormat(strings.TrimPrefix(a, "--format="))
			if err != nil {
				return err
			}
			format = f
		case a == "--profile":
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for --profile")
			}
			profilePath = args[i+1]
			i++
		case strings.HasPrefix(a, "--profile="):
			profilePath = strings.TrimPrefix(a, "--profile=")
		default:
			return fmt.Errorf("unknown cover option: %s", a)
		}
	}
	prof, err := cover.ReadProfile(profilePath)
	if err != nil {
		return err
	}
	cfg := readCoverageConfig(".")
	include := append([]string{}, cfg.include...)
	include = append(include, cli.include...)
	exclude := append([]string{}, cfg.exclude...)
	exclude = append(exclude, cli.exclude...)
	prof = cover.Filter(prof, cover.FilterOptions{Include: include, Exclude: exclude})
	summaries := cover.Summarize(prof)
	min := cfg.min
	if cli.min != nil {
		min = cli.min
	}
	if min != nil {
		if err := cover.CheckMinimum(summaries, *min); err != nil {
			return err
		}
	}
	if sub == "html" {
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			return err
		}
		f, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		defer f.Close()
		return cover.RenderHTML(f, prof, profilePath)
	}
	if format == diag.FormatJSON {
		return cover.RenderJSON(os.Stdout, prof, profilePath, version)
	}
	return cover.RenderText(os.Stdout, summaries)
}

func embedCommand(args []string) error {
	format := cliFormat
	list := false
	file := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--list":
			list = true
		case a == "--format":
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for --format")
			}
			f, err := diag.ParseFormat(args[i+1])
			if err != nil {
				return err
			}
			format = f
			i++
		case strings.HasPrefix(a, "--format="):
			f, err := diag.ParseFormat(strings.TrimPrefix(a, "--format="))
			if err != nil {
				return err
			}
			format = f
		default:
			file = a
		}
	}
	if !list {
		return fmt.Errorf("usage: tya embed --list [--format=json] <file.tya>")
	}
	if file == "" {
		return fmt.Errorf("missing file for tya embed --list")
	}
	items, err := collectEmbedList(file)
	if err != nil {
		return err
	}
	if format == diag.FormatJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(items)
	}
	fmt.Fprintf(os.Stdout, "%-16s %-30s %-30s %8s %-20s\n", "Binding", "Source", "Output", "Size", "Transforms")
	for _, item := range items {
		fmt.Fprintf(os.Stdout, "%-16s %-30s %-30s %8d %-20s\n", item.Binding, item.SourcePath, item.OutputPath, item.Size, strings.Join(item.Transforms, ","))
	}
	return nil
}

type embedListItem struct {
	Binding    string   `json:"binding"`
	SourcePath string   `json:"source_path"`
	OutputPath string   `json:"output_path"`
	Size       int64    `json:"size"`
	Transforms []string `json:"transforms"`
}

func collectEmbedList(file string) ([]embedListItem, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	toks, errs := lexer.Lex(string(raw))
	if len(errs) > 0 {
		return nil, errs[0]
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		return nil, err
	}
	base := filepath.Dir(file)
	var out []embedListItem
	for _, stmt := range prog.Stmts {
		embed, ok := stmt.(*ast.EmbedStmt)
		if !ok {
			continue
		}
		matches := []string{embed.Path}
		if strings.ContainsAny(embed.Path, "*?") {
			hits, err := embedListMatches(base, embed.Path)
			if err != nil {
				return nil, err
			}
			matches = hits
		}
		transforms := embedTransforms(embed.Transforms)
		for _, rel := range matches {
			info, _ := os.Stat(filepath.Join(base, filepath.FromSlash(rel)))
			size := int64(0)
			if info != nil {
				size = info.Size()
			}
			out = append(out, embedListItem{Binding: embed.Name, SourcePath: rel, OutputPath: rel, Size: size, Transforms: transforms})
		}
	}
	return out, nil
}

func embedListMatches(base, pattern string) ([]string, error) {
	if !strings.Contains(pattern, "**") {
		full := filepath.Join(base, filepath.FromSlash(pattern))
		hits, err := filepath.Glob(full)
		if err != nil {
			return nil, err
		}
		out := []string{}
		for _, hit := range hits {
			if info, err := os.Stat(hit); err == nil && !info.IsDir() {
				if rel, err := filepath.Rel(base, hit); err == nil {
					out = append(out, filepath.ToSlash(rel))
				}
			}
		}
		sort.Strings(out)
		return out, nil
	}

	prefix := strings.TrimSuffix(pattern[:strings.Index(pattern, "**")], "/")
	root := filepath.Join(base, filepath.FromSlash(prefix))
	out := []string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

func embedTransforms(m map[string]bool) []string {
	out := []string{}
	for _, k := range []string{"minify", "hash", "gzip"} {
		if m[k] {
			out = append(out, k)
		}
	}
	return out
}

func synthesizeTestSuite(files []string) (string, error) {
	moduleNames := []string{}
	seen := map[string]string{}
	for _, f := range files {
		mod := strings.TrimSuffix(filepath.Base(f), ".tya")
		if existing, ok := seen[mod]; ok {
			return "", fmt.Errorf("duplicate test module name %s: %s and %s", mod, existing, f)
		}
		seen[mod] = f
		moduleNames = append(moduleNames, mod)
	}
	var b strings.Builder
	b.WriteString("import unittest/* as *\n")
	for _, m := range moduleNames {
		b.WriteString("import ")
		b.WriteString(m)
		b.WriteString("\n")
	}
	b.WriteString("\nsuite = TestSuite().discover([")
	for i, m := range moduleNames {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(m)
	}
	b.WriteString("])\n")
	b.WriteString("TestRunner().default().run_and_exit(suite)\n")
	return b.String(), nil
}

func testFiles(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if !strings.HasSuffix(filepath.Base(root), "_test.tya") {
			return nil, fmt.Errorf("test file must match *_test.tya: %s", root)
		}
		return []string{root}, nil
	}
	files := []string{}
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(filepath.Base(path), "_test.tya") {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func checkFile(path string) error {
	defer checker.SetPermissiveLegacy(runner.IsLegacyV01Path(path))()
	nativePlan, err := nativePlanForPath(path)
	if err != nil {
		return err
	}
	nativeNames := []string{}
	if nativePlan != nil {
		nativeNames = append(nativeNames, nativePlan.FuncOrder...)
	}
	defer checker.SetExtraBuiltinNames(nativeNames)()
	if nativePlan != nil {
		defer codegen.SetNativeFunctions(nativePlan.Functions)()
	} else {
		defer codegen.SetNativeFunctions(nil)()
	}
	// tya check on a class/interface file is allowed
	// (read-only, no entry semantics). Skip the entry-only
	// runner.LoadSourceWithModules path and check the class file
	// together with sibling package members.
	isClassFile, err := runner.IsClassFile(path)
	if err != nil {
		return err
	}
	if isClassFile {
		source, modules, origins, err := runner.LoadClassFileWithSiblingOrigins(path)
		if err != nil {
			return err
		}
		toks, errs := lexer.Lex(source)
		if len(errs) > 0 {
			return errs[0]
		}
		prog, _, err := parser.Parse(toks)
		if err != nil {
			return err
		}
		runner.StampOriginFiles(prog, origins)
		diags, err := checker.CheckAll(prog, modules, path, true)
		if err != nil {
			return err
		}
		if len(diags) == 0 {
			_, _, err := codegen.EmitCWithPath(prog, path)
			return err
		}
		emitDiagnostics(diags, path)
		return errStrictReported
	}
	source, modules, origins, err := runner.LoadSourceWithOrigins(path)
	if err != nil {
		return err
	}
	toks, errs := lexer.Lex(source)
	if len(errs) > 0 {
		return errs[0]
	}
	prog, _, err := parser.Parse(toks)
	if err != nil {
		return err
	}
	runner.StampOriginFiles(prog, origins)
	diags, err := checker.CheckAll(prog, modules, path, true)
	if err != nil {
		return err
	}
	commentDiags, cerr := commentPositionDiagnostics(path)
	if cerr == nil {
		diags = append(diags, commentDiags...)
	}
	if len(diags) == 0 {
		_, _, err := codegen.EmitCWithPath(prog, path)
		return err
	}
	emitDiagnostics(diags, path)
	return errStrictReported
}

// commentPositionDiagnostics validates the user's own source file
// (not the inlined import-resolved one) for CANONICAL §3.4
// forbidden comment positions. Block-trailing, file-trailing, and
// orphaned comments not attached to any header, leading, or
// line-end slot become structured diagnostics.
func commentPositionDiagnostics(path string) ([]diag.Diagnostic, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	toks, lcomments, lerrs := lexer.LexWithComments(string(src))
	if len(lerrs) > 0 {
		return nil, lerrs[0]
	}
	infos := make([]parser.CommentInfo, 0, len(lcomments))
	for _, c := range lcomments {
		infos = append(infos, parser.CommentInfo{
			Line: c.Line, Col: c.Col, Indent: c.Indent,
			Text: c.Text, IsFullLine: c.IsFullLine,
		})
	}
	prog, _, err := parser.ParseWithComments(toks, infos)
	if err != nil {
		return nil, err
	}
	orphans := parser.OrphanComments(prog, infos)
	if len(orphans) == 0 {
		return nil, nil
	}
	out := make([]diag.Diagnostic, 0, len(orphans))
	for _, c := range orphans {
		title := "Comment at forbidden position"
		message := "This comment is not attached to a header, a leading-comment block, or a line-end slot."
		hint := "Move the comment immediately above the statement it documents (no blank line between), or to the file header (separated from the body by exactly one blank line)."
		out = append(out, diag.Diagnostic{
			Severity: diag.Error,
			Code:     "TYA-E0150",
			Title:    title,
			Message:  message,
			Primary: diag.Region{
				File:  path,
				Start: diag.Pos{Line: c.Line, Col: c.Col},
				End:   diag.Pos{Line: c.Line, Col: c.Col + 1},
			},
			Hints:  []string{hint},
			Source: "checker",
		})
	}
	return out, nil
}

// errStrictReported signals that diagnostics were already printed.
// Callers should exit with a non-zero status without re-printing.
var errStrictReported = errors.New("__strict_reported__")

// parseGlobalDiagFlags scans os.Args for --format=… and --color=…
// (and the ` ` form) anywhere in the command line. Recognized flags
// are removed from os.Args so the rest of the CLI parsing is unchanged.
//
// The `lint` subcommand owns its own --format namespace
// (text|json output report) and is intentionally exempt — when the
// first non-flag arg is "lint", everything after it is left untouched
// so `tya lint --format=json` reaches lintCommand verbatim.
func parseGlobalDiagFlags() error {
	out := []string{os.Args[0]}
	args := os.Args[1:]
	exemptAfter := -1
	for i, a := range args {
		if !strings.HasPrefix(a, "-") {
			if a == "lint" || a == "doc" || a == "cover" || a == "embed" || a == "version" {
				exemptAfter = i
			}
			break
		}
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if exemptAfter >= 0 && i > exemptAfter {
			out = append(out, a)
			continue
		}
		switch {
		case a == "--json":
			cliFormat = diag.FormatJSON
		case a == "--no-color":
			cliColor = diag.ColorNever
		case strings.HasPrefix(a, "--format="):
			f, err := diag.ParseFormat(strings.TrimPrefix(a, "--format="))
			if err != nil {
				return err
			}
			cliFormat = f
		case a == "--format":
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for --format")
			}
			f, err := diag.ParseFormat(args[i+1])
			if err != nil {
				return err
			}
			cliFormat = f
			i++
		case strings.HasPrefix(a, "--color="):
			c, err := diag.ParseColorMode(strings.TrimPrefix(a, "--color="))
			if err != nil {
				return err
			}
			cliColor = c
		case a == "--color":
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for --color")
			}
			c, err := diag.ParseColorMode(args[i+1])
			if err != nil {
				return err
			}
			cliColor = c
			i++
		default:
			out = append(out, a)
		}
	}
	os.Args = out
	return nil
}

// emitDiagnostics renders strict diagnostics to stderr in the user's
// chosen format. file is used to populate the source map when paths
// in diagnostics differ.
func emitDiagnostics(diags []diag.Diagnostic, file string) {
	if cliFormat == diag.FormatJSON {
		fmt.Fprint(os.Stderr, diag.RenderJSON(diags))
		return
	}
	sm := diag.NewSourceMap()
	seen := map[string]bool{}
	if file != "" && !seen[file] {
		_ = sm.AddFromDisk(file)
		seen[file] = true
	}
	for _, d := range diags {
		if d.Primary.File != "" && !seen[d.Primary.File] {
			_ = sm.AddFromDisk(d.Primary.File)
			seen[d.Primary.File] = true
		}
	}
	opts := diag.RenderOptions{
		Color:   cliColor,
		IsTTY:   isTTY(os.Stderr),
		NoColor: os.Getenv("NO_COLOR") != "",
	}
	fmt.Fprint(os.Stderr, diag.Render(diags, sm, opts))
}

func isTTY(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func formatCommand(args []string) error {
	write := false
	check := false
	path := ""
	for _, arg := range args {
		if arg == "-w" {
			write = true
			continue
		}
		if arg == "--check" {
			check = true
			continue
		}
		if strings.HasPrefix(arg, "-") {
			return fmt.Errorf("unknown format option: %s", arg)
		}
		if path != "" {
			return fmt.Errorf("unexpected format argument: %s", arg)
		}
		path = arg
	}
	if path == "" {
		return fmt.Errorf("missing input file")
	}
	if err := runner.ValidateAnyTyaFileName(path); err != nil {
		return err
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	formatted, err := formatSourceStrict(string(src))
	if err != nil {
		return err
	}
	if check {
		if formatted != string(src) {
			return fmt.Errorf("%s is not in formatted syntax", path)
		}
		return nil
	}
	if write {
		return os.WriteFile(path, []byte(formatted), 0644)
	}
	fmt.Fprint(os.Stdout, formatted)
	return nil
}

func formatSourceStrict(src string) (string, error) {
	toks, lcomments, errs := lexer.LexWithComments(src)
	if len(errs) > 0 {
		return "", errs[0]
	}
	comments := make([]parser.CommentInfo, 0, len(lcomments))
	for _, c := range lcomments {
		comments = append(comments, parser.CommentInfo{
			Line: c.Line, Col: c.Col, Indent: c.Indent,
			Text: c.Text, IsFullLine: c.IsFullLine,
		})
	}
	prog, _, err := parser.ParseWithComments(toks, comments)
	if err != nil {
		return "", err
	}
	out, err := formatter.Unparse(prog)
	if err != nil {
		return formatter.FormatSource(src), nil
	}
	return out, nil
}

// formatSource applies the canonical AST-driven serializer. On
// any lex / parse / unparse error it falls back to the v0.2 text
// pass so editor save hooks remain safe; this fallback is
// invisible to the caller and is expected to shrink as Unparse
// coverage grows.
func formatSource(src string) string {
	toks, lcomments, errs := lexer.LexWithComments(src)
	if len(errs) > 0 {
		return formatter.FormatSource(src)
	}
	comments := make([]parser.CommentInfo, 0, len(lcomments))
	for _, c := range lcomments {
		comments = append(comments, parser.CommentInfo{
			Line: c.Line, Col: c.Col, Indent: c.Indent,
			Text: c.Text, IsFullLine: c.IsFullLine,
		})
	}
	prog, _, err := parser.ParseWithComments(toks, comments)
	if err != nil {
		return formatter.FormatSource(src)
	}
	out, err := formatter.Unparse(prog)
	if err != nil {
		return formatter.FormatSource(src)
	}
	return out
}

func printDiagnostic(path string, err error) {
	var serr *checker.StrictError
	if errors.As(err, &serr) && len(serr.Diags) > 0 {
		emitDiagnostics(serr.Diags, path)
		return
	}
	var perr *parser.ParserError
	if errors.As(err, &perr) && len(perr.Diags) > 0 {
		out := make([]diag.Diagnostic, len(perr.Diags))
		for i, d := range perr.Diags {
			d.Primary.File = path
			out[i] = d
		}
		emitDiagnostics(out, path)
		return
	}
	var rerr *runner.RunnerError
	if errors.As(err, &rerr) && len(rerr.Diags) > 0 {
		out := make([]diag.Diagnostic, len(rerr.Diags))
		for i, d := range rerr.Diags {
			d.Primary.File = path
			out[i] = d
		}
		emitDiagnostics(out, path)
		return
	}
	var cerr *codegen.CodegenError
	if errors.As(err, &cerr) && len(cerr.Diags) > 0 {
		out := make([]diag.Diagnostic, len(cerr.Diags))
		for i, d := range cerr.Diags {
			d.Primary.File = path
			out[i] = d
		}
		emitDiagnostics(out, path)
		return
	}
	var ldiag *lexer.Diagnostic
	if errors.As(err, &ldiag) {
		d := ldiag.Diag
		d.Primary.File = path
		emitDiagnostics([]diag.Diagnostic{d}, path)
		return
	}
	src, readErr := os.ReadFile(path)
	if readErr != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	line, col, msg, ok := parseLineColError(err.Error())
	if !ok {
		emitDiagnostics([]diag.Diagnostic{{
			Severity: diag.Error,
			Code:     "TYA-E0900",
			Title:    "runtime error",
			Message:  err.Error(),
			Primary:  diag.Region{File: path},
			Source:   "runtime",
		}}, path)
		return
	}
	_ = src
	emitDiagnostics([]diag.Diagnostic{{
		Severity: diag.Error,
		Code:     "TYA-E0900",
		Title:    "runtime error",
		Message:  msg,
		Primary: diag.Region{
			File:  path,
			Start: diag.Pos{Line: line, Col: col},
			End:   diag.Pos{Line: line, Col: col + 1},
		},
		Source: "runtime",
	}}, path)
	if cliFormat != diag.FormatJSON {
		fmt.Fprintf(os.Stderr, "%s:%d:%d: %s\n", path, line, col, msg)
	}
}

func parseLineColError(value string) (int, int, string, bool) {
	matches := lineColErrorRE.FindStringSubmatch(value)
	if matches == nil {
		return 0, 0, "", false
	}
	line, lineErr := strconv.Atoi(matches[1])
	col, colErr := strconv.Atoi(matches[2])
	if lineErr != nil || colErr != nil {
		return 0, 0, "", false
	}
	msg := matches[3]
	if before, _, found := strings.Cut(msg, " near "); found {
		msg = before
	}
	return line, col, msg, true
}

type buildCLIOptions struct {
	Path   string
	Output string
	Target string
}

func parseBuildArgs(args []string) (buildCLIOptions, error) {
	opts := buildCLIOptions{Target: "native"}
	for i := 0; i < len(args); i++ {
		if args[i] == "-o" {
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing output after -o")
			}
			opts.Output = args[i+1]
			i++
			continue
		}
		if args[i] == "--target" {
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing output after --target")
			}
			opts.Target = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(args[i], "--target=") {
			opts.Target = strings.TrimPrefix(args[i], "--target=")
			continue
		}
		if strings.HasPrefix(args[i], "-") {
			return opts, fmt.Errorf("unknown build option: %s", args[i])
		}
		if opts.Path != "" {
			return opts, fmt.Errorf("unexpected build argument: %s", args[i])
		}
		opts.Path = args[i]
	}
	if opts.Path == "" {
		return opts, fmt.Errorf("missing input file")
	}
	if opts.Target != "native" && opts.Target != "wasm32-wasi" && opts.Target != "wasm32-browser" {
		return opts, fmt.Errorf("unsupported build target: %s", opts.Target)
	}
	return opts, nil
}

func defaultOutputPath(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func installCommand() error {
	root, err := projectRoot()
	if err != nil {
		return err
	}
	m, lf, err := pkg.Install(root)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Resolved %d packages for %s\n", len(lf.Packages), m.Name)
	for _, p := range lf.Packages {
		fmt.Fprintf(os.Stdout, "  %s %s (%s)\n", p.Name, p.Version, p.Source)
	}
	return nil
}

func updateCommand(target string) error {
	root, err := projectRoot()
	if err != nil {
		return err
	}
	m, lf, err := pkg.Update(root, target)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Updated %s (%d packages)\n", m.Name, len(lf.Packages))
	return nil
}

func addCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("tya add: missing package name")
	}
	root, err := projectRoot()
	if err != nil {
		return err
	}
	dep := pkg.Dependency{Name: args[0]}
	isDev := false
	args = args[1:]
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--dev":
			isDev = true
		case "--git":
			if i+1 >= len(args) {
				return fmt.Errorf("--git requires a URL")
			}
			dep.Source = "git"
			dep.Git = args[i+1]
			i++
		case "--tag":
			if i+1 >= len(args) {
				return fmt.Errorf("--tag requires a value")
			}
			dep.Tag = args[i+1]
			i++
		case "--branch":
			if i+1 >= len(args) {
				return fmt.Errorf("--branch requires a value")
			}
			dep.Branch = args[i+1]
			i++
		case "--rev":
			if i+1 >= len(args) {
				return fmt.Errorf("--rev requires a value")
			}
			dep.Rev = args[i+1]
			i++
		case "--path":
			if i+1 >= len(args) {
				return fmt.Errorf("--path requires a value")
			}
			dep.Source = "path"
			dep.PathRef = args[i+1]
			i++
		default:
			c, perr := pkg.ParseConstraint(a)
			if perr != nil {
				return fmt.Errorf("tya add: %v", perr)
			}
			dep.Constraint = c
			if dep.Source == "" {
				dep.Source = "version"
			}
		}
	}
	if dep.Source == "" {
		return fmt.Errorf("tya add: missing source (constraint, --git, or --path)")
	}
	if err := pkg.AddDependency(root, dep, isDev); err != nil {
		return err
	}
	if dep.Source == "version" {
		fmt.Fprintf(os.Stdout, "Added %s %s to manifest. Run `tya install` after pinning to a git or path source.\n", dep.Name, dep.Constraint.Raw)
		return nil
	}
	_, lf, err := pkg.Install(root)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Added %s. Resolved %d packages.\n", dep.Name, len(lf.Packages))
	return nil
}

func removeCommand(name string) error {
	root, err := projectRoot()
	if err != nil {
		return err
	}
	if err := pkg.RemoveDependency(root, name); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Removed %s.\n", name)
	return nil
}

func outdatedCommand() error {
	root, err := projectRoot()
	if err != nil {
		return err
	}
	lockPath := filepath.Join(root, pkg.LockfileName)
	lf, err := pkg.ReadLockfile(lockPath)
	if err != nil {
		return fmt.Errorf("no lockfile: run `tya install` first")
	}
	if len(lf.Packages) == 0 {
		fmt.Fprintln(os.Stdout, "No locked packages.")
		return nil
	}
	fmt.Fprintf(os.Stdout, "Locked packages (newer-version detection requires fetching, run `tya update` to refresh):\n")
	for _, p := range lf.Packages {
		fmt.Fprintf(os.Stdout, "  %s %s (%s)\n", p.Name, p.Version, p.Source)
	}
	return nil
}

func projectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root, _, err := pkg.FindManifest(dir)
	return root, err
}
