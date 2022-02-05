package parser

/*
퍼서는 입력 데이터를 받아 자료구조를 만들어 내는 소프트웨어 컴포넌트이다.
파서는 자료구조를 만들멸서 입력에 대응하는 구조화된 표현을 더하기도, 구문이 올바른지 검사하기도 한다.
보통 파서 앞에 어휘분석기를 떠로 둔다.
코드가 데이터고 데이터가 코드이다.
파싱에는 하향식, 상향식전략이 있다.
얼리 파싱, 예측성 파싱, 재귀적 하향 파싱
*/
// 여기서는 재귀적하향파싱을 사용
// AST의 루트 노드를 생성하는 것으로 시작해서 점처 아래쪽으로 파싱해나간다.
// let <identifier> = <expression>;
// 표현식은 값을 만들지만, 명령문을 그렇지 않다.
import (
	"MonkeyKids/ast"
	"MonkeyKids/lexer"
	"MonkeyKids/token"
	"fmt"
	"strconv"
)

// 상수 블록에서 INDEX가 가장 마지막행에 있다는 것이 중요
// iota를 사용한 상수 블록에서 가장 마지막행에 있기에 INDEX는 우선순위가 가장 높아진다.
const (
	// 연산자 우선순위 정의
	_ int = iota // iota를 이용해 뒤에 나오는 상수에게 1씩 증가하는 숫자를 값으로 제공
	// _ 는 0이후 나오는 숫자는 1~8
	// *연산자가 == 연산자보다 우선순위가 높은가?
	// 전위 연산자가 호출 표현식보다우선순위가 높은가?
	LOWEST
	EQUALS      // ==
	LESSGREATER // > or <
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X or !X
	CALL        // myFunction(X)
	INDEX       // array[index]
)

type Parser struct {
	l              *lexer.Lexer // 현재의 렉서 인스턴스를 가리키는 포인터
	curToken       token.Token  // 현재 토큰
	peekToken      token.Token  // 그다음 토큰
	errors         []string     // 문자열 슬라이스
	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn
}

type (
	prefixParseFn func() ast.Expression               // 전위 파싱 함수
	infixParseFn  func(ast.Expression) ast.Expression // 중위 파싱 함수 : 중위 연산자의 좌측에 위치

)

// 자기 설명적
// 파서는 반복적으로 토큰을 진행시키면서 현재의 토큰을 검사해 다음에 무엇을 해야할지 결정해야 한다.
// 다음의 할일 이란 또 다른 파싱 함수를 호출하거나 에러를 내는 것
// 그 후에 각각의 파싱 함수는 자기가 할 일을 수행하고 보통은 AST노드를 생성
// 그리고 다시 parseProgram 내의 메인루프가 토큰을 진행하고 다음에 무엇을 해야할지 결정해야 한다.
func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l, errors: []string{}}
	// 토큰을 2개 읽어서 curToken, peekToken 을 세팅
	p.nextToken()
	p.nextToken()

	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.INT, p.parseIntegerLiteral)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)

	p.registerPrefix(token.TRUE, p.parseBoolean)
	p.registerPrefix(token.FALSE, p.parseBoolean)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.IF, p.parseIfExpression)
	p.registerPrefix(token.FUNCTION, p.parseFunctionExpression)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.LBRACE, p.parseHashLiteral)
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral)

	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	p.registerInfix(token.PLUS, p.parseInfixExpression)
	p.registerInfix(token.MINUS, p.parseInfixExpression)
	p.registerInfix(token.SLASH, p.parseInfixExpression)
	p.registerInfix(token.ASTERISK, p.parseInfixExpression)
	p.registerInfix(token.EQ, p.parseInfixExpression)
	p.registerInfix(token.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(token.LT, p.parseInfixExpression)
	p.registerInfix(token.GT, p.parseInfixExpression)
	p.registerInfix(token.LPAREN, p.parseCallExpression)
	p.registerInfix(token.LBRACKET, p.parseIndexExpression)
	return p
}

func (p *Parser) peekError(t token.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead", t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}
	// token.EOF를 만날때 까지 모든 토큰을 대상으로 for-loop문을 반복적으로 호출
	for p.curToken.Type != token.EOF {
		stmt := p.ParseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}
	return program
}

