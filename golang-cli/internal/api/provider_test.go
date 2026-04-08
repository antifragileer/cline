// Package api provides API provider implementations for the Cline CLI.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// mockProviderServer creates a test HTTP server for provider testing
func mockProviderServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	server := httptest.NewServer(handler)
	t.Cleanup(func() {
		server.Close()
	})
	return server
}

// TestProviderFactory tests the provider factory functionality
func TestProviderFactory(t *testing.T) {
	t.Run("Create Anthropic provider", func(t *testing.T) {
		factory := NewProviderFactory()
		factory.SetAnthropicConfig(&AnthropicProviderConfig{
			APIKey: "test-api-key",
			Model:  string(Claude35Sonnet),
		})

		provider, err := factory.CreateProvider(ProviderAnthropic)
		if err != nil {
			t.Fatalf("Failed to create Anthropic provider: %v", err)
		}

		if provider == nil {
			t.Fatal("Provider is nil")
		}

		anthropicProvider, ok := provider.(*AnthropicProvider)
		if !ok {
			t.Fatalf("Expected *AnthropicProvider, got %T", provider)
		}

		if anthropicProvider.GetModel() != Claude35Sonnet {
			t.Errorf("Expected model %s, got %s", Claude35Sonnet, anthropicProvider.GetModel())
		}
	})

	t.Run("Create OpenAI provider", func(t *testing.T) {
		factory := NewProviderFactory()
		factory.SetOpenAIConfig(&OpenAIConfig{
			APIKey: "test-api-key",
		})

		provider, err := factory.CreateProvider(ProviderOpenAI)
		if err != nil {
			t.Fatalf("Failed to create OpenAI provider: %v", err)
		}

		if provider == nil {
			t.Fatal("Provider is nil")
		}
	})

	t.Run("Create OpenRouter provider", func(t *testing.T) {
		factory := NewProviderFactory()
		factory.SetOpenRouterConfig(&OpenRouterConfig{
			APIKey: "test-api-key",
		})

		provider, err := factory.CreateProvider(ProviderOpenRouter)
		if err != nil {
			t.Fatalf("Failed to create OpenRouter provider: %v", err)
		}

		if provider == nil {
			t.Fatal("Provider is nil")
		}
	})

	t.Run("Create Gemini provider", func(t *testing.T) {
		factory := NewProviderFactory()
		factory.SetGeminiConfig(&GeminiConfig{
			APIKey: "test-api-key",
		})

		provider, err := factory.CreateProvider(ProviderGemini)
		if err != nil {
			t.Fatalf("Failed to create Gemini provider: %v", err)
		}

		if provider == nil {
			t.Fatal("Provider is nil")
		}
	})

	t.Run("Create Ollama provider", func(t *testing.T) {
		factory := NewProviderFactory()
		factory.SetOllamaConfig(&OllamaConfig{
			BaseURL: "http://localhost:11434",
		})

		provider, err := factory.CreateProvider(ProviderOllama)
		if err != nil {
			t.Fatalf("Failed to create Ollama provider: %v", err)
		}

		if provider == nil {
			t.Fatal("Provider is nil")
		}
	})

	t.Run("Create LM Studio provider", func(t *testing.T) {
		factory := NewProviderFactory()
		factory.SetLMStudioConfig(&LMStudioConfig{
			BaseURL: "http://localhost:1234",
		})

		provider, err := factory.CreateProvider(ProviderLMStudio)
		if err != nil {
			t.Fatalf("Failed to create LM Studio provider: %v", err)
		}

		if provider == nil {
			t.Fatal("Provider is nil")
		}
	})

	t.Run("Create Bedrock provider returns error", func(t *testing.T) {
		factory := NewProviderFactory()
		factory.SetBedrockConfig(&BedrockConfig{
			Region: "us-east-1",
		})

		_, err := factory.CreateProvider(ProviderBedrock)
		if err == nil {
			t.Fatal("Expected error for Bedrock provider, got nil")
		}
	})

	t.Run("Create unknown provider returns error", func(t *testing.T) {
		factory := NewProviderFactory()
		_, err := factory.CreateProvider(ProviderType("unknown"))
		if err == nil {
			t.Fatal("Expected error for unknown provider, got nil")
		}
	})

	t.Run("Create provider without config returns error", func(t *testing.T) {
		factory := NewProviderFactory()
		_, err := factory.CreateProvider(ProviderOpenAI)
		if err == nil {
			t.Fatal("Expected error when config not set, got nil")
		}
	})
}

