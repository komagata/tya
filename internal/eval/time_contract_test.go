package eval_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tya/internal/runner"
)

func TestRunTimeUnixFormatParse(t *testing.T) {
	out := runTimeProgram(t, strings.Join([]string{
		"import time",
		"t = time.Time().unix(1704067200)",
		"print(t.unix())",
		"print(t.format(\"rfc3339\"))",
		"print(time.Time(\"2024-01-01T00:00:00Z\").parse().unix())",
		"print(time.Time(\"2024-01-01\").parse(\"date\").format(\"date\"))",
		"",
	}, "\n"))
	want := "1704067200\n2024-01-01T00:00:00Z\n1704067200\n2024-01-01\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestRunTimeDurationArithmetic(t *testing.T) {
	out := runTimeProgram(t, strings.Join([]string{
		"import time",
		"a = time.Time().duration(1, { minutes: 1, milliseconds: 500 })",
		"b = time.Time().duration(2)",
		"print(a.seconds())",
		"print(a.milliseconds())",
		"print(a.add(b).seconds())",
		"print(a.sub(b).nanoseconds())",
		"",
	}, "\n"))
	want := "61.5\n61500\n63.5\n59500000000\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestRunTimeUtcLocalBoundaries(t *testing.T) {
	out := runTimeProgram(t, strings.Join([]string{
		"import time",
		"t = time.Time().unix(1704067200)",
		"local = t.local()",
		"print(t.utc().format(\"rfc3339\"))",
		"print(local.unix())",
		"",
	}, "\n"))
	want := "2024-01-01T00:00:00Z\n1704067200\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestRunTimeMonotonicElapsed(t *testing.T) {
	out := runTimeProgram(t, strings.Join([]string{
		"import time",
		"start = time.Time().monotonic()",
		"time.Time().sleep(time.Time().duration(0.001))",
		"elapsed = time.Time().monotonic().sub(start)",
		"print(elapsed.nanoseconds() >= 0)",
		"try",
		"  start.format(\"rfc3339\")",
		"catch err",
		"  print(err[\"code\"])",
		"",
	}, "\n"))
	want := "true\nmonotonic_format\n"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestRunTimeStructuredErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{name: "parse", src: "time.Time(\"bad\").parse()", want: "invalid_timestamp"},
		{name: "layout", src: "time.Time().unix(0).format(\"bad\")", want: "unknown_layout"},
		{name: "option", src: "time.Time().duration(0, { days: 1 })", want: "unknown_option"},
		{name: "wrong kind", src: "time.Time().unix(\"0\")", want: "invalid_seconds"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := runTimeProgram(t, "import time\ntry\n  "+tt.src+"\ncatch err\n  print(err[\"code\"])\n")
			if !strings.Contains(out, tt.want) {
				t.Fatalf("got %q, want %q", out, tt.want)
			}
		})
	}
}

func runTimeProgram(t *testing.T, src string) string {
	t.Helper()
	var out bytes.Buffer
	if err := runTimeProgramWithOutput(t, src, &out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func runTimeProgramError(t *testing.T, src string) error {
	t.Helper()
	var out bytes.Buffer
	return runTimeProgramWithOutput(t, src, &out)
}

func runTimeProgramWithOutput(t *testing.T, src string, out *bytes.Buffer) error {
	t.Helper()
	root := filepath.Clean(filepath.Join("..", ".."))
	stdlib, err := filepath.Abs(filepath.Join(root, "lib"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TYA_LIB_DIR", stdlib)
	path := filepath.Join(t.TempDir(), "time_eval.tya")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	_, err = runner.RunFile(path, strings.NewReader(""), out, nil)
	return err
}
