package openai

import (
	"log"
	"strings"
	"testing"

	"github.com/evmar/ai/rawjson"
)

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
