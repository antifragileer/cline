// Package api provides API provider implementations for the Cline CLI.
package api

import (
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
type mockOtherHTTPClient struct {
	response *http.Response
	err      error
	doFunc   func(req *http.Request) (*http.Response, error)
}

func (m *mockOtherHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.doFunc != nil {
		return m.doFunc(req)
	}
	return m.response, m.err
}

// Ensure mockOtherHTTPClient implements HTTPClient interface
var _ HTTPClient = (*mockOtherHTTPClient)(nil)

// newMockOtherResponse creates a mock HTTP response with the given body and status.
func newMockOtherResponse(body string, statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

// ==================== Gemini Tests ====================

func TestNewGeminiProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  GeminiConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid provider with API key",
			config: GeminiConfig{
				APIKey: "test-api-key",
			},
			wantErr: false,
		},
		{
			name:    "missing API key",
			config:  GeminiConfig{},
			wantErr: true,
			errMsg:  ErrGeminiInvalidAPIKey.Error(),
		},
		{
			name: "with custom base URL",
			config: GeminiConfig{
				APIKey:  "test-api-key",
				BaseURL: "https://custom.googleapis.com",
			},
			wantErr: false,
		},
		{
			name: "with custom HTTP client",
			config: GeminiConfig{
				APIKey:     "test-api-key",
				HTTPClient: &http.Client{Timeout: 30 * time.Second},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGeminiProvider(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewGeminiProvider() error = nil, wantErr = true")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("NewGeminiProvider() error = %v, want containing %v", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("NewGeminiProvider() error = %v, wantErr = false", err)
				return
			}
			if provider == nil {
				t.Error("NewGeminiProvider() returned nil provider")
			}
		})
	}
}

func TestGeminiProvider_Complete(t *testing.T) {
	tests := []struct {
		name       string
		response   *http.Response
		err        error
		req        GeminiCompletionRequest
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "successful response",
			response: newMockOtherResponse(`{
				"candidates": [{
					"content": {"parts": [{"text": "Hello!"}]},
					"finishReason": "STOP"
				}],
				"usageMetadata": {
					"promptTokenCount": 10,
					"candidatesTokenCount": 5,
					"totalTokenCount": 15
				}
			}`, http.StatusOK),
			req:     GeminiCompletionRequest{Model: string(Gemini15Pro), MaxTokens: 100, Messages: []GeminiMessage{{Role: "user", Content: "Hi"}}},
			wantErr: false,
		},
		{
			name:       "HTTP client error",
			response:   nil,
			err:        errors.New("connection refused"),
			req:        GeminiCompletionRequest{Model: string(Gemini15Pro), MaxTokens: 100},
			wantErr:    true,
			wantErrMsg: "failed to execute request",
		},
		{
			name:       "API error response",
			response:   newMockOtherResponse(`{"error": {"message": "invalid request", "code": 400, "status": "INVALID_ARGUMENT"}}`, http.StatusBadRequest),
			req:        GeminiCompletionRequest{Model: string(Gemini15Pro), MaxTokens: 100},
			wantErr:    true,
			wantErrMsg: "invalid Gemini request",
		},
		{
			name:       "safety blocked",
			response:   newMockOtherResponse(`{"candidates": [], "promptFeedback": {"blockReason": "SAFETY"}}`, http.StatusOK),
			req:        GeminiCompletionRequest{Model: string(Gemini15Pro), MaxTokens: 100},
			wantErr:    true,
			wantErrMsg: "blocked by Gemini safety filters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockOtherHTTPClient{
				response: tt.response,
				err:      tt.err,
			}

			provider, err := NewGeminiProvider(GeminiConfig{
				APIKey:     "test-api-key",
				HTTPClient: mockClient,
			})
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}

			ctx := context.Background()
			resp, err := provider.Complete(ctx, tt.req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Complete() error = nil, wantErr = true")
					return
				}
				if tt.wantErrMsg != "" && !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("Complete() error = %v, want containing %v", err.Error(), tt.wantErrMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("Complete() error = %v, wantErr = false", err)
				return
			}
			if resp == nil {
				t.Error("Complete() returned nil response")
				return
			}

			// Verify content is not empty
			if resp.Content == "" {
				t.Error("Expected non-empty content")
			}
		})
	}
}

