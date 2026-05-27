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
			want: "main = -> true\n",
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
			category: "function-head",
			name:     "nil default arguments are preserved",
			input: strings.Join([]string{
				"setup = value = nil -> value",
				"class Codec",
				"  initialize: value = nil ->",
				"    self.value = value",
				"  encode: value = nil, padded = true -> value",
				"",
			}, "\n"),
			want: strings.Join([]string{
				"setup = value = nil -> value",
				"",
				"class Codec",
				"  initialize: value = nil ->",
				"    self.value = value",
				"",
				"  encode: value = nil, padded = true -> value",
				"",
			}, "\n"),
		},
		{
			category: "function-body",
			name:     "single expression block function becomes one line",
			input: strings.Join([]string{
				"foo = ->",
				"  \"aaa\"",
				"",
			}, "\n"),
			want: "foo = -> \"aaa\"\n",
		},
		{
			category: "function-body",
			name:     "final return block function becomes one line",
			input: strings.Join([]string{
				"foo = ->",
				"  return \"aaa\"",
				"",
			}, "\n"),
			want: "foo = -> \"aaa\"\n",
		},
		{
			category: "function-body",
			name:     "multi statement function stays block bodied",
			input: strings.Join([]string{
				"foo = ->",
				"  value = \"aaa\"",
				"  return value",
				"",
			}, "\n"),
			want: strings.Join([]string{
				"foo = ->",
				"  value = \"aaa\"",
				"  value",
				"",
			}, "\n"),
		},
		{
			category: "function-body",
			name:     "long single expression function stays block bodied",
			input: strings.Join([]string{
				"foo = -> \"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\"",
				"",
			}, "\n"),
			want: strings.Join([]string{
				"foo = ->",
				"  \"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\"",
				"",
			}, "\n"),
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
				"  initialize: () ->",
				"    super()",
				"",
			}, "\n"),
			want: strings.Join([]string{
				"class Admin extends User",
				"  initialize: -> super()",
				"",
			}, "\n"),
		},
		{
			category: "class",
			name:     "declaring class receiver becomes Self",
			input: strings.Join([]string{
				"class Box",
				"  static build: () -> Box.new()",
				"",
			}, "\n"),
			want: strings.Join([]string{
				"class Box",
				"  static build: -> Self.new()",
				"",
			}, "\n"),
		},
		{
			category: "class",
			name:     "single expression class method becomes one line",
			input: strings.Join([]string{
				"class Foo",
				"  foo: ->",
				"    \"aaa\"",
				"",
			}, "\n"),
			want: strings.Join([]string{
				"class Foo",
				"  foo: -> \"aaa\"",
				"",
			}, "\n"),
		},
		{
			category: "class",
			name:     "final return class method becomes one line",
			input: strings.Join([]string{
				"class Foo",
				"  foo: ->",
				"    return \"aaa\"",
				"",
			}, "\n"),
			want: strings.Join([]string{
				"class Foo",
				"  foo: -> \"aaa\"",
				"",
			}, "\n"),
		},
		{
			category: "class",
			name:     "single expression static method becomes one line",
			input: strings.Join([]string{
				"class Foo",
				"  static foo: ->",
				"    \"aaa\"",
				"",
			}, "\n"),
			want: strings.Join([]string{
				"class Foo",
				"  static foo: -> \"aaa\"",
				"",
			}, "\n"),
		},
		{
			category: "class",
			name:     "single expression constructor becomes one line",
			input: strings.Join([]string{
				"class Foo",
				"  initialize: ->",
				"    \"aaa\"",
				"",
			}, "\n"),
			want: strings.Join([]string{
				"class Foo",
				"  initialize: -> \"aaa\"",
				"",
			}, "\n"),
		},
		{
			category: "interface",
			name:     "body free requirement stays body free and default becomes one line",
			input: strings.Join([]string{
				"interface Named",
				"  name: ->",
				"  fallback: ->",
				"    \"default\"",
				"",
			}, "\n"),
			want: strings.Join([]string{
				"interface Named",
				"  name: ->",
				"",
				"  fallback: -> \"default\"",
				"",
			}, "\n"),
		},
		{
			category: "interface",
			name:     "adjacent requirements have one blank line",
			input: strings.Join([]string{
				"interface Iterator",
				"  has_next: ->",
				"  next: ->",
				"",
			}, "\n"),
			want: strings.Join([]string{
				"interface Iterator",
				"  has_next: ->",
				"",
				"  next: ->",
				"",
			}, "\n"),
		},
		{
			category: "imports",
			name:     "multiple imports group and sort stdlib before user imports",
			input: strings.Join([]string{
				"import app",
				"import file",
				"main = -> true",
				"",
			}, "\n"),
			want: strings.Join([]string{
				"import",
				"  file",
				"  app",
				"",
				"main = -> true",
				"",
			}, "\n"),
		},
		{
			category: "imports",
			name:     "single wildcard import stays one line",
			input:    "import base64/* as b64\n",
			want:     "import base64/* as b64\n",
		},
		{
			category: "imports",
			name:     "grouped imports accept wildcard and aliases",
			input: strings.Join([]string{
				"import",
				"  net/http/client as client",
				"  base64/*",
				"",
			}, "\n"),
			want: strings.Join([]string{
				"import",
				"  base64/*",
				"  net/http/client as client",
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
