package ast

// 추상 구문 트리
import (
	"MonkeyKids/token"
	"bytes"
	"strings"
)

// 우리가 생성할 AST는 노드로만 구성되고, 각각의 노드는 서로 연결될 것이다.
type Node interface {
	TokenLiteral() string // 토큰에 대응하는 리터럴값을 반환, 디버깅과 테스트 용도로만 사용
	String() string
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

// Program노드 : 루트 노드
type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	} else {
		return ""
	}
}

// LetStatement는 필요한 필드를 모두 가진다.
type LetStatement struct {
	Token token.Token // 토큰
	Name  *Identifier // 식별자
	Value Expression  // 값을 내는 표현식 필드
}

func (ls *LetStatement) statementNode()       {}
func (ls *LetStatement) TokenLiteral() string { return ls.Token.Literal }

// 노드 타입의 수를 가능한 작게 만들기 위해 사용
// 변수 바인딩 이름을 나타내며, 선언한 이름으로 나중에 재상ㅇ
// 표현식의 일부 또는 표현식 전체를 나타내기 위해 사용
type Identifier struct {
	Token token.Token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }

// return <expression>;
type ReturnStatement struct {
	Token       token.Token // "return" 토큰
	ReturnValue Expression
}

func (rs *ReturnStatement) statementNode() {}

func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }

// 왼쪽에서 오른쪽으로 흝어가며 토큰이 조건에 맞으면 처리하고 맞지 않으면 에러로 처리
// 연산자 우선순위
// 표현식 안에서 같은 터압의 토큰들이 여러 위치에서 나타남
// 하양식 연산자 우선순위 (프랫 파싱)
// 문맥 무관 문법과 배커스 나우어 형식에 기반한 파서를 대체할 목적으로 개발
// 문법규칙과 함수를 연관시키는 대신에 프랫은 토큰 타입과 파싱 함수를 연관
// 토큰을 함수와 연관 시킬때, 중위인지 전위 인지에 따라 서로 다른 파싱 함수로 연관
// 표현식은 감싸는 역할만 한다.
// 전위 연산자는 피연산자 앞에 붙는 연산자
// 중위 연산자는 피연산자 뒤에 붙는 연산자
// 연산 순서는 연산자 우선순위 보다 서로 다른 연산자에서 어떤 것이 우선하는 지를 명확히 전달
// 연산자 우선순위를 의존성처럼 생각
// 피연산자가 다음에 나올 연산자에 얼마나 의존하는지를 말한다.
// 프릿 파서의 핵심은 파싱 함수를 토큰 타입과 연관
// 파서가 토큰 타입을 만날 때마다 파싱함수가 적절한 표현식을 파싱하고, 그 표현식을 나타내는 AST노드를 하나 반환
// 각각의 토큰 타입은 토큰이 전위 연산자인지 중위 연산자인지에 따라 최대 2개의 파싱 함수와 연관

type ExpressionStatement struct {
	Token      token.Token
	Expression Expression
}

func (es *ExpressionStatement) statementNode() {}

func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }

// 버퍼을 하나만들고 각 명령문의 String메서드를 호출하여 반환값을 버퍼에 쓴다
// 그러고 나서 버퍼를 문자열로 반환
// 작업 대부분을 *ast.Program.Statements에 위임
func (p *Program) String() string {
	var out bytes.Buffer

	for _, s := range p.Statements {
		out.WriteString(s.String())
	}
	return out.String()
}

func (ls *LetStatement) String() string {
	var out bytes.Buffer
	out.WriteString(ls.TokenLiteral() + " ")
	out.WriteString(ls.Name.String())
	out.WriteString(" = ")
	if ls.Value != nil {
		out.WriteString(ls.Value.String())
	}

	out.WriteString(";")

	return out.String()
}
func (i *Identifier) String() string {
	return i.Value
}
func (rs *ReturnStatement) String() string {
	var out bytes.Buffer

	out.WriteString(rs.TokenLiteral() + " ")
	if rs.ReturnValue != nil {
		out.WriteString(rs.ReturnValue.String())
	}
	out.WriteString(";")

	return out.String()
}
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

type IntegerLiteral struct {
	Token token.Token
	Value int64 // 정수 리터럴이 표현하는 문자의 실젯값을 담을 것
}

func (il *IntegerLiteral) expressionNode() {}

func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

