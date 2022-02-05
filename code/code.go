package code

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// 바이트 코드는 명령어로 되어있음
// 명령어는 바이트열
// 명령어 하나는 명령코드 1개, 피연산자 0개 이상을 가짐
type Instructions []byte

// Bytecode를 정의하지 않는 이유는 임포트 사이클 문제가 발생하기 때문에
type Opcode byte

// 정수리터럴을 컴파일 중에 만나면, 평가뒤에 결과객체 *object.Integer를 추적
// 바이트코드 명령어에서는 앞서 부여한 값으로 *object.Integer를 참조
// 컴파일 후에,명령어를 실행할 수 있도록, 가상머신에 명령어를 전달하면, 모든 상수 표현식을 모아 저장하고 있는 자료구조인
// 상수풀을 같이 전달한다.
// 상수풀 안에 있는 숫자 값은 앞서 상수에 부여된 값이며,
// 상수풀에서는 인덱스를 사용해 저장한 상수를 가져올 수 있다.

const (
	OpConstant Opcode = iota
	OpAdd
	OpPop
	// 중위 표현식
	OpSub
	OpMul
	OpDiv
	// 불리터럴
	// 불리터럴을 OpConstant 명령어로 컴파일되어야 한다. true와 false를 상수로 취급해도 되지만 이럴경우 자원을 낭비하는 꼴이다.
	// 바이트코드에서 사용할 자원만 낭비하는 게 아니라, 컴파일러와 가상 머신에서 사용할 자원도 낭비된다.
	OpTrue
	OpFalse
	// 비교 연산자
	OpEqual
	OpNotEqual
	OpGreaterThan
	// 전위 표현식
	// 1. 필요한 명령 코드를 정의한다.
	// 2. 컴파일러에서 해당 명령 코드를 배출한다.
	// 3. 가상머신에서 처리한다.
	OpMinus // -연산자
	OpBang  // !연산자

	// 가상머신이 조건에 따라 바이트코드 명령어를 다르세 실행하도록 만들려면 어떻게 해야 할까?
	// AST를 순회하면서 실행하는게 아니라 AST를 바이트코드로 바꾸고 평탄화해야 한다.
	// 평탄화하는 이유는 바이트코드가 명령어를 일렬로 늘어 놓게끔 만들어져 있고
	// 자식 노드가 없어서 타고 내려갈 대상 자체가 없기 때문에

	// 조건식 컴파일하기
	// 점프 목적지를 가리키는 화살표는 어떻게 표현해야 할까?
	// 가상 머신에게 어디로 점프하라고 말해줘야 할까?
	// 화살표: 잠재적 명령어 포인터값
	// 오프셋: 가상머신이 점프해서 도착하게 될 명령어 인덱스값
	OpJumpNotTruthy // 조건에 따라 점프하는 명령어
	OpJump          // 점프 명령어, 특정위치로 점프하라
	// Null
	OpNull
	// 바인딩에서 중요한 작업은 이미 식별자에 바인딩된 값을 올바르게 환원하는 일
	// 스택 가장 위에 있는 값을 지금 처리하고 있는 식별자에 바인딩하면 된다.
	// 바인딩 컴파일하기
	// 가상 머신이 OpSetGlobal명령어를 실행 하면, OpSetGlobal이 달린 피연산자를 읽는다.
	// 따라서 스택 가장 위에 있는 요소를 뽑아서 전역 스토어에 저장한다.
	// 이때 피연산자에 담긴 인덱스값에 해당하는 위치에 저장
	// OpGetGlobal을 실행시, OpGetGlobal에 달린 피연산자를 사용해서 전역스토어에서 값을 가져오고,
	// 가져온 값을 스택에 넣는다.
	OpGetGlobal
	OpSetGlobal
	// 문자열
	// 배열은 복합 데이터타입
	// 컴파일타임에 배열을 만들어 상수 풀에 넣은 후 가상머신에 전달하는게 아니라,
	// 가상 머신이 직접 배열을 만들도록 어떤 정보를 주어야 한다.
	OpArray
	// 해시
	OpHash
	// 인덱스가 달릴 객체
	// 인덱스로서 사용할 객체
	OpIndex
	// 함수 호출에 사용할 명령코드
	// 먼저 호출하고 싶은 함수를 스택에서 가져온다.
	// 그리고 OpCall를 배출한다.
	// 그러면 가상머신은 OpCall을 보고 스택 가장 위에 함수를 가져와서 실행
	// 호출할 함수를 스택에 올려두는 단계
	// OpCall 명령어를 배출하는 단계
	OpCall
	// 가상머신이 함수에서 원래 위치로 반환
	// 1. 어떤 결과를 암묵적이든 명시적이든 반환하는 형태
	// 2. 함수 호출 결과로 아무것도 남기지 않는 형태
	OpReturnValue // 가상 머신에게 스택 가장 위에 있는 값을 반환하라고 말함
	OpReturn      // 현재 함수에서 빠져나오라고 말함, 반환값이 없음

	// 지역 바인딩
	// 함수 스코프 안에서만 보이고 접근가능해야 한다.
	// 1) 명령코드를 새로 정의해서 가상 머신이 지역 바인딩을 만들어 내고 가져올 수 있게 만들어야 한다.
	// 2) 컴파일러를 확장해 새로 정의한 명령코드를 올바르게 배출할 수 있어야 한다.
	// 3) 가장 머신에 새로 추가한 명령어를 구현하고, 지역바인딩을 구현하면 된다.
	// 전역 바인딩에 영향을 미쳐서는 안됨
	OpGetLocal
	OpSetLocal
	// 내장 함수용 스코프
	OpGetBuiltin
	OpClosure
	// 자유 변수 컴파일과 환원
	OpGetFree
	// 컴파일러에서 자기 참조를 하는 바인딩을 탐지해서 자유 변수 심벌로 표시하고, OpGetFree 을 배출해서
	// 표시해둔 자유변수를 스택에 올리는 게 아니라, 새로운 명령코드를 하나 배출하도록 만드는 것
	OpCurrentClosure
)

