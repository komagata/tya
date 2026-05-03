package tests

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tya/internal/runner"
)

func TestExamplesGolden(t *testing.T) {
	root := ".."
	cases := map[string]string{
		"args.tya":              "2\nok\n",
		"arithmetic.tya":        "5\n14\n20\n2.5\n2\n-3\ntrue\nnil\nnext year: 21\n",
		"array.tya":             "3\n1\nnil\n4\n4\n3\n20\n",
		"array_function.tya":    "6\n2\n2\ntrue\nfalse\n10\n",
		"class.tya":             "komagata\nHello, komagata\nTya\n",
		"classic/array_sum.tya": "6\n",
		"classic/factorial.tya": "120\n",
		"classic/fib.tya":       "55\n",
		"classic/fizzbuzz.tya":  "1\n2\nFizz\n4\nBuzz\nFizz\n7\n8\nFizz\nBuzz\n11\nFizz\n13\n14\nFizzBuzz\n",
		"classic/gcd.tya":       "6\n",
		"classic/prime.tya":     "true\n",
		"convert.tya":           "20\n42\n2.5\n12\n12.5\n[1, 2]\n",
		"dict_set.tya":          "komagata\n0\n2\ntrue\n0\n",
		"equal.tya":             "false\ntrue\nfalse\n",
		"error.tya":             "error: file not found\nfile not found\n",
		"for.tya":               "12\n0:2\n1:4\n2:6\n",
		"for_object.tya":        "komagata\n2\n",
		"function.tya":          "Hello, komagata\n",
		"hello.tya":             "Hello, Tya\n",
		"if.tya":                "komagata\nmissing\n",
		"logic.tya":             "match\nanonymous\ntrue\n",
		"method.tya":            "1\n2\n",
		"multiple_return.tya":   "komagata\nempty user\n",
		"object.tya":            "Hello, komagata\n",
		"object_builtin.tya":    "true\n2\n2\nfalse\nnil\n",
		"object_inline.tya":     "Hello, komagata\n20\n",
		"prelude.tya":           "1\nnil\n[1, 2]\n6\n",
		"return.tya":            "4\n",
		"string.tya":            "hello-tya\nhello,Tya\ntrue\ntrue\ntrue\n6\n2\nquote: \"tya\"\ny\n",
		"try.tya":               "komagata\nempty user\n",
		"use_module.tya":        "Hello, komagata\n",
		"use_module_decl.tya":   "foo\nbar\n",
		"while.tya":             "10\n11\n",
	}
	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			if name == "args.tya" {
				t.Setenv("TYA_EXAMPLE", "ok")
			}
			var out bytes.Buffer
			args := []string(nil)
			if name == "args.tya" {
				args = []string{"a", "b"}
			}
			err := runner.RunFile(filepath.Join(root, "examples", name), strings.NewReader("komagata\n"), &out, args)
			if err != nil {
				t.Fatal(err)
			}
			if out.String() != want {
				t.Fatalf("got %q, want %q", out.String(), want)
			}
		})
	}
}

func TestFileExampleGolden(t *testing.T) {
	var out bytes.Buffer
	err := runner.RunFile(filepath.Join("..", "examples", "file.tya"), nil, &out, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != "true\nHello from Tya\n" {
		t.Fatalf("got %q", out.String())
	}
	_ = os.Remove("/tmp/tya_example.txt")
}
