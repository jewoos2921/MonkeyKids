package repl

// REPL(Read-Eval-Print-Loop)는 단일 사용자의 입력을 받아서 이를 평가하고,
// 그결과를 사용자에게 다시 반환하는 단순한 컴퓨터 프로그래밍 환경
// 렉서->파서->추상구문트리->내부객체시스템->평가
// READ->PARSE->PRINT->LOOP

import (
	"MonkeyKids/compiler"
	"MonkeyKids/lexer"
	"MonkeyKids/object"
	"MonkeyKids/parser"
	"MonkeyKids/vm"
	"bufio"
	"fmt"
	"io"
)

const PROMPT = ">>"

// 컴파일러와 가상머신을 REPL 에 연동
// 면저 입력을 토큰화하고 파싱한 다음, 컴파일하고 프로그래밍을 실행하면 된다.
// 그리고 전에는 Eval 함수에서 반환값을 출력했지만, 이번에는 가상 머신 스택
// 가장 위에 있는 객체를 출력하도록 바꾸면 된다.
func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)

	var constants []object.Object
	globals := make([]object.Object, vm.GlobalsSize)
	symbolTable := compiler.NewSymbolTable()

	for i, v := range object.Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}

	for {
		fmt.Fprintf(out, PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		l := lexer.New(line)
		p := parser.New(l)

		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			printParserErrors(out, p.Errors())
			continue
		}

		comp := compiler.NewWithStates(symbolTable, constants)
		err := comp.Compile(program)
		if err != nil {
			fmt.Fprintf(out, "Woops! Compilation failed:\n %s\n", err)
			continue
		}

		code := comp.Bytecode()
		constants = code.Constants
		machine := vm.NewWithGlobalsStore(code, globals)

		err = machine.Run()
		if err != nil {
			fmt.Fprintf(out, "Woops! Executing bytecode failed:\n %s\n", err)
			continue
		}
		lastPopped := machine.LastPoppedStackElem()
		io.WriteString(out, lastPopped.Inspect())
		io.WriteString(out, "\n")
	}
}

func printParserErrors(out io.Writer, errors []string) {
	for _, msg := range errors {
		io.WriteString(out, "\t"+msg+"\n")
	}
}

const MONKEY_FACE = `            __,__
   .--.  .-"     "-.  .--.
  / .. \/  .-. .-.  \/ .. \
 | |  '|  /   Y   \  |'  | |
 | \   \  \ 0 | 0 /  /   / |
  \ '- ,\.-"""""""-./, -' /
   ''-' /_   ^ ^   _\ '-''
       |  \._   _./  |
       \   \ '~' /   /
        '._ '-=-' _.'
           '-----'
`

func printParseErrors(out io.Writer, errors []string) {
	io.WriteString(out, MONKEY_FACE)
	io.WriteString(out, "Woops! we ran into some monkey business here!\n")
	io.WriteString(out, " parser errors:\n")
	for _, msg := range errors {
		io.WriteString(out, "\t"+msg+"\n")
	}
}
