package doc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStdlibAPIDocCoverage(t *testing.T) {
	var paths []string
	root := filepath.Join("..", "..", "lib")
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".tya") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var missing []string
	for _, path := range paths {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(string(src), "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(line, "class ") || strings.HasPrefix(line, "interface ") {
				if i == 0 || !strings.HasPrefix(lines[i-1], "#") {
					missing = append(missing, path+":"+trimmed)
				}
				continue
			}
			if strings.HasPrefix(line, "  private ") {
				continue
			}
			if isStdlibMethodLine(line) {
				if i == 0 || !strings.HasPrefix(lines[i-1], "  #") {
					missing = append(missing, path+":"+trimmed)
				}
			}
		}
	}
	if len(missing) > 0 {
		t.Fatalf("missing stdlib API doc comments:\n%s", strings.Join(missing, "\n"))
	}
}

func isStdlibMethodLine(line string) bool {
	if !strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "    ") {
		return false
	}
	trimmed := strings.TrimSpace(line)
	if !strings.Contains(trimmed, " =") {
		return false
	}
	if strings.HasPrefix(trimmed, "static ") {
		trimmed = strings.TrimPrefix(trimmed, "static ")
	}
	if strings.HasPrefix(trimmed, "override ") {
		trimmed = strings.TrimPrefix(trimmed, "override ")
	}
	name := strings.TrimSpace(strings.SplitN(trimmed, "=", 2)[0])
	if name == "" || strings.Contains(name, " ") {
		return false
	}
	return strings.Contains(trimmed, "->") || name == "initialize" || strings.HasSuffix(name, "?")
}

func TestStdlibAPIDocsIncludeRepresentativePackages(t *testing.T) {
	paths := []string{
		filepath.Join("..", "..", "lib", "math", "math.tya"),
		filepath.Join("..", "..", "lib", "file", "file.tya"),
		filepath.Join("..", "..", "lib", "json", "json.tya"),
		filepath.Join("..", "..", "lib", "toml", "toml.tya"),
		filepath.Join("..", "..", "lib", "net", "http", "server.tya"),
		filepath.Join("..", "..", "lib", "net", "socket", "socket.tya"),
		filepath.Join("..", "..", "lib", "template", "template.tya"),
		filepath.Join("..", "..", "lib", "unittest", "test_case.tya"),
	}
	report, err := ExtractReport(paths)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"Math.abs",
		"File.read",
		"Json.parse",
		"Toml.parse",
		"Server.get",
		"Socket.connect",
		"Template.render",
		"TestCase.assert_equal",
	}
	have := map[string]bool{}
	for _, item := range report.Items {
		have[item.Name] = strings.TrimSpace(item.RawDoc) != ""
	}
	for _, name := range want {
		if !have[name] {
			t.Fatalf("missing representative stdlib API doc for %s", name)
		}
	}
}

func TestStdlibHTMLDocsGenerateRepresentativePages(t *testing.T) {
	root := filepath.Join("..", "..", "lib")
	var paths []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".tya") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	report, err := ExtractReport(paths)
	if err != nil {
		t.Fatal(err)
	}
	if HasErrorDiagnostics(report.Diagnostics) {
		t.Fatalf("unexpected stdlib doc error diagnostics: %#v", report.Diagnostics)
	}
	out := t.TempDir()
	site := &Site{Title: "Standard Library API", Items: report.Items}
	if err := site.Generate(out, nil); err != nil {
		t.Fatal(err)
	}
	index, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Json", "File", "Math", "Template", "Server", "Address"} {
		if !strings.Contains(string(index), want) {
			t.Fatalf("generated stdlib index missing %s", want)
		}
	}
	for _, tt := range []struct {
		name string
		want string
	}{
		{"Json", "Json.parse"},
		{"File", "File.read"},
		{"Math", "Math.abs"},
		{"Template", "Template.render"},
		{"Server", "Server.get"},
		{"Address", "Address.parse"},
	} {
		var page DocItem
		for _, item := range report.Items {
			if item.Kind == "class" && item.Name == tt.name {
				page = item
				break
			}
		}
		if page.Name == "" {
			t.Fatalf("missing class page item for %s", tt.name)
		}
		body, err := os.ReadFile(filepath.Join(out, "items", pageFileName(page)))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), tt.want) {
			t.Fatalf("generated stdlib page %s missing %s", pageFileName(page), tt.want)
		}
	}
}
