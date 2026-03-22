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

// Ollama errors
var (
	ErrOllamaInvalidURL         = errors.New("invalid Ollama URL")
	ErrOllamaModelNotFound      = errors.New("Ollama model not found")
	ErrOllamaModelNotRunning    = errors.New("Ollama model not running")
	ErrOllamaInvalidRequest     = errors.New("invalid Ollama request")
	ErrOllamaInvalidResponse    = errors.New("invalid response from Ollama API")
	ErrOllamaProviderError      = errors.New("Ollama provider error")
	ErrOllamaContextCanceled    = errors.New("Ollama request canceled")
	ErrOllamaConnectionFailed   = errors.New("failed to connect to Ollama")
)

// Default settings for Ollama
const (
	DefaultOllamaBaseURL = "http://localhost:11434"
	DefaultOllamaTimeout = 120 * time.Second
)

// OllamaModel represents Ollama model information
type OllamaModel struct {
	Name       string `json:"name"`
	Model      string `json:"model"`
	ModifiedAt string `json:"modified_at"`
	Size       int64  `json:"size"`
	Digest     string `json:"digest"`
	Details    struct {
		Format            string   `json:"format"`
		Family            string   `json:"family"`
		Families          []string `json:"families"`
		ParameterSize     string   `json:"parameter_size"`
		QuantizationLevel string   `json:"quantization_level"`
	} `json:"details"`
}

// OllamaConfig contains configuration for the Ollama provider
type OllamaConfig struct {
	// BaseURL is the base URL for the Ollama API (defaults to DefaultOllamaBaseURL)
	BaseURL string

	// HTTPClient is the HTTP client to use (defaults to http.DefaultClient with timeout)
	HTTPClient HTTPClient
}

// OllamaProvider implements the Provider interface for Ollama
type OllamaProvider struct {
	config     OllamaConfig
	httpClient HTTPClient
	baseURL    string
}

// OllamaMessage represents a message in the conversation for Ollama
type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OllamaUsage represents token usage for an Ollama request
type OllamaUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OllamaCompletionRequest represents a request for Ollama chat completion
type OllamaCompletionRequest struct {
	Model       string
	Messages    []OllamaMessage
	Temperature float64
	MaxTokens   int
	TopP        float64
	TopK        int
	Stream      bool
}

// OllamaCompletionResponse represents a response from Ollama
type OllamaCompletionResponse struct {
	ID      string
	Model   string
	Content string
	Usage   OllamaUsage
}

// OllamaStreamChunk represents a chunk from a streaming response
type OllamaStreamChunk struct {
	Delta        string
	Content      string
	FinishReason string
	Usage        *OllamaUsage
}

// ollamaChatRequest represents the request body for Ollama chat API
type ollamaChatRequest struct {
	Model    string            `json:"model"`
	Messages []ollamaMessage   `json:"messages"`
	Stream   bool              `json:"stream"`
	Options  *ollamaOptions    `json:"options,omitempty"`
}

// ollamaMessage represents a message in Ollama format
type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ollamaOptions represents generation options
type ollamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"num_predict,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	TopK        int     `json:"top_k,omitempty"`
}

// ollamaChatResponse represents a non-streaming response from Ollama chat API
type ollamaChatResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Message   struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done               bool `json:"done"`
	TotalDuration    int64 `json:"total_duration"`
	LoadDuration     int64 `json:"load_duration"`
	PromptEvalCount  int   `json:"prompt_eval_count"`
	PromptEvalDuration int64 `json:"prompt_eval_duration"`
	EvalCount        int   `json:"eval_count"`
	EvalDuration     int64 `json:"eval_duration"`
}

