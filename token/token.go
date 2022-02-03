package token

// 서로 다른 여러 값을 TokenType으로 필요한 만큼 사용가능
// 여러 토큰을 서로 쉽게 구별 가능
// int나 byte의 성능 이점을 따라 가기는 힘듬
type TokenType string

type Token struct {
	Type    TokenType
	Literal string
}

const (
	ILLEGAL = "ILLEGAL" // 토큰이나 문자를 렉서가 알 수 없다는 것
	EOF     = "EOF"     // 파일의 끝

	// 식별자 + 리터럴
	IDENT  = " IDENT"
	INT    = "INT"
	STRING = "STRING"

	// 연산자
	ASSIGN   = "="
	PLUS     = "+"
	MINUS    = "-"
	BANG     = "!"
	ASTERISK = "*"
	SLASH    = "/"
	LT       = "<"
	GT       = ">"

	// 구분자
	COMMA     = ","
	SEMICOLON = ";"

	LPAREN   = "("
	RPAREN   = ")"
	LBRACE   = "{"
	RBRACE   = "}"
	EQ       = "=="
	NOT_EQ   = "!="
	LBRACKET = "["
	RBRACKET = "]"
	COLON    = ":"

	// 예약어
	FUNCTION = "FUNCTION"
	LET      = "LET"
	TRUE     = "TRUE"
	FALSE    = "FALSE"
	IF       = "IF"
	ELSE     = "ELSE"
	RETURN   = "RETURN"
)

var keywords = map[string]TokenType{
	"fn":     FUNCTION,
	"let":    LET,
	"true":   TRUE,
	"false":  FALSE,
	"if":     IF,
	"else":   ELSE,
	"return": RETURN,
}

// 주어진 식별자가 예약어인지 확인
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
