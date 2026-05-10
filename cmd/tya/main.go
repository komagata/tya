package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

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

const version = "0.42.0"

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
			if !errors.Is(err, errStrictReported) {
				printDiagnostic(os.Args[2], err)
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
		args := os.Args[2:]
		for i := 0; i < len(args); i++ {
			a := args[i]
			switch {
			case a == "--cover":
				coverEnabled = true
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
		if err := testCommand(root, coverEnabled, profilePath); err != nil {
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
	fmt.Fprintln(os.Stderr, "       tya format [-w] <file.tya>")
	fmt.Fprintln(os.Stderr, "       tya emit-c <file.tya>")
	fmt.Fprintln(os.Stderr, "       tya test [--cover [--profile FILE]] [path]")
	fmt.Fprintln(os.Stderr, "       tya cover [--format=human|json] [--profile FILE]")
	fmt.Fprintln(os.Stderr, "       tya install")
	fmt.Fprintln(os.Stderr, "       tya update [package]")
	fmt.Fprintln(os.Stderr, "       tya add <name> [<constraint>] [--git URL --tag T] [--path P] [--dev]")
	fmt.Fprintln(os.Stderr, "       tya remove <name>")
	fmt.Fprintln(os.Stderr, "       tya outdated")
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
	_, err := buildExecutableWithCover(path, output, nil)
	return err
}

func buildExecutableWithCover(path string, output string, opt *codegen.CoverageOptions) (*codegen.CoverageRegistry, error) {
	csrc, reg, err := compileToCWithCover(path, opt)
	if err != nil {
		return nil, err
	}
	if output == "" {
		output = defaultOutputPath(path)
	}
	outDir, err := os.MkdirTemp("", "tya-build-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(outDir)
	cfile := filepath.Join(outDir, "main.c")
	if err := os.WriteFile(cfile, []byte(csrc), 0644); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return nil, err
	}
	cc := os.Getenv("CC")
	if cc == "" {
		cc = "cc"
	}
	runtimeDir, err := findRuntimeDir()
	if err != nil {
		return nil, err
	}
	args := []string{cfile, filepath.Join(runtimeDir, "tya_runtime.c")}
	if opt != nil {
		coverC := filepath.Join(runtimeDir, "tya_cover.c")
		if _, err := os.Stat(coverC); err == nil {
			args = append(args, coverC)
		}
	}
	args = append(args, "-I", runtimeDir, "-o", output)
	// v0.42 runtime uses pthread for spawn / await / channels / sync.
	// On macOS/BSD pthread is in libc; on Linux/glibc it lives in -lpthread.
	if runtime.GOOS == "linux" {
		args = append(args, "-lpthread")
	}
	compile := exec.Command(cc, args...)
	compile.Stderr = os.Stderr
	if err := compile.Run(); err != nil {
		return nil, err
	}
	return reg, nil
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
	source, modules, err := runner.LoadSourceWithModules(path)
	if err != nil {
		return "", nil, err
	}
	toks, errs := lexer.Lex(source)
	if len(errs) > 0 {
		return "", nil, errs[0]
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		return "", nil, err
	}
	if err := checker.CheckWithModules(prog, modules); err != nil {
		return "", nil, err
	}
	csrc, reg, err := codegen.EmitCWithCoverage(prog, path, opt)
	if err != nil {
		return "", nil, err
	}
	return csrc, reg, nil
}

func testCommand(root string, coverEnabled bool, profilePath string) error {
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
	combined := strings.Join(pathDirs, string(os.PathListSeparator))
	if prevPath != "" {
		combined = combined + string(os.PathListSeparator) + prevPath
	}
	if err := os.Setenv("TYA_PATH", combined); err != nil {
		return err
	}
	defer os.Setenv("TYA_PATH", prevPath)

	if !coverEnabled {
		if err := compileAndRun(suitePath, nil); err != nil {
			if _, ok := err.(*exec.ExitError); ok {
				return errTestsFailed
			}
			return err
		}
		return nil
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
	if dir := os.Getenv("TYA_STDLIB_DIR"); dir != "" {
		return dir
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		for _, c := range []string{
			filepath.Join(exeDir, "stdlib"),
			filepath.Clean(filepath.Join(exeDir, "..", "share", "tya", "stdlib")),
		} {
			if info, err := os.Stat(c); err == nil && info.IsDir() {
				return c
			}
		}
	}
	if info, err := os.Stat("stdlib"); err == nil && info.IsDir() {
		return "stdlib"
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
	sub := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "report" || a == "html":
			sub = a
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
	if sub == "html" {
		return fmt.Errorf("tya cover html is deferred to a later release")
	}
	prof, err := cover.ReadProfile(profilePath)
	if err != nil {
		return err
	}
	if format == diag.FormatJSON {
		return cover.RenderJSON(os.Stdout, prof, profilePath, version)
	}
	return cover.RenderText(os.Stdout, cover.Summarize(prof))
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
	b.WriteString("import unittest\n")
	for _, m := range moduleNames {
		b.WriteString("import ")
		b.WriteString(m)
		b.WriteString("\n")
	}
	b.WriteString("\nunittest.run([")
	for i, m := range moduleNames {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(m)
	}
	b.WriteString("])\n")
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
	diags, err := checker.CheckAll(prog, modules, path, true)
	if err != nil {
		return err
	}
	commentDiags, cerr := commentPositionDiagnostics(path)
	if cerr == nil {
		diags = append(diags, commentDiags...)
	}
	if len(diags) == 0 {
		return nil
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
	prog, err := parser.ParseWithComments(toks, infos)
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
func parseGlobalDiagFlags() error {
	out := []string{os.Args[0]}
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
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
	path := ""
	for _, arg := range args {
		if arg == "-w" {
			write = true
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
	if err := runner.ValidateFileName(path); err != nil {
		return err
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	formatted := formatSource(string(src))
	if write {
		return os.WriteFile(path, []byte(formatted), 0644)
	}
	fmt.Fprint(os.Stdout, formatted)
	return nil
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
	prog, err := parser.ParseWithComments(toks, comments)
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
	for i := 0; i < 16; i++ {
		if _, err := os.Stat(filepath.Join(dir, pkg.ManifestName)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("no tya.toml found in current directory or any parent")
}
