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

// LMStudio errors
var (
	ErrLMStudioInvalidURL        = errors.New("invalid LM Studio URL")
	ErrLMStudioModelNotFound     = errors.New("LM Studio model not found")
	ErrLMStudioModelNotLoaded    = errors.New("LM Studio model not loaded")
	ErrLMStudioInvalidRequest    = errors.New("invalid LM Studio request")
	ErrLMStudioInvalidResponse   = errors.New("invalid response from LM Studio API")
	ErrLMStudioProviderError     = errors.New("LM Studio provider error")
	ErrLMStudioContextCanceled   = errors.New("LM Studio request canceled")
	ErrLMStudioConnectionFailed  = errors.New("failed to connect to LM Studio")
)

// Default settings for LM Studio
const (
	DefaultLMStudioBaseURL = "http://localhost:1234"
	DefaultLMStudioTimeout = 120 * time.Second
)

// LMStudioConfig contains configuration for the LM Studio provider
type LMStudioConfig struct {
	// BaseURL is the base URL for the LM Studio API (defaults to DefaultLMStudioBaseURL)
	BaseURL string

	// APIKey is an optional API key for authentication (LM Studio supports API key auth)
	APIKey string

	// HTTPClient is the HTTP client to use (defaults to http.DefaultClient with timeout)
	HTTPClient HTTPClient
}

// LMStudioProvider implements the Provider interface for LM Studio
type LMStudioProvider struct {
	config     LMStudioConfig
	httpClient HTTPClient
	baseURL    string
}

// LMStudioMessage represents a message in the conversation for LM Studio
type LMStudioMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LMStudioUsage represents token usage for an LM Studio request
type LMStudioUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LMStudioCompletionRequest represents a request for LM Studio chat completion
type LMStudioCompletionRequest struct {
	Model       string
	Messages    []LMStudioMessage
	Temperature float64
	MaxTokens   int
	TopP        float64
	TopK        int
	Stream      bool
}

// LMStudioCompletionResponse represents a response from LM Studio
type LMStudioCompletionResponse struct {
	ID      string
	Model   string
	Content string
	Usage   LMStudioUsage
}

// LMStudioStreamChunk represents a chunk from a streaming response
type LMStudioStreamChunk struct {
	Delta        string
	Content      string
	FinishReason string
	Usage        *LMStudioUsage
}

