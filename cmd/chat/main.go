package main

import (
	"bufio"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	"github.com/eastlaugh/agent/pkg/agents"
	"github.com/eastlaugh/agent/pkg/openai"
)

// --- Helper Functions ---

func getTime() string {
	return time.Now().Format(time.RFC1123)
}

func getRandom() int {
	return rand.N(100)
}

func add(a int, b int) int {
	return a + b
}

func multiply(a int, b int) int {
	return a * b
}

func concat(s1 string, s2 string) string {
	return s1 + s2
}

func strlen(s string) int {
	return len(s)
}

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

func square(n int) int {
	return n * n
}

func main() {

	client := openai.NewClient(os.Getenv("OPENAI_BASE_URL"), os.Getenv("OPENAI_API_KEY"), os.Getenv("OPENAI_MODEL"))

	myAgent := agents.New(client, nil,
		getTime, "返回服务器当前的系统时间（RFC1123格式）。",
		getRandom, "返回 0-100 之间的随机整数。",
		add, "计算两个整数的和。参数：a, b（整数，用空格分隔）。",
		multiply, "计算两个整数的乘积。参数：a, b（整数，用空格分隔）。",
		concat, "拼接两个字符串。参数：s1, s2（字符串，用空格分隔）。",
		strlen, "计算字符串的长度。参数：s（字符串）。",
		getUserInfo, "查询用户信息。参数：userID（1-3）。",
		square, "计算一个整数的平方。参数：n（整数）。",
		// tools.SearchInternet, "在互联网上搜索信息",
		// tools.HttpGet, "发送 HTTP GET 请求",
	)

	fmt.Println("欢迎使用 Agent 聊天系统！CTRL+C 退出。")

	reader := bufio.NewReader(os.Stdin)
	var messages []openai.Message
	for {
		question, _ := reader.ReadString('\n')
		question = strings.TrimSpace(question)
		if question == "" {
			continue
		}

		iter, ch := myAgent.Iter(messages, question)
		for chunk := range iter {
			fmt.Print(chunk)
		}
		messages = <-ch
		fmt.Println()
	}
}
