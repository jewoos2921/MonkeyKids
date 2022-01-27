package compiler

import (
	"MonkeyKids/ast"
	"MonkeyKids/code"
	"MonkeyKids/object"
	"fmt"
)

// 컴파일러를 고쳐서 마지막으로 배출한 명령어 둘을 추적해야한다.
// 두명령어를 갖는 명령코드와 배출된 위치를 추적할 수 있어야 한다.
type Compiler struct {
	instructions        code.Instructions
	constants           []object.Object
	lastInstruction     EmittedInstruction // 가장 마지막으로 배출한 명려어
	previousInstruction EmittedInstruction // lastInstruction 직전에 배출된 명령어
}

func New() *Compiler {
	return &Compiler{instructions: code.Instructions{}, constants: []object.Object{},
		lastInstruction: EmittedInstruction{}, previousInstruction: EmittedInstruction{}}
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
		c.emit(code.OpJumpNotTruthy, 9999)
		// node.Consequence 은 표현식
		// node.Consequence 에서 마지막 OpPop 명령어만 제거해야 한다.
		err = c.Compile(node.Consequence)
		if err != nil {
			return err
		}

		if c.lastInstructionIsPop() {
			c.removeLastPop()
		}
	case *ast.BlockStatement:
		for _, s := range node.Statements {
			err := c.Compile(s)
			if err != nil {
				return err
			}
		}

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
