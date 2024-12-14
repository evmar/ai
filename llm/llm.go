package llm

type Stream interface {
	Next() (string, error)
}

type Streamed interface {
	CallStreamed(sys string, json bool, prompts []string) (Stream, error)
}
