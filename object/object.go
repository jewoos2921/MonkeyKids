package object

import (
	"MonkeyKids/ast"
	"bytes"
	"fmt"
	"hash/fnv"
	"strings"
)

type ObjectType string

const (
	INTEGER_OBJ      = "INTEGER"
	BOOLEAN_OBJ      = "BOOLEAN"
	NULL_OBJ         = "NULL"
	RETURN_VALUE_OBJ = "RETURN"
	ERROR_OBJ        = "ERROR"
	FUNCTION_OBJ     = "FUNCTION"
	STRING_OBJ       = "STRING"
	BUILTIN_OBJ      = "BUILTIN"
	ARRAY_OBJ        = "ARRAY"
	HASH_OBJ         = "HASH"
)

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
func (e *Error) Inspect() string  { return "ERROR: " + e.Message }

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

type HashTable interface {
	HashKey() HashKey
}
