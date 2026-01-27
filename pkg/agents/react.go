package agents

import (
	"bufio"
	"io"
	"iter"
	"regexp"
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

// 用于把纯文本迭代器转换为 React 风格的迭代器
func ReactIter(it iter.Seq[string]) iter.Seq2[ReAct, string] {
	//暂时用比较丑陋的方式实现

	r, w := io.Pipe()
	go func() {
		for chunk := range it {
			io.WriteString(w, chunk)
		}
		w.Close()
	}()
	var rd = bufio.NewReader(r)
	return func(yield func(ReAct, string) bool) {
		for {
			tmp, err := rd.ReadString('\n')
			if err == io.EOF {
				return
			}
			if err != nil {
				panic(err)
			}
			if !yield(Thinking, tmp) {
				return
			}
		}
	}

}
