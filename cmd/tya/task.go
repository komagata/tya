package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"tya/internal/pkg"
)

// taskCommand implements `tya task [name] [args...]`. With no name it
// prints the [tasks] table from tya.toml. With a name it runs the named
// task under /bin/sh -c, inheriting stdin/stdout/stderr, with the
// project root as CWD and any extra args appended (POSIX-quoted) to the
// command.
//
// Returns (exitCode, err). exitCode is 0 on success; err is non-nil
// when something goes wrong before or instead of running the command.
func taskCommand(args []string) (int, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return 1, err
	}
	root, manifestPath, err := pkg.FindManifest(cwd)
	if err != nil {
		return 1, fmt.Errorf("[TYA-E0902] %v", err)
	}
	m, err := pkg.ReadManifest(manifestPath)
	if err != nil {
		return 1, err
	}
	if len(args) == 0 {
		listTasks(m)
		return 0, nil
	}
	name := args[0]
	extra, watch := parseTaskArgs(args[1:])
	if _, ok := m.Tasks[name]; !ok {
		return 1, fmt.Errorf("[TYA-E0900] no task %q defined in tya.toml", name)
	}
	if err := validateTaskGraph(m, name); err != nil {
		return 1, err
	}
	if err := validateWatchPatterns(m.Tasks[name]); err != nil {
		return 1, err
	}
	if watch {
		return watchTask(root, m, name, extra)
	}
	return runTaskGraph(root, m, name, extra)
}

func parseTaskArgs(args []string) ([]string, bool) {
	extra := []string{}
	watch := false
	passThrough := false
	for _, arg := range args {
		if passThrough {
			extra = append(extra, arg)
			continue
		}
		if arg == "--" {
			passThrough = true
			continue
		}
		if arg == "--watch" {
			watch = true
			continue
		}
		extra = append(extra, arg)
	}
	return extra, watch
}

func runTaskGraph(root string, m *pkg.Manifest, name string, extra []string) (int, error) {
	ran := map[string]bool{}
	return runTaskWithDeps(root, m, name, extra, name, ran)
}

func runTaskWithDeps(root string, m *pkg.Manifest, name string, selectedExtra []string, selected string, ran map[string]bool) (int, error) {
	if ran[name] {
		return 0, nil
	}
	task := m.Tasks[name]
	for _, dep := range task.DependsOn {
		code, err := runTaskWithDeps(root, m, dep, selectedExtra, selected, ran)
		if err != nil || code != 0 {
			return code, err
		}
	}
	extra := []string{}
	if name == selected {
		extra = selectedExtra
	}
	code, err := runTask(root, task, extra)
	if err == nil && code == 0 {
		ran[name] = true
	}
	return code, err
}

func runTask(root string, task pkg.Task, extra []string) (int, error) {
	switch task.Kind {
	case "string":
		return runShell(root, task.Name, 0, task.String, extra, task.Env)
	case "array":
		for i, cmd := range task.Array {
			code, runErr := runShell(root, task.Name, i+1, cmd, extra, task.Env)
			if runErr != nil {
				return code, runErr
			}
			if code != 0 {
				return code, fmt.Errorf("[TYA-E0901] task %q command #%d (%q) failed with exit code %d", task.Name, i+1, cmd, code)
			}
		}
		return 0, nil
	case "parallel":
		return runParallel(root, task.Name, task.Array, extra, task.Env)
	default:
		return 1, fmt.Errorf("internal: task %q has unknown kind %q", task.Name, task.Kind)
	}
}

func validateTaskGraph(m *pkg.Manifest, selected string) error {
	visiting := map[string]bool{}
	visited := map[string]bool{}
	var visit func(string, []string) error
	visit = func(name string, stack []string) error {
		task, ok := m.Tasks[name]
		if !ok {
			parent := selected
			if len(stack) > 0 {
				parent = stack[len(stack)-1]
			}
			return fmt.Errorf("[TYA-E0905] task %q depends on unknown task %q", parent, name)
		}
		if visiting[name] {
			cycle := append(stack, name)
			return fmt.Errorf("[TYA-E0904] task dependency cycle: %s", strings.Join(cycle, " -> "))
		}
		if visited[name] {
			return nil
		}
		visiting[name] = true
		for _, dep := range task.DependsOn {
			if err := visit(dep, append(stack, name)); err != nil {
				return err
			}
		}
		visiting[name] = false
		visited[name] = true
		return nil
	}
	return visit(selected, nil)
}

