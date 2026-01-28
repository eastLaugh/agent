package agents

import (
	"fmt"
	"iter"
	"log"
	"reflect"
	"strings"

	"github.com/eastlaugh/agent/pkg/openai"
	"github.com/eastlaugh/agent/pkg/util"
)

type tool struct {
	Name        string
	Description string
	Func        any
}

func (t *tool) Run(input string) (output string) {
	defer func() {
		if r := recover(); r != nil {
			output = fmt.Sprintf("工具 %s 执行时发生恐慌: %v", t.Name, r)
			log.Println(output)
		}
	}()
	defer func() {
		output = strings.TrimSpace(output)
		if output == "" {
			panic("tool returned empty string")
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
	client   Client
	tools    map[string]tool
	maxSteps int
	prompter func(string) string
}

type Client interface {
	Chat(messages []openai.Message, stop []string) (string, error)
	ChatStream(messages []openai.Message, stop []string) (iter.Seq[string], error)
}

func New(client Client, Prompter func(string) string, args ...any) *Agent {
	if Prompter == nil {
		Prompter = func(prompt string) string { return prompt }
	}

	var agent = &Agent{
		client:   client,
		tools:    make(map[string]tool),
		maxSteps: 10,
		prompter: Prompter,
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
		prompt = a.prompter(prompt)
	}()
	var toolDescriptions strings.Builder
	var toolNames []string

	for name, tool := range a.tools {
		fmt.Fprintf(&toolDescriptions, "// %s\n%s%s\n ", tool.Description, name, util.MarshalFunc(tool.Func))
		toolNames = append(toolNames, name)
	}
	return fmt.Sprintf(`你是一个 ReAct Agent，尽可能回答以下问题。你可以使用以下工具：

%s

使用以下格式：

思考：你应该总是思考该做什么
动作：要采取的动作，应该是 %v 之一
动作输入：动作的参数，对于多个参数以空格隔开，后端通过 fmt.Sscan 传递给工具，即便函数没有参数，也需要提供空输入
观察：动作的结果
...（这种“思考/动作/动作输入/观察”可以重复多次）
思考：我现在知道最终答案了
最终答案：原始输入问题的最终答案

开始！`, toolDescriptions.String(), toolNames)
}

// Deprecated: Use RunStreamIter instead.
// func (a *Agent) Run(w io.Writer, messages []openai.Message, question string) ([]openai.Message, string, error) {
// 	if len(messages) == 0 {
// 		messages = []openai.Message{
// 			{Role: "system", Content: a.SystemPrompt()},
// 		}
// 	}

// 	messages = append(messages, openai.Message{Role: "user", Content: fmt.Sprintf("%s", question)})

// 	for i := 0; i < a.MaxSteps; i++ {

// 		response, err := a.Client.Chat(messages, []string{"观察："})
// 		if err != nil {
// 			return nil, "", fmt.Errorf("LLM 错误: %v", err)
// 		}

// 		fmt.Fprintf(w, "%s\n", response)

// 		// 将 Agent 的回复添加到历史记录
// 		messages = append(messages, openai.Message{Role: "assistant", Content: response})

// 		// 检查最终答案
// 		if match := finalAnswerRegex.FindStringSubmatch(response); match != nil {
// 			return messages, strings.TrimSpace(match[1]), nil
// 		}

// 		// 解析动作
// 		match := actionRegex.FindStringSubmatch(response)
// 		if match == nil {
// 			panic("没有最终答案，也没有动作")
// 		}

// 		toolName := strings.TrimSpace(match[1])
// 		toolInput := strings.TrimSpace(match[2])

// 		tool, ok := a.Tools[toolName]
// 		var observation string
// 		if !ok {
// 			observation = fmt.Sprintf("错误：找不到工具 '%s'。可用工具：%v", toolName, a.Tools)
// 		} else {
// 			observation = tool.Run(toolInput)
// 			log.Printf("已执行工具 [%s]，输入为 [%s]", toolName, toolInput)
// 		}

// 		// 观察
// 		obsMsg := fmt.Sprintf("观察：%s", observation)
// 		fmt.Fprintf(w, "%s\n", obsMsg)
// 		messages = append(messages, openai.Message{Role: "system", Content: obsMsg})
// 	}

// 	return nil, "", fmt.Errorf("达到最大步数仍未找到最终答案")
// }

// // Deprecated: use RunStreamIter instead.
// func (a *Agent) RunStream(w io.Writer, messages []openai.Message, question string) ([]openai.Message, string, error) {
// 	if len(messages) == 0 {
// 		messages = []openai.Message{
// 			{Role: "system", Content: a.SystemPrompt()},
// 		}
// 	}
// 	messages = append(messages, openai.Message{Role: "user", Content: fmt.Sprintf("%s", question)})

// 	for i := 0; i < a.MaxSteps; i++ {
// 		fmt.Fprintf(w, "--- 步骤 %d ---\n", i+1)

// 		iter, err := a.Client.ChatStream(messages, []string{"观察："})
// 		if err != nil {
// 			return nil, "", fmt.Errorf("LLM 错误: %v", err)
// 		}

// 		var response strings.Builder
// 		for chunk := range iter {
// 			fmt.Fprint(w, chunk)
// 			response.WriteString(chunk)
// 		}

// 		fmt.Fprintf(w, "\n")
// 		responseText := response.String()

// 		// 将 Agent 的回复添加到历史记录
// 		messages = append(messages, openai.Message{Role: "assistant", Content: responseText})

// 		// 检查最终答案
// 		if match := finalAnswerRegex.FindStringSubmatch(responseText); match != nil {
// 			return messages, strings.TrimSpace(match[1]), nil
// 		}

// 		// 解析动作
// 		match := actionRegex.FindStringSubmatch(responseText)
// 		if match == nil {
// 			panic("没有最终答案，也没有动作")
// 		}

// 		toolName := strings.TrimSpace(match[1])
// 		toolInput := strings.TrimSpace(match[2])

// 		tool, ok := a.Tools[toolName]
// 		var observation string
// 		if !ok {
// 			observation = fmt.Sprintf("错误：找不到工具 '%s'。可用工具：%v", toolName, a.Tools)
// 		} else {
// 			observation = tool.Run(toolInput)
// 			log.Printf("已执行工具 [%s]，输入为 [%s]", toolName, toolInput)
// 		}

// 		// 观察
// 		obsMsg := fmt.Sprintf("观察：%s", observation)
// 		fmt.Fprintf(w, "%s\n", obsMsg)
// 		messages = append(messages, openai.Message{Role: "system", Content: obsMsg})
// 	}

// 	return nil, "", fmt.Errorf("达到最大步数仍未找到最终答案")
// }

func (a *Agent) Iter(messages []openai.Message, question string) (iter.Seq[string], <-chan []openai.Message) {

	if len(messages) == 0 {
		messages = []openai.Message{
			{Role: "system", Content: a.SystemPrompt()},
		}
	}
	messages = append(messages, openai.Message{Role: "user", Content: question})

	var ch = make(chan []openai.Message, 1)
	var consumed bool
	return func(yield func(string) bool) {
		if consumed {
			panic("agents: consumed iterator")
		}
		consumed = true

		// 首次消费迭代器
		defer close(ch)
		for i := 0; i < a.maxSteps; i++ {

			iter, err := a.client.ChatStream(messages, []string{"观察："})
			if err != nil {
				panic(err)
			}

			var response strings.Builder
			for chunk := range iter {
				response.WriteString(chunk)
				if !yield(chunk) {
					return
				}
			}

			Text := response.String()

			// 将 Agent 的回复添加到历史记录
			messages = append(messages, openai.Message{Role: "assistant", Content: Text})

			// 最终答案
			if match := finalAnswerRegex.FindStringSubmatch(Text); match != nil {
				goto FinalAnswer
			}

			// 解析动作
			match := actionRegex.FindStringSubmatch(Text)
			if match == nil {
				messages = append(messages, openai.Message{Role: "system", Content: "你没有遵循ReAct。你没有输出最终答案，也没有输出动作。请严格按照 ReAct 格式进行。上一条消息将被忽略。Continue!"})
				continue
			}

			toolName := strings.TrimSpace(match[1])
			toolInput := strings.TrimSpace(match[2])

			tool, ok := a.tools[toolName]
			var observation string
			if !ok {
				observation = fmt.Sprintf("错误：找不到工具 '%s'。可用工具：%v", toolName, a.tools)
			} else {
				observation = tool.Run(toolInput)
				// log.Printf("已执行工具 [%s]，输入为 [%s]", toolName, toolInput)
			}

			// 观察
			obsMsg := fmt.Sprintf("观察：%s", observation)
			messages = append(messages, openai.Message{Role: "system", Content: obsMsg})
			if !yield(obsMsg + "\n") {
				return
			}
		}

		panic("达到最大步数仍未找到最终答案")
	FinalAnswer:
		ch <- messages
	}, ch

}

func (a *Agent) AsTool() func(string) string {
	return func(input string) string {
		it, _ := a.Iter(nil, input)
		var result strings.Builder
		for chunk := range it {
			result.WriteString(chunk)
		}
		return result.String()
	}
}

func (agt *Agent) Add(fn any, desc string) {
	if reflect.TypeOf(fn).Kind() != reflect.Func {
		panic("agents: invalid func")
	}

	var name = util.GetFuncName(fn, false)
	if _, ok := agt.tools[name]; ok {
		panic("agents: redundant tool definition")
	}
	agt.tools[name] = tool{
		Name:        name,
		Description: desc,
		Func:        fn,
	}
}
