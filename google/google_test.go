package google

import "testing"

func TestResponse(t *testing.T) {
	resp := `{
  "candidates": [
    {
      "content": {
        "parts": [
          {
            "text": "There's no single \"best\" day of the week"
          }
        ],
        "role": "model"
      },
      "finishReason": "STOP",
      "avgLogprobs": -0.3246549891964825
    }
  ],
  "usageMetadata": {
    "promptTokenCount": 7,
    "candidatesTokenCount": 207,
    "totalTokenCount": 214
  },
  "modelVersion": "gemini-1.5-flash"
}`
	out, err := parseText([]byte(resp))
	if err != nil {
		t.Fatal(err)
	}
	exp := "There's no single \"best\" day of the week"
	if out != exp {
		t.Fatalf("wanted %q, got %q", exp, out)
	}
}
