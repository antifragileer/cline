// Package api provides API provider implementations for the Cline CLI.
package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenRouter errors
var (
	ErrInvalidAPIKey       = errors.New("invalid API key")
	ErrRateLimitExceeded   = errors.New("rate limit exceeded")
	ErrInvalidRequest      = errors.New("invalid request")
	ErrInvalidResponse     = errors.New("invalid response from API")
	ErrProviderError       = errors.New("provider error")
	ErrContextCanceled     = errors.New("request canceled")
	ErrModelNotFound       = errors.New("model not found")
	ErrProviderUnavailable = errors.New("provider unavailable")
)

// DefaultOpenRouterBaseURL is the default base URL for the OpenRouter API
const DefaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"

// DefaultOpenRouterTimeout is the default timeout for OpenRouter requests
const DefaultOpenRouterTimeout = 120 * time.Second

// OpenRouterModel represents supported OpenRouter models
type OpenRouterModel string

// Common OpenRouter models
const (
	// Anthropic models
	OpenRouterClaude35Sonnet OpenRouterModel = "anthropic/claude-3.5-sonnet"
	OpenRouterClaude3Opus    OpenRouterModel = "anthropic/claude-3-opus"
	OpenRouterClaude3Sonnet  OpenRouterModel = "anthropic/claude-3-sonnet"
	OpenRouterClaude3Haiku   OpenRouterModel = "anthropic/claude-3-haiku"

	// OpenAI models
	OpenRouterGPT4o       OpenRouterModel = "openai/gpt-4o"
	OpenRouterGPT4Turbo   OpenRouterModel = "openai/gpt-4-turbo"
	OpenRouterGPT35Turbo  OpenRouterModel = "openai/gpt-3.5-turbo"

	// Meta models
	OpenRouterLlama370B OpenRouterModel = "meta-llama/llama-3-70b-instruct"
	OpenRouterLlama38B  OpenRouterModel = "meta-llama/llama-3-8b-instruct"

	// Google models
	OpenRouterGeminiPro OpenRouterModel = "google/gemini-pro"
	OpenRouterGeminiFlash OpenRouterModel = "google/gemini-flash-1.5"

	// Mistral models
	OpenRouterMistralLarge OpenRouterModel = "mistralai/mistral-large"
	OpenRouterMistralMedium OpenRouterModel = "mistralai/mistral-medium"
)

// openRouterModels is the list of supported OpenRouter models
var openRouterModels = []string{
	string(OpenRouterClaude35Sonnet),
	string(OpenRouterClaude3Opus),
	string(OpenRouterClaude3Sonnet),
	string(OpenRouterClaude3Haiku),
	string(OpenRouterGPT4o),
	string(OpenRouterGPT4Turbo),
	string(OpenRouterGPT35Turbo),
	string(OpenRouterLlama370B),
	string(OpenRouterLlama38B),
	string(OpenRouterGeminiPro),
	string(OpenRouterGeminiFlash),
	string(OpenRouterMistralLarge),
	string(OpenRouterMistralMedium),
}

// OpenRouterMessage represents a message in the conversation for OpenRouter
type OpenRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenRouterUsage represents token usage for an OpenRouter request
type OpenRouterUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// CompletionRequest represents a request for chat completion
type CompletionRequest struct {
	Model       string
	Messages    []OpenRouterMessage
	Temperature float64
	MaxTokens   int
	TopP        float64
	TopK        int
	// Provider preferences for model routing
	Provider *ProviderPreferences
	// Fallback models if primary is unavailable
	FallbackModels []string
}

// ProviderPreferences configures provider routing behavior
type ProviderPreferences struct {
	// Order of provider preference (e.g., ["Anthropic", "OpenAI"])
	Order []string `json:"order,omitempty"`
	// Allow fallbacks to other providers
	AllowFallbacks bool `json:"allow_fallbacks,omitempty"`
	// Require specific providers only
	RequireParameters bool `json:"require_parameters,omitempty"`
	// Data collection policy
	DataCollection string `json:"data_collection,omitempty"`
	// Ignore providers with low context windows
	IgnoreLowContextWindow bool `json:"ignore_low_context_window,omitempty"`
	// Quantization preference
	Quantizations []string `json:"quantizations,omitempty"`
}

// CompletionResponse represents a response from the completion API
type CompletionResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Content string `json:"content"`
	Usage   Usage  `json:"usage"`
}

// StreamChunk represents a chunk in a streaming response
type StreamChunk struct {
	Delta        string `json:"delta,omitempty"`
	Content      string `json:"content,omitempty"`
	FinishReason string `json:"finish_reason,omitempty"`
	Usage        *Usage `json:"usage,omitempty"`
}

