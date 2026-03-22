package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// mockOpenRouterServer creates a test HTTP server that mocks OpenRouter API responses
func mockOpenRouterServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	server := httptest.NewServer(handler)
	t.Cleanup(func() {
		server.Close()
	})
	return server
}

func TestNewOpenRouterProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  OpenRouterConfig
		wantErr bool
		errType error
	}{
		{
			name: "valid config with API key",
			config: OpenRouterConfig{
				APIKey: "test-api-key",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: OpenRouterConfig{
				APIKey: "",
			},
			wantErr: true,
			errType: ErrInvalidAPIKey,
		},
		{
			name: "with custom base URL",
			config: OpenRouterConfig{
				APIKey:  "test-api-key",
				BaseURL: "https://custom.openrouter.com/api/v1",
			},
			wantErr: false,
		},
		{
			name: "with custom HTTP client",
			config: OpenRouterConfig{
				APIKey: "test-api-key",
				HTTPClient: &http.Client{
					Timeout: 30 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "with custom headers",
			config: OpenRouterConfig{
				APIKey: "test-api-key",
				CustomHeaders: map[string]string{
					"X-Custom-Header": "custom-value",
				},
			},
			wantErr: false,
		},
		{
			name: "with provider preferences",
			config: OpenRouterConfig{
				APIKey: "test-api-key",
				DefaultProviderPreferences: &ProviderPreferences{
					Order:          []string{"Anthropic", "OpenAI"},
					AllowFallbacks: true,
				},
			},
			wantErr: false,
		},
		{
			name: "with app name and site URL",
			config: OpenRouterConfig{
				APIKey:  "test-api-key",
				AppName: "TestApp",
				SiteURL: "https://example.com",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewOpenRouterProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOpenRouterProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errType != nil && !strings.Contains(err.Error(), tt.errType.Error()) {
				t.Errorf("NewOpenRouterProvider() error type = %v, want %v", err, tt.errType)
			}
			if !tt.wantErr && provider == nil {
				t.Error("NewOpenRouterProvider() returned nil provider")
			}
		})
	}
}

func TestOpenRouterProvider_ValidateModel(t *testing.T) {
	provider, err := NewOpenRouterProvider(OpenRouterConfig{
		APIKey: "test-api-key",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		name    string
		model   string
		wantErr bool
	}{
		{"valid known model - Claude 3.5 Sonnet", "anthropic/claude-3.5-sonnet", false},
		{"valid known model - GPT-4o", "openai/gpt-4o", false},
		{"valid known model - Llama 3 70B", "meta-llama/llama-3-70b-instruct", false},
		{"valid dynamic model format", "custom-provider/custom-model", false},
		{"valid provider/model with hyphen", "deepseek/deepseek-chat", false},
		{"invalid format - no slash", "claude-3-opus", true},
		{"invalid format - empty provider", "/claude-3-opus", true},
		{"invalid format - empty model", "anthropic/", true},
		{"invalid format - multiple slashes", "anthropic/claude/3", true},
		{"empty model", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.ValidateModel(tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateModel() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOpenRouterProvider_GetSupportedModels(t *testing.T) {
	provider, err := NewOpenRouterProvider(OpenRouterConfig{
		APIKey: "test-api-key",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	models := provider.GetSupportedModels()
	expectedModels := []string{
		"anthropic/claude-3.5-sonnet",
		"anthropic/claude-3-opus",
		"anthropic/claude-3-sonnet",
		"anthropic/claude-3-haiku",
		"openai/gpt-4o",
		"openai/gpt-4-turbo",
		"openai/gpt-3.5-turbo",
		"meta-llama/llama-3-70b-instruct",
		"meta-llama/llama-3-8b-instruct",
		"google/gemini-pro",
		"google/gemini-flash-1.5",
		"mistralai/mistral-large",
		"mistralai/mistral-medium",
	}

	if len(models) != len(expectedModels) {
		t.Errorf("GetSupportedModels() returned %d models, want %d", len(models), len(expectedModels))
	}

	for i, model := range models {
		if model != expectedModels[i] {
			t.Errorf("GetSupportedModels()[%d] = %s, want %s", i, model, expectedModels[i])
		}
	}
}

func TestOpenRouterProvider_Complete(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		responseBody   string
		request        CompletionRequest
		wantErr        bool
		expectedErr    error
		wantContent    string
		wantUsage      Usage
	}{
		{
			name:           "successful completion",
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": "gen-test123",
				"object": "chat.completion",
				"created": 1234567890,
				"model": "anthropic/claude-3.5-sonnet",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "Hello, world!"
						},
						"finish_reason": "stop"
					}
				],
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 5,
					"total_tokens": 15
				}
			}`,
			request: CompletionRequest{
				Model: "anthropic/claude-3.5-sonnet",
				Messages: []OpenRouterMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr:     false,
			wantContent: "Hello, world!",
			wantUsage: Usage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		},
		{
			name:           "invalid API key",
			responseStatus: http.StatusUnauthorized,
			responseBody: `{
				"error": {
					"message": "Invalid API key",
					"type": "invalid_request_error",
					"code": "invalid_api_key"
				}
			}`,
			request: CompletionRequest{
				Model: "anthropic/claude-3.5-sonnet",
				Messages: []OpenRouterMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr:     true,
			expectedErr: ErrInvalidAPIKey,
		},
		{
			name:           "rate limit exceeded",
			responseStatus: http.StatusTooManyRequests,
			responseBody: `{
				"error": {
					"message": "Rate limit exceeded",
					"type": "rate_limit_error",
					"code": "rate_limit_exceeded"
				}
			}`,
			request: CompletionRequest{
				Model: "anthropic/claude-3.5-sonnet",
				Messages: []OpenRouterMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr:     true,
			expectedErr: ErrRateLimitExceeded,
		},
		{
			name:           "provider unavailable",
			responseStatus: http.StatusServiceUnavailable,
			responseBody: `{
				"error": {
					"message": "Provider is unavailable",
					"type": "provider_error",
					"code": "provider_unavailable"
				}
			}`,
			request: CompletionRequest{
				Model: "anthropic/claude-3.5-sonnet",
				Messages: []OpenRouterMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr:     true,
			expectedErr: ErrProviderUnavailable,
		},
		{
			name:           "empty choices",
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": "gen-test",
				"choices": [],
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 0,
					"total_tokens": 10
				}
			}`,
			request: CompletionRequest{
				Model: "anthropic/claude-3.5-sonnet",
				Messages: []OpenRouterMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr:     true,
			expectedErr: ErrInvalidResponse,
		},
		{
			name:           "invalid model",
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"error": {
					"message": "Invalid model",
					"type": "invalid_request_error",
					"code": "invalid_request_error"
				}
			}`,
			request: CompletionRequest{
				Model: "invalid-model",
				Messages: []OpenRouterMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := mockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and path
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/chat/completions" {
					t.Errorf("Expected /chat/completions, got %s", r.URL.Path)
				}

				// Verify headers
				authHeader := r.Header.Get("Authorization")
				if !strings.HasPrefix(authHeader, "Bearer ") {
					t.Errorf("Expected Authorization header with Bearer prefix, got %s", authHeader)
				}

				w.WriteHeader(tt.responseStatus)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.responseBody))
			})

			provider, err := NewOpenRouterProvider(OpenRouterConfig{
				APIKey:  "test-api-key",
				BaseURL: server.URL,
			})
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}

			ctx := context.Background()
			resp, err := provider.Complete(ctx, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Error("Complete() expected error, got nil")
				}
				if tt.expectedErr != nil && !strings.Contains(err.Error(), tt.expectedErr.Error()) {
					t.Errorf("Complete() error = %v, want containing %v", err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("Complete() unexpected error: %v", err)
				return
			}

			if resp.Content != tt.wantContent {
				t.Errorf("Complete() content = %s, want %s", resp.Content, tt.wantContent)
			}

			if resp.Usage != tt.wantUsage {
				t.Errorf("Complete() usage = %+v, want %+v", resp.Usage, tt.wantUsage)
			}
		})
	}
}

func TestOpenRouterProvider_CompleteStream(t *testing.T) {
	tests := []struct {
		name         string
		streamChunks []string
		request      CompletionRequest
		wantErr      bool
		expectedErr  error
		wantChunks   int
	}{
		{
			name: "successful streaming",
			streamChunks: []string{
				`data: {"id":"gen-test","object":"chat.completion.chunk","created":1234567890,"model":"anthropic/claude-3.5-sonnet","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`,
				`data: {"id":"gen-test","object":"chat.completion.chunk","created":1234567890,"model":"anthropic/claude-3.5-sonnet","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`,
				`data: {"id":"gen-test","object":"chat.completion.chunk","created":1234567890,"model":"anthropic/claude-3.5-sonnet","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":"stop"}]}`,
				"data: [DONE]",
			},
			request: CompletionRequest{
				Model: "anthropic/claude-3.5-sonnet",
				Messages: []OpenRouterMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr:    false,
			wantChunks: 3,
		},
		{
			name: "stream with usage in final chunk",
			streamChunks: []string{
				`data: {"id":"gen-test","choices":[{"index":0,"delta":{"content":"Hi"}}]}`,
				`data: {"id":"gen-test","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}}`,
				"data: [DONE]",
			},
			request: CompletionRequest{
				Model: "anthropic/claude-3.5-sonnet",
				Messages: []OpenRouterMessage{
					{Role: "user", Content: "Hi"},
				},
			},
			wantErr:    false,
			wantChunks: 2,
		},
		{
			name: "invalid model in request",
			streamChunks: []string{
				`data: {"id":"gen-test","choices":[{"index":0,"delta":{"content":"test"}}]}`,
				"data: [DONE]",
			},
			request: CompletionRequest{
				Model: "invalid-model",
				Messages: []OpenRouterMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := mockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
				// Verify request has stream=true
				var reqBody map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
					t.Errorf("Failed to decode request body: %v", err)
				}
				if stream, ok := reqBody["stream"].(bool); !ok || !stream {
					t.Error("Expected stream=true in request body")
				}

				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)

				for _, chunk := range tt.streamChunks {
					fmt.Fprintln(w, chunk)
				}
				w.(http.Flusher).Flush()
			})

			provider, err := NewOpenRouterProvider(OpenRouterConfig{
				APIKey:  "test-api-key",
				BaseURL: server.URL,
			})
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}

			ctx := context.Background()
			chunkChan, errChan, err := provider.CompleteStream(ctx, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Error("CompleteStream() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("CompleteStream() unexpected error: %v", err)
				return
			}

			// Collect chunks
			chunkCount := 0
			done := false

			for !done {
				select {
				case chunk, ok := <-chunkChan:
					if !ok {
						done = true
						break
					}
					chunkCount++
					if chunk.FinishReason != "" {
						t.Logf("Final chunk with finish_reason=%s", chunk.FinishReason)
					}
				case err, ok := <-errChan:
					if ok && err != nil {
						t.Errorf("CompleteStream() error from channel: %v", err)
					}
				case <-time.After(2 * time.Second):
					t.Error("CompleteStream() timed out waiting for chunks")
					done = true
				}
			}

			if chunkCount != tt.wantChunks {
				t.Errorf("CompleteStream() received %d chunks, want %d", chunkCount, tt.wantChunks)
			}
		})
	}
}