type Definition struct {
	Name          string // 명령코드
	OperandWidths []int  // 8 비트 (전역 바인딩 고유 숫자 값)
}

var definitions = map[Opcode]*Definition{
	OpConstant:      {"OpConstant", []int{2}},
	OpAdd:           {"OpAdd", []int{}},
	OpPop:           {"OpPop", []int{}},
	OpSub:           {"OpSub", []int{}},
	OpMul:           {"OpMul", []int{}},
	OpDiv:           {"OpDiv", []int{}},
	OpTrue:          {"OpTrue", []int{}},
	OpFalse:         {"OpFalse", []int{}},
	OpEqual:         {"OpEqual", []int{}},
	OpNotEqual:      {"OpNotEqual", []int{}},
	OpGreaterThan:   {"OpGreaterThan", []int{}},
	OpMinus:         {"OpMinus", []int{}},
	OpBang:          {"OpBang", []int{}},
	OpJumpNotTruthy: {"OpJumpNotTruthy", []int{2}},
	OpJump:          {"OpJump", []int{2}},
	OpNull:          {"OpNull", []int{}},
	OpGetGlobal:     {"OpGetGlobal", []int{2}},
	OpSetGlobal:     {"OpSetGlobal", []int{2}},
	// 배열의 크기는 65535로 제한
	OpArray: {"OpArray", []int{2}},
	OpHash:  {"OpHash", []int{2}},
	OpIndex: {"OpIndex", []int{}},
	// 인수가 있는 함수 호출 컴파일 하기
	// 함수 호출 인수는 지역 바인딩을 만드는 특수한 케이스
	// 지역 바인딩은 사용자가 let 문을 사용하며 명시적으로 생성, 그결과를 OpSetLocal 배출
	// 인수는 암묵적으로 이름에 바인딩
	// 호출할 함수를 스택에 넣는다.
	//호출 인수를 스택에 넣는다.
	// OpCall 명령어를 배출
	OpCall:           {"OpCall", []int{1}},
	OpReturnValue:    {"OpReturnValue", []int{}},
	OpReturn:         {"OpReturn", []int{}},
	OpGetLocal:       {"OpGetLocal", []int{1}},
	OpSetLocal:       {"OpSetLocal", []int{1}},
	OpGetBuiltin:     {"OpGetBuiltin", []int{1}},
	OpClosure:        {"OpClosure", []int{2, 1}},
	OpGetFree:        {"OpGetFree", []int{1}},
	OpCurrentClosure: {"OpCurrentClosure", []int{}},
}

func Lookup(op byte) (*Definition, error) {
	def, ok := definitions[Opcode(op)]
	if !ok {
		return nil, fmt.Errorf("opcode %d undefined", op)
	}
	return def, nil
}

func Make(op Opcode, operands ...int) []byte {
	def, ok := definitions[op]

	if !ok {
		return []byte{}
	}

	instructionLen := 1
	for _, w := range def.OperandWidths {
		instructionLen += w
	}

	instruction := make([]byte, instructionLen)
	instruction[0] = byte(op)

	offset := 1
	for i, o := range operands {
		width := def.OperandWidths[i]
		switch width {
		case 2:
			binary.BigEndian.PutUint16(instruction[offset:], uint16(o))

		case 1:
			// 1바이트를 처리
			instruction[offset] = byte(o)
		}
		offset += width
	}
	return instruction
}

func ReadOperands(def *Definition, ins Instructions) ([]int, int) {
	operands := make([]int, len(def.OperandWidths))

	offset := 0

	for i, width := range def.OperandWidths {
		switch width {
		case 2:
			operands[i] = int(ReadUint16(ins[offset:]))
		case 1:
			operands[i] = int(ReadUint8(ins[offset:]))
		}
		offset += width
	}
	return operands, offset
}

func ReadUint8(ins Instructions) uint8 {
	return uint8(ins[0])
}

func ReadUint16(ins Instructions) uint16 {
	return binary.BigEndian.Uint16(ins)
}

func (ins Instructions) String() string {
	var out bytes.Buffer

	i := 0
	for i < len(ins) {
		def, err := Lookup(ins[i])
		if err != nil {
			fmt.Fprintf(&out, "ERROR: %s\n", err)
			continue
		}
		operands, read := ReadOperands(def, ins[i+1:])

		fmt.Fprintf(&out, "%04d %s\n", i, ins.fmtInstruction(def, operands))

		i += 1 + read
	}
	return out.String()
}

func (ins Instructions) fmtInstruction(def *Definition, operands []int) string {
	operandCount := len(def.OperandWidths)

	if len(operands) != operandCount {
		return fmt.Sprintf("ERROR: operand len %d does not match defined %d\n",
			len(operands), operandCount)
	}
	switch operandCount {
	case 0:
		return def.Name
	case 1:
		return fmt.Sprintf("%s %d", def.Name, operands[0])
	case 2:
		return fmt.Sprintf("%s %d %d", def.Name, operands[0], operands[1])

	}
	return fmt.Sprintf("ERROR: unhandled operandCount for %s\n", def.Name)
}
