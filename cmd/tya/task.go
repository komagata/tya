package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

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
	extra := args[1:]
	task, ok := m.Tasks[name]
	if !ok {
		return 1, fmt.Errorf("[TYA-E0900] no task %q defined in tya.toml", name)
	}
	switch task.Kind {
	case "string":
		return runShell(root, name, 0, task.String, extra)
	case "array":
		for i, cmd := range task.Array {
			code, runErr := runShell(root, name, i+1, cmd, extra)
			if runErr != nil {
				return code, runErr
			}
			if code != 0 {
				return code, fmt.Errorf("[TYA-E0901] task %q command #%d (%q) failed with exit code %d", name, i+1, cmd, code)
			}
		}
		return 0, nil
	default:
		return 1, fmt.Errorf("internal: task %q has unknown kind %q", name, task.Kind)
	}
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
	if t.Kind == "array" {
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

func runShell(cwd, taskName string, index int, command string, extraArgs []string) (int, error) {
	full := command
	if len(extraArgs) > 0 {
		full = command + " " + joinShellArgs(extraArgs)
	}
	cmd := exec.Command("/bin/sh", "-c", full)
	cmd.Dir = cwd
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
// four-character sequence '\'' which terminates the quoted region,
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