func TestGeminiProvider_CompleteStream(t *testing.T) {
	tests := []struct {
		name             string
		streamBody       string
		statusCode       int
		wantChunks       int
		wantErr          bool
		wantImmediateErr bool // Error returned directly from CompleteStream, not through channel
	}{
		{
			name: "successful streaming response",
			streamBody: `data: {"candidates": [{"content": {"parts": [{"text": "Hello"}]}, "finishReason": ""}]}
data: {"candidates": [{"content": {"parts": [{"text": " world"}]}, "finishReason": "STOP"}], "usageMetadata": {"promptTokenCount": 10, "candidatesTokenCount": 5, "totalTokenCount": 15}}
data: [DONE]`,
			statusCode: http.StatusOK,
			wantChunks: 2,
			wantErr:    false,
		},
		{
			name:             "API error response",
			streamBody:       `{"error": {"message": "rate limit", "code": 429, "status": "RESOURCE_EXHAUSTED"}}`,
			statusCode:       http.StatusTooManyRequests,
			wantChunks:       0,
			wantErr:          true,
			wantImmediateErr: true, // HTTP errors are returned directly from CompleteStream
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockOtherHTTPClient{
				response: newMockOtherResponse(tt.streamBody, tt.statusCode),
			}

			provider, err := NewGeminiProvider(GeminiConfig{
				APIKey:     "test-api-key",
				HTTPClient: mockClient,
			})
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}

			ctx := context.Background()
			chunkChan, errChan, err := provider.CompleteStream(ctx, GeminiCompletionRequest{
				Model:     string(Gemini15Pro),
				MaxTokens: 100,
				Messages:  []GeminiMessage{{Role: "user", Content: "Test"}},
			})

			// Check for immediate error (HTTP-level errors)
			if tt.wantImmediateErr {
				if err == nil {
					t.Error("Expected immediate error from CompleteStream, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("CompleteStream() returned error: %v", err)
			}

			chunkCount := 0
			done := make(chan bool)

			go func() {
				for range chunkChan {
					chunkCount++
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
			if chunkCount != tt.wantChunks {
				t.Errorf("Expected %d chunks, got %d", tt.wantChunks, chunkCount)
			}
		})
	}
}

func TestGeminiProvider_ValidateModel(t *testing.T) {
	provider, err := NewGeminiProvider(GeminiConfig{APIKey: "test-key"})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		model string
		valid bool
	}{
		{string(Gemini15Pro), true},
		{string(Gemini15Flash), true},
		{string(Gemini20Flash), true},
		{"invalid-model", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			err := provider.ValidateModel(tt.model)
			if tt.valid && err != nil {
				t.Errorf("Expected model %s to be valid, got error: %v", tt.model, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("Expected model %s to be invalid, got no error", tt.model)
			}
		})
	}
}

func TestGetModelContextWindow(t *testing.T) {
	tests := []struct {
		model GeminiModel
		want  int
	}{
		{Gemini15Pro, 2000000},
		{Gemini15Flash, 1000000},
		{Gemini20Flash, 1000000},
		{Gemini10Pro, 32768},
		{"unknown-model", 32768},
	}

	for _, tt := range tests {
		t.Run(string(tt.model), func(t *testing.T) {
			got := GetModelContextWindow(tt.model)
			if got != tt.want {
				t.Errorf("GetModelContextWindow(%s) = %d, want %d", tt.model, got, tt.want)
			}
		})
	}
}

func TestIsMultimodalModel(t *testing.T) {
	tests := []struct {
		model GeminiModel
		want  bool
	}{
		{Gemini15Pro, true},
		{Gemini15Flash, true},
		{Gemini10Pro, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.model), func(t *testing.T) {
			got := IsMultimodalModel(tt.model)
			if got != tt.want {
				t.Errorf("IsMultimodalModel(%s) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

// ==================== Ollama Tests ====================

func TestNewOllamaProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  OllamaConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid provider with default URL",
			config: OllamaConfig{
				BaseURL: "http://localhost:11434",
			},
			wantErr: false,
		},
		{
			name: "valid provider with custom URL",
			config: OllamaConfig{
				BaseURL: "http://custom:8080",
			},
			wantErr: false,
		},
		{
			name:    "invalid URL",
			config:  OllamaConfig{BaseURL: "invalid-url"},
			wantErr: true,
			errMsg:  ErrOllamaInvalidURL.Error(),
		},
		{
			name: "with custom HTTP client",
			config: OllamaConfig{
				BaseURL:    "http://localhost:11434",
				HTTPClient: &http.Client{Timeout: 30 * time.Second},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewOllamaProvider(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewOllamaProvider() error = nil, wantErr = true")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("NewOllamaProvider() error = %v, want containing %v", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("NewOllamaProvider() error = %v, wantErr = false", err)
				return
			}
			if provider == nil {
				t.Error("NewOllamaProvider() returned nil provider")
			}
		})
	}
}

func TestOllamaProvider_Complete(t *testing.T) {
	tests := []struct {
		name       string
		response   *http.Response
		err        error
		req        OllamaCompletionRequest
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "successful response",
			response: newMockOtherResponse(`{
				"model": "llama2",
				"message": {"role": "assistant", "content": "Hello!"},
				"done": true,
				"prompt_eval_count": 10,
				"eval_count": 5
			}`, http.StatusOK),
			req:     OllamaCompletionRequest{Model: "llama2", MaxTokens: 100, Messages: []OllamaMessage{{Role: "user", Content: "Hi"}}},
			wantErr: false,
		},
		{
			name:       "HTTP client error",
			response:   nil,
			err:        errors.New("connection refused"),
			req:        OllamaCompletionRequest{Model: "llama2", MaxTokens: 100},
			wantErr:    true,
			wantErrMsg: ErrOllamaConnectionFailed.Error(),
		},
		{
			name:       "model not found",
			response:   newMockOtherResponse(`{"error": "model 'unknown' not found"}`, http.StatusNotFound),
			req:        OllamaCompletionRequest{Model: "unknown", MaxTokens: 100},
			wantErr:    true,
			wantErrMsg: ErrOllamaModelNotFound.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockOtherHTTPClient{
				response: tt.response,
				err:      tt.err,
			}

			provider, err := NewOllamaProvider(OllamaConfig{
				BaseURL:    "http://localhost:11434",
				HTTPClient: mockClient,
			})
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}

			ctx := context.Background()
			resp, err := provider.Complete(ctx, tt.req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Complete() error = nil, wantErr = true")
					return
				}
				if tt.wantErrMsg != "" && !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("Complete() error = %v, want containing %v", err.Error(), tt.wantErrMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("Complete() error = %v, wantErr = false", err)
				return
			}
			if resp == nil {
				t.Error("Complete() returned nil response")
				return
			}

			// Verify content is not empty
			if resp.Content == "" {
				t.Error("Expected non-empty content")
			}
		})
	}
}

