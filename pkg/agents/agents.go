package agents

import (
	"fmt"
	"io"
	"iter"
	"log"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/eastlaugh/agent/pkg/openai"
	"github.com/eastlaugh/agent/pkg/util"
)

type tool struct {
	Name        string
	Description string
	Func        any
}

func (t *tool) Run(input string) string {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("工具 [%s] 执行时发生恐慌: %v", t.Name, r)
		}
	}()

	fn := reflect.ValueOf(t.Func)
	typ := fn.Type()

	var args []reflect.Value

	// 处理多参数：使用 fmt.Sscan 解析输入
	values := make([]any, typ.NumIn())
	for i := 0; i < typ.NumIn(); i++ {
		values[i] = reflect.New(typ.In(i)).Interface()
	}

	n, err := fmt.Sscan(input, values...)
	if err != nil || n != typ.NumIn() {
		panic(fmt.Sprintf("参数解析失败: 期望 %d 个参数，得到 %d 个，错误: %v\n", typ.NumIn(), n, err))
	}

	for _, v := range values {
		args = append(args, reflect.ValueOf(v).Elem())
	}

	results := fn.Call(args)
	if len(results) == 0 {
		panic("divergent function")
	}

	return util.MarshalReturn(results)
}

type Agent struct {
	Client   Client
	Tools    map[string]tool
	MaxSteps int
	Prompter func(string) string
}

type Client interface {
	Chat(messages []openai.Message, stop []string) (string, error)
	ChatStream(messages []openai.Message, stop []string) (iter.Seq[string], error)
}

func New(client Client, Prompter func(string) string, args ...any) *Agent {
	var agent = &Agent{
		Client:   client,
		Tools:    make(map[string]tool),
		MaxSteps: 10,
		Prompter: Prompter,
	}

	for i := 0; i < len(args); i += 2 {
		if i+1 >= len(args) {
			panic("agents: args expect Func,  Desc (string)")
		}

		fn := args[i]
		desc, ok := args[i+1].(string)

		if !ok {
			panic("agents: invalid func")
		}

		agent.Add(fn, desc)
	}
	return agent
}

func (a *Agent) SystemPrompt() (prompt string) {
	defer func() {
		if a.Prompter != nil {
			prompt = a.Prompter(prompt)
		}
	}()
	var toolDescriptions strings.Builder
	var toolNames []string

	for name, tool := range a.Tools {
		fmt.Fprintf(&toolDescriptions, "// %s\n%s%s\n ", tool.Description, name, util.MarshalFunc(tool.Func))
		toolNames = append(toolNames, name)
	}
	return fmt.Sprintf(`尽可能回答以下问题。你可以使用以下工具：

%s

使用以下格式：

思考：你应该总是思考该做什么
动作：要采取的动作，应该是 %v 之一
动作输入：动作的输入，对于多个参数以空格隔开，后端将使用 fmt.Sscan 传递给工具，即便函数没有参数，也需要提供空输入
观察：动作的结果
...（这种“思考/动作/动作输入/观察”可以重复多次）
思考：我现在知道最终答案了
最终答案：原始输入问题的最终答案

开始！`, toolDescriptions, toolNames)
}

// 用于提取“动作”和“动作输入”的正则表达式
var actionRegex = regexp.MustCompile(`动作：\s*(.+?)\n动作输入：\s*(.*)`)
var finalAnswerRegex = regexp.MustCompile(`最终答案：\s*(.*)`)

