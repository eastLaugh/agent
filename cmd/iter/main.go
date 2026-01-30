package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	"github.com/eastlaugh/agent/pkg/agents"
	"github.com/eastlaugh/agent/pkg/openai"
	"github.com/eastlaugh/agent/pkg/tools"
)

func NewPuzzle() func(num int) string {
	var ans = rand.N(100)
	return func(num int) string {
		if num > ans {
			return "太大了"
		} else if num < ans {
			return "太小了"
		} else {
			return "猜对了"
		}
	}
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

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	go Animation(ctx, 10, "正在启动 Agent 聊天系统")
	time.Sleep(1 * time.Second)
	cancel()

	file, err := os.OpenFile("log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	log.SetOutput(file)

	client := openai.NewClient(os.Getenv("OPENAI_BASE_URL"), os.Getenv("OPENAI_API_KEY"), os.Getenv("OPENAI_MODEL"))

	var agt *agents.Agent
	agt = agents.New(client, nil,
		rand.IntN, "",
		getUserInfo, "用户ID为1到3",
		os.Getenv, "",
		time.Now().Format, "",
		NewPuzzle(), "猜数字游戏",
		tools.SearchInternet, "在互联网上搜索信息，非必要不联网",
		tools.HttpGet, "发送 HTTP GET 请求，非必要不联网",
	)

	fmt.Println("欢迎使用 Agent 聊天系统！CTRL+C 退出。")

	scanner := bufio.NewScanner(os.Stdin)
	var messages []openai.Message
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		var question = scanner.Text()

		if question == "" {
			continue
		}

		iter, ch := agt.Iter(messages, question)
		iter2 := agents.ReactIter(iter)
		for state, chunk := range iter2 {
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
		fmt.Println()
		messages = <-ch
	}

	if err := scanner.Err(); err != nil {
		panic(err)
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
