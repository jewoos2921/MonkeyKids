package evaluator

import "MonkeyKids/object"

var builtins = map[string]*object.Builtin{
	"len": &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			// len 함수를 호출하는 행
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			switch arg := args[0].(type) {
			// 새로 할당한 object.Integer를 반환하는 행
			case *object.String:
				return &object.Integer{Value: int64(len(arg.Value))}
				// 내장 함수를 올바른 인수 개수로 호출하는가, 올바른 타입인가 ?
			default:
				return newError("arguments to len not supported, got %s", args[0].Type())
			}
		},
	},
}
