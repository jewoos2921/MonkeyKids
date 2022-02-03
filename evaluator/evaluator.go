package evaluator

// 평가 전략
// 인터프리터는 컴파일러와 다르게 실핼 가능한 중간 생성물을 남기지 않는다.
// 여기서는 그냥 번역
// AST를 순회하고, 각각의 노드를 방문해서 노드가 갖는 의미대로 즉시 처리한다.
// 트리순회인터프리터라고 부르며 대다수 인터프리터의 원형이다.
// 안쓰는 변수를 제거하거나, 재귀혹은 반복평가에 더욱 적합한 중간표현물(IR)로 변경
// 바이트코드로 변환
// 바이트코드역시 IR의 한 형태
// 바이트코드 하나에는 AST를 나타내는 정보가 빽빽하게 들어가 있다.
// 바이트코드가 구성되는 구체적 형식과 어떤 명령코드로 구성되는지 게스트언어와 호스트언어에 따라 달라진다.
// 또 다른 전략은 AST를 고려하지 않는다.
// 파서는 바이트코드를 직접 배출 -> JIT
// 어떤 구현체는 바이트코드로 컴파일하는 작업을 지나친다.
// 재귀적으로 AST를 순회하지만, AST의 특정 가지를 실행전, 노드를 네이티브 머신 코드로 컴파일,
// 컴파일하는 즉시 실행
import (
	"MonkeyKids/ast"
	"MonkeyKids/object"
	"fmt"
)

// true나 false를 만날때마다 object.Boolean을 다시 만들어야 하는가??
// 미리 만들어논 reference를 사용하자
var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

// 자체평가 표현식
// 평가 관점에서 리터럴을 부를때 쓰는 이름
// 정수 리터럴 하나만 포함하는 표현식문이 입력으로 주어졌을 때, 이 정수 리터럴을 평가하여 다시 정수 그 자체를 반환한다.
// 평가 프로세스를 거치고 나면 입력 언어는 의미를 갖게 된다
// env *object.Environment
// 환경은 인터프리터가 값을 추적할 때 사용하는 객체로, 값을 이름과연관시킨다.
// 그저 문자열과 객체를 연관시키는 해시 맵에 불과
func Eval(node ast.Node, env *object.Environment) object.Object {
	switch node := node.(type) {
	// 명령문
	case *ast.Program:
		return evalProgram(node, env)

	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)

	case *ast.Boolean:
		return nativeBoolToBooleanObject(node.Value)

	case *ast.BlockStatement:
		return evalBlockStatements(node, env)

	case *ast.IfExpression:
		return evalIfExpression(node, env)

	case *ast.ReturnStatement:
		val := Eval(node.ReturnValue, env)
		if isError(val) {
			return val
		}
		return &object.ReturnValue{Value: val}

	case *ast.PrefixExpression:
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node.Operator, right)

	case *ast.InfixExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalInfixExpression(node.Operator, left, right)

	case *ast.Identifier:
		return evalIdentifier(node, env)

	case *ast.LetStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		env.Set(node.Name.Value, val)

	case *ast.FunctionLiteral:
		params := node.Parameters
		body := node.Body
		return &object.Function{Parameters: params, Env: env, Body: body}

		// 인수를 평가하는 동작은 표현식 리스트를 평가하는 동작과 다를바 없다.
		// 표현식
	case *ast.CallExpression:
		function := Eval(node.Function, env)
		if isError(function) {
			return function
		}
		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}
		return applyFunction(function, args)

	case *ast.StringLiteral:
		return &object.String{Value: node.Value}

	case *ast.ArrayLiteral:
		elements := evalExpressions(node.Elements, env)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &object.Array{Elements: elements}

	case *ast.IndexExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		index := Eval(node.Index, env)

		if isError(index) {
			return index
		}
		return evalIndexExpression(left, index)

	case *ast.HashLiteral:
		return evalHashLiteral(node, env)

	case *ast.IntegerLiteral:
		// 순회는 언제나 트리 최상단에서 시작해야 한다.
		return &object.Integer{Value: node.Value}
	}
	return nil
}

// 동등성을 포인터 비교 (left == right) 로 검사
// 정수나 후에 나올 다른 데이터 타입에는 적용 불가
func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return TRUE
	}
	return FALSE
}

func evalPrefixExpression(operator string, right object.Object) object.Object {
	switch operator {
	case "!":
		return evalBangOperatorExpression(right)
	case "-":
		return evalMinusPrefixOperatorExpression(right)

	default:
		return newError("unknown operator: %s%s", operator, right.Type())
	}
}

// !을 처리하는 동작을 구현
func evalBangOperatorExpression(right object.Object) object.Object {
	switch right {
	case TRUE:
		return FALSE
	case FALSE:
		return TRUE
	case NULL:
		return TRUE
	default:
		return FALSE
	}
}

func evalMinusPrefixOperatorExpression(right object.Object) object.Object {
	if right.Type() != object.INTEGER_OBJ {
		return newError("unknown operator: -%s", right.Type())
	}
	value := right.(*object.Integer).Value
	return &object.Integer{Value: -value} // 새로 객체를 할당
}

func evalInfixExpression(operator string,
	left object.Object, right object.Object) object.Object {
	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right)

	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return evalStringInfixExpression(operator, left, right)

	case operator == "==":
		return nativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return nativeBoolToBooleanObject(left != right)

	case left.Type() != right.Type():
		return newError("type mismatch: %s %s %s", left.Type(), operator, right.Type())

	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalIntegerInfixExpression(operator string,
	left object.Object, right object.Object) object.Object {

	leftVal := left.(*object.Integer).Value
	rightVal := right.(*object.Integer).Value

	switch operator {
	case "+":
		return &object.Integer{Value: leftVal + rightVal}
	case "-":
		return &object.Integer{Value: leftVal - rightVal}
	case "*":
		return &object.Integer{Value: leftVal * rightVal}
	case "/":
		return &object.Integer{Value: leftVal / rightVal}
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())

	}
}