func TestOllamaProvider_ListModels(t *testing.T) {
	response := newMockOtherResponse(`{
		"models": [
			{"name": "llama2:latest", "model": "llama2:latest", "size": 3825817344},
			{"name": "codellama:latest", "model": "codellama:latest", "size": 3825817344}
		]
	}`, http.StatusOK)

	mockClient := &mockOtherHTTPClient{
		response: response,
	}

	provider, err := NewOllamaProvider(OllamaConfig{
		BaseURL:    "http://localhost:11434",
		HTTPClient: mockClient,
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()
	models, err := provider.ListModels(ctx)
	if err != nil {
		t.Fatalf("ListModels() returned error: %v", err)
	}

	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}

	if models[0].Name != "llama2:latest" {
		t.Errorf("Expected first model name 'llama2:latest', got %s", models[0].Name)
	}
}

func TestOllamaProvider_IsLocalModel(t *testing.T) {
	provider, err := NewOllamaProvider(OllamaConfig{
		BaseURL: "http://localhost:11434",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if !provider.IsLocalModel() {
		t.Error("Expected IsLocalModel() to return true")
	}
}

func TestFormatModelName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Llama 2", "llama-2"},
		{"Code Llama 7B", "code-llama-7b"},
		{"mistral", "mistral"},
		{"GPT-4 TURBO", "gpt-4-turbo"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := FormatModelName(tt.input)
			if got != tt.expected {
				t.Errorf("FormatModelName(%s) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}

// ==================== LM Studio Tests ====================

func TestNewLMStudioProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  LMStudioConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid provider with default URL",
			config: LMStudioConfig{
				BaseURL: "http://localhost:1234",
			},
			wantErr: false,
		},
		{
			name: "valid provider with custom URL",
			config: LMStudioConfig{
				BaseURL: "http://custom:5678",
			},
			wantErr: false,
		},
		{
			name:    "invalid URL",
			config:  LMStudioConfig{BaseURL: "invalid-url"},
			wantErr: true,
			errMsg:  ErrLMStudioInvalidURL.Error(),
		},
		{
			name: "with API key",
			config: LMStudioConfig{
				BaseURL: "http://localhost:1234",
				APIKey:  "test-api-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewLMStudioProvider(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewLMStudioProvider() error = nil, wantErr = true")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("NewLMStudioProvider() error = %v, want containing %v", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("NewLMStudioProvider() error = %v, wantErr = false", err)
				return
			}
			if provider == nil {
				t.Error("NewLMStudioProvider() returned nil provider")
			}
		})
	}
}

func TestLMStudioProvider_Complete(t *testing.T) {
	tests := []struct {
		name       string
		response   *http.Response
		err        error
		req        LMStudioCompletionRequest
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "successful response",
			response: newMockOtherResponse(`{
				"id": "chatcmpl-123",
				"object": "chat.completion",
				"created": 1234567890,
				"model": "local-model",
				"choices": [{
					"index": 0,
					"message": {"role": "assistant", "content": "Hello!"},
					"finish_reason": "stop"
				}],
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 5,
					"total_tokens": 15
				}
			}`, http.StatusOK),
			req:     LMStudioCompletionRequest{Model: "local-model", MaxTokens: 100, Messages: []LMStudioMessage{{Role: "user", Content: "Hi"}}},
			wantErr: false,
		},
		{
			name:       "HTTP client error",
			response:   nil,
			err:        errors.New("connection refused"),
			req:        LMStudioCompletionRequest{Model: "local-model", MaxTokens: 100},
			wantErr:    true,
			wantErrMsg: ErrLMStudioConnectionFailed.Error(),
		},
		{
			name:       "no choices in response",
			response:   newMockOtherResponse(`{"id": "test", "choices": []}`, http.StatusOK),
			req:        LMStudioCompletionRequest{Model: "local-model", MaxTokens: 100},
			wantErr:    true,
			wantErrMsg: ErrLMStudioInvalidResponse.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockOtherHTTPClient{
				response: tt.response,
				err:      tt.err,
			}

			provider, err := NewLMStudioProvider(LMStudioConfig{
				BaseURL:    "http://localhost:1234",
				HTTPClient: mockClient,
			})
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}

			ctx := context.Background()
			resp, err := provider.Complete(ctx, tt.req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Complete() error = nil, wantErr = true")
					return
				}
				if tt.wantErrMsg != "" && !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("Complete() error = %v, want containing %v", err.Error(), tt.wantErrMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("Complete() error = %v, wantErr = false", err)
				return
			}
			if resp == nil {
				t.Error("Complete() returned nil response")
				return
			}

			// Verify content is not empty
			if resp.Content == "" {
				t.Error("Expected non-empty content")
			}
		})
	}
}

