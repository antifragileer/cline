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

// mockOpenAIServer creates a test HTTP server that mocks OpenAI API responses
func mockOpenAIServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	server := httptest.NewServer(handler)
	t.Cleanup(func() {
		server.Close()
	})
	return server
}

func TestNewOpenAIProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  OpenAIConfig
		wantErr bool
		errType error
	}{
		{
			name: "valid config with API key",
			config: OpenAIConfig{
				APIKey: "test-api-key",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: OpenAIConfig{
				APIKey: "",
			},
			wantErr: true,
			errType: ErrOpenAIInvalidAPIKey,
		},
		{
			name: "with custom base URL",
			config: OpenAIConfig{
				APIKey:  "test-api-key",
				BaseURL: "https://custom.openai.com/v1",
			},
			wantErr: false,
		},
		{
			name: "with Azure base URL",
			config: OpenAIConfig{
				APIKey:  "test-api-key",
				BaseURL: "https://my-resource.openai.azure.com",
			},
			wantErr: false,
		},
		{
			name: "with custom HTTP client",
			config: OpenAIConfig{
				APIKey: "test-api-key",
				HTTPClient: &http.Client{
					Timeout: 30 * time.Second,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewOpenAIProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOpenAIProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !isOpenAIError(err, tt.errType) {
				t.Errorf("NewOpenAIProvider() error type = %v, want %v", err, tt.errType)
			}
			if !tt.wantErr && provider == nil {
				t.Error("NewOpenAIProvider() returned nil provider")
			}
		})
	}
}

func TestOpenAIProvider_ValidateModel(t *testing.T) {
	provider, err := NewOpenAIProvider(OpenAIConfig{
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
		{"valid gpt-4o", "gpt-4o", false},
		{"valid gpt-4", "gpt-4", false},
		{"valid gpt-4-turbo", "gpt-4-turbo", false},
		{"valid gpt-3.5-turbo", "gpt-3.5-turbo", false},
		{"valid gpt-3.5-turbo-16k", "gpt-3.5-turbo-16k", false},
		{"invalid model", "gpt-5", true},
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

func TestOpenAIProvider_GetSupportedModels(t *testing.T) {
	provider, err := NewOpenAIProvider(OpenAIConfig{
		APIKey: "test-api-key",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	models := provider.GetSupportedModels()
	expectedModels := []string{
		"gpt-4o",
		"gpt-4-turbo",
		"gpt-4",
		"gpt-3.5-turbo",
		"gpt-3.5-turbo-16k",
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

func TestOpenAIProvider_Complete(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		responseBody   string
		request        OpenAICompletionRequest
		wantErr        bool
		expectedErr    error
		wantContent    string
		wantUsage      OpenAIUsage
	}{
		{
			name:           "successful completion",
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": "chatcmpl-test",
				"object": "chat.completion",
				"created": 1234567890,
				"model": "gpt-4o",
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
			request: OpenAICompletionRequest{
				Model: "gpt-4o",
				Messages: []OpenAIMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr:     false,
			wantContent: "Hello, world!",
			wantUsage: OpenAIUsage{
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
			request: OpenAICompletionRequest{
				Model: "gpt-4o",
				Messages: []OpenAIMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr:     true,
			expectedErr: ErrOpenAIInvalidAPIKey,
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
			request: OpenAICompletionRequest{
				Model: "gpt-4o",
				Messages: []OpenAIMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr:     true,
			expectedErr: ErrOpenAIRateLimitExceeded,
		},
		{
			name:           "empty choices",
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": "chatcmpl-test",
				"choices": [],
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 0,
					"total_tokens": 10
				}
			}`,
			request: OpenAICompletionRequest{
				Model: "gpt-4o",
				Messages: []OpenAIMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr:     true,
			expectedErr: ErrOpenAIInvalidResponse,
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
			request: OpenAICompletionRequest{
				Model: "invalid-model",
				Messages: []OpenAIMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := mockOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
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

			provider, err := NewOpenAIProvider(OpenAIConfig{
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
				if tt.expectedErr != nil && !isOpenAIError(err, tt.expectedErr) {
					t.Errorf("Complete() error = %v, want %v", err, tt.expectedErr)
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

func TestOpenAIProvider_CompleteStream(t *testing.T) {
	tests := []struct {
		name         string
		streamChunks []string
		request      OpenAICompletionRequest
		wantErr      bool
		expectedErr  error
		wantChunks   int
	}{
		{
			name: "successful streaming",
			streamChunks: []string{
				`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`,
				`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`,
				`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":"stop"}]}`,
				"data: [DONE]",
			},
			request: OpenAICompletionRequest{
				Model: "gpt-4o",
				Messages: []OpenAIMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr:    false,
			wantChunks: 3,
		},
		{
			name: "stream with usage in final chunk",
			streamChunks: []string{
				`data: {"id":"chatcmpl-test","choices":[{"index":0,"delta":{"content":"Hi"}}]}`,
				`data: {"id":"chatcmpl-test","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}}`,
				"data: [DONE]",
			},
			request: OpenAICompletionRequest{
				Model: "gpt-4o",
				Messages: []OpenAIMessage{
					{Role: "user", Content: "Hi"},
				},
			},
			wantErr:    false,
			wantChunks: 2,
		},
		{
			name: "invalid model in request",
			streamChunks: []string{
				`data: {"id":"chatcmpl-test","choices":[{"index":0,"delta":{"content":"test"}}]}`,
				"data: [DONE]",
			},
			request: OpenAICompletionRequest{
				Model: "invalid-model",
				Messages: []OpenAIMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := mockOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
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

			provider, err := NewOpenAIProvider(OpenAIConfig{
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
						// Final chunk should have usage if available
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

func TestOpenAIProvider_CompleteStream_ContextCancellation(t *testing.T) {
	server := mockOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
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

	provider, err := NewOpenAIProvider(OpenAIConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	chunkChan, errChan, err := provider.CompleteStream(ctx, OpenAICompletionRequest{
		Model: "gpt-4o",
		Messages: []OpenAIMessage{
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
		if err != nil && !isOpenAIError(err, ErrOpenAIContextCanceled) {
			t.Errorf("Expected context canceled error, got: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		// No error is also acceptable if channels were closed
	}
}

func TestOpenAIProvider_isAzureEndpoint(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		want    bool
	}{
		{"Azure OpenAI endpoint", "https://my-resource.openai.azure.com", true},
		{"Azure with microsoft.com", "https://my-resource.microsoft.com", true},
		{"Standard OpenAI endpoint", "https://api.openai.com/v1", false},
		{"Custom endpoint", "https://custom.openai.com", false},
		{"Empty endpoint", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewOpenAIProvider(OpenAIConfig{
				APIKey:  "test-api-key",
				BaseURL: tt.baseURL,
			})
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}

			got := provider.isAzureEndpoint()
			if got != tt.want {
				t.Errorf("isAzureEndpoint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOpenAIProvider_AzureHeaders(t *testing.T) {
	t.Run("Azure endpoint uses api-key header", func(t *testing.T) {
		// Create provider with Azure URL directly
		provider, err := NewOpenAIProvider(OpenAIConfig{
			APIKey:  "azure-api-key",
			BaseURL: "https://test.openai.azure.com/openai/deployments/my-deployment",
		})
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		// Test header setting for Azure
		req, _ := http.NewRequest(http.MethodPost, "https://test.openai.azure.com/chat/completions", nil)
		provider.setHeaders(req)

		if req.Header.Get("Authorization") != "" {
			t.Error("Authorization header should be removed for Azure")
		}
		if req.Header.Get("api-key") != "azure-api-key" {
			t.Error("api-key header should be set for Azure")
		}
	})

	t.Run("Standard endpoint uses Authorization header", func(t *testing.T) {
		provider, err := NewOpenAIProvider(OpenAIConfig{
			APIKey:  "standard-api-key",
			BaseURL: "https://api.openai.com/v1",
		})
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		req, _ := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", nil)
		provider.setHeaders(req)

		authHeader := req.Header.Get("Authorization")
		if authHeader != "Bearer standard-api-key" {
			t.Errorf("Expected Authorization header with Bearer, got %s", authHeader)
		}
		if req.Header.Get("api-key") != "" {
			t.Error("api-key header should not be set for standard endpoint")
		}
	})
}

func TestOpenAIProvider_estimateUsage(t *testing.T) {
	provider, err := NewOpenAIProvider(OpenAIConfig{
		APIKey: "test-api-key",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		name       string
		messages   []OpenAIMessage
		completion string
		minTokens  int // Minimum expected tokens (rough estimate)
	}{
		{
			name: "simple message",
			messages: []OpenAIMessage{
				{Role: "user", Content: "Hello"},
			},
			completion: "Hi there!",
			minTokens:  1,
		},
		{
			name: "multiple messages",
			messages: []OpenAIMessage{
				{Role: "system", Content: "You are a helpful assistant"},
				{Role: "user", Content: "Hello, how are you?"},
			},
			completion: "I'm doing well, thank you for asking!",
			minTokens:  5,
		},
		{
			name:       "empty completion",
			messages:   []OpenAIMessage{{Role: "user", Content: "Hello"}},
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

func TestOpenAIProvider_toOpenAIRequest(t *testing.T) {
	provider, err := NewOpenAIProvider(OpenAIConfig{
		APIKey: "test-api-key",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	tests := []struct {
		name       string
		req        OpenAICompletionRequest
		stream     bool
		wantModel  string
		wantStream bool
		wantTemp   float64
		wantMaxTok int
		wantTopP   float64
	}{
		{
			name: "default values",
			req: OpenAICompletionRequest{
				Model: "gpt-4o",
				Messages: []OpenAIMessage{
					{Role: "user", Content: "Hello"},
				},
			},
			stream:     false,
			wantModel:  "gpt-4o",
			wantStream: false,
			wantTemp:   DefaultTemperature,
			wantMaxTok: DefaultMaxTokens,
			wantTopP:   1.0,
		},
		{
			name: "custom values",
			req: OpenAICompletionRequest{
				Model:       "gpt-3.5-turbo",
				Messages:    []OpenAIMessage{{Role: "user", Content: "Hello"}},
				Temperature: 0.5,
				MaxTokens:   100,
				TopP:        0.9,
			},
			stream:     true,
			wantModel:  "gpt-3.5-turbo",
			wantStream: true,
			wantTemp:   0.5,
			wantMaxTok: 100,
			wantTopP:   0.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.toOpenAIInternalRequest(tt.req, tt.stream)

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

func TestOpenAIProvider_OrganizationHeader(t *testing.T) {
	provider, err := NewOpenAIProvider(OpenAIConfig{
		APIKey:       "test-api-key",
		Organization: "org-test123",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", nil)
	provider.setHeaders(req)

	orgHeader := req.Header.Get("OpenAI-Organization")
	if orgHeader != "org-test123" {
		t.Errorf("Expected OpenAI-Organization header = %s, got %s", "org-test123", orgHeader)
	}
}

func TestOpenAIProvider_handleErrorResponse(t *testing.T) {
	provider, err := NewOpenAIProvider(OpenAIConfig{
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
			errType:    ErrOpenAIRateLimitExceeded,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       "Invalid API key",
			wantErr:    true,
			errType:    ErrOpenAIInvalidAPIKey,
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			body:       "Invalid request",
			wantErr:    true,
			errType:    ErrOpenAIInvalidRequest,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			body:       "Internal server error",
			wantErr:    true,
			errType:    ErrOpenAIProviderError,
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
				if tt.errType != nil && !isOpenAIError(err, tt.errType) {
					// Check if error message contains expected type
					if !strings.Contains(err.Error(), tt.errType.Error()) {
						t.Errorf("handleErrorResponse() error = %v, should contain %v", err, tt.errType)
					}
				}
			} else {
				if err != nil {
					t.Errorf("handleErrorResponse() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestOpenAIProvider_convertOpenAIError(t *testing.T) {
	provider, err := NewOpenAIProvider(OpenAIConfig{
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
			wantType: ErrOpenAIRateLimitExceeded,
		},
		{
			name:     "insufficient quota",
			errCode:  "insufficient_quota",
			errMsg:   "Insufficient quota",
			wantType: ErrOpenAIRateLimitExceeded,
		},
		{
			name:     "invalid api key",
			errCode:  "invalid_api_key",
			errMsg:   "Invalid API key",
			wantType: ErrOpenAIInvalidAPIKey,
		},
		{
			name:     "invalid request",
			errCode:  "invalid_request_error",
			errMsg:   "Invalid request",
			wantType: ErrOpenAIInvalidRequest,
		},
		{
			name:     "unknown error",
			errCode:  "unknown_error",
			errMsg:   "Unknown error",
			wantType: ErrOpenAIProviderError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			openAIErr := &openAIInternalError{
				Code:    tt.errCode,
				Message: tt.errMsg,
				Type:    "test",
			}

			err := provider.convertOpenAIError(openAIErr)

			if !strings.Contains(err.Error(), tt.wantType.Error()) {
				t.Errorf("convertOpenAIError() error = %v, should contain %v", err, tt.wantType)
			}
		})
	}
}

// isOpenAIError checks if an error matches the target error
func isOpenAIError(err, target error) bool {
	if err == nil || target == nil {
		return err == target
	}
	return strings.Contains(err.Error(), target.Error())
}
