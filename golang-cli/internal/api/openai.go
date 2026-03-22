package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAIModel represents supported OpenAI models
type OpenAIModel string

const (
	// GPT-4 models
	GPT4o         OpenAIModel = "gpt-4o"
	GPT4Turbo     OpenAIModel = "gpt-4-turbo"
	GPT4          OpenAIModel = "gpt-4"

	// GPT-3.5 models
	GPT35Turbo    OpenAIModel = "gpt-3.5-turbo"
	GPT35Turbo16K OpenAIModel = "gpt-3.5-turbo-16k"
)

// Default settings
const (
	DefaultOpenAITimeout = 120 * time.Second
	DefaultTemperature   = 0.7
	DefaultMaxTokens     = 4096
	OpenAIAPIBaseURL     = "https://api.openai.com/v1"
)

// openAIModels is the list of supported OpenAI models
var openAIModels = []string{
	string(GPT4o),
	string(GPT4Turbo),
	string(GPT4),
	string(GPT35Turbo),
	string(GPT35Turbo16K),
}

// openAIInternalRequest represents the request body for OpenAI chat completions
type openAIInternalRequest struct {
	Model       string                  `json:"model"`
	Messages    []openAIInternalMessage `json:"messages"`
	Temperature float64                 `json:"temperature,omitempty"`
	MaxTokens   int                     `json:"max_tokens,omitempty"`
	TopP        float64                 `json:"top_p,omitempty"`
	Stream      bool                    `json:"stream"`
}

// openAIInternalMessage represents a message in OpenAI format
type openAIInternalMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAIInternalResponse represents a non-streaming response from OpenAI
type openAIInternalResponse struct {
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
	Error *openAIInternalError `json:"error,omitempty"`
}

// openAIInternalStreamResponse represents a streaming chunk from OpenAI
type openAIInternalStreamResponse struct {
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

// openAIInternalError represents an error response from OpenAI
type openAIInternalError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    string `json:"code"`
}

// Error implements the error interface
func (e *openAIInternalError) Error() string {
	return fmt.Sprintf("openai error: %s (type: %s, code: %s)", e.Message, e.Type, e.Code)
}

// OpenAIConfig contains configuration for the OpenAI provider
type OpenAIConfig struct {
	// APIKey is the OpenAI API key (required)
	APIKey string

	// BaseURL is the base URL for the API (defaults to OpenAIAPIBaseURL)
	// Can be used for Azure OpenAI by setting the Azure endpoint
	BaseURL string

	// Organization is the optional OpenAI organization ID
	Organization string

	// HTTPClient is the HTTP client to use (defaults to http.DefaultClient with timeout)
	HTTPClient *http.Client
}

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
	config     OpenAIConfig
	httpClient *http.Client
	baseURL    string
}

// OpenAICompletionRequest represents a request for OpenAI chat completion
type OpenAICompletionRequest struct {
	Model       string
	Messages    []OpenAIMessage
	Temperature float64
	MaxTokens   int
	TopP        float64
	Stream      bool
}

// OpenAIMessage represents a chat message for OpenAI
type OpenAIMessage struct {
	Role    string
	Content string
}

// OpenAICompletionResponse represents a response from OpenAI
type OpenAICompletionResponse struct {
	ID      string
	Model   string
	Content string
	Usage   OpenAIUsage
}

// OpenAIUsage represents token usage information for OpenAI
type OpenAIUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// OpenAIStreamChunk represents a chunk from a streaming response
type OpenAIStreamChunk struct {
	Delta        string
	Content      string
	FinishReason string
	Usage        *OpenAIUsage
}

// Common errors
var (
	ErrOpenAIInvalidAPIKey     = fmt.Errorf("invalid API key")
	ErrOpenAIModelNotFound     = fmt.Errorf("model not found")
	ErrOpenAIInvalidRequest    = fmt.Errorf("invalid request")
	ErrOpenAIRateLimitExceeded = fmt.Errorf("rate limit exceeded")
	ErrOpenAIProviderError     = fmt.Errorf("provider error")
	ErrOpenAIContextCanceled   = fmt.Errorf("context canceled")
	ErrOpenAIInvalidResponse   = fmt.Errorf("invalid response from provider")
)

// NewOpenAIProvider creates a new OpenAI provider with the given configuration
func NewOpenAIProvider(config OpenAIConfig) (*OpenAIProvider, error) {
	if config.APIKey == "" {
		return nil, ErrOpenAIInvalidAPIKey
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: DefaultOpenAITimeout,
		}
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = OpenAIAPIBaseURL
	}

	return &OpenAIProvider{
		config:     config,
		httpClient: httpClient,
		baseURL:    strings.TrimSuffix(baseURL, "/"),
	}, nil
}

// Complete sends a chat completion request and returns the full response
func (p *OpenAIProvider) Complete(ctx context.Context, req OpenAICompletionRequest) (*OpenAICompletionResponse, error) {
	if err := p.ValidateModel(req.Model); err != nil {
		return nil, err
	}

	openAIReq := p.toOpenAIInternalRequest(req, false)

	body, err := json.Marshal(openAIReq)
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

	var openAIResp openAIInternalResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if openAIResp.Error != nil {
		return nil, p.convertOpenAIError(openAIResp.Error)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, ErrOpenAIInvalidResponse
	}

	return &OpenAICompletionResponse{
		ID:      openAIResp.ID,
		Model:   openAIResp.Model,
		Content: openAIResp.Choices[0].Message.Content,
		Usage: OpenAIUsage{
			PromptTokens:     openAIResp.Usage.PromptTokens,
			CompletionTokens: openAIResp.Usage.CompletionTokens,
			TotalTokens:      openAIResp.Usage.TotalTokens,
		},
	}, nil
}