func validateWatchPatterns(task pkg.Task) error {
	for _, pattern := range append(append([]string{}, task.Watch...), task.Ignore...) {
		if strings.Contains(pattern, "**") {
			continue
		}
		if _, err := filepath.Match(filepath.ToSlash(pattern), "probe"); err != nil {
			return fmt.Errorf("[TYA-E0907] task %q has invalid watch pattern %q: %v", task.Name, pattern, err)
		}
	}
	return nil
}

func listTasks(m *pkg.Manifest) {
	if len(m.TaskOrder) == 0 {
		fmt.Fprintln(os.Stdout, "no tasks defined in tya.toml")
		return
	}
	for _, name := range m.TaskOrder {
		t := m.Tasks[name]
		summary := taskSummary(t)
		fmt.Fprintf(os.Stdout, "%s\t%s\n", name, summary)
	}
}

func taskSummary(t pkg.Task) string {
	var s string
	if t.Kind == "array" || t.Kind == "parallel" {
		s = strings.Join(t.Array, " && ")
	} else {
		s = t.String
	}
	s = strings.ReplaceAll(s, "\n", " ")
	const maxLen = 80
	if len(s) > maxLen {
		s = s[:maxLen-1] + "…"
	}
	return s
}

func runShell(cwd, taskName string, index int, command string, extraArgs []string, env map[string]string) (int, error) {
	full := command
	if len(extraArgs) > 0 {
		full = command + " " + joinShellArgs(extraArgs)
	}
	cmd := exec.Command("/bin/sh", "-c", full)
	cmd.Dir = cwd
	cmd.Env = mergeEnv(env)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode(), nil
	}
	return 1, fmt.Errorf("task %q: failed to start /bin/sh: %v", taskName, err)
}

// joinShellArgs POSIX-quotes each arg and joins with spaces so that the
// resulting string can be appended to a /bin/sh -c command. Inside the
// single-quoted form, every literal single-quote is replaced with the
// four-character sequence '\” which terminates the quoted region,
// inserts an escaped single quote, and re-opens the quoted region.
func joinShellArgs(args []string) string {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = shellQuote(a)
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// runParallel runs every cmd concurrently under /bin/sh -c, waits
// for all to finish, prefixes each line of their stdout/stderr with
// `[<i> <truncated cmd>] `, and returns the first non-zero exit
// code observed. Errors starting the children are propagated as
// the second return value.
func runParallel(cwd, taskName string, cmds, extraArgs []string, env map[string]string) (int, error) {
	if len(cmds) == 0 {
		return 0, nil
	}
	type result struct {
		index int
		cmd   string
		code  int
		err   error
	}
	results := make([]result, len(cmds))
	var wg sync.WaitGroup
	wg.Add(len(cmds))
	for i, cmd := range cmds {
		i, cmd := i, cmd
		go func() {
			defer wg.Done()
			code, err := runShellPrefixed(cwd, i+1, cmd, extraArgs, env)
			results[i] = result{index: i + 1, cmd: cmd, code: code, err: err}
		}()
	}
	wg.Wait()

	firstFail := 0
	failures := []string{}
	for _, r := range results {
		if r.err != nil {
			return 1, r.err
		}
		if r.code != 0 {
			if firstFail == 0 {
				firstFail = r.code
			}
			failures = append(failures, fmt.Sprintf("#%d (%q) exit %d", r.index, r.cmd, r.code))
		}
	}
	if firstFail != 0 {
		return firstFail, fmt.Errorf("[TYA-E0903] task %q parallel: %s", taskName, strings.Join(failures, ", "))
	}
	return 0, nil
}

func runShellPrefixed(cwd string, index int, command string, extraArgs []string, env map[string]string) (int, error) {
	full := command
	if len(extraArgs) > 0 {
		full = command + " " + joinShellArgs(extraArgs)
	}
	cmd := exec.Command("/bin/sh", "-c", full)
	cmd.Dir = cwd
	cmd.Env = mergeEnv(env)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return 1, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return 1, err
	}
	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("failed to start /bin/sh: %v", err)
	}
	prefix := taskPrefix(index, command)
	var pipeWg sync.WaitGroup
	pipeWg.Add(2)
	go pumpPrefixed(stdoutPipe, os.Stdout, prefix, &pipeWg)
	go pumpPrefixed(stderrPipe, os.Stderr, prefix, &pipeWg)
	pipeWg.Wait()
	err = cmd.Wait()
	if err == nil {
		return 0, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode(), nil
	}
	return 1, err
}

