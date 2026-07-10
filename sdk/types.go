package sdk

// Message is a single conversation turn in the OpenAI Chat Completions
// shape. Content may be a plain string or an array of content blocks —
// callers that only send text should leave it as string.
type Message struct {
	Role       string         `json:"role"`
	Content    any            `json:"content,omitempty"`
	Name       string         `json:"name,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall     `json:"tool_calls,omitempty"`
	Refusal    string         `json:"refusal,omitempty"`
	Additional map[string]any `json:"-"`
}

// ToolCall mirrors OpenAI's tool_calls entry. Type is always "function"
// for now — cursor-proxy does not implement other tool kinds.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction is the payload of a function-type ToolCall.
type ToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Tool describes one function the model is allowed to call.
type Tool struct {
	Type     string          `json:"type"`
	Function *ToolDefinition `json:"function,omitempty"`
}

// ToolDefinition is the metadata for a callable function.
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// ChatRequest is the input to ChatCompletion / ChatCompletionStream.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Tools       []Tool    `json:"tools,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
	TopP        *float64  `json:"top_p,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	Stop        any       `json:"stop,omitempty"`
	// User is a caller-supplied end-user ID passed through to the
	// backend; cursor-proxy does not act on it but downstream logging
	// may.
	User string `json:"user,omitempty"`
	// Extra fields are marshalled inline for future compatibility.
	Extra map[string]any `json:"-"`
}

// ChatResponse is the non-streaming Chat Completions response.
type ChatResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []ChatChoice `json:"choices"`
	Usage   *Usage       `json:"usage,omitempty"`
}

// ChatChoice is one candidate in the response.
type ChatChoice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason,omitempty"`
}

// ChatStreamChunk is one SSE event from ChatCompletionStream.
type ChatStreamChunk struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []ChatStreamChoice `json:"choices"`
	Usage   *Usage             `json:"usage,omitempty"`
}

// ChatStreamChoice is one delta in a streaming response.
type ChatStreamChoice struct {
	Index        int       `json:"index"`
	Delta        ChatDelta `json:"delta"`
	FinishReason string    `json:"finish_reason,omitempty"`
}

// ChatDelta is the incremental content in a stream chunk.
type ChatDelta struct {
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// Usage is the OpenAI-shape token accounting.
type Usage struct {
	PromptTokens            int                      `json:"prompt_tokens"`
	CompletionTokens        int                      `json:"completion_tokens"`
	TotalTokens             int                      `json:"total_tokens"`
	PromptTokensDetails     *PromptTokensDetails     `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
}

// PromptTokensDetails breaks the prompt count into cached / uncached.
type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

// CompletionTokensDetails breaks the completion count into
// reasoning-vs-text (mirrors OpenAI's o-series shape).
type CompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}

// Model is one entry in the /v1/models response.
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created,omitempty"`
	OwnedBy string `json:"owned_by,omitempty"`
}

// ModelList is the shape of /v1/models.
type ModelList struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// AnthropicRequest is the input to AnthropicMessages / stream.
type AnthropicRequest struct {
	Model     string         `json:"model"`
	MaxTokens int            `json:"max_tokens"`
	Messages  []AnthropicMsg `json:"messages"`
	System    any            `json:"system,omitempty"`
	Stream    bool           `json:"stream,omitempty"`
	Extra     map[string]any `json:"-"`
}

// AnthropicMsg mirrors an Anthropic messages entry. Content is either
// a string or a []ContentBlock.
type AnthropicMsg struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// AnthropicResponse is the non-streaming Anthropic Messages response.
type AnthropicResponse struct {
	ID           string          `json:"id"`
	Type         string          `json:"type"`
	Role         string          `json:"role"`
	Model        string          `json:"model"`
	Content      []ContentBlock  `json:"content"`
	StopReason   string          `json:"stop_reason"`
	StopSequence *string         `json:"stop_sequence"`
	Usage        *AnthropicUsage `json:"usage,omitempty"`
}

// ContentBlock is one block inside an Anthropic message body.
type ContentBlock struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Input any    `json:"input,omitempty"`
}

// AnthropicUsage is Anthropic-shape token accounting.
type AnthropicUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// CountTokensRequest is the input to CountTokens.
type CountTokensRequest struct {
	Model    string         `json:"model"`
	Messages []AnthropicMsg `json:"messages"`
	System   any            `json:"system,omitempty"`
}

// CountTokensResponse is the output.
type CountTokensResponse struct {
	InputTokens int `json:"input_tokens"`
}
