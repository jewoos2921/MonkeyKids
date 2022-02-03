package object

// env *object.Environment
// 환경이란 인터프리터가 값을 추적할 때 사용하는 객체로, 값을 이름과 연관
func NewEnvironment() *Environment {
	s := make(map[string]Object)
	return &Environment{store: s, outer: nil}
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

type Environment struct {
	store map[string]Object
	outer *Environment
}

/* 안쪽 스코프가 있고 바깥쪽 스코프가 있다. 만약 안쪽 스코프에서 값을 찾지 못했다면, 바깥쪽 스코프를 찾아본다.
바깥쪽 스코프는 안쪽 스코프를 감싼다. 안쪽 스코프는 바깥쪽 스코프를 확장한다. */
func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

func (e *Environment) Set(name string, val Object) Object {
	e.store[name] = val
	return val
}

// 카운터 함수를 호출할 때 마다 많은 객체가 할당된다.
// 객체는 메모리에 저장
// 가비지 컬렉터는 매우 유용
// 객체 할당 추적
// 객체 참조 추적
// 미래에 할당할 객체를 위한 충분한 메모리 확보
// 필요없는 객체에서 메모리 수거
