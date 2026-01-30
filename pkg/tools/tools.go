package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/eastlaugh/agent/pkg/agents"
)

// HttpGet 发送 HTTP GET 请求，带 10 秒超时，自动关闭响应体，返回响应体字符串或错误信息
func HttpGet(u string) string {
	// 基础 URL 安全校验（防 SSRF）
	parsed, err := url.Parse(u)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Sprintf("error: invalid URL %q", u)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Sprintf("error: unsupported scheme %q", parsed.Scheme)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return fmt.Sprintf("error: failed to build request: %v", err)
	}
	// 设置通用 User-Agent，避免被简单拦截
	req.Header.Set("User-Agent", "agent-cli/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("error: request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Sprintf("error: HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("error: failed to read body: %v", err)
	}

	return string(body)
}
func SearchInternet(query string) string {
	// URL 编码查询词
	encoded := url.QueryEscape(query)
	u := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", encoded)

	// 复用 HttpGet（保持逻辑复用与统一错误格式）
	return HttpGet(u)
}

// Human-in-the-Loop，将逻辑暴露给前端
func HumanInTheLoop(agt *agents.Agent, dialogue func() bool) func(string) bool {
	return func(input string) bool {
		return dialogue()
	}
}

// 目前还没想好怎么做 Langchain 的 Middleware，先放个雏形
type Middleware[T, R any] func(T) R

func chain[T, R any](next Middleware[T, R]) Middleware[T, R] {

	return func(input T) R {
		// 在这里可以添加前置逻辑
		result := next(input)
		// 在这里可以添加后置逻辑
		return result
	}
}