type PrefixExpression struct {
	Token    token.Token // 전위 연산자 토큰, 예) !
	Operator string      // '='나 '!'의 문자열을 담을 필드
	Right    Expression  // 연산자의 오른쪽에 나올 표현식
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(pe.Operator)
	out.WriteString(pe.Right.String())
	out.WriteString(")")

	return out.String()
}

// 중위 표현식
// <expression> <infix operator> <expression>
type InfixExpression struct {
	Token    token.Token // 연산자 토큰, 예) +
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *InfixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString(" " + ie.Operator + " ")
	out.WriteString(ie.Right.String())
	out.WriteString(")")

	return out.String()
}

// 불 리터럴
type Boolean struct {
	Token token.Token
	Value bool
}

func (b *Boolean) expressionNode() {}

func (b *Boolean) TokenLiteral() string { return b.Token.Literal }
func (b *Boolean) String() string       { return b.Token.Literal }

// if (<condition>) <consequence> else <alternative>
// 지금까지의 성공레시피
// AST 노드를 정의한다.
// 테스트를 작성한다.
// 파싱코드를 작성해 테스트를 통과한다.
type IfExpression struct {
	Token       token.Token // if token
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement
}

func (ie *IfExpression) expressionNode() {}

func (ie *IfExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IfExpression) String() string {
	var out bytes.Buffer

	out.WriteString("if")
	out.WriteString(ie.Condition.String())
	out.WriteString(" ")
	out.WriteString(ie.Consequence.String())

	if ie.Alternative != nil {
		out.WriteString("else")
		out.WriteString(ie.Alternative.String())
	}
	return out.String()
}

type BlockStatement struct {
	Token      token.Token // { 토큰
	Statements []Statement
}

func (bs *BlockStatement) expressionNode() {}

func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStatement) String() string {
	var out bytes.Buffer
	for _, s := range bs.Statements {
		out.WriteString(s.String())
	}

	return out.String()
}

// 함수 리터럴
// fn <parameter> <blockStatement>
// (<parameter one>, <parameter two>, <parameter three>, ...)
type FunctionLiteral struct {
	Token      token.Token     // 'fn' 토큰
	Parameters []*Identifier   // 파라미터 리스트
	Body       *BlockStatement // 함수의 몸체
}

func (fl *FunctionLiteral) expressionNode() {}

func (fl *FunctionLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FunctionLiteral) String() string {
	var out bytes.Buffer

	var params []string
	//params:= []string{}
	for _, p := range fl.Parameters {
		params = append(params, p.String())
	}

	out.WriteString(fl.TokenLiteral())
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(")")
	out.WriteString(fl.Body.String())

	return out.String()
}

// 호출 표현식
// <expression>(<comma separated expressions>)
type CallExpression struct {
	Token     token.Token // 여는 괄호 토큰 '('
	Function  Expression  // 식별자이거나 함수 리터럴
	Arguments []Expression
}

func (ce *CallExpression) expressionNode() {}

func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpression) String() string {
	var out bytes.Buffer

	var args []string
	//params:= []string{}
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}

	out.WriteString(ce.Function.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")

	return out.String()
}

// 문자열 파싱
// 명령문이아니라 표현식이다.
// < sequence of characters >
type StringLiteral struct {
	Token token.Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string       { return sl.Token.Literal }

// 배열 리터럴
type ArrayLiteral struct {
	Token    token.Token // '[' 토큰
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }
func (al *ArrayLiteral) String() string {
	var out bytes.Buffer

	var elements []string
	for _, el := range al.Elements {
		elements = append(elements, el.String())
	}

	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")

	return out.String()
}

// 인덱스 연산자 표현식
// <expression>[<expression>]
type IndexExpression struct {
	Token token.Token // '[' 토큰
	Left  Expression  // 접근의 대상인 객체
	Index Expression
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString("[")
	out.WriteString(ie.Index.String())
	out.WriteString("])")

	return out.String()
}

// 해시
type HashLiteral struct {
	Token token.Token // "{" 토큰
	Pairs map[Expression]Expression
}

func (hl *HashLiteral) expressionNode()      {}
func (hl *HashLiteral) TokenLiteral() string { return hl.Token.Literal }
func (hl *HashLiteral) String() string {
	var out bytes.Buffer

	var pairs []string
	for key, value := range hl.Pairs {
		pairs = append(pairs, key.String()+":"+value.String())
	}
	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")
	return out.String()
}
