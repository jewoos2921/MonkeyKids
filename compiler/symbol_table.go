package compiler

// 심벌 테이블은 식별자에 정보를 연관시키기 위해 사용
// 렉싱 단계에서 코드 생성 단계 까지 컴파일 단계 전반에 걸쳐서 심벌 테이블을 사용할 수 있다.
// 주어진 식별자와 연관되어 데이터를 저장하거나 가져올 때 사용
// 식별자를 심벌이라고 부르는 이유
// 심벌이 사용된 위치, 스코프, 전에 선언된 적이 있는지 여부, 연관된 값이 갖는 타입, 그 밖에 컴파일이나 인터프리팅 등에 유용한 모든 데이터

// 1) 전역 스코프에 있는 식별자를 고유값과 연관시킨다. - 정의하기
// 2) 주어진 식뱔자에 이미 연관된 고유값을 가져온다 - 환원하기

type SymbolScope string

const (
	GlobalScope SymbolScope = "GLOBAL"
)

type Symbol struct {
	Name  string
	Scope SymbolScope
	Index int
}
type SymbolTable struct {
	store          map[string]Symbol
	numDefinitions int
}

func NewSymbolTable() *SymbolTable {
	s := make(map[string]Symbol)
	return &SymbolTable{store: s}
}

func (s *SymbolTable) Define(name string) Symbol {
	symbol := Symbol{
		Name:  name,
		Scope: GlobalScope,
		Index: s.numDefinitions,
	}
	s.store[name] = symbol
	s.numDefinitions++
	return symbol
}

func (s *SymbolTable) Resolve(name string) (Symbol, bool) {
	obj, ok := s.store[name]
	return obj, ok
}
