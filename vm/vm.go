package vm

// 데이터와 프로그램 모두 메모리에 저장한다.
// 콜스택: 스택자료구조로 프로그램의 활성 서브루틴 정보를 저장
// CPU는 현재의 함수를 실행 한뒤 반환주소로 되돌아간다.
// 바이트 코드는 도메인 특화 언어: 사용자 정의 가상머신에 맞게 설계된 맞춤형 기계어
// 가상 머신을 만드는 이유: 사용자 정의 바이트코드 형식을 사용하면 도메인에 특화되도록 만들 수 있음
// 컴파일, 유지보수, 디버깅 유리, 명령어를 더 적게 사용
// 스택머신을 Monkey 언어에서 사용: 이해가 쉽고 말들기 쉽다.
import (
	"MonkeyKids/code"
	"MonkeyKids/compiler"
	"MonkeyKids/object"
	"fmt"
)

const GlobalsSize = 65536
const StackSize = 2048
const MaxFrames = 1024

// true 는 언제나 true, false는 언제나 false 그래서 전역 변수로 정의 (성능면에서)
// 인덱스 범위 초과로 패닉 발생을 방지
var True = &object.Boolean{Value: true}
var False = &object.Boolean{Value: false}

var Null = &object.Null{}

type VM struct {
	constants   []object.Object
	stack       []object.Object // stack은 요소의 수를 나타내는 StackSize만큼의 크기로 미리 할당
	sp          int             // 언제나 다음값을 가리킴. 다라서 스택 최상단은 stack[sp-1],  sp는 언제나 스텍에서 비어있는 다음 슬롯을 가리킨다.
	globals     []object.Object // 가상머신에서 전역 바인딩 구하기
	frames      []*Frame
	framesIndex int
}

func New(bytecode *compiler.Bytecode) *VM {
	mainFn := &object.CompiledFunction{Instructions: bytecode.Instructions}
	mainFrame := NewFrame(mainFn, 0)

	frames := make([]*Frame, MaxFrames)
	frames[0] = mainFrame

	return &VM{
		constants:   bytecode.Constants,
		stack:       make([]object.Object, StackSize),
		sp:          0,
		globals:     make([]object.Object, GlobalsSize),
		frames:      frames,
		framesIndex: 1,
	}
}

