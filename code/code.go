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
	OpArray
	// 해시
	OpHash
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
	OpArray:         {"OpArray", []int{2}},
	OpHash:          {"OpHash", []int{2}},
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
		}
		offset += width
	}
	return operands, offset
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
	}
	return fmt.Sprintf("ERROR: unhandled operandCount for %s\n", def.Name)
}