// TestProviderRegistry tests the provider registry functionality
func TestProviderRegistry(t *testing.T) {
	t.Run("Register and get provider", func(t *testing.T) {
		factory := NewProviderFactory()
		factory.SetOpenAIConfig(&OpenAIConfig{
			APIKey: "test-api-key",
		})

		registry := NewProviderRegistry(factory)

		// Create and register a provider
		provider, err := factory.CreateProvider(ProviderOpenAI)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		registry.Register(ProviderOpenAI, provider)

		// Get the provider from registry
		retrieved, err := registry.Get(ProviderOpenAI)
		if err != nil {
			t.Fatalf("Failed to get provider from registry: %v", err)
		}

		if retrieved == nil {
			t.Fatal("Retrieved provider is nil")
		}
	})

	t.Run("Get provider from factory", func(t *testing.T) {
		factory := NewProviderFactory()
		factory.SetOpenAIConfig(&OpenAIConfig{
			APIKey: "test-api-key",
		})

		registry := NewProviderRegistry(factory)

		// Get provider (should create via factory)
		provider, err := registry.Get(ProviderOpenAI)
		if err != nil {
			t.Fatalf("Failed to get provider: %v", err)
		}

		if provider == nil {
			t.Fatal("Provider is nil")
		}

		// Get again (should return cached)
		provider2, err := registry.Get(ProviderOpenAI)
		if err != nil {
			t.Fatalf("Failed to get provider second time: %v", err)
		}

		if provider != provider2 {
			t.Error("Expected same provider instance (cached)")
		}
	})

	t.Run("Get unregistered provider without factory returns error", func(t *testing.T) {
		registry := NewProviderRegistry(nil)
		_, err := registry.Get(ProviderOpenAI)
		if err == nil {
			t.Fatal("Expected error for unregistered provider, got nil")
		}
	})

	t.Run("Get available providers", func(t *testing.T) {
		factory := NewProviderFactory()
		registry := NewProviderRegistry(factory)

		// Initially empty
		available := registry.GetAvailableProviders()
		if len(available) != 0 {
			t.Errorf("Expected 0 providers initially, got %d", len(available))
		}

		// Register a provider
		mockProvider := &mockGenericProvider{}
		registry.Register(ProviderOpenAI, mockProvider)

		available = registry.GetAvailableProviders()
		if len(available) != 1 {
			t.Errorf("Expected 1 provider, got %d", len(available))
		}

		if available[0] != ProviderOpenAI {
			t.Errorf("Expected provider %s, got %s", ProviderOpenAI, available[0])
		}
	})

	t.Run("Is registered", func(t *testing.T) {
		factory := NewProviderFactory()
		registry := NewProviderRegistry(factory)

		if registry.IsRegistered(ProviderOpenAI) {
			t.Error("Expected OpenAI not to be registered initially")
		}

		mockProvider := &mockGenericProvider{}
		registry.Register(ProviderOpenAI, mockProvider)

		if !registry.IsRegistered(ProviderOpenAI) {
			t.Error("Expected OpenAI to be registered")
		}
	})
}

// mockGenericProvider is a mock implementation of the GenericProvider interface for testing
type mockGenericProvider struct {
	model string
}

func (m *mockGenericProvider) Complete(ctx context.Context, req ProviderCompletionRequest) (*ProviderCompletionResponse, error) {
	return &ProviderCompletionResponse{
		ID:      "test-id",
		Model:   req.Model,
		Content: "mock response",
		Usage: ProviderUsage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}, nil
}

func (m *mockGenericProvider) CompleteStream(ctx context.Context, req ProviderCompletionRequest) (<-chan ProviderStreamChunk, <-chan error, error) {
	chunkChan := make(chan ProviderStreamChunk, 1)
	errChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		chunkChan <- ProviderStreamChunk{
			Delta:        "mock response",
			Content:      "mock response",
			FinishReason: "stop",
			Usage: &ProviderUsage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}
	}()

	return chunkChan, errChan, nil
}

func (m *mockGenericProvider) GetSupportedModels() []string {
	return []string{"mock-model-1", "mock-model-2"}
}

func (m *mockGenericProvider) ValidateModel(model string) error {
	for _, m := range m.GetSupportedModels() {
		if m == model {
			return nil
		}
	}
	return ErrModelNotFound
}

