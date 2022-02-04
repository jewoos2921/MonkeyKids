package object

import (
	"MonkeyKids/ast"
	"MonkeyKids/code"
	"bytes"
	"fmt"
	"hash/fnv"
	"strings"
)

// 트리 순회 평가기
// 호스트언어인 Go로 Monkey의 값을 표현하는 방법
// 객체 표현하기
// 호스트 언어로 표현한 값, 게스트 언어로 사용할 사용자들에게 어떻게 보여주어야 하는가?

const (
	INTEGER_OBJ      = "INTEGER"
	BOOLEAN_OBJ      = "BOOLEAN"
	NULL_OBJ         = "NULL"
	RETURN_VALUE_OBJ = "RETURN"
	ERROR_OBJ        = "ERROR"
	FUNCTION_OBJ     = "FUNCTION"
	STRING_OBJ       = "STRING" // 문자열로 표현하는것은 쉽다. Go언어가 가진 자료형을 재사용, 객체만 정의하면 된다.
	BUILTIN_OBJ      = "BUILTIN"
	ARRAY_OBJ        = "ARRAY"
	HASH_OBJ         = "HASH"
	// 함수 표현하기
	// 만든 명령어를 어디에 저장하며, 어떻게 가상 머신에 넘기는지,
	// 함수 리터럴
	COMPILED_FUNCTION_OBJ = "COMPILED_FUNCTION_OBJ"
)

// 모든값을 Object 인터페이스를 만족하는 구조체로 감쌀 것이다.

type ObjectType string

// Object가 인터페이스인 경우는 모든값은 내부 표현을 다르게 할 필요가 있기 때문
// 불과 정수를 동일한 구조체 필드로 끼워 맞추기 보다 서로 다른 구조체를 2개로 정의하는게 훨씬 쉽다.
type Object interface {
	Type() ObjectType
	Inspect() string
}

// 정수
type Integer struct {
	Value int64
}

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }

func (i *Integer) Inspect() string { return fmt.Sprintf("%d", i.Value) }

type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }

type Null struct{}

func (n *Null) Type() ObjectType { return NULL_OBJ }
func (n *Null) Inspect() string  { return "null" }

// 평가하는 도중에 return 문을 만나면, 원래 반환해야 할 값을 객체 하나로 감싸서 처리한다.
// 그래야만 평가기가 이 객체를 추적 가능: 평가 도중에 평가를 계속 해야할지 말지 결정할 때,
// 이 객체가 필요하기 때문에
// ReturnValue는 다른 객체를 감싸는 래퍼일뿐 다른 내용은 없다.
type ReturnValue struct {
	Value Object
}

func (rv *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (rv *ReturnValue) Inspect() string  { return rv.Value.Inspect() }

// 예외 처리
type Error struct {
	Message string
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }

// 상용 인터츠리터라면 스택 트레이스를 담
// 문제가 발생한 지점의 행과열 번호를 같이 넣어서 단순한 메시지보다 더 많은 정보를 줄 수도 있다.
// 렉서가 토큰에 행과 열번호를 달아놨으면 어렵지 않다.
func (e *Error) Inspect() string { return "ERROR: " + e.Message }

// Env가 있는 이유: 함수는 자기환경에서 움직이기 때문에
type Function struct {
	Parameters []*ast.Identifier
	Body       *ast.BlockStatement
	Env        *Environment
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string {
	var out bytes.Buffer

	var params []string
	for _, p := range f.Parameters {
		params = append(params, p.String())
	}

	out.WriteString("fn")
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") {\n")
	out.WriteString(f.Body.String())
	out.WriteString("\n}")

	return out.String()
}

type String struct {
	Value string
}

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }

type BuiltinFunction func(args ...Object) Object

// 내장 함수
type Builtin struct {
	Fn BuiltinFunction
}

func (b *Builtin) Type() ObjectType { return BUILTIN_OBJ }
func (b *Builtin) Inspect() string  { return "builtin function" }

// 배열 리터럴
type Array struct {
	Elements []Object
}

func (ao *Array) Type() ObjectType { return ARRAY_OBJ }
func (ao *Array) Inspect() string {
	var out bytes.Buffer

	var elements []string
	for _, e := range ao.Elements {
		elements = append(elements, e.Inspect())
	}

	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")

	return out.String()
}

// hashkey
type HashKey struct {
	Type  ObjectType // 타입별로 HashKey가 갖는 범위를 효과적으로 제한 (string이기 때문에)
	Value uint64     // 정수타입을 갖는 실제 해시 키 값을 담는다.
}

// HashKey는 그저 문자열 하나와 정수 하나 만으로 구성되기 때문에 == 연산자로 다른 HashKey과 쉽게 비교 가능
// GO 언어에서는 두 구조체간 타입이 같으면 동등성을 비교가능
// 비어 있지 않은 필드가 모두 같다면 두 구조체는 같다.
// 작은 결점: .Value가 다른 2개의 문자열이, 같은 해시 값을 가질 수 있다는 것
// "hash/fnv" 패키지가 서로 다른 두개의 문자열로 생성한 두 해시값이 같을시 발생
// 해시 충돌이라고 표현
//======================================================================================================================
// 체이닝(separate chaining)
// 버킷을 일종의 리스트로 관리해 해시 충돌을 해결하는 전략이다.
// 해시값이 같다면 같은 버킷에 들어가게 되고, 버킷은 리스트로 관리되므로 기존 리스트에 해시값이 같은 엔트리를 연결해서 처리한다.
//======================================================================================================================
// 오픈 어드레싱(open addressing)
// 버킷 하나에 엔트리 하나를 넣는다. 엔트리를 버킷에 넗으려 할때, 이미 버킷에 채워져 있다면 빈 버킷을 찾아서 넣는다.
// 빈 버킷을 찾는 전략에 따라 구현체가 달라진다.

func (b *Boolean) HashKey() HashKey {
	var value uint64

	if b.Value {
		value = 1
	} else {
		value = 0
	}
	return HashKey{Type: b.Type(), Value: value}
}
func (i *Integer) HashKey() HashKey {
	return HashKey{Type: i.Type(), Value: uint64(i.Value)}
}

func (s *String) HashKey() HashKey {
	h := fnv.New64a()
	h.Write([]byte(s.Value))

	return HashKey{Type: s.Type(), Value: h.Sum64()}
}

type HashPair struct {
	Key   Object
	Value Object
}

type Hash struct {
	Pairs map[HashKey]HashPair
}

func (h *Hash) Type() ObjectType { return HASH_OBJ }
func (h *Hash) Inspect() string {
	var out bytes.Buffer

	var pairs []string

	for _, pair := range h.Pairs {
		pairs = append(pairs, fmt.Sprintf("%s: %s", pair.Key.Inspect(),
			pair.Value.Inspect()))
	}
	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")

	return out.String()
}

type Hashable interface {
	HashKey() HashKey
}

type CompiledFunction struct {
	Instructions code.Instructions
}

func (cf *CompiledFunction) Type() ObjectType { return COMPILED_FUNCTION_OBJ }
func (cf *CompiledFunction) Inspect() string {
	return fmt.Sprintf("compiledFunction[%p]", cf)
}
