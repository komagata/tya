package doc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStdlibAPIDocCoverage(t *testing.T) {
	var paths []string
	root := filepath.Join("..", "..", "stdlib")
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
		filepath.Join("..", "..", "stdlib", "math", "Math.tya"),
		filepath.Join("..", "..", "stdlib", "file", "File.tya"),
		filepath.Join("..", "..", "stdlib", "json", "Json.tya"),
		filepath.Join("..", "..", "stdlib", "toml", "Toml.tya"),
		filepath.Join("..", "..", "stdlib", "net", "http", "Server.tya"),
		filepath.Join("..", "..", "stdlib", "net", "socket", "Socket.tya"),
		filepath.Join("..", "..", "stdlib", "template", "Template.tya"),
		filepath.Join("..", "..", "stdlib", "unittest", "TestCase.tya"),
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
