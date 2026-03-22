package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// mockHTTPClient is a mock HTTP client for testing.
type mockHTTPClient struct {
	response *http.Response
	err      error
	doFunc   func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.doFunc != nil {
		return m.doFunc(req)
	}
	return m.response, m.err
}

// newMockResponse creates a mock HTTP response with the given body and status.
func newMockResponse(body string, statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestNewAnthropicProvider(t *testing.T) {
	tests := []struct {
		name    string
		opts    []ProviderOption
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid provider with API key",
			opts:    []ProviderOption{WithAPIKey("test-api-key")},
			wantErr: false,
		},
		{
			name:    "missing API key",
			opts:    []ProviderOption{},
			wantErr: true,
			errMsg:  "API key is required",
		},
		{
			name: "with custom base URL",
			opts: []ProviderOption{
				WithAPIKey("test-api-key"),
				WithBaseURL("https://custom.anthropic.com"),
			},
			wantErr: false,
		},
		{
			name: "with custom model",
			opts: []ProviderOption{
				WithAPIKey("test-api-key"),
				WithModel(Claude3Opus),
			},
			wantErr: false,
		},
		{
			name: "with custom HTTP client",
			opts: []ProviderOption{
				WithAPIKey("test-api-key"),
				WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewAnthropicProvider(tt.opts...)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewAnthropicProvider() error = nil, wantErr = true")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("NewAnthropicProvider() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("NewAnthropicProvider() error = %v, wantErr = false", err)
				return
			}
			if provider == nil {
				t.Error("NewAnthropicProvider() returned nil provider")
			}
		})
	}
}

func TestAnthropicProvider_CreateMessage(t *testing.T) {
	tests := []struct {
		name       string
		response   *http.Response
		err        error
		req        MessagesRequest
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "successful response",
			response: newMockResponse(`{
				"id": "msg_123",
				"type": "message",
				"role": "assistant",
				"model": "claude-3-5-sonnet-20241022",
				"content": [{"type": "text", "text": "Hello!"}],
				"stop_reason": "end_turn",
				"usage": {"input_tokens": 10, "output_tokens": 5}
			}`, http.StatusOK),
			req:     MessagesRequest{MaxTokens: 100, Messages: []Message{CreateTextMessage(RoleUser, "Hi")}},
			wantErr: false,
		},
		{
			name:       "HTTP client error",
			response:   nil,
			err:        errors.New("connection refused"),
			req:        MessagesRequest{MaxTokens: 100},
			wantErr:    true,
			wantErrMsg: "failed to send request",
		},
		{
			name:       "API error response",
			response:   newMockResponse(`{"error": "invalid request"}`, http.StatusBadRequest),
			req:        MessagesRequest{MaxTokens: 100},
			wantErr:    true,
			wantErrMsg: "API request failed with status 400",
		},
		{
			name:       "invalid JSON response",
			response:   newMockResponse(`invalid json`, http.StatusOK),
			req:        MessagesRequest{MaxTokens: 100},
			wantErr:    true,
			wantErrMsg: "failed to decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockHTTPClient{
				response: tt.response,
				err:      tt.err,
			}

			provider, err := NewAnthropicProvider(
				WithAPIKey("test-api-key"),
				WithHTTPClient(mockClient),
			)
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}

			ctx := context.Background()
			resp, err := provider.CreateMessage(ctx, tt.req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateMessage() error = nil, wantErr = true")
					return
				}
				if tt.wantErrMsg != "" && !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("CreateMessage() error = %v, want containing %v", err.Error(), tt.wantErrMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("CreateMessage() error = %v, wantErr = false", err)
				return
			}
			if resp == nil {
				t.Error("CreateMessage() returned nil response")
				return
			}

			// Verify token tracking
			tracker := provider.GetTokenTracker()
			if tracker.TotalRequests != 1 {
				t.Errorf("Expected 1 request, got %d", tracker.TotalRequests)
			}
		})
	}
}

