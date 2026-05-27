package eval_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tya/internal/runner"
)

func TestRunHmacKnownVectors(t *testing.T) {
	out := runHmacProgram(t, strings.Join([]string{
		"import hmac/* as hmac",
		"message = \"The quick brown fox jumps over the lazy dog\"",
		"print(hmac.Hmac(\"sha256\", \"key\").hexdigest(message))",
		"print(hmac.Hmac(\"sha384\", \"key\").hexdigest(message))",
		"print(hmac.Hmac(\"sha512\", \"key\").hexdigest(message))",
		"print(hmac.Hmac(\"sha256\", \"key\").base64digest(message))",
		"",
	}, "\n"))
	want := strings.Join([]string{
		"f7bc83f430538424b13298e6aa6fb143ef4d59a14946175997479dbc2d1a3cd8",
		"d7f4727e2c0b39ae0f1e40cc96f60242d5b7801841cea6fc592c5d3e1ae50700582a96cf35e1e554995fe4e03381c237",
		"b42af09057bac1e2d41708e48a902e09b5ff7f12ab428a4fe86653c73dd248fb82f948a549f7b791a5b41915ee4d1ec3935357e4e2317250d0372afa2ebeeb3a",
		"97yD9DBThCSxMpjmqm+xQ+9NWaFJRhdZl0edvC0aPNg=",
		"",
	}, "\n")
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestRunHmacStringAndBytesInputs(t *testing.T) {
	out := runHmacProgram(t, strings.Join([]string{
		"import hmac/* as hmac",
		"text = hmac.Hmac(\"sha256\", \"key\").hexdigest(\"message\")",
		"raw = hmac.Hmac(\"sha256\", b\"key\").hexdigest(b\"message\")",
		"print(text == raw)",
		"",
	}, "\n"))
	if out != "true\n" {
		t.Fatalf("got %q", out)
	}
}

func TestRunHmacVerifyEncodings(t *testing.T) {
	out := runHmacProgram(t, strings.Join([]string{
		"import hmac/* as hmac",
		"message = \"The quick brown fox jumps over the lazy dog\"",
		"expected_hex = \"f7bc83f430538424b13298e6aa6fb143ef4d59a14946175997479dbc2d1a3cd8\"",
		"raw = hmac.Hmac(\"sha256\", \"key\").digest(message)",
		"b64 = \"97yD9DBThCSxMpjmqm+xQ+9NWaFJRhdZl0edvC0aPNg=\"",
		"print(hmac.Hmac(\"sha256\", \"key\").verify(message, raw, { encoding: \"raw\" }))",
		"print(hmac.Hmac(\"sha256\", \"key\").verify(message, expected_hex))",
		"print(hmac.Hmac(\"sha256\", \"key\").verify(message, b64, { encoding: \"base64\" }))",
		"print(hmac.Hmac(\"sha256\", \"key\").verify(message, \"00\"))",
		"",
	}, "\n"))
	if out != "true\ntrue\ntrue\nfalse\n" {
		t.Fatalf("got %q", out)
	}
}

func TestRunHmacStructuredErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{name: "algorithm", src: "hmac.Hmac(\"md5\", \"key\").hexdigest(\"message\")", want: "unsupported algorithm"},
		{name: "malformed hex", src: "hmac.Hmac(\"sha256\", \"key\").verify(\"message\", \"f\")", want: "odd-length input"},
		{name: "malformed base64", src: "hmac.Hmac(\"sha256\", \"key\").verify(\"message\", \"????\", { encoding: \"base64\" })", want: "invalid character"},
		{name: "unknown option", src: "hmac.Hmac(\"sha256\", \"key\").verify(\"message\", \"00\", { bad: true })", want: "unknown option bad"},
		{name: "wrong kind", src: "hmac.Hmac(\"sha256\", 1).hexdigest(\"message\")", want: "value must be a string or bytes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runHmacProgramError(t, "import hmac/* as hmac\n"+tt.src+"\n")
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func runHmacProgram(t *testing.T, src string) string {
	t.Helper()
	var out bytes.Buffer
	if err := runHmacProgramWithOutput(t, src, &out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func runHmacProgramError(t *testing.T, src string) error {
	t.Helper()
	var out bytes.Buffer
	return runHmacProgramWithOutput(t, src, &out)
}

func runHmacProgramWithOutput(t *testing.T, src string, out *bytes.Buffer) error {
	t.Helper()
	root := filepath.Clean(filepath.Join("..", ".."))
	stdlib, err := filepath.Abs(filepath.Join(root, "stdlib"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TYA_STDLIB_DIR", stdlib)
	path := filepath.Join(t.TempDir(), "hmac_eval.tya")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	_, err = runner.RunFile(path, strings.NewReader(""), out, nil)
	return err
}