// CompleteStream sends a streaming chat completion request and returns a channel of chunks
func (p *OpenAIProvider) CompleteStream(ctx context.Context, req OpenAICompletionRequest) (<-chan OpenAIStreamChunk, <-chan error, error) {
	if err := p.ValidateModel(req.Model); err != nil {
		return nil, nil, err
	}

	openAIReq := p.toOpenAIInternalRequest(req, true)

	body, err := json.Marshal(openAIReq)
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

	chunkChan := make(chan OpenAIStreamChunk, 10)
	errChan := make(chan error, 1)

	go p.processStream(ctx, req, resp.Body, chunkChan, errChan)

	return chunkChan, errChan, nil
}

// processStream processes the SSE stream from OpenAI
func (p *OpenAIProvider) processStream(ctx context.Context, req OpenAICompletionRequest, body io.ReadCloser, chunkChan chan<- OpenAIStreamChunk, errChan chan<- error) {
	defer close(chunkChan)
	defer close(errChan)
	defer body.Close()

	scanner := bufio.NewScanner(body)

	// Track accumulated usage for the final chunk
	var accumulatedUsage OpenAIUsage
	contentBuffer := ""

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			errChan <- ErrOpenAIContextCanceled
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
				chunkChan <- OpenAIStreamChunk{
					Content:      contentBuffer,
					FinishReason: "stop",
					Usage:        &accumulatedUsage,
				}
			}
			return
		}

		var streamResp openAIInternalStreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			errChan <- fmt.Errorf("failed to decode stream chunk: %w", err)
			return
		}

		if len(streamResp.Choices) == 0 {
			continue
		}

		choice := streamResp.Choices[0]

		// Accumulate usage if provided (OpenAI may send this in the final chunk)
		if streamResp.Usage != nil {
			accumulatedUsage = OpenAIUsage{
				PromptTokens:     streamResp.Usage.PromptTokens,
				CompletionTokens: streamResp.Usage.CompletionTokens,
				TotalTokens:      streamResp.Usage.TotalTokens,
			}
		}

		// Accumulate content for usage calculation if not provided
		contentBuffer += choice.Delta.Content

		chunk := OpenAIStreamChunk{
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
			errChan <- ErrOpenAIContextCanceled
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
func (p *OpenAIProvider) estimateUsage(messages []OpenAIMessage, completion string) OpenAIUsage {
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

	return OpenAIUsage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
	}
}

// GetSupportedModels returns a list of supported model IDs
func (p *OpenAIProvider) GetSupportedModels() []string {
	models := make([]string, len(openAIModels))
	copy(models, openAIModels)
	return models
}

// ValidateModel checks if a model is supported
func (p *OpenAIProvider) ValidateModel(model string) error {
	for _, m := range openAIModels {
		if m == model {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrOpenAIModelNotFound, model)
}

// toOpenAIInternalRequest converts a OpenAICompletionRequest to an OpenAI-specific request
func (p *OpenAIProvider) toOpenAIInternalRequest(req OpenAICompletionRequest, stream bool) openAIInternalRequest {
	messages := make([]openAIInternalMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openAIInternalMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = DefaultTemperature
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = DefaultMaxTokens
	}

	topP := req.TopP
	if topP == 0 {
		topP = 1.0
	}

	return openAIInternalRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		TopP:        topP,
		Stream:      stream,
	}
}

// setHeaders sets the required HTTP headers for OpenAI requests
func (p *OpenAIProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))

	if p.config.Organization != "" {
		req.Header.Set("OpenAI-Organization", p.config.Organization)
	}

	// Azure OpenAI doesn't use Authorization header with Bearer
	// It uses api-key header instead
	if p.isAzureEndpoint() {
		req.Header.Del("Authorization")
		req.Header.Set("api-key", p.config.APIKey)
	}
}

// isAzureEndpoint checks if the configured endpoint is Azure OpenAI
func (p *OpenAIProvider) isAzureEndpoint() bool {
	return strings.Contains(p.baseURL, "azure.com") || strings.Contains(p.baseURL, "microsoft.com")
}

// handleErrorResponse handles HTTP error responses
func (p *OpenAIProvider) handleErrorResponse(resp *http.Response) error {
	if resp.StatusCode < 400 {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d: failed to read error body: %w", resp.StatusCode, err)
	}

	// Try to parse as OpenAI error
	var errResp struct {
		Error *openAIInternalError `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != nil {
		return p.convertOpenAIError(errResp.Error)
	}

	// Fallback to generic error
	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		return ErrOpenAIRateLimitExceeded
	case http.StatusUnauthorized:
		return ErrOpenAIInvalidAPIKey
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", ErrOpenAIInvalidRequest, string(body))
	default:
		return fmt.Errorf("%w: HTTP %d - %s", ErrOpenAIProviderError, resp.StatusCode, string(body))
	}
}

// convertOpenAIError converts an OpenAI error to the appropriate error type
func (p *OpenAIProvider) convertOpenAIError(err *openAIInternalError) error {
	switch err.Code {
	case "rate_limit_exceeded", "insufficient_quota":
		return fmt.Errorf("%w: %s", ErrOpenAIRateLimitExceeded, err.Message)
	case "invalid_api_key":
		return fmt.Errorf("%w: %s", ErrOpenAIInvalidAPIKey, err.Message)
	case "invalid_request_error":
		return fmt.Errorf("%w: %s", ErrOpenAIInvalidRequest, err.Message)
	default:
		return fmt.Errorf("%w: %s", ErrOpenAIProviderError, err.Error())
	}
}