func (m *mockGenericProvider) GetModel() string {
	return m.model
}

func (m *mockGenericProvider) SetModel(model string) {
	m.model = model
}

// TestProviderFactoryCreatesAllProviders tests that the factory can create all provider types
func TestProviderFactoryCreatesAllProviders(t *testing.T) {
	tests := []struct {
		name         string
		providerType ProviderType
		setupFactory func() *ProviderFactory
		validateFunc func(t *testing.T, provider interface{})
	}{
		{
			name:         "OpenAI provider",
			providerType: ProviderOpenAI,
			setupFactory: func() *ProviderFactory {
				f := NewProviderFactory()
				f.SetOpenAIConfig(&OpenAIConfig{APIKey: "test-api-key"})
				return f
			},
			validateFunc: func(t *testing.T, p interface{}) {
				provider, ok := p.(*OpenAIProvider)
				if !ok {
					t.Fatalf("Expected *OpenAIProvider, got %T", p)
				}
				if provider.GetModel() != "gpt-4o" {
					t.Errorf("Expected model gpt-4o, got %s", provider.GetModel())
				}
			},
		},
		{
			name:         "OpenRouter provider",
			providerType: ProviderOpenRouter,
			setupFactory: func() *ProviderFactory {
				f := NewProviderFactory()
				f.SetOpenRouterConfig(&OpenRouterConfig{APIKey: "test-api-key"})
				return f
			},
			validateFunc: func(t *testing.T, p interface{}) {
				provider, ok := p.(*OpenRouterProvider)
				if !ok {
					t.Fatalf("Expected *OpenRouterProvider, got %T", p)
				}
				if provider.GetModel() != string(OpenRouterDefaultModel) {
					t.Errorf("Expected model %s, got %s", OpenRouterDefaultModel, provider.GetModel())
				}
			},
		},
		{
			name:         "Gemini provider",
			providerType: ProviderGemini,
			setupFactory: func() *ProviderFactory {
				f := NewProviderFactory()
				f.SetGeminiConfig(&GeminiConfig{APIKey: "test-api-key"})
				return f
			},
			validateFunc: func(t *testing.T, p interface{}) {
				provider, ok := p.(*GeminiProvider)
				if !ok {
					t.Fatalf("Expected *GeminiProvider, got %T", p)
				}
				if provider.GetModel() != string(Gemini15Pro) {
					t.Errorf("Expected model %s, got %s", Gemini15Pro, provider.GetModel())
				}
			},
		},
		{
			name:         "Ollama provider",
			providerType: ProviderOllama,
			setupFactory: func() *ProviderFactory {
				f := NewProviderFactory()
				f.SetOllamaConfig(&OllamaConfig{BaseURL: "http://localhost:11434"})
				return f
			},
			validateFunc: func(t *testing.T, p interface{}) {
				provider, ok := p.(*OllamaProvider)
				if !ok {
					t.Fatalf("Expected *OllamaProvider, got %T", p)
				}
				if provider.GetBaseURL() != "http://localhost:11434" {
					t.Errorf("Expected base URL http://localhost:11434, got %s", provider.GetBaseURL())
				}
			},
		},
		{
			name:         "LM Studio provider",
			providerType: ProviderLMStudio,
			setupFactory: func() *ProviderFactory {
				f := NewProviderFactory()
				f.SetLMStudioConfig(&LMStudioConfig{BaseURL: "http://localhost:1234"})
				return f
			},
			validateFunc: func(t *testing.T, p interface{}) {
				provider, ok := p.(*LMStudioProvider)
				if !ok {
					t.Fatalf("Expected *LMStudioProvider, got %T", p)
				}
				if provider.GetBaseURL() != "http://localhost:1234" {
					t.Errorf("Expected base URL http://localhost:1234, got %s", provider.GetBaseURL())
				}
			},
		},
		{
			name:         "Anthropic provider",
			providerType: ProviderAnthropic,
			setupFactory: func() *ProviderFactory {
				f := NewProviderFactory()
				f.SetAnthropicConfig(&AnthropicProviderConfig{APIKey: "test-api-key"})
				return f
			},
			validateFunc: func(t *testing.T, p interface{}) {
				provider, ok := p.(*AnthropicProvider)
				if !ok {
					t.Fatalf("Expected *AnthropicProvider, got %T", p)
				}
				if provider.GetModel() != Claude35Sonnet {
					t.Errorf("Expected model %s, got %s", Claude35Sonnet, provider.GetModel())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := tt.setupFactory()
			provider, err := factory.CreateProvider(tt.providerType)
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}

			tt.validateFunc(t, provider)
		})
	}
}

