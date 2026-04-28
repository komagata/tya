package runner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"tya/internal/checker"
	"tya/internal/eval"
	"tya/internal/lexer"
	"tya/internal/parser"
)

var fileNameRE = regexp.MustCompile(`^[a-z][a-z0-9_]*\.tya$`)

func ValidateFileName(path string) error {
	if filepath.Ext(path) != ".tya" || !fileNameRE.MatchString(filepath.Base(path)) {
		return fmt.Errorf("invalid Tya file name: %s", filepath.Base(path))
	}
	return nil
}

func RunFile(path string, in io.Reader, out io.Writer, args []string) error {
	if err := ValidateFileName(path); err != nil {
		return err
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	toks, errs := lexer.Lex(withPrelude(path, string(src)))
	if len(errs) > 0 {
		return errs[0]
	}
	prog, err := parser.Parse(toks)
	if err != nil {
		return err
	}
	if err := checker.Check(prog); err != nil {
		return err
	}
	return eval.RunWithIO(prog, in, out, args)
}

func withPrelude(path, src string) string {
	candidates := []string{
		filepath.Join(filepath.Dir(path), "..", "stdlib", "prelude.tya"),
		filepath.Join("stdlib", "prelude.tya"),
	}
	for _, preludePath := range candidates {
		prelude, err := os.ReadFile(preludePath)
		if err != nil {
			continue
		}
		if strings.HasSuffix(string(prelude), "\n") {
			return string(prelude) + src
		}
		return string(prelude) + "\n" + src
	}
	return src
}
