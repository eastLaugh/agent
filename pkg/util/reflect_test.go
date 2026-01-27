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
