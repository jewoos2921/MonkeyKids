package evaluator

import (
	"MonkeyKids/object"
)

// 내장 함수들
// 내장함수를 스택에 넣는다.
// 호출 인수를 스텍에 넣는다.
// OpCall 명령어로 함수를 호출한다.
var builtins = map[string]*object.Builtin{
	"len":   object.GetBuiltinByName("len"),
	"puts":  object.GetBuiltinByName("puts"),
	"first": object.GetBuiltinByName("first"),
	"last":  object.GetBuiltinByName("last"),
	"rest":  object.GetBuiltinByName("rest"),
	"push":  object.GetBuiltinByName("push"),
}