func (a *Agent) Run(w io.Writer, messages []openai.Message, question string) ([]openai.Message, string, error) {
	if len(messages) == 0 {
		messages = []openai.Message{
			{Role: "system", Content: a.SystemPrompt()},
		}
	}

	messages = append(messages, openai.Message{Role: "user", Content: fmt.Sprintf("%s", question)})

	for i := 0; i < a.MaxSteps; i++ {
		fmt.Fprintf(w, "--- 步骤 %d ---\n", i+1)

		response, err := a.Client.Chat(messages, []string{"观察："})
		if err != nil {
			return nil, "", fmt.Errorf("LLM 错误: %v", err)
		}

		fmt.Fprintf(w, "%s\n", response)

		// 将 Agent 的回复添加到历史记录
		messages = append(messages, openai.Message{Role: "assistant", Content: response})

		// 检查最终答案
		if match := finalAnswerRegex.FindStringSubmatch(response); match != nil {
			return messages, strings.TrimSpace(match[1]), nil
		}

		// 解析动作
		match := actionRegex.FindStringSubmatch(response)
		if match == nil {
			panic("没有最终答案，也没有动作")
		}

		toolName := strings.TrimSpace(match[1])
		toolInput := strings.TrimSpace(match[2])

		tool, ok := a.Tools[toolName]
		var observation string
		if !ok {
			observation = fmt.Sprintf("错误：找不到工具 '%s'。可用工具：%v", toolName, a.Tools)
		} else {
			observation = tool.Run(toolInput)
			log.Printf("已执行工具 [%s]，输入为 [%s]", toolName, toolInput)
		}

		// 观察
		obsMsg := fmt.Sprintf("观察：%s", observation)
		fmt.Fprintf(w, "%s\n", obsMsg)
		messages = append(messages, openai.Message{Role: "system", Content: obsMsg})
	}

	return nil, "", fmt.Errorf("达到最大步数仍未找到最终答案")
}

// RunStream runs the agent with streaming output using iterators
func (a *Agent) RunStream(w io.Writer, messages []openai.Message, question string) ([]openai.Message, string, error) {
	if len(messages) == 0 {
		messages = []openai.Message{
			{Role: "system", Content: a.SystemPrompt()},
		}
	}
	messages = append(messages, openai.Message{Role: "user", Content: fmt.Sprintf("%s", question)})

	for i := 0; i < a.MaxSteps; i++ {
		fmt.Fprintf(w, "--- 步骤 %d ---\n", i+1)

		iter, err := a.Client.ChatStream(messages, []string{"观察："})
		if err != nil {
			return nil, "", fmt.Errorf("LLM 错误: %v", err)
		}

		var response strings.Builder
		for chunk := range iter {
			fmt.Fprint(w, chunk)
			response.WriteString(chunk)
		}

		fmt.Fprintf(w, "\n")
		responseText := response.String()

		// 将 Agent 的回复添加到历史记录
		messages = append(messages, openai.Message{Role: "assistant", Content: responseText})

		// 检查最终答案
		if match := finalAnswerRegex.FindStringSubmatch(responseText); match != nil {
			return messages, strings.TrimSpace(match[1]), nil
		}

		// 解析动作
		match := actionRegex.FindStringSubmatch(responseText)
		if match == nil {
			panic("没有最终答案，也没有动作")
		}

		toolName := strings.TrimSpace(match[1])
		toolInput := strings.TrimSpace(match[2])

		tool, ok := a.Tools[toolName]
		var observation string
		if !ok {
			observation = fmt.Sprintf("错误：找不到工具 '%s'。可用工具：%v", toolName, a.Tools)
		} else {
			observation = tool.Run(toolInput)
			log.Printf("已执行工具 [%s]，输入为 [%s]", toolName, toolInput)
		}

		// 观察
		obsMsg := fmt.Sprintf("观察：%s", observation)
		fmt.Fprintf(w, "%s\n", obsMsg)
		messages = append(messages, openai.Message{Role: "system", Content: obsMsg})
	}

	return nil, "", fmt.Errorf("达到最大步数仍未找到最终答案")
}

func (agt *Agent) AsTool() func(string) string {
	return func(input string) string {
		_, result, err := agt.Run(os.Stderr, nil, input)
		if err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return result
	}
}

func (agt *Agent) Add(fn any, desc string) {
	if reflect.TypeOf(fn).Kind() != reflect.Func {
		panic("agents: invalid func")
	}

	var name = util.GetFuncName(fn, false)
	if _, ok := agt.Tools[name]; ok {
		panic("agents: redundant tool definition")
	}
	agt.Tools[name] = tool{
		Name:        name,
		Description: desc,
		Func:        fn,
	}
}