// 인출-복호화-실행 주기가 구현
func (vm *VM) Run() error {
	var ip int
	var ins code.Instructions
	var op code.Opcode

	for vm.currentFrame().ip < len(vm.currentFrame().Instructions())-1 {
		vm.currentFrame().ip++

		ip = vm.currentFrame().ip
		ins = vm.currentFrame().Instructions()
		op = code.Opcode(ins[ip])

		// 복호화: case를 추가해서 명령어가 가진 피연산자를 복호화한다
		switch op {
		case code.OpConstant:
			// ReadUint16를 ReadOperands대신 쓰는 이유는 속도 때문에
			constIndex := code.ReadUint16(ins[ip+1:])
			vm.currentFrame().ip += 2
			err := vm.Push(vm.constants[constIndex])
			if err != nil {
				return err
			}

		case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv:
			err := vm.executeBinaryOperation(op)
			if err != nil {
				return err
			}

		case code.OpPop:
			vm.Pop()

		case code.OpTrue:
			err := vm.Push(True)
			if err != nil {
				return err
			}

		case code.OpFalse:
			err := vm.Push(False)
			if err != nil {
				return err
			}

		case code.OpEqual, code.OpNotEqual, code.OpGreaterThan:
			err := vm.executeComparison(op)
			if err != nil {
				return err
			}

		case code.OpBang:
			err := vm.executeBangOperator()
			if err != nil {
				return err
			}

		case code.OpMinus:
			err := vm.executeMinusOperator()
			if err != nil {
				return err
			}

		case code.OpJump:
			pos := int(code.ReadUint16(ins[ip+1:]))
			vm.currentFrame().ip = pos - 1 // 점프에서 도착해야할 목적지

		case code.OpJumpNotTruthy:
			pos := int(code.ReadUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2

			condition := vm.Pop()
			if !isTruthy(condition) {
				vm.currentFrame().ip = pos - 1
			}

			// 조건식은 표현식이면 표현식이라면 어떤 것과도 바꿔 쓸 수 있다. : 어떤 표현식이든 가상 머신에서 Null을 만들 수 있다.
			// 가상머신에서는 executeBinaryOperation처럼 의도하지 않은 값이 발생하면 에러처리
			// 명시적으로 Null을 처리해야 하는 함수와 메서드가 있다. : executeBangOperator
		case code.OpNull:
			err := vm.Push(Null)
			if err != nil {
				return err
			}

		case code.OpSetGlobal:
			globalIndex := code.ReadUint16(ins[ip+1:])
			vm.currentFrame().ip += 2

			vm.globals[globalIndex] = vm.Pop()

		case code.OpGetGlobal:
			globalIndex := code.ReadUint16(ins[ip+1:])
			vm.currentFrame().ip += 2
			err := vm.Push(vm.globals[globalIndex])
			if err != nil {
				return err
			}

		case code.OpArray:
			numElements := int(code.ReadUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2

			array := vm.buildArray(vm.sp-numElements, vm.sp)
			vm.sp = vm.sp - numElements

			err := vm.Push(array)
			if err != nil {
				return err
			}

		case code.OpHash:
			numElements := int(code.ReadUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2

			hash, err := vm.buildHash(vm.sp-numElements, vm.sp)
			if err != nil {
				return err
			}
			vm.sp = vm.sp - numElements
			err = vm.Push(hash)
			if err != nil {
				return err
			}

		case code.OpIndex:
			index := vm.Pop()
			left := vm.Pop()

			err := vm.executeIndexExpression(left, index)
			if err != nil {
				return err
			}

		case code.OpCall:
			numArgs := code.ReadUint8(ins[ip+1:])
			vm.currentFrame().ip += 1 // 피연산자 자리에 빈 바이트 하나를 추가한다.
			err := vm.callFunction(int(numArgs))
			if err != nil {
				return err
			}

		case code.OpReturnValue:
			returnValue := vm.Pop()

			frame := vm.popFrame()
			vm.sp = frame.basePointer - 1

			err := vm.Push(returnValue)
			if err != nil {
				return err
			}

		case code.OpReturn:
			frame := vm.popFrame()
			vm.sp = frame.basePointer - 1 // 1을 빼는 이유는 최적화때문에

			err := vm.Push(Null)
			if err != nil {
				return err
			}

		case code.OpSetLocal:
			localIndex := code.ReadUint8(ins[ip+1:])
			vm.currentFrame().ip += 1

			frame := vm.currentFrame()
			vm.stack[frame.basePointer+int(localIndex)] = vm.Pop()

		case code.OpGetLocal:
			localIndex := code.ReadUint8(ins[ip+1:])
			vm.currentFrame().ip += 1

			frame := vm.currentFrame()

			err := vm.Push(vm.stack[frame.basePointer+int(localIndex)])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (vm *VM) Push(o object.Object) error {
	if vm.sp >= StackSize {
		return fmt.Errorf("stack overflow")
	}
	vm.stack[vm.sp] = o
	vm.sp++

	return nil
}

// 방금 꺼낸 요소가 있던 자리를 덮어쓰게 된다
func (vm *VM) Pop() object.Object {
	o := vm.stack[vm.sp-1]
	vm.sp--
	return o
}

func (vm *VM) LastPoppedStackElem() object.Object {
	return vm.stack[vm.sp]
}

func (vm *VM) executeBinaryOperation(op code.Opcode) error {
	right := vm.Pop()
	left := vm.Pop()

	leftType := left.Type()
	rightType := right.Type()

	switch {
	case leftType == object.INTEGER_OBJ && rightType == object.INTEGER_OBJ:
		return vm.executeBinaryIntegerOperation(op, left, right)

	case leftType == object.STRING_OBJ && rightType == object.STRING_OBJ:
		return vm.executeBinaryStringOperation(op, left, right)
	default:
		return fmt.Errorf("unsupported types for binary operation: %s %s",
			leftType, rightType)
	}
}

func (vm *VM) executeBinaryIntegerOperation(op code.Opcode, left object.Object, right object.Object) error {
	leftValue := left.(*object.Integer).Value
	rightValue := right.(*object.Integer).Value
	var result int64

	switch op {
	case code.OpAdd:
		result = leftValue + rightValue
	case code.OpSub:
		result = leftValue - rightValue
	case code.OpMul:
		result = leftValue * rightValue
	case code.OpDiv:
		result = leftValue / rightValue
	default:
		return fmt.Errorf("unknown integer operator: %d", op)
	}
	return vm.Push(&object.Integer{Value: result})
}

func (vm *VM) executeComparison(op code.Opcode) error {
	right := vm.Pop()
	left := vm.Pop()

	if left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ {
		return vm.executeIntegerComparison(op, left, right)
	}
	switch op {
	case code.OpEqual:
		return vm.Push(nativeBoolToBooleanObject(right == left))
	case code.OpNotEqual:
		return vm.Push(nativeBoolToBooleanObject(right != left))
	default:
		return fmt.Errorf("unknown operator: %d (%s %s)",
			op, left.Type(), right.Type())
	}

}

func (vm *VM) executeIntegerComparison(op code.Opcode, left object.Object, right object.Object) error {
	leftValue := left.(*object.Integer).Value
	rightValue := right.(*object.Integer).Value

	switch op {
	case code.OpEqual:
		return vm.Push(nativeBoolToBooleanObject(rightValue == leftValue))
	case code.OpNotEqual:
		return vm.Push(nativeBoolToBooleanObject(rightValue != leftValue))
	case code.OpGreaterThan:
		return vm.Push(nativeBoolToBooleanObject(leftValue > rightValue))
	default:
		return fmt.Errorf("unknown operator: %d", op)
	}
}

func (vm *VM) executeBangOperator() error {
	operand := vm.Pop()

	switch operand {
	case True:
		return vm.Push(False)
	case False:
		return vm.Push(True)
	case Null:
		return vm.Push(True)
	default:
		return vm.Push(False)
	}
}

func (vm *VM) executeMinusOperator() error {
	operand := vm.Pop()

	if operand.Type() != object.INTEGER_OBJ {
		return fmt.Errorf("unsupported type for negation: %s", operand.Type())
	}

	value := operand.(*object.Integer).Value
	return vm.Push(&object.Integer{Value: -value})
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return True
	} else {
		return False
	}
}

func isTruthy(obj object.Object) bool {
	switch obj := obj.(type) {

	case *object.Boolean:
		return obj.Value

	case *object.Null:
		return false

	default:
		return true
	}
}

// 가상 머신에서 사용할 새로운 생성자
func NewWithGlobalsStore(bytecode *compiler.Bytecode, s []object.Object) *VM {
	vm := New(bytecode)
	vm.globals = s
	return vm
}

func (vm *VM) executeBinaryStringOperation(op code.Opcode, left object.Object, right object.Object) error {
	if op != code.OpAdd {
		return fmt.Errorf("unknown string operator: %d", op)
	}
	leftValue := left.(*object.String).Value
	rightValue := right.(*object.String).Value

	return vm.Push(&object.String{Value: leftValue + rightValue})
}

func (vm *VM) buildArray(startIndex int, endIndex int) object.Object {
	elements := make([]object.Object, endIndex-startIndex)

	for i := startIndex; i < endIndex; i++ {
		elements[i-startIndex] = vm.stack[i]
	}
	return &object.Array{Elements: elements}
}

func (vm *VM) buildHash(startIndex int, endIndex int) (object.Object, error) {
	hashedParis := make(map[object.HashKey]object.HashPair)

	for i := startIndex; i < endIndex; i += 2 {
		key := vm.stack[i]
		value := vm.stack[i+1]

		pair := object.HashPair{Key: key, Value: value}

		hashKey, ok := key.(object.Hashable)
		if !ok {
			return nil, fmt.Errorf("unusable as hash key: %s", key.Type())
		}
		hashedParis[hashKey.HashKey()] = pair
	}
	return &object.Hash{Pairs: hashedParis}, nil
}

func (vm *VM) executeIndexExpression(left object.Object, index object.Object) error {
	switch {
	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		return vm.executeArrayIndex(left, index)

	case left.Type() == object.HASH_OBJ:
		return vm.executeHashIndex(left, index)

	default:
		return fmt.Errorf("index operator not supported: %s", left.Type())
	}
}

func (vm *VM) executeArrayIndex(array object.Object, index object.Object) error {
	arrayObject := array.(*object.Array)
	i := index.(*object.Integer).Value
	max := int64(len(arrayObject.Elements) - 1)

	if i < 0 || i > max {
		return vm.Push(Null)
	}
	return vm.Push(arrayObject.Elements[i])
}

func (vm *VM) executeHashIndex(hash object.Object, index object.Object) error {
	hashObject := hash.(*object.Hash)

	key, ok := index.(object.Hashable)
	if !ok {
		return fmt.Errorf("unusable as hash key: %s", index.Type())
	}
	pair, ok := hashObject.Pairs[key.HashKey()]
	if !ok {
		return vm.Push(Null)
	}
	return vm.Push(pair.Value)
}
func (vm *VM) currentFrame() *Frame {
	return vm.frames[vm.framesIndex-1]
}
func (vm *VM) pushFrame(f *Frame) {
	vm.frames[vm.framesIndex] = f
	vm.framesIndex++
}
func (vm *VM) popFrame() *Frame {
	vm.framesIndex--
	return vm.frames[vm.framesIndex]
}

func (vm *VM) callFunction(numArgs int) error {
	// 함수를 호출하기 전에 스택에 지역바인딩을 저장할 빈공간을 할당한다.
	// 가상머신에서 OpSetLocal, OpGetLocal 명령어를 처리할 수 있게 구현한다.
	fn, ok := vm.stack[vm.sp-1-numArgs].(*object.CompiledFunction)
	if !ok {
		return fmt.Errorf("calling non-function")
	}
	if numArgs != fn.NumParameters {
		return fmt.Errorf("wrong number of arguments: want=%d, got=%d", fn.NumParameters, numArgs)
	}
	frame := NewFrame(fn, vm.sp-numArgs)
	vm.pushFrame(frame)

	vm.sp = frame.basePointer + fn.NumLocals

	return nil
}
