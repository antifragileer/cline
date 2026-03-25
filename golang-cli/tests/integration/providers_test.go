// Package integration provides integration tests for the Go CLI.
// This package tests API provider implementations.
package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/cline/cline/golang-cli/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOpenAIProvider tests the OpenAI provider implementation
func TestOpenAIProvider(t *testing.T) {
	t.Run("creates provider with valid config", func(t *testing.T) {
		config := &api.ProviderConfig{
			APIKey:   "test-api-key",
			Model:    "gpt-4",
			Endpoint: "https://api.openai.com/v1",
		}

		provider, err := api.NewOpenAIProvider(config)
		require.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("fails with missing API key", func(t *testing.T) {
		config := &api.ProviderConfig{
			APIKey:   "",
			Model:    "gpt-4",
			Endpoint: "https://api.openai.com/v1",
		}

		_, err := api.NewOpenAIProvider(config)
		assert.Error(t, err)
	})

	t.Run("sends correct request format", func(t *testing.T) {
		// Create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request method
			assert.Equal(t, "POST", r.Method)

			// Verify headers
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

			// Verify path
			assert.Equal(t, "/v1/chat/completions", r.URL.Path)

			// Parse request body
			var reqBody map[string]interface{}
			err := json.NewDecoder(r.Body).Decode(&reqBody)
			require.NoError(t, err)

			// Verify request structure
			assert.Contains(t, reqBody, "model")
			assert.Contains(t, reqBody, "messages")

			// Return mock response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":      "test-id",
				"object":  "chat.completion",
				"created": time.Now().Unix(),
				"model":   "gpt-4",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": "Test response",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     10,
					"completion_tokens": 5,
					"total_tokens":      15,
				},
			})
		}))
		defer server.Close()

		config := &api.ProviderConfig{
			APIKey:   "test-api-key",
			Model:    "gpt-4",
			Endpoint: server.URL,
		}

		provider, err := api.NewOpenAIProvider(config)
		require.NoError(t, err)

		resp, err := provider.Complete("Test prompt")
		require.NoError(t, err)
		assert.Equal(t, "Test response", resp)
	})

	t.Run("handles API errors", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Invalid API key",
					"type":    "authentication_error",
				},
			})
		}))
		defer server.Close()

		config := &api.ProviderConfig{
			APIKey:   "invalid-key",
			Model:    "gpt-4",
			Endpoint: server.URL,
		}

		provider, err := api.NewOpenAIProvider(config)
		require.NoError(t, err)

		_, err = provider.Complete("Test prompt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})
}

// TestAnthropicProvider tests the Anthropic provider implementation
func TestAnthropicProvider(t *testing.T) {
	t.Run("creates provider with valid config", func(t *testing.T) {
		config := &api.ProviderConfig{
			APIKey:   "test-api-key",
			Model:    "claude-3-opus-20240229",
			Endpoint: "https://api.anthropic.com/v1",
		}

		provider, err := api.NewAnthropicProvider(config)
		require.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("sends correct request format", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "test-api-key", r.Header.Get("x-api-key"))
			assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))

			var reqBody map[string]interface{}
			err := json.NewDecoder(r.Body).Decode(&reqBody)
			require.NoError(t, err)

			assert.Contains(t, reqBody, "model")
			assert.Contains(t, reqBody, "messages")
			assert.Contains(t, reqBody, "max_tokens")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":           "test-id",
				"type":         "message",
				"role":         "assistant",
				"content":      []map[string]string{{"type": "text", "text": "Test response"}},
				"model":        "claude-3-opus-20240229",
				"stop_reason":  "end_turn",
				"usage":        map[string]int{"input_tokens": 10, "output_tokens": 5},
			})
		}))
		defer server.Close()

		config := &api.ProviderConfig{
			APIKey:   "test-api-key",
			Model:    "claude-3-opus-20240229",
			Endpoint: server.URL,
		}

		provider, err := api.NewAnthropicProvider(config)
		require.NoError(t, err)

		resp, err := provider.Complete("Test prompt")
		require.NoError(t, err)
		assert.Equal(t, "Test response", resp)
	})
}

// TestOpenRouterProvider tests the OpenRouter provider implementation
func TestOpenRouterProvider(t *testing.T) {
	t.Run("creates provider with valid config", func(t *testing.T) {
		config := &api.ProviderConfig{
			APIKey:   "test-api-key",
			Model:    "anthropic/claude-3-opus",
			Endpoint: "https://openrouter.ai/api/v1",
		}

		provider, err := api.NewOpenRouterProvider(config)
		require.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("sends correct headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.NotEmpty(t, r.Header.Get("HTTP-Referer"))
			assert.NotEmpty(t, r.Header.Get("X-Title"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":      "test-id",
				"choices": []map[string]interface{}{{"message": map[string]string{"content": "Test"}}},
			})
		}))
		defer server.Close()

		config := &api.ProviderConfig{
			APIKey:   "test-api-key",
			Model:    "anthropic/claude-3-opus",
			Endpoint: server.URL,
		}

		provider, err := api.NewOpenRouterProvider(config)
		require.NoError(t, err)

		_, err = provider.Complete("Test")
		require.NoError(t, err)
	})
}

