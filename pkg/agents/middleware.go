package agents

type Hanlder interface {
	Serve(string) string
}

type HandleFunc func(string) string

func (m HandleFunc) Serve(input string) string {
	return m(input)
}

var BaseHandler = HandleFunc(func(input string) string {
	return input
})
