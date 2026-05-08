package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"tya/internal/checker"
	"tya/internal/codegen"
	"tya/internal/formatter"
	"tya/internal/lexer"
	"tya/internal/parser"
	"tya/internal/runner"
)

const version = "0.6.0"

var lineColErrorRE = regexp.MustCompile(`^(\d+):(\d+):\s*(.*)$`)
var errTestsFailed = errors.New("test failed")

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	if os.Args[1] == "version" || os.Args[1] == "--version" {
		fmt.Fprintln(os.Stdout, version)
		return
	}
	switch os.Args[1] {
	case "run":
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		if err := compileAndRun(os.Args[2], os.Args[3:]); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			printDiagnostic(os.Args[2], err)
			os.Exit(1)
		}
		return
	case "build":
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		path, output, err := parseBuildArgs(os.Args[2:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if err := buildExecutable(path, output); err != nil {
			printDiagnostic(path, err)
			os.Exit(1)
		}
		return
	case "check":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		if err := checkFile(os.Args[2]); err != nil {
			printDiagnostic(os.Args[2], err)
			os.Exit(1)
		}
		return
	case "fmt":
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
		if len(os.Args) > 3 {
			usage()
			os.Exit(2)
		}
		root := "."
		if len(os.Args) == 3 {
			root = os.Args[2]
		}
		if err := testCommand(root); err != nil {
			if errors.Is(err, errTestsFailed) {
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
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
	if err := runner.ValidateFileName(path); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !dumpTokens && !emitC && !checkUnused {
		usage()
		os.Exit(2)
	}
	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	source := string(src)
	modules := []string(nil)
	if emitC && !checkUnused {
		source, modules, err = runner.LoadSourceWithModules(path)
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
		prog, err := parser.Parse(toks)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := checker.CheckWithModules(prog, modules); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		csrc, err := codegen.EmitCWithPath(prog, path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Fprint(os.Stdout, csrc)
		return
	}
	if checkUnused {
		source, modules, err = runner.LoadUserSourceWithModules(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
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
	prog, err := parser.Parse(toks)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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
		source, modules, err = runner.LoadSourceWithModules(path)
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
		prog, err = parser.Parse(toks)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := checker.CheckWithModules(prog, modules); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		csrc, err := codegen.EmitCWithPath(prog, path)
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
	fmt.Fprintln(os.Stderr, "       tya fmt [-w] <file.tya>")
	fmt.Fprintln(os.Stderr, "       tya emit-c <file.tya>")
	fmt.Fprintln(os.Stderr, "       tya test [path]")
	fmt.Fprintln(os.Stderr, "       tya version")
}

func compileAndRun(path string, args []string) error {
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
	return run.Run()
}

func buildExecutable(path string, output string) error {
	csrc, err := compileToC(path)
	if err != nil {
		return err
	}
	if output == "" {
		output = defaultOutputPath(path)
	}
	outDir, err := os.MkdirTemp("", "tya-build-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(outDir)
	cfile := filepath.Join(outDir, "main.c")
	if err := os.WriteFile(cfile, []byte(csrc), 0644); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	cc := os.Getenv("CC")
	if cc == "" {
		cc = "cc"
	}
	runtimeDir, err := findRuntimeDir()
	if err != nil {
		return err
	}
	compile := exec.Command(cc, cfile, filepath.Join(runtimeDir, "tya_runtime.c"), "-I", runtimeDir, "-o", output)
	compile.Stderr = os.Stderr
	return compile.Run()
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
	source, modules, err := runner.LoadSourceWithModules(path)
	if err != nil {
		return "", err
	}
	toks, errs := lexer.Lex(source)
	if len(errs) > 0 {
		return "", errs[0]
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		return "", err
	}
	if err := checker.CheckWithModules(prog, modules); err != nil {
		return "", err
	}
	csrc, err := codegen.EmitCWithPath(prog, path)
	if err != nil {
		return "", err
	}
	return csrc, nil
}

func testCommand(root string) error {
	files, err := testFiles(root)
	if err != nil {
		return err
	}
	failed := false
	for _, file := range files {
		if err := compileAndRun(file, nil); err != nil {
			failed = true
			if exitErr, ok := err.(*exec.ExitError); ok {
				_ = exitErr
				continue
			}
			return err
		}
	}
	if failed {
		return errTestsFailed
	}
	return nil
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
	source, modules, err := runner.LoadSourceWithModules(path)
	if err != nil {
		return err
	}
	toks, errs := lexer.Lex(source)
	if len(errs) > 0 {
		return errs[0]
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		return err
	}
	return checker.CheckWithModules(prog, modules)
}

func formatCommand(args []string) error {
	write := false
	path := ""
	for _, arg := range args {
		if arg == "-w" {
			write = true
			continue
		}
		if strings.HasPrefix(arg, "-") {
			return fmt.Errorf("unknown fmt option: %s", arg)
		}
		if path != "" {
			return fmt.Errorf("unexpected fmt argument: %s", arg)
		}
		path = arg
	}
	if path == "" {
		return fmt.Errorf("missing input file")
	}
	if err := runner.ValidateFileName(path); err != nil {
		return err
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	formatted := formatter.FormatSource(string(src))
	if write {
		return os.WriteFile(path, []byte(formatted), 0644)
	}
	fmt.Fprint(os.Stdout, formatted)
	return nil
}

func printDiagnostic(path string, err error) {
	src, readErr := os.ReadFile(path)
	if readErr != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	line, col, msg, ok := parseLineColError(err.Error())
	if !ok {
		fmt.Fprintf(os.Stderr, "%s: %s\n", path, err)
		return
	}
	lines := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
	sourceLine := ""
	if line >= 1 && line <= len(lines) {
		sourceLine = lines[line-1]
	}
	fmt.Fprintf(os.Stderr, "%s:%d:%d: %s\n", path, line, col, msg)
	if sourceLine != "" {
		fmt.Fprintf(os.Stderr, "  %s\n", sourceLine)
		fmt.Fprintf(os.Stderr, "  %s^\n", strings.Repeat(" ", max(0, col-1)))
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

func parseBuildArgs(args []string) (string, string, error) {
	path := ""
	output := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "-o" {
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("missing output after -o")
			}
			output = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(args[i], "-") {
			return "", "", fmt.Errorf("unknown build option: %s", args[i])
		}
		if path != "" {
			return "", "", fmt.Errorf("unexpected build argument: %s", args[i])
		}
		path = args[i]
	}
	if path == "" {
		return "", "", fmt.Errorf("missing input file")
	}
	return path, output, nil
}

func defaultOutputPath(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}