// ollamaStreamResponse represents a streaming chunk from Ollama
type ollamaStreamResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Message   struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done               bool  `json:"done"`
	TotalDuration      int64 `json:"total_duration,omitempty"`
	LoadDuration       int64 `json:"load_duration,omitempty"`
	PromptEvalCount    int   `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64 `json:"prompt_eval_duration,omitempty"`
	EvalCount          int   `json:"eval_count,omitempty"`
	EvalDuration       int64 `json:"eval_duration,omitempty"`
}

// ollamaListResponse represents the response from listing models
type ollamaListResponse struct {
	Models []OllamaModel `json:"models"`
}

// ollamaGenerateRequest represents a request to the generate endpoint
type ollamaGenerateRequest struct {
	Model  string         `json:"model"`
	Prompt string         `json:"prompt"`
	Stream bool           `json:"stream"`
	Options *ollamaOptions `json:"options,omitempty"`
}

// ollamaGenerateResponse represents a response from the generate endpoint
type ollamaGenerateResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	Context   []int  `json:"context"`
}

// ollamaError represents an error response from Ollama
type ollamaError struct {
	Error string `json:"error"`
}

// NewOllamaProvider creates a new Ollama provider with the given configuration
func NewOllamaProvider(config OllamaConfig) (*OllamaProvider, error) {
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: DefaultOllamaTimeout,
		}
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = DefaultOllamaBaseURL
	}

	// Validate URL
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		return nil, ErrOllamaInvalidURL
	}

	return &OllamaProvider{
		config:     config,
		httpClient: httpClient,
		baseURL:    strings.TrimSuffix(baseURL, "/"),
	}, nil
}

// Complete sends a chat completion request and returns the full response
func (p *OllamaProvider) Complete(ctx context.Context, req OllamaCompletionRequest) (*OllamaCompletionResponse, error) {
	ollamaReq := p.toOllamaChatRequest(req, false)

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/chat", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrOllamaConnectionFailed, err.Error())
	}
	defer resp.Body.Close()

	if err := p.handleErrorResponse(resp); err != nil {
		return nil, err
	}

	var ollamaResp ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Calculate usage from response metadata
	usage := OllamaUsage{
		PromptTokens:     ollamaResp.PromptEvalCount,
		CompletionTokens: ollamaResp.EvalCount,
		TotalTokens:      ollamaResp.PromptEvalCount + ollamaResp.EvalCount,
	}

	return &OllamaCompletionResponse{
		ID:      fmt.Sprintf("ollama-%d", time.Now().Unix()),
		Model:   ollamaResp.Model,
		Content: ollamaResp.Message.Content,
		Usage:   usage,
	}, nil
}

// CompleteStream sends a streaming chat completion request and returns a channel of chunks
func (p *OllamaProvider) CompleteStream(ctx context.Context, req OllamaCompletionRequest) (<-chan OllamaStreamChunk, <-chan error, error) {
	ollamaReq := p.toOllamaChatRequest(req, true)

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/chat", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %s", ErrOllamaConnectionFailed, err.Error())
	}

	if err := p.handleErrorResponse(resp); err != nil {
		resp.Body.Close()
		return nil, nil, err
	}

	chunkChan := make(chan OllamaStreamChunk, 10)
	errChan := make(chan error, 1)

	go p.processStream(ctx, resp.Body, chunkChan, errChan)

	return chunkChan, errChan, nil
}

// processStream processes the NDJSON stream from Ollama
func (p *OllamaProvider) processStream(ctx context.Context, body io.ReadCloser, chunkChan chan<- OllamaStreamChunk, errChan chan<- error) {
	defer close(chunkChan)
	defer close(errChan)
	defer body.Close()

	scanner := bufio.NewScanner(body)

	// Track accumulated usage for the final chunk
	var promptEvalCount, evalCount int
	contentBuffer := ""

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			errChan <- ErrOllamaContextCanceled
			return
		default:
		}

		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		var streamResp ollamaStreamResponse
		if err := json.Unmarshal([]byte(line), &streamResp); err != nil {
			errChan <- fmt.Errorf("failed to decode stream chunk: %w", err)
			return
		}

		// Accumulate usage
		if streamResp.PromptEvalCount > 0 {
			promptEvalCount = streamResp.PromptEvalCount
		}
		if streamResp.EvalCount > 0 {
			evalCount = streamResp.EvalCount
		}

		// Extract content
		delta := streamResp.Message.Content
		contentBuffer += delta

		chunk := OllamaStreamChunk{
			Delta:   delta,
			Content: contentBuffer,
		}

		if streamResp.Done {
			chunk.FinishReason = "stop"
			chunk.Usage = &OllamaUsage{
				PromptTokens:     promptEvalCount,
				CompletionTokens: evalCount,
				TotalTokens:      promptEvalCount + evalCount,
			}
		}

		select {
		case chunkChan <- chunk:
		case <-ctx.Done():
			errChan <- ErrOllamaContextCanceled
			return
		}

		if streamResp.Done {
			return
		}
	}

	if err := scanner.Err(); err != nil {
		errChan <- fmt.Errorf("stream read error: %w", err)
	}
}

// Generate generates a completion using the Ollama generate endpoint (legacy)
func (p *OllamaProvider) Generate(ctx context.Context, model, prompt string, stream bool) (*ollamaGenerateResponse, error) {
	req := ollamaGenerateRequest{
		Model:  model,
		Prompt: prompt,
		Stream: stream,
		Options: &ollamaOptions{
			Temperature: 0.7,
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/generate", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrOllamaConnectionFailed, err.Error())
	}
	defer resp.Body.Close()

	if err := p.handleErrorResponse(resp); err != nil {
		return nil, err
	}

	var ollamaResp ollamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &ollamaResp, nil
}

// ListModels returns a list of available models from Ollama
func (p *OllamaProvider) ListModels(ctx context.Context) ([]OllamaModel, error) {
	url := fmt.Sprintf("%s/api/tags", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrOllamaConnectionFailed, err.Error())
	}
	defer resp.Body.Close()

	if err := p.handleErrorResponse(resp); err != nil {
		return nil, err
	}

	var listResp ollamaListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return listResp.Models, nil
}

// PullModel pulls a model from the Ollama registry
func (p *OllamaProvider) PullModel(ctx context.Context, model string, stream bool) (<-chan string, <-chan error, error) {
	type pullRequest struct {
		Name   string `json:"name"`
		Stream bool   `json:"stream"`
	}

	req := pullRequest{
		Name:   model,
		Stream: stream,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/pull", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %s", ErrOllamaConnectionFailed, err.Error())
	}

	if err := p.handleErrorResponse(resp); err != nil {
		resp.Body.Close()
		return nil, nil, err
	}

	statusChan := make(chan string, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(statusChan)
		defer close(errChan)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				select {
				case statusChan <- line:
				case <-ctx.Done():
					errChan <- ErrOllamaContextCanceled
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			errChan <- fmt.Errorf("stream read error: %w", err)
		}
	}()

	return statusChan, errChan, nil
}

// DeleteModel deletes a model from Ollama
func (p *OllamaProvider) DeleteModel(ctx context.Context, model string) error {
	type deleteRequest struct {
		Name string `json:"name"`
	}

	req := deleteRequest{
		Name: model,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/delete", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrOllamaConnectionFailed, err.Error())
	}
	defer resp.Body.Close()

	if err := p.handleErrorResponse(resp); err != nil {
		return err
	}

	return nil
}

// ShowModel shows information about a model
func (p *OllamaProvider) ShowModel(ctx context.Context, model string) (map[string]interface{}, error) {
	type showRequest struct {
		Name string `json:"name"`
	}

	req := showRequest{
		Name: model,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/show", p.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrOllamaConnectionFailed, err.Error())
	}
	defer resp.Body.Close()

	if err := p.handleErrorResponse(resp); err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// toOllamaChatRequest converts an OllamaCompletionRequest to an Ollama-specific request
func (p *OllamaProvider) toOllamaChatRequest(req OllamaCompletionRequest, stream bool) ollamaChatRequest {
	messages := make([]ollamaMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = ollamaMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	options := &ollamaOptions{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		TopP:        req.TopP,
		TopK:        req.TopK,
	}

	// Set defaults if not provided
	if options.Temperature == 0 {
		options.Temperature = 0.7
	}
	if options.MaxTokens == 0 {
		options.MaxTokens = 4096
	}

	return ollamaChatRequest{
		Model:    req.Model,
		Messages: messages,
		Stream:   stream,
		Options:  options,
	}
}

// setHeaders sets the required HTTP headers for Ollama requests
func (p *OllamaProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
}

// handleErrorResponse handles HTTP error responses
func (p *OllamaProvider) handleErrorResponse(resp *http.Response) error {
	if resp.StatusCode < 400 {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d: failed to read error body: %w", resp.StatusCode, err)
	}

	// Try to parse as Ollama error
	var errResp ollamaError
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
		return p.convertOllamaError(errResp.Error, resp.StatusCode)
	}

	// Fallback to generic error
	switch resp.StatusCode {
	case http.StatusNotFound:
		return ErrOllamaModelNotFound
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", ErrOllamaInvalidRequest, string(body))
	default:
		return fmt.Errorf("%w: HTTP %d - %s", ErrOllamaProviderError, resp.StatusCode, string(body))
	}
}

// convertOllamaError converts an Ollama error string to the appropriate error type
func (p *OllamaProvider) convertOllamaError(errMsg string, statusCode int) error {
	lowerErr := strings.ToLower(errMsg)

	if strings.Contains(lowerErr, "model") && strings.Contains(lowerErr, "not found") {
		return fmt.Errorf("%w: %s", ErrOllamaModelNotFound, errMsg)
	}
	if strings.Contains(lowerErr, "connection refused") || strings.Contains(lowerErr, "no such host") {
		return fmt.Errorf("%w: %s", ErrOllamaConnectionFailed, errMsg)
	}

	return fmt.Errorf("%w: %s", ErrOllamaProviderError, errMsg)
}

// GetBaseURL returns the base URL for the provider
func (p *OllamaProvider) GetBaseURL() string {
	return p.baseURL
}

// IsRunning checks if the Ollama server is running
func (p *OllamaProvider) IsRunning(ctx context.Context) bool {
	url := fmt.Sprintf("%s/api/tags", p.baseURL)
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

// FormatModelName formats a model name for Ollama
func FormatModelName(name string) string {
	// Ollama model names should be lowercase with no spaces
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	return name
}

// IsLocalModel returns true since Ollama is always local
func (p *OllamaProvider) IsLocalModel() bool {
	return true
}