func evalIfExpression(ie *ast.IfExpression, env *object.Environment) object.Object {
	condition := Eval(ie.Condition, env)

	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return Eval(ie.Consequence, env)
	} else if ie.Alternative != nil {
		return Eval(ie.Alternative, env)
	} else {
		return NULL
	}
}

func isTruthy(obj object.Object) bool {
	switch obj {
	case NULL:
		return false
	case TRUE:
		return true
	case FALSE:
		return false
	default:
		return true
	}
}

// 명시적으로 반환값을 풀지 않고 각 평가 결과의 Type만 검사
func evalProgram(program *ast.Program, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range program.Statements {
		result = Eval(statement, env)

		switch result := result.(type) {
		case *object.ReturnValue:
			return result.Value

		case *object.Error:
			return result
		}
	}
	return result
}

// 명시적으로 반환값을 풀지 않고 각 평가 결과의 Type만 검사
func evalBlockStatements(block *ast.BlockStatement, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range block.Statements {
		result = Eval(statement, env)

		// object.RETURN_VALUE_OBJ이면 풀지않고 그대로 반환
		if result != nil {
			rt := result.Type()
			if rt == object.RETURN_VALUE_OBJ || rt == object.ERROR_OBJ {
				return result
			}
		}
	}
	return result
}

// 내부 에러처리: 잘못된 연산자를 쓰거나 지원되지 않은 연산을 한다거나 혹은 그 밖에 실행 중에 일어 날수 있는 사용자 또는 내부에러를 말함
// 앞서 작성한 코드에서 어떤 동작으로 처리해야 할지 몰라 그냥 NULL을 반환한 모든 코드를 대체 할것이다.
func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR_OBJ
	}
	return false
}

// 식별자가 주어졌을 때, 현재 환경에서 바인딩된 값을 찾을 수 없다면, 기본값으로 내장함수를 찾도록 만들어야 한다.
func evalIdentifier(node *ast.Identifier,
	env *object.Environment) object.Object {
	if val, ok := env.Get(node.Value); ok {
		return val
	}

	if builtin, ok := builtins[node.Value]; ok {
		return builtin
	}

	return newError("identifier not found: " + node.Value)
}

func evalExpressions(exp []ast.Expression, env *object.Environment) []object.Object {
	var result []object.Object

	for _, e := range exp {
		evaluated := Eval(e, env)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}
	return result
}

func applyFunction(fn object.Object, args []object.Object) object.Object {

	switch fn := fn.(type) {
	case *object.Function:
		extendEnv := extendFunctionEnv(fn, args)
		evaluated := Eval(fn.Body, extendEnv)
		return unwrapReturnValue(evaluated)

	case *object.Builtin:
		return fn.Fn(args...)
	default:
		return newError("not a function: %s", fn.Type())
	}

}

func extendFunctionEnv(fn *object.Function,
	args []object.Object) *object.Environment {

	env := object.NewEnclosedEnvironment(fn.Env)

	for paramIdx, param := range fn.Parameters {
		env.Set(param.Value, args[paramIdx])
	}
	return env
}

func unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*object.ReturnValue); ok {
		return returnValue.Value
	}
	return obj
}

func evalStringInfixExpression(operator string, left object.Object, right object.Object) object.Object {
	if operator != "+" {
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}

	leftVal := left.(*object.String).Value
	rightVal := right.(*object.String).Value

	return &object.String{Value: leftVal + rightVal}
}

func evalIndexExpression(left object.Object, index object.Object) object.Object {
	switch {
	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		return evalArrayIndexExpression(left, index)

	case left.Type() == object.HASH_OBJ:
		return evalHashIndexExpression(left, index)
	default:
		return newError("index operator not supported: %s", left.Type())
	}
}

// 주어진 인덱스가 범위를 벗어나는지 검사하고 벗어나면 NULL을 반환, 그렇지 않으면 해당 요소를 반환
func evalArrayIndexExpression(array object.Object, index object.Object) object.Object {
	arrayObject := array.(*object.Array)
	idx := index.(*object.Integer).Value
	max := int64(len(arrayObject.Elements) - 1)
	if idx < 0 || idx > max {
		return NULL
	}
	return arrayObject.Elements[idx]
}

func evalHashLiteral(node *ast.HashLiteral, env *object.Environment) object.Object {
	pairs := make(map[object.HashKey]object.HashPair)

	for keyNode, valueNode := range node.Pairs {
		key := Eval(keyNode, env)
		if isError(key) {
			return key
		}

		hashKey, ok := key.(object.Hashable)

		if !ok {
			return newError("unusable as hash key: %s", key.Type())
		}
		value := Eval(valueNode, env)
		if isError(value) {
			return value
		}

		hashed := hashKey.HashKey()
		pairs[hashed] = object.HashPair{Key: key, Value: value}
	}

	return &object.Hash{Pairs: pairs}
}

func evalHashIndexExpression(hash object.Object, index object.Object) object.Object {
	hashObject := hash.(*object.Hash)
	key, ok := index.(object.Hashable)
	if !ok {
		return newError("unusable as has key: %s", index.Type())
	}
	pair, ok := hashObject.Pairs[key.HashKey()]
	if !ok {
		return NULL
	}
	return pair.Value
}
