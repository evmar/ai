package google

// The underlying API appears to be protobufs, and the official API uses them.
// Using JSON here just avoids pulling in protobuf code.

type GenerateContentResponse struct {
	Candidates []*Candidate `json:"candidates"`
	// PromptFeedback *PromptFeedback `json:"promptFeedback"`
	// UsageMetadata *UsageMetadata `json:"usageMetadata"`
}

type Candidate struct {
	Content      *Content `json:"content"`
	FinishReason string   `json:"finishReason"`
	// ... more fields
}

type Content struct {
	Parts []*Part `json:"parts"`
	Role  string  `json:"role"`
}

type Part struct {
	Text string `json:"text"`
}