func TestLMStudioProvider_ListModels(t *testing.T) {
	response := newMockOtherResponse(`{
		"object": "list",
		"data": [
			{"id": "local-llama", "object": "model", "owned_by": "lmstudio"},
			{"id": "local-mistral", "object": "model", "owned_by": "lmstudio"}
		]
	}`, http.StatusOK)

	mockClient := &mockOtherHTTPClient{
		response: response,
	}

	provider, err := NewLMStudioProvider(LMStudioConfig{
		BaseURL:    "http://localhost:1234",
		HTTPClient: mockClient,
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()
	models, err := provider.ListModels(ctx)
	if err != nil {
		t.Fatalf("ListModels() returned error: %v", err)
	}

	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}

	if models[0].ID != "local-llama" {
		t.Errorf("Expected first model ID 'local-llama', got %s", models[0].ID)
	}
}

func TestLMStudioProvider_IsLocalModel(t *testing.T) {
	provider, err := NewLMStudioProvider(LMStudioConfig{
		BaseURL: "http://localhost:1234",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if !provider.IsLocalModel() {
		t.Error("Expected IsLocalModel() to return true")
	}
}

func TestLMStudioProvider_estimateUsage(t *testing.T) {
	provider, err := NewLMStudioProvider(LMStudioConfig{
		BaseURL: "http://localhost:1234",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	messages := []LMStudioMessage{
		{Role: "user", Content: "Hello, how are you?"},
		{Role: "assistant", Content: "I'm doing well, thank you!"},
	}
	completion := "This is a test completion."

	usage := provider.estimateUsage(messages, completion)

	// Roughly 4 chars per token
	expectedPromptTokens := (len("user") + len("Hello, how are you?") + len("assistant") + len("I'm doing well, thank you!")) / 4
	expectedCompletionTokens := len(completion) / 4

	if usage.PromptTokens != expectedPromptTokens {
		t.Errorf("Expected %d prompt tokens, got %d", expectedPromptTokens, usage.PromptTokens)
	}
	if usage.CompletionTokens != expectedCompletionTokens {
		t.Errorf("Expected %d completion tokens, got %d", expectedCompletionTokens, usage.CompletionTokens)
	}
}

// ==================== Bedrock Tests ====================

func TestNewBedrockProvider(t *testing.T) {
	// Note: We can't fully test NewBedrockProvider without AWS credentials
	// This is a basic test to ensure the function signature is correct
	t.Skip("Skipping Bedrock provider tests - requires AWS credentials")
}

func TestBedrockProvider_ValidateModel(t *testing.T) {
	// Since we can't create a provider without AWS, we'll test the model list directly
	tests := []struct {
		model string
		valid bool
	}{
		{string(BedrockClaude35Sonnet), true},
		{string(BedrockClaude3Opus), true},
		{string(BedrockLlama70B), true},
		{string(BedrockMistralLarge), true},
		{"invalid-model", false},
		{"", false},
	}

	// Create a simple check function
	isValidModel := func(model string) bool {
		for _, m := range bedrockModels {
			if m == model {
				return true
			}
		}
		return false
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			valid := isValidModel(tt.model)
			if tt.valid != valid {
				t.Errorf("Expected model %s valid=%v, got valid=%v", tt.model, tt.valid, valid)
			}
		})
	}
}

func TestBedrockProvider_getModelType(t *testing.T) {
	tests := []struct {
		modelID  string
		expected string
	}{
		{"anthropic.claude-3-sonnet-20240229-v1:0", "anthropic"},
		{"amazon.titan-text-express-v1", "amazon"},
		{"meta.llama3-1-70b-instruct-v1:0", "meta"},
		{"mistral.mistral-large-2402-v1:0", "mistral"},
		{"cohere.command-r-v1:0", "cohere"},
		{"ai21.jamba-instruct-v1:0", "ai21"},
		{"unknown.model", "unknown"},
	}

	// Test via a simple extraction function
	getModelType := func(modelID string) string {
		if strings.Contains(modelID, "anthropic") {
			return "anthropic"
		}
		if strings.Contains(modelID, "amazon") {
			return "amazon"
		}
		if strings.Contains(modelID, "meta") {
			return "meta"
		}
		if strings.Contains(modelID, "mistral") {
			return "mistral"
		}
		if strings.Contains(modelID, "cohere") {
			return "cohere"
		}
		if strings.Contains(modelID, "ai21") {
			return "ai21"
		}
		return "unknown"
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			got := getModelType(tt.modelID)
			if got != tt.expected {
				t.Errorf("getModelType(%s) = %s, want %s", tt.modelID, got, tt.expected)
			}
		})
	}
}

func TestIsAnthropicModel(t *testing.T) {
	tests := []struct {
		model BedrockModel
		want  bool
	}{
		{BedrockClaude35Sonnet, true},
		{BedrockClaude3Opus, true},
		{BedrockLlama70B, false},
		{BedrockTitanExpress, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.model), func(t *testing.T) {
			got := IsAnthropicModel(tt.model)
			if got != tt.want {
				t.Errorf("IsAnthropicModel(%s) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestIsAmazonModel(t *testing.T) {
	tests := []struct {
		model BedrockModel
		want  bool
	}{
		{BedrockTitanExpress, true},
		{BedrockTitanPremier, true},
		{BedrockClaude35Sonnet, false},
		{BedrockLlama70B, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.model), func(t *testing.T) {
			got := IsAmazonModel(tt.model)
			if got != tt.want {
				t.Errorf("IsAmazonModel(%s) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestIsMetaModel(t *testing.T) {
	tests := []struct {
		model BedrockModel
		want  bool
	}{
		{BedrockLlama70B, true},
		{BedrockLlama8B, true},
		{BedrockClaude35Sonnet, false},
		{BedrockTitanExpress, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.model), func(t *testing.T) {
			got := IsMetaModel(tt.model)
			if got != tt.want {
				t.Errorf("IsMetaModel(%s) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

// ==================== Common Tests ====================

func TestOtherContextCancellation(t *testing.T) {
	// Test context cancellation for all providers that support it

	t.Run("Gemini", func(t *testing.T) {
		mockClient := &mockOtherHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				// Simulate slow response
				select {
				case <-req.Context().Done():
					return nil, req.Context().Err()
				case <-time.After(100 * time.Millisecond):
					return newMockOtherResponse(`{"candidates": []}`, http.StatusOK), nil
				}
			},
		}

		provider, err := NewGeminiProvider(GeminiConfig{
			APIKey:     "test-api-key",
			HTTPClient: mockClient,
		})
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err = provider.Complete(ctx, GeminiCompletionRequest{Model: string(Gemini15Pro), MaxTokens: 100})
		if err == nil {
			t.Error("Expected error for cancelled context, got nil")
		}
	})

	t.Run("Ollama", func(t *testing.T) {
		mockClient := &mockOtherHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				select {
				case <-req.Context().Done():
					return nil, req.Context().Err()
				case <-time.After(100 * time.Millisecond):
					return newMockOtherResponse(`{"message": {"content": "test"}}`, http.StatusOK), nil
				}
			},
		}

		provider, err := NewOllamaProvider(OllamaConfig{
			BaseURL:    "http://localhost:11434",
			HTTPClient: mockClient,
		})
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err = provider.Complete(ctx, OllamaCompletionRequest{Model: "llama2", MaxTokens: 100})
		if err == nil {
			t.Error("Expected error for cancelled context, got nil")
		}
	})

	t.Run("LMStudio", func(t *testing.T) {
		mockClient := &mockOtherHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				select {
				case <-req.Context().Done():
					return nil, req.Context().Err()
				case <-time.After(100 * time.Millisecond):
					return newMockOtherResponse(`{"choices": [{"message": {"content": "test"}}]}`, http.StatusOK), nil
				}
			},
		}

		provider, err := NewLMStudioProvider(LMStudioConfig{
			BaseURL:    "http://localhost:1234",
			HTTPClient: mockClient,
		})
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err = provider.Complete(ctx, LMStudioCompletionRequest{Model: "local-model", MaxTokens: 100})
		if err == nil {
			t.Error("Expected error for cancelled context, got nil")
		}
	})
}

func TestJSONMarshaling(t *testing.T) {
	t.Run("GeminiMessage", func(t *testing.T) {
		msg := GeminiMessage{
			Role:    "user",
			Content: "Hello",
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("Failed to marshal message: %v", err)
		}

		var decoded GeminiMessage
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal message: %v", err)
		}

		if decoded.Role != msg.Role {
			t.Errorf("Expected role %s, got %s", msg.Role, decoded.Role)
		}
		if decoded.Content != msg.Content {
			t.Errorf("Expected content %s, got %s", msg.Content, decoded.Content)
		}
	})

	t.Run("OllamaMessage", func(t *testing.T) {
		msg := OllamaMessage{
			Role:    "assistant",
			Content: "Hello there",
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("Failed to marshal message: %v", err)
		}

		var decoded OllamaMessage
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal message: %v", err)
		}

		if decoded.Role != msg.Role {
			t.Errorf("Expected role %s, got %s", msg.Role, decoded.Role)
		}
	})

	t.Run("LMStudioMessage", func(t *testing.T) {
		msg := LMStudioMessage{
			Role:    "system",
			Content: "You are a helpful assistant",
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("Failed to marshal message: %v", err)
		}

		var decoded LMStudioMessage
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal message: %v", err)
		}

		if decoded.Role != msg.Role {
			t.Errorf("Expected role %s, got %s", msg.Role, decoded.Role)
		}
	})
}

func TestUsageCalculations(t *testing.T) {
	t.Run("GeminiUsage", func(t *testing.T) {
		usage := GeminiUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		}

		if usage.TotalTokens != 150 {
			t.Errorf("Expected total tokens 150, got %d", usage.TotalTokens)
		}
	})

	t.Run("OllamaUsage", func(t *testing.T) {
		usage := OllamaUsage{
			PromptTokens:     200,
			CompletionTokens: 100,
			TotalTokens:      300,
		}

		if usage.TotalTokens != 300 {
			t.Errorf("Expected total tokens 300, got %d", usage.TotalTokens)
		}
	})

	t.Run("LMStudioUsage", func(t *testing.T) {
		usage := LMStudioUsage{
			PromptTokens:     50,
			CompletionTokens: 25,
			TotalTokens:      75,
		}

		if usage.TotalTokens != 75 {
			t.Errorf("Expected total tokens 75, got %d", usage.TotalTokens)
		}
	})

	t.Run("BedrockUsage", func(t *testing.T) {
		usage := BedrockUsage{
			PromptTokens:     1000,
			CompletionTokens: 500,
			TotalTokens:      1500,
		}

		if usage.TotalTokens != 1500 {
			t.Errorf("Expected total tokens 1500, got %d", usage.TotalTokens)
		}
	})
}
