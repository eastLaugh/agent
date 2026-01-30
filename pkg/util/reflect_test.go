package util_test

import (
	"testing"

	"github.com/eastlaugh/agent/pkg/util"
)

func TestMarshalFunc(t *testing.T) {
	res := util.MarshalFunc(TestMarshalFunc)
	println(res)
	res = util.MarshalFunc(util.MarshalFunc)
	println(res)
}

func hello(a int, b int, c int) bool { return true }

func TestMarshalFuncCall(t *testing.T) {
	res := util.MarshalFuncCall(hello, 1, 2, 3)
	println(res) // github.com/eastlaugh/agent/pkg/util_test.hello(1, 2, 3)

	res = util.MarshalFuncCall(util.GetFuncName, "test", true)
	println(res) // github.com/eastlaugh/agent/pkg/util.GetFuncName("test", true)
}