// TestProviderStreaming tests streaming capabilities
func TestProviderStreaming(t *testing.T) {
	t.Run("OpenAI streaming response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check for streaming flag
			var reqBody map[string]interface{}
			json.NewDecoder(r.Body).Decode(&reqBody)
			assert.True(t, reqBody["stream"].(bool))

			// Set up SSE response
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			// Write SSE events
			events := []string{
				`data: {"id":"1","object":"chat.completion.chunk","choices":[{"delta":{"content":"Hello"}}]}`,
				`data: {"id":"1","object":"chat.completion.chunk","choices":[{"delta":{"content":" world"}}]}`,
				`data: [DONE]`,
			}

			for _, event := range events {
				w.Write([]byte(event + "\n\n"))
				w.(http.Flusher).Flush()
				time.Sleep(10 * time.Millisecond)
			}
		}))
		defer server.Close()

		config := &api.ProviderConfig{
			APIKey:   "test-api-key",
			Model:    "gpt-4",
			Endpoint: server.URL,
		}

		provider, err := api.NewOpenAIProvider(config)
		require.NoError(t, err)

		// Test streaming
		chunks, err := provider.CompleteStream("Test prompt")
		require.NoError(t, err)

		var result string
		for chunk := range chunks {
			result += chunk
		}

		assert.Equal(t, "Hello world", result)
	})
}

// TestProviderErrorHandling tests error handling across providers
func TestProviderErrorHandling(t *testing.T) {
	t.Run("handles network errors", func(t *testing.T) {
		config := &api.ProviderConfig{
			APIKey:   "test-api-key",
			Model:    "gpt-4",
			Endpoint: "http://localhost:99999", // Invalid port
		}

		provider, err := api.NewOpenAIProvider(config)
		require.NoError(t, err)

		_, err = provider.Complete("Test")
		assert.Error(t, err)
	})

	t.Run("handles timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		config := &api.ProviderConfig{
			APIKey:       "test-api-key",
			Model:        "gpt-4",
			Endpoint:     server.URL,
			Timeout:      100 * time.Millisecond,
		}

		provider, err := api.NewOpenAIProvider(config)
		require.NoError(t, err)

		_, err = provider.Complete("Test")
		assert.Error(t, err)
	})

	t.Run("handles rate limiting", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Rate limit exceeded",
					"type":    "rate_limit_error",
				},
			})
		}))
		defer server.Close()

		config := &api.ProviderConfig{
			APIKey:   "test-api-key",
			Model:    "gpt-4",
			Endpoint: server.URL,
		}

		provider, err := api.NewOpenAIProvider(config)
		require.NoError(t, err)

		_, err = provider.Complete("Test")
		assert.Error(t, err)
	})
}

// TestProviderConfiguration tests provider configuration loading
func TestProviderConfiguration(t *testing.T) {
	t.Run("loads from environment variables", func(t *testing.T) {
		// Set environment variables
		os.Setenv("OPENAI_API_KEY", "env-api-key")
		os.Setenv("OPENAI_MODEL", "gpt-4-turbo")
		defer func() {
			os.Unsetenv("OPENAI_API_KEY")
			os.Unsetenv("OPENAI_MODEL")
		}()

		config := api.LoadProviderConfigFromEnv("openai")
		assert.Equal(t, "env-api-key", config.APIKey)
		assert.Equal(t, "gpt-4-turbo", config.Model)
	})

	t.Run("loads from config file", func(t *testing.T) {
		// Create temporary config file
		tempDir, err := os.MkdirTemp("", "config-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		configPath := tempDir + "/providers.json"
		configData := map[string]interface{}{
			"openai": map[string]string{
				"api_key":  "file-api-key",
				"model":    "gpt-4",
				"endpoint": "https://custom.openai.com",
			},
		}

		data, _ := json.Marshal(configData)
		err = os.WriteFile(configPath, data, 0644)
		require.NoError(t, err)

		config := api.LoadProviderConfigFromFile(configPath, "openai")
		assert.Equal(t, "file-api-key", config.APIKey)
		assert.Equal(t, "gpt-4", config.Model)
	})
}

// BenchmarkProviderComplete benchmarks provider completion
func BenchmarkProviderComplete(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "Benchmark response"}},
			},
		})
	}))
	defer server.Close()

	config := &api.ProviderConfig{
		APIKey:   "test-api-key",
		Model:    "gpt-4",
		Endpoint: server.URL,
	}

	provider, err := api.NewOpenAIProvider(config)
	if err != nil {
		b.Fatalf("Failed to create provider: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := provider.Complete("Benchmark prompt")
		if err != nil {
			b.Errorf("Complete failed: %v", err)
		}
	}
}