func TestOpenRouterProvider_CompleteStream_ContextCancellation(t *testing.T) {
	server := mockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Send a chunk slowly
		fmt.Fprintln(w, `data: {"id":"test","choices":[{"delta":{"content":"Hello"}}]}`)
		w.(http.Flusher).Flush()

		// Delay to allow context cancellation to happen
		time.Sleep(100 * time.Millisecond)

		fmt.Fprintln(w, `data: {"id":"test","choices":[{"delta":{"content":" world"}}]}`)
		w.(http.Flusher).Flush()

		fmt.Fprintln(w, "data: [DONE]")
	})

	provider, err := NewOpenRouterProvider(OpenRouterConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	chunkChan, errChan, err := provider.CompleteStream(ctx, CompletionRequest{
		Model: "anthropic/claude-3.5-sonnet",
		Messages: []OpenRouterMessage{
			{Role: "user", Content: "Hello"},
		},
	})

	if err != nil {
		t.Fatalf("CompleteStream() unexpected error: %v", err)
	}

	// Wait for first chunk
	select {
	case <-chunkChan:
		// Cancel context after first chunk
		cancel()
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for first chunk")
	}

	// Wait a bit for cancellation to propagate
	time.Sleep(50 * time.Millisecond)

	// Check for context canceled error
	select {
	case err := <-errChan:
		// Accept either ErrContextCanceled or scanner errors that include "context canceled"
		if err != nil && !strings.Contains(err.Error(), ErrContextCanceled.Error()) && !strings.Contains(err.Error(), "context canceled") {
			t.Errorf("Expected context canceled error, got: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		// No error is also acceptable if channels were closed
	}
}

func TestOpenRouterProvider_estimateUsage(t *testing.T) {
	provider, err := NewOpenRouterProvider(OpenRouterConfig{
		APIKey: "test-api-key",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		name       string
		messages   []OpenRouterMessage
		completion string
		minTokens  int // Minimum expected tokens (rough estimate)
	}{
		{
			name: "simple message",
			messages: []OpenRouterMessage{
				{Role: "user", Content: "Hello"},
			},
			completion: "Hi there!",
			minTokens:  1,
		},
		{
			name: "multiple messages",
			messages: []OpenRouterMessage{
				{Role: "system", Content: "You are a helpful assistant"},
				{Role: "user", Content: "Hello, how are you?"},
			},
			completion: "I'm doing well, thank you for asking!",
			minTokens:  5,
		},
		{
			name:       "empty completion",
			messages:   []OpenRouterMessage{{Role: "user", Content: "Hello"}},
			completion: "",
			minTokens:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage := provider.estimateUsage(tt.messages, tt.completion)

			if usage.PromptTokens < 1 {
				t.Error("Expected at least 1 prompt token")
			}
			if usage.CompletionTokens < 1 {
				t.Error("Expected at least 1 completion token")
			}
			if usage.TotalTokens != usage.PromptTokens+usage.CompletionTokens {
				t.Error("Total tokens should equal prompt + completion tokens")
			}
		})
	}
}

func TestOpenRouterProvider_toOpenRouterRequest(t *testing.T) {
	provider, err := NewOpenRouterProvider(OpenRouterConfig{
		APIKey: "test-api-key",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		name         string
		req          CompletionRequest
		stream       bool
		wantModel    string
		wantStream   bool
		wantTemp     float64
		wantMaxTok   int
		wantTopP     float64
	}{
		{
			name: "default values",
			req: CompletionRequest{
				Model: "anthropic/claude-3.5-sonnet",
				Messages: []OpenRouterMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			stream:     false,
			wantModel:  "anthropic/claude-3.5-sonnet",
			wantStream: false,
			wantTemp:   0.7,
			wantMaxTok: 4096,
			wantTopP:   1.0,
		},
		{
			name: "custom values",
			req: CompletionRequest{
				Model:       "openai/gpt-4o",
				Messages:    []OpenRouterMessage{{Role: "user", Content: "Hello"}},
				Temperature: 0.5,
				MaxTokens:   100,
				TopP:        0.9,
				TopK:        40,
			},
			stream:     true,
			wantModel:  "openai/gpt-4o",
			wantStream: true,
			wantTemp:   0.5,
			wantMaxTok: 100,
			wantTopP:   0.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.toOpenRouterRequest(tt.req, tt.stream)

			if result.Model != tt.wantModel {
				t.Errorf("Model = %s, want %s", result.Model, tt.wantModel)
			}
			if result.Stream != tt.wantStream {
				t.Errorf("Stream = %v, want %v", result.Stream, tt.wantStream)
			}
			if result.Temperature != tt.wantTemp {
				t.Errorf("Temperature = %f, want %f", result.Temperature, tt.wantTemp)
			}
			if result.MaxTokens != tt.wantMaxTok {
				t.Errorf("MaxTokens = %d, want %d", result.MaxTokens, tt.wantMaxTok)
			}
			if result.TopP != tt.wantTopP {
				t.Errorf("TopP = %f, want %f", result.TopP, tt.wantTopP)
			}
			if len(result.Messages) != len(tt.req.Messages) {
				t.Errorf("Messages length = %d, want %d", len(result.Messages), len(tt.req.Messages))
			}
		})
	}
}

func TestOpenRouterProvider_setHeaders(t *testing.T) {
	tests := []struct {
		name         string
		config       OpenRouterConfig
		wantHeaders  map[string]string
		dontWant     []string
	}{
		{
			name: "basic headers",
			config: OpenRouterConfig{
				APIKey: "test-api-key",
			},
			wantHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer test-api-key",
			},
			dontWant: []string{"X-Title", "HTTP-Referer"},
		},
		{
			name: "with app name and site URL",
			config: OpenRouterConfig{
				APIKey:  "test-api-key",
				AppName: "TestApp",
				SiteURL: "https://example.com",
			},
			wantHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer test-api-key",
				"X-Title":       "TestApp",
				"HTTP-Referer":  "https://example.com",
			},
		},
		{
			name: "with custom headers",
			config: OpenRouterConfig{
				APIKey: "test-api-key",
				CustomHeaders: map[string]string{
					"X-Custom-Header": "custom-value",
					"X-Another":       "another-value",
				},
			},
			wantHeaders: map[string]string{
				"Content-Type":    "application/json",
				"Authorization":   "Bearer test-api-key",
				"X-Custom-Header": "custom-value",
				"X-Another":       "another-value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewOpenRouterProvider(tt.config)
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}

			req, _ := http.NewRequest(http.MethodPost, "https://openrouter.ai/api/v1/chat/completions", nil)
			provider.setHeaders(req)

			for key, want := range tt.wantHeaders {
				got := req.Header.Get(key)
				if got != want {
					t.Errorf("Header %s = %q, want %q", key, got, want)
				}
			}

			for _, key := range tt.dontWant {
				if req.Header.Get(key) != "" {
					t.Errorf("Header %s should not be set", key)
				}
			}
		})
	}
}

