package openai

import (
	"log"
	"strings"
	"testing"

	"github.com/evmar/ai/rawjson"
)

func TestOk(t *testing.T) {
	responseText := `{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "created": 1736971375,
  "model": "gpt-3.5-turbo-0125",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! How can I assist you today?",
        "refusal": null
      },
      "logprobs": null,
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 13,
    "completion_tokens": 10,
    "total_tokens": 23,
    "prompt_tokens_details": {
      "cached_tokens": 0,
      "audio_tokens": 0
    },
    "completion_tokens_details": {
      "reasoning_tokens": 0,
      "audio_tokens": 0,
      "accepted_prediction_tokens": 0,
      "rejected_prediction_tokens": 0
    }
  },
  "service_tier": "default",
  "system_fingerprint": null
}`
	msg, err := parse([]byte(responseText))
	if err != nil {
		t.Fatal(err)
	}
	if msg != "Hello! How can I assist you today?" {
		t.Fatalf("wanted Hello! How can I assist you today?, got %q", msg)
	}
}

func TestQuotaError(t *testing.T) {
	quotaErrorText := `{
		"error": {
			"message": "You exceeded your current quota, please check your plan and billing details. For more information on this error, read the docs: https://platform.openai.com/docs/guides/error-codes/api-errors.",
			"type": "insufficient_quota",
			"param": null,
			"code": "insufficient_quota"
		}
	}`

	j, err := rawjson.Parse([]byte(quotaErrorText))
	if err != nil {
		t.Fatal(err)
	}

	{
		err := getError(j)
		if err == nil {
			t.Fatalf("wanted err, got %v", err)
		}
		exp := "openai: You exceeded your current quota"
		if !strings.HasPrefix(err.Error(), exp) {
			t.Fatalf("wanted prefix %q, got %q", exp, err.Error())
		}
		log.Println(err)
	}
}
