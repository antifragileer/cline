// Package api provides model discovery functionality for API providers
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ModelDiscovery provides functionality to fetch available models from providers
type ModelDiscovery struct {
	httpClient *http.Client
}

// NewModelDiscovery creates a new model discovery client
func NewModelDiscovery() *ModelDiscovery {
	return &ModelDiscovery{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewModelDiscoveryWithClient creates a model discovery client with a custom HTTP client
func NewModelDiscoveryWithClient(client *http.Client) *ModelDiscovery {
	return &ModelDiscovery{
		httpClient: client,
	}
}

// DiscoveredModel represents a model discovered from a provider
type DiscoveredModel struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	MaxTokens     int    `json:"max_tokens"`
	ContextWindow int    `json:"context_window"`
}

// ProviderModelResponse represents the response from a provider's models endpoint
type ProviderModelResponse struct {
	Data []DiscoveredModel `json:"data"`
}

// FetchOpenAIModels fetches available models from OpenAI
func (md *ModelDiscovery) FetchOpenAIModels(apiKey string) ([]DiscoveredModel, error) {
	if apiKey == "" {
		return nil, ErrOpenAIInvalidAPIKey
	}

	req, err := http.NewRequest("GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := md.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch models: HTTP %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Filter for chat completion models
	var models []DiscoveredModel
	for _, m := range result.Data {
		// Only include GPT models
		if isGPTModel(m.ID) {
			models = append(models, DiscoveredModel{
				ID:   m.ID,
				Name: m.ID,
			})
		}
	}

	return models, nil
}

// FetchAnthropicModels returns the list of known Anthropic models
// Note: Anthropic doesn't have a public models API, so we return the known models
func (md *ModelDiscovery) FetchAnthropicModels(apiKey string) ([]DiscoveredModel, error) {
	// Test the API key first with a simple request
	if apiKey == "" {
		return nil, ErrInvalidAPIKey
	}

	// Since Anthropic doesn't expose a models endpoint, we return known models
	// but verify the API key is valid
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	provider, err := NewAnthropicProvider(WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	// Make a simple request to validate the key
	_, err = provider.CreateMessage(ctx, AnthropicMessagesRequest{
		Model:     Claude35Haiku, // Use smallest model for validation
		MaxTokens: 1,
		Messages:  []AnthropicMessage{CreateTextMessage(RoleUser, "Hi")},
	})

	if err != nil {
		// API key validation failed
		return nil, fmt.Errorf("invalid API key: %w", err)
	}

	// Return known Anthropic models
	return []DiscoveredModel{
		{ID: string(Claude37Sonnet), Name: "Claude 3.7 Sonnet", MaxTokens: 8192, ContextWindow: 200000},
		{ID: string(Claude35Sonnet), Name: "Claude 3.5 Sonnet", MaxTokens: 8192, ContextWindow: 200000},
		{ID: string(Claude35Haiku), Name: "Claude 3.5 Haiku", MaxTokens: 4096, ContextWindow: 200000},
		{ID: string(Claude3Opus), Name: "Claude 3 Opus", MaxTokens: 4096, ContextWindow: 200000},
		{ID: string(Claude3Sonnet), Name: "Claude 3 Sonnet", MaxTokens: 4096, ContextWindow: 200000},
		{ID: string(Claude3Haiku), Name: "Claude 3 Haiku", MaxTokens: 4096, ContextWindow: 200000},
	}, nil
}

// FetchOpenRouterModels fetches available models from OpenRouter
func (md *ModelDiscovery) FetchOpenRouterModels(apiKey string) ([]DiscoveredModel, error) {
	req, err := http.NewRequest("GET", "https://openrouter.ai/api/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := md.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch models: HTTP %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			Description   string `json:"description"`
			ContextLength int    `json:"context_length"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var models []DiscoveredModel
	for _, m := range result.Data {
		models = append(models, DiscoveredModel{
			ID:            m.ID,
			Name:          m.Name,
			Description:   m.Description,
			ContextWindow: m.ContextLength,
		})
	}

	return models, nil
}

// FetchGeminiModels fetches available models from Google Gemini
func (md *ModelDiscovery) FetchGeminiModels(apiKey string) ([]DiscoveredModel, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models?key=%s", apiKey)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := md.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch models: HTTP %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Models []struct {
			Name             string `json:"name"`
			DisplayName      string `json:"displayName"`
			Description      string `json:"description"`
			InputTokenLimit  int    `json:"inputTokenLimit"`
			OutputTokenLimit int    `json:"outputTokenLimit"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var models []DiscoveredModel
	for _, m := range result.Models {
		// Extract model ID from name (remove "models/" prefix)
		modelID := m.Name
		if len(modelID) > 7 && modelID[:7] == "models/" {
			modelID = modelID[7:]
		}

		models = append(models, DiscoveredModel{
			ID:            modelID,
			Name:          m.DisplayName,
			Description:   m.Description,
			MaxTokens:     m.OutputTokenLimit,
			ContextWindow: m.InputTokenLimit,
		})
	}

	return models, nil
}

// FetchOllamaModels fetches available models from a local Ollama instance
func (md *ModelDiscovery) FetchOllamaModels(baseURL string) ([]DiscoveredModel, error) {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	url := baseURL + "/api/tags"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := md.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch models: HTTP %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Models []struct {
			Name       string `json:"name"`
			Model      string `json:"model"`
			ModifiedAt string `json:"modified_at"`
			Size       int64  `json:"size"`
			Digest     string `json:"digest"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var models []DiscoveredModel
	for _, m := range result.Models {
		models = append(models, DiscoveredModel{
			ID:   m.Name,
			Name: m.Name,
		})
	}

	return models, nil
}

// isGPTModel checks if a model ID is a GPT model
func isGPTModel(modelID string) bool {
	// Include GPT models
	if len(modelID) >= 3 && modelID[:3] == "gpt" {
		return true
	}
	// Include o1 models
	if len(modelID) >= 2 && modelID[:2] == "o1" {
		return true
	}
	// Include o3 models
	if len(modelID) >= 2 && modelID[:2] == "o3" {
		return true
	}
	return false
}

// ValidateProviderAuth validates authentication with a provider
func (md *ModelDiscovery) ValidateProviderAuth(providerType ProviderType, apiKey, baseURL string) error {
	switch providerType {
	case ProviderAnthropic:
		return md.validateAnthropicAuth(apiKey)
	case ProviderOpenAI:
		return md.validateOpenAIAuth(apiKey)
	case ProviderOpenRouter:
		return md.validateOpenRouterAuth(apiKey)
	case ProviderGemini:
		return md.validateGeminiAuth(apiKey)
	case ProviderOllama:
		return md.validateOllamaAuth(baseURL)
	case ProviderLMStudio:
		return md.validateLMStudioAuth(baseURL)
	default:
		return fmt.Errorf("unsupported provider for validation: %s", providerType)
	}
}

func (md *ModelDiscovery) validateAnthropicAuth(apiKey string) error {
	if apiKey == "" {
		return ErrInvalidAPIKey
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	provider, err := NewAnthropicProvider(WithAPIKey(apiKey))
	if err != nil {
		return err
	}

	// Make a minimal request to validate the key
	_, err = provider.CreateMessage(ctx, AnthropicMessagesRequest{
		Model:     Claude35Haiku,
		MaxTokens: 1,
		Messages:  []AnthropicMessage{CreateTextMessage(RoleUser, "Hi")},
	})

	if err != nil {
		// Check if it's an auth error
		apiErr, ok := IsAnthropicAPIError(err)
		if ok && apiErr.Type == ErrorTypeAuthentication {
			return fmt.Errorf("%w: %s", ErrInvalidAPIKey, apiErr.Message)
		}
		// For other errors, the key might still be valid
		// We only fail on authentication errors
	}

	return nil
}

func (md *ModelDiscovery) validateOpenAIAuth(apiKey string) error {
	if apiKey == "" {
		return ErrOpenAIInvalidAPIKey
	}

	req, err := http.NewRequest("GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := md.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate API key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return ErrOpenAIInvalidAPIKey
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("validation failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

func (md *ModelDiscovery) validateOpenRouterAuth(apiKey string) error {
	// OpenRouter doesn't require an API key for model listing
	// but we can validate by fetching models
	_, err := md.FetchOpenRouterModels(apiKey)
	return err
}

func (md *ModelDiscovery) validateGeminiAuth(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}

	// Try to fetch models as validation
	_, err := md.FetchGeminiModels(apiKey)
	return err
}

func (md *ModelDiscovery) validateOllamaAuth(baseURL string) error {
	// Ollama doesn't use API keys, just check connectivity
	_, err := md.FetchOllamaModels(baseURL)
	return err
}

func (md *ModelDiscovery) validateLMStudioAuth(baseURL string) error {
	if baseURL == "" {
		baseURL = "http://localhost:1234"
	}

	// LM Studio uses OpenAI-compatible API
	req, err := http.NewRequest("GET", baseURL+"/v1/models", nil)
	if err != nil {
		return err
	}

	resp, err := md.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to LM Studio: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("LM Studio returned HTTP %d", resp.StatusCode)
	}

	return nil
}
