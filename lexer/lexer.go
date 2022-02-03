package lexer

// 소스코드->토큰->추상구문트리
// 소스코드를 토큰열로 변환
// 렉싱(어휘분석)
// 가장먼저 토큰을 정의해야 한다
// 렉서는 소스코드를 입력으로 받고 표헌하는 토큰열을 결과로 출력
// 흝어가면서 토큰을 인식할 때마다 결과를 출력한다.
// 버퍼도 필요업고 토큰을 저장할 필요 없다.
// 상용버전에서는 파일 이름과 행번호를 토큰에 붙여, 렉싱에서 생긴 에러와 파싱에서 생긴 에러를 더 쉽게 추적
import "MonkeyKids/token"

// ASCII 문자만 지원
// 유니코드와 UTF8을 지원하기 위해서는 ch를 rune타입으로 바꾸고 다음 문자들을 읽는 방식을 바꿔야 한다.
// 유니코드는 문자 하나에 여러개의 바이트가 할당
type Lexer struct {
	input string
	// 입력 문자를 가리키는 포인터가 2개인 이유는 다음 처리 대상을 알아내려면 입력 문자열에서 다음 문자를 '미리 살펴봄' 과 동시에
	// 현재 문자를 보존할 수 있어야 한다.
	position     int  // 입력에서 현재 위치(현재 문자를 가리킴)
	readPosition int  // 입력에서 현재 읽는 위치(현재 문자의 다음을 가리킴)
	ch           byte // 현재 조사하고 있는 문자
}

func New(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		// 만약 끝에 도달시 0을 삽입
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition // 항상 다음에 읽어야할  위치
	l.readPosition += 1         // 항상 마지막으로 읶은 위치
}
func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	l.skipWhitespace()
	switch l.ch {
	// 두문자 토큰을 case문 하나를 추가 하지 앟는 이유
	// byte인 l.ch를 문자열인 "=="과 비교가 불가
	case '=':
		if l.peekChar() == '=' {
			// readChar 호출전에 l.ch를 지역 변수에 저장
			// 현재 문자를 기억한 상태에서 안전하게 렉서를 진행
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.EQ, Literal: literal}
		} else {
			tok = newToken(token.ASSIGN, l.ch)
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.NOT_EQ, Literal: literal}
		} else {
			tok = newToken(token.BANG, l.ch)
		}
	case ';':
		tok = newToken(token.SEMICOLON, l.ch)
	case '(':
		tok = newToken(token.LPAREN, l.ch)
	case ')':
		tok = newToken(token.RPAREN, l.ch)
	case ',':
		tok = newToken(token.COMMA, l.ch)
	case '+':
		tok = newToken(token.PLUS, l.ch)
	case '-':
		tok = newToken(token.MINUS, l.ch)
	case '/':
		tok = newToken(token.SLASH, l.ch)
	case '<':
		tok = newToken(token.LT, l.ch)
	case '>':
		tok = newToken(token.GT, l.ch)
	case '*':
		tok = newToken(token.ASTERISK, l.ch)
	case '{':
		tok = newToken(token.LBRACE, l.ch)
	case '}':
		tok = newToken(token.RBRACE, l.ch)
		// 문자열
	case '"':
		tok.Type = token.STRING
		tok.Literal = l.readString()
		// 배열
		// 개별적으로 접근이 가능
	case '[':
		tok = newToken(token.LBRACKET, l.ch)
	case ']':
		tok = newToken(token.RBRACKET, l.ch)
	case ':':
		tok = newToken(token.COLON, l.ch)

	case 0:
		tok.Literal = ""
		tok.Type = token.EOF
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			return tok // 조기종료(꼭 필요)
		} else if isDigit(l.ch) {
			tok.Type = token.INT
			tok.Literal = l.readNumber()
			return tok
		} else {
			tok = newToken(token.ILLEGAL, l.ch)
		}
	}
	l.readChar()
	return tok
}

func newToken(tokenType token.TokenType, ch byte) token.Token {
	return token.Token{Type: tokenType, Literal: string(ch)}
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

// readChar과 비슷,l.position과 l.readPosition을 증가시키지 않는다.
//다음에 나올 입력을 미리 살펴보고 싶은 것
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) readString() string {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
	}
	return l.input[position:l.position]
}