// TestProviderErrorHandling tests error handling across providers
func TestProviderErrorHandling(t *testing.T) {
	t.Run("OpenAI rate limit error", func(t *testing.T) {
		server := mockProviderServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			resp := map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Rate limit exceeded",
					"type":    "rate_limit_error",
					"code":    "rate_limit_exceeded",
				},
			}
			json.NewEncoder(w).Encode(resp)
		})

		provider, err := NewOpenAIProvider(OpenAIConfig{
			APIKey:  "test-api-key",
			BaseURL: server.URL,
		})
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		_, err = provider.Complete(context.Background(), OpenAICompletionRequest{
			Model: "gpt-4o",
			Messages: []OpenAIMessage{
				{Role: "user", Content: "Hello"},
			},
		})

		if err == nil {
			t.Fatal("Expected error for rate limit")
		}

		if !strings.Contains(err.Error(), "rate limit") {
			t.Errorf("Expected rate limit error, got: %v", err)
		}
	})

	t.Run("OpenAI invalid API key error", func(t *testing.T) {
		server := mockProviderServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			resp := map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Invalid API key",
					"type":    "invalid_request_error",
					"code":    "invalid_api_key",
				},
			}
			json.NewEncoder(w).Encode(resp)
		})

		provider, err := NewOpenAIProvider(OpenAIConfig{
			APIKey:  "test-api-key",
			BaseURL: server.URL,
		})
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		_, err = provider.Complete(context.Background(), OpenAICompletionRequest{
			Model: "gpt-4o",
			Messages: []OpenAIMessage{
				{Role: "user", Content: "Hello"},
			},
		})

		if err == nil {
			t.Fatal("Expected error for invalid API key")
		}

		if !strings.Contains(err.Error(), "API key") {
			t.Errorf("Expected API key error, got: %v", err)
		}
	})

	t.Run("OpenRouter provider unavailable error", func(t *testing.T) {
		server := mockProviderServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			resp := map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Provider is unavailable",
					"type":    "provider_error",
					"code":    "provider_unavailable",
				},
			}
			json.NewEncoder(w).Encode(resp)
		})

		provider, err := NewOpenRouterProvider(OpenRouterConfig{
			APIKey:  "test-api-key",
			BaseURL: server.URL,
		})
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		_, err = provider.Complete(context.Background(), CompletionRequest{
			Model: "anthropic/claude-3.5-sonnet",
			Messages: []OpenRouterMessage{
				{Role: "user", Content: "Hello"},
			},
		})

		if err == nil {
			t.Fatal("Expected error for provider unavailable")
		}

		if !strings.Contains(err.Error(), "unavailable") {
			t.Errorf("Expected provider unavailable error, got: %v", err)
		}
	})
}

// TestProviderContextCancellation tests context cancellation handling
func TestProviderContextCancellation(t *testing.T) {
	t.Run("OpenAI context cancellation", func(t *testing.T) {
		server := mockProviderServer(t, func(w http.ResponseWriter, r *http.Request) {
			// Simulate slow response
			time.Sleep(100 * time.Millisecond)
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `data: {"id":"test","choices":[{"delta":{"content":"Hello"}}]}`)
			w.(http.Flusher).Flush()
			time.Sleep(200 * time.Millisecond)
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
		cancel() // Cancel immediately

		_, err = provider.Complete(ctx, OpenAICompletionRequest{
			Model: "gpt-4o",
			Messages: []OpenAIMessage{
				{Role: "user", Content: "Hello"},
			},
		})

		if err == nil {
			t.Fatal("Expected error for cancelled context")
		}
	})
}

// TestProviderTypes tests provider type constants
func TestProviderTypes(t *testing.T) {
	tests := []struct {
		providerType ProviderType
		expected     string
	}{
		{ProviderAnthropic, "anthropic"},
		{ProviderOpenAI, "openai"},
		{ProviderOpenRouter, "openrouter"},
		{ProviderGemini, "gemini"},
		{ProviderBedrock, "bedrock"},
		{ProviderOllama, "ollama"},
		{ProviderLMStudio, "lmstudio"},
	}

	for _, tt := range tests {
		t.Run(string(tt.providerType), func(t *testing.T) {
			if string(tt.providerType) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.providerType)
			}
		})
	}
}

