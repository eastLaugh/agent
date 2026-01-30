package util

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

// CallFunc 通过反射调用函数，input 为空格分隔的参数，返回结果字符串和解析后的参数
func CallFunc(fn any, input string) (output string, argsAny []any) {
	rv := reflect.ValueOf(fn)
	typ := rv.Type()

	values := make([]any, typ.NumIn())
	for i := 0; i < typ.NumIn(); i++ {
		values[i] = reflect.New(typ.In(i)).Interface()
	}

	n, err := fmt.Sscan(input, values...)
	if err != nil || n != typ.NumIn() {
		panic(fmt.Sprintf("参数解析失败: 期望 %d 个参数，得到 %d 个，错误: %v", typ.NumIn(), n, err))
	}

	var args []reflect.Value
	for _, v := range values {
		elem := reflect.ValueOf(v).Elem()
		args = append(args, elem)
		argsAny = append(argsAny, elem.Interface())
	}

	results := rv.Call(args)
	if len(results) == 0 {
		panic("divergent function")
	}
	return MarshalReturn(results), argsAny
}

// FormatToolLog 格式化工具调用日志
func FormatToolLog(name, desc string, fn any, args []any, output string) string {
	var lines []string
	if desc != "" {
		lines = append(lines, "// "+desc)
	}
	lines = append(lines, name+MarshalFunc(fn))
	lines = append(lines, fmt.Sprintf("%s = %s", output, MarshalFuncCall(fn, args...)))
	return strings.Join(lines, "\n")
}

func marshalReturn(values []reflect.Value) string {
	var parts []string
	for _, v := range values {
		parts = append(parts, fmt.Sprintf("%v", v.Interface()))
	}
	return strings.Join(parts, ", ")
}

// MarshalReturn 将 []reflect.Value 序列化为字符串
func MarshalReturn(values []reflect.Value) string {
	if len(values) == 0 {
		return ""
	}
	return marshalReturn(values)
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

// MarshalFuncCall 将函数调用序列化为 Go 风格字符串，如 math/rand/v2.IntN(100)
func MarshalFuncCall(fn any, args ...any) string {
	name := GetFuncName(fn, false)
	var parts []string
	for _, arg := range args {
		parts = append(parts, fmt.Sprintf("%#v", arg))
	}
	return fmt.Sprintf("%s(%s)", name, strings.Join(parts, ", "))
}

func GetFuncName(fn any, short bool) string {
	funcName := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	if short {
		parts := strings.Split(funcName, ".")
		shortName := parts[len(parts)-1]
		return shortName
	}
	return funcName
}
