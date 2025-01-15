package llm

type Stream interface {
	Next() (string, error)
}

type Streamed interface {
	CallStreamed(sys string, json bool, msgs []string) (Stream, error)
}

type Message interface{}

type Prompt struct {
	System   string
	JSON     bool
	Messages []string
}