func mergeEnv(overrides map[string]string) []string {
	if len(overrides) == 0 {
		return os.Environ()
	}
	out := os.Environ()
	index := map[string]int{}
	for i, entry := range out {
		if eq := strings.IndexByte(entry, '='); eq >= 0 {
			index[entry[:eq]] = i
		}
	}
	for key, value := range overrides {
		entry := key + "=" + value
		if i, ok := index[key]; ok {
			out[i] = entry
		} else {
			out = append(out, entry)
		}
	}
	return out
}

type watchSnapshot map[string]time.Time

func watchTask(root string, m *pkg.Manifest, name string, extra []string) (int, error) {
	task := m.Tasks[name]
	snap, err := snapshotTask(root, task)
	if err != nil {
		return 1, err
	}
	for {
		code, runErr := runTaskGraph(root, m, name, extra)
		if runErr != nil || code != 0 {
			return code, runErr
		}
		for {
			time.Sleep(200 * time.Millisecond)
			next, err := snapshotTask(root, task)
			if err != nil {
				return 1, err
			}
			if snapshotChanged(snap, next) {
				time.Sleep(150 * time.Millisecond)
				snap, _ = snapshotTask(root, task)
				break
			}
		}
	}
}

func snapshotTask(root string, task pkg.Task) (watchSnapshot, error) {
	out := watchSnapshot{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if shouldIgnoreWatchDir(rel) || matchesAny(task.Ignore, rel+"/") {
				return filepath.SkipDir
			}
			return nil
		}
		if !shouldWatchFile(rel, task.Watch) || matchesAny(task.Ignore, rel) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		out[rel] = info.ModTime()
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("[TYA-E0908] task %q watch failed: %v", task.Name, err)
	}
	return out, nil
}

func shouldIgnoreWatchDir(rel string) bool {
	base := filepath.Base(rel)
	if base == ".git" || base == "node_modules" || base == "_site" || base == "dist" || base == "build" || base == "target" {
		return true
	}
	return strings.HasPrefix(base, ".") && base != "."
}

func shouldWatchFile(rel string, patterns []string) bool {
	if len(patterns) > 0 {
		return matchesAny(patterns, rel)
	}
	if rel == "tya.toml" || strings.HasSuffix(rel, ".tya") {
		return true
	}
	for _, prefix := range []string{"src/", "tests/", "stdlib/", "examples/"} {
		if strings.HasPrefix(rel, prefix) {
			return true
		}
	}
	return false
}

func matchesAny(patterns []string, rel string) bool {
	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}
		if ok, err := matchWatchPattern(filepath.ToSlash(pattern), rel); err == nil && ok {
			return true
		}
	}
	return false
}

func matchWatchPattern(pattern, rel string) (bool, error) {
	if strings.HasSuffix(pattern, "/**") {
		return strings.HasPrefix(rel, strings.TrimSuffix(pattern, "**")), nil
	}
	if strings.Contains(pattern, "/**/") {
		parts := strings.SplitN(pattern, "/**/", 2)
		return strings.HasPrefix(rel, parts[0]+"/") && strings.HasSuffix(rel, strings.TrimPrefix(parts[1], "*")), nil
	}
	return filepath.Match(pattern, rel)
}

func snapshotChanged(a, b watchSnapshot) bool {
	if len(a) != len(b) {
		return true
	}
	for path, mod := range a {
		if !b[path].Equal(mod) {
			return true
		}
	}
	return false
}

func taskPrefix(index int, command string) string {
	const maxLen = 16
	c := command
	if len(c) > maxLen {
		c = c[:maxLen-1] + "…"
	}
	return fmt.Sprintf("[%d %s] ", index, c)
}

func pumpPrefixed(r io.Reader, w io.Writer, prefix string, wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		fmt.Fprintln(w, prefix+scanner.Text())
	}
}