// 파서가 표현식을 파싱할 수 있게 만든다.
// 명령문이 LET, RETURN 밖에 없기 때문에 ,이 두경우가 아닐경우 표현식문으로 파싱
func (p *Parser) ParseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.LET:
		return p.parseLetStatement()
	case token.RETURN:
		return p.parseReturnStatement()

	default:
		return p.parseExpressionStatement()

	}
}

func (p *Parser) parseLetStatement() *ast.LetStatement {
	stmt := &ast.LetStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	if fl, ok := stmt.Value.(*ast.FunctionLiteral); ok {
		fl.Name = stmt.Name.Value
	}

	for !p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

// 단정 함수
// 다음 토큰 타입를 검사해 토큰 간의 순서를 올바르게 강제할 용도로 사용
func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	p.nextToken()

	stmt.ReturnValue = p.parseExpression(LOWEST)

	for !p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

// 가장 낮은 우선 순위값을 parseExpression에 넘긴다.
// parseExpressionStatement을 표현식 파싱을 시작하는 최고 수준 메서드로 동작
// 전달 받은 우선순위의 수준을 모르는 상태에서 LOWEST를 사용한다.
// parsePrefixExpression은 PREFIX우선순위를 parseExpression에 넘기는데 전위 표현식을 파싱해야 한다.
func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {

	stmt := &ast.ExpressionStatement{Token: p.curToken}
	stmt.Expression = p.parseExpression(LOWEST)

	// 세미콜론은 선택적
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

// 규격화된 에러메시지를 파서의 errors필드에 추가
func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	msg := fmt.Sprintf("no prefix parse function for %s found", t)
	p.errors = append(p.errors, msg)
}

// parseExpression이 호출될 때, precedence의 값은 parseExpression메서드를 호출하는 현재의 시점에서 갖게 되는 오른쪽으로 묶이는
// 힘을 나타낸다. (RBP)
// 힘이 강할수록 현재의 표현식 오른쪽에 더 많은 토큰/연산자/피연산자를 묶을수 있다.
// 최댓값이라면 ,그동안 파싱한것은 (leftExp에 들어있는 노드) 다음연산자와 여관된 infixParseFn에 전달이 불가
// InfixExpression노드의 왼쪽 자식 노드로 결정불가 : 반복문의 조건이 언젠나 false로 평가
func (p *Parser) parseExpression(precedence int) ast.Expression {

	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}

	leftExp := prefix()

	// 목표는 더 높은 우선순위를 가진 연산자를 포함하는 표현식이, 트리상에서 더 깊게 위치하도록 만드는 데 있다.
	// 다음 연산자/토큰의 LBP가 현재 LBP보다 강한지를 검사
	// 강하면, 그시점까지 파싱한 노드는 다음 연산자에 의해 왼쪽에서 오른쪽으로 빨려 들어간다.
	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}
		p.nextToken()

		leftExp = infix(leftExp)
	}
	return leftExp
}

// Token에 현재 필드를 채우고, Value에 현재 토큰이 갖는 리터럴값을 채워서 반환
// nextToken을 사용햐 curToken, peekToken을 진행안함
// 모든 파싱함수는 같은 규약을 따르게 될것
// 현재 파싱함수와 연관된 토큰 타입이 curToken안 성턀 파싱 함수에 진입,
// 파싱하고자 하는 표현식 타입의 마지막 토큰이 curToken이 되도록 함수를 종료한다.
func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	//defer untrace(trace("parseIntegerLiteral"))
	lit := &ast.IntegerLiteral{Token: p.curToken}

	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as integer", p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}
	lit.Value = value
	return lit
}

// parsePrefixExpression이 호출될시 curToken은 BANG이거나 MINUS이다.
// 전위 표현식을 적절히 파싱하려면 하나이상의 토큰을 소모해야한다.
// 토큰을 진행시키고 parseExpression을 다시 호출
// 전위 연산자의 우선순위를 인수로 넘긴다.
func (p *Parser) parsePrefixExpression() ast.Expression {

	expression := &ast.PrefixExpression{Token: p.curToken, Operator: p.curToken.Literal}

	p.nextToken()

	// parseExpression(PREFIX)은 절대로 1을 -1로 파싱할수도 다른 infixParseFn에 넘길일도 없다.
	expression.Right = p.parseExpression(PREFIX)

	return expression
}

