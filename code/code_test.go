package code

import (
	"testing"
)

func TestMake(t *testing.T) {
	tests := []struct {
		op       Opcode
		operands []int
		expected []byte
	}{
		{OpConstant, []int{65534}, []byte{byte(OpConstant), 255, 254}},
		{OpAdd, []int{}, []byte{byte(OpAdd)}},
		{OpGetLocal, []int{255}, []byte{byte(OpGetLocal), 255}},
		{OpClosure, []int{65534, 255}, []byte{byte(OpClosure), 255, 254, 255}},
	}

	for _, tt := range tests {
		instruction := Make(tt.op, tt.operands...)

		if len(instruction) != len(tt.expected) {
			t.Errorf("instruction has wrong length. want=%d, got=%d",
				len(tt.expected), len(instruction))
		}
		for i, b := range tt.expected {
			if instruction[i] != tt.expected[i] {
				t.Errorf("wrong byte at pos=%d. want=%d, got=%d", i, b, instruction[i])
			}
		}
	}
}

func TestInstructionsString(t *testing.T) {
	instructions := []Instructions{
		Make(OpAdd), // 카운터 값
		Make(OpGetLocal, 1),
		Make(OpConstant, 2),
		Make(OpConstant, 65535),
		Make(OpClosure, 65535, 255),
	}
	expected := `0000 OpAdd
0001 OpGetLocal 1 
0003 OpConstant 2 
0006 OpConstant 65535
0009 OpClosure 65535 255`
	concatted := Instructions{}
	for _, ins := range instructions {
		concatted = append(concatted, ins...)
	}

	if concatted.String() != expected {
		t.Errorf("instructions wrongly formatted.\nwant=%q\ngot=%q",
			expected, concatted.String())
	}
}

func TestReadOperands(t *testing.T) {
	tests := []struct {
		op        Opcode
		operands  []int
		bytesRead int
	}{
		{OpConstant, []int{65535}, 2},
		{OpGetLocal, []int{255}, 1},
		{OpClosure, []int{65535, 255}, 3},
	}
	for _, tt := range tests {
		// 부호화된 명령어를 만들고 명령어의 정의와 명령어에서 피연산자를 포함하는부분 슬라이스를 함께 ReadOperands에 인수로 전달
		instruction := Make(tt.op, tt.operands...)

		def, err := Lookup(byte(tt.op))
		if err != nil {
			t.Fatalf("definition not found: %q\n", err)
		}

		// 복호화된 피연산자를 반환, 복호화된 피연산자를 읽을 때
		// 바이트를 얼마만큼의 크기로 읽어야 하는지 알려준다
		operandsRead, n := ReadOperands(def, instruction[1:])
		if n != tt.bytesRead {
			t.Fatalf("n wrong. want=%q, got=%d", tt.bytesRead, n)
		}
		for i, want := range tt.operands {
			if operandsRead[i] != want {
				t.Errorf("operand wroong. want=%d, got=%d", want, operandsRead[i])
			}
		}
	}
}
