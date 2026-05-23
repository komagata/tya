package formatter

import (
	"strings"
	"testing"
)

func TestRewriteCatalog(t *testing.T) {
	tests := []struct {
		category string
		name     string
		input    string
		want     string
	}{
		{
			category: "function-head",
			name:     "zero-argument block function omits parentheses",
			input: strings.Join([]string{
				"main = () ->",
				"  return true",
				"",
			}, "\n"),
			want: strings.Join([]string{
				"main = ->",
				"  return true",
				"",
			}, "\n"),
		},
		{
			category: "function-head",
			name:     "zero-argument expression function omits parentheses",
			input:    "answer = () -> 42\n",
			want:     "answer = -> 42\n",
		},
		{
			category: "function-head",
			name:     "parenthesized params and trailing comma normalize",
			input:    "add = (a, b,) -> a + b\n",
			want:     "add = a, b -> a + b\n",
		},
		{
			category: "collection",
			name:     "array trailing comma is removed",
			input:    "items = [1, 2,]\n",
			want:     "items = [1, 2]\n",
		},
		{
			category: "collection",
			name:     "dictionary trailing comma is removed",
			input:    "user = { name: \"Ada\", age: 20, }\n",
			want:     "user = { name: \"Ada\", age: 20 }\n",
		},
		{
			category: "call",
			name:     "call trailing comma is removed",
			input:    "print(add(1, 2,))\n",
			want:     "print(add(1, 2))\n",
		},
		{
			category: "string",
			name:     "single-quoted string becomes double-quoted",
			input:    "name = 'Tya'\n",
			want:     "name = \"Tya\"\n",
		},
		{
			category: "string",
			name:     "single-quoted interpolation braces stay literal",
			input:    "template = 'Hello {name}'\n",
			want:     "template = \"Hello {{name}}\"\n",
		},
		{
			category: "class",
			name:     "legacy class inheritance operator becomes extends",
			input: strings.Join([]string{
				"class Admin < User",
				"  initialize = () ->",
				"    super()",
				"",
			}, "\n"),
			want: strings.Join([]string{
				"class Admin extends User",
				"  initialize = ->",
				"    super()",
				"",
			}, "\n"),
		},
		{
			category: "class",
			name:     "declaring class receiver becomes Self",
			input: strings.Join([]string{
				"class Box",
				"  static build = () -> Box.new()",
				"",
			}, "\n"),
			want: strings.Join([]string{
				"class Box",
				"  static build = -> Self.new()",
				"",
			}, "\n"),
		},
		{
			category: "imports",
			name:     "stdlib imports sort before user imports",
			input: strings.Join([]string{
				"import app",
				"import file",
				"main = -> true",
				"",
			}, "\n"),
			want: strings.Join([]string{
				"import file",
				"",
				"import app",
				"",
				"main = -> true",
				"",
			}, "\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.category+"/"+tt.name, func(t *testing.T) {
			got, err := unparseSource(t, tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got:\n%swant:\n%s", got, tt.want)
			}
			again, err := unparseSource(t, got)
			if err != nil {
				t.Fatal(err)
			}
			if again != got {
				t.Fatalf("not idempotent:\nfirst:\n%s\nsecond:\n%s", got, again)
			}
		})
	}
}