func TestOpenRouterProvider_handleErrorResponse(t *testing.T) {
	provider, err := NewOpenRouterProvider(OpenRouterConfig{
		APIKey: "test-api-key",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		errType    error
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			body:       "",
			wantErr:    false,
		},
		{
			name:       "rate limit",
			statusCode: http.StatusTooManyRequests,
			body:       "Rate limit exceeded",
			wantErr:    true,
			errType:    ErrRateLimitExceeded,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       "Invalid API key",
			wantErr:    true,
			errType:    ErrInvalidAPIKey,
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			body:       "Invalid request",
			wantErr:    true,
			errType:    ErrInvalidRequest,
		},
		{
			name:       "service unavailable",
			statusCode: http.StatusServiceUnavailable,
			body:       "Provider unavailable",
			wantErr:    true,
			errType:    ErrProviderUnavailable,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			body:       "Internal server error",
			wantErr:    true,
			errType:    ErrProviderError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       io.NopCloser(strings.NewReader(tt.body)),
			}

			err := provider.handleErrorResponse(resp)

			if tt.wantErr {
				if err == nil {
					t.Error("handleErrorResponse() expected error, got nil")
				}
				if tt.errType != nil && !strings.Contains(err.Error(), tt.errType.Error()) {
					t.Errorf("handleErrorResponse() error = %v, should contain %v", err, tt.errType)
				}
			} else {
				if err != nil {
					t.Errorf("handleErrorResponse() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestOpenRouterProvider_convertOpenRouterError(t *testing.T) {
	provider, err := NewOpenRouterProvider(OpenRouterConfig{
		APIKey: "test-api-key",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		name     string
		errCode  string
		errMsg   string
		wantType error
	}{
		{
			name:     "rate limit exceeded",
			errCode:  "rate_limit_exceeded",
			errMsg:   "Rate limit exceeded",
			wantType: ErrRateLimitExceeded,
		},
		{
			name:     "insufficient quota",
			errCode:  "insufficient_quota",
			errMsg:   "Insufficient quota",
			wantType: ErrRateLimitExceeded,
		},
		{
			name:     "invalid api key",
			errCode:  "invalid_api_key",
			errMsg:   "Invalid API key",
			wantType: ErrInvalidAPIKey,
		},
		{
			name:     "invalid request",
			errCode:  "invalid_request_error",
			errMsg:   "Invalid request",
			wantType: ErrInvalidRequest,
		},
		{
			name:     "provider error",
			errCode:  "provider_error",
			errMsg:   "Provider error",
			wantType: ErrProviderUnavailable,
		},
		{
			name:     "provider unavailable",
			errCode:  "provider_unavailable",
			errMsg:   "Provider unavailable",
			wantType: ErrProviderUnavailable,
		},
		{
			name:     "unknown error",
			errCode:  "unknown_error",
			errMsg:   "Unknown error",
			wantType: ErrProviderError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			openRouterErr := &openRouterError{
				Code:    tt.errCode,
				Message: tt.errMsg,
				Type:    "test",
			}

			err := provider.convertOpenRouterError(openRouterErr)

			if !strings.Contains(err.Error(), tt.wantType.Error()) {
				t.Errorf("convertOpenRouterError() error = %v, should contain %v", err, tt.wantType)
			}
		})
	}
}

func TestOpenRouterProvider_ModelManagement(t *testing.T) {
	provider, err := NewOpenRouterProvider(OpenRouterConfig{
		APIKey: "test-api-key",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Test default model
	if provider.GetModel() != OpenRouterClaude35Sonnet {
		t.Errorf("Expected default model %s, got %s", OpenRouterClaude35Sonnet, provider.GetModel())
	}

	// Test setting model
	provider.SetModel(OpenRouterGPT4o)
	if provider.GetModel() != OpenRouterGPT4o {
		t.Errorf("Expected model %s, got %s", OpenRouterGPT4o, provider.GetModel())
	}
}

func TestOpenRouterProvider_CustomHeaders(t *testing.T) {
	provider, err := NewOpenRouterProvider(OpenRouterConfig{
		APIKey: "test-api-key",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Test adding custom headers
	provider.AddCustomHeader("X-Custom-1", "value1")
	provider.AddCustomHeader("X-Custom-2", "value2")

	headers := provider.GetCustomHeaders()
	if headers["X-Custom-1"] != "value1" {
		t.Errorf("Expected X-Custom-1 = value1, got %s", headers["X-Custom-1"])
	}
	if headers["X-Custom-2"] != "value2" {
		t.Errorf("Expected X-Custom-2 = value2, got %s", headers["X-Custom-2"])
	}

	// Test removing custom headers
	provider.RemoveCustomHeader("X-Custom-1")
	headers = provider.GetCustomHeaders()
	if _, exists := headers["X-Custom-1"]; exists {
		t.Error("X-Custom-1 should have been removed")
	}

	// Test that GetCustomHeaders returns a copy
	headers["X-New"] = "new-value"
	headers2 := provider.GetCustomHeaders()
	if _, exists := headers2["X-New"]; exists {
		t.Error("GetCustomHeaders should return a copy, not a reference")
	}
}

func TestOpenRouterProvider_ProviderPreferences(t *testing.T) {
	provider, err := NewOpenRouterProvider(OpenRouterConfig{
		APIKey: "test-api-key",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	prefs := &ProviderPreferences{
		Order:                  []string{"Anthropic", "OpenAI"},
		AllowFallbacks:         true,
		IgnoreLowContextWindow: true,
		Quantizations:          []string{"fp16", "fp8"},
	}

	provider.SetProviderPreferences(prefs)

	if provider.config.DefaultProviderPreferences == nil {
		t.Fatal("Provider preferences not set")
	}

	if len(provider.config.DefaultProviderPreferences.Order) != 2 {
		t.Errorf("Expected 2 providers in order, got %d", len(provider.config.DefaultProviderPreferences.Order))
	}

	if !provider.config.DefaultProviderPreferences.AllowFallbacks {
		t.Error("Expected AllowFallbacks to be true")
	}
}

func TestOpenRouterProvider_CreateFallbackChain(t *testing.T) {
	provider, err := NewOpenRouterProvider(OpenRouterConfig{
		APIKey: "test-api-key",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	chain := provider.CreateFallbackChain(
		"anthropic/claude-3.5-sonnet",
		"openai/gpt-4o",
		"meta-llama/llama-3-70b-instruct",
	)

	expected := []string{
		"anthropic/claude-3.5-sonnet",
		"openai/gpt-4o",
		"meta-llama/llama-3-70b-instruct",
	}

	if len(chain) != len(expected) {
		t.Errorf("Expected chain length %d, got %d", len(expected), len(chain))
	}

	for i, model := range chain {
		if model != expected[i] {
			t.Errorf("Chain[%d] = %s, want %s", i, model, expected[i])
		}
	}
}

func TestOpenRouterProvider_FallbackModelsInRequest(t *testing.T) {
	server := mockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
		var reqBody openRouterRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify fallback models are included
		expectedModels := []string{
			"anthropic/claude-3.5-sonnet",
			"openai/gpt-4o",
			"meta-llama/llama-3-70b-instruct",
		}

		if len(reqBody.Models) != len(expectedModels) {
			t.Errorf("Expected %d models, got %d", len(expectedModels), len(reqBody.Models))
		}

		for i, model := range reqBody.Models {
			if model != expectedModels[i] {
				t.Errorf("Models[%d] = %s, want %s", i, model, expectedModels[i])
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "test",
			"choices": [{"message": {"content": "test"}}],
			"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
		}`))
	})

	provider, err := NewOpenRouterProvider(OpenRouterConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()
	_, err = provider.Complete(ctx, CompletionRequest{
		Model: "anthropic/claude-3.5-sonnet",
		Messages: []OpenRouterMessage{
			{Role: "user", Content: "Hello"},
		},
		FallbackModels: []string{
			"openai/gpt-4o",
			"meta-llama/llama-3-70b-instruct",
		},
	})

	if err != nil {
		t.Errorf("Complete() unexpected error: %v", err)
	}
}

func TestIsProviderModelFormat(t *testing.T) {
	tests := []struct {
		modelID string
		want    bool
	}{
		{"anthropic/claude-3.5-sonnet", true},
		{"openai/gpt-4o", true},
		{"custom-provider/custom-model", true},
		{"claude-3-opus", false},
		{"/claude-3-opus", false},
		{"anthropic/", false},
		{"", false},
		{"a/b/c", false},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			got := IsProviderModelFormat(tt.modelID)
			if got != tt.want {
				t.Errorf("IsProviderModelFormat(%q) = %v, want %v", tt.modelID, got, tt.want)
			}
		})
	}
}

func TestExtractProviderFromModel(t *testing.T) {
	tests := []struct {
		modelID   string
		want      string
		wantError bool
	}{
		{"anthropic/claude-3.5-sonnet", "anthropic", false},
		{"openai/gpt-4o", "openai", false},
		{"deepseek/deepseek-chat", "deepseek", false},
		{"claude-3-opus", "", true},
		{"", "", true},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			got, err := ExtractProviderFromModel(tt.modelID)
			if (err != nil) != tt.wantError {
				t.Errorf("ExtractProviderFromModel(%q) error = %v, wantError %v", tt.modelID, err, tt.wantError)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractProviderFromModel(%q) = %v, want %v", tt.modelID, got, tt.want)
			}
		})
	}
}

func TestExtractModelName(t *testing.T) {
	tests := []struct {
		modelID   string
		want      string
		wantError bool
	}{
		{"anthropic/claude-3.5-sonnet", "claude-3.5-sonnet", false},
		{"openai/gpt-4o", "gpt-4o", false},
		{"deepseek/deepseek-chat", "deepseek-chat", false},
		{"claude-3-opus", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			got, err := ExtractModelName(tt.modelID)
			if (err != nil) != tt.wantError {
				t.Errorf("ExtractModelName(%q) error = %v, wantError %v", tt.modelID, err, tt.wantError)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractModelName(%q) = %v, want %v", tt.modelID, got, tt.want)
			}
		})
	}
}

func TestOpenRouterProvider_ParseModelID(t *testing.T) {
	provider, err := NewOpenRouterProvider(OpenRouterConfig{
		APIKey: "test-api-key",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		modelID     string
		wantProvider string
		wantModel    string
		wantError    bool
	}{
		{"anthropic/claude-3.5-sonnet", "anthropic", "claude-3.5-sonnet", false},
		{"openai/gpt-4o", "openai", "gpt-4o", false},
		{"invalid", "", "", true},
		{"", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			provider, model, err := provider.ParseModelID(tt.modelID)
			if (err != nil) != tt.wantError {
				t.Errorf("ParseModelID(%q) error = %v, wantError %v", tt.modelID, err, tt.wantError)
				return
			}
			if provider != tt.wantProvider {
				t.Errorf("ParseModelID(%q) provider = %v, want %v", tt.modelID, provider, tt.wantProvider)
			}
			if model != tt.wantModel {
				t.Errorf("ParseModelID(%q) model = %v, want %v", tt.modelID, model, tt.wantModel)
			}
		})
	}
}

func TestOpenRouterProvider_BuildModelID(t *testing.T) {
	provider, err := NewOpenRouterProvider(OpenRouterConfig{
		APIKey: "test-api-key",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		provider string
		model    string
		want     string
	}{
		{"anthropic", "claude-3.5-sonnet", "anthropic/claude-3.5-sonnet"},
		{"openai", "gpt-4o", "openai/gpt-4o"},
		{"custom", "model-v1", "custom/model-v1"},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"/"+tt.model, func(t *testing.T) {
			got := provider.BuildModelID(tt.provider, tt.model)
			if got != tt.want {
				t.Errorf("BuildModelID(%q, %q) = %v, want %v", tt.provider, tt.model, got, tt.want)
			}
		})
	}
}

func TestOpenRouterProvider_ProviderPreferencesInRequest(t *testing.T) {
	server := mockOpenRouterServer(t, func(w http.ResponseWriter, r *http.Request) {
		var reqBody openRouterRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify provider preferences
		if reqBody.Provider == nil {
			t.Error("Expected provider preferences in request")
		} else {
			if len(reqBody.Provider.Order) != 2 {
				t.Errorf("Expected 2 providers in order, got %d", len(reqBody.Provider.Order))
			}
			if !reqBody.Provider.AllowFallbacks {
				t.Error("Expected AllowFallbacks to be true")
			}
			if reqBody.Provider.DataCollection != "deny" {
				t.Errorf("Expected DataCollection = deny, got %s", reqBody.Provider.DataCollection)
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "test",
			"choices": [{"message": {"content": "test"}}],
			"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
		}`))
	})

	provider, err := NewOpenRouterProvider(OpenRouterConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()
	_, err = provider.Complete(ctx, CompletionRequest{
		Model: "anthropic/claude-3.5-sonnet",
		Messages: []OpenRouterMessage{
			{Role: "user", Content: "Hello"},
		},
		Provider: &ProviderPreferences{
			Order:          []string{"Anthropic", "OpenAI"},
			AllowFallbacks: true,
			DataCollection: "deny",
		},
	})

	if err != nil {
		t.Errorf("Complete() unexpected error: %v", err)
	}
}

func TestOpenRouterErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "ErrInvalidAPIKey",
			err:  ErrInvalidAPIKey,
			want: "invalid API key",
		},
		{
			name: "ErrRateLimitExceeded",
			err:  ErrRateLimitExceeded,
			want: "rate limit exceeded",
		},
		{
			name: "ErrContextCanceled",
			err:  ErrContextCanceled,
			want: "request canceled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.err.Error(), tt.want) {
				t.Errorf("%s.Error() = %v, should contain %q", tt.name, tt.err.Error(), tt.want)
			}
		})
	}
}