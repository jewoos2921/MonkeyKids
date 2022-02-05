package compiler

// 심벌 테이블은 식별자에 정보를 연관시키기 위해 사용
// 렉싱 단계에서 코드 생성 단계 까지 컴파일 단계 전반에 걸쳐서 심벌 테이블을 사용할 수 있다.
// 주어진 식별자와 연관되어 데이터를 저장하거나 가져올 때 사용
// 식별자를 심벌이라고 부르는 이유
// 심벌이 사용된 위치, 스코프, 전에 선언된 적이 있는지 여부, 연관된 값이 갖는 타입, 그 밖에 컴파일이나 인터프리팅 등에 유용한 모든 데이터

// 1) 전역 스코프에 있는 식별자를 고유값과 연관시킨다 - 정의하기
// 2) 주어진 식뱔자에 이미 연관된 고유값을 가져온다 - 환원하기

type SymbolScope string

const (
	LocalScope    SymbolScope = "LOCAL"
	GlobalScope   SymbolScope = "GLOBAL" // 스코프를 구분할 필요가 있다.
	BuiltinScope  SymbolScope = "BUILTIN"
	FreeScope     SymbolScope = "FREE"
	FunctionScope SymbolScope = "FUNCTION"
)

// 심벌을 처리할 때 필요한 정보를 담음
type Symbol struct {
	Name  string
	Scope SymbolScope
	Index int
}

type SymbolTable struct {
	Outer          *SymbolTable
	store          map[string]Symbol
	numDefinitions int
	FreeSymbols    []Symbol
}

func NewSymbolTable() *SymbolTable {
	s := make(map[string]Symbol)
	var free []Symbol
	return &SymbolTable{store: s, FreeSymbols: free}
}

func (s *SymbolTable) Define(name string) Symbol {
	symbol := Symbol{
		Name:  name,
		Index: s.numDefinitions}
	if s.Outer == nil {
		symbol.Scope = GlobalScope
	} else {
		symbol.Scope = LocalScope
	}

	s.store[name] = symbol
	s.numDefinitions++
	return symbol
}

// 호출된 SymbolTable에서 심벌을 찾는 일
// 재귀적으로 outer 심벌 테이블을 계속 타고 올라가도록 만들어야 하며,
// 계속 타고 올라가다가 심벌을 찾으면 반환하고, 그렇지 않으면 호출한 곳에다 해당 심벌이 정의되지 않았다는 것을 알려줘야 한다.
// ? 만약 어떤 심벌을 지역 스코프에 정의하고 더 깉은 스코프에서 환원하면 그 심벌은 지역 스코프를 갖게 될 텐데, 바깥쪽 스코프에
// 정의되어 있는데 지역 스코프라고 정의해도 되는가??
func (s *SymbolTable) Resolve(name string) (Symbol, bool) {
	obj, ok := s.store[name]
	if !ok && s.Outer != nil {
		obj, ok = s.Outer.Resolve(name)
		if !ok {

			return obj, ok
		}

		if obj.Scope == GlobalScope || obj.Scope == BuiltinScope {
			return obj, ok
		}
		free := s.defineFree(obj)

		return free, true
	}
	return obj, ok
}

func (s *SymbolTable) DefineBuiltin(index int, name string) Symbol {
	symbol := Symbol{Name: name, Scope: BuiltinScope, Index: index}
	s.store[name] = symbol
	return symbol
}

func NewEnclosedSymbolTable(outer *SymbolTable) *SymbolTable {
	s := NewSymbolTable()
	s.Outer = outer
	return s
}
func (s *SymbolTable) defineFree(original Symbol) Symbol {
	s.FreeSymbols = append(s.FreeSymbols, original)

	symbol := Symbol{Name: original.Name, Index: len(s.FreeSymbols) - 1}
	symbol.Scope = FreeScope

	s.store[original.Name] = symbol
	return symbol
}
func (s *SymbolTable) DefineFunctionName(name string) Symbol {
	symbol := Symbol{Name: name, Index: 0, Scope: FunctionScope}
	s.store[name] = symbol
	return symbol
}
