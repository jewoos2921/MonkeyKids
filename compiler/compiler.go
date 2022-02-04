package compiler

import (
	"MonkeyKids/ast"
	"MonkeyKids/code"
	"MonkeyKids/object"
	"fmt"
	"sort"
)

// 컴파일러 : 실행 프로그램을 만들어낸다.
// 프런트엔드에서는 소스언어로 된 소스코드를 읽어 들여 특정 데이터 구조로 변환한다.
// 인터프리터와 컴파일러 프러트엔드 양쪽모두 렉서와 파서로 구성
// 컴파일러는 AST를 평가시 아무것도 출력하지 않는다. 목적언어로 된 소스코드를 생성
// 소스->렉서&파서->AST->옵티마이저->내부표현->코드생성기->머신코드
// 옵티마이저: IR생성->불필요한 코드 제거, 단순한 산술연산 미리 계산, 반복문 몸체에 있을 필요가 없을 코드밖으로 빼기
// 코드제네레이터가 목적언어로 된 코드 생성
// 백엔드라고 칭함
// 옵티마이저가 IR을 여러 패스에 걸쳐서 처리
// 컴파일러는 변환이 핵심

// 컴파일러를 고쳐서 마지막으로 배출한 명령어 둘을 추적해야한다.
// 두명령어를 갖는 명령코드와 배출된 위치를 추적할 수 있어야 한다.
type Compiler struct {
	constants   []object.Object
	symbolTable *SymbolTable
	scopes      []CompilationScope
	scopeIndex  int
}

/*
		최소한의 컴파일러
전달받은 AST를 순회한다.
*ast.IntegerLiteral노드를 찾는다.
*ast.IntegerLiteral을 평가한 다음 object.Integer 객체로 변환한다.
변환한 객체를 상수 풀에 추가한다.
상수 풀에 있는 상수를 참조하는 OpConstant명령어를 배출한다.
*/
func New() *Compiler {
	mainScope := CompilationScope{
		instructions:        code.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}
	return &Compiler{
		constants:   []object.Object{},
		symbolTable: NewSymbolTable(),
		scopes:      []CompilationScope{mainScope},
		scopeIndex:  0,
	}
}

