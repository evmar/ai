package google

import (
	"bufio"
	"bytes"
	"io"
	"testing"
)

func TestStreamedResponse(t *testing.T) {
	raw := `[{
    "candidates": [
      {
        "content": {
          "parts": [
            {
              "text": "That"
            }
          ],
          "role": "model"
        }
      }
    ]
  }
  ,
  {
    "candidates": [
      {
        "content": {
          "parts": [
            {
              "text": "'s a fun question, and the answer is totally subjective! There's"
            }
          ],
          "role": "model"
        }
      }
    ]
  }
  ]
  `

	r := NewStreamedReader(bufio.NewReader(bytes.NewReader([]byte(raw))))
	var resp GenerateContentResponse
	if err := r.Read(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Candidates[0].Content.Parts[0].Text != "That" {
		t.Errorf("expected 'That', got %q", resp.Candidates[0].Content.Parts[0].Text)
	}
	if err := r.Read(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Candidates[0].Content.Parts[0].Text != "'s a fun question, and the answer is totally subjective! There's" {
		t.Errorf("expected second part of streamed response, got %q", resp.Candidates[0].Content.Parts[0].Text)
	}
	if err := r.Read(&resp); err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
}