func TestAnthropicProvider_CreateMessageStream(t *testing.T) {
	tests := []struct {
		name       string
		streamBody string
		statusCode int
		wantEvents int
		wantErr    bool
	}{
		{
			name: "successful streaming response",
			streamBody: `data: {"type": "message_start", "message": {"id": "msg_123", "role": "assistant", "model": "claude-3-5-sonnet-20241022"}}
data: {"type": "content_block_start", "index": 0, "content_block": {"type": "text", "text": ""}}
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "text": "Hello"}}
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "text_delta", "text": " world"}}
data: {"type": "content_block_stop", "index": 0}
data: {"type": "message_stop", "message": {"usage": {"input_tokens": 10, "output_tokens": 5}}}
data: [DONE]`,
			statusCode: http.StatusOK,
			wantEvents: 6,
			wantErr:    false,
		},
		{
			name:       "API error response",
			streamBody: `{"error": "rate limit exceeded"}`,
			statusCode: http.StatusTooManyRequests,
			wantEvents: 0,
			wantErr:    true,
		},
		{
			name: "stream with thinking blocks",
			streamBody: `data: {"type": "message_start", "message": {"id": "msg_123", "role": "assistant"}}
data: {"type": "content_block_start", "index": 0, "content_block": {"type": "thinking", "thinking": ""}}
data: {"type": "content_block_delta", "index": 0, "delta": {"type": "thinking_delta", "thinking": "Let me think..."}}
data: {"type": "content_block_stop", "index": 0}
data: {"type": "content_block_start", "index": 1, "content_block": {"type": "text", "text": ""}}
data: {"type": "content_block_delta", "index": 1, "delta": {"type": "text_delta", "text": "Result"}}
data: {"type": "content_block_stop", "index": 1}
data: {"type": "message_stop"}
data: [DONE]`,
			statusCode: http.StatusOK,
			wantEvents: 8,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockHTTPClient{
				response: newMockResponse(tt.streamBody, tt.statusCode),
			}

			provider, err := NewAnthropicProvider(
				WithAPIKey("test-api-key"),
				WithHTTPClient(mockClient),
			)
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}

			ctx := context.Background()
			eventChan, errChan := provider.CreateMessageStream(ctx, MessagesRequest{
				MaxTokens: 100,
				Messages:  []Message{CreateTextMessage(RoleUser, "Test")},
			})

			eventCount := 0
			done := make(chan bool)

			go func() {
				for range eventChan {
					eventCount++
				}
				done <- true
			}()

			var streamErr error
			go func() {
				for err := range errChan {
					if err != nil {
						streamErr = err
					}
				}
			}()

			<-done

			if tt.wantErr {
				if streamErr == nil {
					t.Error("Expected error from stream, got nil")
				}
				return
			}
			if streamErr != nil {
				t.Errorf("Unexpected error from stream: %v", streamErr)
			}
			if eventCount != tt.wantEvents {
				t.Errorf("Expected %d events, got %d", tt.wantEvents, eventCount)
			}
		})
	}
}

func TestAnthropicProvider_SetModel(t *testing.T) {
	provider, err := NewAnthropicProvider(WithAPIKey("test-api-key"))
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Test default model
	if provider.GetModel() != Claude35Sonnet {
		t.Errorf("Expected default model %s, got %s", Claude35Sonnet, provider.GetModel())
	}

	// Test setting model
	provider.SetModel(Claude3Opus)
	if provider.GetModel() != Claude3Opus {
		t.Errorf("Expected model %s, got %s", Claude3Opus, provider.GetModel())
	}
}

func TestTokenTracker(t *testing.T) {
	tracker := &TokenTracker{}

	// Test initial state
	if tracker.TotalTokens() != 0 {
		t.Errorf("Expected 0 total tokens, got %d", tracker.TotalTokens())
	}

	// Test update
	tracker.Update(Usage{InputTokens: 100, OutputTokens: 50})
	if tracker.TotalInputTokens != 100 {
		t.Errorf("Expected 100 input tokens, got %d", tracker.TotalInputTokens)
	}
	if tracker.TotalOutputTokens != 50 {
		t.Errorf("Expected 50 output tokens, got %d", tracker.TotalOutputTokens)
	}
	if tracker.TotalRequests != 1 {
		t.Errorf("Expected 1 request, got %d", tracker.TotalRequests)
	}
	if tracker.TotalTokens() != 150 {
		t.Errorf("Expected 150 total tokens, got %d", tracker.TotalTokens())
	}

	// Test multiple updates
	tracker.Update(Usage{InputTokens: 200, OutputTokens: 100})
	if tracker.TotalRequests != 2 {
		t.Errorf("Expected 2 requests, got %d", tracker.TotalRequests)
	}
	if tracker.TotalTokens() != 450 {
		t.Errorf("Expected 450 total tokens, got %d", tracker.TotalTokens())
	}

	// Test reset
	tracker.Reset()
	if tracker.TotalTokens() != 0 {
		t.Errorf("Expected 0 total tokens after reset, got %d", tracker.TotalTokens())
	}
	if tracker.TotalRequests != 0 {
		t.Errorf("Expected 0 requests after reset, got %d", tracker.TotalRequests)
	}
}