// openRouterRequest represents the request body for OpenRouter chat completions
type openRouterRequest struct {
	Model       string               `json:"model"`
	Messages    []openRouterMessage  `json:"messages"`
	Temperature float64              `json:"temperature,omitempty"`
	MaxTokens   int                  `json:"max_tokens,omitempty"`
	TopP        float64              `json:"top_p,omitempty"`
	TopK        int                  `json:"top_k,omitempty"`
	Stream      bool                 `json:"stream"`
	Provider    *ProviderPreferences `json:"provider,omitempty"`
	// Models for fallback
	Models []string `json:"models,omitempty"`
	// Include provider info in response
	IncludeReasoning bool `json:"include_reasoning,omitempty"`
}

// openRouterMessage represents a message in OpenRouter format
type openRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openRouterResponse represents a non-streaming response from OpenRouter
type openRouterResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *openRouterError `json:"error,omitempty"`
}

// openRouterStreamResponse represents a streaming chunk from OpenRouter
type openRouterStreamResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// openRouterError represents an error response from OpenRouter
type openRouterError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
	Param   string `json:"param"`
}

// Error implements the error interface
func (e *openRouterError) Error() string {
	return fmt.Sprintf("openrouter error: %s (type: %s, code: %s)", e.Message, e.Type, e.Code)
}

// OpenRouterConfig contains configuration for the OpenRouter provider
type OpenRouterConfig struct {
	// APIKey is the OpenRouter API key (required)
	APIKey string

	// BaseURL is the base URL for the API (defaults to DefaultOpenRouterBaseURL)
	BaseURL string

	// HTTPClient is the HTTP client to use (defaults to http.DefaultClient with timeout)
	HTTPClient *http.Client

	// DefaultProviderPreferences sets default provider routing preferences
	DefaultProviderPreferences *ProviderPreferences

	// CustomHeaders are additional headers to include in requests
	CustomHeaders map[string]string

	// AppName is the application name for the X-Title header
	AppName string

	// SiteURL is the site URL for the HTTP-Referer header
	SiteURL string
}

// OpenRouterProvider implements the Provider interface for OpenRouter
type OpenRouterProvider struct {
	config     OpenRouterConfig
	httpClient *http.Client
	baseURL    string
	model      OpenRouterModel
}

// NewOpenRouterProvider creates a new OpenRouter provider with the given configuration
func NewOpenRouterProvider(config OpenRouterConfig) (*OpenRouterProvider, error) {
	if config.APIKey == "" {
		return nil, ErrInvalidAPIKey
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: DefaultOpenRouterTimeout,
		}
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = DefaultOpenRouterBaseURL
	}

	return &OpenRouterProvider{
		config:     config,
		httpClient: httpClient,
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		model:      OpenRouterClaude35Sonnet,
	}, nil
}

