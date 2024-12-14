package google

import (
	"encoding/json"
	"fmt"
	"io"
)

type state int

const (
	Initial state = iota
	Reading
	Finished
)

type StreamedReader struct {
	dec   *json.Decoder
	state state
}

func NewStreamedReader(r io.Reader) *StreamedReader {
	return &StreamedReader{dec: json.NewDecoder(r)}
}

func (s *StreamedReader) Read(v any) error {
	if s.state == Initial {
		t, err := s.dec.Token()
		if err != nil {
			return err
		}
		r := t.(json.Delim)
		if r != '[' {
			return fmt.Errorf("expected '[', got %q", t)
		}
		s.state = Reading
	}

	if s.state == Reading {
		if s.dec.More() {
			return s.dec.Decode(v)
		}
		s.state = Finished
	}

	return io.EOF
}
