package vm

import (
	"MonkeyKids/code"
	"MonkeyKids/object"
)

// 함수른 중첩해서 호출할 수 있으며, 호출과 관계된 정보는 후입선출방식으로 접근한다.
// 프레임: 호출 프레임, 스택 프레임의 줄임말
// 자료구조이며 함수 실행과 관련된 정보를 담음
// 스택 안에 이미 지정된 영역에 존재
// 반환주소, 햔재 함수 호출에 사용된 인수와 지역 변수가 저장
type Frame struct {
	fn          *object.CompiledFunction // 프레임이 참조할 컴파일된 함수를 가리키는 포인터
	ip          int                      // 현재 프레임에서 현재 함수가 사용할 명령어 포인터
	basePointer int                      // 현재의 호출프레임 스택 최하단을 가리키는 포인터
}

// basePointer: 재시작 버튼같이 사용하기 위해서  ,지역 바인딩을 참조하는데 사용하기 위해서

func NewFrame(fn *object.CompiledFunction, basePointer int) *Frame {
	return &Frame{
		fn:          fn,
		ip:          -1,
		basePointer: basePointer,
	}
}

func (f *Frame) Instructions() code.Instructions {
	return f.fn.Instructions
}