func TestExtractTextContent(t *testing.T) {
	tests := []struct {
		name     string
		response *MessagesResponse
		want     string
	}{
		{
			name: "single text block",
			response: &MessagesResponse{
				Content: []ContentBlock{
					{Type: ContentTypeText, Text: "Hello world"},
				},
			},
			want: "Hello world",
		},
		{
			name: "multiple text blocks",
			response: &MessagesResponse{
				Content: []ContentBlock{
					{Type: ContentTypeText, Text: "Hello"},
					{Type: ContentTypeText, Text: "world"},
				},
			},
			want: "Helloworld",
		},
		{
			name: "mixed content blocks",
			response: &MessagesResponse{
				Content: []ContentBlock{
					{Type: ContentTypeText, Text: "Hello"},
					{Type: ContentTypeThinking, Thinking: "thinking..."},
					{Type: ContentTypeText, Text: "world"},
				},
			},
			want: "Helloworld",
		},
		{
			name:     "empty content",
			response: &MessagesResponse{Content: []ContentBlock{}},
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTextContent(tt.response)
			if got != tt.want {
				t.Errorf("ExtractTextContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractThinkingContent(t *testing.T) {
	tests := []struct {
		name     string
		response *MessagesResponse
		want     string
	}{
		{
			name: "single thinking block",
			response: &MessagesResponse{
				Content: []ContentBlock{
					{Type: ContentTypeThinking, Thinking: "I need to analyze this..."},
				},
			},
			want: "I need to analyze this...",
		},
		{
			name: "multiple thinking blocks",
			response: &MessagesResponse{
				Content: []ContentBlock{
					{Type: ContentTypeThinking, Thinking: "First thought"},
					{Type: ContentTypeThinking, Thinking: "Second thought"},
				},
			},
			want: "First thoughtSecond thought",
		},
		{
			name: "mixed content blocks",
			response: &MessagesResponse{
				Content: []ContentBlock{
					{Type: ContentTypeText, Text: "Hello"},
					{Type: ContentTypeThinking, Thinking: "thinking..."},
					{Type: ContentTypeText, Text: "world"},
				},
			},
			want: "thinking...",
		},
		{
			name: "no thinking blocks",
			response: &MessagesResponse{
				Content: []ContentBlock{
					{Type: ContentTypeText, Text: "Just text"},
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractThinkingContent(tt.response)
			if got != tt.want {
				t.Errorf("ExtractThinkingContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCreateTextMessage(t *testing.T) {
	msg := CreateTextMessage(RoleUser, "Hello")

	if msg.Role != RoleUser {
		t.Errorf("Expected role %s, got %s", RoleUser, msg.Role)
	}
	if len(msg.Content) != 1 {
		t.Errorf("Expected 1 content block, got %d", len(msg.Content))
	}
	if msg.Content[0].Type != ContentTypeText {
		t.Errorf("Expected content type %s, got %s", ContentTypeText, msg.Content[0].Type)
	}
	if msg.Content[0].Text != "Hello" {
		t.Errorf("Expected text 'Hello', got %q", msg.Content[0].Text)
	}
}

func TestIsThinkingModel(t *testing.T) {
	tests := []struct {
		model ClaudeModel
		want  bool
	}{
		{Claude3Opus, true},
		{Claude35Sonnet, true},
		{Claude3Sonnet, false},
		{Claude3Haiku, false},
		{Claude35Haiku, false},
		{"unknown-model", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.model), func(t *testing.T) {
			got := IsThinkingModel(tt.model)
			if got != tt.want {
				t.Errorf("IsThinkingModel(%s) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestMessagesRequest_JSONMarshaling(t *testing.T) {
	temp := 0.7
	topP := 0.9
	topK := 40

	req := MessagesRequest{
		Model:         Claude35Sonnet,
		MaxTokens:     1000,
		Messages:      []Message{CreateTextMessage(RoleUser, "Hello")},
		System:        "You are a helpful assistant",
		Stream:        true,
		Temperature:   &temp,
		TopP:          &topP,
		TopK:          &topK,
		StopSequences: []string{"\n\nHuman:", "\n\nAssistant:"},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Verify the JSON contains expected fields
	jsonStr := string(data)
	expectedFields := []string{
		`"model":"claude-3-5-sonnet-20241022"`,
		`"max_tokens":1000`,
		`"stream":true`,
		`"temperature":0.7`,
		`"top_p":0.9`,
		`"top_k":40`,
		`"system":"You are a helpful assistant"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON missing expected field: %s", field)
		}
	}

	// Test unmarshaling
	var decoded MessagesRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	if decoded.Model != req.Model {
		t.Errorf("Expected model %s, got %s", req.Model, decoded.Model)
	}
	if decoded.MaxTokens != req.MaxTokens {
		t.Errorf("Expected max_tokens %d, got %d", req.MaxTokens, decoded.MaxTokens)
	}
	if *decoded.Temperature != *req.Temperature {
		t.Errorf("Expected temperature %f, got %f", *req.Temperature, *decoded.Temperature)
	}
}

func TestContentBlock_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name  string
		block ContentBlock
	}{
		{
			name:  "text block",
			block: ContentBlock{Type: ContentTypeText, Text: "Hello"},
		},
		{
			name:  "thinking block",
			block: ContentBlock{Type: ContentTypeThinking, Thinking: "I think...", Signature: "sig123"},
		},
		{
			name:  "tool use block",
			block: ContentBlock{Type: ContentTypeToolUse, ID: "tool_123", Name: "calculator", Input: json.RawMessage(`{"x":1,"y":2}`)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.block)
			if err != nil {
				t.Fatalf("Failed to marshal block: %v", err)
			}

			var decoded ContentBlock
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal block: %v", err)
			}

			if decoded.Type != tt.block.Type {
				t.Errorf("Expected type %s, got %s", tt.block.Type, decoded.Type)
			}
		})
	}
}

func TestAnthropicProvider_parseSSEStream(t *testing.T) {
	provider, err := NewAnthropicProvider(WithAPIKey("test-key"))
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		name       string
		input      string
		wantEvents int
		wantErr    bool
	}{
		{
			name: "valid SSE stream",
			input: `data: {"type": "message_start", "message": {"id": "1"}}
data: {"type": "content_block_start", "index": 0}
data: [DONE]`,
			wantEvents: 2,
			wantErr:    false,
		},
		{
			name: "stream with comments and empty lines",
			input: `: this is a comment

data: {"type": "message_start", "message": {"id": "1"}}
: another comment

data: [DONE]`,
			wantEvents: 1,
			wantErr:    false,
		},
		{
			name:       "invalid JSON in stream",
			input:      `data: {invalid json}`,
			wantEvents: 0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventChan := make(chan StreamEvent, 10)
			reader := strings.NewReader(tt.input)

			err := provider.parseSSEStream(reader, eventChan)
			close(eventChan)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error from parseSSEStream, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			eventCount := 0
			for range eventChan {
				eventCount++
			}

			if eventCount != tt.wantEvents {
				t.Errorf("Expected %d events, got %d", tt.wantEvents, eventCount)
			}
		})
	}
}

func TestAnthropicProvider_setHeaders(t *testing.T) {
	provider, err := NewAnthropicProvider(
		WithAPIKey("test-api-key"),
		WithBaseURL("https://api.anthropic.com"),
	)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	provider.setHeaders(req)

	tests := []struct {
		header   string
		expected string
	}{
		{"Content-Type", "application/json"},
		{"X-Api-Key", "test-api-key"},
		{"Anthropic-Version", AnthropicAPIVersion},
		{"Accept", "text/event-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			got := req.Header.Get(tt.header)
			if got != tt.expected {
				t.Errorf("Header %s = %q, want %q", tt.header, got, tt.expected)
			}
		})
	}
}

func TestStreamEvent_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name  string
		event StreamEvent
	}{
		{
			name:  "message_start event",
			event: StreamEvent{Type: EventTypeMessageStart, Message: &MessagesResponse{ID: "msg_123"}},
		},
		{
			name:  "content_block_delta event",
			event: StreamEvent{Type: EventTypeContentBlockDelta, Index: 0, Delta: &ContentDelta{Type: ContentTypeText, Text: "Hello"}},
		},
		{
			name:  "message_stop event",
			event: StreamEvent{Type: EventTypeMessageStop, Usage: &Usage{InputTokens: 10, OutputTokens: 5}},
		},
		{
			name:  "error event",
			event: StreamEvent{Type: EventTypeError, Error: &StreamError{Type: "invalid_request_error", Message: "Invalid request"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.event)
			if err != nil {
				t.Fatalf("Failed to marshal event: %v", err)
			}

			var decoded StreamEvent
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal event: %v", err)
			}

			if decoded.Type != tt.event.Type {
				t.Errorf("Expected type %s, got %s", tt.event.Type, decoded.Type)
			}
		})
	}
}

func TestContextCancellation(t *testing.T) {
	mockClient := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			// Simulate slow response
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(100 * time.Millisecond):
				return newMockResponse(`{"id": "msg_123"}`, http.StatusOK), nil
			}
		},
	}

	provider, err := NewAnthropicProvider(
		WithAPIKey("test-api-key"),
		WithHTTPClient(mockClient),
	)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = provider.CreateMessage(ctx, MessagesRequest{MaxTokens: 100})
	if err == nil {
		t.Error("Expected error for cancelled context, got nil")
	}
}

func TestMessagesResponse_StopReasons(t *testing.T) {
	tests := []struct {
		stopReason   StopReason
		shouldStop   bool
		description  string
	}{
		{StopReasonEndTurn, true, "natural end"},
		{StopReasonMaxTokens, true, "max tokens reached"},
		{StopReasonStopSequence, true, "stop sequence encountered"},
		{StopReasonToolUse, true, "tool use requested"},
		{"", false, "empty stop reason"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			resp := &MessagesResponse{
				StopReason: tt.stopReason,
				Content:    []ContentBlock{{Type: ContentTypeText, Text: "Test"}},
			}

			// Verify the stop reason is preserved
			if resp.StopReason != tt.stopReason {
				t.Errorf("Expected stop reason %q, got %q", tt.stopReason, resp.StopReason)
			}

			// Verify content exists
			if len(resp.Content) == 0 {
				t.Error("Expected content blocks")
			}
		})
	}
}

func TestImageSource(t *testing.T) {
	source := ImageSource{
		Type:      "base64",
		MediaType: "image/png",
		Data:      "iVBORw0KGgo...",
	}

	data, err := json.Marshal(source)
	if err != nil {
		t.Fatalf("Failed to marshal image source: %v", err)
	}

	var decoded ImageSource
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal image source: %v", err)
	}

	if decoded.Type != source.Type {
		t.Errorf("Expected type %s, got %s", source.Type, decoded.Type)
	}
	if decoded.MediaType != source.MediaType {
		t.Errorf("Expected media type %s, got %s", source.MediaType, decoded.MediaType)
	}
	if decoded.Data != source.Data {
		t.Errorf("Expected data %s, got %s", source.Data, decoded.Data)
	}
}

func TestToolAndToolChoice(t *testing.T) {
	tool := Tool{
		Name:        "calculator",
		Description: "Performs calculations",
		InputSchema: json.RawMessage(`{"type": "object", "properties": {"expression": {"type": "string"}}}`),
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("Failed to marshal tool: %v", err)
	}

	var decoded Tool
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal tool: %v", err)
	}

	if decoded.Name != tool.Name {
		t.Errorf("Expected name %s, got %s", tool.Name, decoded.Name)
	}

	toolChoice := ToolChoice{
		Type: "auto",
	}

	data, err = json.Marshal(toolChoice)
	if err != nil {
		t.Fatalf("Failed to marshal tool choice: %v", err)
	}

	var decodedChoice ToolChoice
	if err := json.Unmarshal(data, &decodedChoice); err != nil {
		t.Fatalf("Failed to unmarshal tool choice: %v", err)
	}

	if decodedChoice.Type != toolChoice.Type {
		t.Errorf("Expected type %s, got %s", toolChoice.Type, decodedChoice.Type)
	}
}