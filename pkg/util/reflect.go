package util

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

// MarshalReturn 将 []reflect.Value 序列化为字符串
func MarshalReturn(values []reflect.Value) string {
	if len(values) == 0 {
		return ""
	}

	var parts []string
	for _, v := range values {
		parts = append(parts, fmt.Sprintf("%v", v.Interface()))
	}

	return strings.Join(parts, ", ")
}

// MarshalFunc 序列化函数签名，返回函数名和参数/返回值类型
func MarshalFunc(fn any) string {
	t := reflect.TypeOf(fn)
	if t.Kind() != reflect.Func {
		return ""
	}

	// 参数
	var params []string
	for i := 0; i < t.NumIn(); i++ {
		params = append(params, t.In(i).String())
	}

	// 返回值
	var returns []string
	for i := 0; i < t.NumOut(); i++ {
		returns = append(returns, t.Out(i).String())
	}

	paramsStr := strings.Join(params, ", ")
	returnsStr := strings.Join(returns, ", ")

	if returnsStr != "" {
		returnsStr = " (" + returnsStr + ")"
	}

	return fmt.Sprintf("(%s)%s", paramsStr, returnsStr)
}

// func MarshalFuncWithArgs(fn any, args ...any) string {
// 	t := reflect.TypeOf(fn)
// 	if t.Kind() != reflect.Func {
// 		panic("not func")
// 	}

// 	// 参数
// 	var params []string
// 	for i := 0; i < t.NumIn(); i++ {
// 		params = append(params, t.In(i).String())
// 	}

// 	// 返回值
// 	var returns []string
// 	for i := 0; i < t.NumOut(); i++ {
// 		returns = append(returns, t.Out(i).String())
// 	}

// 	paramsStr := strings.Join(params, ", ")
// 	returnsStr := strings.Join(returns, ", ")

// 	if returnsStr != "" {
// 		returnsStr = " (" + returnsStr + ")"
// 	}

// 	return fmt.Sprintf("(%s)%s", paramsStr, returnsStr)

// }

func GetFuncName(fn any, short bool) string {
	funcName := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	if short {
		parts := strings.Split(funcName, ".")
		shortName := parts[len(parts)-1]
		return shortName
	}
	return funcName
}