// 우선순위 테이블
// 연산자간 우선순위는 우선 순위 테이블에 저장
// 토큰타입과 토큰타입이 갖는 우선순위가 서로 연관
// 우선순위 값 자체는 이전에 정의한 상수,
// 하나씩 값이 증가하는 정수 값이다.
var precedences = map[token.TokenType]int{
	token.EQ: EQUALS, token.NOT_EQ: EQUALS,
	token.LT: LESSGREATER, token.GT: LESSGREATER,
	token.PLUS: SUM, token.MINUS: SUM,
	token.SLASH: PRODUCT, token.ASTERISK: PRODUCT,
	token.LPAREN:   CALL,
	token.LBRACKET: INDEX,
}

// peekToken이 갖는 토큰 타입과 연관된 우선 순위를 반환
// LBP 왼쪽으로 묶이는 힘
// 결과 값이 바로 다음연산자 혹은 다음 p.peekToken 이 가지는 LBPl
func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

// curToken이 갖는 토큰 타입과 연관된 우선 순위를 반환
func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {

	expression := &ast.InfixExpression{Token: p.curToken, Operator: p.curToken.Literal, Left: left}

	precedence := p.curPrecedence()
	p.nextToken()
	// "+" 연산자를 우결합하게 만들기 위해
	// 예) (a + b) + c) 가 아니라 ((a + (b + c))
	if expression.Operator == "+" {
		expression.Right = p.parseExpression(precedence - 1) // 이 값을 감소시켜서 RBP를 작게 만들어야함
	} else {
		expression.Right = p.parseExpression(precedence)
	}
	return expression
}

func (p *Parser) parseBoolean() ast.Expression {
	return &ast.Boolean{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	return exp
}

func (p *Parser) parseIfExpression() ast.Expression {
	expression := &ast.IfExpression{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	p.nextToken()

	expression.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	expression.Consequence = p.parseBlockStatement()

	// else 문 확인
	if p.peekTokenIs(token.ELSE) {
		p.nextToken()

		if !p.expectPeek(token.LBRACE) {
			return nil
		}
		expression.Alternative = p.parseBlockStatement()
	}
	return expression
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken}

	block.Statements = []ast.Statement{}

	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.ParseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}
	return block
}

func (p *Parser) parseFunctionExpression() ast.Expression {
	lit := &ast.FunctionLiteral{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	lit.Parameters = p.parseFunctionParameters()

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	lit.Body = p.parseBlockStatement()

	return lit
}

func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	var identifiers []*ast.Identifier
	//identifiers := []*ast.Identifier{}

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return identifiers
	}

	p.nextToken()

	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	identifiers = append(identifiers, ident)

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		identifiers = append(identifiers, ident)
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return identifiers

}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Token: p.curToken, Function: function}
	exp.Arguments = p.parseExpressionList(token.RPAREN)
	return exp
}

func (p *Parser) parseCallArguments() []ast.Expression {
	var args []ast.Expression

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return args
	}

	p.nextToken()
	args = append(args, p.parseExpression(LOWEST))

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
	}
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	return args
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}

	array.Elements = p.parseExpressionList(token.RBRACKET)

	return array
}

func (p *Parser) parseExpressionList(end token.TokenType) []ast.Expression {
	var list []ast.Expression

	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(LOWEST))
	}
	if !p.expectPeek(end) {
		return nil
	}
	return list
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{Token: p.curToken, Left: left}

	p.nextToken()
	exp.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	return exp
}

func (p *Parser) parseHashLiteral() ast.Expression {
	hash := &ast.HashLiteral{Token: p.curToken}

	hash.Pairs = make(map[ast.Expression]ast.Expression)

	for !p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		key := p.parseExpression(LOWEST)

		if !p.expectPeek(token.COLON) {
			return nil
		}

		p.nextToken()
		value := p.parseExpression(LOWEST)

		hash.Pairs[key] = value

		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			return nil
		}
	}
	if !p.expectPeek(token.RBRACE) {
		return nil
	}
	return hash
}