// TestProviderUsageCalculations tests usage calculation helpers
func TestProviderUsageCalculations(t *testing.T) {
	t.Run("Total tokens calculation", func(t *testing.T) {
		usage := ProviderUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150, // Must be explicitly set
		}
		expected := 150
		if usage.TotalTokens != expected {
			t.Errorf("Expected %d total tokens, got %d", expected, usage.TotalTokens)
		}
	})

	t.Run("Zero usage", func(t *testing.T) {
		usage := ProviderUsage{}
		if usage.TotalTokens != 0 {
			t.Errorf("Expected 0 total tokens, got %d", usage.TotalTokens)
		}
	})
}

// TestProviderMessageStruct tests the ProviderMessage struct
func TestProviderMessageStruct(t *testing.T) {
	msg := ProviderMessage{
		Role:    "user",
		Content: "Hello",
	}

	if msg.Role != "user" {
		t.Errorf("Expected role 'user', got %s", msg.Role)
	}
	if msg.Content != "Hello" {
		t.Errorf("Expected content 'Hello', got %s", msg.Content)
	}

	// Test JSON marshaling
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	var decoded ProviderMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	if decoded.Role != msg.Role {
		t.Errorf("Expected role %s, got %s", msg.Role, decoded.Role)
	}
	if decoded.Content != msg.Content {
		t.Errorf("Expected content %s, got %s", msg.Content, decoded.Content)
	}
}

// TestProviderCompletionRequestStruct tests the ProviderCompletionRequest struct
func TestProviderCompletionRequestStruct(t *testing.T) {
	req := ProviderCompletionRequest{
		Model:       "gpt-4o",
		Messages:    []ProviderMessage{{Role: "user", Content: "Hello"}},
		Temperature: 0.7,
		MaxTokens:   100,
		TopP:        0.9,
		Stream:      true,
	}

	if req.Model != "gpt-4o" {
		t.Errorf("Expected model gpt-4o, got %s", req.Model)
	}
	if len(req.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(req.Messages))
	}
	if req.Temperature != 0.7 {
		t.Errorf("Expected temperature 0.7, got %f", req.Temperature)
	}
	if req.MaxTokens != 100 {
		t.Errorf("Expected max tokens 100, got %d", req.MaxTokens)
	}
	if req.TopP != 0.9 {
		t.Errorf("Expected top_p 0.9, got %f", req.TopP)
	}
	if !req.Stream {
		t.Error("Expected stream to be true")
	}
}

// TestProviderCompletionResponseStruct tests the ProviderCompletionResponse struct
func TestProviderCompletionResponseStruct(t *testing.T) {
	resp := ProviderCompletionResponse{
		ID:      "test-id",
		Model:   "gpt-4o",
		Content: "Hello!",
		Usage: ProviderUsage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	if resp.ID != "test-id" {
		t.Errorf("Expected ID test-id, got %s", resp.ID)
	}
	if resp.Model != "gpt-4o" {
		t.Errorf("Expected model gpt-4o, got %s", resp.Model)
	}
	if resp.Content != "Hello!" {
		t.Errorf("Expected content 'Hello!', got %s", resp.Content)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("Expected 15 total tokens, got %d", resp.Usage.TotalTokens)
	}
}

// TestProviderStreamChunkStruct tests the ProviderStreamChunk struct
func TestProviderStreamChunkStruct(t *testing.T) {
	usage := &ProviderUsage{
		PromptTokens:     10,
		CompletionTokens: 5,
		TotalTokens:      15,
	}

	chunk := ProviderStreamChunk{
		Delta:        "Hello",
		Content:      "Hello",
		FinishReason: "stop",
		Usage:        usage,
	}

	if chunk.Delta != "Hello" {
		t.Errorf("Expected delta 'Hello', got %s", chunk.Delta)
	}
	if chunk.Content != "Hello" {
		t.Errorf("Expected content 'Hello', got %s", chunk.Content)
	}
	if chunk.FinishReason != "stop" {
		t.Errorf("Expected finish_reason 'stop', got %s", chunk.FinishReason)
	}
	if chunk.Usage == nil {
		t.Fatal("Expected usage to not be nil")
	}
	if chunk.Usage.TotalTokens != 15 {
		t.Errorf("Expected 15 total tokens, got %d", chunk.Usage.TotalTokens)
	}
}
