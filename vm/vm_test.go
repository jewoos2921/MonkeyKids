package vm

import (
	"MonkeyKids/ast"
	"MonkeyKids/compiler"
	"MonkeyKids/lexer"
	"MonkeyKids/object"
	"MonkeyKids/parser"
	"fmt"
	"testing"
)

func parse(input string) *ast.Program {
	l := lexer.New(input)
	p := parser.New(l)
	return p.ParseProgram()
}

func testIntegerObject(expected int64, actual object.Object) error {
	result, ok := actual.(*object.Integer)

	if !ok {
		return fmt.Errorf("object is not Integer. got=%T (%+v)", actual, actual)
	}

	if result.Value != expected {
		return fmt.Errorf("object has wrong value.  got=%d, want=%d", result.Value, expected)
	}
	return nil
}

type vmTestCase struct {
	input    string
	expected interface{}
}

/*
1. 입력을 렉싱, 파싱한다.
2. AST를 만든다.
3. 만든 AST를 compiler에 전달한다.
4. 컴파일 에러가 있는지 검사한다.
5. *compiler.Bytecode를 New 함수에 넘긴다.
*/
func runVmTests(t *testing.T, tests []vmTestCase) {
	t.Helper()

	for _, tt := range tests {
		program := parse(tt.input)

		comp := compiler.New()
		err := comp.Compile(program)
		if err != nil {
			t.Fatalf("compiler error: %s", err)
		}
		vm := New(comp.Bytecode())
		err = vm.Run()
		if err != nil {
			t.Fatalf("vm error: %s", err)
		}
		stackElem := vm.LastPoppedStackElem()

		testExpectedObject(t, tt.expected, stackElem)
	}
}

func testExpectedObject(t *testing.T, expected interface{}, actual object.Object) {
	t.Helper()

	switch expected := expected.(type) {
	case int:
		err := testIntegerObject(int64(expected), actual)
		if err != nil {
			t.Errorf("testIntegerObject failed: %s", err)
		}
	case bool:
		err := testBooleanObject(bool(expected), actual)
		if err != nil {
			t.Errorf("testBooleanObject failed: %s", err)
		}
	case *object.Null:
		if actual != Null {
			t.Errorf("object is not Null: %T (%+v)", actual, actual)
		}

	}
}

func TestIntegerArithmetic(t *testing.T) {
	tests := []vmTestCase{
		{"1", 1},
		{"2", 2},
		{"1 + 2", 3},
		{"1 - 2", -1},
		{"1 * 2", 2},

		{"50 / 2 * 2 + 10 - 5", 55},
		{"5 + 5 + 5 + 5 - 10", 10},
		{"5 * 2 + 10", 20},
		{"2 * 2 * 2 * 2 * 2", 32},
		{"5 + 2 * 10", 25},
		{"5 * (2 + 10)", 60},

		{"-1", -1},
		{"-10", -10},
		{"-50 + 100 + -50", 0},
		{"(5 + 10 * 2 + 15 / 3) * 2 + -10 ", 50},
	}

	runVmTests(t, tests)
}

// 아무것도 스택에 넣지 못한 상태에서 뭔가를 꺼내려 했기에, 뭔가를 꺼냐려 했기에, 인덱스 범위 초과로 인해 패닉이 발생
// Go 언어에서 패닉은 프로그램을 지속할 수 없으면 사용한다. 따라서 패닉이 발생한 즉시 현재 함수의 실행을 종료한다.
// 그리고 지연함수를 실행하면서 고루틴 스택을 타고 올라간다. 이런 프로세스가 고루틴 스택의 최상단에 도달하면 프로그램이 죽는다.
func TestBooleanExpression(t *testing.T) {
	tests := []vmTestCase{
		{"true", true},
		{"false", false},
		{"1 == 1", true},
		{"1 != 1", false},
		{"1 == 2", false},
		{"1 != 2", true},
		{"true == true", true},
		{"false == false", true},
		{"true == false", false},
		{"true != false", true},
		{"false != true", true},
		{"(1 < 2) == true", true},
		{"(1 < 2) == false", false},
		{"(1 > 2) == true", false},
		{"(1 > 2) == false", true},
		{"!true", false},
		{"!false", true},
		{"!!true", true},
		{"!!false", false},
		{"!5", false},
		{"!!5", true},
	}
	runVmTests(t, tests)
}

func testBooleanObject(expected bool, actual object.Object) error {
	result, ok := actual.(*object.Boolean)
	if !ok {
		return fmt.Errorf("object is not Boolean. got=%T (%+v)", actual, actual)
	}

	if result.Value != expected {
		return fmt.Errorf("object has wrong value.  got=%t, want=%t", result.Value, expected)
	}
	return nil
}

func TestConditionals(t *testing.T) {
	tests := []vmTestCase{
		{"if (true) { 10 }", 10},
		{"if (true) { 10 } else { 20 }", 10},
		{"if (false) { 10 } else { 20 }", 20},
		{"if (1) { 10 }", 10},
		{"if (1 < 2) { 10 }", 10},
		{"if (1 < 2) { 10 } else { 20 }", 10},
		{"if (1 > 2) { 10 } else { 20 }", 20},
		// 둘 다 조건이 참 같은 값이 아니므로 얼터너티브 평가가 강제힌다.
		// 그러나 둘다 얼터너티브기 없으므로 스택에 남은 값이 Null 이 되길 기대한다.
		// 그냥하면 패닉이 발생
		// 조건식 다음에 배출한 OpPop 명령어 때문: 어떤 값도 만들지 않았는데,
		// 가상 머신은 빈 스택에서 뭔가를 꺼내려하니 문제가 발생 - vm.Null을 스택에 넣도록 고쳐야한다.

		// 스택에 vm.Null을 넣으려면 두가지 선결 조건이 존재
		// 1) 명령코드를 정의해 가상 머신에 vm.Null을 스택에 넣으라고 알려줘야 한다.
		// 2) 조건식이 얼터너티브를 갖지 않을 때, 얼터너티브를 삽입하도록 컴파일러를 고쳐야 한다.
		// 이때 삽입된 얼터너티브에는 vm.Null을 스택에 넣는 새로 정의한 명령코드만 포함하게 된다.
		{"if (1 > 2) { 10 }", Null},
		{"if (false) { 10 }", Null},
	}

	runVmTests(t, tests)
}