func (c *Compiler) Compile(node ast.Node) error {
	switch node := node.(type) {

	case *ast.Program:
		for _, s := range node.Statements {
			err := c.Compile(s)
			if err != nil {
				return err
			}
		}

	case *ast.ExpressionStatement:
		err := c.Compile(node.Expression)
		if err != nil {
			return err
		}
		c.emit(code.OpPop)

	case *ast.InfixExpression:
		// < 연산자를 처리할 때 , 피연산자의 순서를 변경하기 원해서
		// if node.Operator == "<"를 가장 먼저 배치
		// '컴파일 타임'에 비교연산 < 을 비교연산 > 로 바꾼다.
		if node.Operator == "<" {
			err := c.Compile(node.Right)
			if err != nil {
				return err
			}
			err = c.Compile(node.Left)
			if err != nil {
				return err
			}
			c.emit(code.OpGreaterThan)
			return nil
		}

		err := c.Compile(node.Left)
		if err != nil {
			return err
		}
		err = c.Compile(node.Right)
		if err != nil {
			return err
		}

		switch node.Operator {
		case "+":
			c.emit(code.OpAdd)
		case "-":
			c.emit(code.OpSub)
		case "*":
			c.emit(code.OpMul)
		case "/":
			c.emit(code.OpDiv)
		case ">":
			c.emit(code.OpGreaterThan)
		case "==":
			c.emit(code.OpEqual)
		case "!=":
			c.emit(code.OpNotEqual)

		default:
			// 컴파일 방법을 알 수 없는 중위 연산자를 만났을 때 에러를 반환하게 만든다.
			return fmt.Errorf("unknown operator %s", node.Operator)
		}

	case *ast.IntegerLiteral:
		integer := &object.Integer{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(integer))

	case *ast.Boolean:
		if node.Value {
			c.emit(code.OpTrue)
		} else {
			c.emit(code.OpFalse)
		}

	case *ast.PrefixExpression:
		err := c.Compile(node.Right)
		if err != nil {
			return err
		}
		switch node.Operator {
		case "!":
			c.emit(code.OpBang)
		case "-":
			c.emit(code.OpMinus)
		default:
			return fmt.Errorf("unknown operator %s", node.Operator)

		}

	case *ast.IfExpression:
		err := c.Compile(node.Condition)
		if err != nil {
			return err
		}
		// OpJumpNotTruthy 명령어에 쓰레기값 9999를 넣어서 배출
		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)

		// jumpNotTruthyPos 는 명령어를 찾을때 사용할 값이다.
		// OpPop 명령어가 있는지 검사하고, 있다면 제거하는 작업을 마친 이후를 말한다.
		// node.Consequence 은 표현식
		// node.Consequence 에서 마지막 OpPop 명령어만 제거해야 한다.
		err = c.Compile(node.Consequence)
		// 백패칭 사용
		// AST를 한 번만 순회하는 컴파일러르 단일 패스 컴파일러라 부르는데,
		// 이런 컴파일러에는 백 패칭은 아주 흔히 사용되는 기법이다.
		// 진보된 컴파일러에서는
		// 점프 명령어가 뛰어야 할 목적지로 얼마나 뛰어야 할지 실제로 알기전 까지는 비워두고,
		// AST를 한 번 더 순회하는 다음 패스에서, 얼마나 뛰어야 할지 알아낸 뒤에 값을 채운다.
		if err != nil {
			return err
		}

		if c.lastInstructionIs(code.OpPop) {
			c.removeLastPop()
		}

		// OpJump는 조건이 참 같은 값을로 핵석됐을 때, 조건식 else 분기를 지나가야 한다.
		// OpJump는 컨시퀀스에 속해 있는 셈
		// OpJump 명령어에 쓰레기값 9999를 넣어서 배출
		jumpPos := c.emit(code.OpJump, 9999)

		afterConsequencePos := len(c.currentInstructions()) // 다음에 배출할 명령어가 갖는 오프셋 값을 계산한다.
		c.changedOperand(jumpNotTruthyPos, afterConsequencePos)

		if node.Alternative == nil {
			c.emit(code.OpNull)
		} else {
			err := c.Compile(node.Alternative)
			if err != nil {
				return err
			}
			if c.lastInstructionIs(code.OpPop) {
				c.removeLastPop()
			}
		}
		afterAlternativePos := len(c.currentInstructions())
		c.changedOperand(jumpPos, afterAlternativePos)

	case *ast.BlockStatement:
		for _, s := range node.Statements {
			err := c.Compile(s)
			if err != nil {
				return err
			}
		}

	case *ast.LetStatement:
		// let문을 만나면 가장 먼저 연산자= 오른편에 있는 표현식을 컴파일
		// 이표현식이 만들어내는 Value가 식별자 이름에 바인딩한다.
		// 여기서 표현식을 컴파일한다는 것는 표현식이 만들어낸 값을 가상 머신이 스택에 넣도록 지시
		err := c.Compile(node.Value)
		if err != nil {
			return err
		}
		symbol := c.symbolTable.Define(node.Name.Value)
		c.emit(code.OpSetGlobal, symbol.Index)

	case *ast.Identifier:
		symbol, ok := c.symbolTable.Resolve(node.Value)
		// 가상 머신에서는 바이트 코드를 넘기기 전에 에러를 던질 수 있다.
		if !ok {
			return fmt.Errorf("undefined variable %s", node.Value)
		}

		c.emit(code.OpGetGlobal, symbol.Index)

	case *ast.StringLiteral:
		str := &object.String{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(str))

	case *ast.ArrayLiteral:
		for _, el := range node.Elements {
			err := c.Compile(el)
			if err != nil {
				return err
			}
		}
		c.emit(code.OpArray, len(node.Elements))

	case *ast.HashLiteral:
		var keys []ast.Expression
		for k := range node.Pairs {
			keys = append(keys, k)
		}
		// Go언어에서 map으로 range를 사용해서 반복하면, 순회 순서가 특정되지 않으며 다음에
		// 다시 반복했을 때 같은 순서가 보장되지 않는다.
		// 만약 안정된 반복순서가 필요하다면 이런 반복 순서를 명시하고
		// 있는 별도의 자료구조를 관리해야 한다.
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].String() < keys[j].String()
		})

		for _, k := range keys {
			err := c.Compile(k)
			if err != nil {
				return err
			}
			err = c.Compile(node.Pairs[k])
			if err != nil {
				return err
			}
		}
		c.emit(code.OpHash, len(node.Pairs)*2)

	case *ast.IndexExpression:
		err := c.Compile(node.Left)
		if err != nil {
			return err
		}
		err = c.Compile(node.Index)
		if err != nil {
			return err
		}
		c.emit(code.OpIndex)

	case *ast.FunctionLiteral:
		// 함수를 컴파일할 때 배출될 명령어가 저장되는 위치를 바꾸는 것
		c.enterScope()

		err := c.Compile(node.Body)
		if err != nil {
			return err
		}

		if c.lastInstructionIs(code.OpPop) {
			c.replaceLastPopWithReturn()
		}

		if !c.lastInstructionIs(code.OpReturnValue) {
			c.emit(code.OpReturn)
		}

		instructions := c.leaveScope()

		compiledFn := &object.CompiledFunction{Instructions: instructions}
		c.emit(code.OpConstant, c.addConstant(compiledFn))

	case *ast.ReturnStatement:
		// 반환값 자체를 컴파일
		err := c.Compile(node.ReturnValue)
		if err != nil {
			return err
		}
		c.emit(code.OpReturnValue)

	case *ast.CallExpression:
		// 가상 머신이 사용할 데이터만 변화시키면 된다.
		// 명령어와 명령어 포인터를 변경해야 한다.
		// 만약 가상 머신 실행 중에 명령어와 명령어 포인터를 변경할 수 있다면, 함수를 실행할 수 있다.
		err := c.Compile(node.Function)
		if err != nil {
			return err
		}
		c.emit(code.OpCall)

	}

	return nil
}

