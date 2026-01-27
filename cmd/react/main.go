package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"time"

	"github.com/eastlaugh/agent/pkg/agents"
	"github.com/eastlaugh/agent/pkg/llm"
)

// --- Helper Functions ---

func getTime() string {
	return time.Now().Format(time.RFC1123)
}

func getRandom() int {
	return rand.N(100)
}

// 多参数函数：计算两个数的和
func add(a int, b int) int {
	return a + b
}

// 多参数函数：计算两个数的乘积
func multiply(a int, b int) int {
	return a * b
}

// 多参数函数：字符串拼接
func concat(s1 string, s2 string) string {
	return s1 + s2
}

// 多参数函数：计算字符串长度
func strlen(s string) int {
	return len(s)
}

// 查询用户信息
func getUserInfo(userID int) string {
	userDB := map[int]string{
		1: "Alice (age: 28, city: Beijing)",
		2: "Bob (age: 32, city: Shanghai)",
		3: "Charlie (age: 25, city: Shenzhen)",
	}
	if info, ok := userDB[userID]; ok {
		return info
	}
	return fmt.Sprintf("User %d not found", userID)
}

// 计算数字的平方
func square(n int) int {
	return n * n
}

// --- Main ---

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("请设置 OPENAI_API_KEY 环境变量。")
	}
	baseURL := os.Getenv("OPENAI_BASE_URL")

	// 1. 初始化客户端
	client := llm.NewClient(baseURL, apiKey, "qwen-plus")

	mathProficientAgent := agents.New(client,
		add, "计算两个整数的和。参数：a, b（整数，用空格分隔）。",
		multiply, "计算两个整数的乘积。参数：a, b（整数，用空格分隔）。",
		square, "计算一个整数的平方。参数：n（整数）。",
	)

	// 2. 初始化 Agent 并注册多个工具
	myAgent := agents.New(client,
		getTime, "返回服务器当前的系统时间（RFC1123格式）。",
		getRandom, "返回 0-100 之间的随机整数。",
		add, "计算两个整数的和。参数：a, b（整数，用空格分隔）。",
		multiply, "计算两个整数的乘积。参数：a, b（整数，用空格分隔）。",
		concat, "拼接两个字符串。参数：s1, s2（字符串，用空格分隔）。",
		strlen, "计算字符串的长度。参数：s（字符串）。",
		getUserInfo, "查询用户信息。参数：userID（1-3）。",
		mathProficientAgent.AsTool(), "数学专家 Agent，擅长执行复杂的数学计算任务。计算数学问题时请调用它。",
	)

	// 3. 运行多个复杂问题来测试框架

	answer, err := myAgent.Run(os.Stdout, "请告诉我用户 2 的信息，并计算他的年龄加上 5 的平方是多少？然后告诉我现在的时间。")
	if err != nil {
		fmt.Printf("错误：%v\n", err)
	}
	fmt.Printf("%s\n", answer)

}
