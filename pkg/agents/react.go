package agents

import (
	"iter"
	"regexp"
	"strings"
	"unicode/utf8"
)

// 用于提取“动作”和“动作输入”的正则表达式
var actionRegex = regexp.MustCompile(`动作：\s*(.+?)\n动作输入：\s*(.*)`)
var finalAnswerRegex = regexp.MustCompile(`最终答案：\s*(.*)`)

type ReAct uint8

const (
	Thinking ReAct = 1 << iota
	Acting
	Observing
	Answering
)

func (r ReAct) String() string {
	switch r {
	case Thinking:
		return "thinking"
	case Acting:
		return "acting"
	case Observing:
		return "observing"
	case Answering:
		return "answering"
	default:
		return "unknown"
	}
}

// 用于把纯文本迭代器转换为 React 风格的迭代器
func ReactIter(it iter.Seq[string]) iter.Seq2[ReAct, string] {
	return func(yield func(ReAct, string) bool) {
		var currentState ReAct = Thinking
		var buffer string

		markers := []struct {
			word  string
			state ReAct
		}{
			{"动作输入：", Acting},
			{"最终答案：", Answering},
			{"思考：", Thinking},
			{"动作：", Acting},
			{"观察：", Observing},
		}

		for chunk := range it {
			buffer += chunk

			for {
				earliestIdx := -1
				mLen := 0
				var nextState ReAct

				for _, m := range markers {
					if idx := strings.Index(buffer, m.word); idx != -1 {
						if earliestIdx == -1 || idx < earliestIdx {
							earliestIdx = idx
							mLen = len(m.word)
							nextState = m.state
						}
					}
				}

				if earliestIdx == -1 {
					// 【关键修正】确保切在 UTF-8 字符边界上！
					if len(buffer) > 30 {
						// 留出足够的空间（20字节），确保不会切断任何标记词或中文字符
						// 我们只吐出前面确定安全的部分
						safeCut := len(buffer) - 20

						// 寻找最近的一个合法 UTF-8 字符边界
						for !utf8.ValidString(buffer[:safeCut]) && safeCut > 0 {
							safeCut--
						}

						if safeCut > 0 {
							if !yield(currentState, buffer[:safeCut]) {
								return
							}
							buffer = buffer[safeCut:]
						}
					}
					break
				}

				// 1. 标识符之前的内容吐出去
				if earliestIdx > 0 {
					if !yield(currentState, buffer[:earliestIdx]) {
						return
					}
				}

				// 2. 剥离标识符，切换状态
				currentState = nextState
				buffer = buffer[earliestIdx+mLen:]
			}
		}

		if len(buffer) > 0 {
			yield(currentState, buffer)
		}
	}
}