// Complete sends a chat completion request and returns the full response
func (p *OpenRouterProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if err := p.ValidateModel(req.Model); err != nil {
		return nil, err
	}

	openRouterReq := p.toOpenRouterRequest(req, false)

	body, err := json.Marshal(openRouterReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if err := p.handleErrorResponse(resp); err != nil {
		return nil, err
	}

	var openRouterResp openRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&openRouterResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if openRouterResp.Error != nil {
		return nil, p.convertOpenRouterError(openRouterResp.Error)
	}

	if len(openRouterResp.Choices) == 0 {
		return nil, ErrInvalidResponse
	}

	return &CompletionResponse{
		ID:      openRouterResp.ID,
		Model:   openRouterResp.Model,
		Content: openRouterResp.Choices[0].Message.Content,
		Usage: Usage{
			PromptTokens:     openRouterResp.Usage.PromptTokens,
			CompletionTokens: openRouterResp.Usage.CompletionTokens,
			TotalTokens:      openRouterResp.Usage.TotalTokens,
		},
	}, nil
}

// CompleteStream sends a streaming chat completion request and returns a channel of chunks
func (p *OpenRouterProvider) CompleteStream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, <-chan error, error) {
	if err := p.ValidateModel(req.Model); err != nil {
		return nil, nil, err
	}

	openRouterReq := p.toOpenRouterRequest(req, true)

	body, err := json.Marshal(openRouterReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if err := p.handleErrorResponse(resp); err != nil {
		resp.Body.Close()
		return nil, nil, err
	}

	chunkChan := make(chan StreamChunk, 10)
	errChan := make(chan error, 1)

	go p.processStream(ctx, req, resp.Body, chunkChan, errChan)

	return chunkChan, errChan, nil
}

// processStream processes the SSE stream from OpenRouter
func (p *OpenRouterProvider) processStream(ctx context.Context, req CompletionRequest, body io.ReadCloser, chunkChan chan<- StreamChunk, errChan chan<- error) {
	defer close(chunkChan)
	defer close(errChan)
	defer body.Close()

	scanner := bufio.NewScanner(body)

	// Track accumulated usage for the final chunk
	var accumulatedUsage Usage
	contentBuffer := ""

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			errChan <- ErrContextCanceled
			return
		default:
		}

		line := scanner.Text()

		// SSE format: data: {...}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Stream terminator
		if data == "[DONE]" {
			// Send final chunk with accumulated usage
			if accumulatedUsage.TotalTokens > 0 {
				chunkChan <- StreamChunk{
					Content:      contentBuffer,
					FinishReason: "stop",
					Usage:        &accumulatedUsage,
				}
			}
			return
		}

		var streamResp openRouterStreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			errChan <- fmt.Errorf("failed to decode stream chunk: %w", err)
			return
		}

		if len(streamResp.Choices) == 0 {
			continue
		}

		choice := streamResp.Choices[0]

		// Accumulate usage if provided (OpenRouter may send this in the final chunk)
		if streamResp.Usage != nil {
			accumulatedUsage = Usage{
				PromptTokens:     streamResp.Usage.PromptTokens,
				CompletionTokens: streamResp.Usage.CompletionTokens,
				TotalTokens:      streamResp.Usage.TotalTokens,
			}
		}

		// Accumulate content for usage calculation if not provided
		contentBuffer += choice.Delta.Content

		chunk := StreamChunk{
			Delta:   choice.Delta.Content,
			Content: contentBuffer,
		}

		if choice.FinishReason != "" {
			chunk.FinishReason = choice.FinishReason

			// Estimate usage if not provided by the API
			if accumulatedUsage.TotalTokens == 0 {
				accumulatedUsage = p.estimateUsage(req.Messages, contentBuffer)
			}

			chunk.Usage = &accumulatedUsage
		}

		select {
		case chunkChan <- chunk:
		case <-ctx.Done():
			errChan <- ErrContextCanceled
			return
		}

		// If finished, we can stop processing
		if choice.FinishReason != "" {
			return
		}
	}

	if err := scanner.Err(); err != nil {
		errChan <- fmt.Errorf("stream read error: %w", err)
	}
}

// estimateUsage estimates token usage when the API doesn't provide it
func (p *OpenRouterProvider) estimateUsage(messages []OpenRouterMessage, completion string) Usage {
	// Rough estimation: ~4 characters per token for English text
	promptChars := 0
	for _, msg := range messages {
		promptChars += len(msg.Role) + len(msg.Content)
	}
	completionChars := len(completion)

	promptTokens := promptChars / 4
	if promptTokens < 1 {
		promptTokens = 1
	}

	completionTokens := completionChars / 4
	if completionTokens < 1 {
		completionTokens = 1
	}

	return Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
	}
}

// GetSupportedModels returns a list of supported model IDs
func (p *OpenRouterProvider) GetSupportedModels() []string {
	models := make([]string, len(openRouterModels))
	copy(models, openRouterModels)
	return models
}

// ValidateModel checks if a model is supported
func (p *OpenRouterProvider) ValidateModel(model string) error {
	// OpenRouter supports dynamic model routing with provider/model format
	// Allow any model that follows the provider/model pattern or is in our known list
	
	// Check if it's in our known models list
	for _, m := range openRouterModels {
		if m == model {
			return nil
		}
	}
	
	// Check if it follows the provider/model format
	parts := strings.Split(model, "/")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		// Valid provider/model format
		return nil
	}
	
	return fmt.Errorf("%w: %s", ErrModelNotFound, model)
}

// ParseModelID parses a model ID in provider/model format
func (p *OpenRouterProvider) ParseModelID(modelID string) (provider string, model string, err error) {
	parts := strings.Split(modelID, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid model ID format: %s (expected provider/model)", modelID)
	}
	return parts[0], parts[1], nil
}

// BuildModelID builds a model ID from provider and model name
func (p *OpenRouterProvider) BuildModelID(provider, model string) string {
	return fmt.Sprintf("%s/%s", provider, model)
}

// toOpenRouterRequest converts a CompletionRequest to an OpenRouter-specific request
func (p *OpenRouterProvider) toOpenRouterRequest(req CompletionRequest, stream bool) openRouterRequest {
	messages := make([]openRouterMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openRouterMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	topP := req.TopP
	if topP == 0 {
		topP = 1.0
	}

	// Use provider preferences from request or fall back to defaults
	providerPrefs := req.Provider
	if providerPrefs == nil && p.config.DefaultProviderPreferences != nil {
		providerPrefs = p.config.DefaultProviderPreferences
	}

	// Build models list with fallback models
	models := []string{req.Model}
	if len(req.FallbackModels) > 0 {
		models = append(models, req.FallbackModels...)
	}

	return openRouterRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		TopP:        topP,
		TopK:        req.TopK,
		Stream:      stream,
		Provider:    providerPrefs,
		Models:      models,
	}
}