// lmStudioChatRequest represents the request body for LM Studio chat API (OpenAI-compatible)
type lmStudioChatRequest struct {
	Model       string              `json:"model"`
	Messages    []lmStudioMessage   `json:"messages"`
	Temperature float64             `json:"temperature,omitempty"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	TopP        float64             `json:"top_p,omitempty"`
	Stream      bool                `json:"stream"`
}

// lmStudioMessage represents a message in LM Studio format
type lmStudioMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// lmStudioChatResponse represents a non-streaming response from LM Studio (OpenAI-compatible)
type lmStudioChatResponse struct {
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
}

// lmStudioStreamResponse represents a streaming chunk from LM Studio
type lmStudioStreamResponse struct {
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

// lmStudioModel represents a loaded model in LM Studio
type lmStudioModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
}

// lmStudioModelsResponse represents the response from listing models
type lmStudioModelsResponse struct {
	Object string          `json:"object"`
	Data   []lmStudioModel `json:"data"`
}

// lmStudioModelInfo represents detailed model information
type lmStudioModelInfo struct {
	ID       string                 `json:"id"`
	Object   string                 `json:"object"`
	Created  int64                  `json:"created"`
	OwnedBy  string                 `json:"owned_by"`
	Meta     map[string]interface{} `json:"meta"`
}

// lmStudioError represents an error response from LM Studio
type lmStudioError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// NewLMStudioProvider creates a new LM Studio provider with the given configuration
func NewLMStudioProvider(config LMStudioConfig) (*LMStudioProvider, error) {
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: DefaultLMStudioTimeout,
		}
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = DefaultLMStudioBaseURL
	}

	// Validate URL
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		return nil, ErrLMStudioInvalidURL
	}

	return &LMStudioProvider{
		config:     config,
		httpClient: httpClient,
		baseURL:    strings.TrimSuffix(baseURL, "/"),
	}, nil
}

// Complete sends a chat completion request and returns the full response
func (p *LMStudioProvider) Complete(ctx context.Context, req LMStudioCompletionRequest) (*LMStudioCompletionResponse, error) {
	lmReq := p.toLMStudioChatRequest(req, false)

	body, err := json.Marshal(lmReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/chat/completions", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrLMStudioConnectionFailed, err.Error())
	}
	defer resp.Body.Close()

	if err := p.handleErrorResponse(resp); err != nil {
		return nil, err
	}

	var lmResp lmStudioChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&lmResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(lmResp.Choices) == 0 {
		return nil, ErrLMStudioInvalidResponse
	}

	return &LMStudioCompletionResponse{
		ID:      lmResp.ID,
		Model:   lmResp.Model,
		Content: lmResp.Choices[0].Message.Content,
		Usage: LMStudioUsage{
			PromptTokens:     lmResp.Usage.PromptTokens,
			CompletionTokens: lmResp.Usage.CompletionTokens,
			TotalTokens:      lmResp.Usage.TotalTokens,
		},
	}, nil
}

// CompleteStream sends a streaming chat completion request and returns a channel of chunks
func (p *LMStudioProvider) CompleteStream(ctx context.Context, req LMStudioCompletionRequest) (<-chan LMStudioStreamChunk, <-chan error, error) {
	lmReq := p.toLMStudioChatRequest(req, true)

	body, err := json.Marshal(lmReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/chat/completions", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %s", ErrLMStudioConnectionFailed, err.Error())
	}

	if err := p.handleErrorResponse(resp); err != nil {
		resp.Body.Close()
		return nil, nil, err
	}

	chunkChan := make(chan LMStudioStreamChunk, 10)
	errChan := make(chan error, 1)

	go p.processStream(ctx, req, resp.Body, chunkChan, errChan)

	return chunkChan, errChan, nil
}

// processStream processes the SSE stream from LM Studio
func (p *LMStudioProvider) processStream(ctx context.Context, req LMStudioCompletionRequest, body io.ReadCloser, chunkChan chan<- LMStudioStreamChunk, errChan chan<- error) {
	defer close(chunkChan)
	defer close(errChan)
	defer body.Close()

	scanner := bufio.NewScanner(body)

	// Track accumulated usage for the final chunk
	var accumulatedUsage LMStudioUsage
	contentBuffer := ""

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			errChan <- ErrLMStudioContextCanceled
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
				chunkChan <- LMStudioStreamChunk{
					Content:      contentBuffer,
					FinishReason: "stop",
					Usage:        &accumulatedUsage,
				}
			}
			return
		}

		var streamResp lmStudioStreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			errChan <- fmt.Errorf("failed to decode stream chunk: %w", err)
			return
		}

		if len(streamResp.Choices) == 0 {
			continue
		}

		choice := streamResp.Choices[0]

		// Accumulate usage if provided (LM Studio may send this in the final chunk)
		if streamResp.Usage != nil {
			accumulatedUsage = LMStudioUsage{
				PromptTokens:     streamResp.Usage.PromptTokens,
				CompletionTokens: streamResp.Usage.CompletionTokens,
				TotalTokens:      streamResp.Usage.TotalTokens,
			}
		}

		// Accumulate content for usage calculation if not provided
		contentBuffer += choice.Delta.Content

		chunk := LMStudioStreamChunk{
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
			errChan <- ErrLMStudioContextCanceled
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

// ListModels returns a list of available models from LM Studio
func (p *LMStudioProvider) ListModels(ctx context.Context) ([]lmStudioModel, error) {
	url := fmt.Sprintf("%s/v1/models", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrLMStudioConnectionFailed, err.Error())
	}
	defer resp.Body.Close()

	if err := p.handleErrorResponse(resp); err != nil {
		return nil, err
	}

	var modelsResp lmStudioModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return modelsResp.Data, nil
}

// GetModelInfo returns detailed information about a specific model
func (p *LMStudioProvider) GetModelInfo(ctx context.Context, modelID string) (*lmStudioModelInfo, error) {
	url := fmt.Sprintf("%s/v1/models/%s", p.baseURL, modelID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrLMStudioConnectionFailed, err.Error())
	}
	defer resp.Body.Close()

	if err := p.handleErrorResponse(resp); err != nil {
		return nil, err
	}

	var modelInfo lmStudioModelInfo
	if err := json.NewDecoder(resp.Body).Decode(&modelInfo); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &modelInfo, nil
}

// LoadModel loads a model in LM Studio (if supported by the API)
func (p *LMStudioProvider) LoadModel(ctx context.Context, modelPath string) error {
	type loadRequest struct {
		ModelPath string `json:"model_path"`
	}

	req := loadRequest{
		ModelPath: modelPath,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/models/load", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrLMStudioConnectionFailed, err.Error())
	}
	defer resp.Body.Close()

	if err := p.handleErrorResponse(resp); err != nil {
		return err
	}

	return nil
}

// UnloadModel unloads the currently loaded model
func (p *LMStudioProvider) UnloadModel(ctx context.Context) error {
	url := fmt.Sprintf("%s/v1/models/unload", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrLMStudioConnectionFailed, err.Error())
	}
	defer resp.Body.Close()

	if err := p.handleErrorResponse(resp); err != nil {
		return err
	}

	return nil
}

// toLMStudioChatRequest converts an LMStudioCompletionRequest to an LM Studio-specific request
func (p *LMStudioProvider) toLMStudioChatRequest(req LMStudioCompletionRequest, stream bool) lmStudioChatRequest {
	messages := make([]lmStudioMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = lmStudioMessage{
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

	return lmStudioChatRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		TopP:        topP,
		Stream:      stream,
	}
}

// estimateUsage estimates token usage when the API doesn't provide it
func (p *LMStudioProvider) estimateUsage(messages []LMStudioMessage, completion string) LMStudioUsage {
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

	return LMStudioUsage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
	}
}

// setHeaders sets the required HTTP headers for LM Studio requests
func (p *LMStudioProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")

	// Add API key if provided
	if p.config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))
	}
}

// handleErrorResponse handles HTTP error responses
func (p *LMStudioProvider) handleErrorResponse(resp *http.Response) error {
	if resp.StatusCode < 400 {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d: failed to read error body: %w", resp.StatusCode, err)
	}

	// Try to parse as LM Studio error
	var errResp lmStudioError
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		return p.convertLMStudioError(&errResp)
	}

	// Fallback to generic error
	switch resp.StatusCode {
	case http.StatusNotFound:
		return ErrLMStudioModelNotFound
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", ErrLMStudioInvalidRequest, string(body))
	case http.StatusServiceUnavailable:
		return ErrLMStudioModelNotLoaded
	default:
		return fmt.Errorf("%w: HTTP %d - %s", ErrLMStudioProviderError, resp.StatusCode, string(body))
	}
}

// convertLMStudioError converts an LM Studio error to the appropriate error type
func (p *LMStudioProvider) convertLMStudioError(err *lmStudioError) error {
	if err.Error.Message == "" {
		return ErrLMStudioProviderError
	}

	lowerMsg := strings.ToLower(err.Error.Message)

	if strings.Contains(lowerMsg, "model") && strings.Contains(lowerMsg, "not found") {
		return fmt.Errorf("%w: %s", ErrLMStudioModelNotFound, err.Error.Message)
	}
	if strings.Contains(lowerMsg, "model") && (strings.Contains(lowerMsg, "not loaded") || strings.Contains(lowerMsg, "no model")) {
		return fmt.Errorf("%w: %s", ErrLMStudioModelNotLoaded, err.Error.Message)
	}
	if strings.Contains(lowerMsg, "connection refused") || strings.Contains(lowerMsg, "no such host") {
		return fmt.Errorf("%w: %s", ErrLMStudioConnectionFailed, err.Error.Message)
	}

	return fmt.Errorf("%w: %s", ErrLMStudioProviderError, err.Error.Message)
}

// GetSupportedModels returns a list of supported model IDs
func (p *LMStudioProvider) GetSupportedModels() []string {
	// LM Studio supports any OpenAI-compatible model
	// In practice, users would call ListModels() to get loaded models
	return []string{}
}

// ValidateModel checks if a model is supported
func (p *LMStudioProvider) ValidateModel(model string) error {
	// LM Studio supports any model name
	if model == "" {
		return fmt.Errorf("%w: model name cannot be empty", ErrLMStudioModelNotFound)
	}
	return nil
}

// GetModel returns the current default model
func (p *LMStudioProvider) GetModel() string {
	return ""
}

// SetModel sets the default model
func (p *LMStudioProvider) SetModel(model string) {
	// No-op for LM Studio since models are specified per-request
}

// GetBaseURL returns the base URL for the provider
func (p *LMStudioProvider) GetBaseURL() string {
	return p.baseURL
}

// IsRunning checks if the LM Studio server is running
func (p *LMStudioProvider) IsRunning(ctx context.Context) bool {
	url := fmt.Sprintf("%s/v1/models", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// IsLocalModel returns true since LM Studio is always local
func (p *LMStudioProvider) IsLocalModel() bool {
	return true
}

// GetLoadedModel returns the currently loaded model, if any
func (p *LMStudioProvider) GetLoadedModel(ctx context.Context) (string, error) {
	models, err := p.ListModels(ctx)
	if err != nil {
		return "", err
	}

	if len(models) == 0 {
		return "", ErrLMStudioModelNotLoaded
	}

	// Return the first loaded model
	return models[0].ID, nil
}
