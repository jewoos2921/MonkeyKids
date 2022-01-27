package code

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type Instructions []byte

type Opcode byte

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
	// 조건식 컴파일하기
	OpJumpNotTruthy // 조건에 따라 점프하는 명령어
	OpJump          // 점프 명령어
)

type Definition struct {
	Name          string
	OperandWidths []int
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
