package agents

import (
	"fmt"
	"testing"
)

func Test(t *testing.T) {
	agt := New(nil,
		func(a int, b int, str string) (int, string) {
			return a + b, str
		}, "函数描述",
		func() {
			//
		}, "哈哈",
		fmt.Print, "打印输入内容",
	)
	println(agt.generateSystemPrompt())

}
