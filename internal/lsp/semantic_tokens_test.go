package lsp

import "testing"

func TestSemanticTokensClassifyStructAndRecord(t *testing.T) {
	src := "struct User\n  name\n\nrecord Point\n  x\n"
	tokens := decodeSemanticTokens(SemanticTokensFor(src).Data)

	assertSemanticToken(t, tokens, 0, 0, len("struct"), stKeyword, 0)
	assertSemanticToken(t, tokens, 0, 7, len("User"), stClass, 1<<2)
	assertSemanticToken(t, tokens, 3, 0, len("record"), stKeyword, 0)
	assertSemanticToken(t, tokens, 3, 7, len("Point"), stClass, 1<<2)
}

type decodedSemanticToken struct {
	line   int
	col    int
	length int
	kind   int
	mods   uint32
}

func decodeSemanticTokens(data []uint32) []decodedSemanticToken {
	out := []decodedSemanticToken{}
	line := 0
	col := 0
	for i := 0; i+4 < len(data); i += 5 {
		line += int(data[i])
		if data[i] == 0 {
			col += int(data[i+1])
		} else {
			col = int(data[i+1])
		}
		out = append(out, decodedSemanticToken{
			line:   line,
			col:    col,
			length: int(data[i+2]),
			kind:   int(data[i+3]),
			mods:   data[i+4],
		})
	}
	return out
}

func assertSemanticToken(t *testing.T, tokens []decodedSemanticToken, line, col, length, kind int, mods uint32) {
	t.Helper()
	for _, tok := range tokens {
		if tok.line == line && tok.col == col {
			if tok.length != length || tok.kind != kind || tok.mods != mods {
				t.Fatalf(
					"token at %d:%d = length %d kind %d mods %d, want length %d kind %d mods %d",
					line,
					col,
					tok.length,
					tok.kind,
					tok.mods,
					length,
					kind,
					mods,
				)
			}
			return
		}
	}
	t.Fatalf("missing token at %d:%d in %#v", line, col, tokens)
}
