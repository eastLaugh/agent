package main

import (
	"bufio"
	"context"
	"fmt"
	"math"
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

// --- Main ---

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	go Animation(ctx, 10, "正在启动 Agent 聊天系统")
	time.Sleep(1 * time.Second)
	cancel()

	apiKey := os.Getenv("OPENAI_API_KEY")
	baseURL := os.Getenv("OPENAI_BASE_URL")
	client := openai.NewClient(baseURL, apiKey, "qwen-plus")

	myAgent := agents.New(client, nil,
		getTime, "返回服务器当前的系统时间（RFC1123格式）。",
		getRandom, "返回 0-100 之间的随机整数。",
		add, "计算两个整数的和。参数：a, b（整数，用空格分隔）。",
		multiply, "计算两个整数的乘积。参数：a, b（整数，用空格分隔）。",
		concat, "拼接两个字符串。参数：s1, s2（字符串，用空格分隔）。",
		strlen, "计算字符串的长度。参数：s（字符串）。",
		getUserInfo, "查询用户信息。参数：userID（1-3）。",
		square, "计算一个整数的平方。参数：n（整数）。",
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

		// 运行 Agent 并输出结果

		iter, _, err := myAgent.RunStreamIter(messages, question)
		if err != nil {
			panic(err)
		}

		{
			iter := agents.ReactIter(iter)
			for state, chunk := range iter {
				switch state {
				case agents.Thinking:
					fmt.Print(Gray(chunk))
				case agents.Acting:
					fmt.Print(Blue(chunk))
				case agents.Observing:
					fmt.Print(Red(chunk))
				case agents.Answering:
					fmt.Print(chunk)
				default:
					panic(state)
				}
			}
		}

		fmt.Println("\n" + strings.Repeat("-", 50))
	}
}

func Animation(ctx context.Context, maxDots float64, tooltip string) {
	var tk = time.NewTicker(100 * time.Millisecond)
	defer tk.Stop()
	for {

		select {
		case <-ctx.Done():
			fmt.Print("\r\033[K") // 清除动画行
			return
		case <-tk.C:
			// \r 回到行首，\033[K 是清除从光标到行末的内容，防止残留
			y := math.Sin(float64(time.Now().UnixNano()) / 1e9 * 2 * math.Pi)
			y++
			fmt.Printf("\r%s %s\033[K", tooltip, strings.Repeat(".", int(maxDots*y)))
		}

	}
}

func Red(input string) string {
	return fmt.Sprintf("\033[31m%s\033[0m", input)
}

func Green(input string) string {
	return fmt.Sprintf("\033[32m%s\033[0m", input)
}

func Blue(input string) string {
	return fmt.Sprintf("\033[34m%s\033[0m", input)
}

func Gray(input string) string {
	return fmt.Sprintf("\033[90m%s\033[0m", input)
}
