package llm

import "github.com/evmar/ai/image"

// TODO: unify streaming with Call interface

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
	Images   []*image.LoadedImage
}
