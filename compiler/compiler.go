package compiler

import (
	"MonkeyKids/ast"
	"MonkeyKids/code"
	"MonkeyKids/object"
	"fmt"
	"sort"
)

// 컴파일러를 고쳐서 마지막으로 배출한 명령어 둘을 추적해야한다.
// 두명령어를 갖는 명령코드와 배출된 위치를 추적할 수 있어야 한다.
type Compiler struct {
	instructions        code.Instructions
	constants           []object.Object
	lastInstruction     EmittedInstruction // 가장 마지막으로 배출한 명려어
	previousInstruction EmittedInstruction // lastInstruction 직전에 배출된 명령어
	symbolTable         *SymbolTable
}

func New() *Compiler {
	return &Compiler{instructions: code.Instructions{}, constants: []object.Object{},
		lastInstruction: EmittedInstruction{}, previousInstruction: EmittedInstruction{},
		symbolTable: NewSymbolTable()}
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
		if err != nil {
			return err
		}

		if c.lastInstructionIsPop() {
			c.removeLastPop()
		}

		// OpJump는 조건이 참 같은 값을로 핵석됐을 때, 조건식 else 분기를 지나가야 한다.
		// OpJump는 컨시퀀스에 속해 있는 셈
		// OpJump 명령어에 쓰레기값 9999를 넣어서 배출
		jumpPos := c.emit(code.OpJump, 9999)

		afterConsequencePos := len(c.instructions) // 다음에 배출할 명령어가 갖는 오프셋 값을 계산한다.
		c.changedOperand(jumpNotTruthyPos, afterConsequencePos)

		if node.Alternative == nil {
			c.emit(code.OpNull)
		} else {
			err := c.Compile(node.Alternative)
			if err != nil {
				return err
			}
			if c.lastInstructionIsPop() {
				c.removeLastPop()
			}
		}
		afterAlternativePos := len(c.instructions)
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
		}
		c.emit(code.OpHash, len(node.Pairs)*2)
	}

	return nil
}

func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode{Instructions: c.instructions, Constants: c.constants}
}

type Bytecode struct {
	Instructions code.Instructions
	Constants    []object.Object
}

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
	posNewInstruction := len(c.instructions)
	c.instructions = append(c.instructions, ins...)
	return posNewInstruction
}

type EmittedInstruction struct {
	Opcode   code.Opcode
	Position int
}

// 마지막 명령어가 가진 명령 코드를 타입 안정성(명령코드를 byte로 변환하거나, byte를 명령코드로 변환할 필요가 없다는 뜻)있게 검사할 수 있다.
func (c *Compiler) setLastInstruction(op code.Opcode, pos int) {
	previous := c.lastInstruction
	last := EmittedInstruction{Opcode: op, Position: pos}

	c.previousInstruction = previous
	c.lastInstruction = last
}

// 마지막 명령코드가 OpPop 인지 검사한다.
func (c *Compiler) lastInstructionIsPop() bool {
	return c.lastInstruction.Opcode == code.OpPop
}

// 	c.instructions에서 마지막 명령어를 잘라내고 	c.instructions을 previousInstruction로 바꾼다.
func (c *Compiler) removeLastPop() {
	c.instructions = c.instructions[:c.lastInstruction.Position]
	c.lastInstruction = c.previousInstruction
}

// instructions 슬라이스의 임의의 오프셋 값에 위치한 명령어를 바꿀때 사용
func (c *Compiler) replaceInstruction(pos int, newInstruction []byte) {
	for i := 0; i < len(newInstruction); i++ {
		c.instructions[pos+i] = newInstruction[i]
	}
}

// 피연산자 변경
func (c *Compiler) changedOperand(opPos int, operand int) {
	op := code.Opcode(c.instructions[opPos])
	newInstruction := code.Make(op, operand)

	c.replaceInstruction(opPos, newInstruction)
}

func NewWithStates(s *SymbolTable, constants []object.Object) *Compiler {
	compiler := New()
	compiler.symbolTable = s
	compiler.constants = constants
	return compiler
}