func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode{Instructions: c.currentInstructions(),
		Constants: c.constants}
}

type Bytecode struct {
	Instructions code.Instructions
	Constants    []object.Object
}

// 명령어를 만들고 만든 명령어를 결과에 추가한다.
// 지금 만들어낸 명령어의 시작 위치를 반환한다.
func (c *Compiler) emit(op code.Opcode, operands ...int) int {
	ins := code.Make(op, operands...)
	pos := c.addInstruction(ins)

	// 두 필드를 만들어야 한다.
	c.setLastInstruction(op, pos)

	return pos
}

func (c *Compiler) addConstant(obj object.Object) int {
	c.constants = append(c.constants, obj)
	return len(c.constants) - 1
}

func (c *Compiler) addInstruction(ins []byte) int {
	posNewInstruction := len(c.currentInstructions())
	updatedInstructions := append(c.currentInstructions(), ins...)
	c.scopes[c.scopeIndex].instructions = updatedInstructions
	return posNewInstruction
}

type EmittedInstruction struct {
	Opcode   code.Opcode
	Position int
}

// 마지막 명령어가 가진 명령 코드를 타입 안정성(명령코드를 byte로 변환하거나, byte를 명령코드로 변환할 필요가 없다는 뜻)있게 검사할 수 있다.
func (c *Compiler) setLastInstruction(op code.Opcode, pos int) {
	previous := c.scopes[c.scopeIndex].lastInstruction
	last := EmittedInstruction{Opcode: op, Position: pos}

	c.scopes[c.scopeIndex].previousInstruction = previous
	c.scopes[c.scopeIndex].lastInstruction = last
}

// 	c.instructions에서 마지막 명령어를 잘라내고 	c.instructions을 previousInstruction로 바꾼다.
func (c *Compiler) removeLastPop() {
	last := c.scopes[c.scopeIndex].lastInstruction
	previous := c.scopes[c.scopeIndex].previousInstruction

	old := c.currentInstructions()
	new := old[:last.Position]

	c.scopes[c.scopeIndex].instructions = new
	c.scopes[c.scopeIndex].lastInstruction = previous
}

// instructions 슬라이스의 임의의 오프셋 값에 위치한 명령어를 바꿀때 사용
func (c *Compiler) replaceInstruction(pos int, newInstruction []byte) {
	ins := c.currentInstructions()
	for i := 0; i < len(newInstruction); i++ {
		ins[pos+i] = newInstruction[i]
	}
}

// 피연산자 변경만 변경하는 게 아니라 바뀐 피연산자의 명령어를
// 다시 바꾸어 기존 명령어를 새로운 명령어로 갈아치운다
// 이때 명령어 타입이 갖고 명령어 길이가 변하지 않는 명령어만 바꿀수 있다.
func (c *Compiler) changedOperand(opPos int, operand int) {
	op := code.Opcode(c.currentInstructions()[opPos])
	newInstruction := code.Make(op, operand)

	c.replaceInstruction(opPos, newInstruction)
}

func (c *Compiler) currentInstructions() code.Instructions {
	return c.scopes[c.scopeIndex].instructions
}

// REPL에서 전역 상태가 유지되도록
func NewWithStates(s *SymbolTable, constants []object.Object) *Compiler {
	compiler := New()
	compiler.symbolTable = s
	compiler.constants = constants
	return compiler
}

// 스포크 추가
// 슬라이스 하나와 두개의 개별 필드 lastInstruction, previousInstruction을 이용해 배출한 명령어를 추적하는 대신,
// 이셋을 컴파일 스코프로 엮고 컴파일 스코프 스택으로 사용한다는 의미
type CompilationScope struct {
	instructions        code.Instructions
	lastInstruction     EmittedInstruction
	previousInstruction EmittedInstruction
}

func (c *Compiler) enterScope() {
	scope := CompilationScope{
		instructions:        code.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}
	c.scopes = append(c.scopes, scope)
	c.scopeIndex++
}

func (c *Compiler) leaveScope() code.Instructions {
	instructions := c.currentInstructions()

	c.scopes = c.scopes[:len(c.scopes)-1]
	c.scopeIndex--

	return instructions
}

// 마지막 명령코드가 OpPop 인지 검사한다.
func (c *Compiler) lastInstructionIs(op code.Opcode) bool {
	if len(c.currentInstructions()) == 0 {
		return false
	}
	return c.scopes[c.scopeIndex].lastInstruction.Opcode == op
}

func (c *Compiler) replaceLastPopWithReturn() {
	lastPos := c.scopes[c.scopeIndex].lastInstruction.Position
	c.replaceInstruction(lastPos, code.Make(code.OpReturnValue))

	c.scopes[c.scopeIndex].lastInstruction.Opcode = code.OpReturnValue
}
