package token

type Type string

const (
	EOF     Type = "EOF"
	ILLEGAL Type = "ILLEGAL"

	IDENT  Type = "IDENT"
	INT    Type = "INT"
	FLOAT  Type = "FLOAT"
	STRING Type = "STRING"

	NEWLINE Type = "NEWLINE"
	INDENT  Type = "INDENT"
	DEDENT  Type = "DEDENT"

	ASSIGN   Type = "="
	EQ       Type = "=="
	NEQ      Type = "!="
	LT       Type = "<"
	LTE      Type = "<="
	GT       Type = ">"
	GTE      Type = ">="
	COLON    Type = ":"
	COMMA    Type = ","
	DOT      Type = "."
	PLUS     Type = "+"
	MINUS    Type = "-"
	STAR     Type = "*"
	SLASH    Type = "/"
	PERCENT  Type = "%"
	ARROW    Type = "->"
	LPAREN   Type = "("
	RPAREN   Type = ")"
	LBRACKET Type = "["
	RBRACKET Type = "]"
	LBRACE   Type = "{"
	RBRACE   Type = "}"
)

type Token struct {
	Type   Type
	Lexeme string
	Line   int
	Col    int
}