// setHeaders sets the required HTTP headers for OpenRouter requests
func (p *OpenRouterProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))
	
	// OpenRouter-specific headers
	if p.config.AppName != "" {
		req.Header.Set("X-Title", p.config.AppName)
	}
	if p.config.SiteURL != "" {
		req.Header.Set("HTTP-Referer", p.config.SiteURL)
	}

	// Apply custom headers
	for key, value := range p.config.CustomHeaders {
		req.Header.Set(key, value)
	}
}

// handleErrorResponse handles HTTP error responses
func (p *OpenRouterProvider) handleErrorResponse(resp *http.Response) error {
	if resp.StatusCode < 400 {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d: failed to read error body: %w", resp.StatusCode, err)
	}

	// Try to parse as OpenRouter error
	var errResp struct {
		Error *openRouterError `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != nil {
		return p.convertOpenRouterError(errResp.Error)
	}

	// Fallback to generic error
	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		return ErrRateLimitExceeded
	case http.StatusUnauthorized:
		return ErrInvalidAPIKey
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", ErrInvalidRequest, string(body))
	case http.StatusServiceUnavailable:
		return ErrProviderUnavailable
	default:
		return fmt.Errorf("%w: HTTP %d - %s", ErrProviderError, resp.StatusCode, string(body))
	}
}

// convertOpenRouterError converts an OpenRouter error to the appropriate error type
func (p *OpenRouterProvider) convertOpenRouterError(err *openRouterError) error {
	switch err.Code {
	case "rate_limit_exceeded", "insufficient_quota":
		return fmt.Errorf("%w: %s", ErrRateLimitExceeded, err.Message)
	case "invalid_api_key":
		return fmt.Errorf("%w: %s", ErrInvalidAPIKey, err.Message)
	case "invalid_request_error":
		return fmt.Errorf("%w: %s", ErrInvalidRequest, err.Message)
	case "provider_error", "provider_unavailable":
		return fmt.Errorf("%w: %s", ErrProviderUnavailable, err.Message)
	default:
		return fmt.Errorf("%w: %s", ErrProviderError, err.Error())
	}
}

// GetModel returns the current default model
func (p *OpenRouterProvider) GetModel() OpenRouterModel {
	return p.model
}

// SetModel sets the default model
func (p *OpenRouterProvider) SetModel(model OpenRouterModel) {
	p.model = model
}

// SetProviderPreferences sets the default provider preferences
func (p *OpenRouterProvider) SetProviderPreferences(prefs *ProviderPreferences) {
	p.config.DefaultProviderPreferences = prefs
}

// AddCustomHeader adds a custom header to be included in requests
func (p *OpenRouterProvider) AddCustomHeader(key, value string) {
	if p.config.CustomHeaders == nil {
		p.config.CustomHeaders = make(map[string]string)
	}
	p.config.CustomHeaders[key] = value
}

// RemoveCustomHeader removes a custom header
func (p *OpenRouterProvider) RemoveCustomHeader(key string) {
	delete(p.config.CustomHeaders, key)
}

// GetCustomHeaders returns the current custom headers
func (p *OpenRouterProvider) GetCustomHeaders() map[string]string {
	// Return a copy to prevent external modification
	headers := make(map[string]string, len(p.config.CustomHeaders))
	for k, v := range p.config.CustomHeaders {
		headers[k] = v
	}
	return headers
}

// CreateFallbackChain creates a fallback chain for model routing
func (p *OpenRouterProvider) CreateFallbackChain(primaryModel string, fallbacks ...string) []string {
	chain := []string{primaryModel}
	chain = append(chain, fallbacks...)
	return chain
}

// IsProviderModelFormat checks if a model ID follows the provider/model format
func IsProviderModelFormat(modelID string) bool {
	parts := strings.Split(modelID, "/")
	return len(parts) == 2 && parts[0] != "" && parts[1] != ""
}

// ExtractProviderFromModel extracts the provider from a model ID
func ExtractProviderFromModel(modelID string) (string, error) {
	parts := strings.Split(modelID, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid model ID format: %s", modelID)
	}
	return parts[0], nil
}

// ExtractModelName extracts the model name from a model ID
func ExtractModelName(modelID string) (string, error) {
	parts := strings.Split(modelID, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid model ID format: %s", modelID)
	}
	return parts[1], nil
}